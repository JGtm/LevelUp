package replay

// document_weapon_changes_test.go — la projection des prises et des lâchers sur l'axe du
// document.
//
// POURQUOI CES TESTS EXISTENT. Le golden d'assemblage ne couvre PAS ce calque : son fixture
// d'entrées a été figé avant lui et ne porte aucun changement d'arme. Sans les tests ci-dessous,
// la couche de projection — filtrage des ré-annonces, conversion en frames — n'aurait aucune
// couverture de non-régression.
//
// LES TESTS DE LA BORNE `until` ONT ÉTÉ RETIRÉS AVEC ELLE (schéma 27) : l'affichage de l'arme
// au sol vit dans `groundWeapons`, borné par l'observation, et ses tests avec lui.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// wcOrigin / wcStep : une origine et un pas ronds, pour que les frames attendues se lisent.
const (
	wcOrigin = uint64(1_000_000)
	wcStep   = uint64(100_000) // 10 frames par seconde
)

func TestBuildWeaponChangesEcarteLesReannonces(t *testing.T) {
	in := []types.HeldWeaponChange{
		{TimestampUS: wcOrigin, Slot: 7, Family: 0xAABBCCDD, Previous: grammar.NoWeaponVariant,
			Kind: types.HeldWeaponTaken},
		{TimestampUS: wcOrigin + 500_000, Slot: 7, Family: 0x11223344,
			Previous: grammar.NoWeaponVariant, Kind: types.HeldWeaponRestated},
	}
	got, cov := buildWeaponChanges(in, wcOrigin, wcStep)
	if len(got) != 1 {
		t.Fatalf("publiés = %d, attendu 1 : une ré-annonce d'arme déjà portée au spawn n'est "+
			"PAS un ramassage et ne doit pas gonfler le compte", len(got))
	}
	if cov.Restated != 1 || cov.Published != 1 || cov.Decoded != 2 {
		t.Errorf("couverture = %+v, attendu decoded=2 published=1 restated=1", cov)
	}
	if got[0].W != "aabbccdd" {
		t.Errorf("W = %q, attendu la FAMILLE en hexa 8 chiffres (même convention que Loadout.W)",
			got[0].W)
	}
}

func TestBuildWeaponChangesEcarteAvantOrigine(t *testing.T) {
	in := []types.HeldWeaponChange{
		{TimestampUS: wcOrigin - 1, Slot: 3, Family: 0xAABBCCDD, Kind: types.HeldWeaponTaken},
	}
	got, cov := buildWeaponChanges(in, wcOrigin, wcStep)
	if len(got) != 0 || cov.BeforeOrigin != 1 {
		t.Fatalf("publiés=%d beforeOrigin=%d : un rejeu ne montre pas ce qui précède sa "+
			"première frame", len(got), cov.BeforeOrigin)
	}
}

func TestBuildWeaponChangesFrameEtLacher(t *testing.T) {
	// Le lâcher tombe à 2 s après l'origine, soit la frame 20 au pas de 100 ms.
	in := []types.HeldWeaponChange{
		{TimestampUS: wcOrigin + 2_000_000, Slot: 5, Family: grammar.NoWeaponVariant,
			Previous: 0x2C0E7F6C, Kind: types.HeldWeaponDropped},
	}
	got, cov := buildWeaponChanges(in, wcOrigin, wcStep)
	if len(got) != 1 {
		t.Fatalf("publiés = %d, attendu 1", len(got))
	}
	if got[0].T != 20 {
		t.Errorf("T = %d, attendu 20 (2 s après l'origine au pas de 100 ms)", got[0].T)
	}
	if got[0].W != "" {
		t.Errorf("W = %q, attendu vide : sur un lâcher l'emplacement n'a plus d'arme", got[0].W)
	}
	if got[0].From != "2c0e7f6c" {
		t.Errorf("From = %q, attendu la famille lâchée : c'est elle qui nomme l'arme au sol",
			got[0].From)
	}
	if cov.Dropped != 1 {
		t.Errorf("couverture dropped = %d, attendu 1", cov.Dropped)
	}
}

func TestSpawnSetFromRendLeRelevePrecedent(t *testing.T) {
	pred := spawnSetFrom([]types.KeyframeLoadout{
		{Slot: 9, TimestampUS: 100, Families: []uint32{1, 2}},
		{Slot: 9, TimestampUS: 300, Families: []uint32{3}},
	}, nil, nil)
	if pred == nil {
		t.Fatal("prédicat nil alors que des loadouts existent")
	}
	st, ok := pred(9, 200)
	if !ok || !st.Families[1] || !st.Families[2] || st.Families[3] {
		t.Errorf("à t=200 le relevé retenu doit être celui de t=100 : got=%v ok=%v", st.Families, ok)
	}
	if _, ok := pred(42, 200); ok {
		t.Error("un slot sans relevé doit rendre ok=false, pas un ensemble vide — les deux ne " +
			"veulent pas dire la même chose pour le classement d'une première émission")
	}
}

