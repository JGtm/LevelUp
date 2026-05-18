//go:build integration

// Tests d'intégration de RecordHistoryRepo (record_history dans stats.duckdb).
// Vérifie : Append append-only, ListRecent triée DESC, isolement par user_id+title_slug.

package duckdb

import (
	"context"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/progression/records"
)

func newHistoryRepoForTest(t *testing.T) *RecordHistoryRepo {
	t.Helper()
	db := setupPrestigeDB(t, migration.TargetPlayer)
	return NewRecordHistoryRepo(db)
}

func TestRecordHistoryRepo_AppendAndList(t *testing.T) {
	repo := newHistoryRepoForTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	entries := []records.RecordHistory{
		{ID: "h1", UserID: "u1", TitleSlug: "halo_infinite", Metric: "performance_score",
			Period: records.RecordPeriod30d, Value: 75, AchievedAt: now.AddDate(0, 0, -3)},
		{ID: "h2", UserID: "u1", TitleSlug: "halo_infinite", Metric: "performance_score",
			Period: records.RecordPeriod30d, Value: 82, AchievedAt: now.AddDate(0, 0, -1)},
		{ID: "h3", UserID: "u1", TitleSlug: "halo_infinite", Metric: "kda",
			Period: records.RecordPeriodAllTime, Value: 1.7, AchievedAt: now},
	}
	for _, h := range entries {
		if err := repo.Append(ctx, h); err != nil {
			t.Fatalf("Append %s: %v", h.ID, err)
		}
	}

	got, err := repo.ListRecent(ctx, "u1", "halo_infinite", 10)
	if err != nil {
		t.Fatalf("ListRecent: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	// Tri DESC sur achieved_at : h3 (now), h2 (J-1), h1 (J-3)
	if got[0].ID != "h3" || got[1].ID != "h2" || got[2].ID != "h1" {
		t.Errorf("ordre incorrect : %v %v %v, want h3 h2 h1", got[0].ID, got[1].ID, got[2].ID)
	}
}

func TestRecordHistoryRepo_ListRecent_Limit(t *testing.T) {
	repo := newHistoryRepoForTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	for i := 0; i < 5; i++ {
		if err := repo.Append(ctx, records.RecordHistory{
			ID: "h" + string(rune('a'+i)), UserID: "u1", TitleSlug: "halo_infinite",
			Metric: "performance_score", Period: records.RecordPeriod30d,
			Value: float64(50 + i), AchievedAt: now.AddDate(0, 0, -i),
		}); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	got, err := repo.ListRecent(ctx, "u1", "halo_infinite", 2)
	if err != nil {
		t.Fatalf("ListRecent: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len = %d, want 2 (limit)", len(got))
	}
}

func TestRecordHistoryRepo_FiltersByUserAndTitle(t *testing.T) {
	repo := newHistoryRepoForTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	if err := repo.Append(ctx, records.RecordHistory{
		ID: "h1", UserID: "u1", TitleSlug: "halo_infinite",
		Metric: "performance_score", Period: records.RecordPeriod30d,
		Value: 80, AchievedAt: now,
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	// Autre titre — devrait être exclu.
	if err := repo.Append(ctx, records.RecordHistory{
		ID: "h2", UserID: "u1", TitleSlug: "another_title",
		Metric: "performance_score", Period: records.RecordPeriod30d,
		Value: 70, AchievedAt: now,
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	// Autre user_id — devrait être exclu.
	if err := repo.Append(ctx, records.RecordHistory{
		ID: "h3", UserID: "u2", TitleSlug: "halo_infinite",
		Metric: "performance_score", Period: records.RecordPeriod30d,
		Value: 60, AchievedAt: now,
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, err := repo.ListRecent(ctx, "u1", "halo_infinite", 0) // limit 0 → default
	if err != nil {
		t.Fatalf("ListRecent: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("len = %d, want 1 (u1+halo_infinite only)", len(got))
	}
}
