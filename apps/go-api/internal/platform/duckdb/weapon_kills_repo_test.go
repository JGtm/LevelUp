//go:build integration

// Package duckdb — weapon_kills_repo_test.go : tests WeaponKillsRepo.
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

const wkTestXUIDOther = "xuid_player_002"

// seedWeaponKills insere des kills d'arme pour deux joueurs sur deux matchs.
// IDs choisis : 42 (>2, valide) et 100 (>2, valide). 0/1/2 sont exclus par Q16.
//
// double-write player+shared (cf. seedMedals). Le repo
// WeaponKillsRepo lit désormais via SharedReadDB() → pdb.Shared.
func seedWeaponKills(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	ctx := context.Background()

	// Joueur 1 : 4 kills BR (42), 2 kills AR (100) sur m1 ; 1 kill BR sur m2.
	// Joueur 2 : 3 kills BR sur m1.
	wkInsert := `INSERT INTO shared.weapon_kills (match_id, xuid, weapon_id) VALUES (?,?,?)`
	for i := 0; i < 4; i++ {
		execOnSharedDBs(t, pdb, ctx, wkInsert, "m1", pTestXUID, uint64(42))
	}
	for i := 0; i < 2; i++ {
		execOnSharedDBs(t, pdb, ctx, wkInsert, "m1", pTestXUID, uint64(100))
	}

	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.match_registry (match_id, start_time) VALUES (?, ?)`,
		"m2", "2025-01-11 14:00:00+00")
	execOnSharedDBs(t, pdb, ctx, wkInsert, "m2", pTestXUID, uint64(42))

	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.match_participants
		 (match_id,xuid,gamertag,outcome,kills,deaths,team_id,grenade_kills,melee_kills)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		"m1", wkTestXUIDOther, "OtherPlayer", 3, 5, 5, 2, 1, 2)
	for i := 0; i < 3; i++ {
		execOnSharedDBs(t, pdb, ctx, wkInsert, "m1", wkTestXUIDOther, uint64(42))
	}

	execOnSharedDBs(t, pdb, ctx,
		`UPDATE shared.match_participants
		 SET grenade_kills = ?, melee_kills = ?
		 WHERE match_id = ? AND xuid = ?`,
		3, 4, "m1", pTestXUID)

	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.xuid_aliases (xuid, gamertag) VALUES (?, ?)`,
		wkTestXUIDOther, "OtherPlayer")

	// Inserer le label FR pour weapon_id 100 dans metadata (1 seule fois — séparé)
	if _, err := pdb.Metadata.Exec(ctx,
		`INSERT INTO weapon_labels (weapon_id,name_en,name_fr) VALUES (?,?,?)`,
		uint64(100), "Assault Rifle", "AR75",
	); err != nil {
		t.Fatalf("seed weapon_label 100: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

func TestWeaponKillsRepo_Load_RejectsEmptyMatchIDs(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewWeaponKillsRepo(pdb)
	_, err := repo.LoadWeaponKillsAggregated(context.Background(), "halo_infinite",
		port.WeaponKillFilters{Gamertag: pTestGamertag})
	if err == nil {
		t.Fatal("attendu erreur (MatchIDs vide)")
	}
	if !errors.Is(err, port.ErrWeaponKillFiltersTooBroad) {
		t.Errorf("err = %v, want ErrWeaponKillFiltersTooBroad", err)
	}
}

func TestWeaponKillsRepo_Load_RejectsNoXUIDOrGamertag(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewWeaponKillsRepo(pdb)
	_, err := repo.LoadWeaponKillsAggregated(context.Background(), "halo_infinite",
		port.WeaponKillFilters{MatchIDs: []string{"m1"}})
	if err == nil {
		t.Fatal("attendu erreur (Gamertag/XUIDs vide)")
	}
	if !errors.Is(err, port.ErrWeaponKillFiltersTooBroad) {
		t.Errorf("err = %v, want ErrWeaponKillFiltersTooBroad", err)
	}
}

func TestWeaponKillsRepo_Load_RejectsNegativeMinKills(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewWeaponKillsRepo(pdb)
	_, err := repo.LoadWeaponKillsAggregated(context.Background(), "halo_infinite",
		port.WeaponKillFilters{
			MatchIDs: []string{"m1"},
			Gamertag: pTestGamertag,
			MinKills: -1,
		})
	if err == nil {
		t.Fatal("attendu erreur (MinKills negatif)")
	}
	if !errors.Is(err, port.ErrWeaponKillFiltersInvalid) {
		t.Errorf("err = %v, want ErrWeaponKillFiltersInvalid", err)
	}
}

// ---------------------------------------------------------------------------
// Capability gating
// ---------------------------------------------------------------------------

func TestWeaponKillsRepo_Load_CapabilityNotSupported_NoTable(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	execOnSharedDBs(t, pdb, ctx, "DROP VIEW shared.v_weapon_kills")
	execOnSharedDBs(t, pdb, ctx, "DROP TABLE shared.weapon_kills")

	repo := NewWeaponKillsRepo(pdb)
	_, err := repo.LoadWeaponKillsAggregated(ctx, "halo_infinite",
		port.WeaponKillFilters{
			MatchIDs: []string{"m1"},
			Gamertag: pTestGamertag,
		})
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("err = %v, want ErrCapabilityNotSupported", err)
	}
}

// ---------------------------------------------------------------------------
// Querying
// ---------------------------------------------------------------------------

func TestWeaponKillsRepo_Load_ByGamertag_AggregatesByWeapon(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedWeaponKills(t, pdb)

	repo := NewWeaponKillsRepo(pdb)
	rows, err := repo.LoadWeaponKillsAggregated(context.Background(), "halo_infinite",
		port.WeaponKillFilters{
			MatchIDs: []string{"m1", "m2"},
			Gamertag: pTestGamertag,
		})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Joueur 1 : 5 kills BR (4 m1 + 1 m2), 2 kills AR (100, m1).
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}

	got := map[int64]int{}
	gotLabels := map[int64]string{}
	for _, r := range rows {
		if r.XUID != pTestXUID {
			t.Errorf("XUID = %q, want %q", r.XUID, pTestXUID)
		}
		if r.IsGrenadeMelee {
			t.Errorf("IsGrenadeMelee=true sans IncludeGrenadeMelee")
		}
		got[r.WeaponID] = r.Kills
		gotLabels[r.WeaponID] = r.Label
	}
	if got[42] != 5 {
		t.Errorf("kills[BR=42] = %d, want 5", got[42])
	}
	if got[100] != 2 {
		t.Errorf("kills[AR=100] = %d, want 2", got[100])
	}
	if gotLabels[100] != "AR75" {
		t.Errorf("label[100] = %q, want AR75", gotLabels[100])
	}
}

