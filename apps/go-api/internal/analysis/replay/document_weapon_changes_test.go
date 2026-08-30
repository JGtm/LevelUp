package replay

// document_weapon_changes_test.go — la projection des prises et des lâchers sur l'axe du
// document.
//
// POURQUOI CES TESTS EXISTENT. Le golden d'assemblage ne couvre PAS ce calque : son fixture
// d'entrées a été figé avant lui et ne porte aucun changement d'arme. Sans les tests ci-dessous,
// la couche de projection — filtrage des ré-annonces, conversion en frames, borne d'affichage —
// n'aurait aucune couverture de non-régression.

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
	got, cov := buildWeaponChanges(in, wcOrigin, wcStep, 0)
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
	got, cov := buildWeaponChanges(in, wcOrigin, wcStep, 0)
	if len(got) != 0 || cov.BeforeOrigin != 1 {
		t.Fatalf("publiés=%d beforeOrigin=%d : un rejeu ne montre pas ce qui précède sa "+
			"première frame", len(got), cov.BeforeOrigin)
	}
}

func TestBuildWeaponChangesFrameEtBorneDAffichage(t *testing.T) {
	// Le lâcher tombe à 2 s après l'origine, soit la frame 20 au pas de 100 ms.
	const gravityHammer = 0x2C0E7F6C // hors catalogue de test : la table par défaut s'applique
	in := []filmdec.HeldWeaponChange{
		{TimestampUS: wcOrigin + 2_000_000, Slot: 5, Family: filmdec.NoWeaponVariant,
			Previous: gravityHammer, Kind: filmdec.HeldWeaponDropped},
	}
	got, _ := buildWeaponChanges(in, wcOrigin, wcStep, 0)
	if len(got) != 1 {
		t.Fatalf("publiés = %d, attendu 1", len(got))
	}
	if got[0].T != 20 {
		t.Errorf("T = %d, attendu 20 (2 s après l'origine au pas de 100 ms)", got[0].T)
	}
	if got[0].W != "" {
		t.Errorf("W = %q, attendu vide : sur un lâcher l'emplacement n'a plus d'arme", got[0].W)
	}
	want := 20 + weaponDespawnDefaultSeconds*10
	if got[0].Until != want {
		t.Errorf("Until = %d, attendu %d (frame du lâcher + durée d'affichage)", got[0].Until, want)
	}
}

func TestBuildWeaponChangesBorneAuDernierFrame(t *testing.T) {
	in := []filmdec.HeldWeaponChange{
		{TimestampUS: wcOrigin, Slot: 5, Family: filmdec.NoWeaponVariant, Previous: 0xDEADBEEF,
			Kind: filmdec.HeldWeaponDropped},
	}
	got, _ := buildWeaponChanges(in, wcOrigin, wcStep, 4)
	if len(got) != 1 || got[0].Until != 4 {
		t.Fatalf("Until = %v, attendu 4 : la borne d'affichage ne dépasse jamais la dernière "+
			"frame du document", got)
	}
}

func TestBuildWeaponChangesLacherSansPrecedentNaPasDeBorne(t *testing.T) {
	// Un lâcher dont l'arme précédente est inconnue ne peut pas nommer de durée d'affichage :
	// publier une borne reviendrait à choisir une durée au hasard.
	in := []filmdec.HeldWeaponChange{
		{TimestampUS: wcOrigin, Slot: 5, Family: filmdec.NoWeaponVariant,
			Previous: filmdec.NoWeaponVariant, Kind: filmdec.HeldWeaponDropped},
	}
	got, _ := buildWeaponChanges(in, wcOrigin, wcStep, 0)
	if len(got) != 1 {
		t.Fatalf("publiés = %d, attendu 1 : le lâcher reste publié, seule sa borne manque", len(got))
	}
	if got[0].Until != 0 {
		t.Errorf("Until = %d, attendu 0 : sans arme précédente, aucune durée n'est nommable",
			got[0].Until)
	}
}

func TestDespawnSecondsFamilleInconnueRendLaPlusCourte(t *testing.T) {
	if s := despawnSecondsFor(0xFFFFFFFE); s != weaponDespawnDefaultSeconds {
		t.Errorf("durée = %d, attendu %d : une famille hors table s'efface TÔT plutôt que de "+
			"rester affichée sur une carte où elle n'est peut-être plus", s, weaponDespawnDefaultSeconds)
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
