package grammar

// keyframe_entrees_test.go — LA TABLE D ENTREES D IMAGE-CLE ET LA FERMETURE SOUS RSLOT, sur des payloads
// synthetiques (plan `.ai/PLAN_RI_RESIDU_IMAGES_CLES_2026-10-14.md`, R1.2).

import "testing"

// entreeLiberee ecrit la forme d un slot libere chez l ecrivain (`FUN_1408f1948`) : identifiant,
// archetype et mot +0xc a 0xffffffff, R(4) et R(8) nuls — 108 bits.
func (w *bitWriter) entreeLiberee() {
	w.bits(0xffffffff, 32)
	w.bits(0xffffffff, 32)
	w.bits(0xffffffff, 32)
	w.bits(0, 4)
	w.bits(0, 8)
}

// recordVide ecrit un record d archetype `ti`, generation 1, sans etat par defaut ni composant (n1 = n2
// = 0) : 108 + 32 + 32 = 172 bits.
func (w *bitWriter) recordVide(slot, ti uint64) {
	w.bits(1<<30|slot, 32)
	w.bits(ti, 32)
	w.bits(0, 32)
	w.bits(0, 4)
	w.bits(0, 8)
	w.bits(0, 32) // n1
	w.bits(0, 32) // n2
}

// tableSynthetique : record de slot 5 au bit 0, entrees intercalees `milieu`, puis record de slot 9.
// Rend le payload et le bit du record de slot 9.
func tableSynthetique(milieu func(w *bitWriter)) ([]byte, int) {
	var w bitWriter
	w.recordVide(5, 0)
	milieu(&w)
	want := w.n
	w.recordVide(9, 1)
	w.bits(0, 64) // marge : la lecture d un en-tete ne deborde pas
	return w.buf, want
}

func fermeSynthetique(t *testing.T, pay []byte, want, slotSuivant int) bool {
	t.Helper()
	reg, ctx := twoEmptyArchReg(), ContexteParDefaut()
	tr := WalkKeyframeFullState(pay, 0, reg, ctx)
	if tr.DesyncAt >= 0 || tr.EndBit != 172 {
		t.Fatalf("record de tete mal lu : desync %d, fin %d", tr.DesyncAt, tr.EndBit)
	}
	b := keyframeBorne{Bit: 0, Want: want, Slot: 5, SlotSuivant: slotSuivant, TI: 0}
	return fermeSurLAncreSuivante(pay, tr, b, reg, ctx)
}

// TestLesEntreesLibereesSeLisentEntreDeuxAncres : entre les slots 5 et 9, trois entrees liberees de
// 108 bits ; la fermeture les lit et atterrit sur l ancre. Avec un ecart de slots qui ne correspond
// pas au nombre d entrees, elle ne ferme pas.
func TestLesEntreesLibereesSeLisentEntreDeuxAncres(t *testing.T) {
	pay, want := tableSynthetique(func(w *bitWriter) {
		w.entreeLiberee()
		w.entreeLiberee()
		w.entreeLiberee()
	})
	if want != 172+3*108 {
		t.Fatalf("payload synthetique : ancre au bit %d", want)
	}
	if !fermeSynthetique(t, pay, want, 9) {
		t.Error("trois entrees liberees entre les slots 5 et 9 : la fermeture doit les lire")
	}
	for _, s := range []int{8, 10} {
		if fermeSynthetique(t, pay, want, s) {
			t.Errorf("slot suivant %d : %d entrees attendues, 3 lues, la fermeture ne doit pas tenir", s, s-5-1)
		}
	}
}

// TestUnRecordIntercaleSeLitCommeLeJeu : une entree a archetype entre deux entrees liberees se lit par
// son etat complet ; la fermeture tient sur les trois entrees.
func TestUnRecordIntercaleSeLitCommeLeJeu(t *testing.T) {
	pay, want := tableSynthetique(func(w *bitWriter) {
		w.entreeLiberee()
		w.recordVide(7, 1)
		w.entreeLiberee()
	})
	if !fermeSynthetique(t, pay, want, 9) {
		t.Error("libre, record de 172 bits, libre : la fermeture doit les lire")
	}
}

// TestSansEntreeIntercaleeLEcartDeSlotsEstNul : un record qui finit sur l ancre suivante ne ferme que si
// les deux slots se suivent — dans une table indexee par slot, chaque slot intermediaire a son entree.
func TestSansEntreeIntercaleeLEcartDeSlotsEstNul(t *testing.T) {
	pay, want := tableSynthetique(func(*bitWriter) {})
	if !fermeSynthetique(t, pay, want, 6) {
		t.Error("slots 5 et 6 contigus : la fermeture doit tenir")
	}
	if fermeSynthetique(t, pay, want, 9) {
		t.Error("slots 5 et 9 sans entree intercalee : impossible dans une table indexee par slot")
	}
}

// TestUneEntreeIllisibleArreteLaLecture : un mot d archetype hors de la table du jeu (ni 0xffffffff,
// ni < 50) n est pas une frontiere d entree.
func TestUneEntreeIllisibleArreteLaLecture(t *testing.T) {
	pay, want := tableSynthetique(func(w *bitWriter) {
		w.bits(0xffffffff, 32)
		w.bits(60, 32)
		w.bits(0, 44)
	})
	if fermeSynthetique(t, pay, want, 7) {
		t.Error("archetype 60 : l entree est illisible, la fermeture ne doit pas tenir")
	}
	total := len(pay) * 8
	if g := lireEntreeDImageCle(pay, 172, total, 108); g != entreeIllisible {
		t.Errorf("archetype 60 : genre %d, attendu illisible", g)
	}
	if g := lireEntreeDImageCle(pay, total-100, total, 108); g != entreeIllisible {
		t.Errorf("entree qui deborde du payload : genre %d, attendu illisible", g)
	}
}

// TestLaFermetureNeSeLitPasAuDelaDeLAncre : une traversee trop proche de l ancre pour y loger les
// entrees attendues ne ferme pas, quel que soit le contenu.
func TestLaFermetureNeSeLitPasAuDelaDeLAncre(t *testing.T) {
	pay, want := tableSynthetique(func(w *bitWriter) { w.entreeLiberee() })
	if fermeSynthetique(t, pay, want, 8) {
		t.Error("deux entrees attendues dans 108 bits : la fermeture ne doit pas tenir")
	}
	if !fermeSynthetique(t, pay, want, 7) {
		t.Error("une entree liberee entre les slots 5 et 7 : la fermeture doit tenir")
	}
}
