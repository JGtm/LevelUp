package grammar

// ecrivain_invariants_test.go — LES REGLES DE L ECRIVAIN DANS LA DEFINITION DE LA FERMETURE
// (`ecrivain_invariants.go`). Pour chaque regle, un vecteur que l ecrivain peut produire (rien n est
// contredit) et un vecteur qu il ne peut pas produire (la regle est nommee) ; puis la regle de bout
// en bout, sur un paquet ferme au bit pres qui la contredit : il n est plus ferme.

import "testing"

// recordDeTest rend un record de la vue B, masque ecrivable.
func recordDeTest(typ int, slot uint32) FrameRecord {
	return FrameRecord{Type: typ, Slot: slot, DesyncAt: -1}
}

// premiereRegle juge des records et une vue C, comme la marche (premiere regle).
func premiereRegle(recs []FrameRecord, rejet bool, c FluxVueC) InvariantEcrivain {
	return jugerLePaquet(recs, rejet, c, false).premiere
}

// TestOrdreDeLaVueB : `FUN_14076b9c8` concatene NEW, DELTA, DEL ; `FUN_142f2e174` parcourt les
// slots par index croissant.
func TestOrdreDeLaVueB(t *testing.T) {
	n, d, x := recNew, recDelta, recDel
	ecrivables := [][]FrameRecord{
		{recordDeTest(n, 9), recordDeTest(n, 12), recordDeTest(d, 3), recordDeTest(d, 4), recordDeTest(x, 1)},
		{recordDeTest(d, 3)},
		{recordDeTest(n, 700), recordDeTest(x, 2)},
		nil,
	}
	for i, recs := range ecrivables {
		if v := premiereRegle(recs, false, FluxVueC{}); v != InvariantAucun {
			t.Errorf("vecteur ecrivable %d : %s", i, v)
		}
	}
	impossibles := [][]FrameRecord{
		{recordDeTest(d, 3), recordDeTest(n, 5)},  // un NEW apres un DELTA
		{recordDeTest(x, 3), recordDeTest(d, 9)},  // un DELTA apres un DEL
		{recordDeTest(d, 5), recordDeTest(d, 3)},  // slots decroissants
		{recordDeTest(n, 5), recordDeTest(n, 5)},  // un slot deux fois
		{recordDeTest(x, 8), recordDeTest(x, 8)},  // idem pour les DEL
		{recordDeTest(d, 9), recordDeTest(n, 10)}, // un NEW apres un DELTA, slot plus grand
	}
	for i, recs := range impossibles {
		if v := premiereRegle(recs, false, FluxVueC{}); v != InvariantOrdreVueB {
			t.Errorf("vecteur impossible %d : %s, attendu %s", i, v, InvariantOrdreVueB)
		}
	}
}

// masqueEcrit ecrit un masque comme `FUN_142e2da44` : epars (`0`, R(3), R(6)...) ou dense (`1`, R(64)).
func masqueEcrit(dense bool, val uint64, idx ...uint64) []byte {
	var w bitWriter
	if dense {
		w.bit(1)
		w.bits(val, 64)
		return w.buf
	}
	w.bit(0)
	w.bits(uint64(len(idx)), 3)
	for _, i := range idx {
		w.bits(i, 6)
	}
	return w.buf
}

