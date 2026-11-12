//go:build integration

// progression_store_test.go : PlayerProgressionStore et NewCampaignScope contre de vraies bases
// migrées (stats du joueur, base sociale partagée, référentiel metadata).
package duckdb

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/progression/milestones"
	"levelup/go-api/internal/progression/records"
	"levelup/go-api/internal/progression/streaks"
)

const (
	storeTestXUID  = "xuid-progression-store"
	storeTestTitle = "halo_infinite"
)

func newProgressionStoreFixture(t *testing.T) (*PlayerProgressionStore, *PlayerDB) {
	t.Helper()
	pdb := &PlayerDB{
		Player:       setupPrestigeDB(t, migration.TargetPlayer),
		SharedSocial: setupPrestigeDB(t, migration.TargetSharedSocial),
		Metadata:     setupPrestigeDB(t, migration.TargetMetadata),
		XUID:         storeTestXUID,
		Gamertag:     "StoreTester",
		TitleSlug:    storeTestTitle,
	}
	return NewPlayerProgressionStore(pdb), pdb
}

func TestPlayerProgressionStore_VideEtIsole(t *testing.T) {
	store, _ := newProgressionStoreFixture(t)
	ctx := context.Background()

	if got, err := store.ListStreaks(ctx, storeTestTitle); err != nil || len(got) != 0 {
		t.Errorf("ListStreaks sur base vide = %v, %v", got, err)
	}
	if got, err := store.ListPersonalBests(ctx); err != nil || len(got) != 0 {
		t.Errorf("ListPersonalBests sur base vide = %v, %v", got, err)
	}
	if got, err := store.ListRecordHistory(ctx, storeTestTitle, 10); err != nil || len(got) != 0 {
		t.Errorf("ListRecordHistory sur base vide = %v, %v", got, err)
	}
	if got, err := store.ListMilestonesEarned(ctx, storeTestTitle); err != nil || len(got) != 0 {
		t.Errorf("ListMilestonesEarned sur base vide = %v, %v", got, err)
	}
}

func TestPlayerProgressionStore_LitLesTablesDuJoueurEtDuTitre(t *testing.T) {
	store, pdb := newProgressionStoreFixture(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	if err := NewStreaksRepo(pdb.Player).Upsert(ctx, streaks.Streak{
		ID: "s1", UserID: storeTestXUID, TitleSlug: storeTestTitle, Type: streaks.StreakTypeDailyPlay,
		StartedAt: now.AddDate(0, 0, -2), CurrentLength: 3, BestLength: 3, LastIncrementAt: &now,
		ShieldsAvailable: streaks.MaxShieldsPerMonth, Status: streaks.StreakStatusActive,
	}); err != nil {
		t.Fatalf("seed streak: %v", err)
	}
	if got, err := store.ListStreaks(ctx, storeTestTitle); err != nil || len(got) != 1 || got[0].ID != "s1" {
		t.Fatalf("ListStreaks = %+v, %v", got, err)
	}
	if got, err := store.ListStreaks(ctx, "halo_5"); err != nil || len(got) != 0 {
		t.Fatalf("ListStreaks d'un autre titre = %+v, %v ; attendu vide", got, err)
	}

	if err := NewPersonalRecordsRepo(pdb).Upsert(ctx, records.PersonalRecord{
		XUID: storeTestXUID, Metric: "performance_score", Period: records.RecordPeriodAllTime,
		Value: 92, AchievedAt: &now, AchievedMatchID: "m1", UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed PB: %v", err)
	}
	if got, err := store.ListPersonalBests(ctx); err != nil || len(got) != 1 || got[0].Value != 92 {
		t.Fatalf("ListPersonalBests = %+v, %v", got, err)
	}

	hist := NewRecordHistoryRepo(pdb.Player)
	for i, id := range []string{"h1", "h2", "h3"} {
		if err := hist.Append(ctx, records.RecordHistory{
			ID: id, UserID: storeTestXUID, TitleSlug: storeTestTitle, Metric: "performance_score",
			Period: records.RecordPeriodAllTime, Value: float64(80 + i), AchievedAt: now.Add(time.Duration(i) * time.Hour),
		}); err != nil {
			t.Fatalf("seed history %s: %v", id, err)
		}
	}
	got, err := store.ListRecordHistory(ctx, storeTestTitle, 2)
	if err != nil || len(got) != 2 || got[0].ID != "h3" {
		t.Fatalf("ListRecordHistory(limit 2) = %+v, %v ; attendu les 2 plus récents, h3 d'abord", got, err)
	}

	if err := NewMilestoneCatalogRepo(pdb.Metadata).Upsert(ctx, milestones.CatalogEntry{
		ID: "h.matches.10", TitleSlug: storeTestTitle, Metric: "matches_played", Threshold: 10,
		Title: i18n.FromColumns("Decimus", "Decimus"),
	}); err != nil {
		t.Fatalf("seed catalog: %v", err)
	}
	if err := NewMilestoneEarnedRepo(pdb.Player).Append(ctx, milestones.Earned{
		UserID: storeTestXUID, TitleSlug: storeTestTitle, MilestoneID: "h.matches.10", EarnedAt: now,
	}); err != nil {
		t.Fatalf("seed earned: %v", err)
	}
	if cat, err := store.ListMilestoneCatalog(ctx, storeTestTitle); err != nil || len(cat) != 1 || cat[0].ID != "h.matches.10" {
		t.Fatalf("ListMilestoneCatalog = %+v, %v", cat, err)
	}
	if earned, err := store.ListMilestonesEarned(ctx, storeTestTitle); err != nil || len(earned) != 1 || earned[0].MilestoneID != "h.matches.10" {
		t.Fatalf("ListMilestonesEarned = %+v, %v", earned, err)
	}
}

func TestNewCampaignScope_LieLeJoueur(t *testing.T) {
	_, pdb := newProgressionStoreFixture(t)
	sc := NewCampaignScope(pdb)
	if sc.XUID != storeTestXUID || sc.Repo == nil || sc.Samples == nil {
		t.Fatalf("scope = %+v ; attendu XUID %s et deux dépôts", sc, storeTestXUID)
	}
}
