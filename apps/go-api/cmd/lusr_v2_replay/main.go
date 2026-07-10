//go:build cgo

// lusr_v2_replay — Phase 1d : replay du LUSR v2 sur l'historique complet
// des joueurs trackés et comparaison aux cibles validées.
//
// Le cmd configure l'env LEVELUP_LUSR_V2_ENABLED=1 puis appelle le shadow
// runner (sync.RunLUSRV2Shadow) sur chaque joueur. Le shadow est idempotent
// (watermark via last_match_at), donc pour avoir une trace propre on offre
// l'option --reset qui truncate player_skill_state_v2 avant le run.
//
// Usage :
//
//	go run -tags cgo ./cmd/lusr_v2_replay --reset
//	go run -tags cgo ./cmd/lusr_v2_replay --reset Madina97294 JGtm
//
// Sortie : rapport markdown sur stdout (pipe possible dans .ai/).
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"os"
	"sort"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	lusync "levelup/go-api/internal/sync"
)

const sharedDBPath = "data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"

// defaultPlayers — les 4 joueurs trackés actuellement.
var defaultPlayers = []string{"Madina97294", "Chocoboflor", "JGtm", "XxDaemonGamerxX"}

// targets : cibles validées (memory/reference_lusr_target_levels.md).
// Note : exprimées en termes qualitatifs ; le mapping vers la grille μ
// native TrueSkill se fait à l'œil sur le rapport pour Phase 1d.
var targets = map[string]string{
	"Madina97294":     "fin Platine / début Diamant (joueur fort)",
	"Chocoboflor":     "milieu/bas Or (joueur moyen)",
	"JGtm":            "milieu/bas Or (joueur moyen)",
	"XxDaemonGamerxX": "Bronze (joueur faible)",
}

func main() {
	dbPath := flag.String("db", sharedDBPath, "chemin vers shared_matches_v2.duckdb")
	reset := flag.Bool("reset", false, "truncate player_skill_state_v2 avant le replay (état frais depuis priors)")
	flag.Parse()
	players := flag.Args()
	if len(players) == 0 {
		players = defaultPlayers
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	// Active le flag shadow programatiquement — le cmd EST le caller.
	if err := os.Setenv("LEVELUP_LUSR_V2_ENABLED", "1"); err != nil {
		slog.Error("setenv LEVELUP_LUSR_V2_ENABLED", "err", err)
		os.Exit(1)
	}

	db := openShared(*dbPath)
	defer db.Close()

	if *reset {
		if err := resetSkillV2State(context.Background(), db); err != nil {
			slog.Error("reset state", "err", err)
			os.Exit(1)
		}
		slog.Info("player_skill_state_v2 truncated")
	}

	xuidByGT := resolveXUIDs(db, players)
	ctx := context.Background()

	results := make(map[string]map[string]playerGroupSummary)
	for _, gt := range players {
		xuid := xuidByGT[strings.ToLower(gt)]
		if xuid == "" {
			slog.Warn("xuid introuvable", "gamertag", gt)
			continue
		}
		processed, err := lusync.RunLUSRV2Shadow(ctx, nil, lusync.NewPinnedSharedAccess(db), xuid)
		if err != nil {
			slog.Warn("RunLUSRV2Shadow", "gamertag", gt, "err", err)
			continue
		}
		slog.Info("shadow done", "gamertag", gt, "processed", processed)
		results[gt] = queryFinalStates(ctx, db, xuid)
	}

	// Phase 3e v2 : charge les seuils tier depuis lusr_hyperparams_v2_latest
	// (seedés par la migration shared_seed_tier_boundaries_v2). Fallback sur
	// defaults Go si la DB n'a pas encore reçu le seed.
	resolver := newTierResolver(ctx, db, []string{"arena_slayer", "arena_objectif", "btb", "chaos"})

	writeReport(os.Stdout, players, xuidByGT, results, resolver)
}

type playerGroupSummary struct {
	Mu         float64
	Sigma      float64
	Experience int
}

func openShared(path string) *sql.DB {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		slog.Error("open shared", "err", err, "path", path)
		os.Exit(1)
	}
	return db
}

