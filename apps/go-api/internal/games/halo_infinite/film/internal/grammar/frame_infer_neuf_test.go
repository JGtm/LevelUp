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
	o.refuserUnNeuf(w, &FrameRecord{Slot: 123, TypeIndex: 2}) // un observateur nil ne compte pas
	o.jugerLesNeufsRefuses(nil)
	o.solderLesNeufsRefuses()
	o = NouvelleObservation()
	o.refuserUnNeuf(w, &FrameRecord{Slot: 500, TypeIndex: 36})
	if o.NeufsContreUnVivant != 1 || len(o.neufsRefuses) != 1 || o.neufsRefuses[0] != (neufRefuse{500, 36, 35}) {
		t.Errorf("compteur %d, en attente %+v : attendu 1 et {500 36 35}", o.NeufsContreUnVivant, o.neufsRefuses)
	}
}

// TestLeVerdictDesNeufsRefuses (constat DFIX-R6) : l image-cle suivante SEPARE la lecture fausse
// (elle redonne au slot l archetype du vivant) de la vraie creation perdue (elle lui donne celui du
// NEW : le DEL du vivant n avait pas ete lu) ; ni l un ni l autre, ou aucune image-cle : indecis.
func TestLeVerdictDesNeufsRefuses(t *testing.T) {
	o := NouvelleObservation()
	o.neufsRefuses = []neufRefuse{{slot: 1, neuf: 2, vivant: 4}, {slot: 2, neuf: 2, vivant: 4},
		{slot: 3, neuf: 2, vivant: 4}, {slot: 4, neuf: 2, vivant: 4}}
	o.jugerLesNeufsRefuses(map[uint32]uint32{1: 4, 2: 2, 3: 7})
	if o.NeufsRefusesLecturesFausses != 1 || o.NeufsRefusesCreationsPerdues != 1 || o.NeufsRefusesIndecis != 2 ||
		len(o.neufsRefuses) != 0 {
		t.Fatalf("fausses %d, perdues %d, indecis %d, en attente %d : attendu 1, 1, 2, 0",
			o.NeufsRefusesLecturesFausses, o.NeufsRefusesCreationsPerdues, o.NeufsRefusesIndecis, len(o.neufsRefuses))
	}
	o.neufsRefuses = []neufRefuse{{slot: 9, neuf: 2, vivant: 4}}
	o.solderLesNeufsRefuses() // la fin du film : aucune image-cle ne le juge
	if o.NeufsRefusesIndecis != 3 || len(o.neufsRefuses) != 0 {
		t.Fatalf("indecis %d apres le solde, attendu 3", o.NeufsRefusesIndecis)
	}
	if got := archetypesDeclares([]KeyframeRec{{Slot: 5, TI: 9}}, map[uint32]uint32{5: 38, 6: 30}); got[5] != 9 || got[6] != 30 {
		t.Fatalf("archetypes declares %v : la chaine prime sur la table, la table complete", got)
	}
}

// newEmpty ecrit un record NEW d un archetype SANS composant : type R(1)=0 puis R(2)=1, id R(11)
// + etiquette R(2), typeIndex R(6), etat par defaut de largeur nulle (archetype sans
// deserialiseur porte), porte R(1)=0, masque vide (R(1)=0 + R(3)=0).
func (w *bitWriter) newEmpty(slot uint32, tag uint64, ti uint64) {
	w.bit(0)
	w.bits(recNew, 2)
	w.bits(uint64(slot), 11)
	w.bits(tag, 2)
	w.bits(ti, 6)
	w.bit(0)
	w.bit(0)
	w.bits(0, 3)
}

// TestUnNeufContreUnVivantNeLiePasEtLaMarcheContinue eprouve le BRANCHEMENT de
// [corpsDeRecordNeuf] sur une trame synthetique decodee par [DecodeFrameInfer] (constat DFIX-R1 de
// la revue adverse : le predicat seul etait teste). Un NEW propre d un AUTRE archetype sur un slot
// lie en dur par l image-cle : la liaison reste celle de l image-cle, les records suivants de la
// trame sont lus, et le refus se compte. MUTATION : `case false && contreditUneEntiteVivante(...)`
// -> la liaison passe a l archetype du NEW, ROUGE.
func TestUnNeufContreUnVivantNeLiePasEtLaMarcheContinue(t *testing.T) {
	reg := &Registry{Archetypes: []Archetype{{Index: 0}, {Index: 1}, {Index: 2}, {Index: 3}, {Index: 4}}}
	w := NewWorld(reg)
	w.BindImageCle(1, 123, 4) // la chaine de l image-cle : liaison EN DUR, archetype 4
	w.BindImageCle(1, 124, 4)

	var bw bitWriter
	bw.newEmpty(123, 1, 2) // NEW propre, archetype 2, sur un slot vivant d archetype 4
	bw.newEmpty(300, 1, 2) // le record suivant : un NEW sur un slot LIBRE, il se lie
	bw.deltaEmpty(124)     // puis un delta du slot vivant voisin
	bw.end()

	emptyCfg.Obs = NouvelleObservation()
	withChain(false, func() {
		recs, _ := DecodeFrameInfer(bw.buf, w, emptyCfg)
		if len(recs) != 3 {
			t.Fatalf("records lus %d, attendu 3 (la marche continue apres le NEW refuse) : %+v", len(recs), recs)
		}
		for _, r := range recs {
			if r.DesyncAt != -1 {
				t.Fatalf("record slot %d desynchronise a %d", r.Slot, r.DesyncAt)
			}
		}
	})
	if ti, ok := w.ArchetypeForSlot(123); !ok || ti != 4 {
		t.Errorf("slot 123 lie a %d (%v), attendu 4 : le NEW refuse a ecrase l entite vivante", ti, ok)
	}
	if ti, ok := w.ArchetypeForSlot(300); !ok || ti != 2 {
		t.Errorf("slot 300 lie a %d (%v), attendu 2 : un NEW sur un slot libre se lie", ti, ok)
	}
	if got := emptyCfg.Obs.NeufsContreUnVivant; got != 1 || len(emptyCfg.Obs.neufsRefuses) != 1 {
		t.Errorf("NeufsContreUnVivant %d, en attente %d : attendu 1 et 1", got, len(emptyCfg.Obs.neufsRefuses))
	}
}
