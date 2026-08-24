// cmd/backfill-team-scores — corrige `match_registry.team_0_score` / `team_1_score`
// (Halo Infinite) sur les matchs dont la base porte autre chose que le score AFFICHÉ.
//
// # CE QU'IL RÉPARE, ET POURQUOI UN RE-SYNC NE SUFFIT PAS
//
// Mesure du 2026-08-24 sur les 1 934 matchs du corpus
// (`.ai/V7.5/replay2d/RAPPORT_QUALITE_SCORE_EQUIPE.md`) : 80 lignes portent une valeur qui
// n'est pas le score affiché — 69 d'entre elles portent, au tick près, le compteur brut du
// mode (`Teams[].Stats.ZonesStats.StrongholdScoringTicks`). Héritage d'une période où l'API
// 343 servait ce compteur dans `CoreStats.Score` ; elle a été corrigée entre avril et mai
// 2026, et les 396 matchs entrés depuis sont justes à 100 %. Le défaut est donc CLOS À LA
// SOURCE : il ne reste qu'un résidu.
//
// Une resynchronisation ne le répare PAS : `persistMatchRegistry`
// (`internal/persist/shared_persister.go:154`) est un INSERT NU, sans `ON CONFLICT` — un
// match déjà présent n'est jamais réécrit. D'où cet outil.
//
// # CE QU'IL NE FAIT PAS
//
// Il ne fait JAMAIS confiance au fichier de liste pour les VALEURS : le TSV ne fournit que
// des `match_id`, et chaque score est re-téléchargé à l'exécution. Il n'écrit que si la
// valeur diffère, jamais un NULL, jamais un négatif, jamais hors bornes du SMALLINT de la
// colonne. Un match sans camps 0 et 1 (FFA) est sauté et loggé.
//
// # ÉCRITURE
//
// `--dry-run` est le DÉFAUT : sans `--apply`, aucune écriture, l'outil imprime ce qu'il
// ferait. Avec `--apply`, l'écriture est un `UPDATE … WHERE match_id = ?` row-by-row
// sérialisé sous lease writer `dblease.KindSharedMatches` — jamais un
// `UPDATE … FROM (VALUES …)`, forme interdite par `internal/sync/no_art_patterns_test.go`
// (déclencheur ART #23046). Idempotent : rejouer l'outil ne change plus rien.
//
// Le mode `--apply` ouvre la base en lecture-écriture : le SERVEUR DOIT ÊTRE ARRÊTÉ
// (ADR 0013, un seul writer par base). Le mode dry-run, lui, lit par `OpenReadForQuery` et
// tourne serveur allumé.
//
// # USAGE
//
//	# 1. répétition à blanc (serveur allumé, aucune écriture)
//	LEVELUP_REPO_ROOT=<repo qui porte data/> \
//	  go run ./cmd/backfill-team-scores --gamertag JGtm
//
//	# 2. application (SERVEUR ARRÊTÉ)
//	LEVELUP_REPO_ROOT=<repo qui porte data/> \
//	  go run ./cmd/backfill-team-scores --gamertag JGtm --apply
//
//	# un seul match
//	go run ./cmd/backfill-team-scores --match 7344d24f-0154-4949-80ad-e2b781c122f1 --apply
//
// Un `--ids-file` relatif est résolu depuis la racine du dépôt (`LEVELUP_REPO_ROOT`), pas
// depuis le répertoire courant. Auth : `MultiUserTokenStore` (ADR 0023), aucune re-capture.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/dblease"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	go_sync "levelup/go-api/internal/sync"
)

// defaultIDsFile : la liste nominative produite par la phase 1 du lot, versionnée.
const defaultIDsFile = ".ai/V7.5/replay2d/registre_film/score_equipe_ecarts_2026-08-24.tsv"

// writerLeaseTimeout borne l'attente du lease : au-delà, un autre process tient la base
// et l'outil doit le dire plutôt que d'attendre indéfiniment.
const writerLeaseTimeout = 30 * time.Second

// tally compte les issues. Publié tel quel en fin de course.
type tally struct {
	read, identical, fixed, skipped, failed int
}

