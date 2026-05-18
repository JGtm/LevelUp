package service

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/openspartan"
)

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
	svc := NewOpenSpartanImportService(nil)
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
	svc := NewOpenSpartanImportService(sharedDB)

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