// TestSpawnSetFromNeLitJamaisLAvenir — LOT M3.2, ROUGE SUR LA BASE : sans relevé passé, le
// prédicat rendait le PREMIER relevé du slot, fût-il postérieur — une arme ramassée entre-temps
// y figurait déjà, et sa prise se classait « ré-annonce », écartée du document.
func TestSpawnSetFromNeLitJamaisLAvenir(t *testing.T) {
	pred := spawnSetFrom([]types.KeyframeLoadout{
		{Slot: 9, TimestampUS: 300, Families: []uint32{3}},
	}, nil, nil)
	if st, ok := pred(9, 200); ok {
		t.Fatalf("à t=200 aucun relevé ne précède : le relevé de t=300 (%v) a été lu dans l'avenir",
			st.Families)
	}
}

// TestSpawnSetFromBorneALaVie : un relevé antérieur à la CRÉATION du corps qui occupe le slot
// appartient à la vie précédente ; la création voyage dans `DebutDeVie`, même sans relevé.
func TestSpawnSetFromBorneALaVie(t *testing.T) {
	pred := spawnSetFrom([]types.KeyframeLoadout{
		{Slot: 9, TimestampUS: 100, Families: []uint32{1}},
	}, nil, []grammar.BipedCreation{{Slot: 9, TimestampUS: 50}, {Slot: 9, TimestampUS: 250}})
	if st, ok := pred(9, 260); ok || st.DebutDeVie != 250 {
		t.Errorf("t=260, vie née à 250 : ok=%v (attendu false) début=%d (attendu 250)", ok, st.DebutDeVie)
	}
	if st, ok := pred(9, 200); !ok || !st.Families[1] || st.DebutDeVie != 50 {
		t.Errorf("t=200, vie née à 50 : ok=%v familles=%v début=%d", ok, st.Families, st.DebutDeVie)
	}
}

// TestSpawnSetFromDotationDeNaissance : la dotation de naissance situe ses familles par
// emplacement, et l'emporte sur un relevé d'image-clé du MÊME instant.
func TestSpawnSetFromDotationDeNaissance(t *testing.T) {
	births := []types.BirthLoadout{{Slot: 9, TimestampUS: 250, Weapons: []types.BirthWeapon{
		{Emplacement: 0, Family: 0xA}, {Emplacement: 1, Family: 0xB},
		{Emplacement: 2, Family: grammar.NoWeaponVariant},
	}}}
	pred := spawnSetFrom([]types.KeyframeLoadout{{Slot: 9, TimestampUS: 250, Families: []uint32{0xC}}},
		births, []grammar.BipedCreation{{Slot: 9, TimestampUS: 250}})
	st, ok := pred(9, 300)
	if !ok || st.ParEmplacement[0] != 0xA || st.ParEmplacement[1] != 0xB ||
		st.ParEmplacement[2] != grammar.NoWeaponVariant || st.Families[0xC] || !st.Families[0xA] {
		t.Fatalf("dotation mal rendue : ok=%v par emplacement=%v familles=%v", ok, st.ParEmplacement,
			st.Families)
	}
}

// TestBuildWeaponChangesPublieLEmplacement : `k` porte le rang de l'emplacement — 0 compris,
// d'où le pointeur (schéma 69, lot M3.2).
func TestBuildWeaponChangesPublieLEmplacement(t *testing.T) {
	in := []types.HeldWeaponChange{
		{TimestampUS: wcOrigin, Slot: 3, Emplacement: 0, Family: 0xAABBCCDD, Previous: grammar.NoWeaponVariant,
			Kind: types.HeldWeaponTaken},
		{TimestampUS: wcOrigin, Slot: 3, Emplacement: 1, Family: grammar.NoWeaponVariant, Previous: 0x11223344,
			Kind: types.HeldWeaponDropped},
	}
	got, _ := buildWeaponChanges(in, wcOrigin, wcStep)
	if len(got) != 2 || got[0].K == nil || *got[0].K != 0 || got[1].K == nil || *got[1].K != 1 {
		t.Fatalf("emplacements publiés %+v : attendu k=0 puis k=1", got)
	}
}
