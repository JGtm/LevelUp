package replay

// vehicle_shots_test.go — LA SECONDE PORTE DES TIRS, testee SANS film : c'est de la logique
// pure (`vehicle_shots.go`), et elle doit se verifier sans les trois minutes de decodage que
// coute un instrument. Les chiffres de terrain, eux, vivent dans `vehicules_v4_tirs_test.go`.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// vsClock est l'horloge des cas : origine 0, pas 100 ms, 1 000 frames.
func vsClock() replayClock {
	return replayClock{origin: 0, step: 100_000, frames: 1000}
}

// vsDoc monte un document minimal : une vie de vehicule occupee par le slot 10, une trajectoire
// publiee pour ce slot, et une couverture de tirs equilibree.
func vsDoc(rides []VehicleRide, samples []VehicleSample) *ReplayDocument {
	seat := 0
	if len(rides) == 0 {
		rides = []VehicleRide{{T0: 10, T1: 40, Slot: 10, Seat: &seat, Src: VehicleRideSrcGap}}
	}
	return &ReplayDocument{
		Tracks:   []Track{{Slot: 10}},
		Vehicles: []VehicleTrack{{Slot: 700, Gen: 1, T0: 0, T1: 90, T1Max: 90, Samples: samples, Rides: rides}},
		Coverage: &Coverage{
			Shots:    LayerCoverage{Available: 3, Attached: 1, NoSlot: 2},
			Vehicles: &VehicleCoverage{UnknownChassis: map[string]int{}},
			Verdict:  map[string]string{"shots": "partiel : moins des deux tiers rattachés"},
		},
	}
}

// vsOrphan fabrique un orphelin « sans slot » du joueur 3, a la frame demandee.
func vsOrphan(frame int, weapon uint64) orphanShot {
	return orphanShot{
		ev: filmdec.FireEvent{
			TimestampUS: uint64(frame) * 100_000, FilmIndex: 3, WeaponID: weapon,
		},
		reason: reasonNoSlot,
	}
}

func vsOwn() OwnerReport { return OwnerReport{Owner: map[uint32]int{10: 3}} }

// TestTirEnVehiculePosePendantUnEpisode — LE CAS NOMINAL : le tir tombe dans l'episode, il sort
// a la position INTERPOLEE du vehicule et porte le slot de celui-ci.
func TestTirEnVehiculePosePendantUnEpisode(t *testing.T) {
	doc := vsDoc(nil, []VehicleSample{{T: 10, X: 0, Y: 0}, {T: 30, X: 20, Y: 40}})
	attachVehicleShots(doc, []orphanShot{vsOrphan(20, 0x11725DC400000000)}, vsOwn(), vsClock())
	if len(doc.Shots) != 1 {
		t.Fatalf("tirs publies = %d, attendu 1", len(doc.Shots))
	}
	s := doc.Shots[0]
	if s.Vehicle == nil || *s.Vehicle != 700 {
		t.Errorf("marqueur vehicule = %v, attendu 700", s.Vehicle)
	}
	// A mi-chemin des deux echantillons : l'interpolation, pas le plus proche.
	if s.X != 10 || s.Y != 20 {
		t.Errorf("position = (%v, %v), attendu (10, 20) — interpolation entre les echantillons", s.X, s.Y)
	}
	if s.Slot != 10 || s.Weapon != "0x11725DC400000000" {
		t.Errorf("tireur/arme = %d/%q, attendu 10/0x11725DC400000000", s.Slot, s.Weapon)
	}
	c := doc.Coverage.Shots
	if c.Attached != 2 || c.NoSlot != 1 {
		t.Errorf("couverture = attached %d / noSlot %d, attendu 2 / 1", c.Attached, c.NoSlot)
	}
	if !c.Balanced() {
		t.Errorf("l'invariant de couverture est rompu : %+v", c)
	}
	if doc.Coverage.Verdict["shots"] != VerdictNominal {
		t.Errorf("verdict = %q, attendu recalcule apres la seconde porte", doc.Coverage.Verdict["shots"])
	}
	if doc.Coverage.Vehicles.Shots != 1 {
		t.Errorf("coverage.vehicles.shots = %d, attendu 1", doc.Coverage.Vehicles.Shots)
	}
}

// TestTirHorsEpisodeResteOrphelin — un orphelin qu'aucun episode ne couvre ne bouge PAS : ni
// publie, ni sorti de son compteur de rejet. C'est le cas nominal d'un tir a pied que le pont
// n'a pas su placer.
func TestTirHorsEpisodeResteOrphelin(t *testing.T) {
	doc := vsDoc(nil, []VehicleSample{{T: 10, X: 0, Y: 0}, {T: 30, X: 20, Y: 40}})
	attachVehicleShots(doc, []orphanShot{vsOrphan(80, 0)}, vsOwn(), vsClock())
	if len(doc.Shots) != 0 {
		t.Fatalf("tirs publies = %d, attendu 0", len(doc.Shots))
	}
	if c := doc.Coverage.Shots; c.Attached != 1 || c.NoSlot != 2 || !c.Balanced() {
		t.Errorf("couverture modifiee alors que rien n'a ete rattache : %+v", c)
	}
	if doc.Coverage.Vehicles.ShotsNoRide != 1 {
		t.Errorf("shotsNoRide = %d, attendu 1", doc.Coverage.Vehicles.ShotsNoRide)
	}
}

