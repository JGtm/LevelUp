package replay

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/equipenergy"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// Les lectures ci-dessous sont celles du film 0d9a9af9 (repulseur de WolfzyLIVE, slot 513) et de
// 1b2d9e08 (translocateur de JGtm, slot 535), au contexte de la cuisson (2026-10-14), origines du
// film 14 436 et 9 062 ms ; le pas est de 100 ms.
const (
	origine0d9a = uint64(14_436_000)
	origine1b2d = uint64(9_062_000)
	pas100ms    = uint64(100_000)
)

func lecture(ms float64, slot, porteur uint32, k types.CarriedReadKind) types.CarriedEquipmentRead {
	return types.CarriedEquipmentRead{TimestampUS: uint64(ms * 1000), Object: types.LifeKey{Slot: slot, Gen: 1},
		Carrier: porteur, Kind: k}
}

func avecEnergie(l types.CarriedEquipmentRead, q uint32) types.CarriedEquipmentRead {
	l.HasEnergy, l.Energy = true, q
	return l
}

func avecActivation(l types.CarriedEquipmentRead) types.CarriedEquipmentRead {
	l.HasActivation = true
	return l
}

func catalogueDuTest() *equipenergy.Catalog {
	return &equipenergy.Catalog{Entries: []equipenergy.Entry{
		{Pickup: "0x7ca85adc", Carried: "0x1e79ebda", PickupEnergy: 0.6, CarriedEnergy: 1, Threshold: 0.2, Delta: -0.2},
		{Pickup: "0x730dc70f", Carried: "0xa1344fc2", PickupEnergy: 1, CarriedEnergy: 1, Threshold: 1, Delta: -1},
		{Pickup: "0x2974c233", Carried: "0xc12e5469", PickupEnergy: 0.66, CarriedEnergy: 1, Threshold: 0.33, Delta: -0.33},
		{Pickup: "0x8e2dc574", Carried: "0x37c87a13", PickupEnergy: 1, CarriedEnergy: 1, Threshold: 1, Delta: -1},
		{Pickup: "0x32d97758", Carried: "0x5c8e2316", PickupEnergy: 0.5, CarriedEnergy: 1, Threshold: 0.1, Delta: -0.5},
	}}
}

var famillesDuTest = map[uint32]string{0x7ca85adc: "repulsor", 0x730dc70f: "translocator_beacon",
	0x2974c233: "wall", 0x8e2dc574: "wall", 0x32d97758: "repair_field"}

func paletteDuTest() *AbilityPalette {
	return &AbilityPalette{Families: map[int]string{6: "repulsor", 11: "translocator_beacon", 2: "wall", 23: "repair_field"}}
}

// TestRepulseurTroisUsages : 0d9a9af9 — ramasse a 0,6 (3 usages), puis 2, 1, 0 aux trois activations ;
// la suppression date la fin.
func TestRepulseurTroisUsages(t *testing.T) {
	reads := []types.CarriedEquipmentRead{
		avecEnergie(lecture(78332.1, 1566, 513, types.CarriedNew), 9830),
		avecActivation(lecture(85839.3, 1566, 513, types.CarriedDelta)),
		avecEnergie(lecture(85855.9, 1566, 513, types.CarriedDelta), 6553),
		avecActivation(lecture(95566.0, 1566, 513, types.CarriedDelta)),
		avecEnergie(lecture(95583.3, 1566, 513, types.CarriedDelta), 3277),
		avecActivation(lecture(106614.4, 1566, 513, types.CarriedDelta)),
		avecEnergie(lecture(106630.8, 1566, 513, types.CarriedDelta), 1),
		lecture(107464.7, 1566, 513, types.CarriedDel),
	}
	changes := []types.EquipmentChange{{TimestampUS: 78332100, Slot: 513, Kind: types.EquipmentTaken, Rank: 6}}
	out, poses, cov := buildCarriedEquipment(entreeDEquipementPorte{reads: reads, changes: changes,
		palette: paletteDuTest(), couts: coutsParFamille(catalogueDuTest(), famillesDuTest),
		tracks: []Track{{Slot: 513}}, origin: origine0d9a, step: pas100ms})
	if len(out) != 1 || len(poses) != 0 || cov.WithCost != 1 {
		t.Fatalf("publies %d, poses %d, couverture %+v", len(out), len(poses), cov)
	}
	var n []int
	for _, e := range out[0].Energy {
		if e.N == nil {
			t.Fatalf("lecture sans usages restants : %+v", e)
		}
		n = append(n, *e.N)
	}
	if want := []int{3, 2, 1, 0}; len(n) != 4 || n[0] != want[0] || n[1] != want[1] || n[2] != want[2] || n[3] != want[3] {
		t.Fatalf("usages restants %v, attendu %v", n, want)
	}
	if len(out[0].Uses) != 3 || out[0].T1 == nil || *out[0].T1 != 930 {
		t.Fatalf("activations %v, fin %v", out[0].Uses, out[0].T1)
	}
}

