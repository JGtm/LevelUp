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
// Sprint B1 commit 8k.3 : double-write player+shared (cf. seedMedals). Le repo
// WeaponKillsRepo lit désormais via SharedReadDB() → pdb.Shared.
func seedWeaponKills(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	ctx := context.Background()

	// Joueur 1 : 4 kills BR (42), 2 kills AR (100) sur m1 ; 1 kill BR sur m2.
	// Joueur 2 : 3 kills BR sur m1.
	// Le seed écrit dans player ET shared pour rester compatible avec les tests
	// legacy (queries via pdb.Player) tout en alimentant SharedReadDB() (pdb.Shared).
	for _, db := range []*DB{pdb.Player, pdb.Shared} {
		// Joueur 1 m1 — 4 BR (42), 2 AR (100)
		for i := 0; i < 4; i++ {
			if _, err := db.Exec(ctx,
				`INSERT INTO shared.weapon_kills (match_id, xuid, weapon_id) VALUES (?,?,?)`,
				"m1", pTestXUID, uint64(42),
			); err != nil {
				t.Fatalf("seed wk m1 BR: %v", err)
			}
		}
		for i := 0; i < 2; i++ {
			if _, err := db.Exec(ctx,
				`INSERT INTO shared.weapon_kills (match_id, xuid, weapon_id) VALUES (?,?,?)`,
				"m1", pTestXUID, uint64(100),
			); err != nil {
				t.Fatalf("seed wk m1 AR: %v", err)
			}
		}

		// Ajouter m2 dans match_registry pour pouvoir y inserer des participants/kills
		if _, err := db.Exec(ctx,
			`INSERT INTO shared.match_registry (match_id, start_time)
			 VALUES (?, ?)`,
			"m2", "2025-01-11 14:00:00+00",
		); err != nil {
			t.Fatalf("seed match_registry m2: %v", err)
		}

		if _, err := db.Exec(ctx,
			`INSERT INTO shared.weapon_kills (match_id, xuid, weapon_id) VALUES (?,?,?)`,
			"m2", pTestXUID, uint64(42),
		); err != nil {
			t.Fatalf("seed wk m2: %v", err)
		}

		// Joueur 2 sur m1 : 3 kills BR uniquement
		if _, err := db.Exec(ctx,
			`INSERT INTO shared.match_participants
			 (match_id,xuid,gamertag,outcome,kills,deaths,team_id,grenade_kills,melee_kills)
			 VALUES (?,?,?,?,?,?,?,?,?)`,
			"m1", wkTestXUIDOther, "OtherPlayer", 3, 5, 5, 2, 1, 2,
		); err != nil {
			t.Fatalf("seed participant other: %v", err)
		}
		for i := 0; i < 3; i++ {
			if _, err := db.Exec(ctx,
				`INSERT INTO shared.weapon_kills (match_id, xuid, weapon_id) VALUES (?,?,?)`,
				"m1", wkTestXUIDOther, uint64(42),
			); err != nil {
				t.Fatalf("seed wk other: %v", err)
			}
		}

		// Mettre a jour les colonnes grenade/melee du joueur principal sur m1
		if _, err := db.Exec(ctx,
			`UPDATE shared.match_participants
			 SET grenade_kills = ?, melee_kills = ?
			 WHERE match_id = ? AND xuid = ?`,
			3, 4, "m1", pTestXUID,
		); err != nil {
			t.Fatalf("update grenade/melee: %v", err)
		}

		// Alias xuid pour le joueur 2 (necessaire pour Gamertag-based filter)
		if _, err := db.Exec(ctx,
			`INSERT INTO shared.xuid_aliases (xuid, gamertag) VALUES (?, ?)`,
			wkTestXUIDOther, "OtherPlayer",
		); err != nil {
			t.Fatalf("seed alias: %v", err)
		}
	}

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
	// Drop dans player ET shared : le repo lit via SharedReadDB() (pdb.Shared)
	// depuis le commit 8k.3.
	for _, db := range []*DB{pdb.Player, pdb.Shared} {
		if _, err := db.Exec(ctx, "DROP VIEW shared.v_weapon_kills"); err != nil {
			t.Fatalf("DROP VIEW: %v", err)
		}
		if _, err := db.Exec(ctx, "DROP TABLE shared.weapon_kills"); err != nil {
			t.Fatalf("DROP TABLE: %v", err)
		}
	}

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
