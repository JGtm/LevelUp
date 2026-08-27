//go:build cgo

// recompute_perfnote — recompute RÉEL des notes de performance et du LUSR sur les
// DBs joueur locales (lot 4 du plan .ai/PLAN_PERF_NOTE_OBJECTIFS.md).
//
// Outil jetable, à exécuter SERVEUR ARRÊTÉ (modèle mono-process : un writer par
// DB). Il existe parce qu'aucun binaire existant ne réunit les trois exigences du
// lot : (1) câbler les classifiers de chaîne — cmd/levelup ne les pose PAS, et un
// recompute lancé depuis un binaire non câblé classerait tout le ranked en
// `ranked_slayer` EN SILENCE (découverte du lot 1) ; (2) recalculer la perf par le
// chemin backfill AVEC medal_exploit ; (3) rejouer le LUSR par le chemin canonique
// v2 (RecomputeLUSRCanonicalForPlayer) et pas le batch v1 mort.
//
// Modes :
//
//	-mode report   lecture seule (player RO, shared attachée RO en `sh`) : tables de contrôle
//	-mode sql      lecture seule, requête ad hoc avec la shared attachée en `sh`
//	-mode perf     ÉCRIT la player DB : notes force=true, medal_exploit inclus
//	-mode lusr     ÉCRIT la player DB ET la shared : replay LUSR v2 canonique complet
//
// Usage (depuis apps/go-api) :
//
//	go run -tags cgo ./cmd/recompute_perfnote -mode report -data-root ../.. -label AVANT
//	go run -tags cgo ./cmd/recompute_perfnote -mode perf   -data-root ../..
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/skillchain"
	lusync "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/skill"
)

const titleSlug = "halo_infinite"

var defaultPlayers = []string{"JGtm", "Chocoboflor", "Madina97294", "XxDaemonGamerxX"}

// runEnv porte le contexte d'exécution commun aux quatre modes.
type runEnv struct {
	paths   *title.PathResolver
	players []string
	label   string
}

func main() {
	// Seams AVANT tout calcul (exigence B4.1 du plan).
	wireClassifiers()

	mode := flag.String("mode", "report", "report | sql | perf | lusr")
	dataRoot := flag.String("data-root", "../..", "racine du repo (depuis apps/go-api)")
	players := flag.String("players", "", "gamertags séparés par des virgules (défaut : les 4 suivis)")
	label := flag.String("label", "", "étiquette imprimée en tête du rapport (ex: AVANT / APRES)")
	query := flag.String("sql", "", "mode sql : requête à exécuter (shared attachée en `sh`)")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	env := runEnv{
		paths:   title.NewPathResolver(*dataRoot),
		players: splitPlayers(*players),
		label:   *label,
	}
	ctx := context.Background()

	var err error
	switch *mode {
	case "report":
		err = runReport(ctx, env)
	case "sql":
		err = runSQL(ctx, env, *query)
	case "perf":
		err = runPerf(ctx, env)
	case "lusr":
		err = runLUSR(ctx, env)
	default:
		err = fmt.Errorf("mode inconnu: %q", *mode)
	}
	if err != nil {
		slog.Error("recompute_perfnote: échec", "mode", *mode, "err", err)
		os.Exit(1)
	}
}

// wireClassifiers pose les deux seams de classification puis fail-fast si l'un
// manque — symétrie stricte avec le boot serveur (cmd/server/main.go:1707-1730).
// Sans le seam de famille objectif, GetPerformanceChain retomberait sur son
// fallback `ranked_slayer` sans le dire ; sans le seam LUSR, playlist_group
// serait faux. Les deux écrivent des données persistées : fail-fast obligatoire.
func wireClassifiers() {
	lusync.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain)
	lusync.SetObjectiveFamilyClassifier(skillchain.IsObjectiveSubMode)
	if err := lusync.ValidateLUSRChainClassifierWired(); err != nil {
		slog.Error("recompute_perfnote: classifier LUSR non câblé", "err", err)
		os.Exit(1)
	}
	if err := lusync.ValidateObjectiveFamilyClassifierWired(); err != nil {
		slog.Error("recompute_perfnote: classifier famille objectif non câblé", "err", err)
		os.Exit(1)
	}
}

