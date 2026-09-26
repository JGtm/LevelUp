package grammar

import "testing"

// frame_closure_test.go — LA CARTE DE FERMETURE DES TRAMES DELTA SUR DES PAQUETS SYNTHETIQUES
// (lot J4.0, `frame_closure.go`). Chaque paquet est ecrit bit a bit dans l ordre du
// frame-processeur : bit de configuration, vue A, vue B (records a index de 13 bits, cf.
// `debut_de_liste_test.go`), vue C, puis le bourrage a zero jusqu a l octet.

// composantSansLecteurJ40 : un nom qu aucun deserialiseur ne porte — la marche doit s y arreter.
const composantSansLecteurJ40 = "composant-j40-sans-lecteur"

// cadreDeCarte est le cadre des paquets synthetiques : index de 13 bits, amorce de production
// (configuration + vue A), grammaire de production.
func cadreDeCarte() FrameConfig {
	c := cadreDeTete
	c.PacketPreambleBits = DefaultPacketPreambleBits
	return c
}

// mondeDeCarte rend un monde ou les slots 123 et 124 sont vivants sous l archetype 4, dont les
// composants sont `comps`.
func mondeDeCarte(comps ...string) *World {
	reg := &Registry{Archetypes: []Archetype{{Index: 0}, {Index: 1}, {Index: 2}, {Index: 3},
		{Index: 4, Components: comps}}}
	w := NewWorld(reg)
	w.BindImageCle(1, 123, 4)
	w.BindImageCle(1, 124, 4)
	return w
}

// teteDePaquet ecrit le bit de configuration et une vue A vide.
func (w *bitWriter) teteDePaquet() {
	w.bit(1)
	w.bit(0)
}

// deltaMasque13 ecrit un record DELTA dont le masque clairseme annonce les composants `idx`.
func (w *bitWriter) deltaMasque13(slot uint32, idx ...uint64) {
	w.bit(1)
	w.bits(uint64(slot), 13)
	w.bits(1, 2)
	w.bit(0) // selecteur de base
	w.bit(0) // masque clairseme
	w.bits(uint64(len(idx)), 3)
	for _, i := range idx {
		w.bits(i, 6)
	}
}

// finDeVueB ecrit le record de type 0 qui clot la vue B.
func (w *bitWriter) finDeVueB() { w.bits(0, 3) }

// vueCUneEntree ecrit une vue C portant une entree de controle, terminateur compris.
func (w *bitWriter) vueCUneEntree() {
	ecrireEntreeComplete(w, entreeCompleteOpts{index: 2, gachette: 0b100000,
		genreCible: genreCibleCategorie1, modeVect: modeVecteurDirection})
	w.bit(0)
}

// marcherUn marche un payload depuis la tete du paquet sous une mesure neuve et rend sa carte.
func marcherUn(t *testing.T, w *World, utiles UsagesProduit, pays ...[]byte) FrameClosureReport {
	t.Helper()
	m := nouvelleMesureDesTrames(w.Reg, utiles, cadreDeCarte())
	for _, pay := range pays {
		m.marcherPaquet(pay, w, DefaultPacketPreambleBits)
	}
	return m.rapport()
}

// TestFrameClosure_PaquetFermeAuBitPres : trois vues lues jusqu a leur terminateur, le curseur
// sur la fin du paquet (reste nul) — le paquet et son record sont FERMES ; le meme paquet suivi
// d un octet ne l est plus, et aucun de ses records ne l est.
func TestFrameClosure_PaquetFermeAuBitPres(t *testing.T) {
	var bw bitWriter
	bw.teteDePaquet()
	bw.deltaMasque13(123)
	bw.finDeVueB()
	bw.vueCUneEntree()
	ferme := append([]byte(nil), bw.buf...)

	r := marcherUn(t, mondeDeCarte(), nil, ferme)
	if r.Paquets != 1 || r.PaquetsFermes != 1 {
		t.Fatalf("paquets %d fermes %d, attendu 1/1", r.Paquets, r.PaquetsFermes)
	}
	for v := VueMessages; v <= VueControle; v++ {
		if s := r.Vues[v]; s.Atteints != 1 || s.Terminees != 1 || s.Fermes != 1 {
			t.Errorf("vue %d : %+v, attendu atteinte, terminee et fermee", v, s)
		}
	}
	if a := r.Archetypes[4]; a.Deltas != 1 || a.DeltasFermes != 1 || a.Blocking != "" {
		t.Errorf("ti=4 : %+v, attendu 1 delta ferme, aucun bloquant", a)
	}
	if r.Utiles.EntreesDeControleFermees != 1 {
		t.Errorf("entrees de controle fermees %d, attendu 1", r.Utiles.EntreesDeControleFermees)
	}

	trop := append(append([]byte(nil), ferme...), 0xff)
	r = marcherUn(t, mondeDeCarte(), nil, trop)
	if r.PaquetsFermes != 0 || r.Vues[VueControle].Fermes != 0 || r.Archetypes[4].DeltasFermes != 0 {
		t.Fatalf("octet de trop : %+v, attendu aucun paquet ni record ferme", r)
	}
	if n := r.Vues[VueControle].Arrets[causeTerminateurHorsCadre]; n != 1 {
		t.Errorf("vue C : arret %q compte %d, attendu 1 (arrets %v)", causeTerminateurHorsCadre, n,
			r.Vues[VueControle].Arrets)
	}
}

