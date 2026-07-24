// Package duckdb — vehicle_commendation_stats_repo_test.go : tests :memory: de la
// lecture SCOPÉE des compteurs véhicules détruits (commendations natives,
// match_commendations) et vol à la tire / hijacks (médailles, medals_earned,
// résolues par nom via medal_definitions avec repli sur les ids constants
// documentés). Test interne (package duckdb) pour construire un PlayerDB minimal
// (Shared + Metadata) sans passer par openPlayerDB.
package duckdb

import (
	"context"
	"testing"
)

// seedVehicleCommendationFixture peuple les deux DB in-memory :
//   - metadata.commendation_definitions : 2 destructeurs de véhicule, 1 « Grand
//     Theft » DÉCOY (ne doit PLUS être compté — hijack vient des médailles
//     maintenant, cf. package doc), + 2 autres leurres qui NE doivent PAS être
//     comptés (agrégat « Vehicle Mastery », splatter).
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
		// DÉCOY : cette commendation n'existe PAS réellement dans le référentiel H5
		// prod (fact-check corpus 2026-07-23) — conservée ici pour PROUVER qu'elle
		// n'est plus résolue comme hijack (ni comme véhicule détruit, son nom ne
		// matche aucun motif « Destructeur »/« Destroyer »).
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
		// En scope (xA, m1..m3) : banshee 2+3=5, wraith 1 → véhicules=6.
		`INSERT INTO match_commendations VALUES ('m1','xA','u-banshee',2,2)`,
		`INSERT INTO match_commendations VALUES ('m2','xA','u-banshee',3,5)`,
		`INSERT INTO match_commendations VALUES ('m1','xA','u-wraith',1,1)`,
		// DÉCOY en scope : ne doit PAS remonter dans Hijacks (plus de source hijack
		// côté commendations) ni dans VehiclesDestroyed (nom hors motif).
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

// seedHijackMedalFixture peuple metadata.medal_definitions (résolution PAR NOM,
// avec des ids VOLONTAIREMENT différents des constantes documentées — prouve que
// la résolution par nom prime, pas un simple passthrough) + shared.medals_earned
// (counts scopés + leurres hors scope/autre joueur/autre médaille).
func seedHijackMedalFixture(t *testing.T, meta, shared *DB) {
	t.Helper()
	ctx := context.Background()

	metaDDL := []string{
		`CREATE TABLE medal_definitions (
			medal_name_id BIGINT PRIMARY KEY,
			name_en VARCHAR
		)`,
		`INSERT INTO medal_definitions VALUES (777001, 'Hijack')`,
		`INSERT INTO medal_definitions VALUES (777002, 'Skyjack')`,
		// Leurre : autre médaille, ne doit pas être résolue en hijack.
		`INSERT INTO medal_definitions VALUES (777003, 'Killjoy')`,
	}
	for _, s := range metaDDL {
		if _, err := meta.Exec(ctx, s); err != nil {
			t.Fatalf("seed meta %q: %v", s, err)
		}
	}

	sharedDDL := []string{
		`CREATE TABLE medals_earned (
			match_id VARCHAR, xuid VARCHAR, medal_name_id BIGINT, count SMALLINT,
			PRIMARY KEY (match_id, xuid, medal_name_id)
		)`,
		// En scope (xA, m1..m3) : Hijack 3+2=5, Skyjack 1 → hijacks=6.
		`INSERT INTO medals_earned (match_id, xuid, medal_name_id, count) VALUES ('m1','xA',777001,3)`,
		`INSERT INTO medals_earned (match_id, xuid, medal_name_id, count) VALUES ('m2','xA',777001,2)`,
		`INSERT INTO medals_earned (match_id, xuid, medal_name_id, count) VALUES ('m1','xA',777002,1)`,
		// Leurres/hors-scope (ne doivent PAS être comptés) :
		`INSERT INTO medals_earned (match_id, xuid, medal_name_id, count) VALUES ('m1','xA',777003,50)`,  // autre médaille (Killjoy)
		`INSERT INTO medals_earned (match_id, xuid, medal_name_id, count) VALUES ('m4','xA',777001,100)`, // match hors scope
		`INSERT INTO medals_earned (match_id, xuid, medal_name_id, count) VALUES ('m1','xB',777001,50)`,  // autre joueur
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
		t.Errorf("VehiclesDestroyed = %d, want 6 (banshee 5 + wraith 1 ; splatter/mastery/hijack-décoy/m4/xB exclus)", got.VehiclesDestroyed)
	}
	// Aucune table medal_definitions/medals_earned dans cette fixture : le repli sur
	// les ids constants documentés est posé, mais medals_earned est absent →
	// dégradation gracieuse à 0 (cf. TestVehicleCommendationStats_HijackMedals_*
	// pour la source réelle). Prouve aussi que le décoy commendation 'u-hijack' (4,
	// en scope m2) n'est PLUS compté comme hijack.
	if got.Hijacks != 0 {
		t.Errorf("Hijacks = %d, want 0 (plus de source hijack commendation ; medals_earned absent ici)", got.Hijacks)
	}
}

