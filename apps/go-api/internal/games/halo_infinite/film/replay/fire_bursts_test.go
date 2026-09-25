package replay

// fire_bursts_test.go — LA PUBLICATION DU TIR CONTINU, testee sans film (logique pure,
// `fire_bursts.go`). Les chiffres de terrain vivent dans les instruments du lot M4b.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// fbRafale fabrique une rafale lue de la gachette principale, entre deux frames (pas de 100 ms).
func fbRafale(index, t0, t1, arme int) types.ContinuousFireBurst {
	return types.ContinuousFireBurst{FilmIndex: index, Weapon: arme, StartUS: uint64(t0) * 100_000,
		EndUS: uint64(t1) * 100_000, StartBound: types.ContinuousFireBoundPressed,
		EndBound: types.ContinuousFireBoundReleased}
}

// fbDoc monte un Ghost (slot 700, chassis du parc) pilote par le slot 10 (joueur 3) de 10 a 40, et
// la piste a pied du slot 11 (joueur 4) de 0 a 90, dotee d un pistolet a plasma et d un Rayon de
// Sentinelle.
func fbDoc() *ReplayDocument {
	seat := 0
	return &ReplayDocument{
		Tracks: []Track{{Slot: 10, Points: []Point{{T: 0}, {T: 90}}}, {Slot: 11, Points: []Point{{T: 0}, {T: 90}}}},
		Vehicles: []VehicleTrack{{Slot: 700, Gen: 1, Chassis: "5b80c406", Family: familleGhost, T0: 0, T1: 90,
			T1Max: 90, Rides: []VehicleRide{{T0: 10, T1: 40, Slot: 10, Seat: &seat, Src: VehicleRideSrcFilm}}}},
		Loadouts: []Loadout{{T: 0, Slot: 11, W: []string{"0xC3542946", "0xA0955E9E"}}},
	}
}

func fbPublier(doc *ReplayDocument, rs ...types.ContinuousFireBurst) ([]FireBurst, *ContinuousFireCoverage) {
	return buildFireBursts(doc, rs, types.ContinuousFireStats{Scanned: true}, map[uint32]int{10: 3, 11: 4},
		vsClock())
}

// TestRafaleDuPiloteDuGhostPoseeSurLeGhost — le cas du retour utilisateur (81c02726) : la gachette
// tenue du pilote devient une rafale du canon du Ghost, posee sur le Ghost, a la cadence du tag.
func TestRafaleDuPiloteDuGhostPoseeSurLeGhost(t *testing.T) {
	out, cov := fbPublier(fbDoc(), fbRafale(3, 12, 30, 0))
	if len(out) != 1 {
		t.Fatalf("rafales publiees = %d, attendu 1 (couverture %+v)", len(out), cov)
	}
	f := out[0]
	if f.Vehicle == nil || *f.Vehicle != 700 || f.Slot != 10 || f.Weapon != VehicleWeaponKey(0x00015435) ||
		f.Rate != 7.5 || f.T0 != 12 || f.T1 != 30 {
		t.Errorf("rafale = %+v, attendu v 700, slot 10, arme du Ghost, 7,5/s, 12..30", f)
	}
	if cov.OnVehicle != 1 || cov.Published != 1 || !cov.balanced() {
		t.Errorf("couverture = %+v", cov)
	}
}

// TestRafaleBorneeALEpisode — la gachette tenue au-dela de la descente ne tire plus le canon.
func TestRafaleBorneeALEpisode(t *testing.T) {
	out, cov := fbPublier(fbDoc(), fbRafale(3, 35, 60, 0))
	if len(out) != 1 || out[0].T1 != 40 || cov.ClippedToMount != 1 {
		t.Errorf("rafale = %+v, borne %d, attendu fin 40 et une borne", out, cov.ClippedToMount)
	}
}

// TestRafaleAPiedSuitLArmeEnMain — a pied, l emplacement du bloc d action designe l arme : le Rayon
// de Sentinelle tire en continu, la CHARGE du pistolet a plasma n est pas un tir.
func TestRafaleAPiedSuitLArmeEnMain(t *testing.T) {
	// -1 : la sentinelle du bloc d action (aucun emplacement ecrit).
	out, cov := fbPublier(fbDoc(), fbRafale(4, 20, 25, 1), fbRafale(4, 50, 60, 0), fbRafale(4, 70, 71, -1))
	if len(out) != 1 || out[0].Weapon != "0xA0955E9E" || out[0].Rate != 60 || out[0].Vehicle != nil ||
		out[0].Slot != 11 {
		t.Fatalf("rafales = %+v, attendu le seul Rayon de Sentinelle a pied", out)
	}
	if cov.OnFoot != 1 || cov.NotContinuous != 1 || cov.WeaponUnknown != 1 || !cov.balanced() {
		t.Errorf("couverture = %+v, attendu 1 a pied, 1 arme non continue, 1 arme inconnue", cov)
	}
}

