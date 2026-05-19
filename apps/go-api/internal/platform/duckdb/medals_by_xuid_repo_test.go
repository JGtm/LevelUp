//go:build integration

// Package duckdb — medals_by_xuid_repo_test.go : tests MedalsByXUIDRepo.
//
// Lancer avec : go test -tags=integration ./internal/platform/duckdb/ -v
package duckdb

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

const mxTestXUIDOther = "xuid_player_002"

// seedMedals insere des medailles pour deux joueurs sur deux matchs.
//
// Sprint B1 commit 8k.3 : double-write player+shared. Le repo MedalsByXUIDRepo
// lit désormais via SharedReadDB() → pdb.Shared (séparé de pdb.Player). Pour
// rester compatible avec les autres tests qui scannent shared.* via pdb.Player
// (chaîne héritée des migrations 8c/8d), on insère dans les deux DBs.
func seedMedals(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	ctx := context.Background()

	// Ajouter m2 dans match_registry pour coherence des tests multi-matchs
	for _, db := range []*DB{pdb.Player, pdb.Shared} {
		if _, err := db.Exec(ctx,
			`INSERT INTO shared.match_registry (match_id, start_time)
			 VALUES (?, ?)`,
			"m2", "2025-01-11 14:00:00+00",
		); err != nil {
			t.Fatalf("seed match_registry m2: %v", err)
		}
	}

	medals := []struct {
		matchID string
		xuid    string
		medalID uint64
		count   int
	}{
		{"m1", pTestXUID, 1001, 2},       // Killing Spree
		{"m1", pTestXUID, 1002, 1},       // Double Kill
		{"m2", pTestXUID, 1001, 3},       // Killing Spree (autre match)
		{"m1", mxTestXUIDOther, 1003, 1}, // Triple Kill
	}
	for _, db := range []*DB{pdb.Player, pdb.Shared} {
		for _, m := range medals {
			if _, err := db.Exec(ctx,
				`INSERT INTO shared.medals_earned
				 (medal_id, medal_name_id, xuid, match_id, count)
				 VALUES (?, ?, ?, ?, ?)`,
				m.medalID, m.medalID, m.xuid, m.matchID, m.count,
			); err != nil {
				t.Fatalf("seed medals_earned: %v", err)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

func TestMedalsByXUIDRepo_Load_RejectsEmptyMatchIDs(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewMedalsByXUIDRepo(pdb)
	_, err := repo.LoadMedalsForMatchesByXUID(context.Background(), "halo_infinite",
		port.MedalsByXUIDFilters{XUIDs: []string{pTestXUID}})
	if err == nil {
		t.Fatal("attendu erreur (MatchIDs vide)")
	}
	if !errors.Is(err, port.ErrMedalsByXUIDFiltersTooBroad) {
		t.Errorf("err = %v, want ErrMedalsByXUIDFiltersTooBroad", err)
	}
}

func TestMedalsByXUIDRepo_Load_RejectsEmptyXUIDs(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewMedalsByXUIDRepo(pdb)
	_, err := repo.LoadMedalsForMatchesByXUID(context.Background(), "halo_infinite",
		port.MedalsByXUIDFilters{MatchIDs: []string{"m1"}})
	if err == nil {
		t.Fatal("attendu erreur (XUIDs vide)")
	}
	if !errors.Is(err, port.ErrMedalsByXUIDFiltersTooBroad) {
		t.Errorf("err = %v, want ErrMedalsByXUIDFiltersTooBroad", err)
	}
}

func TestMedalsByXUIDRepo_Load_RejectsNegativeLimit(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewMedalsByXUIDRepo(pdb)
	_, err := repo.LoadMedalsForMatchesByXUID(context.Background(), "halo_infinite",
		port.MedalsByXUIDFilters{
			MatchIDs: []string{"m1"},
			XUIDs:    []string{pTestXUID},
			Limit:    -1,
		})
	if err == nil {
		t.Fatal("attendu erreur (Limit negative)")
	}
	if !errors.Is(err, port.ErrMedalsByXUIDFiltersInvalid) {
		t.Errorf("err = %v, want ErrMedalsByXUIDFiltersInvalid", err)
	}
}

// ---------------------------------------------------------------------------
// Capability gating
// ---------------------------------------------------------------------------

func TestMedalsByXUIDRepo_Load_CapabilityNotSupported_NoTable(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	// Drop dans les deux DBs : le repo lit via SharedReadDB() (pdb.Shared)
	// depuis le commit 8k.3, mais pdb.Player conserve aussi la table pour
	// les tests legacy.
	for _, db := range []*DB{pdb.Player, pdb.Shared} {
		if _, err := db.Exec(ctx, "DROP TABLE shared.medals_earned"); err != nil {
			t.Fatalf("DROP TABLE: %v", err)
		}
	}

	repo := NewMedalsByXUIDRepo(pdb)
	_, err := repo.LoadMedalsForMatchesByXUID(ctx, "halo_infinite",
		port.MedalsByXUIDFilters{
			MatchIDs: []string{"m1"},
			XUIDs:    []string{pTestXUID},
		})
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("err = %v, want ErrCapabilityNotSupported", err)
	}
}

// ---------------------------------------------------------------------------
// Querying
// ---------------------------------------------------------------------------

func TestMedalsByXUIDRepo_Load_FiltersByMatchAndXUID(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedMedals(t, pdb)

	repo := NewMedalsByXUIDRepo(pdb)
	rows, err := repo.LoadMedalsForMatchesByXUID(context.Background(), "halo_infinite",
		port.MedalsByXUIDFilters{
			MatchIDs: []string{"m1"},
			XUIDs:    []string{pTestXUID},
		})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// p1 m1 : 2 medailles (1001 count=2, 1002 count=1)
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2 (p1 only on m1), rows=%+v", len(rows), rows)
	}
	if rows[0].MedalID != 1001 || rows[0].Count != 2 {
		t.Errorf("rows[0] = %+v, want MedalID=1001 Count=2", rows[0])
	}
	if rows[0].XUID != pTestXUID || rows[0].MatchID != "m1" {
		t.Errorf("rows[0] xuid/match = %q/%q", rows[0].XUID, rows[0].MatchID)
	}
}

func TestMedalsByXUIDRepo_Load_MultiPlayers(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedMedals(t, pdb)

	repo := NewMedalsByXUIDRepo(pdb)
	rows, err := repo.LoadMedalsForMatchesByXUID(context.Background(), "halo_infinite",
		port.MedalsByXUIDFilters{
			MatchIDs: []string{"m1", "m2"},
			XUIDs:    []string{pTestXUID, mxTestXUIDOther},
		})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// p1 m1 : 2 rows ; p1 m2 : 1 row ; p2 m1 : 1 row -> 4 total
	if len(rows) != 4 {
		t.Fatalf("len(rows) = %d, want 4, rows=%+v", len(rows), rows)
	}
}

func TestMedalsByXUIDRepo_Load_AppliesLimit(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedMedals(t, pdb)

	repo := NewMedalsByXUIDRepo(pdb)
	rows, err := repo.LoadMedalsForMatchesByXUID(context.Background(), "halo_infinite",
		port.MedalsByXUIDFilters{
			MatchIDs: []string{"m1", "m2"},
			XUIDs:    []string{pTestXUID, mxTestXUIDOther},
			Limit:    2,
		})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2 (Limit)", len(rows))
	}
}

func TestMedalsByXUIDRepo_Load_NoResults(t *testing.T) {
	pdb := newTestPlayerDB(t)
	// Pas de seedMedals : la table existe mais vide
	repo := NewMedalsByXUIDRepo(pdb)
	rows, err := repo.LoadMedalsForMatchesByXUID(context.Background(), "halo_infinite",
		port.MedalsByXUIDFilters{
			MatchIDs: []string{"m1"},
			XUIDs:    []string{pTestXUID},
		})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("len(rows) = %d, want 0", len(rows))
	}
}

// Vérifie qu'on ne renvoie aucune ligne pour un XUID absent
func TestMedalsByXUIDRepo_Load_UnknownXUID(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedMedals(t, pdb)

	repo := NewMedalsByXUIDRepo(pdb)
	rows, err := repo.LoadMedalsForMatchesByXUID(context.Background(), "halo_infinite",
		port.MedalsByXUIDFilters{
			MatchIDs: []string{"m1"},
			XUIDs:    []string{"xuid_unknown_xyz"},
		})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("len(rows) = %d, want 0", len(rows))
	}
}
