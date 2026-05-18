//go:build integration

// Tests d'intégration de PersonalRecordsRepo (player_records dans shared_social.duckdb).
// Vérifie : Get/Upsert/ListByXUID, migration `period` correctement appliquée,
// nullable previous_value/previous_achieved_at, isolement entre xuids.

package duckdb

import (
	"context"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/progression/records"
)

// newRecordsRepoForTest construit un PersonalRecordsRepo branché sur une
// shared_social.duckdb temp avec les migrations appliquées.
func newRecordsRepoForTest(t *testing.T) *PersonalRecordsRepo {
	t.Helper()
	socialDB := setupPrestigeDB(t, migration.TargetSharedSocial)
	pdb := &PlayerDB{
		SharedSocial: socialDB,
		XUID:         "xuid-test",
		Gamertag:     "TestPlayer",
	}
	return NewPersonalRecordsRepo(pdb)
}

func TestPersonalRecordsRepo_UpsertGet_Roundtrip(t *testing.T) {
	repo := newRecordsRepoForTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	achieved := now.Add(-12 * time.Hour)

	pr := records.PersonalRecord{
		XUID:            "x1",
		Metric:          "performance_score",
		Period:          records.RecordPeriodAllTime,
		Value:           87.5,
		AchievedAt:      &achieved,
		AchievedMatchID: "match_42",
		UpdatedAt:       now,
	}
	if err := repo.Upsert(ctx, pr); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	got, err := repo.Get(ctx, "x1", "performance_score", records.RecordPeriodAllTime)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.Value != 87.5 {
		t.Errorf("Value = %.2f, want 87.5", got.Value)
	}
	if got.AchievedMatchID != "match_42" {
		t.Errorf("AchievedMatchID = %s, want match_42", got.AchievedMatchID)
	}
	if got.PreviousValue != nil {
		t.Errorf("PreviousValue should be nil, got %v", got.PreviousValue)
	}
}

func TestPersonalRecordsRepo_Upsert_OverwritesWithPrevious(t *testing.T) {
	repo := newRecordsRepoForTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	v1 := 80.0
	achieved1 := now.Add(-48 * time.Hour)
	pr1 := records.PersonalRecord{
		XUID: "x1", Metric: "kda", Period: records.RecordPeriod30d,
		Value: v1, AchievedAt: &achieved1, AchievedMatchID: "m1", UpdatedAt: now,
	}
	if err := repo.Upsert(ctx, pr1); err != nil {
		t.Fatalf("Upsert v1: %v", err)
	}

	// Nouveau PB qui bat v1.
	achieved2 := now
	pr2 := records.PersonalRecord{
		XUID: "x1", Metric: "kda", Period: records.RecordPeriod30d,
		Value: 92, AchievedAt: &achieved2, AchievedMatchID: "m2",
		PreviousValue:      &v1,
		PreviousAchievedAt: &achieved1,
		UpdatedAt:          now,
	}
	if err := repo.Upsert(ctx, pr2); err != nil {
		t.Fatalf("Upsert v2: %v", err)
	}

	got, err := repo.Get(ctx, "x1", "kda", records.RecordPeriod30d)
	if err != nil || got == nil {
		t.Fatalf("Get: got=%v err=%v", got, err)
	}
	if got.Value != 92 {
		t.Errorf("Value = %.2f, want 92", got.Value)
	}
	if got.PreviousValue == nil || *got.PreviousValue != 80 {
		t.Errorf("PreviousValue = %v, want 80", got.PreviousValue)
	}
	if got.PreviousAchievedAt == nil {
		t.Errorf("PreviousAchievedAt should be set")
	}
}

func TestPersonalRecordsRepo_ListByXUID_TwoPeriodsDifferentMetrics(t *testing.T) {
	repo := newRecordsRepoForTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	records2 := []records.PersonalRecord{
		{XUID: "x1", Metric: "performance_score", Period: records.RecordPeriod30d, Value: 75, UpdatedAt: now},
		{XUID: "x1", Metric: "performance_score", Period: records.RecordPeriodAllTime, Value: 90, UpdatedAt: now},
		{XUID: "x1", Metric: "kda", Period: records.RecordPeriod30d, Value: 1.4, UpdatedAt: now},
		// Autre joueur (devrait être exclu)
		{XUID: "x2", Metric: "performance_score", Period: records.RecordPeriod30d, Value: 60, UpdatedAt: now},
	}
	for _, pr := range records2 {
		if err := repo.Upsert(ctx, pr); err != nil {
			t.Fatalf("Upsert: %v", err)
		}
	}

	got, err := repo.ListByXUID(ctx, "x1")
	if err != nil {
		t.Fatalf("ListByXUID: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("ListByXUID len = %d, want 3 (x2 excluded)", len(got))
	}
}

// Vérifie qu'une migration appliquée taggant les anciennes lignes en period='all_time'
// laisse la table accessible avec la nouvelle PK composite.
func TestPersonalRecordsRepo_PeriodColumnPresent(t *testing.T) {
	repo := newRecordsRepoForTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// Insère 3 PB sur 3 périodes pour la même métrique → la PK composite
	// (xuid, metric, period) doit accepter les 3 sans conflit.
	for _, p := range records.AllRecordPeriods() {
		pr := records.PersonalRecord{
			XUID: "x1", Metric: "performance_score", Period: p,
			Value: 70 + float64(len(string(p))), UpdatedAt: now,
		}
		if err := repo.Upsert(ctx, pr); err != nil {
			t.Fatalf("Upsert period %s: %v", p, err)
		}
	}

	got, err := repo.ListByXUID(ctx, "x1")
	if err != nil {
		t.Fatalf("ListByXUID: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("len = %d, want 3 (one per period)", len(got))
	}
}
