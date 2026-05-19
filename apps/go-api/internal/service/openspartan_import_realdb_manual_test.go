package service

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/openspartan"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync"
)

// TestRealDB_FullPipeline runs the complete OpenSpartan pipeline against a
// real database: import → DuckDB writes → post-import recompute (sessions /
// perf_score / citations). Skipped without OPENSPARTAN_DB_PATH so CI stays
// green.
//
// Run with:
//
//	OPENSPARTAN_DB_PATH=/path/to/{xuid}.db \
//	OPENSPARTAN_OWNER_XUID=2533274823110022 \
//	OPENSPARTAN_OWNER_GAMERTAG=YourGamertag \
//	go test ./internal/service/ -run TestRealDB_FullPipeline -v
func TestRealDB_FullPipeline(t *testing.T) {
	path := os.Getenv("OPENSPARTAN_DB_PATH")
	if path == "" {
		t.Skip("OPENSPARTAN_DB_PATH not set; skipping full-pipeline smoke test")
	}
	owner := os.Getenv("OPENSPARTAN_OWNER_XUID")
	if owner == "" {
		t.Skip("OPENSPARTAN_OWNER_XUID not set; skipping full-pipeline smoke test")
	}
	gamertag := os.Getenv("OPENSPARTAN_OWNER_GAMERTAG")
	if gamertag == "" {
		gamertag = "SmokeTestOwner"
	}

	// Build a self-contained temp repo: shared DuckDB + metadata DuckDB +
	// player DBs all live under tempDir, so the post-import service can
	// create the player DB on first call without polluting real data.
	tempDir := t.TempDir()
	cfg := &config.AppConfig{RepoRoot: tempDir, DemoMode: false}

	sharedPath := config.SharedDBPath(cfg, "")
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0o755); err != nil {
		t.Fatalf("mkdir shared: %v", err)
	}
	// Open through the process-level RW cache so the post-import service
	// shares this single connection rather than trying to open a second
	// exclusive RW handle (which would block on DuckDB's exclusive lock).
	sharedHandle, err := platform_duckdb.OpenReadWriteShared(sharedPath)
	if err != nil {
		t.Fatalf("OpenReadWriteShared: %v", err)
	}
	t.Cleanup(func() { sharedHandle.Close() })
	sharedDB := sharedHandle.SQLDb()
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

	metaPath := titlePkg.NewPathResolver(cfg.RepoRoot).MetadataDBPath(titlePkg.DefaultSlug)
	if err := os.MkdirAll(filepath.Dir(metaPath), 0o755); err != nil {
		t.Fatalf("mkdir meta: %v", err)
	}
	// Same caching rationale as shared: post-import re-opens metadata via
	// OpenReadWriteShared, so we go through the cache here to share the
	// single RW handle.
	metaHandle, err := platform_duckdb.OpenReadWriteShared(metaPath)
	if err != nil {
		t.Fatalf("OpenReadWriteShared meta: %v", err)
	}
	t.Cleanup(func() { metaHandle.Close() })
	if _, err := metaHandle.SQLDb().Exec(`
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
		t.Fatalf("create metadata: %v", err)
	}

	importSvc := NewOpenSpartanImportService(sharedDB, "")
	postImportSvc := NewOpenSpartanPostImportService(cfg, sharedDB)

	// ─── STAGE 1 — IMPORT ────────────────────────────────────────────────────
	t.Log("=== STAGE 1: Import (raw rows into shared DuckDB) ===")
	importResult, err := importSvc.Import(context.Background(), owner, path, ImportOptions{
		Source:   "smoke_test_full_pipeline",
		StashDir: filepath.Join(tempDir, "players"),
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	t.Logf("  detected_owner_xuid : %s", importResult.DetectedOwnerXUID)
	t.Logf("  confidence          : %s", importResult.Confidence)
	t.Logf("  inserted_matches    : %d / %d", importResult.InsertedMatches, importResult.TotalMatches)
	t.Logf("  inserted_participants: %d", importResult.InsertedParticipants)
	t.Logf("  inserted_medals     : %d", importResult.InsertedMedals)
	t.Logf("  inserted_highlights : %d", importResult.InsertedHighlights)
	t.Logf("  inserted_aliases    : %d", importResult.InsertedAliases)
	t.Logf("  stashed_friends     : %d", importResult.StashedFriends)
	t.Logf("  errors              : %d", len(importResult.Errors))
	if importResult.InsertedMatches == 0 {
		t.Fatal("expected at least one match inserted")
	}

	// ─── STAGE 2 — POST-IMPORT RECOMPUTE ─────────────────────────────────────
	t.Log("=== STAGE 2: Post-import recompute (sessions / perf_score / citations) ===")
	postResult, err := postImportSvc.Run(context.Background(), owner, gamertag, importResult.InsertedMatchIDs, PostImportOptions{})
	if err != nil {
		t.Fatalf("PostImport.Run: %v", err)
	}
	t.Logf("  sessions_touched     : %d", postResult.SessionsTouched)
	t.Logf("  perf_scores_touched  : %d", postResult.PerfScoresTouched)
	t.Logf("  citations_backfilled : %v", postResult.CitationsBackfilled)
	t.Logf("  errors               : %d", len(postResult.Errors))
	for i, e := range postResult.Errors {
		if i >= 5 {
			t.Logf("    ... (%d more)", len(postResult.Errors)-5)
			break
		}
		t.Logf("    [%s/%s] %s", e.Stage, e.MatchID, e.Err)
	}

	// ─── STAGE 3 — VERIFY PLAYER DB WAS CREATED + ENRICHMENT ROWS PRIMED ─────
	t.Log("=== STAGE 3: Verify player_match_enrichment populated ===")
	playerPath := config.PlayerDBPath(cfg, "", gamertag)
	playerDB, err := sql.Open("duckdb", playerPath+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open player DB: %v", err)
	}
	defer playerDB.Close()

	var nEnrich, nWithSession, nWithPerf int
	playerDB.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment`).Scan(&nEnrich)
	playerDB.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment WHERE session_id IS NOT NULL`).Scan(&nWithSession)
	playerDB.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment WHERE performance_score IS NOT NULL`).Scan(&nWithPerf)
	t.Logf("  total_enrichment_rows     : %d", nEnrich)
	t.Logf("  rows_with_session_id      : %d", nWithSession)
	t.Logf("  rows_with_performance_score: %d", nWithPerf)

	if nEnrich != importResult.InsertedMatches {
		t.Errorf("enrichment rows count (%d) should match inserted_matches (%d)", nEnrich, importResult.InsertedMatches)
	}
	// Sessions: must be populated since we have 100+ matches well spread in time.
	if nWithSession == 0 {
		t.Error("expected at least one row with session_id populated")
	}
	// Perf scores: only filled once a chain has 10+ matches; with 451 matches
	// from a real account, at least some chains should cross the threshold.
	if importResult.InsertedMatches > 50 && nWithPerf == 0 {
		t.Logf("note: 0 perf_scores even with %d matches — may be expected if all matches are in low-volume chains", importResult.InsertedMatches)
	}
	// Owner is in match_participants of shared.
	var ownerHits int
	sharedDB.QueryRow(`SELECT COUNT(*) FROM match_participants WHERE xuid = ?`, owner).Scan(&ownerHits)
	if ownerHits == 0 {
		t.Error("owner XUID never appeared in match_participants")
	}
	_ = openspartan.ConfidenceHigh // keep openspartan import referenced
}

