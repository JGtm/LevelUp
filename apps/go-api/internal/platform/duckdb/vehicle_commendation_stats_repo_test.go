// Package duckdb — vehicle_commendation_stats_repo_test.go : tests :memory: de la
// lecture SCOPÉE des compteurs véhicules détruits / vol à la tire depuis les
// commendations natives (match_commendations) résolues par nom via
// commendation_definitions. Test interne (package duckdb) pour construire un PlayerDB
// minimal (Shared + Metadata) sans passer par openPlayerDB.
package duckdb

import (
	"context"
	"testing"
)

// seedVehicleCommendationFixture peuple les deux DB in-memory :
//   - metadata.commendation_definitions : 2 destructeurs de véhicule, 1 hijack, +
//     2 leurres qui NE doivent PAS être comptés (agrégat « Vehicle Mastery », splatter).
//   - shared.match_commendations : counts par (match, xuid, commendation) — dont des
//     lignes HORS scope (autre match, autre xuid) et une ligne « leurre » filtrée.
func seedVehicleCommendationFixture(t *testing.T, meta, shared *DB) {
	t.Helper()
	ctx := context.Background()

	metaDDL := []string{
		`CREATE TABLE commendation_definitions (
			commendation_id VARCHAR PRIMARY KEY,
			name_en VARCHAR,
			name_fr VARCHAR
		)`,
		`INSERT INTO commendation_definitions VALUES ('u-banshee','Banshee Destroyer','Destructeur de banshees')`,
		// Forme apostrophe : « Destructeur d'apparitions » doit matcher LIKE 'Destructeur %'.
		`INSERT INTO commendation_definitions VALUES ('u-wraith','Wraith Destroyer','Destructeur d''apparitions')`,
		`INSERT INTO commendation_definitions VALUES ('u-hijack','Grand Theft','Vol à la tire')`,
		// Leurres : agrégat véhicule (exclu explicitement) + splatter (ne matche aucun motif).
		`INSERT INTO commendation_definitions VALUES ('u-mastery','Vehicle Mastery','Maîtrise de véhicule')`,
		`INSERT INTO commendation_definitions VALUES ('u-splatter','Splatter','Écrasement')`,
	}
	for _, s := range metaDDL {
		if _, err := meta.Exec(ctx, s); err != nil {
			t.Fatalf("seed meta %q: %v", s, err)
		}
	}

	sharedDDL := []string{
		`CREATE TABLE match_commendations (
			match_id VARCHAR, xuid VARCHAR, commendation_id VARCHAR,
			count INTEGER, progress INTEGER,
			PRIMARY KEY (match_id, xuid, commendation_id)
		)`,
		// En scope (xA, m1..m3) : banshee 2+3=5, wraith 1 → véhicules=6 ; hijack 4.
		`INSERT INTO match_commendations VALUES ('m1','xA','u-banshee',2,2)`,
		`INSERT INTO match_commendations VALUES ('m2','xA','u-banshee',3,5)`,
		`INSERT INTO match_commendations VALUES ('m1','xA','u-wraith',1,1)`,
		`INSERT INTO match_commendations VALUES ('m2','xA','u-hijack',4,4)`,
		// Leurres/hors-scope (ne doivent PAS être comptés) :
		`INSERT INTO match_commendations VALUES ('m3','xA','u-splatter',9,9)`,    // splatter non résolu
		`INSERT INTO match_commendations VALUES ('m1','xA','u-mastery',77,77)`,   // agrégat exclu
		`INSERT INTO match_commendations VALUES ('m4','xA','u-banshee',100,100)`, // match hors scope
		`INSERT INTO match_commendations VALUES ('m1','xB','u-banshee',50,50)`,   // autre joueur
	}
	for _, s := range sharedDDL {
		if _, err := shared.Exec(ctx, s); err != nil {
			t.Fatalf("seed shared %q: %v", s, err)
		}
	}
}

