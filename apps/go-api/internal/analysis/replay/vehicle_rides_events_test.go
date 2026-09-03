package replay

// vehicle_rides_events_test.go — GARDE-RAILS de la machine d etats d occupation. Sur fixtures,
// SANS environnement ni film : ils tournent dans la suite ordinaire.
//
// CHAQUE TEST CORRESPOND A UNE DECISION ECRITE dans l en-tete de `vehicle_rides_events.go`, et
// aucun ne fabrique un seuil : les rayons et tolerances employes ici sont ceux de la production.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// vehEvt fabrique un evenement de la liste, occupant en bande.
func vehEvt(kind int, slot uint32, tUS uint64, seat uint32) filmdec.VehicleEvent {
	return filmdec.VehicleEvent{
		Kind: kind, TimestampUS: tUS, OccupantPresent: true, OccupantInBand: true,
		OccupantSlot: slot, Seat: seat, SeatValid: true,
	}
}

// TestVehicleEpisodeMachineForms : les quatre formes que la machine doit rendre.
func TestVehicleEpisodeMachineForms(t *testing.T) {
	const slot = uint32(500)
	pts := []filmdec.BipedPosition{
		vehPos(slot, 1_000_000, 0, 0), vehPos(slot, 2_000_000, 0, 0),
		vehPos(slot, 9_000_000, 0, 0),
	}
	bySlot := map[uint32][]filmdec.BipedPosition{slot: pts}
	cases := []struct {
		name    string
		evs     []filmdec.VehicleEvent
		want    int
		borders []int
		open    []bool
		starts  []uint64
	}{
		{
			name: "embarquement puis sortie = deux bords",
			evs: []filmdec.VehicleEvent{
				vehEvt(filmdec.EventBipedBoardVehicle, slot, 3_000_000, 0),
				vehEvt(filmdec.EventUnitExitVehicle, slot, 8_000_000, 0),
			},
			want: 1, borders: []int{2}, open: []bool{false}, starts: []uint64{3_000_000},
		},
		{
			name: "sortie seule : le debut est le dernier point replique avant elle",
			evs: []filmdec.VehicleEvent{
				vehEvt(filmdec.EventUnitExitVehicle, slot, 8_000_000, 0),
			},
			want: 1, borders: []int{1}, open: []bool{false}, starts: []uint64{2_000_000},
		},
		{
			name: "embarquement seul = SILENCE TERMINAL",
			evs: []filmdec.VehicleEvent{
				vehEvt(filmdec.EventBipedBoardVehicle, slot, 3_000_000, 0),
			},
			want: 1, borders: []int{1}, open: []bool{true}, starts: []uint64{3_000_000},
		},
		{
			name: "deux embarquements sans sortie : le premier se ferme au second",
			evs: []filmdec.VehicleEvent{
				vehEvt(filmdec.EventBipedBoardVehicle, slot, 3_000_000, 0),
				vehEvt(filmdec.EventBipedBoardVehicle, slot, 6_000_000, 0),
			},
			want: 2, borders: []int{1, 1}, open: []bool{false, true},
			starts: []uint64{3_000_000, 6_000_000},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			boards, exits := vehicleEventsByOccupant(c.evs)
			got := vehicleEventEpisodes(boards, exits, bySlot)
			if len(got) != c.want {
				t.Fatalf("episodes = %d, attendu %d (%+v)", len(got), c.want, got)
			}
			for i := range got {
				if got[i].borders != c.borders[i] || got[i].openEnd != c.open[i] ||
					got[i].startUS != c.starts[i] {
					t.Fatalf("episode %d = {debut %d bords %d ouvert %v}, attendu "+
						"{debut %d bords %d ouvert %v}", i, got[i].startUS, got[i].borders,
						got[i].openEnd, c.starts[i], c.borders[i], c.open[i])
				}
			}
		})
	}
}

// TestVehicleEpisodeSeatFromExit : le siege de la SORTIE prime sur celui de l embarquement.
func TestVehicleEpisodeSeatFromExit(t *testing.T) {
	const slot = uint32(500)
	evs := []filmdec.VehicleEvent{
		vehEvt(filmdec.EventBipedBoardVehicle, slot, 3_000_000, 3),
		vehEvt(filmdec.EventUnitExitVehicle, slot, 8_000_000, 0),
	}
	boards, exits := vehicleEventsByOccupant(evs)
	got := vehicleEventEpisodes(boards, exits, nil)
	if len(got) != 1 || got[0].seat == nil || *got[0].seat != 0 {
		t.Fatalf("siege attendu 0 (celui de la sortie), obtenu %+v", got)
	}
}