// TestTirAmbiguDeuxVehicules — DEUX vehicules distincts portent un episode du meme tireur au
// meme instant : on ne tranche pas, on compte. Publier l'un des deux affirmerait ce que la
// mesure ne dit pas.
func TestTirAmbiguDeuxVehicules(t *testing.T) {
	seat := 0
	doc := vsDoc(nil, []VehicleSample{{T: 10, X: 0, Y: 0}})
	doc.Vehicles = append(doc.Vehicles, VehicleTrack{
		Slot: 800, Gen: 1, T1Max: 90,
		Samples: []VehicleSample{{T: 10, X: 50, Y: 50}},
		Rides:   []VehicleRide{{T0: 10, T1: 40, Slot: 10, Seat: &seat, Src: VehicleRideSrcGap}},
	})
	attachVehicleShots(doc, []orphanShot{vsOrphan(20, 0)}, vsOwn(), vsClock())
	if len(doc.Shots) != 0 {
		t.Fatalf("tirs publies = %d, attendu 0 (ambigu)", len(doc.Shots))
	}
	if doc.Coverage.Vehicles.ShotsAmbiguous != 1 {
		t.Errorf("shotsAmbiguous = %d, attendu 1", doc.Coverage.Vehicles.ShotsAmbiguous)
	}
	if c := doc.Coverage.Shots; !c.Balanced() {
		t.Errorf("l'invariant de couverture est rompu : %+v", c)
	}
}

// TestTirEnVehiculeSansTrajectoirePubliee — MEME PORTE QUE LES TIRS A PIED : sans trajectoire
// publiee pour le tireur, le tir change de compteur (Unpublished) au lieu d'etre publie.
func TestTirEnVehiculeSansTrajectoirePubliee(t *testing.T) {
	doc := vsDoc(nil, []VehicleSample{{T: 10, X: 0, Y: 0}})
	doc.Tracks = nil
	attachVehicleShots(doc, []orphanShot{vsOrphan(20, 0)}, vsOwn(), vsClock())
	if len(doc.Shots) != 0 {
		t.Fatalf("tirs publies = %d, attendu 0", len(doc.Shots))
	}
	c := doc.Coverage.Shots
	if c.Attached != 1 || c.NoSlot != 1 || c.Unpublished != 1 || !c.Balanced() {
		t.Errorf("couverture = %+v, attendu noSlot -> unpublished sans rompre l'invariant", c)
	}
}

// TestVehiclePosAtBornes — la position TENUE hors de la plage des echantillons, et la NAISSANCE
// quand le vehicule n'a jamais bouge. Extrapoler ferait sortir le tir de la carte.
func TestVehiclePosAtBornes(t *testing.T) {
	tr := VehicleTrack{Samples: []VehicleSample{{T: 10, X: 1, Y: 2}, {T: 20, X: 3, Y: 4}}}
	for _, tc := range []struct {
		frame int
		x, y  float32
	}{
		{0, 1, 2}, {10, 1, 2}, {15, 2, 3}, {20, 3, 4}, {99, 3, 4},
	} {
		x, y, ok := vehiclePosAt(tr, tc.frame)
		if !ok || x != tc.x || y != tc.y {
			t.Errorf("frame %d -> (%v, %v, %v), attendu (%v, %v, true)", tc.frame, x, y, ok, tc.x, tc.y)
		}
	}
	spawn := VehicleTrack{Spawn: &VehicleSpawn{X: 7, Y: 8}}
	if x, y, ok := vehiclePosAt(spawn, 50); !ok || x != 7 || y != 8 {
		t.Errorf("sans echantillon : (%v, %v, %v), attendu la naissance (7, 8, true)", x, y, ok)
	}
	if _, _, ok := vehiclePosAt(VehicleTrack{}, 50); ok {
		t.Errorf("ni echantillon ni naissance : la position ne doit PAS etre inventee")
	}
}

// TestTirEnVehiculeSansEpisodeNeTouchePasLeDocument — garde de non-regression : un document sans
// vehicule (film d'arene) sort strictement inchange.
func TestTirEnVehiculeSansEpisodeNeTouchePasLeDocument(t *testing.T) {
	doc := vsDoc(nil, nil)
	doc.Vehicles = nil
	avant := doc.Coverage.Shots
	attachVehicleShots(doc, []orphanShot{vsOrphan(20, 0)}, vsOwn(), vsClock())
	if doc.Coverage.Shots != avant || len(doc.Shots) != 0 {
		t.Errorf("document modifie sans aucun vehicule : %+v", doc.Coverage.Shots)
	}
}