func splitPlayers(csv string) []string {
	if strings.TrimSpace(csv) == "" {
		return defaultPlayers
	}
	var out []string
	for _, p := range strings.Split(csv, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// openDB ouvre une DuckDB locale. readOnly=true pose access_mode=read_only et
// borne le pool à 1 connexion (l'ATTACH du rapport doit rester visible d'une
// requête à l'autre). En écriture le pool reste libre : le batch de perf lit des
// rows tout en écrivant, une connexion unique le bloquerait.
func openDB(path string, readOnly bool) (*sql.DB, error) {
	dsn := path
	if readOnly {
		dsn += "?access_mode=read_only"
	}
	db, err := sql.Open("duckdb", dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	if readOnly {
		db.SetMaxOpenConns(1)
	}
	return db, nil
}

// resolveXUID résout le xuid d'un gamertag via v_gamertag_lookup (shared).
func resolveXUID(ctx context.Context, shared *sql.DB, gamertag string) (string, error) {
	var xuid sql.NullString
	err := shared.QueryRowContext(ctx,
		`SELECT xuid FROM v_gamertag_lookup WHERE LOWER(gamertag) = LOWER(?) LIMIT 1`,
		gamertag).Scan(&xuid)
	if err != nil {
		return "", fmt.Errorf("resolveXUID %s: %w", gamertag, err)
	}
	if strings.TrimSpace(xuid.String) == "" {
		return "", fmt.Errorf("resolveXUID %s: xuid vide", gamertag)
	}
	return strings.TrimSpace(xuid.String), nil
}

// checkpoint force le flush du WAL (best-effort, loggé — jamais avalé).
func checkpoint(ctx context.Context, db *sql.DB, what string) {
	if _, err := db.ExecContext(ctx, `CHECKPOINT`); err != nil {
		slog.WarnContext(ctx, "recompute_perfnote: CHECKPOINT échoué", "db", what, "err", err)
	}
}

// runPerf recalcule les notes des joueurs en force=true, par le chemin qui
// inclut medal_exploit. La shared est ouverte en LECTURE SEULE : le batch de
// perf n'écrit que la player DB, l'ouvrir en RW n'apporterait qu'un risque.
func runPerf(ctx context.Context, env runEnv) error {
	shared, err := openDB(env.paths.SharedDBPath(titleSlug), true)
	if err != nil {
		return err
	}
	defer shared.Close()

	metadataPath := env.paths.MetadataDBPath(titleSlug)
	fmt.Printf("=== RECOMPUTE PERF (force=true, medal_exploit inclus) — %s ===\n",
		time.Now().Format(time.RFC3339))

	for _, gt := range env.players {
		xuid, err := resolveXUID(ctx, shared, gt)
		if err != nil {
			return err
		}
		player, err := openDB(env.paths.PlayerDBPath(titleSlug, gt), false)
		if err != nil {
			return err
		}
		start := time.Now()
		n, err := lusync.RecomputePerformanceScoresWithMedals(ctx, player, shared, metadataPath, xuid, true)
		if err != nil {
			player.Close()
			return fmt.Errorf("recompute perf %s: %w", gt, err)
		}
		checkpoint(ctx, player, gt)
		player.Close()
		fmt.Printf("  %-18s xuid=%s  updated=%d  (%s)\n", gt, xuid, n, time.Since(start).Round(time.Millisecond))
	}
	return nil
}

// runLUSR rejoue le LUSR v2 canonique complet de chaque joueur.
//
// Les deux flags d'environnement sont OBLIGATOIRES ici : DefaultLUSRModeIfUnset
// n'est appelée qu'au boot serveur, et les CLI restent volontairement sur le
// défaut v1 + opt-in explicite (skill_v2_canonical.go:46-57). Sans eux le replay
// serait shadow-only : aucune ligne canonique `LUSR` écrite. Même pose que
// cmd/lusr_v2_canonical_backfill.
//
// La shared est ouverte en RW : RecomputeLUSRCanonicalForPlayer y insère la
// sentinelle de reset du watermark puis les états player_skill_state_v2.
func runLUSR(ctx context.Context, env runEnv) error {
	if err := os.Setenv("LEVELUP_LUSR_V2_ENABLED", "1"); err != nil {
		return fmt.Errorf("setenv LEVELUP_LUSR_V2_ENABLED: %w", err)
	}
	if err := os.Setenv("LEVELUP_LUSR_CANONICAL", "LUSR_V2"); err != nil {
		return fmt.Errorf("setenv LEVELUP_LUSR_CANONICAL: %w", err)
	}
	// Anti no-op SILENCIEUX : les trois gates du replay (capability du titre, v2
	// activé, v2 canonique) dégradent tous en « 0 match traité » sans erreur. On
	// les vérifie ICI, avant d'ouvrir la moindre DB en écriture.
	if !skill.SlugHasLUSR(titleSlug) {
		return fmt.Errorf("titre %s sans capability LUSR : le replay serait un no-op", titleSlug)
	}
	if !lusync.IsLUSRV2Enabled() || !lusync.IsLUSRV2Canonical() {
		return fmt.Errorf("LUSR v2 non activé/non canonique : le replay n'écrirait aucune ligne canonique")
	}

	shared, err := openDB(env.paths.SharedDBPath(titleSlug), false)
	if err != nil {
		return err
	}
	defer shared.Close()

	fmt.Printf("=== RECOMPUTE LUSR v2 CANONIQUE (replay complet) — %s ===\n",
		time.Now().Format(time.RFC3339))

	for _, gt := range env.players {
		xuid, err := resolveXUID(ctx, shared, gt)
		if err != nil {
			return err
		}
		player, err := openDB(env.paths.PlayerDBPath(titleSlug, gt), false)
		if err != nil {
			return err
		}
		start := time.Now()
		n, err := lusync.RecomputeLUSRCanonicalForPlayer(ctx, player, shared, xuid)
		if err != nil {
			player.Close()
			return fmt.Errorf("recompute lusr %s: %w", gt, err)
		}
		checkpoint(ctx, player, gt)
		player.Close()
		fmt.Printf("  %-18s xuid=%s  processed=%d  (%s)\n", gt, xuid, n, time.Since(start).Round(time.Millisecond))
	}
	checkpoint(ctx, shared, "shared")
	return nil
}