// TestMergeVehicleEventsOrder : a instant EGAL la sortie passe avant l embarquement — sinon
// « descendre puis remonter » fabriquerait un episode de duree nulle.
func TestMergeVehicleEventsOrder(t *testing.T) {
	b := []filmdec.VehicleEvent{vehEvt(filmdec.EventBipedBoardVehicle, 1, 5_000_000, 0)}
	e := []filmdec.VehicleEvent{vehEvt(filmdec.EventUnitExitVehicle, 1, 5_000_000, 0)}
	got := mergeVehicleEvents(b, e)
	if len(got) != 2 || got[0].Kind != filmdec.EventUnitExitVehicle {
		t.Fatalf("ordre attendu [sortie, embarquement], obtenu %+v", got)
	}
}

// TestVehicleEpisodeCoversGap : la regle anti-doublon ne ferme la porte du repli que sur un
// recouvrement du MEME occupant.
func TestVehicleEpisodeCoversGap(t *testing.T) {
	eps := []vehicleEpisode{{slot: 500, startUS: 3_000_000, endUS: 8_000_000}}
	cases := []struct {
		name string
		gap  vehicleGap
		want bool
	}{
		{"recouvre", vehicleGap{slot: 500, startUS: 4_000_000, endUS: 9_000_000}, true},
		{"autre occupant", vehicleGap{slot: 501, startUS: 4_000_000, endUS: 9_000_000}, false},
		{"posterieur", vehicleGap{slot: 500, startUS: 9_000_000, endUS: 12_000_000}, false},
		{"anterieur", vehicleGap{slot: 500, startUS: 1_000_000, endUS: 2_000_000}, false},
	}
	for _, c := range cases {
		if got := vehicleEpisodeCovers(eps, c.gap); got != c.want {
			t.Errorf("%s : couvre = %v, attendu %v", c.name, got, c.want)
		}
	}
}

// TestVehicleRideFromEpisodeAnchors : le rattachement passe par l ancre de DEBUT, retombe sur
// celle de FIN, et REFUSE quand aucune n est sous le rayon. Le dernier cas est le temoin : sans
// lui, le test ne prouverait pas que la porte existe.
func TestVehicleRideFromEpisodeAnchors(t *testing.T) {
	const occ, veh = uint32(500), uint32(770)
	key := filmdec.EquipmentLifeKey{Slot: veh, Gen: 1}
	in := vehicleRideInputs{
		vehBySlot: map[uint32][]filmdec.BipedPosition{veh: {
			vehPos(veh, 2_000_000, 10, 10), vehPos(veh, 9_000_000, 10, 10),
		}},
		lives: []vehicleLife{{key: key, loUS: 0, hiUS: 30_000_000}},
		own:   OwnerReport{SlotXUID: map[uint32]uint64{occ: 42}},
		clock: vehClock(),
	}
	cases := []struct {
		name       string
		start, end filmdec.BipedPosition
		open       bool
		want       bool
	}{
		{"ancre de debut sous le rayon", vehPos(occ, 2_000_000, 10.5, 10),
			vehPos(occ, 9_000_000, 40, 40), false, true},
		{"repli sur l ancre de fin", vehPos(occ, 2_000_000, 40, 40),
			vehPos(occ, 9_000_000, 10.5, 10), false, true},
		{"TEMOIN : les deux ancres hors rayon", vehPos(occ, 2_000_000, 40, 40),
			vehPos(occ, 9_000_000, 40, 40), false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bySlot := map[uint32][]filmdec.BipedPosition{occ: {c.start, c.end}}
			ep := vehicleEpisode{slot: occ, startUS: 3_000_000, endUS: 8_000_000,
				borders: 2, openEnd: c.open}
			gotKey, r, _, ok := vehicleRideFromEpisode(ep, bySlot, in)
			if ok != c.want {
				t.Fatalf("rattache = %v, attendu %v", ok, c.want)
			}
			if !ok {
				return
			}
			if gotKey != key {
				t.Fatalf("vie = %+v, attendue %+v", gotKey, key)
			}
			if r.XUID != "42" || r.Src != VehicleRideSrcEvent {
				t.Fatalf("episode publie = %+v (xuid/src inattendus)", r)
			}
		})
	}
}