// TestRealDB_OpenSpartanImport_DryRun runs the import in DryRun mode against
// a real OpenSpartan database. NO writes happen — the service walks every
// match, maps it, but stops short of inserting in DuckDB. Useful to validate
// the file end-to-end without side effects.
//
// Run with:
//
//	OPENSPARTAN_DB_PATH=/path/to/{xuid}.db \
//	OPENSPARTAN_OWNER_XUID=2533274823110022 \
//	go test ./internal/service/ -run TestRealDB_OpenSpartanImport_DryRun -v
func TestRealDB_OpenSpartanImport_DryRun(t *testing.T) {
	path := os.Getenv("OPENSPARTAN_DB_PATH")
	if path == "" {
		t.Skip("OPENSPARTAN_DB_PATH not set; skipping dry-run smoke test")
	}
	owner := os.Getenv("OPENSPARTAN_OWNER_XUID")
	if owner == "" {
		t.Skip("OPENSPARTAN_OWNER_XUID not set; skipping dry-run smoke test")
	}

	// Open the OpenSpartan db directly to surface DetectOwner details that
	// would otherwise be swallowed by the service.
	r, err := openspartan.Open(path)
	if err != nil {
		t.Fatalf("openspartan.Open: %v", err)
	}
	defer r.Close()
	xuid, conf, err := r.DetectOwner(context.Background(), path)
	if err != nil {
		t.Fatalf("DetectOwner: %v", err)
	}
	t.Logf("DetectOwner: xuid=%s confidence=%s (session expected=%s)", xuid, conf, owner)
	count, _ := r.MatchCount(context.Background())
	hl, _ := r.HighlightCount(context.Background())
	aliases, _ := r.LoadXuidAliases(context.Background())
	friends, _ := r.LoadFriends(context.Background())
	t.Logf("Source counts: matches=%d highlights=%d aliases=%d friends=%d",
		count, hl, len(aliases), len(friends))

	// Run the service in DryRun mode — sharedDB is unused (no writes).
	svc := NewOpenSpartanImportService(nil, "")
	result, err := svc.Import(context.Background(), owner, path, ImportOptions{
		DryRun:   true,
		Source:   "dry_run",
		StashDir: t.TempDir(), // would be ignored but kept consistent
	})
	if err != nil {
		t.Fatalf("Import DryRun: %v", err)
	}

	t.Logf("DryRun result:")
	t.Logf("  detected_owner_xuid : %s", result.DetectedOwnerXUID)
	t.Logf("  confidence          : %s", result.Confidence)
	t.Logf("  total_matches       : %d", result.TotalMatches)
	t.Logf("  WOULD insert :")
	t.Logf("    matches           : %d", result.InsertedMatches)
	t.Logf("    participants      : %d", result.InsertedParticipants)
	t.Logf("    medals            : %d", result.InsertedMedals)
	t.Logf("    highlights        : %d", result.InsertedHighlights)
	t.Logf("    xuid_aliases      : %d", result.InsertedAliases)
	t.Logf("    stashed_friends   : %d", result.StashedFriends)
	t.Logf("  errors              : %d", len(result.Errors))
	for i, e := range result.Errors {
		if i >= 5 {
			t.Logf("    ... (%d more errors)", len(result.Errors)-5)
			break
		}
		t.Logf("    [%s/%s] %s", e.Stage, e.MatchID, e.Err)
	}
	if result.InsertedMatches == 0 {
		t.Fatal("DryRun should at least count >=1 match on a real database")
	}
}