// failleCreee : la lecture de la pile i28 qui nomme la faille creee par la pose (canal de production,
// 1b2d9e08, 2026-10-15) — sa creation est dans le paquet de la pose, position i0 lue.
func failleCreee(ms, neMs float64, objet, porteur, slot uint32, x, y, z float32) types.CarriedEquipmentRead {
	l := lecture(ms, objet, porteur, types.CarriedDelta)
	l.HasHandles, l.Handles = true, 1
	l.Tracked = []types.LifeKey{{Slot: slot, Gen: 1}}
	l.Spawned = []types.SpawnedObject{{Life: types.LifeKey{Slot: slot, Gen: 1}, TimestampUS: uint64(neMs * 1000),
		TypeIndex: 38, HasPosition: true, X: x, Y: y, Z: z}}
	return l
}

// TestPoseDeFailleDatee : 1b2d9e08, JGtm (535) — l energie du translocateur porte tombe a 0 a 164,11 s
// (frame 1550 : le controle de trajectoire est 164,06 s), 1,635 s apres l activation ; la faille est
// l objet que ce paquet cree (1822, nomme par la pile au paquet suivant), a sa position i0.
func TestPoseDeFailleDatee(t *testing.T) {
	reads := []types.CarriedEquipmentRead{
		lecture(161077.8, 1816, 535, types.CarriedNew),
		avecActivation(lecture(162478.1, 1816, 535, types.CarriedDelta)),
		avecEnergie(lecture(164113.6, 1816, 535, types.CarriedDelta), 0),
		failleCreee(164129.9, 164113.6, 1816, 535, 1822, 17.34, 135.50, 73.98),
		lecture(194210.5, 1816, 535, types.CarriedDel),
	}
	changes := []types.EquipmentChange{{TimestampUS: 161077800, Slot: 535, Kind: types.EquipmentTaken, Rank: 11}}
	out, poses, cov := buildCarriedEquipment(entreeDEquipementPorte{reads: reads, changes: changes,
		palette: paletteDuTest(), couts: coutsParFamille(catalogueDuTest(), famillesDuTest),
		tracks: []Track{{Slot: 535}}, origin: origine1b2d, step: pas100ms})
	if len(out) != 1 || len(poses) != 1 || cov.PositionedByObject != 1 {
		t.Fatalf("publies %d, poses %+v, couverture %+v", len(out), poses, cov)
	}
	p := poses[0]
	if p.T != 1550 || p.A == nil || *p.A != 1534 || p.X == nil || *p.X != 17.34 || *p.Y != 135.50 ||
		p.Src != sourceFailleObjet {
		t.Fatalf("pose %+v", p)
	}
	if e := out[0].Energy[0]; !e.Tag || e.N == nil || *e.N != 1 {
		t.Fatalf("energie a la prise %+v : celle de l objet porte du catalogue (1 usage)", e)
	}
}

// TestPoseSansEchangeSituee : 1b2d9e08, 531 — la pose (frame 1256) n est suivie d AUCUN echange ; la
// faille est quand meme situee, par l objet que la pose cree (1766, (0,73 ; 145,50 ; 76,43)).
func TestPoseSansEchangeSituee(t *testing.T) {
	reads := []types.CarriedEquipmentRead{
		lecture(131000.0, 1763, 531, types.CarriedNew),
		avecActivation(lecture(133097.0, 1763, 531, types.CarriedDelta)),
		avecEnergie(lecture(134734.2, 1763, 531, types.CarriedDelta), 0),
		failleCreee(134750.3, 134734.2, 1763, 531, 1766, 0.73, 145.50, 76.43),
	}
	changes := []types.EquipmentChange{{TimestampUS: 131000000, Slot: 531, Kind: types.EquipmentTaken, Rank: 11}}
	_, poses, cov := buildCarriedEquipment(entreeDEquipementPorte{reads: reads, changes: changes,
		palette: paletteDuTest(), couts: coutsParFamille(catalogueDuTest(), famillesDuTest),
		tracks: []Track{{Slot: 531}}, origin: origine1b2d, step: pas100ms})
	if len(poses) != 1 || poses[0].T != 1256 || poses[0].X == nil || *poses[0].X != 0.73 ||
		poses[0].Src != sourceFailleObjet || cov.Unpositioned != 0 {
		t.Fatalf("poses %+v, couverture %+v", poses, cov)
	}
}

