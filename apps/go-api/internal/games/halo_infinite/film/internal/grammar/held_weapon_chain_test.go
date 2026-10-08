package grammar

// held_weapon_chain_test.go — LA QUALIFICATION DES ÉMISSIONS D'ARME CONTRE LA DOTATION DE
// NAISSANCE, et la chaîne coupée à chaque vie (lot M3.2 de la campagne « retours rejeu »,
// 2026-09-23).

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
	"levelup/go-api/internal/games/weapons/filmshell"
)

// naissanceAB : une vie née à t=100, arme A à l'emplacement 0, B au 1, le 2 vide.
func naissanceAB() SpawnState {
	return SpawnState{
		Families:       map[uint32]bool{0xA: true, 0xB: true},
		ParEmplacement: map[int]uint32{0: 0xA, 1: 0xB, 2: noVariant},
		DebutDeVie:     100,
	}
}

// TestPremiereEmissionContreLaDotationDeNaissance : la première émission d'un emplacement se lit
// contre l'arme de naissance de CET emplacement, qui devient `Previous`.
func TestPremiereEmissionContreLaDotationDeNaissance(t *testing.T) {
	spawn := func(uint32, uint64) (SpawnState, bool) { return naissanceAB(), true }
	for _, c := range []struct {
		nom         string
		emplacement int
		famille     uint32
		kind        types.HeldWeaponChangeKind
		previous    uint32
	}{
		{"meme arme", 0, 0xA, types.HeldWeaponRestated, noVariant},
		{"echange", 0, 0xC, types.HeldWeaponSwapped, 0xA},
		{"prise sur emplacement vide", 2, 0xC, types.HeldWeaponTaken, noVariant},
		{"lacher nomme", 1, noVariant, types.HeldWeaponDropped, 0xB},
	} {
		ch := types.HeldWeaponChange{TimestampUS: 200, Slot: 9, SlotIndex: 43 + c.emplacement,
			Emplacement: c.emplacement, Family: c.famille, Previous: noVariant}
		newHeldWeaponChain(spawn).qualifier(&ch, 1)
		if ch.Kind != c.kind || ch.Previous != c.previous {
			t.Errorf("%s : %s depuis %08x, attendu %s depuis %08x", c.nom, ch.Kind, ch.Previous,
				c.kind, c.previous)
		}
	}
}

// TestLaChaineDesEmissionsSeCoupeAChaqueVie : un slot se réattribue. La première émission de la
// SECONDE vie se lit contre SA naissance — pas contre la dernière famille de la vie précédente,
// qui en ferait un « échange » depuis une arme que ce corps n'a jamais portée.
func TestLaChaineDesEmissionsSeCoupeAChaqueVie(t *testing.T) {
	spawn := func(_ uint32, at uint64) (SpawnState, bool) {
		if at >= 500 {
			return SpawnState{Families: map[uint32]bool{0xB: true},
				ParEmplacement: map[int]uint32{0: 0xB}, DebutDeVie: 500}, true
		}
		return naissanceAB(), true
	}
	chaine := newHeldWeaponChain(spawn)
	vie1 := types.HeldWeaponChange{TimestampUS: 200, Slot: 9, SlotIndex: 43, Family: 0xC, Previous: noVariant}
	chaine.qualifier(&vie1, 1)
	if vie1.Kind != types.HeldWeaponSwapped || vie1.Previous != 0xA {
		t.Fatalf("vie 1 : %s depuis %08x, attendu swapped depuis a", vie1.Kind, vie1.Previous)
	}
	vie2 := types.HeldWeaponChange{TimestampUS: 600, Slot: 9, SlotIndex: 43, Family: 0xB, Previous: noVariant}
	if chaine.qualifier(&vie2, 1) {
		t.Error("la première émission d'une vie ne peut pas « répéter » la vie précédente")
	}
	if vie2.Kind != types.HeldWeaponRestated || vie2.Previous != noVariant {
		t.Fatalf("vie 2 : %s depuis %08x, attendu restated (l'arme de sa naissance)", vie2.Kind,
			vie2.Previous)
	}
	suite := types.HeldWeaponChange{TimestampUS: 700, Slot: 9, SlotIndex: 43, Family: 0xD, Previous: noVariant}
	chaine.qualifier(&suite, 1)
	if suite.Kind != types.HeldWeaponSwapped || suite.Previous != 0xB {
		t.Fatalf("suite de la vie 2 : %s depuis %08x, attendu swapped depuis b", suite.Kind, suite.Previous)
	}
}

