package service

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/sync"
)

const (
	postImportTestXUID     = "2533274823110022"
	postImportTestGamertag = "TestOwner"
)

// postImportTestEnv bundles the dependencies the post-import tests share.
type postImportTestEnv struct {
	cfg      *config.AppConfig
	sharedDB *sql.DB
}

// setupPostImportEnv creates a temp repo root, opens shared + metadata
// DuckDBs with the schemas the recompute stages query. The player DB is
// NOT created here — the post-import service creates it itself via
// sync.OpenPlayerDB the first time it runs.
func setupPostImportEnv(t *testing.T) *postImportTestEnv {
	t.Helper()
	tempDir := t.TempDir()
	cfg := &config.AppConfig{RepoRoot: tempDir, DemoMode: false}

	// Shared DuckDB.
	sharedPath := config.SharedDBPath(cfg, "")
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0o755); err != nil {
		t.Fatalf("mkdir shared: %v", err)
	}
	sharedDB, err := sql.Open("duckdb", sharedPath)
	if err != nil {
		t.Fatalf("open shared: %v", err)
	}
	t.Cleanup(func() { _ = sharedDB.Close() })
	if err := sync.EnsureSharedSchema(sharedDB); err != nil {
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	if _, err := sharedDB.Exec(`
		CREATE SEQUENCE IF NOT EXISTS highlight_events_id_seq;
		CREATE TABLE IF NOT EXISTS highlight_events (
			id INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
			match_id VARCHAR NOT NULL, event_type VARCHAR NOT NULL,
			time_ms INTEGER, xuid VARCHAR, type_hint INTEGER, raw_json VARCHAR
		);
	`); err != nil {
		t.Fatalf("create highlight_events: %v", err)
	}

	// Metadata DuckDB (schema only — leaving citation_mappings empty makes
	// BackfillMatchCitations return nil after the "no mappings" short-circuit).
	metaPath := titlePkg.NewPathResolver(cfg.RepoRoot).MetadataDBPath(titlePkg.DefaultSlug)
	if err := os.MkdirAll(filepath.Dir(metaPath), 0o755); err != nil {
		t.Fatalf("mkdir meta: %v", err)
	}
	metaDB, err := sql.Open("duckdb", metaPath)
	if err != nil {
		t.Fatalf("open meta: %v", err)
	}
	if _, err := metaDB.Exec(`
		CREATE TABLE IF NOT EXISTS citation_mappings (
			citation_name_norm    VARCHAR,
			citation_name_display VARCHAR,
			mapping_type          VARCHAR,
			category              VARCHAR,
			medal_id              BIGINT,
			medal_ids             VARCHAR,
			stat_name             VARCHAR,
			award_name            VARCHAR,
			custom_function       VARCHAR,
			composite_children    VARCHAR,
			tier_targets          VARCHAR,
			enabled               BOOLEAN DEFAULT TRUE
		);
		CREATE TABLE IF NOT EXISTS weapon_labels (weapon_id UBIGINT, label VARCHAR);
	`); err != nil {
		t.Fatalf("create metadata schema: %v", err)
	}
	_ = metaDB.Close()

	return &postImportTestEnv{cfg: cfg, sharedDB: sharedDB}
}

// insertTestMatch writes the minimum match_registry + match_participants
// rows the recompute stages will find when scanning by XUID.
func insertTestMatch(t *testing.T, db *sql.DB, matchID, xuid string, start time.Time) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO match_registry (
			match_id, start_time, end_time, start_time_utc, end_time_utc,
			mode_category, duration_seconds, playable_duration_seconds,
			first_sync_by, first_sync_at, last_updated_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, 'Other', 600, 600, 'test', ?, ?, ?, ?)`,
		matchID, start, start.Add(10*time.Minute), start.UTC(), start.UTC().Add(10*time.Minute),
		time.Now().UTC(), time.Now().UTC(), time.Now().UTC(), time.Now().UTC(),
	); err != nil {
		t.Fatalf("insert match_registry: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO match_participants (
			match_id, xuid, gamertag, team_id, outcome, rank,
			kills, deaths, assists, score
		) VALUES (?, ?, ?, 0, 2, 1, 10, 5, 2, 1000)`,
		matchID, xuid, postImportTestGamertag,
	); err != nil {
		t.Fatalf("insert match_participants: %v", err)
	}
}

// ─── TESTS ────────────────────────────────────────────────────────────────────