func main() {
	gamertag := flag.String("gamertag", "JGtm", "Gamertag dont les tokens servent à l'auth API (db_profiles.json)")
	idsFile := flag.String("ids-file", defaultIDsFile, "Liste de match_id (TSV à en-tête `match_id`, ou une colonne nue). Relatif = depuis la racine du dépôt")
	single := flag.String("match", "", "Traiter CE seul match_id (ignore --ids-file)")
	apply := flag.Bool("apply", false, "ÉCRIRE réellement. Sans ce drapeau : répétition à blanc, aucune écriture")
	rps := flag.Int("rps", 4, "Requêtes API par seconde")
	flag.Parse()

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		fatal("config: %v", err)
	}
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	sharedPath := pr.SharedDBPath(titlePkg.DefaultSlug)

	matchIDs, err := resolveMatchIDs(*single, *idsFile, cfg.RepoRoot)
	if err != nil {
		fatal("%v", err)
	}

	client, err := newAPIClient(ctx, cfg, pr, *gamertag, *rps)
	if err != nil {
		fatal("%v", err)
	}

	slog.InfoContext(ctx, "backfill_team_scores: démarrage",
		"matchs", len(matchIDs), "apply", *apply, "shared_db", sharedPath)

	t, err := run(ctx, runDeps{
		client: client, sharedPath: sharedPath, matchIDs: matchIDs, apply: *apply,
	})
	if err != nil {
		fatal("%v", err)
	}

	slog.InfoContext(ctx, "backfill_team_scores: terminé",
		"apply", *apply, "lus", t.read, "identiques", t.identical,
		"corriges", t.fixed, "skippes", t.skipped, "echecs", t.failed)
	if !*apply && t.fixed > 0 {
		slog.WarnContext(ctx, "backfill_team_scores: répétition à blanc — RIEN n'a été écrit ; relancer avec --apply (serveur arrêté) pour appliquer",
			"a_corriger", t.fixed)
	}
	if t.failed > 0 {
		os.Exit(1)
	}
}

// runDeps regroupe ce dont la boucle a besoin (≤ 5 paramètres, seuil CLAUDE.md).
type runDeps struct {
	client     *go_sync.HaloAPIClient
	sharedPath string
	matchIDs   []string
	apply      bool
}

// run ouvre la base selon le mode, puis traite les matchs un par un.
func run(ctx context.Context, d runDeps) (tally, error) {
	var t tally
	db, writer, closeDB, err := openRegistry(d.sharedPath, d.apply)
	if err != nil {
		return t, err
	}
	defer closeDB()
	if writer != nil {
		defer writer.Release()
	}

	for _, id := range d.matchIDs {
		processMatch(ctx, d, db, writer, id, &t)
	}
	return t, nil
}

// processMatch traite UN match : fetch, lecture du registre, décision, écriture éventuelle.
func processMatch(ctx context.Context, d runDeps, db *sql.DB, w *dblease.LeasedWriter, id string, t *tally) {
	matchJSON, err := d.client.GetMatchStats(ctx, id)
	if err != nil {
		slog.ErrorContext(ctx, "backfill_team_scores: GetMatchStats échoué", "match_id", id, "err", err)
		t.failed++
		return
	}
	cur, found, err := loadRegistryScores(ctx, db, id)
	if err != nil {
		slog.ErrorContext(ctx, "backfill_team_scores: lecture match_registry échouée", "match_id", id, "err", err)
		t.failed++
		return
	}
	if !found {
		slog.WarnContext(ctx, "backfill_team_scores: match absent de match_registry — sauté", "match_id", id)
		t.skipped++
		return
	}
	t.read++

	dec := Decide(matchJSON, cur)
	switch dec.Verdict {
	case VerdictIdentical:
		t.identical++
		slog.DebugContext(ctx, "backfill_team_scores: identique", "match_id", id, "cause", dec.Reason)
		return
	case VerdictSkipNoTeams, VerdictSkipImplausible:
		t.skipped++
		slog.WarnContext(ctx, "backfill_team_scores: sauté",
			"match_id", id, "verdict", string(dec.Verdict), "cause", dec.Reason)
		return
	}

	slog.InfoContext(ctx, "backfill_team_scores: correction",
		"match_id", id, "apply", d.apply,
		"avant", formatScores(dec.Old), "apres", fmt.Sprintf("%d/%d", dec.NewTeam0, dec.NewTeam1),
		"cause", dec.Reason)
	if !d.apply {
		t.fixed++
		return
	}
	if err := applyFix(ctx, w, id, dec); err != nil {
		slog.ErrorContext(ctx, "backfill_team_scores: UPDATE échoué", "match_id", id, "err", err)
		t.failed++
		return
	}
	t.fixed++
}

// applyFix écrit UNE ligne. La forme est imposée par le garde-rail anti-ART : un UPDATE
// row-by-row, `WHERE match_id = ?`, avec toutes les valeurs liées à des placeholders.
// Aucune autre forme n'est acceptable sur `match_registry` (cf. `criticalMatchTables`).
func applyFix(ctx context.Context, w *dblease.LeasedWriter, matchID string, d Decision) error {
	if w == nil {
		return errors.New("écriture demandée sans lease writer (bug d'appel)")
	}
	if !d.Writable() {
		return fmt.Errorf("écriture demandée sur un verdict non écrivable (%s)", d.Verdict)
	}
	res, err := w.ExecContext(ctx,
		`UPDATE match_registry SET team_0_score = ?, team_1_score = ? WHERE match_id = ?`,
		d.NewTeam0, d.NewTeam1, matchID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil // pilote sans RowsAffected : l'UPDATE a réussi, on ne peut pas compter
	}
	if n != 1 {
		return fmt.Errorf("UPDATE a touché %d lignes au lieu de 1", n)
	}
	return nil
}

