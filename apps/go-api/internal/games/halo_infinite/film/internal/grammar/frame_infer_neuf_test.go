package grammar

import "testing"

// frame_infer_neuf_test.go — UN NEW NE REMPLACE PAS UNE ENTITE VIVANTE (lot D-fix, 2026-09-24).
//
// Le defaut mesure (`0797ce72`, chunk 12) : un NEW « slot 123, ti 2 », lecture fausse atteinte par
// la traversee du NEW de bipede, ecrasait le `ti 4` que l image-cle et chaque paquet donnaient a ce
// slot. ROUGE AVANT : le NEW propre se liait toujours (`BindFull`), quelle que soit la liaison.
func TestUnNeufNeRemplacePasUneEntiteVivante(t *testing.T) {
	w := NewWorld(nil)
	w.BindImageCle(1, 123, 4) // la chaine de l image-cle : liaison EN DUR
	w.BindFull(1<<30|500, 35) // un NEW propre anterieur : EN DUR aussi
	w.BindDatum(600, 30)      // la table de datums : liaison SOUPLE
	cas := []struct {
		nom     string
		slot    uint32
		ti      uint32
		contred bool
	}{
		{"autre archetype sur une liaison d image-cle", 123, 2, true},
		{"autre archetype sur une liaison de NEW", 500, 36, true},
		{"meme archetype : re-creation sans contradiction", 123, 4, false},
		{"liaison de datum : elle cede au NEW", 600, 35, false},
		{"slot libre", 700, 35, false},
	}
	for _, c := range cas {
		rec := &FrameRecord{Type: recNew, Slot: c.slot, TypeIndex: c.ti}
		if got := contreditUneEntiteVivante(w, rec); got != c.contred {
			t.Errorf("%s : contredit %v, attendu %v", c.nom, got, c.contred)
		}
	}
	w.Unbind(123) // un DEL lu libere l entree
	if contreditUneEntiteVivante(w, &FrameRecord{Type: recNew, Slot: 123, TypeIndex: 2}) {
		t.Error("apres un DEL, le NEW d un autre archetype ne contredit plus rien")
	}
	var o *Observation
	o.compterNeufContreUnVivant() // un observateur nil ne compte pas et ne panique pas
	o = NouvelleObservation()
	o.compterNeufContreUnVivant()
	if o.NeufsContreUnVivant != 1 {
		t.Errorf("compteur %d, attendu 1", o.NeufsContreUnVivant)
	}
}