func TestPostImport_NewService_DefaultLoggerNotNil(t *testing.T) {
	svc := NewOpenSpartanPostImportService(&config.AppConfig{})
	if svc.log == nil {
		t.Error("default logger should be slog.Default(), got nil")
	}
}

func TestPostImport_ApplyDefaults_FillsZeroValues(t *testing.T) {
	got := applyPostImportDefaults(PostImportOptions{})
	if got.TitleSlug != titlePkg.DefaultSlug {
		t.Errorf("TitleSlug default: want %s, got %s", titlePkg.DefaultSlug, got.TitleSlug)
	}
	if got.SessionGapMinutes != 15 {
		t.Errorf("SessionGapMinutes default: want 15, got %d", got.SessionGapMinutes)
	}
}

func TestPostImport_ApplyDefaults_PreservesExplicitValues(t *testing.T) {
	in := PostImportOptions{
		TitleSlug:         "halo_5",
		SessionGapMinutes: 30,
		ForcePerfScores:   true,
	}
	got := applyPostImportDefaults(in)
	if got.TitleSlug != "halo_5" {
		t.Errorf("TitleSlug: want preserved 'halo_5', got %s", got.TitleSlug)
	}
	if got.SessionGapMinutes != 30 {
		t.Errorf("SessionGapMinutes: want preserved 30, got %d", got.SessionGapMinutes)
	}
	if !got.ForcePerfScores {
		t.Error("ForcePerfScores: want preserved true")
	}
}

func TestPostImport_ApplyDefaults_RejectsNegativeGap(t *testing.T) {
	got := applyPostImportDefaults(PostImportOptions{SessionGapMinutes: -5})
	if got.SessionGapMinutes != 15 {
		t.Errorf("negative gap should fall back to 15, got %d", got.SessionGapMinutes)
	}
}

func TestPostImport_Run_RejectsEmptyXUID(t *testing.T) {
	env := setupPostImportEnv(t)
	svc := NewOpenSpartanPostImportService(env.cfg)

	_, err := svc.Run(context.Background(), "", postImportTestGamertag, nil, PostImportOptions{})
	if err == nil {
		t.Fatal("expected error on empty xuid")
	}
}

func TestPostImport_Run_RejectsEmptyGamertag(t *testing.T) {
	env := setupPostImportEnv(t)
	svc := NewOpenSpartanPostImportService(env.cfg)

	_, err := svc.Run(context.Background(), postImportTestXUID, "", nil, PostImportOptions{})
	if err == nil {
		t.Fatal("expected error on empty gamertag")
	}
}