// TestVehicleRideTerminalSilenceClosesAtLifeEnd : un embarquement sans sortie se ferme a la FIN
// DE VIE du vehicule, pas a l instant de l embarquement.
func TestVehicleRideTerminalSilenceClosesAtLifeEnd(t *testing.T) {
	const occ, veh = uint32(500), uint32(770)
	key := filmdec.EquipmentLifeKey{Slot: veh, Gen: 1}
	in := vehicleRideInputs{
		vehBySlot: map[uint32][]filmdec.BipedPosition{veh: {vehPos(veh, 2_000_000, 10, 10)}},
		lives:     []vehicleLife{{key: key, loUS: 0, hiUS: 21_000_000}},
		clock:     vehClock(),
	}
	bySlot := map[uint32][]filmdec.BipedPosition{occ: {vehPos(occ, 2_000_000, 10.5, 10)}}
	ep := vehicleEpisode{slot: occ, startUS: 3_000_000, endUS: 3_000_000,
		borders: 1, openEnd: true}
	_, r, resolved, ok := vehicleRideFromEpisode(ep, bySlot, in)
	if !ok {
		t.Fatal("le silence terminal doit se rattacher par son ancre de debut")
	}
	if resolved.endUS != 21_000_000 {
		t.Fatalf("fin de l episode = %d, attendue la fin de vie 21000000", resolved.endUS)
	}
	// origine 1 s, pas 100 ms : 3 s -> frame 20, 21 s -> frame 200.
	if r.T0 != 20 || r.T1 != 200 {
		t.Fatalf("bornes publiees = [%d..%d], attendues [20..200]", r.T0, r.T1)
	}
	if r.Src != VehicleRideSrcMixed {
		t.Fatalf("provenance = %q, attendue %q", r.Src, VehicleRideSrcMixed)
	}
}

// TestVehicleEpisodeReappearanceClosesOpenEnd : un embarquement sans sortie dont l occupant
// RE-EMET une position se ferme a cette reapparition, pas a la fin de vie du vehicule. C est le
// cas « mort a bord puis respawn », qui produisait un episode de 90 s sur `0d76e8f1`.
func TestVehicleEpisodeReappearanceClosesOpenEnd(t *testing.T) {
	const occ, veh = uint32(500), uint32(770)
	pts := []filmdec.BipedPosition{
		vehPos(occ, 2_000_000, 10.5, 10), vehPos(occ, 11_000_000, 30, 30),
	}
	evs := []filmdec.VehicleEvent{vehEvt(filmdec.EventBipedBoardVehicle, occ, 3_000_000, 0)}
	boards, exits := vehicleEventsByOccupant(evs)
	eps := vehicleEventEpisodes(boards, exits, map[uint32][]filmdec.BipedPosition{occ: pts})
	if len(eps) != 1 || !eps[0].openEnd || eps[0].reappearUS != 11_000_000 {
		t.Fatalf("episode = %+v, attendu ouvert avec reapparition a 11000000", eps)
	}
	key := filmdec.EquipmentLifeKey{Slot: veh, Gen: 1}
	in := vehicleRideInputs{
		vehBySlot: map[uint32][]filmdec.BipedPosition{veh: {vehPos(veh, 2_000_000, 10, 10)}},
		lives:     []vehicleLife{{key: key, loUS: 0, hiUS: 90_000_000}},
		clock:     vehClock(),
	}
	_, r, resolved, ok := vehicleRideFromEpisode(eps[0], map[uint32][]filmdec.BipedPosition{occ: pts}, in)
	if !ok {
		t.Fatal("l episode doit se rattacher par son ancre de debut")
	}
	if resolved.endUS != 11_000_000 {
		t.Fatalf("fin = %d, attendue la reapparition 11000000 (et non la fin de vie 90000000)",
			resolved.endUS)
	}
	// origine 1 s, pas 100 ms : 3 s -> frame 20, 11 s -> frame 100.
	if r.T0 != 20 || r.T1 != 100 {
		t.Fatalf("bornes publiees = [%d..%d], attendues [20..100]", r.T0, r.T1)
	}
}