func TestWeaponKillsRepo_Load_ByXUIDs_MultiPlayers(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedWeaponKills(t, pdb)

	repo := NewWeaponKillsRepo(pdb)
	rows, err := repo.LoadWeaponKillsAggregated(context.Background(), "halo_infinite",
		port.WeaponKillFilters{
			MatchIDs: []string{"m1"},
			XUIDs:    []string{pTestXUID, wkTestXUIDOther},
		})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// p1 m1 : BR 4, AR 2 ; p2 m1 : BR 3 -> 3 lignes
	if len(rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3 (p1 BR/AR + p2 BR), rows=%+v", len(rows), rows)
	}
}

func TestWeaponKillsRepo_Load_MinKillsFilter(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedWeaponKills(t, pdb)

	repo := NewWeaponKillsRepo(pdb)
	rows, err := repo.LoadWeaponKillsAggregated(context.Background(), "halo_infinite",
		port.WeaponKillFilters{
			MatchIDs: []string{"m1"},
			Gamertag: pTestGamertag,
			MinKills: 3,
		})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// p1 m1 : BR=4 (>=3) garde, AR=2 (<3) filtre.
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if rows[0].WeaponID != 42 || rows[0].Kills != 4 {
		t.Errorf("got %+v, want WeaponID=42 Kills=4", rows[0])
	}
}

