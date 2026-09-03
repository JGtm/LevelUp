package replay

// vehicle_rides_aim_test.go — LA SERIE DE VISEE D UN OCCUPANT : echantillonnage, bornes,
// serialisation et couverture.
//
// CE QUE CES TESTS PROTEGENT, ET C EST UNE LECON DU LOT V11 : la visee publiee ici est la SEULE
// direction qu un artilleur ou un passager obtienne (l entite tourelle ne replique rien —
// refutee avec temoin, cf. la chronique du schema 31). Une regression silencieuse sur cette
// serie ne se verrait pas a l ecran : le client retomberait sur le cap du chassis, c est-a-dire
// sur une direction PLAUSIBLE mais fausse de 15,7 a 21,8 deg en mediane.

import (
	"encoding/json"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// aimClockTest : origine zero, une frame = 100 ms, 1 000 frames — la grille du document.
func aimClockTest() replayClock {
	return replayClock{origin: 0, step: 100_000, frames: 1000}
}

// aimRawTest fabrique une lecture brute a l instant voulu. Les scalaires bruts sont ceux du
// composant `i21` ; on ne les reinterprete pas ici — l accesseur de `filmdec` fait foi.
func aimRawTest(slot uint32, tUS uint64, yaw, pitch uint32) filmdec.BipedAim {
	return filmdec.BipedAim{Slot: slot, TimestampUS: tUS, YawRaw: yaw, PitchRaw: pitch}
}

// TestVehicleRideAimEchantillonnageDeterministe : un point par frame, LE PREMIER OBSERVE GAGNE —
// la meme regle que `vehicleSamplesOf` et `decimateTracks`. Le film replique a 5-46 lectures par
// seconde pour 10 frames par seconde : sans cette regle, la serie porterait le flux brut.
func TestVehicleRideAimEchantillonnageDeterministe(t *testing.T) {
	clock := aimClockTest()
	// Trois lectures DANS la frame 1 (100, 130 et 190 ms), une dans la frame 2.
	aims := []filmdec.BipedAim{
		aimRawTest(7, 100_000, 1000, 1024),
		aimRawTest(7, 130_000, 2000, 1024),
		aimRawTest(7, 190_000, 3000, 1024),
		aimRawTest(7, 200_000, 4000, 1024),
	}
	got := vehicleRideAimOf(aims, 0, 1_000_000, clock)
	if len(got) != 2 {
		t.Fatalf("points publies = %d, attendu 2 (une frame = un point)", len(got))
	}
	if got[0].T != 1 || got[1].T != 2 {
		t.Fatalf("frames publiees = %d, %d — attendu 1, 2", got[0].T, got[1].T)
	}
	// LE PREMIER OBSERVE GAGNE : la frame 1 doit porter la lecture de 100 ms, pas celle de 190.
	veut := headingForJSON(aims[0].AimHeadingDeg())
	if got[0].H != veut {
		t.Fatalf("frame 1 : cap = %v, attendu %v (la PREMIERE lecture de la frame)", got[0].H, veut)
	}
	// DETERMINISME : deux appels sur la meme entree rendent exactement la meme serie.
	again := vehicleRideAimOf(aims, 0, 1_000_000, clock)
	if len(again) != len(got) {
		t.Fatalf("second appel : %d points contre %d", len(again), len(got))
	}
	for i := range got {
		if again[i] != got[i] {
			t.Fatalf("second appel, point %d : %+v contre %+v", i, again[i], got[i])
		}
	}
}

// TestVehicleRideAimBornesDeLEpisode : la serie ne deborde JAMAIS la fenetre de l episode. Une
// lecture hors fenetre appartient a un autre moment de la vie du joueur — a pied, ou dans un
// autre vehicule.
func TestVehicleRideAimBornesDeLEpisode(t *testing.T) {
	clock := aimClockTest()
	aims := []filmdec.BipedAim{
		aimRawTest(7, 100_000, 1000, 1024),   // AVANT l embarquement
		aimRawTest(7, 500_000, 2000, 1024),   // dedans
		aimRawTest(7, 900_000, 3000, 1024),   // dedans
		aimRawTest(7, 1_500_000, 4000, 1024), // APRES la sortie
	}
	got := vehicleRideAimOf(aims, 400_000, 1_000_000, clock)
	if len(got) != 2 {
		t.Fatalf("points publies = %d, attendu 2 (les deux lectures DANS la fenetre)", len(got))
	}
	if got[0].T != 5 || got[1].T != 9 {
		t.Fatalf("frames publiees = %d, %d — attendu 5, 9", got[0].T, got[1].T)
	}
}

// TestVehicleRideAimVideSansLecture : aucune lecture, aucun point — jamais une serie inventee.
// C est le cas que le client doit reconnaitre pour retomber sur le cap du chassis.
func TestVehicleRideAimVideSansLecture(t *testing.T) {
	clock := aimClockTest()
	if got := vehicleRideAimOf(nil, 0, 1_000_000, clock); got != nil {
		t.Fatalf("aucune lecture : serie = %+v, attendu nil", got)
	}
	autreSlot := []filmdec.BipedAim{aimRawTest(9, 500_000, 1000, 1024)}
	if got := vehicleRideAimOf(vehicleAimBySlot(autreSlot)[7], 0, 1_000_000, clock); got != nil {
		t.Fatalf("slot sans lecture : serie = %+v, attendu nil", got)
	}
}

// TestVehicleAimBySlotTrieEtSepare : l index par slot separe les occupants et TRIE chaque serie.
// Deux occupants du MEME vehicule ont deux visees distinctes — c est tout l interet du lot.
func TestVehicleAimBySlotTrieEtSepare(t *testing.T) {
	idx := vehicleAimBySlot([]filmdec.BipedAim{
		aimRawTest(7, 900_000, 1000, 1024),
		aimRawTest(8, 100_000, 2000, 1024),
		aimRawTest(7, 100_000, 3000, 1024),
	})
	if len(idx) != 2 {
		t.Fatalf("slots indexes = %d, attendu 2", len(idx))
	}
	if len(idx[7]) != 2 || idx[7][0].TimestampUS != 100_000 {
		t.Fatalf("slot 7 : %+v — attendu 2 lectures triees par instant", idx[7])
	}
	if len(idx[8]) != 1 {
		t.Fatalf("slot 8 : %d lectures, attendu 1", len(idx[8]))
	}
}

// TestClampVehicleRideAimSuitSonEpisode : la serie est coupee comme son episode. Sans cela, un
// episode ramene a `T0` garderait des points AVANT sa propre borne publiee.
func TestClampVehicleRideAimSuitSonEpisode(t *testing.T) {
	aim := []VehicleAim{{T: 2, H: 10}, {T: 5, H: 20}, {T: 9, H: 30}}
	got := clampVehicleRideAim(aim, 4, 6)
	if len(got) != 1 || got[0].T != 5 {
		t.Fatalf("serie clampee = %+v, attendu le seul point de frame 5", got)
	}
	if got := clampVehicleRideAim(aim, 20, 30); got != nil {
		t.Fatalf("serie entierement hors fenetre = %+v, attendu nil", got)
	}
	// CAS NOMINAL : rien a couper, la tranche d origine est rendue telle quelle.
	if got := clampVehicleRideAim(aim, 0, 100); len(got) != 3 {
		t.Fatalf("serie deja dans la fenetre : %d points, attendu 3", len(got))
	}
}

// TestVehicleRideAimSerialisation : la forme publiee. Le PIEGE omitempty est le point du test —
// un cap qui s arrondit a 0 sort en 360 (le meme angle), une elevation a plat est OMISE (son
// absence VEUT DIRE « a plat », cf. `Point.P`).
func TestVehicleRideAimSerialisation(t *testing.T) {
	r := VehicleRide{T0: 1, T1: 9, Slot: 7, Src: VehicleRideSrcEvent, Aim: []VehicleAim{
		{T: 1, H: 360, P: -12.3},
		{T: 2, H: 91.5},
	}}
	blob, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("serialisation : %v", err)
	}
	veut := `{"t0":1,"t1":9,"slot":7,"src":"event","aim":[{"t":1,"h":360,"p":-12.3},{"t":2,"h":91.5}]}`
	if string(blob) != veut {
		t.Fatalf("JSON = %s\nattendu = %s", blob, veut)
	}
	// SANS SERIE, LE CHAMP DISPARAIT : un artefact sans visee ne porte pas un tableau vide.
	blob, err = json.Marshal(VehicleRide{T0: 1, T1: 9, Slot: 7, Src: VehicleRideSrcGap})
	if err != nil {
		t.Fatalf("serialisation sans visee : %v", err)
	}
	if string(blob) != `{"t0":1,"t1":9,"slot":7,"src":"gap"}` {
		t.Fatalf("JSON sans visee = %s — le champ `aim` doit etre omis", blob)
	}
}

// TestTallyVehicleRidesCompteLaVisee : les quatre compteurs de couverture, et surtout leur
// DENOMINATEUR. `AimSamples` seul ne dit pas si la serie est continue ou trouee ; c est
// `AimRideFrames` qui le dit.
func TestTallyVehicleRidesCompteLaVisee(t *testing.T) {
	cov := VehicleCoverage{UnknownChassis: map[string]int{}}
	tallyVehicleRides([]VehicleRide{
		{T0: 0, T1: 9, Slot: 7, Src: VehicleRideSrcEvent, Aim: []VehicleAim{{T: 0, H: 10}, {T: 3, H: 20}}},
		{T0: 20, T1: 24, Slot: 8, Src: VehicleRideSrcGap},
	}, &cov)
	if cov.RidesWithAim != 1 {
		t.Errorf("episodes avec visee = %d, attendu 1", cov.RidesWithAim)
	}
	if cov.AimSamples != 2 {
		t.Errorf("points de visee = %d, attendu 2", cov.AimSamples)
	}
	// 10 frames pour le premier episode (bornes INCLUSIVES), 5 pour le second.
	if cov.AimRideFrames != 15 {
		t.Errorf("frames d episode = %d, attendu 15", cov.AimRideFrames)
	}
}