// TestMasqueEcrit : `FUN_142e2da44` ecrit en epars au plus sept composants, index croissants ; en
// dense au-dela. [consumeMask] rend le meme masque que [lireMasque] et consomme les memes bits.
func TestMasqueEcrit(t *testing.T) {
	cas := []struct {
		nom    string
		buf    []byte
		masque uint64
		bits   int
		want   InvariantEcrivain
	}{
		{"epars vide", masqueEcrit(false, 0), 0, 4, InvariantAucun},
		{"epars croissant", masqueEcrit(false, 0, 3, 5, 63), 1<<3 | 1<<5 | 1<<63, 4 + 18, InvariantAucun},
		{"epars de sept", masqueEcrit(false, 0, 0, 1, 2, 3, 4, 5, 6), 0x7f, 4 + 42, InvariantAucun},
		{"dense de huit", masqueEcrit(true, 0xff), 0xff, 65, InvariantAucun},
		{"epars decroissant", masqueEcrit(false, 0, 5, 3), 1<<3 | 1<<5, 4 + 12, InvariantMasqueEparsNonCroissant},
		{"epars repete", masqueEcrit(false, 0, 4, 4), 1 << 4, 4 + 12, InvariantMasqueEparsNonCroissant},
		{"dense de sept", masqueEcrit(true, 0x7f), 0x7f, 65, InvariantMasqueDenseCourt},
		{"dense vide", masqueEcrit(true, 0), 0, 65, InvariantMasqueDenseCourt},
	}
	for _, c := range cas {
		br := LecteurSur(c.buf)
		m, v := lireMasque(br)
		if m != c.masque || v != c.want || br.BitPos() != c.bits {
			t.Errorf("%s : masque %#x regle %s fin %d, attendu %#x %s %d", c.nom, m, v, br.BitPos(), c.masque, c.want, c.bits)
		}
		br2 := LecteurSur(c.buf)
		if m2 := consumeMask(br2); m2 != m || br2.BitPos() != br.BitPos() {
			t.Errorf("%s : consumeMask rend %#x (fin %d), lireMasque %#x (fin %d)", c.nom, m2, br2.BitPos(), m, br.BitPos())
		}
	}
}

// TestMasqueHorsArchetypeALaTraversee : `FUN_142e2da44` ne pose aucun bit au-dela des `n`
// composants du descripteur. Le bit est juge a la traversee, qui connait l archetype.
func TestMasqueHorsArchetypeALaTraversee(t *testing.T) {
	arch := Archetype{Index: 4, Components: []string{"a", "b", "c"}}
	for _, c := range []struct {
		masque uint64
		want   InvariantEcrivain
	}{{0, InvariantAucun}, {1 << 63, InvariantMasqueHorsArchetype}, {1 << 3, InvariantMasqueHorsArchetype}} {
		tr := EntityTrace{DesyncAt: -1, Mask: c.masque}
		traverseComponentLoop(LecteurSur(nil), arch, &tr)
		if tr.MasqueNonEcrit != c.want {
			t.Errorf("masque %#x sur 3 composants : %s, attendu %s", c.masque, tr.MasqueNonEcrit, c.want)
		}
	}
	plein := Archetype{Index: 35, Components: make([]string, 64)}
	tr := EntityTrace{DesyncAt: -1}
	traverseComponentLoop(LecteurSur(nil), plein, &tr)
	if tr.MasqueNonEcrit != InvariantAucun || masqueHorsArchetype(^uint64(0), 64) {
		t.Errorf("archetype de 64 composants : aucun bit n est au-dela")
	}
	recs := []FrameRecord{{Type: recDelta, Slot: 3, DesyncAt: -1, Trace: EntityTrace{MasqueNonEcrit: InvariantMasqueHorsArchetype}}}
	if v := premiereRegle(recs, false, FluxVueC{}); v != InvariantMasqueHorsArchetype {
		t.Errorf("record a masque hors archetype : %s", v)
	}
}

// entreeDeTest rend une entree de controle d index `i`, sans en-tete ni bloc.
func entreeDeTest(i int) EntreeDeControle {
	return EntreeDeControle{Index: i, Champs: champsAbsents()}
}

