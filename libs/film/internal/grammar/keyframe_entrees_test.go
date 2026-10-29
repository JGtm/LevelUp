package grammar

// keyframe_entrees_test.go — LA TABLE D ENTREES D IMAGE-CLE ET LA FERMETURE SOUS RSLOT, sur des payloads
// synthetiques (plan `.ai/archive/V7.5/PLAN_RI_RESIDU_IMAGES_CLES_2026-10-14.md`, R1.2).

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

// tableDEntrees ecrit une table synthetique : le prefixe d un bit, puis une entree par element de
// `slots` (-1 = slot libere, sinon un record vide de ce slot), puis une marge.
func tableDEntrees(slots ...int) []byte {
	var w bitWriter
	w.bit(0)
	for _, s := range slots {
		if s < 0 {
			w.entreeLiberee()
			continue
		}
		w.recordVide(uint64(s), 0) //nolint:gosec // slot de test
	}
	w.bits(0, 7)
	return w.buf
}

// TestLaMarcheDeterministeLitLesSlotsLiberes : `WalkKeyframeRecords` traverse les entrees liberees au
// lieu d y voir la fin de table, et rend les records des slots 0 et 3. MUTATION — l identifiant
// 0xffffffff repris pour une sentinelle de fin : ROUGE.
func TestLaMarcheDeterministeLitLesSlotsLiberes(t *testing.T) {
	recs, stop := WalkKeyframeRecords(tableDEntrees(0, -1, -1, 3), twoEmptyArchReg(), ContexteParDefaut())
	if len(recs) != 2 || recs[0].Slot != 0 || recs[1].Slot != 3 || stop != KeyframeStopEnd {
		t.Fatalf("records %+v, arret %v : attendu les slots 0 et 3, fin normale", recs, stop)
	}
	if recs[1].BitStart != 1+172+2*108 {
		t.Errorf("record du slot 3 au bit %d, attendu %d", recs[1].BitStart, 1+172+2*108)
	}
}

// TestLaMarcheDeterministeExigeLeSlotDeLEntree : un record dont le slot n est pas l index de son
// entree arrete la marche (la table est indexee par slot). MUTATION — accepter tout slot croissant :
// ROUGE.
func TestLaMarcheDeterministeExigeLeSlotDeLEntree(t *testing.T) {
	recs, stop := WalkKeyframeRecords(tableDEntrees(0, -1, 5), twoEmptyArchReg(), ContexteParDefaut())
	if len(recs) != 1 || stop != KeyframeStopSlot {
		t.Fatalf("records %+v, arret %v : attendu un record puis l arret sur le slot 5 a l entree 2", recs, stop)
	}
}

// TestLeChainageNeCompteQueLesRecords : vingt slots liberes entre deux records se traversent sans
// consommer le budget de records intercales (16). MUTATION — compter les entrees liberees dans le
// budget : ROUGE.
func TestLeChainageNeCompteQueLesRecords(t *testing.T) {
	slots := []int{4}
	for range 20 {
		slots = append(slots, -1)
	}
	slots = append(slots, 25)
	pay := tableDEntrees(slots...)
	want := 1 + 172 + 20*108
	res := ChainKeyframeRecords(pay, twoEmptyArchReg(), 1+172, want, 4, ContexteParDefaut())
	if !res.Reached || res.Skipped != 0 || res.SkippedSansArchetype != 20 {
		t.Fatalf("chainage %+v : attendu atteint, 0 record et 20 entrees sans archetype", res)
	}
	// Un record intercale porte le slot de son entree : le slot 9 a l entree du slot 6 arrete le chainage.
	pay = tableDEntrees(4, -1, 9, 10)
	res = ChainKeyframeRecords(pay, twoEmptyArchReg(), 1+172, 1+172+108+172, 4, ContexteParDefaut())
	if res.Reached || res.Stop != KeyframeStopSlot {
		t.Errorf("record du slot 9 a l entree du slot 6 : %+v, attendu l arret sur le slot", res)
	}
}

// TestLeChainageFaibleReconnaitUnSlotLibere : le temoin de chainage des balayages tient devant un slot
// libere comme devant un record, pas au milieu d une entree. MUTATION — ne reconnaitre que l en-tete
// d un record : ROUGE.
func TestLeChainageFaibleReconnaitUnSlotLibere(t *testing.T) {
	pay, want := tableSynthetique(func(w *bitWriter) { w.entreeLiberee() })
	ctx := ContexteParDefaut()
	for _, c := range []struct {
		pos  int
		vrai bool
	}{{172, true}, {want, true}, {173, false}, {172 + 40, false}} {
		if got := finitSurUneEntree(pay, c.pos, ctx); got != c.vrai {
			t.Errorf("bit %d : chaine %v, attendu %v", c.pos, got, c.vrai)
		}
	}
}

