//go:build integration

// Package duckdb — weapon_accuracy_repo_test.go : tests WeaponAccuracyRepo.
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

const waTestXUIDOther = "xuid_player_002"

// seedWeaponAccuracy insère des rows de précision pour deux joueurs / deux matchs.
// Joueur 1 (pTestXUID) : weapon 42 → 60 tirés / 30 touchés sur m1, +40/30 sur m2
//
//	(agrégé : 100/60 = 0.60) ; weapon 100 → 50/45 sur m1 (0.90).
//
// Joueur 2 : weapon 42 → 80/8 sur m1 (0.10) — ne doit PAS remonter pour le J1.
func seedWeaponAccuracy(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	ctx := context.Background()
	ins := `INSERT INTO shared.weapon_accuracy (match_id, xuid, weapon_id, shots_fired, shots_landed, drops) VALUES (?,?,?,?,?,?)`

	execOnSharedDBs(t, pdb, ctx, ins, "m1", pTestXUID, uint64(42), 60, 30, 1)
	execOnSharedDBs(t, pdb, ctx, ins, "m2", pTestXUID, uint64(42), 40, 30, 1)
	execOnSharedDBs(t, pdb, ctx, ins, "m1", pTestXUID, uint64(100), 50, 45, 1)
	execOnSharedDBs(t, pdb, ctx, ins, "m1", waTestXUIDOther, uint64(42), 80, 8, 1)

	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.xuid_aliases (xuid, gamertag) VALUES (?, ?)`,
		waTestXUIDOther, "OtherPlayer")
	// Label FR pour weapon 100 (parité weapon_labels-first ; 42 reste sans label).
	if _, err := pdb.Metadata.Exec(ctx,
		`INSERT INTO weapon_labels (weapon_id,name_en,name_fr) VALUES (?,?,?)`,
		uint64(100), "Magnum", "Pistolet"); err != nil {
		t.Fatalf("seed weapon_label 100: %v", err)
	}
	if _, err := pdb.Metadata.Exec(ctx,
		`INSERT INTO weapon_labels (weapon_id,name_en,name_fr) VALUES (?,?,?)`,
		uint64(42), "BR75", "BR75"); err != nil {
		t.Fatalf("seed weapon_label 42: %v", err)
	}
}

func TestWeaponAccuracyRepo_Load_RejectsBroadFilters(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewWeaponAccuracyRepo(pdb)
	_, err := repo.LoadWeaponAccuracyAggregated(context.Background(), "halo_5",
		port.WeaponAccuracyFilters{Gamertag: pTestGamertag})
	if !errors.Is(err, port.ErrWeaponAccuracyFiltersTooBroad) {
		t.Errorf("MatchIDs vide → err = %v, want ErrWeaponAccuracyFiltersTooBroad", err)
	}
}

// TestWeaponAccuracyRepo_Load_AggregatesPerWeapon vérifie l'agrégation SQL :
// SUM par (xuid, weapon_id), filtre gamertag→xuid (xuid_aliases), label résolu.
func TestWeaponAccuracyRepo_Load_AggregatesPerWeapon(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedWeaponAccuracy(t, pdb)
	repo := NewWeaponAccuracyRepo(pdb)

	rows, err := repo.LoadWeaponAccuracyAggregated(context.Background(), "halo_5",
		port.WeaponAccuracyFilters{MatchIDs: []string{"m1", "m2"}, Gamertag: pTestGamertag})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// 2 armes pour pTestXUID (42 et 100) ; le joueur 2 est exclu.
	if len(rows) != 2 {
		t.Fatalf("len = %d, want 2 (%+v)", len(rows), rows)
	}
	byWeapon := map[int64]port.WeaponAccuracyRow{}
	for _, r := range rows {
		byWeapon[r.WeaponID] = r
	}
	w42 := byWeapon[42]
	if w42.ShotsFired != 100 || w42.ShotsLanded != 60 {
		t.Errorf("weapon 42 = %d/%d, want 100/60 (agrégé m1+m2)", w42.ShotsLanded, w42.ShotsFired)
	}
	if w42.Label != "BR75" {
		t.Errorf("weapon 42 label = %q, want BR75", w42.Label)
	}
	w100 := byWeapon[100]
	if w100.ShotsFired != 50 || w100.ShotsLanded != 45 || w100.Label != "Pistolet" {
		t.Errorf("weapon 100 = %d/%d %q, want 45/50 Pistolet", w100.ShotsLanded, w100.ShotsFired, w100.Label)
	}
}

// TestWeaponAccuracyRepo_Load_MissingTable simule un titre sans la table
// (capability absente) → games.ErrCapabilityNotSupported.
func TestWeaponAccuracyRepo_Load_MissingTable(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	// Le repo lit via SharedReadDB() → conn `player`, où weapon_accuracy est la vue
	// bridge. La supprimer simule un titre sans la table (capability absente).
	db, release, err := pdb.SharedReadDB().Get(ctx)
	if err != nil {
		t.Fatalf("shared reader: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DROP VIEW weapon_accuracy`); err != nil {
		t.Fatalf("drop bridge view: %v", err)
	}
	release()
	repo := NewWeaponAccuracyRepo(pdb)
	_, err = repo.LoadWeaponAccuracyAggregated(ctx, "halo_infinite",
		port.WeaponAccuracyFilters{MatchIDs: []string{"m1"}, Gamertag: pTestGamertag})
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("table absente → err = %v, want ErrCapabilityNotSupported", err)
	}
}