func TestPostImport_Run_NoMatchIDsSkipsCitations(t *testing.T) {
	env := setupPostImportEnv(t)
	svc := NewOpenSpartanPostImportService(env.cfg)

	result, err := svc.Run(context.Background(), postImportTestXUID, postImportTestGamertag, nil, PostImportOptions{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.CitationsBackfilled {
		t.Error("CitationsBackfilled should be false when matchIDs is empty")
	}
	// With no matches in shared, sessions/perf return 0 — that's OK, not an error.
	if result.SessionsTouched != 0 {
		t.Errorf("SessionsTouched: want 0 (no matches), got %d", result.SessionsTouched)
	}
}

func TestPostImport_Run_CreatesPlayerDBOnFirstCall(t *testing.T) {
	env := setupPostImportEnv(t)
	svc := NewOpenSpartanPostImportService(env.cfg)

	// Verify the player DB does NOT exist before the run.
	playerPath := config.PlayerDBPath(env.cfg, "", postImportTestGamertag)
	if _, err := os.Stat(playerPath); err == nil {
		t.Fatalf("player DB should not exist before the test, found %s", playerPath)
	}

	if _, err := svc.Run(context.Background(), postImportTestXUID, postImportTestGamertag, nil, PostImportOptions{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Now it should exist.
	if _, err := os.Stat(playerPath); err != nil {
		t.Errorf("player DB should be created by the run at %s: %v", playerPath, err)
	}
}

func TestPostImport_Run_RecomputesAllStagesWithMatches(t *testing.T) {
	env := setupPostImportEnv(t)
	// Insert 3 matches in shared so the recompute stages have something to scan.
	insertTestMatch(t, env.sharedDB, "post-m1", postImportTestXUID, time.Now().Add(-3*time.Hour))
	insertTestMatch(t, env.sharedDB, "post-m2", postImportTestXUID, time.Now().Add(-2*time.Hour))
	insertTestMatch(t, env.sharedDB, "post-m3", postImportTestXUID, time.Now().Add(-1*time.Hour))

	svc := NewOpenSpartanPostImportService(env.cfg)
	matchIDs := []string{"post-m1", "post-m2", "post-m3"}
	result, err := svc.Run(context.Background(), postImportTestXUID, postImportTestGamertag, matchIDs, PostImportOptions{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, e := range result.Errors {
		t.Logf("stage %s: %s", e.Stage, e.Err)
	}

	// Sessions: 3 matches sufficiently spaced (1h apart, gap=15min default)
	// → 3 distinct sessions written. Verify >0.
	if result.SessionsTouched <= 0 {
		t.Errorf("SessionsTouched: want >0 (3 matches inserted), got %d", result.SessionsTouched)
	}
	// PerfScores: only 3 matches in arena_slayer chain, below the 10-match
	// threshold → batch returns 0 updated. That's the expected behaviour.
	if result.PerfScoresTouched != 0 {
		t.Errorf("PerfScoresTouched: want 0 (below 10-match threshold), got %d", result.PerfScoresTouched)
	}
	// Citations: empty mappings → backfill is a no-op but returns nil error.
	if !result.CitationsBackfilled {
		t.Error("CitationsBackfilled should be true even with empty mappings (no-op success)")
	}
}

func TestPostImport_PostImportError_StructFields(t *testing.T) {
	// Sanity-check the public error struct fields stay stable (consumed by the
	// handler when building the job Result).
	e := PostImportError{Stage: "sessions", MatchID: "m-1", Err: "boom"}
	if e.Stage != "sessions" || e.MatchID != "m-1" || e.Err != "boom" {
		t.Errorf("unexpected fields: %+v", e)
	}
}

func TestPostImport_Run_EnsuresPlayerEnrichmentRowsAreCreated(t *testing.T) {
	env := setupPostImportEnv(t)
	matchIDs := []string{"prime-m1", "prime-m2", "prime-m3"}
	// Pre-insert the matches in shared so RecalculatePlayerSessions has data.
	for i, mid := range matchIDs {
		insertTestMatch(t, env.sharedDB, mid, postImportTestXUID,
			time.Now().Add(time.Duration(-3+i)*time.Hour))
	}

	svc := NewOpenSpartanPostImportService(env.cfg)
	result, err := svc.Run(context.Background(), postImportTestXUID, postImportTestGamertag, matchIDs, PostImportOptions{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, e := range result.Errors {
		t.Logf("stage %s match %s: %s", e.Stage, e.MatchID, e.Err)
	}

	// Open the player DB and verify the enrichment rows landed.
	playerPath := config.PlayerDBPath(env.cfg, "", postImportTestGamertag)
	playerDB, err := sql.Open("duckdb", playerPath+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open player DB: %v", err)
	}
	defer playerDB.Close()
	for _, mid := range matchIDs {
		var n int
		if err := playerDB.QueryRow(
			`SELECT COUNT(*) FROM player_match_enrichment WHERE match_id = ?`, mid,
		).Scan(&n); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if n != 1 {
			t.Errorf("enrichment row for %s: want 1, got %d", mid, n)
		}
	}
}

func TestPostImport_Run_EnsurePrimerIdempotent(t *testing.T) {
	env := setupPostImportEnv(t)
	matchIDs := []string{"idem-m1", "idem-m2"}
	for _, mid := range matchIDs {
		insertTestMatch(t, env.sharedDB, mid, postImportTestXUID, time.Now().Add(-time.Hour))
	}
	svc := NewOpenSpartanPostImportService(env.cfg)

	// 1st run primes the rows.
	if _, err := svc.Run(context.Background(), postImportTestXUID, postImportTestGamertag, matchIDs, PostImportOptions{}); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	// 2nd run must NOT error on the ON CONFLICT DO NOTHING path.
	result, err := svc.Run(context.Background(), postImportTestXUID, postImportTestGamertag, matchIDs, PostImportOptions{})
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	for _, e := range result.Errors {
		if e.Stage == "prime_enrichment" {
			t.Errorf("idempotent primer should not record errors: %+v", e)
		}
	}
	// And the row count must still be exactly len(matchIDs).
	playerPath := config.PlayerDBPath(env.cfg, "", postImportTestGamertag)
	playerDB, _ := sql.Open("duckdb", playerPath+"?access_mode=read_only")
	defer playerDB.Close()
	var n int
	playerDB.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment WHERE match_id IN ('idem-m1', 'idem-m2')`).Scan(&n)
	if n != 2 {
		t.Errorf("expected 2 enrichment rows after 2 idempotent primers, got %d", n)
	}
}