// queueDeTable ecrit, apres le prefixe d un bit, le record du slot 0x1ffe - 3, les trois slots liberes de la fin de table, puis
// `bourrage` bits nuls. Rend le payload.
func queueDeTable(bourrage int) []byte {
	var w bitWriter
	w.bit(0)
	w.recordVide(keyframeEntreesDeTable-4, 0)
	w.entreeLiberee()
	w.entreeLiberee()
	w.entreeLiberee()
	w.bits(0, bourrage)
	return w.buf
}

// TestLeDernierRecordFermeSurLaFinDeTable : le dernier record d un paquet ferme quand les entrees qui
// le suivent jusqu a la derniere de la table finissent dans le dernier octet du payload ; un octet de
// plus, ou une entree de moins, et il ne ferme pas. MUTATION — accepter une fin a plus d un octet du
// payload : ROUGE.
func TestLeDernierRecordFermeSurLaFinDeTable(t *testing.T) {
	reg, ctx := twoEmptyArchReg(), ContexteParDefaut()
	b := keyframeBorne{Bit: 1, Want: -1, Slot: keyframeEntreesDeTable - 4, SlotSuivant: -1}
	for _, c := range []struct {
		bourrage int
		ferme    bool
	}{{0, true}, {7, true}, {8, false}} {
		pay := queueDeTable(c.bourrage)
		tr := WalkKeyframeFullState(pay, 1, reg, ctx)
		if got := fermeSurLaFinDeTable(pay, tr, b, reg, ctx); got != c.ferme {
			t.Errorf("bourrage de %d bit(s) (payload de %d bits) : ferme %v, attendu %v", c.bourrage, len(pay)*8, got, c.ferme)
		}
	}
	b.Slot--
	pay := queueDeTable(0)
	if fermeSurLaFinDeTable(pay, WalkKeyframeFullState(pay, 1, reg, ctx), b, reg, ctx) {
		t.Error("slot 0x1ffe - 5 : quatre entrees attendues, trois lues, la queue ne doit pas fermer")
	}
}

// TestUneAncreSuivanteDeSlotInferieurNeFermePas : la traversee finit exactement sur l ancre suivante,
// mais le slot de celle-ci ne depasse pas celui du record (meme slot, ou slot inferieur) — dans une table
// indexee par slot, elle ne peut pas le suivre. MUTATION — retirer la garde `n < 0` : la lecture de -1
// ou -2 entrees n avance pas, et le record fermerait ; ROUGE.
func TestUneAncreSuivanteDeSlotInferieurNeFermePas(t *testing.T) {
	pay, want := tableSynthetique(func(*bitWriter) {})
	for _, s := range []int{5, 4} {
		if fermeSynthetique(t, pay, want, s) {
			t.Errorf("slot suivant %d pour un record de slot 5 : la fermeture ne doit pas tenir", s)
		}
	}
}

// TestUnRecordSuiviDUneAncreNeFermePasSurLaFinDeTable : la fin de table ne juge que le DERNIER record
// d un paquet ; un record que suit une ancre (`Want >= 0`) se juge sur elle, meme quand la queue de la
// table fermerait. MUTATION — retirer la garde `b.Want >= 0` : ROUGE.
func TestUnRecordSuiviDUneAncreNeFermePasSurLaFinDeTable(t *testing.T) {
	reg, ctx := twoEmptyArchReg(), ContexteParDefaut()
	pay := queueDeTable(0)
	tr := WalkKeyframeFullState(pay, 1, reg, ctx)
	b := keyframeBorne{Bit: 1, Want: -1, Slot: keyframeEntreesDeTable - 4, SlotSuivant: -1}
	if !fermeSurLaFinDeTable(pay, tr, b, reg, ctx) {
		t.Fatal("temoin : sans ancre suivante, la queue doit fermer")
	}
	b.Want, b.SlotSuivant = tr.EndBit, keyframeEntreesDeTable-3
	if fermeSurLaFinDeTable(pay, tr, b, reg, ctx) {
		t.Error("record suivi d une ancre : la fin de table ne doit pas le juger")
	}
}
