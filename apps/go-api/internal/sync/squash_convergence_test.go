//go:build integration

package sync_test

// squash_convergence_test.go — C3 du plan de correctifs revue 2026-07-17 (findings M2/M3/M4).
//
// Prouve que, quel que soit l'état INTERMÉDIAIRE d'une player DB legacy (backup restauré à
// une date où un step additif n'avait pas encore été déployé mais la sentinelle DM-5 l'était
// déjà, ou pré-conversion append-only de player_csr_snapshots), un BOOT COMPLET
// (RunForDB(TargetPlayer) puis OpenPlayerDB) CONVERGE : OpenPlayerDB réussit, la vue
// player_csr_snapshots_latest est liée et requêtable, un INSERT persist LUSR (avec
// expected_win_prob) et un INSERT challenges (avec les colonnes render) réussissent. Et un
// SECOND boot est idempotent (aucun échec, schéma réparé stable).
//
// Les 4 fixtures sont construites PROGRAMMATIQUEMENT (pas de binaire committé) :
//   (a) pré-2026-05-24  : ANCIEN player_csr_snapshots (PK(playlist_id, season_id), sans
//                         id/written_at) — DDL exact de git show 37264462f^ → réparé par C1
//                         (EnsurePlayerSchema.EnsurePlayerCSRSnapshotsAppendOnly).
//   (b) 05-24 → 05-28   : sentinelle DM-5 présente, colonne expected_win_prob ABSENTE de
//                         match_skill_rank → réparé par l'ensure additif A4 (chemin DM-5).
//   (c) pré-06-22       : sentinelle présente, colonnes render de challenge_snapshots ABSENTES
//                         → réparé par l'ensure additif A4 (PREUVE de C2).
//   (d) pré-07-11       : sentinelle présente, table engagement_response_bins ABSENTE → réparé
//                         par l'ensure additif A4.

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	duckdb "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync"

	_ "github.com/duckdb/duckdb-go/v2"
)

// oldPlayerCSRSnapshotsDDL — schéma EXACT de player_csr_snapshots pré-2026-05-24
// (git show 37264462f^:apps/go-api/internal/sync/schema.go), sans id/written_at.
const oldPlayerCSRSnapshotsDDL = `
CREATE TABLE player_csr_snapshots (
	playlist_id                   VARCHAR NOT NULL,
	playlist_name                 VARCHAR,
	queue                         VARCHAR,
	input                         VARCHAR,
	season_id                     VARCHAR NOT NULL,
	current_value                 FLOAT,
	current_tier                  VARCHAR,
	current_sub_tier              SMALLINT,
	current_measurement_remaining INTEGER,
	season_value                  FLOAT,
	season_tier                   VARCHAR,
	season_sub_tier               SMALLINT,
	alltime_value                 FLOAT,
	alltime_tier                  VARCHAR,
	alltime_sub_tier              SMALLINT,
	fetched_at                    TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
	PRIMARY KEY (playlist_id, season_id)
)`

const sentinelName = "player_append_only_csr_snapshots_v1"

func TestSquashConvergence_PlayerDBIntermediateStates(t *testing.T) {
	// Filet : câbler le provider de steps title-owned (identique au TestMain du package)
	// — robustesse si ce test tourne sans les autres fichiers _test du package.
	migration.SetTitleStepsProvider(halomigrations.StepsFor)

	cases := []struct {
		name  string
		build func(t *testing.T, path string)
	}{
		{"a_pre_2026_05_24_old_csr_snapshots", buildFixtureOldCSRSnapshots},
		{"b_sentinel_without_expected_win_prob", buildFixtureNoExpectedWinProb},
		{"c_without_challenge_render_columns", buildFixtureNoChallengeRenderColumns},
		{"d_without_engagement_response_bins", buildFixtureNoEngagementResponseBins},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "stats.duckdb")
			t.Cleanup(func() { duckdb.EvictAndCloseCached(path) })

			tc.build(t, path)

			// Boot 1 : convergence.
			h1 := bootPlayerDB(t, path)
			assertConverged(t, h1.SQLDb(), "boot1")
			pcs1 := countRows(t, h1.SQLDb(), "player_csr_snapshots")
			mustHaveColumn(t, h1.SQLDb(), "player_csr_snapshots", "id")
			h1.Close()

			// Boot 2 : idempotence (aucun échec, schéma réparé stable, pas d'orphelin).
			h2 := bootPlayerDB(t, path)
			assertConverged(t, h2.SQLDb(), "boot2")
			pcs2 := countRows(t, h2.SQLDb(), "player_csr_snapshots")
			mustHaveColumn(t, h2.SQLDb(), "player_csr_snapshots", "id")
			if hasTable(t, h2.SQLDb(), "player_csr_snapshots__appendonly") {
				t.Error("idempotence: table orpheline player_csr_snapshots__appendonly ne devrait pas apparaître")
			}
			h2.Close()

			if pcs1 != pcs2 {
				t.Errorf("idempotence: player_csr_snapshots count boot1=%d boot2=%d (doit être stable)", pcs1, pcs2)
			}
		})
	}
}

