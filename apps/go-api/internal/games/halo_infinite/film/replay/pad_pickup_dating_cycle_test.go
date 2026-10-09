package replay

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// pad_pickup_dating_cycle_test.go — UNE SEULE PRISE DE SOCLE PAR RÉAPPARITION DE L'ARME (repli
// `repli_prise_de_socle_premiere_du_cycle`, pad_pickup_dating_cycle.go).

// daterLesOccupations enchaîne les deux phases de la datation, sans relevé entre elles.
func daterLesOccupations(pads []WeaponPad, picks []PadPickup, pickups []Pickup, loc localiserRamasseur) PadDatingStats {
	return lireLesOccupations(pads, picks, pickups).trancherParLeCycle(loc)
}

// localiserFixe rend la position déclarée pour chaque instant de ramassage ; un instant absent
// n'est pas localisé.
func localiserFixe(pos map[int][3]float32) localiserRamasseur {
	return func(p Pickup) ([3]float32, bool) {
		x, ok := pos[p.T]
		return x, ok
	}
}

const famCycle = 0x11223344

// socleDuCycle : un socle à l'origine dont l'arme apparaît à 100, est REMPLACÉE à 250 par une
// seconde apparition (fenêtre de la première ouverte au-delà), puis réapparaît à 900.
func socleDuCycle() []WeaponPad {
	return []WeaponPad{{
		Weapon: padWeaponForm(famCycle),
		Presence: []PadPresence{
			{T0: 100, TLow: 150, THigh: 350},
			{T0: 250, TLow: 250, THigh: 350},
			{T0: 900, TLow: 1000, THigh: 1200},
		},
	}}
}

func occupationsDuCycle() []PadPickup {
	return []PadPickup{
		{Pad: 0, TLow: 150, THigh: 350},
		{Pad: 0, TLow: 250, THigh: 350},
		{Pad: 0, TLow: 1000, THigh: 1200},
	}
}

// LE CAS DU NEEDLER : une prise ailleurs sur la carte (160), la prise au socle à la réapparition
// (250), puis l'arme lâchée par le preneur reprise au sol près du socle (260). La lecture ne
// tranche aucune des deux fenêtres ; la règle désigne 250 pour la seconde apparition, et la
// première, remplacée avant toute prise au socle, reste sans date.
//
// MUTATION : retirer la borne de l'apparition suivante ([finDuCycle]) — ROUGE (la première
// occupation revendique aussi 250, les deux s'abstiennent). Retirer la condition AU SOCLE — ROUGE
// (la première occupation prend 160). Prendre le dernier candidat au lieu du premier — ROUGE.
func TestUneSeulePriseDeSocleParReapparition(t *testing.T) {
	pickups := []Pickup{
		{T: 160, W: pickupWeaponForm(famCycle), Kind: PickupWeapon, XUID: "ailleurs"},
		{T: 250, W: pickupWeaponForm(famCycle), Kind: PickupWeapon, XUID: "preneur"},
		{T: 260, W: pickupWeaponForm(famCycle), Kind: PickupWeapon, XUID: "au_sol"},
		{T: 1100, W: pickupWeaponForm(famCycle), Kind: PickupWeapon, XUID: "suivant"},
	}
	loc := localiserFixe(map[int][3]float32{
		160: {20, 0, 0}, 250: {0.4, 0, 0}, 260: {1.2, 0, 0}, 1100: {30, 0, 0},
	})

	sans := occupationsDuCycle()
	stSans := daterLesOccupations(socleDuCycle(), sans, pickups, nil)
	if stSans.Dated != 1 || stSans.FirstOfCycle != 0 || stSans.Ambiguous != 2 {
		t.Fatalf("sans localisation : stats = %+v, attendu dated=1 (lecture de 1100) firstOfCycle=0 ambiguous=2", stSans)
	}

	picks := occupationsDuCycle()
	st := daterLesOccupations(socleDuCycle(), picks, pickups, loc)
	if picks[0].T != nil {
		t.Errorf("apparition remplacee : t = %d, attendu aucune date", *picks[0].T)
	}
	if picks[1].T == nil || *picks[1].T != 250 || picks[1].XUID == nil || *picks[1].XUID != "preneur" {
		t.Errorf("reapparition : t = %v, xuid = %v, attendu 250 et \"preneur\"", picks[1].T, picks[1].XUID)
	}
	// LA LECTURE PRIME : l'unique ramassage de la troisième fenêtre la date, même loin du socle.
	if picks[2].T == nil || *picks[2].T != 1100 {
		t.Errorf("troisieme occupation : t = %v, attendu 1100 (lecture inchangee)", picks[2].T)
	}
	if st.Occupations != 3 || st.Dated != 2 || st.Named != 2 || st.FirstOfCycle != 1 ||
		st.Ambiguous != 1 || st.Uncovered != 0 {
		t.Errorf("stats = %+v, attendu occupations=3 dated=2 named=2 firstOfCycle=1 ambiguous=1", st)
	}
}