// loadRegistryScores lit les deux colonnes. found=false quand le match n'est pas au registre.
func loadRegistryScores(ctx context.Context, db *sql.DB, matchID string) (RegistryScores, bool, error) {
	var t0, t1 sql.NullInt64
	err := db.QueryRowContext(ctx,
		`SELECT team_0_score, team_1_score FROM match_registry WHERE match_id = ?`,
		matchID).Scan(&t0, &t1)
	if errors.Is(err, sql.ErrNoRows) {
		return RegistryScores{}, false, nil
	}
	if err != nil {
		return RegistryScores{}, false, err
	}
	return RegistryScores{Team0: nullIntPtr(t0), Team1: nullIntPtr(t1)}, true, nil
}

func nullIntPtr(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

// openRegistry ouvre la base selon le mode.
//
// Dry-run : `OpenReadForQuery` — lecture qui cohabite avec un serveur qui tient la base.
// Apply   : `OpenReadWrite` + lease writer — échoue si un autre process tient la base,
// ce qui EST le comportement voulu (ADR 0013 : un seul writer).
func openRegistry(path string, apply bool) (*sql.DB, *dblease.LeasedWriter, func(), error) {
	if !apply {
		db, closer, err := duckdbpkg.OpenReadForQuery(path)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("ouverture lecture de %s : %w", path, err)
		}
		return db, nil, closer, nil
	}
	handle, err := duckdbpkg.OpenReadWrite(path)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("ouverture écriture de %s (serveur LevelUp arrêté ?) : %w", path, err)
	}
	db := handle.SQLDb()
	w, err := dblease.AcquireWriter(db, path, dblease.KindSharedMatches, writerLeaseTimeout)
	if err != nil {
		_ = handle.Close()
		return nil, nil, nil, fmt.Errorf("lease writer sur %s : %w", path, err)
	}
	return db, w, func() { _ = handle.Close() }, nil
}

// resolveMatchIDs rend la liste de travail : le match unique s'il est fourni, sinon le
// fichier. Un chemin relatif est résolu depuis la racine du dépôt.
func resolveMatchIDs(single, idsFile, repoRoot string) ([]string, error) {
	if single != "" {
		return []string{single}, nil
	}
	path := idsFile
	if !filepath.IsAbs(path) {
		path = filepath.Join(repoRoot, path)
	}
	return LoadMatchIDs(path)
}

// newAPIClient résout le xuid du porteur puis rafraîchit ses tokens (ADR 0023).
// AUCUNE re-capture : un refresh token valide se rafraîchit ; s'il est mort, on s'arrête
// en le disant plutôt que d'en fabriquer un nouveau.
func newAPIClient(ctx context.Context, cfg *config.AppConfig, pr *titlePkg.PathResolver,
	gamertag string, rps int) (*go_sync.HaloAPIClient, error) {
	xuid, err := resolveXUID(cfg, gamertag)
	if err != nil {
		return nil, fmt.Errorf("résolution du xuid de %s : %w", gamertag, err)
	}
	store := auth_platform.NewMultiUserTokenStore(pr.WatcherTokensDir())
	exch, err := auth_platform.RefreshHaloTokensViaStoreFirst(
		ctx, store, auth_platform.NewSISUProvider(), xuid, gamertag, auth_platform.LegacyAuthInputs{})
	if err != nil || exch == nil || exch.Tokens == nil {
		return nil, fmt.Errorf("auth %s (xuid %s) : %w — diagnostiquer (RT tourné perdu, mauvais xuid), NE PAS re-capturer", gamertag, xuid, err)
	}
	return go_sync.NewHaloAPIClient(exch.Tokens.SpartanToken, exch.Tokens.ClearanceToken, rps), nil
}

// resolveXUID lit le xuid d'un gamertag dans db_profiles.json.
func resolveXUID(cfg *config.AppConfig, gamertag string) (string, error) {
	players, err := cfg.LoadPlayers(titlePkg.DefaultSlug)
	if err != nil {
		return "", err
	}
	for i := range players {
		if players[i].Gamertag == gamertag {
			return players[i].XUID, nil
		}
	}
	return "", fmt.Errorf("gamertag %q absent de db_profiles.json", gamertag)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