func openVehicleTestMemDB(t *testing.T) *DB {
	t.Helper()
	db, err := OpenReadWrite(":memory:")
	if err != nil {
		t.Fatalf("OpenReadWrite(:memory:): %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestVehicleCommendationStats_ScopedSum(t *testing.T) {
	meta := openVehicleTestMemDB(t)
	shared := openVehicleTestMemDB(t)
	seedVehicleCommendationFixture(t, meta, shared)

	repo := NewVehicleCommendationStatsRepo(&PlayerDB{Shared: shared, Metadata: meta, XUID: "xA"})
	got, err := repo.LoadVehicleDestructionStats(context.Background(), "halo_5", []string{"m1", "m2", "m3"}, "xA")
	if err != nil {
		t.Fatalf("LoadVehicleDestructionStats: %v", err)
	}
	if got.VehiclesDestroyed != 6 {
		t.Errorf("VehiclesDestroyed = %d, want 6 (banshee 5 + wraith 1 ; splatter/mastery/m4/xB exclus)", got.VehiclesDestroyed)
	}
	if got.Hijacks != 4 {
		t.Errorf("Hijacks = %d, want 4", got.Hijacks)
	}
}

func TestVehicleCommendationStats_ReferentialAbsent(t *testing.T) {
	shared := openVehicleTestMemDB(t)
	// Metadata SANS table commendation_definitions → résolution échoue → 0/0 sans erreur.
	emptyMeta := openVehicleTestMemDB(t)
	repo := NewVehicleCommendationStatsRepo(&PlayerDB{Shared: shared, Metadata: emptyMeta, XUID: "xA"})
	got, err := repo.LoadVehicleDestructionStats(context.Background(), "halo_5", []string{"m1"}, "xA")
	if err != nil {
		t.Fatalf("attendu dégradation gracieuse (nil err), got %v", err)
	}
	if got.VehiclesDestroyed != 0 || got.Hijacks != 0 {
		t.Errorf("référentiel absent → attendu 0/0, got %+v", got)
	}
}

func TestVehicleCommendationStats_ReferentialEmpty(t *testing.T) {
	shared := openVehicleTestMemDB(t)
	meta := openVehicleTestMemDB(t)
	// Table présente mais VIDE (aucun nom ne matche) → 0/0 sans erreur.
	if _, err := meta.Exec(context.Background(),
		`CREATE TABLE commendation_definitions (commendation_id VARCHAR, name_en VARCHAR, name_fr VARCHAR)`); err != nil {
		t.Fatalf("ddl: %v", err)
	}
	repo := NewVehicleCommendationStatsRepo(&PlayerDB{Shared: shared, Metadata: meta, XUID: "xA"})
	got, err := repo.LoadVehicleDestructionStats(context.Background(), "halo_5", []string{"m1"}, "xA")
	if err != nil || got.VehiclesDestroyed != 0 || got.Hijacks != 0 {
		t.Errorf("référentiel vide → attendu 0/0 sans erreur, got %+v err=%v", got, err)
	}
}

func TestVehicleCommendationStats_NeutralInputs(t *testing.T) {
	meta := openVehicleTestMemDB(t)
	shared := openVehicleTestMemDB(t)
	seedVehicleCommendationFixture(t, meta, shared)
	repo := NewVehicleCommendationStatsRepo(&PlayerDB{Shared: shared, Metadata: meta, XUID: "xA"})
	ctx := context.Background()

	// xuid vide → 0/0 (pas d'accès DB).
	if got, err := repo.LoadVehicleDestructionStats(ctx, "halo_5", []string{"m1"}, "  "); err != nil || got.VehiclesDestroyed != 0 || got.Hijacks != 0 {
		t.Errorf("xuid vide → 0/0, got %+v err=%v", got, err)
	}
	// matchIDs vide → 0/0.
	if got, err := repo.LoadVehicleDestructionStats(ctx, "halo_5", nil, "xA"); err != nil || got.VehiclesDestroyed != 0 || got.Hijacks != 0 {
		t.Errorf("matchIDs vide → 0/0, got %+v err=%v", got, err)
	}
	// pdb nil → 0/0.
	nilRepo := NewVehicleCommendationStatsRepo(nil)
	if got, err := nilRepo.LoadVehicleDestructionStats(ctx, "halo_5", []string{"m1"}, "xA"); err != nil || got.VehiclesDestroyed != 0 || got.Hijacks != 0 {
		t.Errorf("pdb nil → 0/0, got %+v err=%v", got, err)
	}
}