func resetSkillV2State(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `DELETE FROM player_skill_state_v2`)
	return err
}

func resolveXUIDs(db *sql.DB, gamertags []string) map[string]string {
	out := map[string]string{}
	if len(gamertags) == 0 {
		return out
	}
	placeholders := strings.Repeat("?,", len(gamertags))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]interface{}, len(gamertags))
	for i, gt := range gamertags {
		args[i] = strings.ToLower(gt)
	}
	rows, err := db.Query("SELECT lower(gamertag), xuid FROM xuid_aliases "+
		"WHERE lower(gamertag) IN ("+placeholders+") ORDER BY last_seen DESC NULLS LAST", args...)
	if err != nil {
		slog.Error("resolveXUIDs query", "err", err)
		return out
	}
	defer rows.Close() //nolint:errcheck
	for rows.Next() {
		var lg, x string
		if rows.Scan(&lg, &x) == nil {
			if _, exists := out[lg]; !exists {
				out[lg] = x
			}
		}
	}
	return out
}

func queryFinalStates(ctx context.Context, db *sql.DB, xuid string) map[string]playerGroupSummary {
	out := map[string]playerGroupSummary{}
	rows, err := db.QueryContext(ctx, `
		SELECT playlist_group, mu, sigma, experience
		FROM player_skill_state_v2_latest
		WHERE xuid = ?
		ORDER BY playlist_group`, xuid)
	if err != nil {
		slog.Warn("queryFinalStates", "xuid", xuid, "err", err)
		return out
	}
	defer rows.Close() //nolint:errcheck
	for rows.Next() {
		var group string
		var s playerGroupSummary
		if err := rows.Scan(&group, &s.Mu, &s.Sigma, &s.Experience); err == nil {
			out[group] = s
		}
	}
	return out
}

// tierResolver charge les seuils tier par playlist_group depuis lusr_hyperparams_v2
// au démarrage, et retourne un closure qui formate les labels (ex: "Or III").
// Si aucun seuil n'est persisté pour un groupe, fallback sur les defaults Go.
//
// Phase 3e v2 : les defaults Go reflètent exactement les valeurs persistées par
// la migration shared_seed_tier_boundaries_v2 — donc à l'état initial le résultat
// est identique. La distinction sert quand un sysadmin ou un batch Phase 5
// écrasera certains seuils dans la DB.
type tierResolver struct {
	perGroup map[string][]skillv2.TierBoundary
}

func newTierResolver(ctx context.Context, db *sql.DB, groups []string) *tierResolver {
	r := &tierResolver{perGroup: make(map[string][]skillv2.TierBoundary, len(groups))}
	for _, g := range groups {
		hp := loadGroupHyperparams(ctx, db, g)
		r.perGroup[g] = skillv2.TierBoundariesFromHyperparams(hp)
	}
	return r
}

// Tier retourne le label complet (ex: "Or IV") pour (group, μ).
func (r *tierResolver) Tier(group string, mu float64) string {
	bs, ok := r.perGroup[group]
	if !ok {
		bs = skillv2.DefaultTierBoundaries()
	}
	return skillv2.FormatTierLabel(mu, bs)
}

// loadGroupHyperparams lit les hyperparams (name → value) d'un playlist_group
// depuis lusr_hyperparams_v2_latest. Retourne nil si la vue est vide pour ce
// groupe — TierBoundariesFromHyperparams gérera le fallback.
func loadGroupHyperparams(ctx context.Context, db *sql.DB, group string) map[string]float64 {
	rows, err := db.QueryContext(ctx, `
		SELECT name, value FROM lusr_hyperparams_v2_latest
		WHERE playlist_group = ?`, group)
	if err != nil {
		slog.Warn("loadGroupHyperparams", "group", group, "err", err)
		return nil
	}
	defer rows.Close() //nolint:errcheck
	out := map[string]float64{}
	for rows.Next() {
		var n string
		var v float64
		if rows.Scan(&n, &v) == nil {
			out[n] = v
		}
	}
	return out
}