// TestFrameClosure_ComposantSansLecteurNommeLeBloquant : un delta dont le masque annonce un
// composant non porte. La vue B ne se termine pas, la vue C n est pas atteinte, et le bloquant
// est `ti=<a> i<idx> <nom>` — pour le record qui bute ET pour le record propre qui le precede
// dans le meme paquet, que ce port fermerait aussi.
func TestFrameClosure_ComposantSansLecteurNommeLeBloquant(t *testing.T) {
	var bw bitWriter
	bw.teteDePaquet()
	bw.deltaMasque13(124)
	bw.deltaMasque13(123, 0)
	bw.bits(0x2a5, 10) // la charge que personne ne sait lire
	r := marcherUn(t, mondeDeCarte(composantSansLecteurJ40), nil, bw.buf)

	const want = "ti=4 i0 " + composantSansLecteurJ40
	b := r.Vues[VueEntites]
	if b.Atteints != 1 || b.Terminees != 0 || b.Arrets[want] != 1 {
		t.Fatalf("vue B : %+v, attendu atteinte, non terminee, arretee par %q", b, want)
	}
	if c := r.Vues[VueControle]; c.Atteints != 0 {
		t.Errorf("vue C atteinte (%+v) derriere une vue B non terminee", c)
	}
	if a := r.Archetypes[4]; a.Deltas != 2 || a.DeltasFermes != 0 || a.Blocking != want {
		t.Errorf("ti=4 : %+v, attendu 2 deltas non fermes, bloquant %q", a, want)
	}
	got := r.Bloquants[want]
	if got.Paquets != 1 || got.TI != 4 || got.Index != 0 || got.Composant != composantSansLecteurJ40 {
		t.Errorf("bloquant %q : %+v, attendu 1 paquet, ti 4, i0, %s", want, got, composantSansLecteurJ40)
	}
}

// TestFrameClosure_ArretDeLaVueDeControleCompteParCause : chaque cause d arret de la vue C
// ([ArretVueC]) est comptee sous son nom, et un terminateur qui ne ferme pas le paquet aussi.
func TestFrameClosure_ArretDeLaVueDeControleCompteParCause(t *testing.T) {
	vueC := map[string]func(*bitWriter){
		nomArretVueC(ArretVueCDebordement): func(w *bitWriter) {
			w.bit(1)
			w.bits(kindVueCControle, 2) // l octet s acheve ici : l entree deborde
		},
		nomArretVueC(ArretVueCKindNonPorte): func(w *bitWriter) {
			w.bit(1)
			w.bits(kindVueCSecond, 2)
		},
		nomArretVueC(ArretVueCBlocBC): func(w *bitWriter) {
			w.bit(1)
			w.bits(kindVueCControle, 2)
			w.bit(0)
			w.bits(3, largeurIndexControle)
			w.bit(0) // pas de bloc de 0x68
			w.bit(1) // mais le bloc de 0xbc
		},
		nomArretVueC(ArretVueCPlafond): func(w *bitWriter) {
			for range plafondToursVueC {
				w.bit(1)
				w.bits(kindVueCNeant, 2)
			}
		},
		causeTerminateurHorsCadre: func(w *bitWriter) {
			w.bit(0)
			w.bits(0xff, 8)
		},
	}
	var pays [][]byte
	for _, ecrire := range vueC {
		var bw bitWriter
		bw.teteDePaquet()
		bw.finDeVueB()
		ecrire(&bw)
		pays = append(pays, bw.buf)
	}
	r := marcherUn(t, mondeDeCarte(), nil, pays...)
	c := r.Vues[VueControle]
	if c.Atteints != len(vueC) || c.Fermes != 0 || c.Terminees != 1 {
		t.Errorf("vue C : %+v, attendu %d atteintes, 1 terminee (hors cadre), 0 fermee", c, len(vueC))
	}
	for cause := range vueC {
		if c.Arrets[cause] != 1 {
			t.Errorf("cause %q comptee %d fois, attendu 1 (arrets %v)", cause, c.Arrets[cause], c.Arrets)
		}
		if r.Bloquants[cause].Paquets != 1 {
			t.Errorf("bloquant %q : %+v, attendu 1 paquet", cause, r.Bloquants[cause])
		}
	}
}