// TestVueCEcrite : `FUN_142f2c3b0` (32 tampons par joueur, index croissant), `FUN_14076b0e8`
// (kind 0, bit d en-tete a 0), `FUN_1406d5bf4` (jamais le code 63).
func TestVueCEcrite(t *testing.T) {
	bloc := entreeDeTest(4)
	bloc.Bloc, bloc.Champs.Analogique = true, [2]int{0x1f, 0x3e}
	ecrivable := FluxVueC{Porte: true, Kinds: []int{0, 0, 0}, Entrees: []EntreeDeControle{entreeDeTest(0), bloc, entreeDeTest(31)}}
	if v := premiereRegle(nil, false, ecrivable); v != InvariantAucun {
		t.Errorf("vue C ecrivable : %s", v)
	}
	horsBloc := entreeDeTest(1)
	horsBloc.Champs.Analogique = [2]int{63, 63} // sans bloc 0x68, le couple n est pas lu : rien a juger
	if v := premiereRegle(nil, false, FluxVueC{Porte: true, Kinds: []int{0}, Entrees: []EntreeDeControle{horsBloc}}); v != InvariantAucun {
		t.Errorf("code 63 hors bloc : %s", v)
	}
	trop := make([]EntreeDeControle, 33)
	for i := range trop {
		trop[i] = entreeDeTest(i)
	}
	enTete := entreeDeTest(2)
	enTete.Champs.Cdc04 = 5
	code63 := bloc
	code63.Champs.Analogique = [2]int{0x1f, 63}
	for _, c := range []struct {
		nom  string
		flux FluxVueC
		want InvariantEcrivain
	}{
		{"kind 3", FluxVueC{Porte: true, Kinds: []int{0, 3}, Entrees: []EntreeDeControle{entreeDeTest(1)}}, InvariantVueCKind},
		{"33 entrees", FluxVueC{Porte: true, Entrees: trop}, InvariantVueCTropDEntrees},
		{"index decroissants", FluxVueC{Porte: true, Entrees: []EntreeDeControle{entreeDeTest(3), entreeDeTest(1)}}, InvariantVueCIndex},
		{"index repete", FluxVueC{Porte: true, Entrees: []EntreeDeControle{entreeDeTest(3), entreeDeTest(3)}}, InvariantVueCIndex},
		{"en-tete cdc04", FluxVueC{Porte: true, Entrees: []EntreeDeControle{enTete}}, InvariantVueCEnTete},
		{"code analogique 63", FluxVueC{Porte: true, Entrees: []EntreeDeControle{code63}}, InvariantVueCCodeAnalogique},
	} {
		if v := premiereRegle(nil, false, c.flux); v != c.want {
			t.Errorf("%s : %s, attendu %s", c.nom, v, c.want)
		}
	}
}

// TestPremiereRegleEtEnsemble : la marche s arrete a la premiere regle (sortie, records, vue C) ;
// la carte les releve toutes.
func TestPremiereRegleEtEnsemble(t *testing.T) {
	recs := []FrameRecord{recordDeTest(recDelta, 5), recordDeTest(recDelta, 3)}
	flux := FluxVueC{Porte: true, Kinds: []int{3}}
	j := jugerLePaquet(recs, true, flux, true)
	if j.premiere != InvariantSortieParRejet {
		t.Errorf("premiere regle %s, attendu la sortie par rejet", j.premiere)
	}
	want := uint32(1)<<InvariantSortieParRejet | 1<<InvariantOrdreVueB | 1<<InvariantVueCKind
	if j.ensemble != want {
		t.Errorf("ensemble %#x, attendu %#x", j.ensemble, want)
	}
	if j := jugerLePaquet(recs, false, flux, false); j.premiere != InvariantOrdreVueB || j.ensemble != 1<<InvariantOrdreVueB {
		t.Errorf("marche : %s %#x, attendu l ordre seul", j.premiere, j.ensemble)
	}
}