func writeReport(w *os.File, players []string, xuidByGT map[string]string, results map[string]map[string]playerGroupSummary, resolver *tierResolver) {
	fmt.Fprintf(w, "# LUSR v2 — Phase 1d : replay sur joueurs trackés\n\n")
	fmt.Fprintf(w, "Replay du shadow runner sur tout l'historique LUSR-éligible des joueurs ci-dessous, ")
	fmt.Fprintf(w, "depuis l'état frais (priors par défaut TrueSkill : μ_0=25, σ_0=25/3, β=σ_0/2, τ=σ_0/100).\n\n")
	fmt.Fprintf(w, "## Vue d'ensemble\n\n")
	fmt.Fprintf(w, "| Joueur | XUID | Cible | Groupe le plus joué | μ (skill latent) | σ (incertitude) | Tier inféré | exp |\n")
	fmt.Fprintf(w, "|---|---|---|---|---:|---:|---|---:|\n")

	for _, gt := range players {
		xuid := xuidByGT[strings.ToLower(gt)]
		target := targets[gt]
		if target == "" {
			target = "—"
		}
		groups := results[gt]
		if len(groups) == 0 {
			fmt.Fprintf(w, "| %s | %s | %s | — | — | — | — | — |\n", gt, xuid, target)
			continue
		}
		// Sélectionne le groupe avec le plus d'expérience.
		var best string
		var bestExp int = -1
		for g, s := range groups {
			if s.Experience > bestExp {
				best = g
				bestExp = s.Experience
			}
		}
		s := groups[best]
		fmt.Fprintf(w, "| %s | %s | %s | %s | %.2f | %.2f | %s | %d |\n",
			gt, xuid, target, best, s.Mu, s.Sigma, resolver.Tier(best, s.Mu), s.Experience)
	}

	fmt.Fprintf(w, "\n## Détail par joueur × groupe\n\n")
	for _, gt := range players {
		groups := results[gt]
		if len(groups) == 0 {
			fmt.Fprintf(w, "### %s — aucun match\n\n", gt)
			continue
		}
		fmt.Fprintf(w, "### %s (cible : %s)\n\n", gt, targets[gt])
		fmt.Fprintf(w, "| Groupe | μ | σ | Tier inféré | exp |\n")
		fmt.Fprintf(w, "|---|---:|---:|---|---:|\n")
		gnames := make([]string, 0, len(groups))
		for g := range groups {
			gnames = append(gnames, g)
		}
		sort.Strings(gnames)
		for _, g := range gnames {
			s := groups[g]
			fmt.Fprintf(w, "| %s | %.2f | %.2f | %s | %d |\n",
				g, s.Mu, s.Sigma, resolver.Tier(g, s.Mu), s.Experience)
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "## Lecture\n\n")
	fmt.Fprintf(w, "- **μ ≈ 25** = niveau d'un joueur médian (prior de départ). 95%% des nouveaux joueurs en %g..%g.\n",
		math.Round(25-3*25.0/3.0), math.Round(25+3*25.0/3.0))
	fmt.Fprintf(w, "- **σ** : incertitude. σ → 0 = skill très bien connu ; σ ≈ σ_0 (8.33) = peu joué.\n")
	fmt.Fprintf(w, "- Mapping tier purement indicatif — à calibrer sur les résultats puis figer dans `lusr_hyperparams_v2` (`tier_boundary_*`).\n\n")
	fmt.Fprintf(w, "## Décision\n\n")
	fmt.Fprintf(w, "1. Validation qualitative : les μ doivent ordonner correctement les joueurs selon leurs niveaux connus.\n")
	fmt.Fprintf(w, "2. Si OK → on peut envisager Phase 2 (squadOffset) + commencer à calibrer le mapping μ → grille [1000..2000] pour la bascule.\n")
	fmt.Fprintf(w, "3. Si non OK → soit le modèle classique TrueSkill est insuffisant (besoin de Phase 3 kills/deaths), soit les priors doivent être ajustés (Sigma0 plus large pour bouger plus vite, par exemple).\n")
}
