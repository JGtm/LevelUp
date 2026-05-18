package openspartan

import (
	"context"
	"os"
	"testing"
)

// TestRealDB_Smoke is a manual smoke test against a real OpenSpartan database
// path supplied via OPENSPARTAN_DB_PATH. Skipped when the env var is unset, so
// the test suite remains green on CI machines without a real fixture.
//
// Run locally with:
//
//	OPENSPARTAN_DB_PATH=/path/to/{xuid}.db go test ./internal/openspartan/ -run TestRealDB -v
func TestRealDB_Smoke(t *testing.T) {
	path := os.Getenv("OPENSPARTAN_DB_PATH")
	if path == "" {
		t.Skip("OPENSPARTAN_DB_PATH not set; skipping manual smoke test")
	}

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open(%q): %v", path, err)
	}
	defer r.Close()

	ctx := context.Background()
	n, err := r.MatchCount(ctx)
	if err != nil {
		t.Fatalf("MatchCount: %v", err)
	}
	t.Logf("MatchCount: %d", n)
	if n == 0 {
		t.Fatal("real database should contain at least one match")
	}

	xuid, conf, err := r.DetectOwner(ctx, path)
	if err != nil {
		t.Fatalf("DetectOwner: %v", err)
	}
	t.Logf("DetectOwner: xuid=%s confidence=%s", xuid, conf)
	if xuid == "" {
		t.Fatal("DetectOwner returned empty xuid")
	}

	var parsed, errored int
	for pm, err := range r.Matches(ctx) {
		if err != nil {
			errored++
			continue
		}
		if pm == nil || pm.MatchID == "" {
			t.Errorf("parsed match has empty MatchID")
		}
		parsed++
	}
	t.Logf("Matches iterator: parsed=%d errored=%d (total=%d)", parsed, errored, n)
	if parsed == 0 {
		t.Fatal("expected at least one match parsed successfully")
	}
}