// TestVerdictDeVueC : ferme = ferme au bit pres ET aucune regle contredite ; la sortie par rejet se
// dit meme sur un paquet qui ne ferme pas au bit.
func TestVerdictDeVueC(t *testing.T) {
	pay := []byte{0x00} // la vue C lue jusqu au bit 1 : reste de 7 bits nuls
	flux := FluxVueC{Porte: true, Vide: true}
	if l := verdictDeVueC(pay, 1, flux, []FrameRecord{recordDeTest(recDelta, 3)}, false); !l.Fermee || !l.FermeeAuBit || l.Invariant != InvariantAucun {
		t.Errorf("paquet ecrivable : %+v", l)
	}
	l := verdictDeVueC(pay, 1, flux, []FrameRecord{recordDeTest(recDelta, 5), recordDeTest(recDelta, 3)}, false)
	if l.Fermee || !l.FermeeAuBit || l.Invariant != InvariantOrdreVueB || l.Entrees != nil {
		t.Errorf("ordre contredit : %+v, attendu ferme au bit, non ferme, sans entree", l)
	}
	if l := verdictDeVueC(pay, 1, flux, nil, true); l.Fermee || !l.FermeeAuBit || l.Invariant != InvariantSortieParRejet {
		t.Errorf("sortie par rejet : %+v", l)
	}
	if l := verdictDeVueC([]byte{0xff}, 1, flux, nil, true); l.FermeeAuBit || l.Invariant != InvariantSortieParRejet {
		t.Errorf("sortie par rejet non fermee : %+v", l)
	}
	if l := verdictDeVueC([]byte{0xff}, 1, flux, []FrameRecord{recordDeTest(recDelta, 5), recordDeTest(recDelta, 3)}, false); l.FermeeAuBit || l.Invariant != InvariantAucun {
		t.Errorf("non ferme au bit : les regles hors sortie ne se jugent pas : %+v", l)
	}
}

// TestPaquetFermeAuBitQuiContreditLOrdre : de bout en bout sur la marche de la carte. Un paquet dont
// la vue B lit le DELTA 124 avant le DELTA 123 ferme au bit pres, mais l ecrivain ne l ecrit pas :
// il n est plus ferme, et sa cause est la regle.
func TestPaquetFermeAuBitQuiContreditLOrdre(t *testing.T) {
	var bw bitWriter
	bw.teteDePaquet()
	bw.deltaMasque13(124)
	bw.deltaMasque13(123)
	bw.finDeVueB()
	bw.bit(0) // vue C vide
	d := marcherUnDetaille(t, mondeDeCarte(), bw.buf)
	if d.Fermee || !d.FermeeAuBit || d.Invariant != InvariantOrdreVueB || d.Cause != InvariantOrdreVueB.String() {
		t.Fatalf("detail %+v : attendu ferme au bit, non ferme, cause %q", d, InvariantOrdreVueB)
	}
	var ok bitWriter
	ok.teteDePaquet()
	ok.deltaMasque13(123)
	ok.deltaMasque13(124)
	ok.finDeVueB()
	ok.bit(0)
	if d := marcherUnDetaille(t, mondeDeCarte(), ok.buf); !d.Fermee || d.Invariant != InvariantAucun {
		t.Fatalf("ordre ecrit : %+v, attendu ferme", d)
	}
	rep := marcherUn(t, mondeDeCarte(), nil, bw.buf, ok.buf)
	if rep.PaquetsFermes != 1 || rep.PaquetsFermesAuBit != 2 || rep.Bloquants[InvariantOrdreVueB.String()].Paquets != 1 {
		t.Errorf("carte : fermes %d, fermes au bit %d, bloquants %+v", rep.PaquetsFermes, rep.PaquetsFermesAuBit, rep.Bloquants)
	}
}

// TestPaquetFermeAuBitQuiContreditLeMasque : un DELTA dont le masque annonce le composant d index
// 5 d un archetype qui n en a qu un ferme au bit pres ; l ecrivain ne pose aucun bit au-dela.
func TestPaquetFermeAuBitQuiContreditLeMasque(t *testing.T) {
	w := mondeDeCarte("object-scale-component")
	var bw bitWriter
	bw.teteDePaquet()
	bw.deltaMasque13(123, 5)
	bw.finDeVueB()
	bw.bit(0)
	d := marcherUnDetaille(t, w, bw.buf)
	if d.Fermee || !d.FermeeAuBit || d.Invariant != InvariantMasqueHorsArchetype {
		t.Fatalf("detail %+v : attendu ferme au bit, non ferme, masque au-dela de l archetype", d)
	}
}