func TestVehicleCommendationStats_ReferentialAbsent(t *testing.T) {
	shared := openVehicleTestMemDB(t)
	// Metadata SANS table commendation_definitions NI medal_definitions → résolution
	// véhicule échoue (0 UUID) ; résolution hijack retombe sur les ids constants mais
	// medals_earned est absent côté shared → 0/0 sans erreur.
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

// TestVehicleCommendationStats_HijackMedals_ScopedSum : somme scopée (xuid +
// matchIDs) de Hijack + Skyjack, résolus PAR NOM via medal_definitions (ids ≠
// constantes documentées dans cette fixture — prouve la priorité nominale).
// Exclusion vérifiée : autre médaille (Killjoy), autre match, autre joueur.
func TestVehicleCommendationStats_HijackMedals_ScopedSum(t *testing.T) {
	meta := openVehicleTestMemDB(t)
	shared := openVehicleTestMemDB(t)
	seedHijackMedalFixture(t, meta, shared)

	repo := NewVehicleCommendationStatsRepo(&PlayerDB{Shared: shared, Metadata: meta, XUID: "xA"})
	got, err := repo.LoadVehicleDestructionStats(context.Background(), "halo_5", []string{"m1", "m2", "m3"}, "xA")
	if err != nil {
		t.Fatalf("LoadVehicleDestructionStats: %v", err)
	}
	if got.Hijacks != 6 {
		t.Errorf("Hijacks = %d, want 6 (Hijack 3+2=5 + Skyjack 1 ; Killjoy/m4/xB exclus)", got.Hijacks)
	}
	if got.VehiclesDestroyed != 0 {
		t.Errorf("VehiclesDestroyed = %d, want 0 (aucune commendation_definitions dans cette fixture)", got.VehiclesDestroyed)
	}
}

// TestVehicleCommendationStats_HijackMedals_ConstantFallback : medal_definitions
// absent (aucune table) → repli sur les ids constants documentés
// (hijackMedalNamesToConstantIDs) — vérifie que la somme scopée fonctionne bien
// end-to-end avec CES ids-là, pas seulement que la résolution ne plante pas.
func TestVehicleCommendationStats_HijackMedals_ConstantFallback(t *testing.T) {
	emptyMeta := openVehicleTestMemDB(t) // pas de medal_definitions → repli constant
	shared := openVehicleTestMemDB(t)
	ctx := context.Background()
	ddl := []string{
		`CREATE TABLE medals_earned (
			match_id VARCHAR, xuid VARCHAR, medal_name_id BIGINT, count SMALLINT,
			PRIMARY KEY (match_id, xuid, medal_name_id)
		)`,
		// Constantes documentées (package doc vehicle_commendation_stats_repo.go) :
		// Hijack=1219497744, Skyjack=1801925525.
		`INSERT INTO medals_earned (match_id, xuid, medal_name_id, count) VALUES ('m1','xA',1219497744,4)`,
		`INSERT INTO medals_earned (match_id, xuid, medal_name_id, count) VALUES ('m1','xA',1801925525,1)`,
		`INSERT INTO medals_earned (match_id, xuid, medal_name_id, count) VALUES ('m1','xB',1219497744,9)`, // autre joueur
	}
	for _, s := range ddl {
		if _, err := shared.Exec(ctx, s); err != nil {
			t.Fatalf("seed shared %q: %v", s, err)
		}
	}

	repo := NewVehicleCommendationStatsRepo(&PlayerDB{Shared: shared, Metadata: emptyMeta, XUID: "xA"})
	got, err := repo.LoadVehicleDestructionStats(ctx, "halo_5", []string{"m1"}, "xA")
	if err != nil {
		t.Fatalf("LoadVehicleDestructionStats: %v", err)
	}
	if got.Hijacks != 5 {
		t.Errorf("Hijacks (repli constantes) = %d, want 5 (4+1 ; xB exclu)", got.Hijacks)
	}
}

// TestVehicleCommendationStats_HijackMedals_ReferentialAbsent : ni
// medal_definitions ni medals_earned n'existent → dégradation gracieuse (0, nil
// err) — mission explicite (jamais de 500 pour cette fun-stat annexe).
func TestVehicleCommendationStats_HijackMedals_ReferentialAbsent(t *testing.T) {
	shared := openVehicleTestMemDB(t)    // pas de table medals_earned
	emptyMeta := openVehicleTestMemDB(t) // pas de table medal_definitions
	repo := NewVehicleCommendationStatsRepo(&PlayerDB{Shared: shared, Metadata: emptyMeta, XUID: "xA"})
	got, err := repo.LoadVehicleDestructionStats(context.Background(), "halo_5", []string{"m1"}, "xA")
	if err != nil {
		t.Fatalf("attendu dégradation gracieuse (nil err), got %v", err)
	}
	if got.Hijacks != 0 {
		t.Errorf("référentiel + medals_earned absents → attendu Hijacks=0, got %d", got.Hijacks)
	}
}
