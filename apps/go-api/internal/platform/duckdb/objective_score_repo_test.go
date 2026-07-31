//go:build integration

// Package duckdb — objective_score_repo_test.go : tests ObjectiveScoreRepo.
//
// Lancer avec : go test -tags=integration -run ObjectiveScore ./internal/platform/duckdb/ -v
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	titlepkg "levelup/go-api/internal/domain/title"

	"levelup/go-api/internal/analysis/objectivescore"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/migration"
)

const osTestMatchID = "m_objscore_001"

// newObjectiveScoreTestPlayerDB ouvre une mem DB, applique TOUTES les migrations
// shared (dont shared_objective_score_v1), puis construit un PlayerDB legacy RW.
func newObjectiveScoreTestPlayerDB(t *testing.T) *PlayerDB {
	t.Helper()
	sqlDB, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open mem: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	if err := migration.RunForDB(sqlDB, migration.TargetShared); err != nil {
		t.Fatalf("RunForDB(Shared): %v", err)
	}
	shared := newTestDB(sqlDB, ":memory:")

	return &PlayerDB{
		Shared:       shared,
		SharedReader: LegacySharedReader(shared),
		XUID:         pTestXUID,
		Gamertag:     pTestGamertag,
		TitleSlug:    titlepkg.DefaultSlug,
	}
}

func sampleScoreFrames() []objectivescore.ScoreFrame {
	return []objectivescore.ScoreFrame{
		{TimeMS: 0, Team0: 0, Team1: 0, Source: "film_strongholds_t2", Confidence: "approx"},
		{TimeMS: 20000, Team0: 50, Team1: 30, Source: "film_strongholds_t2", Confidence: "approx"},
		{TimeMS: 40000, Team0: 193, Team1: 112, Source: "film_strongholds_t2", Confidence: "approx"},
	}
}

func TestObjectiveScoreRepo_WriteThenLoad_RoundTrip(t *testing.T) {
	pdb := newObjectiveScoreTestPlayerDB(t)
	repo := NewObjectiveScoreRepo(pdb)
	ctx := context.Background()

	if err := repo.WriteMatch(ctx, osTestMatchID, sampleScoreFrames()); err != nil {
		t.Fatalf("WriteMatch: %v", err)
	}
	got, err := repo.LoadMatch(ctx, osTestMatchID)
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(frames) = %d, want 3", len(got))
	}
	// Ordonné par time_ms ASC.
	if got[0].TimeMS != 0 || got[2].TimeMS != 40000 {
		t.Fatalf("time order = [%d..%d], want [0..40000]", got[0].TimeMS, got[2].TimeMS)
	}
	last := got[2]
	if last.Team0 != 193 || last.Team1 != 112 {
		t.Errorf("last score = %d-%d, want 193-112", last.Team0, last.Team1)
	}
	if last.Source != "film_strongholds_t2" || last.Confidence != "approx" {
		t.Errorf("last source/conf = %q/%q", last.Source, last.Confidence)
	}
}

func TestObjectiveScoreRepo_WriteMatch_ReplaceIdempotent(t *testing.T) {
	pdb := newObjectiveScoreTestPlayerDB(t)
	repo := NewObjectiveScoreRepo(pdb)
	ctx := context.Background()

	if err := repo.WriteMatch(ctx, osTestMatchID, sampleScoreFrames()); err != nil {
		t.Fatalf("WriteMatch #1: %v", err)
	}
	// Ré-écrit un set RÉDUIT : le DELETE doit purger les 3 anciennes frames.
	replaced := []objectivescore.ScoreFrame{
		{TimeMS: 0, Team0: 0, Team1: 0, Source: "film_koth_t2_b", Confidence: "approx"},
		{TimeMS: 25000, Team0: 4, Team1: 2, Source: "film_koth_t2_b", Confidence: "approx"},
	}
	if err := repo.WriteMatch(ctx, osTestMatchID, replaced); err != nil {
		t.Fatalf("WriteMatch #2: %v", err)
	}
	got, err := repo.LoadMatch(ctx, osTestMatchID)
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(frames) = %d, want 2 (replace)", len(got))
	}
	if got[1].Source != "film_koth_t2_b" || got[1].Team0 != 4 {
		t.Errorf("got[1] = %+v, want koth 4-2", got[1])
	}
}

func TestObjectiveScoreRepo_WriteMatch_EmptyIsNoOp(t *testing.T) {
	pdb := newObjectiveScoreTestPlayerDB(t)
	repo := NewObjectiveScoreRepo(pdb)
	ctx := context.Background()

	if err := repo.WriteMatch(ctx, osTestMatchID, sampleScoreFrames()); err != nil {
		t.Fatalf("WriteMatch seed: %v", err)
	}
	if err := repo.WriteMatch(ctx, osTestMatchID, nil); err != nil {
		t.Fatalf("WriteMatch(nil): %v", err)
	}
	if err := repo.WriteMatch(ctx, osTestMatchID, []objectivescore.ScoreFrame{}); err != nil {
		t.Fatalf("WriteMatch([]): %v", err)
	}
	got, err := repo.LoadMatch(ctx, osTestMatchID)
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("len(frames) = %d, want 3 (no-op préserve)", len(got))
	}
}

func TestObjectiveScoreRepo_LoadMatch_Empty(t *testing.T) {
	pdb := newObjectiveScoreTestPlayerDB(t)
	repo := NewObjectiveScoreRepo(pdb)
	got, err := repo.LoadMatch(context.Background(), "no_such_match")
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len(frames) = %d, want 0", len(got))
	}
}

func TestObjectiveScoreRepo_CapabilityNotSupported_NoTable(t *testing.T) {
	pdb := newObjectiveScoreTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Shared.Exec(ctx, "DROP TABLE match_objective_score_timeline"); err != nil {
		t.Fatalf("drop: %v", err)
	}
	repo := NewObjectiveScoreRepo(pdb)
	if err := repo.WriteMatch(ctx, osTestMatchID, sampleScoreFrames()); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("WriteMatch err = %v, want ErrCapabilityNotSupported", err)
	}
	if _, err := repo.LoadMatch(ctx, osTestMatchID); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("LoadMatch err = %v, want ErrCapabilityNotSupported", err)
	}
}