// TestEnsurePlayerSchema_C4_UnrepairableBindReturnsExplicitError verrouille C4 : un échec de
// bind au boot qui n'est PAS réparable par la réparation append-only ne doit pas rester un échec
// PERMANENT SILENCIEUX. Le filet doit journaliser puis retourner une erreur EXPLICITE exploitable
// (jamais de panic, jamais d'avalement). On force le cas en occupant le NOM de la vue
// player_csr_snapshots_latest par une TABLE : CREATE OR REPLACE VIEW échouera, et la réparation
// (introspection de player_csr_snapshots) n'y peut rien.
func TestEnsurePlayerSchema_C4_UnrepairableBindReturnsExplicitError(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE player_csr_snapshots_latest (x INTEGER)`); err != nil {
		t.Fatalf("seed table-squattant-le-nom-de-vue: %v", err)
	}

	err = sync.EnsurePlayerSchema(context.Background(), db)
	if err == nil {
		t.Fatal("EnsurePlayerSchema aurait dû retourner une erreur explicite (bind vue impossible)")
	}
	if !strings.Contains(err.Error(), "intervention requise") {
		t.Errorf("erreur pas assez explicite (filet C4) : %v", err)
	}
}

// bootPlayerDB reproduit le boot serveur d'une player DB : migrations (RunForDB, boot 3b)
// PUIS ouverture provider (OpenPlayerDB → EnsurePlayerSchema). Retourne le handle ouvert
// (l'appelant le ferme).
func bootPlayerDB(t *testing.T, path string) *duckdb.DB {
	t.Helper()
	mdb, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("open rw (migrations): %v", err)
	}
	if err := migration.RunForDB(mdb.SQLDb(), migration.TargetPlayer); err != nil {
		mdb.Close()
		t.Fatalf("RunForDB(player): %v", err)
	}
	mdb.Close()

	h, err := sync.OpenPlayerDB(path)
	if err != nil {
		t.Fatalf("OpenPlayerDB (boot doit converger): %v", err)
	}
	return h
}

// assertConverged vérifie la vue _latest + les deux écritures qui cassaient (LUSR / challenges).
// suffix rend les clés d'INSERT uniques par boot (append-only : évite tout faux conflit de PK).
func assertConverged(t *testing.T, db *sql.DB, suffix string) {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM player_csr_snapshots_latest`).Scan(&n); err != nil {
		t.Fatalf("player_csr_snapshots_latest non liée/requêtable: %v", err)
	}
	// persist LUSR — liste de colonnes de lusr_append_only_persister (dont expected_win_prob).
	if _, err := db.Exec(`
		INSERT INTO match_skill_rank
			(match_id, rating_type, rating_value, rating_deviation,
			 tier, tier_fr, sub_tier, tier_label,
			 rating_delta, playlist_group, expected_win_prob, start_time)
		VALUES (?, 'LUSR', 1450, 40, 'Diamant', 'Diamant', 3, 'Diamant 3', 12, 'ranked', 0.55, NULL)`,
		"conv-lusr-"+suffix); err != nil {
		t.Fatalf("persist LUSR (expected_win_prob) échoué: %v", err)
	}
	// persist challenges — liste de colonnes de persist_sink_challenges (dont render).
	if _, err := db.Exec(`
		INSERT INTO challenge_snapshots
			(snapshot_at, xuid, challenge_path, challenge_id,
			 status, progress_current, progress_target, xp_reward,
			 can_reroll, expires_at, state_hash, title, description, image_url, display_path)
		VALUES (CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP), 'u1', ?, 'c1', 'active', 1, 5, 100, false, NULL, ?, 'Titre', 'Desc', 'img://x', 'disp/x')`,
		"path/"+suffix, "hash-"+suffix); err != nil {
		t.Fatalf("persist challenges (colonnes render) échoué: %v", err)
	}
}

// ── Fixtures ─────────────────────────────────────────────────────────────────

