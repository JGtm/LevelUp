package grammar

// debut_non_prouve_test.go — LA MARCHE QUI PART DU SECOND RANG DE [debutParFermetureRangee] NE
// MODIFIE PAS LE MONDE (`debut_non_prouve.go`) : ses NEW ne lient pas, ses DEL ne delient pas ;
// elle lit et rend les memes records.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// listeNonProuvee ecrit un paquet dont la seule lecture ferme au bit pres en contredisant
// l ordre de la vue B (NEW 300 puis NEW 290) et qui porte un DEL du slot vivant 124.
func listeNonProuvee() []byte {
	var bw bitWriter
	bw.neuf13(300, 2)
	bw.neuf13(290, 2)
	bw.del13(124)
	bw.finDeVueB()
	bw.bit(0) // vue C vide
	return bw.buf
}

// marcher marche `pay` depuis `debut` comme la marche des trames ([DecodeFrameViewsCurseur]).
func marcher(pay []byte, w *World, debut int) []FrameRecord {
	recs, _, _ := DecodeFrameViewsCurseur(pay, w, cadreDeTete, MovementStateViews, debut)
	return recs
}

// TestLaMarcheDuSecondRangNeModifiePasLeMonde : depuis le debut pris au second rang, les deux NEW
// et le DEL sont lus, mais aucun slot n est lie et le slot 124 reste vivant ; la meme marche sans
// l annonce (son temoin) lie les deux NEW et delie 124. MUTATION : retirer
// `w.marquerDebutNonProuve` de [debutParFermetureRangee] (mutation du monde reintroduite), ROUGE.
func TestLaMarcheDuSecondRangNeModifiePasLeMonde(t *testing.T) {
	pay := listeNonProuvee()
	w := mondeDeCarte()
	debut, rang := debutParFermetureRangee(pay, []int{0}, w, cadreDeTete)
	if rang != lecture.DebutParFermetureAuBit || debut != 0 {
		t.Fatalf("debut %d (rang %d), attendu 0 au second rang", debut, rang)
	}
	recs := marcher(pay, w, debut)
	if len(recs) != 3 {
		t.Fatalf("%d records lus, attendu 3 : la marche lit ses records", len(recs))
	}
	for _, s := range []uint32{300, 290} {
		if _, lie := w.ArchetypeForSlot(s); lie {
			t.Errorf("slot %d lie par une marche non prouvee", s)
		}
	}
	if recs[0].Liaison != lecture.LiaisonAucune {
		t.Errorf("liaison du NEW %v, attendu aucune", recs[0].Liaison)
	}
	if ti, lie := w.ArchetypeForSlot(124); !lie || ti != 4 {
		t.Errorf("slot 124 : %d (%v), attendu 4 vivant — un DEL non prouve ne delie pas", ti, lie)
	}

	temoin := mondeDeCarte()
	marcher(pay, temoin, 0)
	if _, lie := temoin.ArchetypeForSlot(300); !lie {
		t.Error("temoin : le NEW 300 n est pas lie hors du second rang")
	}
	if _, lie := temoin.ArchetypeForSlot(124); lie {
		t.Error("temoin : le DEL 124 n est pas applique hors du second rang")
	}
}

// TestLAnnonceNeVautQuePourLaMarcheQuiSuit : l annonce est retiree par la premiere marche, qu elle
// la concerne ou non. Une marche d un autre point de depart la retire sans en profiter, et la
// marche suivante, meme du bon point de depart, lie a nouveau.
func TestLAnnonceNeVautQuePourLaMarcheQuiSuit(t *testing.T) {
	pay := listeNonProuvee()
	w := mondeDeCarte()
	w.marquerDebutNonProuve(pay, 0)
	var autre bitWriter
	autre.neuf13(310, 2)
	autre.finDeVueB()
	marcher(autre.buf, w, 0) // un autre payload : l annonce ne le concerne pas
	if _, lie := w.ArchetypeForSlot(310); !lie {
		t.Error("une marche que l annonce ne designe pas a ete figee")
	}
	marcher(pay, w, 0)
	if _, lie := w.ArchetypeForSlot(300); !lie {
		t.Error("l annonce a survecu a la marche qui l a retiree")
	}
}

// TestLAnnonceDesigneUnBitDuPayload : une annonce posee sur le MEME payload a un autre bit que le
// debut de la marche ne la designe pas (c est le cas de D-LR-3 : un debut que la marche ne prend
// pas tel quel) ; la marche lie et delie comme son temoin. MUTATION : ignorer le bit dans
// [World.prendreDebutNonProuve], ROUGE.
func TestLAnnonceDesigneUnBitDuPayload(t *testing.T) {
	pay := listeNonProuvee()
	w := mondeDeCarte()
	w.marquerDebutNonProuve(pay, 1)
	marcher(pay, w, 0)
	if _, lie := w.ArchetypeForSlot(300); !lie {
		t.Error("une annonce d un autre bit du meme payload a fige la marche : le NEW 300 n est pas lie")
	}
	if _, lie := w.ArchetypeForSlot(124); lie {
		t.Error("une annonce d un autre bit du meme payload a fige la marche : le DEL 124 n est pas applique")
	}
}

// TestUneMarcheNonProuveeCompteLeNeufQuiContreditUnVivant : dans une marche non prouvee, un NEW
// qui contredit une entite vivante (slot 124, archetype 4 de l image-cle, relu en archetype 2) est
// refuse ET compte, comme dans toute marche ; le slot garde son archetype. MUTATION : juger la
// marche non prouvee AVANT la contradiction dans [corpsDeRecordNeuf], ROUGE (le refus n est plus
// compte).
func TestUneMarcheNonProuveeCompteLeNeufQuiContreditUnVivant(t *testing.T) {
	var bw bitWriter
	bw.neuf13(124, 2)
	bw.neuf13(300, 2)
	bw.finDeVueB()
	bw.bit(0) // vue C vide
	w := mondeDeCarte()
	cfg := cadreDeTete
	cfg.Obs = NouvelleObservation()
	w.marquerDebutNonProuve(bw.buf, 0)
	recs, _, _ := DecodeFrameViewsCurseur(bw.buf, w, cfg, MovementStateViews, 0)
	if len(recs) != 2 {
		t.Fatalf("%d records lus, attendu 2", len(recs))
	}
	if cfg.Obs.NeufsContreUnVivant != 1 || len(cfg.Obs.neufsRefuses) != 1 ||
		cfg.Obs.neufsRefuses[0] != (neufRefuse{124, 2, 4}) {
		t.Errorf("refus comptes %d, en attente %+v : attendu 1 et {124 2 4}", cfg.Obs.NeufsContreUnVivant, cfg.Obs.neufsRefuses)
	}
	if ti, lie := w.ArchetypeForSlot(124); !lie || ti != 4 {
		t.Errorf("slot 124 : %d (%v), attendu 4 vivant", ti, lie)
	}
	if _, lie := w.ArchetypeForSlot(300); lie {
		t.Error("slot 300 lie par une marche non prouvee")
	}
}