// TestPoseRepliDuPorteur : sans objet cree lisible, la faille prend la position du porteur lue dans sa
// piste a l image de la pose, nommee et comptee ; sans point a cette image, elle reste sans position.
func TestPoseRepliDuPorteur(t *testing.T) {
	reads := []types.CarriedEquipmentRead{
		lecture(131000.0, 1763, 531, types.CarriedNew),
		avecEnergie(lecture(134734.2, 1763, 531, types.CarriedDelta), 0),
	}
	changes := []types.EquipmentChange{{TimestampUS: 131000000, Slot: 531, Kind: types.EquipmentTaken, Rank: 11}}
	entree := entreeDEquipementPorte{reads: reads, changes: changes, palette: paletteDuTest(),
		couts: coutsParFamille(catalogueDuTest(), famillesDuTest), origin: origine1b2d, step: pas100ms,
		tracks: []Track{{Slot: 531, Points: []Point{{T: 1255, X: 9, Y: 9}, {T: 1256, X: 0.7, Y: 145.4, Z: 76.4}}}}}
	_, poses, cov := buildCarriedEquipment(entree)
	if len(poses) != 1 || poses[0].Src != sourceFaillePorteur || *poses[0].X != 0.7 || cov.PositionedByCarrier != 1 {
		t.Fatalf("poses %+v, couverture %+v", poses, cov)
	}
	entree.tracks = []Track{{Slot: 531, Points: []Point{{T: 1255, X: 9, Y: 9}}}}
	_, poses, cov = buildCarriedEquipment(entree)
	if len(poses) != 1 || poses[0].X != nil || poses[0].Src != "" || cov.Unpositioned != 1 {
		t.Fatalf("poses %+v, couverture %+v", poses, cov)
	}
}

// TestMurDeuxFiches : la famille du mur porte deux fiches de couts differents ; l energie ecrite a la
// prise (0,66, celle du ramassable de la premiere) choisit, sinon aucun compte.
func TestMurDeuxFiches(t *testing.T) {
	couts := coutsParFamille(catalogueDuTest(), famillesDuTest)
	if c, ok := choisirLeCout(couts["wall"], avecEnergie(lecture(0, 1, 513, types.CarriedNew), 10812)); !ok ||
		c.variation != -0.33 {
		t.Fatalf("energie 0,66 : %+v %v", c, ok)
	}
	if _, ok := choisirLeCout(couts["wall"], lecture(0, 1, 513, types.CarriedNew)); ok {
		t.Fatal("sans energie ecrite, deux fiches en desaccord ne choisissent rien")
	}
}

// TestObjetSansAnnonceNonPublie : un objet cree attache sans annonce i48 du porteur dans son paquet
// n est pas publie, et se compte.
func TestObjetSansAnnonceNonPublie(t *testing.T) {
	reads := []types.CarriedEquipmentRead{lecture(78332.1, 1566, 513, types.CarriedNew)}
	out, _, cov := buildCarriedEquipment(entreeDEquipementPorte{reads: reads, palette: paletteDuTest(),
		tracks: []Track{{Slot: 513}}, origin: origine0d9a, step: pas100ms})
	if len(out) != 0 || cov.WithoutRank != 1 {
		t.Fatalf("publies %d, couverture %+v", len(out), cov)
	}
}

// TestRepulseurEpuiseNePoseRien : un repulseur dont l energie tombe a 0 apres sa creation ne pose
// aucune faille — la regle de pose est celle du seul translocateur.
func TestRepulseurEpuiseNePoseRien(t *testing.T) {
	reads := []types.CarriedEquipmentRead{
		avecEnergie(lecture(78332.1, 1566, 513, types.CarriedNew), 3277),
		avecActivation(lecture(85839.3, 1566, 513, types.CarriedDelta)),
		avecEnergie(lecture(85855.9, 1566, 513, types.CarriedDelta), 0),
	}
	changes := []types.EquipmentChange{{TimestampUS: 78332100, Slot: 513, Kind: types.EquipmentTaken, Rank: 6}}
	out, poses, cov := buildCarriedEquipment(entreeDEquipementPorte{reads: reads, changes: changes,
		palette: paletteDuTest(), couts: coutsParFamille(catalogueDuTest(), famillesDuTest),
		tracks: []Track{{Slot: 513}}, origin: origine0d9a, step: pas100ms})
	if len(out) != 1 || len(poses) != 0 || cov.RiftPoses != 0 {
		t.Fatalf("publies %d, poses %+v, couverture %+v", len(out), poses, cov)
	}
}

// origine5676 : l origine de 5676a9ba (Insolence), 24 381 ms.
const origine5676 = uint64(24_381_000)

func avecSuivis(l types.CarriedEquipmentRead, slots ...uint32) types.CarriedEquipmentRead {
	l.HasHandles, l.Handles = true, uint32(len(slots)) //nolint:gosec // test
	for _, s := range slots {
		l.Tracked = append(l.Tracked, types.LifeKey{Slot: s, Gen: 1})
	}
	return l
}