// TestRealDB_OpenSpartanImport is a manual end-to-end smoke test against a
// real OpenSpartan database, importing into a temporary DuckDB. Skipped
// without OPENSPARTAN_DB_PATH so CI stays green.
//
// Run locally with:
//
//	OPENSPARTAN_DB_PATH=/path/to/{xuid}.db \
//	OPENSPARTAN_OWNER_XUID=2533274823110022 \
//	go test ./internal/service/ -run TestRealDB_OpenSpartanImport -v
func TestRealDB_OpenSpartanImport(t *testing.T) {
	path := os.Getenv("OPENSPARTAN_DB_PATH")
	if path == "" {
		t.Skip("OPENSPARTAN_DB_PATH not set; skipping manual smoke test")
	}
	owner := os.Getenv("OPENSPARTAN_OWNER_XUID")
	if owner == "" {
		t.Skip("OPENSPARTAN_OWNER_XUID not set; skipping manual smoke test")
	}

	sharedDB := setupSharedDB(t)
	dir := t.TempDir()
	svc := NewOpenSpartanImportService(sharedDB, "")

	result, err := svc.Import(context.Background(), owner, path, ImportOptions{
		Source:   "smoke_test",
		StashDir: filepath.Join(dir, "players"),
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	t.Logf("Result: detected=%s confidence=%s total=%d inserted_matches=%d inserted_participants=%d inserted_medals=%d inserted_highlights=%d inserted_aliases=%d stashed_friends=%d errors=%d",
		result.DetectedOwnerXUID, result.Confidence,
		result.TotalMatches, result.InsertedMatches, result.InsertedParticipants,
		result.InsertedMedals, result.InsertedHighlights,
		result.InsertedAliases, result.StashedFriends, len(result.Errors))

	for _, e := range result.Errors {
		t.Logf("  error stage=%s match=%s err=%s", e.Stage, e.MatchID, e.Err)
	}

	if result.InsertedMatches == 0 {
		t.Fatal("expected at least one match inserted")
	}
	if float64(result.InsertedMatches) < 0.95*float64(result.TotalMatches) {
		t.Errorf("inserted/total ratio too low: %d/%d (<95%%)", result.InsertedMatches, result.TotalMatches)
	}

	// Spot-check the rows actually landed in DuckDB.
	var nReg, nPart, nMed, nHl, nAli int
	for tbl, ptr := range map[string]*int{
		"match_registry":     &nReg,
		"match_participants": &nPart,
		"medals_earned":      &nMed,
		"highlight_events":   &nHl,
		"xuid_aliases":       &nAli,
	} {
		if err := sharedDB.QueryRow(`SELECT COUNT(*) FROM ` + tbl).Scan(ptr); err != nil {
			t.Fatalf("count %s: %v", tbl, err)
		}
	}
	t.Logf("DuckDB row counts: registry=%d participants=%d medals=%d highlights=%d aliases=%d", nReg, nPart, nMed, nHl, nAli)
	if nReg != result.InsertedMatches {
		t.Errorf("registry rows (%d) != inserted matches (%d)", nReg, result.InsertedMatches)
	}
	// Sanity: the owner XUID should be present in match_participants for at least one match.
	var ownerHits int
	if err := sharedDB.QueryRow(`SELECT COUNT(*) FROM match_participants WHERE xuid = ?`, owner).Scan(&ownerHits); err != nil {
		t.Fatalf("query owner hits: %v", err)
	}
	if ownerHits == 0 {
		t.Fatal("owner XUID never appeared in match_participants")
	}
	_ = sql.ErrNoRows // silence unused import if scopes shift later
}
