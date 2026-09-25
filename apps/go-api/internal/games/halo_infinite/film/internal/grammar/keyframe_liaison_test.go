package grammar

import "testing"

// keyframe_liaison_test.go — L IMAGE-CLE OUBLIE CE QU ELLE NE PORTE PLUS (lot D-fix, 2026-09-24).
//
// Le defaut mesure (`a0c36016`) : un slot lie par le NEW d un objet `ti 30` au chunk 27, dont le
// DEL n est pas lu, restait lie jusqu au chunk 39 ; le bipede ne sur ce slot au chunk 38 s y
// decodait sous `ti 30`, et ses etats de mouvement etaient perdus. ROUGE AVANT : la liaison du
// chunk ne savait que poser (chaine, puis table de datums sans ecraser) — la liaison du mort
// survivait a toutes les images-cles qui ne le portaient pas.

// slotMortDeTest : un slot qu AUCUNE image-cle des bobines ne porte (au-dela de toute table
// mesuree), lie comme par le NEW d une entite dont la suppression n a pas ete lue.
const slotMortDeTest = 0x3fff0

func TestLImageCleOublieCeQuElleNePortePlus(t *testing.T) {
	fc := NewFilmContext(bobineFilm(t, "fb1a1a72"))
	reg, err := fc.Registry()
	if err != nil {
		t.Fatal(err)
	}
	marche := fc.MarcheDImageCle()
	lues := 0
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for pi, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			w := NewWorld(reg)
			w.BindFull(slotMortDeTest, 30)
			mp := marche.Marcher(pk.Payload(data))
			ecarte := -1
			if len(mp.Ecartes) > 0 {
				ecarte = mp.Ecartes[0].Slot
				w.BindFull(uint32(ecarte|1<<30), 30) //nolint:gosec // slot borne par le walker
			}
			l := lierLeChunkAuMonde(w, marche, data, pks[pi:pi+1], nil)
			if _, lie := w.ArchetypeForSlot(slotMortDeTest); lie || l.Oubliees < 1 {
				t.Fatalf("c%d p%d : la liaison d un slot que l image-cle ne porte pas survit (oubliees %d)",
					c, pi, l.Oubliees)
			}
			for _, r := range mp.Records {
				//nolint:gosec // slot et TI bornes par le walker
				if ti, ok := w.ArchetypeForSlot(uint32(r.Slot)); !ok || ti != uint32(r.TI) {
					t.Fatalf("c%d p%d slot %d : record de la chaine lie a %d/%v, attendu %d", c, pi,
						r.Slot, ti, ok, r.TI)
				}
			}
			// UN CANDIDAT ECARTE N EST PAS UNE ABSENCE PROUVEE (principe du lot D-fix) : sa liaison
			// reste quand ni la chaine ni la table ne le portent.
			if ecarte >= 0 {
				if _, lie := w.ArchetypeForSlot(uint32(ecarte)); !lie { //nolint:gosec // idem
					t.Fatalf("c%d p%d : le slot %d, candidat ecarte par la marche, a ete oublie", c, pi, ecarte)
				}
			}
			lues++
		}
	}
	if lues == 0 {
		t.Fatal("aucune image-cle sur la bobine")
	}
}

func TestOublierLesSlotsNonPortes(t *testing.T) {
	w := NewWorld(nil)
	w.BindFull(10, 35)
	w.BindDatum(11, 30)
	w.BindFull(12, 7)
	if n := w.OublierLesSlotsNonPortes(map[uint32]bool{10: true, 12: true}); n != 1 {
		t.Fatalf("oubliees %d, attendu 1", n)
	}
	if _, lie := w.ArchetypeForSlot(11); lie {
		t.Fatal("le slot 11, non porte, reste lie")
	}
	for _, s := range []uint32{10, 12} {
		if _, lie := w.ArchetypeForSlot(s); !lie {
			t.Fatalf("le slot %d, porte, a ete oublie", s)
		}
	}
}