// (a) DB minimale portant l'ANCIEN player_csr_snapshots (le reste est créé par le boot).
func buildFixtureOldCSRSnapshots(t *testing.T, path string) {
	t.Helper()
	h, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer h.Close()
	db := h.SQLDb()
	if _, err := db.Exec(oldPlayerCSRSnapshotsDDL); err != nil {
		t.Fatalf("old player_csr_snapshots DDL: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO player_csr_snapshots (playlist_id, season_id, current_value) VALUES (?, ?, ?)`,
		"ranked-arena", "csrseason13-2", 1502); err != nil {
		t.Fatalf("seed old csr snapshot: %v", err)
	}
}

// (b) sentinelle DM-5 présente, expected_win_prob absente de match_skill_rank.
// DuckDB refuse DROP COLUMN si un index dépend d'une colonne POSITIONNÉE APRÈS elle (written_at)
// ou si la vue _latest (SELECT *) la référence : on retire d'abord la vue + les index msr (tous
// recréés par playerSchemaSQL au boot de convergence, CREATE IF NOT EXISTS / CREATE OR REPLACE).
func buildFixtureNoExpectedWinProb(t *testing.T, path string) {
	buildPostSquashSentinelFixture(t, path, []string{
		`DROP VIEW IF EXISTS match_skill_rank_latest`,
		`DROP INDEX IF EXISTS idx_msr_match_lookup`,
		`DROP INDEX IF EXISTS idx_msr_rating_type`,
		`DROP INDEX IF EXISTS idx_msr_playlist`,
		`ALTER TABLE match_skill_rank DROP COLUMN expected_win_prob`,
	})
}

// (c) sentinelle présente, colonnes render de challenge_snapshots absentes (PREUVE C2).
// DuckDB refuse ALTER DROP COLUMN tant qu'un index existe sur la table : on retire d'abord les
// index challenge_snapshots (le heal DM-5 A4 rétablit les COLONNES render — c'est le périmètre
// exact du finding « INSERT challenges casse » ; les index sont orthogonaux à cette convergence).
func buildFixtureNoChallengeRenderColumns(t *testing.T, path string) {
	buildPostSquashSentinelFixture(t, path, []string{
		`DROP INDEX IF EXISTS idx_challenge_snapshots_xuid_time`,
		`DROP INDEX IF EXISTS idx_challenge_snapshots_path_time`,
		`ALTER TABLE challenge_snapshots DROP COLUMN title`,
		`ALTER TABLE challenge_snapshots DROP COLUMN description`,
		`ALTER TABLE challenge_snapshots DROP COLUMN image_url`,
		`ALTER TABLE challenge_snapshots DROP COLUMN display_path`,
	})
}

// (d) sentinelle présente, table engagement_response_bins absente.
func buildFixtureNoEngagementResponseBins(t *testing.T, path string) {
	buildPostSquashSentinelFixture(t, path, []string{
		`DROP TABLE IF EXISTS engagement_response_bins`,
	})
}

// buildPostSquashSentinelFixture matérialise un backup PRÉ-SQUASH porteur de la sentinelle :
// un boot complet établit le schéma courant, puis on rebascule le ledger (retire le marqueur de
// squash, insère la sentinelle DM-5) et on reproduit le TROU précis (mutations).
func buildPostSquashSentinelFixture(t *testing.T, path string, mutations []string) {
	t.Helper()
	h := bootPlayerDB(t, path)
	defer h.Close()
	db := h.SQLDb()

	if _, err := db.Exec(
		`DELETE FROM schema_migrations WHERE name = ?`, halomigrations.BaselinePlayerV1Name); err != nil {
		t.Fatalf("ledger delete baseline: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO schema_migrations (name, description, schema_done, backfill_done, title_slug)
		VALUES (?, 'sentinelle DM-5 (fixture)', TRUE, TRUE, 'halo_infinite')`, sentinelName); err != nil {
		t.Fatalf("ledger insert sentinel: %v", err)
	}
	for _, stmt := range mutations {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("mutation fixture (%s): %v", stmt, err)
		}
	}
}

// ── petits helpers ───────────────────────────────────────────────────────────

func countRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func mustHaveColumn(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM information_schema.columns WHERE table_name = ? AND column_name = ?`,
		table, column).Scan(&n); err != nil {
		t.Fatalf("check %s.%s: %v", table, column, err)
	}
	if n == 0 {
		t.Fatalf("colonne %s.%s absente après convergence", table, column)
	}
}

func hasTable(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?`, table).Scan(&n); err != nil {
		t.Fatalf("check table %s: %v", table, err)
	}
	return n > 0
}