// TestRafaleDUnePriseDateeApresLaDotation — une prise datee entre la dotation et la rafale change
// l arme de son emplacement.
func TestRafaleDUnePriseDateeApresLaDotation(t *testing.T) {
	doc := fbDoc()
	k := 0
	doc.WeaponChanges = []WeaponChange{{T: 40, Slot: 11, Kind: WeaponSwapped, W: "a0955e9e", From: "c3542946", K: &k}}
	out, _ := fbPublier(doc, fbRafale(4, 50, 55, 0))
	if len(out) != 1 || out[0].Weapon != "0xA0955E9E" {
		t.Errorf("rafales = %+v, attendu le Rayon pris a la frame 40", out)
	}
}

// TestRafaleDuKlaxonNonDessinee — la gachette du conducteur d un Warthog est son klaxon : aucune
// arme a tir continu, la rafale est comptee et jamais dessinee.
func TestRafaleDuKlaxonNonDessinee(t *testing.T) {
	doc := fbDoc()
	doc.Vehicles[0].Chassis, doc.Vehicles[0].Family = "fe32c0f4", familleWarthog
	out, cov := fbPublier(doc, fbRafale(3, 12, 20, 0))
	if len(out) != 0 || cov.VehicleNoWeapon != 1 || !cov.balanced() {
		t.Errorf("rafales = %+v, couverture %+v : attendu 0 et une monture sans arme", out, cov)
	}
}

// TestRafaleDUnAutreBitComptee — seule la gachette principale de la main 0 est publiee.
func TestRafaleDUnAutreBitComptee(t *testing.T) {
	r := fbRafale(3, 12, 20, 0)
	r.Barrel = true
	out, cov := fbPublier(fbDoc(), r)
	if len(out) != 0 || cov.OtherInput != 1 || !cov.balanced() {
		t.Errorf("rafales = %+v, couverture %+v", out, cov)
	}
}

// TestRafaleDuRemplacantParSaPlace — l index d une rafale est la PLACE : le remplacant assis a la
// place 3 la tire, pas le partant.
func TestRafaleDuRemplacantParSaPlace(t *testing.T) {
	doc := fbDoc()
	doc.Roster = []RosterEntry{
		{XUID: "1", FilmIndex: 3, Seat: 3, Presence: []PresenceInterval{{From: 0, To: 5}}},
		{XUID: "2", FilmIndex: 4, Seat: 3, Presence: []PresenceInterval{{From: 6, To: 90}}},
	}
	out, cov := fbPublier(doc, fbRafale(3, 20, 25, 1))
	if len(out) != 1 || out[0].Slot != 11 || cov.ByPlace != 1 {
		t.Errorf("rafales = %+v, par la place %d : attendu la piste du remplacant (slot 11)", out, cov.ByPlace)
	}
}

// TestTrousProjetesSurLesFramesEntieres — une frame n est muette que si le trou la couvre entiere,
// et les trous qui se touchent fusionnent.
func TestTrousProjetesSurLesFramesEntieres(t *testing.T) {
	r := fbRafale(3, 12, 30, 0)
	r.Holes = []types.ContinuousFireHole{{StartUS: 1_250_000, EndUS: 1_290_000},
		{StartUS: 1_450_000, EndUS: 1_700_000}, {StartUS: 1_700_000, EndUS: 1_900_000}}
	out, _ := fbPublier(fbDoc(), r)
	if len(out) != 1 || len(out[0].Holes) != 1 || out[0].Holes[0] != (FireBurstHole{T0: 15, T1: 19}) {
		t.Errorf("trous = %+v, attendu un seul [15, 19)", out)
	}
}

// TestCoupsALaCadenceEtALaMontee — les coups se posent a la cadence du tag, et une arme qui monte
// en cadence (LAAG) commence lentement.
func TestCoupsALaCadenceEtALaMontee(t *testing.T) {
	ghost := FireBurst{T0: 0, T1: 10, Rate: 7.5}
	if n := coupsDeLaRafale(ghost, 100); n != 8 {
		t.Errorf("Ghost sur une seconde : %d coups, attendu 8 (a 0 puis toutes les 133 ms)", n)
	}
	laag := FireBurst{T0: 0, T1: 10, Rate: 18, Rate0: 5, Ramp: 1.6}
	if n := coupsDeLaRafale(laag, 100); n < 8 || n > 14 {
		t.Errorf("LAAG sur une seconde de montee : %d coups, attendu entre 8 et 14", n)
	}
	ghost.Holes = []FireBurstHole{{T0: 0, T1: 9}}
	if n := coupsDeLaRafale(ghost, 100); n != 1 {
		t.Errorf("Ghost muet jusqu a la frame 9 : %d coups, attendu 1 (a 9,33)", n)
	}
}

// TestSansMarcheDesTramesAucuneCouverture — un artefact reconstruit sans la marche n affirme rien.
func TestSansMarcheDesTramesAucuneCouverture(t *testing.T) {
	out, cov := buildFireBursts(fbDoc(), nil, types.ContinuousFireStats{}, nil, vsClock())
	if out != nil || cov != nil {
		t.Errorf("rafales %v couverture %v : attendu nil, nil", out, cov)
	}
}