// UN RAMASSAGE QU'UNE LECTURE A DATÉ N'EST JAMAIS REPRIS, et un candidat plus tôt non localisé
// interdit de conclure.
//
// MUTATION : retirer le filtre `pris` de [premiereAuSocle] — ROUGE (l'occupation 1 reprend 20).
// Sauter un candidat non localisé au lieu de s'abstenir — ROUGE (l'occupation 2 prend 70).
func TestPremierePriseDuCycleRespecteLaLectureEtSAbstient(t *testing.T) {
	const autre = 0x55667788
	pads := []WeaponPad{
		{Weapon: padWeaponForm(famCycle), Presence: []PadPresence{{T0: 0, TLow: 10, THigh: 30}}},
		{Weapon: padWeaponForm(famCycle), Presence: []PadPresence{{T0: 0, TLow: 15, THigh: 40}}},
		{Weapon: padWeaponForm(autre), X: 50, Presence: []PadPresence{{T0: 0, TLow: 50, THigh: 80}}},
	}
	pickups := []Pickup{
		{T: 20, W: pickupWeaponForm(famCycle), Kind: PickupWeapon, XUID: "a"},
		{T: 35, W: pickupWeaponForm(famCycle), Kind: PickupWeapon, XUID: "b"},
		{T: 60, W: pickupWeaponForm(autre), Kind: PickupWeapon, XUID: "c"},
		{T: 70, W: pickupWeaponForm(autre), Kind: PickupWeapon, XUID: "d"},
	}
	picks := []PadPickup{
		{Pad: 0, TLow: 10, THigh: 30}, // lecture : 20, seul dans sa fenetre
		{Pad: 1, TLow: 15, THigh: 40}, // 20 et 35 : 20 est pris par la lecture, 35 loin
		{Pad: 2, TLow: 50, THigh: 80}, // 60 non localise, 70 au socle
	}
	loc := localiserFixe(map[int][3]float32{20: {0, 0, 0}, 35: {25, 0, 0}, 70: {50, 0, 0}})
	st := daterLesOccupations(pads, picks, pickups, loc)
	if picks[0].T == nil || *picks[0].T != 20 {
		t.Fatalf("occupation 0 : t = %v, attendu 20 (lecture)", picks[0].T)
	}
	if picks[1].T != nil {
		t.Errorf("occupation 1 : t = %d, attendu aucune date (20 est a la lecture, 35 loin du socle)", *picks[1].T)
	}
	if picks[2].T != nil {
		t.Errorf("occupation 2 : t = %d, attendu aucune date (60, plus tot, n est pas localise)", *picks[2].T)
	}
	if st.Dated != 1 || st.FirstOfCycle != 0 || st.Ambiguous != 2 {
		t.Errorf("stats = %+v, attendu dated=1 firstOfCycle=0 ambiguous=2", st)
	}
}

