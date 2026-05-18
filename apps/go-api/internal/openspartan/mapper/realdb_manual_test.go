package mapper

import (
	"context"
	"errors"
	"os"
	"testing"

	"levelup/go-api/internal/openspartan"
)

// TestRealDB_MapAllMatches is a manual end-to-end smoke test: it walks a real
// OpenSpartan database with the openspartan.Reader, runs MapMatch on each
// row, and reports how many succeed vs fail. Skipped without
// OPENSPARTAN_DB_PATH so CI stays green.
//
// Run locally with:
//
//	OPENSPARTAN_DB_PATH=/path/to/{xuid}.db go test ./internal/openspartan/mapper/ -run TestRealDB -v
func TestRealDB_MapAllMatches(t *testing.T) {
	path := os.Getenv("OPENSPARTAN_DB_PATH")
	if path == "" {
		t.Skip("OPENSPARTAN_DB_PATH not set; skipping manual smoke test")
	}

	r, err := openspartan.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer r.Close()

	var (
		parsed, mapped int
		mapErrors      = make(map[string]int) // error message → count
		invalidMatch   int
		futureMatch    int
	)
	for pm, err := range r.Matches(context.Background()) {
		if err != nil {
			mapErrors["reader: "+err.Error()]++
			continue
		}
		parsed++
		if _, err := MapMatch(pm, MapOptions{Source: "smoke_test"}); err != nil {
			if errors.Is(err, ErrInvalidMatch) {
				invalidMatch++
			} else if errors.Is(err, ErrFutureMatch) {
				futureMatch++
			} else {
				mapErrors["map: "+err.Error()]++
			}
			continue
		}
		mapped++
	}
	t.Logf("parsed=%d mapped=%d invalid=%d future=%d", parsed, mapped, invalidMatch, futureMatch)
	for msg, n := range mapErrors {
		t.Logf("error[%d]: %s", n, msg)
	}
	if mapped == 0 {
		t.Fatal("expected at least one match to map successfully on a real database")
	}
	// Allow some tolerance for legacy or corrupt rows, but most should map.
	if float64(mapped) < 0.95*float64(parsed) {
		t.Errorf("mapped/parsed ratio too low: %d/%d (<95%%)", mapped, parsed)
	}
}