func finDeSuivi(ms float64, slot uint32) types.CarriedEquipmentRead {
	l := lecture(ms, 2039, 539, types.CarriedTrackedEnd)
	l.Tracked = []types.LifeKey{{Slot: slot, Gen: 1}}
	return l
}

// TestChampDeReparationFinLue : 5676a9ba, champ de reparation de 539 (lectures du canal de production,
// 2026-10-15). La pile i28 nomme le champ engendre a chaque usage (1123, puis 1138) ; sa suppression
// date sa fin, meme quand elle suit la fin de l objet porte (le second champ survit a 134,998 s).
func TestChampDeReparationFinLue(t *testing.T) {
	reads := []types.CarriedEquipmentRead{
		lecture(90183.0, 2039, 539, types.CarriedNew),
		avecActivation(avecEnergie(lecture(118540.0, 2039, 539, types.CarriedDelta), 8192)),
		avecSuivis(lecture(118775.4, 2039, 539, types.CarriedDelta), 1123),
		avecSuivis(lecture(130254.9, 2039, 539, types.CarriedDelta)),
		finDeSuivi(130456.1, 1123),
		avecActivation(avecEnergie(lecture(133759.2, 2039, 539, types.CarriedDelta), 0)),
		avecSuivis(lecture(134024.8, 2039, 539, types.CarriedDelta), 1138),
		lecture(134997.8, 2039, 539, types.CarriedDel),
		finDeSuivi(135728.8, 1138),
	}
	changes := []types.EquipmentChange{{TimestampUS: 90183000, Slot: 539, Kind: types.EquipmentTaken, Rank: 23}}
	out, _, cov := buildCarriedEquipment(entreeDEquipementPorte{reads: reads, changes: changes,
		palette: paletteDuTest(), couts: coutsParFamille(catalogueDuTest(), famillesDuTest),
		tracks: []Track{{Slot: 539}}, origin: origine5676, step: pas100ms})
	if len(out) != 1 || cov.TrackedObjects != 2 || cov.TrackedEnds != 2 {
		t.Fatalf("publies %d, couverture %+v", len(out), cov)
	}
	tr := out[0].Tracked
	if len(tr) != 2 || tr[0].T0 != 943 || tr[0].T1 == nil || *tr[0].T1 != 1060 ||
		tr[1].T0 != 1096 || tr[1].T1 == nil || *tr[1].T1 != 1113 {
		t.Fatalf("objets engendres %+v", tr)
	}
	if out[0].T1 == nil || *out[0].T1 != 1106 || *tr[1].T1 <= *out[0].T1 {
		t.Fatalf("fin de l objet porte %v : le second champ finit apres elle", out[0].T1)
	}
}

// TestFailleFinitASuppressionLue : la faille de TestPoseDeFailleDatee (objet 1822 cree par la pose de
// 535) finit a la suppression de cet objet, que la pile i28 de l objet porte ecrit (`tracked-end`,
// instant construit pour le test, apres la fin de l objet porte) ; sans cette lecture, la pose sort
// sans fin et se compte au repli.
func TestFailleFinitASuppressionLue(t *testing.T) {
	fin := lecture(196000.0, 1816, 535, types.CarriedTrackedEnd)
	fin.Tracked = []types.LifeKey{{Slot: 1822, Gen: 1}}
	reads := []types.CarriedEquipmentRead{
		lecture(161077.8, 1816, 535, types.CarriedNew),
		avecEnergie(lecture(164113.6, 1816, 535, types.CarriedDelta), 0),
		failleCreee(164129.9, 164113.6, 1816, 535, 1822, 17.34, 135.50, 73.98),
		lecture(194210.5, 1816, 535, types.CarriedDel),
		fin,
	}
	entree := entreeDEquipementPorte{reads: reads, palette: paletteDuTest(),
		changes: []types.EquipmentChange{{TimestampUS: 161077800, Slot: 535, Kind: types.EquipmentTaken, Rank: 11}},
		couts:   coutsParFamille(catalogueDuTest(), famillesDuTest),
		tracks:  []Track{{Slot: 535}}, origin: origine1b2d, step: pas100ms}
	_, poses, cov := buildCarriedEquipment(entree)
	if len(poses) != 1 || poses[0].T1 == nil || *poses[0].T1 != 1869 || cov.RiftEndsRead != 1 ||
		cov.RiftEndsUnread != 0 {
		t.Fatalf("poses %+v, couverture %+v", poses, cov)
	}
	entree.reads = reads[:4]
	_, poses, cov = buildCarriedEquipment(entree)
	if len(poses) != 1 || poses[0].T1 != nil || cov.RiftEndsRead != 0 || cov.RiftEndsUnread != 1 {
		t.Fatalf("sans suppression lue : poses %+v, couverture %+v", poses, cov)
	}
}
