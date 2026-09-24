package grammar

// keyframe_world_preuve_test.go — L'ELECTION NE CONTREDIT PLUS UN RECORD PROUVE (lot D-fix,
// 2026-09-24) : ce que keyframe_world_preuve.go doit tenir.
//
//	PREUVE-BOBINE  Sur les trois images-cles des bobines ou l'election perdait le joueur gere de
//	               l'index 0 (`bcb6d393` morceau 1 paquet 0 ; `fb1a1a72` morceau 1 paquets 0 et
//	               1), la marche SANS preuve elit la fausse ancre 192 et perd 1280..1298 (temoin :
//	               le test mord), la marche DU FILM les garde tous, n'a plus 192, et compte UNE
//	               refutation. Le repli n'y ecarte plus aucun candidat ti=9.
//	PREUVE-ELIRE   L'election refutable, sur des candidats et des preuves poses a la main : l'elu
//	               contredit par un record prouve est refuse ; un elu que rien de prouve ne
//	               contredit reste ; quand tout est refute, la recherche garde l'election d'avant.
//	PREUVE-VIDE    Un record VIDE (`n1 <= 0`) ne prouve rien, meme quand sa marche atterrit sur un
//	               en-tete valide (la lecture decalee d'une chaine de records vides en est une).

import "testing"

// preuveBobine est une image-cle ou l'election d'avant le lot perdait des records.
type preuveBobine struct {
	film           string
	chunk, paquet  int
	premier, apres int // les records perdus : premier..apres inclus
}

func preuvesBobines() []preuveBobine {
	return []preuveBobine{
		{"bcb6d393", 1, 0, 1280, 1298},
		{"fb1a1a72", 1, 0, 1280, 1298},
		{"fb1a1a72", 1, 1, 1280, 1298},
	}
}

// slotsDesRecords rend l'ensemble des slots d'une marche, et l'archetype de chacun.
func slotsDesRecords(recs []KeyframeRec) map[int]int {
	out := make(map[int]int, len(recs))
	for _, r := range recs {
		out[r.Slot] = r.TI
	}
	return out
}

// TestMarcheDuFilmGardeLesRecordsDAvantMatch execute PREUVE-BOBINE.
func TestMarcheDuFilmGardeLesRecordsDAvantMatch(t *testing.T) {
	for _, c := range preuvesBobines() {
		fc := NewFilmContext(bobineFilm(t, c.film))
		raw, pks, ok := fc.ChunkAt(c.chunk)
		if !ok || c.paquet >= len(pks) || pks[c.paquet].Type != PacketTypeKeyframe {
			t.Fatalf("%s morceau %d paquet %d : image-cle introuvable", c.film, c.chunk, c.paquet)
		}
		pay := pks[c.paquet].Payload(raw)
		sans := slotsDesRecords(WalkKeyframeWorld(pay))
		// LE TEMOIN : sans preuve, la fausse ancre est elue et le joueur gere de l'index 0 perdu.
		if _, elue := sans[192]; !elue || sans[1297] == managedPlayerTypeIndex {
			t.Fatalf("%s m%d p%d : la marche sans preuve n'elit plus 192 ou garde 1297 — le temoin "+
				"ne mord plus, la bobine a change", c.film, c.chunk, c.paquet)
		}
		mp := fc.MarcheDImageCle().Marcher(pay)
		avec := slotsDesRecords(mp.Records)
		if _, elue := avec[192]; elue {
			t.Errorf("%s m%d p%d : la fausse ancre 192 est encore elue", c.film, c.chunk, c.paquet)
		}
		for s := c.premier; s <= c.apres; s++ {
			if _, lu := avec[s]; !lu {
				t.Errorf("%s m%d p%d : record %d perdu par la marche du film", c.film, c.chunk, c.paquet, s)
			}
		}
		if avec[1297] != managedPlayerTypeIndex {
			t.Errorf("%s m%d p%d : le joueur gere de l'index 0 (slot 1297) n'est pas lu ti=9", c.film,
				c.chunk, c.paquet)
		}
		if mp.Stats.Refutations != 1 {
			t.Errorf("%s m%d p%d : %d refutation(s), attendu 1", c.film, c.chunk, c.paquet, mp.Stats.Refutations)
		}
		for _, e := range mp.Ecartes {
			if e.TI == managedPlayerTypeIndex {
				t.Errorf("%s m%d p%d : le repli ecarte encore un candidat ti=9 (slot %d)", c.film,
					c.chunk, c.paquet, e.Slot)
			}
		}
	}
}

// candidatsPoses construit une recherche dont les candidats et les preuves sont poses a la main :
// la preuve n'est jamais recalculee (payload vide), seule la memoire repond.
func candidatsPoses(cands []kfCandidat, prouves map[int]bool) *kfRecherche {
	return &kfRecherche{cands: cands, prouves: prouves, preuve: &PreuveDImageCle{}}
}