// TestFrameClosure_BloquantLePlusFrequentDepartageParNom : la regle de [KeyframeClosure] — le
// composant qui arrete le PLUS de records, et a egalite le nom le plus petit.
func TestFrameClosure_BloquantLePlusFrequentDepartageParNom(t *testing.T) {
	const zz, aa = "zz-j40-sans-lecteur", "aa-j40-sans-lecteur"
	paquet := func(idx uint64) []byte {
		var bw bitWriter
		bw.teteDePaquet()
		bw.deltaMasque13(123, idx)
		bw.bits(0x155, 10)
		return bw.buf
	}
	surZZ, surAA := paquet(0), paquet(1)
	r := marcherUn(t, mondeDeCarte(zz, aa), nil, surAA, surZZ)
	const labelZZ, labelAA = "ti=4 i0 " + zz, "ti=4 i1 " + aa
	if got := r.Archetypes[4].Blocking; got != labelZZ {
		t.Errorf("egalite 1-1 : bloquant %q, attendu %q (le plus petit nom)", got, labelZZ)
	}
	if got := r.BloquantPrincipal(); got != labelZZ {
		t.Errorf("egalite 1-1 : premier bloquant %q, attendu %q", got, labelZZ)
	}
	r = marcherUn(t, mondeDeCarte(zz, aa), nil, surAA, surZZ, surAA)
	if got := r.Archetypes[4].Blocking; got != labelAA {
		t.Errorf("2-1 : bloquant %q, attendu %q (le plus frequent)", got, labelAA)
	}
	if got := r.BloquantPrincipal(); got != labelAA {
		t.Errorf("2-1 : premier bloquant %q, attendu %q", got, labelAA)
	}
}

// TestFrameClosure_RecordUtileFermeCompteAParte : un record dont le masque annonce un composant a
// usage produit se compte A PART — ferme dans un paquet ferme, en jeu dans un paquet qui ne l est
// pas — et la cle ignore le suffixe `-component`. Un record sans composant utile n y entre pas.
func TestFrameClosure_RecordUtileFermeCompteAParte(t *testing.T) {
	paquet := func(queue uint64, largeur int) []byte {
		var bw bitWriter
		bw.teteDePaquet()
		bw.deltaMasque13(123, 0)
		bw.bits(5, 3) // game-engine-current-state : R(3)
		bw.deltaMasque13(124)
		bw.finDeVueB()
		bw.bit(0) // vue C vide
		bw.bits(queue, largeur)
		return bw.buf
	}
	utiles := UsagesProduit{CleComposant(4, "game-engine-current-state"): true}
	w := mondeDeCarte(compGameEngineCurrentState)
	r := marcherUn(t, w, utiles, paquet(0, 0), paquet(0xff, 8))

	if u := r.Utiles; u.Records != 2 || u.RecordsFermes != 1 {
		t.Fatalf("utiles %+v, attendu 1 ferme sur 2", u)
	}
	a := r.Archetypes[4]
	if a.Deltas != 4 || a.DeltasFermes != 2 || a.Utiles != 2 || a.UtilesFermes != 1 {
		t.Errorf("ti=4 : %+v, attendu 4 deltas (2 fermes) dont 2 utiles (1 ferme)", a)
	}
	if got := r.Bloquants[causeTerminateurHorsCadre].UtilesEnJeu; got != 1 {
		t.Errorf("utiles en jeu derriere %q : %d, attendu 1", causeTerminateurHorsCadre, got)
	}
}
