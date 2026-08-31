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

	"levelup/go-api/internal/analysis/filmdec"
)

// wcOrigin / wcStep : une origine et un pas ronds, pour que les frames attendues se lisent.
const (
	wcOrigin = uint64(1_000_000)
	wcStep   = uint64(100_000) // 10 frames par seconde
)

func TestBuildWeaponChangesEcarteLesReannonces(t *testing.T) {
	in := []filmdec.HeldWeaponChange{
		{TimestampUS: wcOrigin, Slot: 7, Family: 0xAABBCCDD, Previous: filmdec.NoWeaponVariant,
			Kind: filmdec.HeldWeaponTaken},
		{TimestampUS: wcOrigin + 500_000, Slot: 7, Family: 0x11223344,
			Previous: filmdec.NoWeaponVariant, Kind: filmdec.HeldWeaponRestated},
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
	in := []filmdec.HeldWeaponChange{
		{TimestampUS: wcOrigin - 1, Slot: 3, Family: 0xAABBCCDD, Kind: filmdec.HeldWeaponTaken},
	}
	got, cov := buildWeaponChanges(in, wcOrigin, wcStep)
	if len(got) != 0 || cov.BeforeOrigin != 1 {
		t.Fatalf("publiés=%d beforeOrigin=%d : un rejeu ne montre pas ce qui précède sa "+
			"première frame", len(got), cov.BeforeOrigin)
	}
}

func TestBuildWeaponChangesFrameEtLacher(t *testing.T) {
	// Le lâcher tombe à 2 s après l'origine, soit la frame 20 au pas de 100 ms.
	in := []filmdec.HeldWeaponChange{
		{TimestampUS: wcOrigin + 2_000_000, Slot: 5, Family: filmdec.NoWeaponVariant,
			Previous: 0x2C0E7F6C, Kind: filmdec.HeldWeaponDropped},
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
	pred := spawnSetFrom([]filmdec.KeyframeLoadout{
		{Slot: 9, TimestampUS: 100, Families: []uint32{1, 2}},
		{Slot: 9, TimestampUS: 300, Families: []uint32{3}},
	})
	if pred == nil {
		t.Fatal("prédicat nil alors que des loadouts existent")
	}
	set, ok := pred(9, 200)
	if !ok || !set[1] || !set[2] || set[3] {
		t.Errorf("à t=200 le relevé retenu doit être celui de t=100 : got=%v ok=%v", set, ok)
	}
	if _, ok := pred(42, 200); ok {
		t.Error("un slot sans relevé doit rendre ok=false, pas un ensemble vide — les deux ne " +
			"veulent pas dire la même chose pour le classement d'une première émission")
	}
}