// TestUneAnnonceNEstPasUnChangement : un emplacement annoncé vide dont l'occupant n'est pas connu,
// et une émission qui répète la famille précédente de la vie, sont des RÉ-ANNONCES — le document
// ne les publie pas. Un lâcher se lit seulement quand l'occupant est connu.
func TestUneAnnonceNEstPasUnChangement(t *testing.T) {
	sansDotation := func(uint32, uint64) (SpawnState, bool) {
		return SpawnState{Families: map[uint32]bool{0xA: true}, DebutDeVie: 100}, true
	}
	chaine := newHeldWeaponChain(sansDotation)
	vide := types.HeldWeaponChange{TimestampUS: 200, Slot: 9, SlotIndex: 45, Emplacement: 2, Family: noVariant,
		Previous: noVariant}
	chaine.qualifier(&vide, 1)
	if vide.Kind != types.HeldWeaponRestated {
		t.Fatalf("emplacement annonce vide sans occupant connu : %s, attendu une re-annonce", vide.Kind)
	}
	encoreVide := types.HeldWeaponChange{TimestampUS: 300, Slot: 9, SlotIndex: 45, Emplacement: 2,
		Family: noVariant, Previous: noVariant}
	if repete := chaine.qualifier(&encoreVide, 1); !repete || encoreVide.Kind != types.HeldWeaponRestated {
		t.Fatalf("re-annonce du vide : %s (repetition %v), attendu une re-annonce comptee", encoreVide.Kind, repete)
	}
	prise := types.HeldWeaponChange{TimestampUS: 400, Slot: 9, SlotIndex: 45, Emplacement: 2, Family: 0xC,
		Previous: noVariant}
	chaine.qualifier(&prise, 1)
	if prise.Kind != types.HeldWeaponTaken {
		t.Fatalf("prise sur l emplacement vide : %s", prise.Kind)
	}
	memeArme := types.HeldWeaponChange{TimestampUS: 500, Slot: 9, SlotIndex: 45, Emplacement: 2, Family: 0xC,
		Previous: noVariant}
	if chaine.qualifier(&memeArme, 1); memeArme.Kind != types.HeldWeaponRestated {
		t.Fatalf("la meme arme re-annoncee : %s, attendu une re-annonce", memeArme.Kind)
	}
	lacher := types.HeldWeaponChange{TimestampUS: 600, Slot: 9, SlotIndex: 45, Emplacement: 2, Family: noVariant,
		Previous: noVariant}
	if chaine.qualifier(&lacher, 1); lacher.Kind != types.HeldWeaponDropped || lacher.Previous != 0xC {
		t.Fatalf("lacher de l arme connue : %s depuis %08x, attendu un lacher nomme", lacher.Kind, lacher.Previous)
	}
}

// naissanceMainsNues : une vie née à t=100 dont la dotation tient A à l'emplacement 0 et l'objet
// « mains nues » au 2 — la forme que le jeu remet à chaque naissance (mise en place d'une manche).
func naissanceMainsNues() SpawnState {
	return SpawnState{
		Families:       map[uint32]bool{0xA: true, filmshell.UnarmedFamily: true},
		ParEmplacement: map[int]uint32{0: 0xA, 2: filmshell.UnarmedFamily},
		DebutDeVie:     100,
	}
}

// TestLesMainsNuesValentRienEnMain (découverte D9 du lot des arrêts de la vue B) : contre une
// dotation qui tient les mains nues, l'annonce « emplacement vide » ne change rien — aucun lâcher
// des mains nues ; la prise qui suit est une PRISE, jamais un échange depuis les mains nues.
func TestLesMainsNuesValentRienEnMain(t *testing.T) {
	spawn := func(uint32, uint64) (SpawnState, bool) { return naissanceMainsNues(), true }
	chaine := newHeldWeaponChain(spawn)
	emission := func(ts uint64, fam uint32) types.HeldWeaponChange {
		ch := types.HeldWeaponChange{TimestampUS: ts, Slot: 9, SlotIndex: 45, Emplacement: 2, Family: fam,
			Previous: noVariant}
		chaine.qualifier(&ch, 1)
		return ch
	}
	if vide := emission(200, noVariant); vide.Kind != types.HeldWeaponRestated {
		t.Fatalf("annonce vide contre les mains nues de naissance : %s depuis %08x, attendu une re-annonce",
			vide.Kind, vide.Previous)
	}
	if prise := emission(300, 0xC); prise.Kind != types.HeldWeaponTaken || prise.Previous != noVariant {
		t.Fatalf("prise apres l annonce vide : %s depuis %08x, attendu une prise", prise.Kind, prise.Previous)
	}
	// Sans l'annonce vide (la base qui ne la lisait pas), la première émission se juge contre la
	// dotation elle-même : c'est encore une prise.
	direct := types.HeldWeaponChange{TimestampUS: 300, Slot: 9, SlotIndex: 45, Emplacement: 2, Family: 0xC,
		Previous: noVariant}
	newHeldWeaponChain(spawn).qualifier(&direct, 1)
	if direct.Kind != types.HeldWeaponTaken || direct.Previous != noVariant {
		t.Fatalf("prise contre les mains nues de naissance : %s depuis %08x, attendu une prise", direct.Kind,
			direct.Previous)
	}
}

// TestAucunChangementNePorteLesMainsNues : dans la chaîne d'une vie, les mains nues ne sont jamais
// l'objet d'un changement qualifié — ni pris (hors la remise sur emplacement vide, que la
// publication écarte), ni échangé, ni lâché.
func TestAucunChangementNePorteLesMainsNues(t *testing.T) {
	u := filmshell.UnarmedFamily
	for _, c := range []struct {
		nom      string
		prev     uint32
		famille  uint32
		kind     types.HeldWeaponChangeKind
		previous uint32
		publiee  uint32
	}{
		{"remise sur emplacement vide", noVariant, u, types.HeldWeaponTaken, noVariant, u},
		{"mains nues re-annoncees", u, u, types.HeldWeaponRestated, u, u},
		{"vide contre mains nues", u, noVariant, types.HeldWeaponRestated, u, noVariant},
		{"arme contre mains nues", u, 0xC, types.HeldWeaponTaken, noVariant, 0xC},
		{"mains nues apres une arme", 0xC, u, types.HeldWeaponDropped, 0xC, noVariant},
		{"echange entre deux armes", 0xC, 0xD, types.HeldWeaponSwapped, 0xC, 0xD},
	} {
		ch := types.HeldWeaponChange{Family: c.famille, Previous: c.prev}
		qualifyHeldWeaponChange(&ch, true, SpawnState{}, false)
		if ch.Kind != c.kind || ch.Previous != c.previous || ch.Family != c.publiee {
			t.Errorf("%s : %s %08x depuis %08x, attendu %s %08x depuis %08x", c.nom, ch.Kind, ch.Family,
				ch.Previous, c.kind, c.publiee, c.previous)
		}
	}
}
