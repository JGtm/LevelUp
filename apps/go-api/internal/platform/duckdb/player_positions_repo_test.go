//go:build integration

// Package duckdb — player_positions_repo_test.go : tests PlayerPositionsRepo.
//
// Lancer avec : go test -tags=integration -run PlayerPositions ./internal/platform/duckdb/ -v
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	titlepkg "levelup/go-api/internal/domain/title"

	"levelup/go-api/internal/analysis/positions"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/migration"
)

const ppTestMatchID = "m_positions_001"

// newPlayerPositionsTestPlayerDB ouvre une mem DB, applique TOUTES les migrations
// shared (dont shared_match_player_positions_v1), puis construit un PlayerDB dont
// le SharedReader pointe sur cette conn (RW en legacy).
func newPlayerPositionsTestPlayerDB(t *testing.T) *PlayerDB {
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

func samplePositions() []positions.PlayerPosition {
	return []positions.PlayerPosition{
		{TimeMS: 0, X: 25.6, Y: 10.4, Z: 1.2, Team: 0},
		{TimeMS: 0, X: -6.0, Y: -24.0, Z: -2.8, Team: 1},
		{TimeMS: 20000, X: 34.8, Y: 13.5, Z: 0.5, Team: positions.TeamUnknown},
	}
}

// ---------------------------------------------------------------------------
// Round-trip
// ---------------------------------------------------------------------------

func TestPlayerPositionsRepo_WriteThenLoad_RoundTrip(t *testing.T) {
	pdb := newPlayerPositionsTestPlayerDB(t)
	repo := NewPlayerPositionsRepo(pdb)
	ctx := context.Background()

	if err := repo.WriteMatch(ctx, ppTestMatchID, samplePositions()); err != nil {
		t.Fatalf("WriteMatch: %v", err)
	}

	got, err := repo.LoadMatch(ctx, ppTestMatchID)
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(positions) = %d, want 3", len(got))
	}

	// Ordonné par time_ms ASC : les 2 premiers à t=0, le 3e à t=20000.
	if got[0].TimeMS != 0 || got[2].TimeMS != 20000 {
		t.Fatalf("time order = [%d,..,%d], want [0,..,20000]", got[0].TimeMS, got[2].TimeMS)
	}

	// Valeurs float32 préservées (REAL = float32 natif, pas d'arrondi attendu).
	last := got[2]
	if last.X != 34.8 || last.Y != 13.5 || last.Z != 0.5 {
		t.Errorf("last pos = (%.2f,%.2f,%.2f), want (34.80,13.50,0.50)", last.X, last.Y, last.Z)
	}
	if last.Team != positions.TeamUnknown {
		t.Errorf("last.Team = %d, want %d (TeamUnknown)", last.Team, positions.TeamUnknown)
	}

	// Une position team=1 doit subsister (parmi les 2 à t=0).
	var hasTeam1 bool
	for _, p := range got {
		if p.Team == 1 {
			hasTeam1 = true
		}
	}
	if !hasTeam1 {
		t.Errorf("aucune position team=1 retrouvée")
	}
}

// ---------------------------------------------------------------------------
// Idempotence DELETE-replace
// ---------------------------------------------------------------------------

func TestPlayerPositionsRepo_WriteMatch_ReplaceIdempotent(t *testing.T) {
	pdb := newPlayerPositionsTestPlayerDB(t)
	repo := NewPlayerPositionsRepo(pdb)
	ctx := context.Background()

	if err := repo.WriteMatch(ctx, ppTestMatchID, samplePositions()); err != nil {
		t.Fatalf("WriteMatch #1: %v", err)
	}

	// Ré-écrit avec un set RÉDUIT (1 seule position) : le DELETE doit purger les 3
	// anciennes avant l'INSERT.
	replaced := []positions.PlayerPosition{
		{TimeMS: 5000, X: 1.0, Y: 2.0, Z: 3.0, Team: 0},
	}
	if err := repo.WriteMatch(ctx, ppTestMatchID, replaced); err != nil {
		t.Fatalf("WriteMatch #2: %v", err)
	}

	got, err := repo.LoadMatch(ctx, ppTestMatchID)
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(positions) = %d, want 1 (replace)", len(got))
	}
	if got[0].TimeMS != 5000 {
		t.Errorf("got[0].TimeMS = %d, want 5000", got[0].TimeMS)
	}
}

// ---------------------------------------------------------------------------
// No-op garde-fou
// ---------------------------------------------------------------------------

func TestPlayerPositionsRepo_WriteMatch_EmptyIsNoOp(t *testing.T) {
	pdb := newPlayerPositionsTestPlayerDB(t)
	repo := NewPlayerPositionsRepo(pdb)
	ctx := context.Background()

	if err := repo.WriteMatch(ctx, ppTestMatchID, samplePositions()); err != nil {
		t.Fatalf("WriteMatch seed: %v", err)
	}
	// nil et [] sont des no-op : ne doivent PAS effacer l'existant.
	if err := repo.WriteMatch(ctx, ppTestMatchID, nil); err != nil {
		t.Fatalf("WriteMatch(nil): %v", err)
	}
	if err := repo.WriteMatch(ctx, ppTestMatchID, []positions.PlayerPosition{}); err != nil {
		t.Fatalf("WriteMatch([]): %v", err)
	}

	got, err := repo.LoadMatch(ctx, ppTestMatchID)
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("len(positions) = %d, want 3 (no-op préserve)", len(got))
	}
}

// LoadMatch sur un match absent retourne un slice vide, pas une erreur.
func TestPlayerPositionsRepo_LoadMatch_Empty(t *testing.T) {
	pdb := newPlayerPositionsTestPlayerDB(t)
	repo := NewPlayerPositionsRepo(pdb)

	got, err := repo.LoadMatch(context.Background(), "no_such_match")
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len(positions) = %d, want 0", len(got))
	}
}

// ---------------------------------------------------------------------------
// Capability gating
// ---------------------------------------------------------------------------

func TestPlayerPositionsRepo_CapabilityNotSupported_NoTable(t *testing.T) {
	pdb := newPlayerPositionsTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Shared.Exec(ctx, "DROP TABLE match_player_positions"); err != nil {
		t.Fatalf("drop: %v", err)
	}

	repo := NewPlayerPositionsRepo(pdb)

	if err := repo.WriteMatch(ctx, ppTestMatchID, samplePositions()); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("WriteMatch err = %v, want ErrCapabilityNotSupported", err)
	}
	if _, err := repo.LoadMatch(ctx, ppTestMatchID); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("LoadMatch err = %v, want ErrCapabilityNotSupported", err)
	}
}
