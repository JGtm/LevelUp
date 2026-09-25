package replay

// vehicle_shots_unit_test.go — UN TIR DONT L UNITE EST UN VEHICULE SE POSE SUR LUI (lot M4b.4,
// `vehicle_shots_unit.go`).

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// TestLeTirSansTireurSePoseSurLUniteEtSonOccupant — un record qui ne nomme pas de tireur mais dont
// la reference 0 est le vehicule 700 : pose sur lui, au nom de son occupant unique.
func TestLeTirSansTireurSePoseSurLUniteEtSonOccupant(t *testing.T) {
	doc := vsDoc(nil, []VehicleSample{{T: 10, X: 0, Y: 0}, {T: 30, X: 20, Y: 40}})
	o := vsOrphan(20, 0x11725DC400000000)
	o.ev.FilmIndex, o.ev.Unit = -1, grammar.UnitRef{Present: true, Slot: 700}
	attachVehicleShots(doc, []orphanShot{o}, vsOwn(), vsClock())
	if len(doc.Shots) != 1 || doc.Shots[0].Slot != 10 || *doc.Shots[0].Vehicle != 700 {
		t.Fatalf("tirs = %+v, attendu un tir du slot 10 pose sur 700", doc.Shots)
	}
	if c := doc.Coverage.Vehicles; c.ShotsByUnit != 1 || c.ShotsByUnitNoRide != 1 {
		t.Errorf("par l unite %d, sans episode %d : attendu 1, 1", c.ShotsByUnit, c.ShotsByUnitNoRide)
	}
}

// TestLUniteDesigneLeVehiculeQuandLEpisodeHesite — le tireur a un episode sur DEUX vehicules a la
// fois (artefact du pont) : l unite tranche, la ou l episode seul comptait un tir ambigu.
func TestLUniteDesigneLeVehiculeQuandLEpisodeHesite(t *testing.T) {
	doc := vsDoc(nil, []VehicleSample{{T: 10, X: 0, Y: 0}, {T: 30, X: 20, Y: 40}})
	seat := 0
	doc.Vehicles = append(doc.Vehicles, VehicleTrack{Slot: 701, Gen: 1, T0: 0, T1: 90, T1Max: 90,
		Samples: []VehicleSample{{T: 0, X: 500, Y: 500}},
		Rides:   []VehicleRide{{T0: 10, T1: 40, Slot: 10, Seat: &seat, Src: VehicleRideSrcProximity}}})
	o := vsOrphan(20, 0x11725DC400000000)
	o.ev.HasShooter, o.ev.Unit = true, grammar.UnitRef{Present: true, Slot: 701}
	attachVehicleShots(doc, []orphanShot{o}, vsOwn(), vsClock())
	if len(doc.Shots) != 1 || *doc.Shots[0].Vehicle != 701 || doc.Coverage.Vehicles.ShotsAmbiguous != 0 {
		t.Fatalf("tirs = %+v, ambigus %d : attendu un tir pose sur 701", doc.Shots, doc.Coverage.Vehicles.ShotsAmbiguous)
	}
	if c := doc.Coverage.Vehicles; c.ShotsByUnit != 1 || c.ShotsByUnitNoRide != 0 {
		t.Errorf("par l unite %d, sans episode %d : attendu 1, 0", c.ShotsByUnit, c.ShotsByUnitNoRide)
	}
}

// TestLaReferenceZeroTrancheEntreDeuxPiecesDuMemePorteur — INTEGRATION DE LA VAGUE D (2026-09-25),
// UNE SEULE REGLE : la reference 0 (M4b.4) passe AVANT l ambiguite des episodes (M7b, jugee sur le
// vehicule). Le meme tireur tenu par DEUX pieces distinctes du meme porteur est ambigu pour
// l episode seul (`TestTirDeDeuxPiecesDistinctesDuMemePorteurEstAmbigu`) ; quand le record nomme
// l unite tireuse (la piece 702), le film tranche : le tir sort du PORTEUR de cette piece.
func TestLaReferenceZeroTrancheEntreDeuxPiecesDuMemePorteur(t *testing.T) {
	doc := vsTourelle()
	seat := 0
	porteur := &VehicleLifeRef{Slot: 701, Gen: 1}
	doc.Vehicles[0].Carrier = porteur
	doc.Vehicles = append(doc.Vehicles, VehicleTrack{
		Slot: 702, Gen: 1, Chassis: "f4c45d71", T0: 0, T1: 90, T1Max: 90, Carrier: porteur,
		Spawn: &VehicleSpawn{X: 500, Y: 500},
		Rides: []VehicleRide{{T0: 10, T1: 40, Slot: 10, Seat: &seat, Src: VehicleRideSrcProximity}},
	})
	o := vsOrphan(30, 0x0BB6976B00000000)
	o.ev.HasShooter, o.ev.Unit = true, grammar.UnitRef{Present: true, Slot: 702}
	attachVehicleShots(doc, []orphanShot{o}, vsOwn(), vsClock())
	if len(doc.Shots) != 1 || *doc.Shots[0].Vehicle != 701 || doc.Shots[0].X != 30 {
		t.Fatalf("tirs = %+v, attendu un tir pose sur le porteur 701 en x = 30", doc.Shots)
	}
	c := doc.Coverage.Vehicles
	if c.ShotsAmbiguous != 0 || c.ShotsByUnit != 1 || c.ShotsOnCarrier != 1 {
		t.Errorf("ambigus %d, par l unite %d, sur le porteur %d : attendu 0, 1, 1",
			c.ShotsAmbiguous, c.ShotsByUnit, c.ShotsOnCarrier)
	}
}
