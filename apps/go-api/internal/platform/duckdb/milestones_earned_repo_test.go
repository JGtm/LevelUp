//go:build integration

// Tests d'intégration de MilestoneEarnedRepo (stats.duckdb).

package duckdb

import (
	"context"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/progression/milestones"
)

func newEarnedRepoForTest(t *testing.T) *MilestoneEarnedRepo {
	t.Helper()
	db := setupPrestigeDB(t, migration.TargetPlayer)
	return NewMilestoneEarnedRepo(db)
}

func TestMilestoneEarnedRepo_AppendAndIsEarned(t *testing.T) {
	repo := newEarnedRepoForTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	got, err := repo.IsEarned(ctx, "u1", "halo_infinite", "h.matches.100")
	if err != nil {
		t.Fatalf("IsEarned: %v", err)
	}
	if got {
		t.Errorf("IsEarned should be false on empty repo")
	}

	if err := repo.Append(ctx, milestones.Earned{
		UserID: "u1", TitleSlug: "halo_infinite", MilestoneID: "h.matches.100", EarnedAt: now,
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, err = repo.IsEarned(ctx, "u1", "halo_infinite", "h.matches.100")
	if err != nil {
		t.Fatalf("IsEarned: %v", err)
	}
	if !got {
		t.Errorf("IsEarned should be true after Append")
	}
}

func TestMilestoneEarnedRepo_Append_Idempotent(t *testing.T) {
	repo := newEarnedRepoForTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	later := now.Add(24 * time.Hour)

	if err := repo.Append(ctx, milestones.Earned{
		UserID: "u1", TitleSlug: "halo_infinite", MilestoneID: "h.matches.100", EarnedAt: now,
	}); err != nil {
		t.Fatalf("Append 1st: %v", err)
	}
	// 2e Append avec un earned_at différent — doit être ignoré.
	if err := repo.Append(ctx, milestones.Earned{
		UserID: "u1", TitleSlug: "halo_infinite", MilestoneID: "h.matches.100", EarnedAt: later,
	}); err != nil {
		t.Fatalf("Append 2nd: %v", err)
	}

	list, err := repo.ListByUser(ctx, "u1", "halo_infinite")
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1 (idempotence)", len(list))
	}
	if !list[0].EarnedAt.Equal(now) {
		t.Errorf("EarnedAt should stay at first Append value (%v), got %v", now, list[0].EarnedAt)
	}
}

func TestMilestoneEarnedRepo_ListByUser_FiltersAndOrdersDESC(t *testing.T) {
	repo := newEarnedRepoForTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	entries := []milestones.Earned{
		{UserID: "u1", TitleSlug: "halo_infinite", MilestoneID: "old", EarnedAt: now.AddDate(0, 0, -10)},
		{UserID: "u1", TitleSlug: "halo_infinite", MilestoneID: "recent", EarnedAt: now.AddDate(0, 0, -1)},
		{UserID: "u1", TitleSlug: "halo_infinite", MilestoneID: "newest", EarnedAt: now},
		{UserID: "u2", TitleSlug: "halo_infinite", MilestoneID: "other_user", EarnedAt: now},
	}
	for _, e := range entries {
		if err := repo.Append(ctx, e); err != nil {
			t.Fatalf("Append %s: %v", e.MilestoneID, err)
		}
	}

	got, err := repo.ListByUser(ctx, "u1", "halo_infinite")
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3 (u2 excluded)", len(got))
	}
	if got[0].MilestoneID != "newest" || got[1].MilestoneID != "recent" || got[2].MilestoneID != "old" {
		t.Errorf("ordre DESC incorrect : %v", []string{got[0].MilestoneID, got[1].MilestoneID, got[2].MilestoneID})
	}
}