// UN RAMASSAGE QUE LE REPLI DÉSIGNE POUR DEUX OCCUPATIONS N'EN DATE AUCUNE : deux socles voisins
// de la même arme, le ramasseur à moins de 1,5 m des deux.
//
// MUTATION : retirer le compte des revendications de [premieresPrisesDuCycle] — ROUGE.
func TestPremierePriseDuCycleDisputeeNeDateRien(t *testing.T) {
	pads := []WeaponPad{
		{Weapon: padWeaponForm(famCycle), Presence: []PadPresence{{T0: 0, TLow: 10, THigh: 30}}},
		{Weapon: padWeaponForm(famCycle), X: 1, Presence: []PadPresence{{T0: 0, TLow: 10, THigh: 30}}},
	}
	pickups := []Pickup{
		{T: 20, W: pickupWeaponForm(famCycle), Kind: PickupWeapon, XUID: "a"},
		{T: 25, W: pickupWeaponForm(famCycle), Kind: PickupWeapon, XUID: "b"},
	}
	picks := []PadPickup{{Pad: 0, TLow: 10, THigh: 30}, {Pad: 1, TLow: 10, THigh: 30}}
	loc := localiserFixe(map[int][3]float32{20: {0.5, 0, 0}, 25: {0.5, 0, 0}})
	st := daterLesOccupations(pads, picks, pickups, loc)
	if picks[0].T != nil || picks[1].T != nil {
		t.Errorf("t = %v / %v, attendu aucune date : 20 est designe pour les deux socles", picks[0].T, picks[1].T)
	}
	if st.Dated != 0 || st.FirstOfCycle != 0 || st.Ambiguous != 2 {
		t.Errorf("stats = %+v, attendu dated=0 firstOfCycle=0 ambiguous=2", st)
	}
}

// UNE APPARITION QUE LA FENÊTRE NE RETROUVE PAS SANS AMBIGUÏTÉ NE BORNE AUCUN CYCLE : le repli
// s'abstient.
func TestFinDuCycleRefuseUneFenetrePartagee(t *testing.T) {
	pad := WeaponPad{Presence: []PadPresence{{T0: 0, TLow: 10, THigh: 30}, {T0: 5, TLow: 10, THigh: 30}}}
	if _, ok := finDuCycle(pad, PadPickup{TLow: 10, THigh: 30}); ok {
		t.Error("deux apparitions sous la meme fenetre : attendu faux")
	}
	if _, ok := finDuCycle(pad, PadPickup{TLow: 11, THigh: 30}); ok {
		t.Error("fenetre absente de la presence : attendu faux")
	}
	seule := WeaponPad{Presence: []PadPresence{{T0: 0, TLow: 10, THigh: 30}, {T0: 40, TLow: 40, THigh: 60}}}
	if fin, ok := finDuCycle(seule, PadPickup{TLow: 10, THigh: 30}); !ok || fin != 40 {
		t.Errorf("fin = %d, %v ; attendu 40, vrai", fin, ok)
	}
}

