//go:build integration

// Tests d'intégration du repo Streaks contre une vraie stats.duckdb.
// Vérifie : round-trip Upsert/GetActive, exclusion des streaks broken sur
// GetActive, List historique complet, gestion des champs nullables.

package duckdb

import (
	"context"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/progression/streaks"
)

func newStreaksRepoForTest(t *testing.T) *StreaksRepo {
	t.Helper()
	db := setupPrestigeDB(t, migration.TargetPlayer)
	return NewStreaksRepo(db)
}

func TestStreaksRepo_UpsertAndGetActive_Roundtrip(t *testing.T) {
	repo := newStreaksRepoForTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	s := streaks.Streak{
		ID:               "streak_001",
		UserID:           "user_42",
		TitleSlug:        "halo_infinite",
		Type:             streaks.StreakTypeDailyPlay,
		StartedAt:        now.AddDate(0, 0, -3),
		CurrentLength:    4,
		BestLength:       4,
		LastIncrementAt:  &now,
		ShieldsUsed:      0,
		ShieldsAvailable: streaks.MaxShieldsPerMonth,
		Status:           streaks.StreakStatusActive,
	}
	if err := repo.Upsert(ctx, s); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := repo.GetActive(ctx, s.UserID, s.TitleSlug, s.Type)
	if err != nil {
		t.Fatalf("GetActive: %v", err)
	}
	if got == nil {
		t.Fatal("GetActive returned nil, expected streak")
	}
	if got.ID != s.ID {
		t.Errorf("ID = %s, want %s", got.ID, s.ID)
	}
	if got.CurrentLength != 4 {
		t.Errorf("CurrentLength = %d, want 4", got.CurrentLength)
	}
	if got.LastIncrementAt == nil || !got.LastIncrementAt.Equal(now) {
		t.Errorf("LastIncrementAt mismatch: got %v, want %v", got.LastIncrementAt, now)
	}
	if got.Threshold != nil {
		t.Errorf("Threshold should be nil for daily_play, got %v", got.Threshold)
	}
}

func TestStreaksRepo_GetActive_ExcludesBroken(t *testing.T) {
	repo := newStreaksRepoForTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	broken := streaks.Streak{
		ID:            "streak_broken",
		UserID:        "user_42",
		TitleSlug:     "halo_infinite",
		Type:          streaks.StreakTypeDailyPlay,
		StartedAt:     now.AddDate(0, 0, -10),
		CurrentLength: 7, BestLength: 7,
		ShieldsAvailable: streaks.MaxShieldsPerMonth,
		Status:           streaks.StreakStatusBroken,
		BrokenAt:         &now,
	}
	if err := repo.Upsert(ctx, broken); err != nil {
		t.Fatalf("Upsert broken: %v", err)
	}

	got, err := repo.GetActive(ctx, broken.UserID, broken.TitleSlug, broken.Type)
	if err != nil {
		t.Fatalf("GetActive: %v", err)
	}
	if got != nil {
		t.Errorf("GetActive returned %v, want nil (broken excluded)", got)
	}
}

func TestStreaksRepo_UpsertOverwrites(t *testing.T) {
	repo := newStreaksRepoForTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	s := streaks.Streak{
		ID:            "streak_overwrite",
		UserID:        "user_42",
		TitleSlug:     "halo_infinite",
		Type:          streaks.StreakTypeWeeklyPlay,
		StartedAt:     now.AddDate(0, 0, -14),
		CurrentLength: 2, BestLength: 2,
		ShieldsAvailable: streaks.MaxShieldsPerMonth,
		Status:           streaks.StreakStatusActive,
	}
	if err := repo.Upsert(ctx, s); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}

	// Incrémenter et resauver.
	s.CurrentLength = 3
	s.BestLength = 3
	s.LastIncrementAt = &now
	if err := repo.Upsert(ctx, s); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}

	got, err := repo.GetActive(ctx, s.UserID, s.TitleSlug, s.Type)
	if err != nil || got == nil {
		t.Fatalf("GetActive: got=%v err=%v", got, err)
	}
	if got.CurrentLength != 3 {
		t.Errorf("after overwrite CurrentLength = %d, want 3", got.CurrentLength)
	}
}

func TestStreaksRepo_List_AllStatuses(t *testing.T) {
	repo := newStreaksRepoForTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	active := streaks.Streak{
		ID: "s_active", UserID: "user_42", TitleSlug: "halo_infinite",
		Type: streaks.StreakTypeDailyPlay, StartedAt: now.AddDate(0, 0, -2),
		CurrentLength: 3, BestLength: 3,
		ShieldsAvailable: streaks.MaxShieldsPerMonth, Status: streaks.StreakStatusActive,
	}
	broken := streaks.Streak{
		ID: "s_broken", UserID: "user_42", TitleSlug: "halo_infinite",
		Type: streaks.StreakTypeWeeklyPlay, StartedAt: now.AddDate(0, 0, -20),
		CurrentLength: 1, BestLength: 1,
		ShieldsAvailable: streaks.MaxShieldsPerMonth,
		Status:           streaks.StreakStatusBroken, BrokenAt: &now,
	}
	if err := repo.Upsert(ctx, active); err != nil {
		t.Fatalf("Upsert active: %v", err)
	}
	if err := repo.Upsert(ctx, broken); err != nil {
		t.Fatalf("Upsert broken: %v", err)
	}

	list, err := repo.List(ctx, "user_42", "halo_infinite")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("List len = %d, want 2 (active + broken)", len(list))
	}
}

func TestStreaksRepo_Threshold_NullableRoundtrip(t *testing.T) {
	repo := newStreaksRepoForTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	thr := 1.5
	s := streaks.Streak{
		ID: "s_perf", UserID: "user_42", TitleSlug: "halo_infinite",
		Type: streaks.StreakTypeDailyPerf, StartedAt: now,
		CurrentLength: 1, BestLength: 1,
		LastIncrementAt:  &now,
		Threshold:        &thr,
		ShieldsAvailable: streaks.MaxShieldsPerMonth,
		Status:           streaks.StreakStatusActive,
	}
	if err := repo.Upsert(ctx, s); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := repo.GetActive(ctx, s.UserID, s.TitleSlug, s.Type)
	if err != nil || got == nil {
		t.Fatalf("GetActive: got=%v err=%v", got, err)
	}
	if got.Threshold == nil || *got.Threshold != 1.5 {
		t.Errorf("Threshold roundtrip failed: got %v, want 1.5", got.Threshold)
	}
}
