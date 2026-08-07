//go:build integration

// Package migration — tests d'intégration CGO pour applyAppendOnlyMatchEnrichment
// (player_match_enrichment append-only + vue merge-on-read par-groupe, #23046).
//
// Le test CENTRAL (Merge) prouve la sémantique merge-on-read : écritures partielles
// par étape (stage), reconstitution multi-colonnes, reset à NULL légitime préservé,
// toggle booléen bidirectionnel, fallback legacy. C'est le POC du design.

package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupLegacyMatchEnrichment crée une PME au schéma de base (PK match_id, comme
// schema.go) + rows ; ensurePMEColumns de la migration ajoutera les colonnes additives.
func setupLegacyMatchEnrichment(t *testing.T, seed func(*sql.DB)) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE player_match_enrichment (
			match_id               VARCHAR PRIMARY KEY,
			performance_score      FLOAT,
			session_id             VARCHAR,
			session_label          VARCHAR,
			is_with_friends        BOOLEAN DEFAULT FALSE,
			teammates_signature    VARCHAR,
			known_teammates_count  SMALLINT,
			friends_xuids          VARCHAR,
			had_bot_teammate       BOOLEAN,
			is_excluded            BOOLEAN DEFAULT FALSE,
			psa_checked_at         TIMESTAMP,
			created_at             TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			updated_at             TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);
		CREATE INDEX idx_pme_session ON player_match_enrichment(session_id);
	`); err != nil {
		t.Fatalf("create legacy: %v", err)
	}
	if seed != nil {
		seed(db)
	}
	return db
}

// TestMatchEnrichmentAppendOnly_Basic — migration applique : id/stage/written_at
// présents, rows legacy préservées, vue accessible et reflète l'état legacy.
func TestMatchEnrichmentAppendOnly_Basic(t *testing.T) {
	db := setupLegacyMatchEnrichment(t, func(db *sql.DB) {
		if _, err := db.Exec(`INSERT INTO player_match_enrichment (match_id, performance_score, session_id) VALUES ('m1', 1.5, 'sessA'), ('m2', 2.0, 'sessB')`); err != nil {
			t.Fatalf("seed: %v", err)
		}
	})

	if err := applyAppendOnlyMatchEnrichment(db); err != nil {
		t.Fatalf("apply: %v", err)
	}

	for _, col := range []string{"id", "stage", "written_at", "engagement_score", "performance_chain"} {
		ok, err := columnExists(db, "player_match_enrichment", col)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Errorf("colonne %q absente après migration", col)
		}
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("rows = %d, want 2 (préservées en legacy)", n)
	}

	// La vue reflète l'état legacy.
	var perf float64
	var sess string
	if err := db.QueryRow(`SELECT performance_score, session_id FROM player_match_enrichment_latest WHERE match_id='m1'`).Scan(&perf, &sess); err != nil {
		t.Fatalf("query view: %v", err)
	}
	if perf != 1.5 || sess != "sessA" {
		t.Errorf("vue legacy m1 = (%v, %q), want (1.5, sessA)", perf, sess)
	}
}

// TestMatchEnrichmentAppendOnly_MergePartialWrites — LE test du merge-on-read :
// 3 INSERT partiels par étape (perf / session / engagement) d'un même match →
// la vue reconstitue les 3 colonnes simultanément.
func TestMatchEnrichmentAppendOnly_MergePartialWrites(t *testing.T) {
	db := setupLegacyMatchEnrichment(t, nil)
	if err := applyAppendOnlyMatchEnrichment(db); err != nil {
		t.Fatalf("apply: %v", err)
	}

	stmts := []string{
		`INSERT INTO player_match_enrichment (match_id, performance_score, stage, written_at) VALUES ('mX', 2.5, 'perf', TIMESTAMP '2026-01-01 10:00:00')`,
		`INSERT INTO player_match_enrichment (match_id, session_id, session_label, stage, written_at) VALUES ('mX', 'sess1', 'Session 1', 'session', TIMESTAMP '2026-01-01 10:01:00')`,
		`INSERT INTO player_match_enrichment (match_id, engagement_score, engagement_score_brut, mode_category, stage, written_at) VALUES ('mX', 0.8, 0.5, 'PvP_ranked', 'engagement', TIMESTAMP '2026-01-01 10:02:00')`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("insert partial: %v\nSQL: %s", err, s)
		}
	}

	var perf, eng float64
	var sess, mode string
	if err := db.QueryRow(`
		SELECT performance_score, session_id, engagement_score, mode_category
		FROM player_match_enrichment_latest WHERE match_id='mX'
	`).Scan(&perf, &sess, &eng, &mode); err != nil {
		t.Fatalf("query merged: %v", err)
	}
	if perf != 2.5 {
		t.Errorf("performance_score = %v, want 2.5 (étape perf)", perf)
	}
	if sess != "sess1" {
		t.Errorf("session_id = %q, want sess1 (étape session)", sess)
	}
	if eng != 0.8 {
		t.Errorf("engagement_score = %v, want 0.8 (étape engagement)", eng)
	}
	if mode != "PvP_ranked" {
		t.Errorf("mode_category = %q, want PvP_ranked", mode)
	}

	// 1 seule ligne logique par match_id dans la vue.
	var rows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment_latest WHERE match_id='mX'`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 1 {
		t.Errorf("vue rows pour mX = %d, want 1", rows)
	}
}