// LE PREMIER RAMASSAGE AU SOCLE DU CYCLE EST DÉJÀ DATÉ PAR UNE LECTURE (celle d'un autre socle de
// la même arme) : il est la prise du cycle, le suivant au socle une reprise au sol. L'occupation
// s'abstient au lieu de prendre ce suivant.
//
// MUTATION : sauter un ramassage `pris` au lieu de s'abstenir dans [premiereAuSocle] — ROUGE
// (l'occupation 1 prend 30).
func TestPremierePriseDuCycleDejaLueFaitSAbstenir(t *testing.T) {
	pads := []WeaponPad{
		{Weapon: padWeaponForm(famCycle), X: 40, Presence: []PadPresence{{T0: 0, TLow: 10, THigh: 25}}},
		{Weapon: padWeaponForm(famCycle), Presence: []PadPresence{{T0: 0, TLow: 15, THigh: 40}}},
	}
	pickups := []Pickup{
		{T: 20, W: pickupWeaponForm(famCycle), Kind: PickupWeapon, XUID: "p"},
		{T: 30, W: pickupWeaponForm(famCycle), Kind: PickupWeapon, XUID: "q"},
	}
	picks := []PadPickup{
		{Pad: 0, TLow: 10, THigh: 25}, // lecture : 20, seul dans sa fenetre
		{Pad: 1, TLow: 15, THigh: 40}, // 20 (au socle, deja lu) puis 30 (au socle)
	}
	loc := localiserFixe(map[int][3]float32{20: {1, 0, 0}, 30: {0.5, 0, 0}})
	st := daterLesOccupations(pads, picks, pickups, loc)
	if picks[0].T == nil || *picks[0].T != 20 {
		t.Fatalf("occupation 0 : t = %v, attendu 20 (lecture)", picks[0].T)
	}
	if picks[1].T != nil {
		t.Errorf("occupation 1 : t = %d, attendu aucune date (20, premier au socle, est deja lu)", *picks[1].T)
	}
	if st.Dated != 1 || st.FirstOfCycle != 0 || st.Ambiguous != 1 {
		t.Errorf("stats = %+v, attendu dated=1 firstOfCycle=0 ambiguous=1", st)
	}

	// Contre-epreuve : le meme ramassage lu, mais LOIN du socle 1 — le suivant au socle est pris.
	picks = []PadPickup{{Pad: 0, TLow: 10, THigh: 25}, {Pad: 1, TLow: 15, THigh: 40}}
	loin := localiserFixe(map[int][3]float32{20: {40, 0, 0}, 30: {0.5, 0, 0}})
	st = daterLesOccupations(pads, picks, pickups, loin)
	if picks[1].T == nil || *picks[1].T != 30 || st.FirstOfCycle != 1 {
		t.Errorf("occupation 1 : t = %v, firstOfCycle = %d ; attendu 30 et 1", picks[1].T, st.FirstOfCycle)
	}
}

// LES DEUX REFUS DU LOCALISATEUR : deux ramassages bruts sous la même clé (vie, frame, objet), et un
// ramassage antérieur à l'origine du document. Le cas nominal passe l'horodatage EXACT.
//
// MUTATION : retirer le refus des doublons de [localiserParLeCanalNatif] — ROUGE ; retirer la garde
// de l origine — ROUGE (le ramassage de 50 µs prend la frame -1 et s y localise).
func TestLocaliserParLeCanalNatifRefuse(t *testing.T) {
	const fam = 0x11223344
	clock := replayClock{origin: 1_000, step: 100_000}
	natifs := []types.BipedPickup{
		{TimestampUS: 1_000 + 250_000, Slot: 512, CatalogID: fam}, // frame 2, unique
		{TimestampUS: 1_000 + 510_000, Slot: 513, CatalogID: fam}, // frame 5, doublon
		{TimestampUS: 1_000 + 590_000, Slot: 513, CatalogID: fam}, // frame 5, doublon
		{TimestampUS: 50, Slot: 514, CatalogID: fam},              // avant l origine
	}
	vus := map[uint64]bool{}
	loc := localiserParLeCanalNatif(natifs, clock, func(slot uint32, ts uint64) (float32, float32, float32, bool) {
		vus[ts] = true
		return float32(slot), 0, 0, true
	})
	if pos, ok := loc(Pickup{T: 2, Slot: 512, W: pickupWeaponForm(fam)}); !ok || pos[0] != 512 || !vus[251_000] {
		t.Errorf("cas nominal : pos = %v, ok = %v, horodatages vus = %v ; attendu 512, vrai, 251000", pos, ok, vus)
	}
	if _, ok := loc(Pickup{T: 5, Slot: 513, W: pickupWeaponForm(fam)}); ok {
		t.Error("doublon de cle : attendu non localise")
	}
	if _, ok := loc(Pickup{T: -1, Slot: 514, W: pickupWeaponForm(fam)}); ok {
		t.Error("ramassage avant l origine : attendu non localise")
	}
}