// TestElectionRefutable execute PREUVE-ELIRE.
func TestElectionRefutable(t *testing.T) {
	vrai := kfCandidat{gen: 1, slot: 1280, bit: 100, ti: 38}
	faux := kfCandidat{gen: 1, slot: 192, bit: 200, ti: 1}
	cas := []struct {
		nom     string
		cands   []kfCandidat
		prouves map[int]bool
		elu     int
		refutes int
	}{
		{"l elu contredit par un record prouve est refuse", []kfCandidat{vrai, faux},
			map[int]bool{100: true}, 100, 1},
		{"sans preuve, l election d avant", []kfCandidat{vrai, faux}, map[int]bool{}, 200, 0},
		{"un elu coherent avec le record prouve reste",
			[]kfCandidat{{gen: 1, slot: 300, bit: 100}, {gen: 1, slot: 400, bit: 200}},
			map[int]bool{200: true}, 100, 0},
		{"tout refute : aucun elu", []kfCandidat{{gen: 1, slot: 500, bit: 100}, {gen: 1, slot: 400, bit: 200}},
			map[int]bool{100: true, 200: true}, -1, 2},
	}
	for _, c := range cas {
		elu, refutes := candidatsPoses(c.cands, c.prouves).elire(122)
		if elu != c.elu || refutes != c.refutes {
			t.Errorf("%s : elu %d refutes %d, attendu %d et %d", c.nom, elu, refutes, c.elu, c.refutes)
		}
	}
}

// TestContradictionDOrdre : deux candidats ne peuvent etre deux records que si l'ordre de leurs
// bits est celui de leurs slots.
func TestContradictionDOrdre(t *testing.T) {
	a := kfCandidat{slot: 10, bit: 100}
	for _, c := range []struct {
		b    kfCandidat
		want bool
	}{
		{kfCandidat{slot: 11, bit: 200}, false},
		{kfCandidat{slot: 9, bit: 200}, true},
		{kfCandidat{slot: 10, bit: 200}, true},
		{kfCandidat{slot: 11, bit: 50}, true},
		{kfCandidat{slot: 9, bit: 50}, false},
	} {
		if got := a.contredit(c.b); got != c.want || c.b.contredit(a) != c.want {
			t.Errorf("slot %d bit %d contre slot %d bit %d : %v, attendu %v (et symetrique)", a.slot, a.bit,
				c.b.slot, c.b.bit, got, c.want)
		}
	}
}

// TestPreuveRefuseUnRecordVide execute PREUVE-VIDE : deux en-tetes consecutifs de records sans
// etat par defaut ni composants (`n1 = n2 = 0`), le second exactement la ou la marche du premier
// s'arrete — la fermeture a lieu, et la preuve la refuse quand meme.
func TestPreuveRefuseUnRecordVide(t *testing.T) {
	fc := NewFilmContext(bobineFilm(t, "bcb6d393"))
	p := fc.PreuveDImageCle()
	if p == nil {
		t.Fatal("bobine sans preuve : registre illisible ?")
	}
	cadre := p.ctx.Profil.Cadre
	largeur := cadre.EnTeteBits + 2*cadre.MotDeTailleBits
	pay := make([]byte, (1+2*largeur+64)/8+8)
	ecrire := func(bit int, v uint64, n int) {
		for i := 0; i < n; i++ {
			if v>>(uint(n-1-i))&1 == 1 {
				pay[(bit+i)>>3] |= 0x80 >> uint((bit+i)&7)
			}
		}
	}
	enTete := func(bit, slot int) {
		ecrire(bit, uint64(1<<30|slot), 32)
		ecrire(bit+32, 41, 32) // un archetype sous le cap objet
	}
	enTete(1, 10)
	enTete(1+largeur, 11)
	tr := WalkKeyframeFullState(pay, 1, p.reg, p.ctx)
	if tr.EndBit != 1+largeur {
		t.Fatalf("la marche du record vide finit a %d, attendu %d : le temoin ne ferme plus", tr.EndBit, 1+largeur)
	}
	if p.prouve(pay, 1, 10) {
		t.Fatal("un record vide qui ferme sur un en-tete valide est dit PROUVE")
	}
}

// TestLesPreuvesContradictoiresSeComptent (constat DFIX-R8) : quand TOUS les candidats d une
// fenetre sont contredits par un record prouve (deux preuves se contredisent), la recherche garde
// l elu d avant la preuve — et le COMPTE, au lieu de retomber en silence. ROUGE AVANT : aucun
// compteur, `refutes` jete.
func TestLesPreuvesContradictoiresSeComptent(t *testing.T) {
	w := &bitWriter{}
	// archetype 40 : lu decale d un bit, le mot d archetype vaut 80 ou 81 (>= 50) — aucune ancre
	// parasite ne s ajoute aux deux candidats.
	a := w.n
	kfEcrireRecord(w, 1, 20, 40, 50)
	b := w.n
	kfEcrireRecord(w, 1, 15, 40, 50)
	for i := 0; i < 2100; i++ {
		w.bits(kfSent, 32)
	}
	r := &kfRecherche{buf: w.buf, total: len(w.buf) * 8, maxWin: kfScanFenetreBits,
		preuve: &PreuveDImageCle{}, prouves: map[int]bool{a: true, b: true}}
	iss := r.suivante(0, 10)
	if !iss.contradictoire || iss.at != b || iss.dec != kfElection || iss.refutes != 0 {
		t.Fatalf("issue %+v : attendu contradictoire, l elu d avant la preuve (bit %d), election, 0 refute", iss, b)
	}
	var st KeyframeWalkStats
	st.compter(iss)
	st.Ajouter(st)
	if st.PreuvesContradictoires != 2 || st.Refutations != 0 {
		t.Fatalf("stats %+v : attendu 2 preuves contradictoires (compte puis cumul), 0 refutation", st)
	}
}