// TestMatchEnrichmentAppendOnly_NullReset — reset légitime à NULL préservé :
// engagement_score=0.9 puis une version engagement sans engagement_score (insufficient_history)
// → la vue rend NULL (PAS l'ancienne valeur). C'est le piège du last_value IGNORE NULLS.
func TestMatchEnrichmentAppendOnly_NullReset(t *testing.T) {
	db := setupLegacyMatchEnrichment(t, nil)
	if err := applyAppendOnlyMatchEnrichment(db); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO player_match_enrichment (match_id, engagement_score, engagement_score_brut, stage, written_at) VALUES ('mR', 0.9, 0.5, 'engagement', TIMESTAMP '2026-01-01 10:00:00')`); err != nil {
		t.Fatalf("insert v1: %v", err)
	}
	// v2 : insufficient_history → engagement_score omis (NULL), brut+confidence écrits.
	if _, err := db.Exec(`INSERT INTO player_match_enrichment (match_id, engagement_score_brut, engagement_score_confidence, stage, written_at) VALUES ('mR', 0.4, 'insufficient_history', 'engagement', TIMESTAMP '2026-01-01 10:01:00')`); err != nil {
		t.Fatalf("insert v2: %v", err)
	}
	var eng sql.NullFloat64
	var conf sql.NullString
	if err := db.QueryRow(`SELECT engagement_score, engagement_score_confidence FROM player_match_enrichment_latest WHERE match_id='mR'`).Scan(&eng, &conf); err != nil {
		t.Fatal(err)
	}
	if eng.Valid {
		t.Errorf("engagement_score = %v, want NULL (reset insufficient_history préservé)", eng.Float64)
	}
	if !conf.Valid || conf.String != "insufficient_history" {
		t.Errorf("engagement_score_confidence = %v, want insufficient_history", conf)
	}
}

// TestMatchEnrichmentAppendOnly_BooleanToggle — toggle bidirectionnel : is_excluded
// TRUE puis FALSE → la vue rend FALSE (dernière valeur explicite gagne).
func TestMatchEnrichmentAppendOnly_BooleanToggle(t *testing.T) {
	db := setupLegacyMatchEnrichment(t, nil)
	if err := applyAppendOnlyMatchEnrichment(db); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO player_match_enrichment (match_id, is_excluded, stage, written_at) VALUES ('mB', TRUE, 'exclusion', TIMESTAMP '2026-01-01 10:00:00')`); err != nil {
		t.Fatalf("exclude: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO player_match_enrichment (match_id, is_excluded, stage, written_at) VALUES ('mB', FALSE, 'exclusion', TIMESTAMP '2026-01-01 10:01:00')`); err != nil {
		t.Fatalf("un-exclude: %v", err)
	}
	var excl bool
	if err := db.QueryRow(`SELECT is_excluded FROM player_match_enrichment_latest WHERE match_id='mB'`).Scan(&excl); err != nil {
		t.Fatal(err)
	}
	if excl {
		t.Error("is_excluded = TRUE, want FALSE (un-exclude doit gagner)")
	}

	// match jamais touché par exclusion → COALESCE FALSE.
	if _, err := db.Exec(`INSERT INTO player_match_enrichment (match_id, performance_score, stage, written_at) VALUES ('mNoExcl', 1.0, 'perf', TIMESTAMP '2026-01-01 10:00:00')`); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT is_excluded FROM player_match_enrichment_latest WHERE match_id='mNoExcl'`).Scan(&excl); err != nil {
		t.Fatal(err)
	}
	if excl {
		t.Error("is_excluded = TRUE pour match jamais exclu, want FALSE (COALESCE)")
	}
}

// TestMatchEnrichmentAppendOnly_LegacyOverride — un stage-row override le socle legacy
// pour SA colonne, sans toucher les autres (qui restent au legacy).
func TestMatchEnrichmentAppendOnly_LegacyOverride(t *testing.T) {
	db := setupLegacyMatchEnrichment(t, func(db *sql.DB) {
		if _, err := db.Exec(`INSERT INTO player_match_enrichment (match_id, performance_score, session_id) VALUES ('mL', 1.0, 'oldSess')`); err != nil {
			t.Fatalf("seed: %v", err)
		}
	})
	if err := applyAppendOnlyMatchEnrichment(db); err != nil {
		t.Fatalf("apply: %v", err)
	}
	// Override perf seulement (étape perf), plus récent que le legacy.
	if _, err := db.Exec(`INSERT INTO player_match_enrichment (match_id, performance_score, stage, written_at) VALUES ('mL', 9.0, 'perf', TIMESTAMP '2030-01-01 10:00:00')`); err != nil {
		t.Fatalf("override: %v", err)
	}
	var perf float64
	var sess string
	if err := db.QueryRow(`SELECT performance_score, session_id FROM player_match_enrichment_latest WHERE match_id='mL'`).Scan(&perf, &sess); err != nil {
		t.Fatal(err)
	}
	if perf != 9.0 {
		t.Errorf("performance_score = %v, want 9.0 (override perf)", perf)
	}
	if sess != "oldSess" {
		t.Errorf("session_id = %q, want oldSess (legacy non touché)", sess)
	}
}

// TestMatchEnrichmentAppendOnly_LiveBaseline — le collect live écrit UNE row
// multi-colonnes stage='live' (baseline pour les matchs collectés après migration).
// La vue lit cette baseline pour toute colonne sans owner-stage ; un writer owner-stage
// override SA colonne sans toucher les autres (qui restent au live). live > legacy.
func TestMatchEnrichmentAppendOnly_LiveBaseline(t *testing.T) {
	db := setupLegacyMatchEnrichment(t, nil)
	if err := applyAppendOnlyMatchEnrichment(db); err != nil {
		t.Fatalf("apply: %v", err)
	}
	// Collect live : 1 row stage='live' portant perf + session + teammates.
	if _, err := db.Exec(`INSERT INTO player_match_enrichment (match_id, performance_score, session_id, session_label, teammates_signature, stage, written_at) VALUES ('mLive', 3.0, 'sessLive', 'Live Session', 'sigLive', 'live', TIMESTAMP '2026-02-01 10:00:00')`); err != nil {
		t.Fatalf("live insert: %v", err)
	}
	// La vue lit la baseline live pour les 3 colonnes.
	var perf float64
	var sess, sig string
	if err := db.QueryRow(`SELECT performance_score, session_id, teammates_signature FROM player_match_enrichment_latest WHERE match_id='mLive'`).Scan(&perf, &sess, &sig); err != nil {
		t.Fatalf("query live baseline: %v", err)
	}
	if perf != 3.0 || sess != "sessLive" || sig != "sigLive" {
		t.Errorf("baseline live = (%v, %q, %q), want (3.0, sessLive, sigLive)", perf, sess, sig)
	}

	// Owner-stage perf override (plus récent) : perf change, session/teammates restent live.
	if _, err := db.Exec(`INSERT INTO player_match_enrichment (match_id, performance_score, stage, written_at) VALUES ('mLive', 9.5, 'perf', TIMESTAMP '2026-02-01 11:00:00')`); err != nil {
		t.Fatalf("perf override: %v", err)
	}
	if err := db.QueryRow(`SELECT performance_score, session_id, teammates_signature FROM player_match_enrichment_latest WHERE match_id='mLive'`).Scan(&perf, &sess, &sig); err != nil {
		t.Fatalf("query after override: %v", err)
	}
	if perf != 9.5 {
		t.Errorf("performance_score = %v, want 9.5 (owner-stage perf override)", perf)
	}
	if sess != "sessLive" || sig != "sigLive" {
		t.Errorf("session/teammates = (%q, %q), want (sessLive, sigLive) (live non touché)", sess, sig)
	}

	// live > legacy : un match avec legacy ET live → live gagne sur la baseline.
	if _, err := db.Exec(`INSERT INTO player_match_enrichment (match_id, performance_score, session_id) VALUES ('mBoth', 1.0, 'oldLegacy')`); err != nil {
		t.Fatalf("legacy seed: %v", err)
	}
	// La row ci-dessus est stage='legacy' par DEFAULT. Ajoute une row live plus complète.
	if _, err := db.Exec(`INSERT INTO player_match_enrichment (match_id, session_id, stage, written_at) VALUES ('mBoth', 'newLive', 'live', TIMESTAMP '2026-02-01 12:00:00')`); err != nil {
		t.Fatalf("live over legacy: %v", err)
	}
	var sessBoth string
	if err := db.QueryRow(`SELECT session_id FROM player_match_enrichment_latest WHERE match_id='mBoth'`).Scan(&sessBoth); err != nil {
		t.Fatalf("query mBoth: %v", err)
	}
	if sessBoth != "newLive" {
		t.Errorf("session_id = %q, want newLive (live > legacy)", sessBoth)
	}
}

// TestMatchEnrichmentAppendOnly_Idempotent — 2e apply = no-op, rows préservées, vue OK.
func TestMatchEnrichmentAppendOnly_Idempotent(t *testing.T) {
	db := setupLegacyMatchEnrichment(t, func(db *sql.DB) {
		_, _ = db.Exec(`INSERT INTO player_match_enrichment (match_id, performance_score) VALUES ('m1', 1.0), ('m2', 2.0)`)
	})
	if err := applyAppendOnlyMatchEnrichment(db); err != nil {
		t.Fatalf("apply #1: %v", err)
	}
	var n1 int
	_ = db.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment`).Scan(&n1)
	if err := applyAppendOnlyMatchEnrichment(db); err != nil {
		t.Fatalf("apply #2: %v", err)
	}
	var n2 int
	_ = db.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment`).Scan(&n2)
	if n2 != n1 {
		t.Errorf("count change après re-apply: %d -> %d", n1, n2)
	}
	// vue toujours fonctionnelle.
	var v int
	if err := db.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment_latest`).Scan(&v); err != nil {
		t.Errorf("vue cassée après 2e apply: %v", err)
	}
}

