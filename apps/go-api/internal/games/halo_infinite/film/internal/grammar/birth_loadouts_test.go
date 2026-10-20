package grammar

// birth_loadouts_test.go — LA DOTATION DE NAISSANCE NE SE REND QUE SI LE RECORD SE FERME (lot M3.2
// de la campagne « retours rejeu », 2026-09-23).
//
// Paquets SYNTHÉTIQUES : un record NEW de bipède (forme de `default_state_ds32_test.go`) suivi
// soit d'un record DELTA propre sur un slot que le monde connaît — la dotation est rendue —,
// soit d'un en-tête qu'aucune lecture ne confirme — rien n'est rendu, et le refus est compté.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// Les deux familles de la dotation synthétique, et le slot du corps.
const (
	naissanceFamille1 = 0x48C19D2D
	naissanceFamille2 = 0xF408190F
	naissanceSlot     = 530
	voisinSlot        = 7 // un slot d'un archétype sans composant, lié au monde du test
)

// ecrireNaissance écrit l'en-tête NEW d'un bipède puis son corps : état par défaut, porte,
// masque clairsemé {43, 44} et les deux emplacements d'arme (queue nulle).
func ecrireNaissance(w *bitWriter) {
	w.bit(0)     // pas un delta
	w.bits(1, 2) // recNew
	w.bits(naissanceSlot, woNewSlotBits)
	w.bits(1, woNewGenBits)
	w.bits(BipedTypeIndex, woNewTIBits)
	ecrireEtatParDefautBipede(w, true, 0x7F5C4A1C)
	w.bit(1) // porte du record NEW
	w.bit(0) // masque clairsemé
	w.bits(2, 3)
	w.bits(43, 6)
	w.bits(44, 6)
	queue := largeurArmeQueueNulle() - 65
	for _, fam := range []uint64{naissanceFamille1, naissanceFamille2} {
		w.bit(1)
		w.bits(fam, 32)
		w.bits(0x11111111, 32)
		w.bits(0, queue)
	}
}

// ecrireDeltaVide écrit un record DELTA sur `slot`, sans ligne de base ni composant.
func ecrireDeltaVide(w *bitWriter, slot uint64) {
	w.bit(1) // recDelta
	w.bits(slot, 13)
	w.bits(1, 2) // tag
	w.bit(0)     // ligne de base
	w.bit(0)     // masque clairsemé
	w.bits(0, 3) // aucun composant
}

// scanDeNaissanceDeTest rend une lecture de naissance sur un monde qui connaît le slot voisin.
func scanDeNaissanceDeTest() (*birthScan, *types.BirthLoadoutStats) {
	reg := registreBipedeArmes()
	st := &types.BirthLoadoutStats{}
	s := newBirthScan(nil, reg, weaponEmplacements(reg.Archetypes[BipedTypeIndex]), st)
	s.monde.BindFull(1<<30|voisinSlot, 1) // archétype 1 : aucun composant
	return s, st
}

// TestNaissanceFermeeParUnDeltaPropre : le record est suivi d'un delta propre sur un slot lié —
// la dotation est rendue, emplacement par emplacement.
func TestNaissanceFermeeParUnDeltaPropre(t *testing.T) {
	w := &bitWriter{}
	ecrireNaissance(w)
	ecrireDeltaVide(w, voisinSlot)
	w.bits(0, 64)
	s, st := scanDeNaissanceDeTest()
	bl, ok := s.lire(w.buf, BipedCreation{Slot: naissanceSlot, Generation: 1, TimestampUS: 42})
	if !ok {
		t.Fatalf("dotation non rendue : %+v", *st)
	}
	want := []types.BirthWeapon{{Emplacement: 0, Family: naissanceFamille1, Low: 0x11111111},
		{Emplacement: 1, Family: naissanceFamille2, Low: 0x11111111}}
	if len(bl.Weapons) != 2 || bl.Weapons[0] != want[0] || bl.Weapons[1] != want[1] {
		t.Fatalf("emplacements %+v, attendu %+v", bl.Weapons, want)
	}
	if st.Read != 1 || st.ClosedByDelta != 1 || st.Unconfirmed != 0 {
		t.Fatalf("compteurs %+v : attendu 1 lue, fermée par un delta", *st)
	}
}

// TestNaissanceNonConfirmeeNeRendRien : ce qui suit le record n'est confirmé par RIEN (un en-tête
// NEW d'un slot inconnu, que la table anticipée ne déclare pas) — aucune dotation, un refus compté.
func TestNaissanceNonConfirmeeNeRendRien(t *testing.T) {
	w := &bitWriter{}
	ecrireNaissance(w)
	w.bit(0)
	w.bits(1, 2) // recNew
	w.bits(900, woNewSlotBits)
	w.bits(1, woNewGenBits)
	w.bits(12, woNewTIBits)
	w.bits(0, 64)
	s, st := scanDeNaissanceDeTest()
	if bl, ok := s.lire(w.buf, BipedCreation{Slot: naissanceSlot, Generation: 1}); ok {
		t.Fatalf("dotation rendue sans fermeture : %+v", bl)
	}
	if st.Unconfirmed != 1 || st.Read != 0 {
		t.Fatalf("compteurs %+v : attendu 1 non confirmée, 0 lue", *st)
	}
}

// TestNaissanceDeBruitNeRendRien : après un état par défaut valide, le corps n'est que du bruit
// — la porte et le masque se lisent dans des bits quelconques. La traversée ne se ferme pas et
// RIEN n'est rendu : pas une dotation lue de travers.
func TestNaissanceDeBruitNeRendRien(t *testing.T) {
	w := &bitWriter{}
	w.bit(0)
	w.bits(1, 2)
	w.bits(naissanceSlot, woNewSlotBits)
	w.bits(1, woNewGenBits)
	w.bits(BipedTypeIndex, woNewTIBits)
	ecrireEtatParDefautBipede(w, false, 0) // porte finale fermée : aucun R(32) à lire
	w.bits(0x7F5C4A1C, 32)                 // du bruit là où la porte et le masque sont attendus
	w.bits(0, 256)
	s, st := scanDeNaissanceDeTest()
	if bl, ok := s.lire(w.buf, BipedCreation{Slot: naissanceSlot, Generation: 1}); ok {
		t.Fatalf("dotation rendue sur une traversée de bruit : %+v", bl)
	}
	if st.Read != 0 || st.Desync+st.Overflow+st.Unconfirmed+st.NoWeaponComponent != 1 {
		t.Fatalf("compteurs %+v : attendu un refus compté, aucune lecture", *st)
	}
}
