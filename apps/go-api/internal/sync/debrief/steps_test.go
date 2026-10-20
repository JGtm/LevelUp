//go:build cgo

package debrief

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
)

func TestReportStatsEtFinDeMatch(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	fin := time.Date(2026, 10, 17, 20, 0, 0, 0, time.UTC)
	if _, err := db.Exec(`CREATE TABLE match_registry (match_id VARCHAR, end_time_utc TIMESTAMPTZ)`); err != nil {
		t.Fatalf("schéma: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO match_registry VALUES ('m1', ?), ('m2', NULL)`, fin); err != nil {
		t.Fatalf("insert: %v", err)
	}
	observability.EnableMatchSteps(true)
	t.Cleanup(func() { observability.EnableMatchSteps(false) })

	ReportStatsWritten("halo_infinite", []string{"m1", "m2"})
	ReportMatchEnds(context.Background(), db, "halo_infinite", []string{"m1", "m2"})

	var stats, fins int
	for _, ev := range observability.DrainMatchSteps() {
		switch ev.Step {
		case domain.MatchStepStatsWritten:
			stats++
		case domain.MatchStepEnd:
			fins++
			if ev.MatchID != "m1" || !ev.At.Equal(fin) {
				t.Fatalf("fin de match : %+v", ev)
			}
		}
	}
	if stats != 2 || fins != 1 {
		t.Fatalf("stats=%d fins=%d, attendu 2 et 1 (m2 sans fin connue)", stats, fins)
	}

	ReportStatsWritten("halo_infinite", nil)
	ReportMatchEnds(context.Background(), db, "halo_infinite", nil)
	if got := observability.DrainMatchSteps(); len(got) != 0 {
		t.Fatalf("aucun inséré, aucun instant : %+v", got)
	}
}