// TestMatchEnrichmentAppendOnly_OrphanRecovery — __appendonly orphelin (crash mid-swap)
// récupéré en tête, puis rebuild complet.
func TestMatchEnrichmentAppendOnly_OrphanRecovery(t *testing.T) {
	db := setupLegacyMatchEnrichment(t, func(db *sql.DB) {
		_, _ = db.Exec(`INSERT INTO player_match_enrichment (match_id, performance_score) VALUES ('m1', 1.0), ('m2', 2.0), ('m3', 3.0)`)
	})
	if _, err := db.Exec(`CREATE TABLE player_match_enrichment__appendonly AS SELECT * FROM player_match_enrichment`); err != nil {
		t.Fatalf("create orphan: %v", err)
	}
	if _, err := db.Exec(`DROP TABLE player_match_enrichment`); err != nil {
		t.Fatalf("drop main: %v", err)
	}
	if err := applyAppendOnlyMatchEnrichment(db); err != nil {
		t.Fatalf("apply (recovery): %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("rows après recovery = %d, want 3", n)
	}
	hasID, _ := columnExists(db, "player_match_enrichment", "id")
	if !hasID {
		t.Error("id absent après recovery+rebuild")
	}
}

// TestMatchEnrichmentAppendOnly_NoTable_NoOp — DB sans table → no-op.
func TestMatchEnrichmentAppendOnly_NoTable_NoOp(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := applyAppendOnlyMatchEnrichment(db); err != nil {
		t.Errorf("no-op attendu sans table, got: %v", err)
	}
}
