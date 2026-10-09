package grammar

// default_state_ds32_test.go — LA DERNIERE FEUILLE DE L'ETAT PAR DEFAUT DU BIPEDE LIT SON R(32)
// QUAND SA PORTE VAUT 1 (lot M3.2 de la campagne « retours rejeu », 2026-09-23).
//
// LES DEUX PREMIERS TESTS ETAIENT ROUGES SUR LA BASE `fe7079f41` (preuve rejouee au lot) : la
// lecture s'arretait 32 bits trop tot, la porte et le masque de presence du record NEW se lisaient
// dans ce mot de 32 bits, et aucune arme n'etait lue a sa place.
//
// Les records sont SYNTHETIQUES : la forme mesuree d'un record NEW de naissance (selecteur par
// defaut, derniere porte a 1, masque clairseme annoncant les emplacements d'arme) avec des valeurs
// choisies pour que la moindre derive de lecture change la famille lue.

import "testing"

// ecrireEtatParDefautBipede ecrit un etat par defaut de bipede au selecteur par defaut (13), sans
// nom de representation, et le termine par la derniere feuille : sa porte, puis son R(32) quand
// elle vaut 1.
func ecrireEtatParDefautBipede(w *bitWriter, porteFinale bool, motFinal uint64) {
	mpp := ProfilDeBalayageParDefaut().MPP
	w.bit(0) // g0 : le selecteur reste 13
	w.bit(0) // gRep : pas de nom de representation
	w.bit(1) // FUN_1407f2058 (selecteur > 10) : porte a 1, pas de R(5)
	// FUN_14080cfe8, le bloc des proprietes multijoueur.
	w.bits(0x155, mpp.Lead) // R(lead)
	w.bits(0xCAFEF00D, 32)  // R(32)
	w.bit(1)                // porte du variant-name a 1 : consultation, 0 bit
	w.bit(0)                // porte du R(18)
	w.bit(0)                // FUN_14080d524 : porte fermee
	w.bits(0, 2)            // R(2)
	w.bits(0, mpp.Index)    // R(index)
	w.bits(0, 3)            // compte nul
	w.bit(0)                // FUN_14080d4d0 : porte fermee
	w.bit(0)                // queue G3 : porte fermee
	w.bit(0)                // gC6
	w.bit(0)                // R(1) inconditionnel
	w.bit(0)                // FUN_14080d69c : porte fermee
	w.bits(0x5A5A5, 19)     // FUN_14076dc04
	w.bit(1)                // selecteur > 5 : R(1)
	if porteFinale {        // selecteur >= 12 : la derniere feuille
		w.bit(1)
		w.bits(motFinal, 32)
		return
	}
	w.bit(0)
}

// TestEtatParDefautBipedeLitLeR32DeLaDerniereFeuille : la lecture s'arrete EXACTEMENT apres le
// R(32) quand la porte vaut 1, et apres la seule porte quand elle vaut 0.
func TestEtatParDefautBipedeLitLeR32DeLaDerniereFeuille(t *testing.T) {
	for _, c := range []struct {
		nom   string
		porte bool
	}{{"porte a 1", true}, {"porte a 0", false}} {
		w := &bitWriter{}
		ecrireEtatParDefautBipede(w, c.porte, 0x7F5C4A1C)
		fin := w.n
		w.bits(0xFFFF, 16) // la suite du record
		br := LecteurSur(w.buf)
		consumeBipedDefaultState(br)
		if br.BitPos() != fin {
			t.Errorf("%s : l'etat par defaut finit au bit %d, attendu %d", c.nom, br.BitPos(), fin)
		}
	}
}

// registreBipedeArmes rend un registre dont l'archetype 35 porte ses quatre emplacements d'arme
// aux index 43..46, comme le registre des films mesures (`registry_test.go`), et aucun autre
// composant nomme : le masque du record decide seul de ce qui se lit.
func registreBipedeArmes() *Registry {
	reg := &Registry{Archetypes: make([]Archetype, BipedTypeIndex+1)}
	for i := range reg.Archetypes {
		reg.Archetypes[i].Index = i
	}
	comps := make([]string, archetypeBlockSlots)
	for i := 43; i <= 46; i++ {
		comps[i] = compWeaponStateTypeInfo
	}
	reg.Archetypes[BipedTypeIndex].Components = comps
	return reg
}

// largeurArmeQueueNulle rend la largeur d'un emplacement d'arme present dont tout ce qui suit la
// famille et la variante vaut zero, telle que le lecteur de production la consomme.
func largeurArmeQueueNulle() int {
	w := &bitWriter{}
	w.bit(1)
	w.bits(0x48C19D2D, 32)
	w.bits(0x11111111, 32)
	w.bits(0, 1024)
	br := LecteurSur(w.buf)
	consumeWeaponStateTypeInfoVariant(br)
	return br.BitPos()
}

// TestTraverseEntityNaissanceLitLesArmesI43I44 : un record NEW de bipede dont le masque annonce
// i43 et i44 rend, dans l'ordre, les deux familles et leurs variantes.
func TestTraverseEntityNaissanceLitLesArmesI43I44(t *testing.T) {
	type arme struct{ fam, low uint32 }
	voulu := []arme{{0x48C19D2D, 0x9E2A4C11}, {0xF408190F, 0x0B1D5E77}}
	queue := largeurArmeQueueNulle() - 65
	w := &bitWriter{}
	w.bits(BipedTypeIndex, 6)
	ecrireEtatParDefautBipede(w, true, 0x7F5C4A1C)
	w.bit(1)     // porte du record NEW
	w.bit(0)     // masque clairseme
	w.bits(2, 3) // deux index
	w.bits(43, 6)
	w.bits(44, 6)
	for _, a := range voulu {
		w.bit(1)
		w.bits(uint64(a.fam), 32)
		w.bits(uint64(a.low), 32)
		w.bits(0, queue)
	}
	w.bits(0, 64) // la suite du paquet

	var lus []arme
	obs := NouvelleObservation()
	obs.HeldWeaponHook = func(h, l uint32) { lus = append(lus, arme{h, l}) }
	br := LecteurSur(w.buf)
	br.PoserObservation(obs)
	tr := TraverseEntity(br, registreBipedeArmes(), 0)
	if tr.DesyncAt != -1 {
		t.Fatalf("traversee desynchronisee a i%d (masque %x)", tr.DesyncAt, tr.Mask)
	}
	if len(lus) != len(voulu) || lus[0] != voulu[0] || lus[1] != voulu[1] {
		t.Fatalf("armes lues %x, attendu %x (masque %x)", lus, voulu, tr.Mask)
	}
	if want := w.n - 64; tr.EndBit != want {
		t.Fatalf("le record finit au bit %d, attendu %d", tr.EndBit, want)
	}
}
