package grammar

import "testing"

// neufs_prouves_test.go — un NEW refuse contre une entite vivante est lie quand la fermeture de sa
// trame le prouve, et seulement alors.

// mondeAvecUnVivant rend un monde ou le slot 553 est lie en dur a l archetype 20 (generation 0),
// et le record NEW (archetype 35, generation 1, en-tete au bit 900) que la marche vient de refuser.
func mondeAvecUnVivant() (*World, *FrameRecord) {
	w := NewWorld(&Registry{})
	w.BindFull(553, 20)
	rec := &FrameRecord{Type: recNew, ID: 1<<30 | 553, Slot: 553, TypeIndex: 35, HeaderBit: 900}
	if !contreditUneEntiteVivante(w, rec) {
		panic("le NEW doit contredire l entite vivante")
	}
	w.suspendreUnNeuf(rec)
	return w, rec
}

// TestUnNeufProuveParSaTrameEstLie : la trame prouve tout (0), ou depuis un debut localise avant le
// NEW : il est lie, avec son identifiant complet.
func TestUnNeufProuveParSaTrameEstLie(t *testing.T) {
	for _, prouvee := range []uint32{0, 899, 900} {
		w, rec := mondeAvecUnVivant()
		w.lierLesNeufsProuves(prouvee)
		ti, ok := w.ArchetypeForSlot(553)
		if !ok || ti != 35 || w.slots[553].FullID != rec.ID {
			t.Errorf("prouvee des %d : slot 553 -> ti %d (lie %v), attendu le NEW (ti 35)", prouvee, ti, ok)
		}
		if len(w.neufsSuspendus) != 0 {
			t.Errorf("prouvee des %d : la liste des suspendus n est pas videe", prouvee)
		}
	}
}

// TestUnNeufNonProuveResteRefuse : la trame ne prouve rien, ou seulement apres le NEW : le monde
// garde l entite vivante.
func TestUnNeufNonProuveResteRefuse(t *testing.T) {
	for _, prouvee := range []uint32{rienDeProuve, 901} {
		w, _ := mondeAvecUnVivant()
		w.lierLesNeufsProuves(prouvee)
		if ti, _ := w.ArchetypeForSlot(553); ti != 20 {
			t.Errorf("prouvee des %d : slot 553 -> ti %d, attendu l entite vivante (ti 20)", prouvee, ti)
		}
	}
}

// TestLesMarchesDEssaiNeLientRien : la marche d une trame commence par oublier les NEW suspendus
// des marches d essai qui l ont precedee.
func TestLesMarchesDEssaiNeLientRien(t *testing.T) {
	w, _ := mondeAvecUnVivant()
	w.oublierLesNeufsSuspendus()
	w.lierLesNeufsProuves(0)
	if ti, _ := w.ArchetypeForSlot(553); ti != 20 {
		t.Errorf("slot 553 -> ti %d : un NEW d une marche d essai a ete lie", ti)
	}
}