func TestWeaponKillsRepo_Load_IncludeGrenadeMelee(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedWeaponKills(t, pdb)

	repo := NewWeaponKillsRepo(pdb)
	rows, err := repo.LoadWeaponKillsAggregated(context.Background(), "halo_infinite",
		port.WeaponKillFilters{
			MatchIDs:            []string{"m1"},
			Gamertag:            pTestGamertag,
			IncludeGrenadeMelee: true,
		})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// p1 m1 : BR (4, false), AR (2, false), grenade sentinel 0 (3, true), melee sentinel 1 (4, true)
	gmCount := 0
	armes := 0
	gmTotals := map[int64]int{}
	for _, r := range rows {
		if r.IsGrenadeMelee {
			gmCount++
			gmTotals[r.WeaponID] = r.Kills
		} else {
			armes++
		}
	}
	if armes != 2 {
		t.Errorf("armes = %d, want 2", armes)
	}
	if gmCount != 2 {
		t.Errorf("grenade/melee rows = %d, want 2", gmCount)
	}
	if gmTotals[weaponIDGrenadeSentinel] != 3 {
		t.Errorf("grenade kills = %d, want 3", gmTotals[weaponIDGrenadeSentinel])
	}
	if gmTotals[weaponIDMeleeSentinel] != 4 {
		t.Errorf("melee kills = %d, want 4", gmTotals[weaponIDMeleeSentinel])
	}
}

// TestWeaponKillsRepo_Load_MechanicKills_H5KillKind (V72-15.3) : la colonne mechanic_kills
// compte, par arme, les kills dont kill_kind <> 'weapon' (mêlée/assassinat attribués à
// l'arme TENUE sur H5). Kills reste le total (breakdown inchangé) ; mechanic_kills alimente
// le retrait anti-double-comptage côté fragdist. kill_kind NULL (Infinite) → 0.
func TestWeaponKillsRepo_Load_MechanicKills_H5KillKind(t *testing.T) {
	pdb := newTestPlayerDB(t) // seed pTestXUID <-> pTestGamertag dans xuid_aliases
	ctx := context.Background()
	wkInsert := `INSERT INTO shared.weapon_kills (match_id, xuid, weapon_id, kill_kind) VALUES (?,?,?,?)`
	// Arme tenue 77 : 2 kills d'arme + 2 mêlées + 1 assassinat (tous attribués à 77).
	execOnSharedDBs(t, pdb, ctx, wkInsert, "mh5", pTestXUID, uint64(77), "weapon")
	execOnSharedDBs(t, pdb, ctx, wkInsert, "mh5", pTestXUID, uint64(77), "weapon")
	execOnSharedDBs(t, pdb, ctx, wkInsert, "mh5", pTestXUID, uint64(77), "melee")
	execOnSharedDBs(t, pdb, ctx, wkInsert, "mh5", pTestXUID, uint64(77), "melee")
	execOnSharedDBs(t, pdb, ctx, wkInsert, "mh5", pTestXUID, uint64(77), "assassination")

	repo := NewWeaponKillsRepo(pdb)
	rows, err := repo.LoadWeaponKillsAggregated(ctx, "halo_infinite",
		port.WeaponKillFilters{MatchIDs: []string{"mh5"}, Gamertag: pTestGamertag})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if rows[0].Kills != 5 {
		t.Errorf("Kills = %d, want 5 (total inchangé)", rows[0].Kills)
	}
	if rows[0].MechanicKills != 3 {
		t.Errorf("MechanicKills = %d, want 3 (2 mêlées + 1 assassinat, kill_kind <> 'weapon')", rows[0].MechanicKills)
	}
}

func TestWeaponKillsRepo_Load_NoResults(t *testing.T) {
	pdb := newTestPlayerDB(t)
	// Pas de seedWeaponKills : la table existe mais vide
	repo := NewWeaponKillsRepo(pdb)
	rows, err := repo.LoadWeaponKillsAggregated(context.Background(), "halo_infinite",
		port.WeaponKillFilters{
			MatchIDs: []string{"m1"},
			Gamertag: pTestGamertag,
		})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("len(rows) = %d, want 0", len(rows))
	}
}
