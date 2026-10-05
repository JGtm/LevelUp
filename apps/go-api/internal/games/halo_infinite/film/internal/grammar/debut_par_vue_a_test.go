package grammar

// debut_par_vue_a_test.go — LA FIN DE LA VUE A FIXE LE DEBUT DE LA VUE B (lot VA, etape V2 ;
// decisions de l utilisateur du 2026-10-04), et la tete de la vue B suit un bit nul.
//
// Vecteurs ecrits d apres l ecrivain du tick (`FUN_142f2c3b0` : vue A, un bit 0, vue B, bout a
// bout) et d apres les lecteurs des messages de la vue A (`vue_a_lecture.go`).

import (
	"os"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// zoomCourt ecrit un message `unit_zoom` (genre 21) aux trois gardes fermees, puis sa charge R(2)
// (`FUN_14080cb98`).
func zoomCourt(w *bitWriter) {
	w.ecrireEnTeteDeMessage(21)
	w.bits(2, 2)
}

// signature123 ecrit la signature stricte au cadre de carte (35 bits) : un DELTA du slot 123,
// generation 1, baseline R(1) = 0, masque epars d un composant (index 0), le composant
// `high-frequency` sur 8 bits.
func (w *bitWriter) signature123() {
	w.bit(1)
	w.bits(uint64(marchSignatureSlot), 13)
	w.bits(1, 2)
	w.bit(0) // baseline : aucune reference
	w.bit(0) // masque epars
	w.bits(1, 3)
	w.bits(0, 6)
	w.bits(0x5a, composantHauteFrequenceBits)
}

// scriptPortantUneSignature ecrit un message Script (genre 15) sans prefixe, dont les bits de
// charge sont un bit nul puis une signature stricte : une signature que la recherche trouve DANS la
// vue A.
func scriptPortantUneSignature(w *bitWriter) {
	w.ecrireEnTeteDeMessage(15)
	w.bits(0, largeurTeteScript)
	w.bits(36, largeurLongueurScript)
	w.bit(0)
	w.signature123()
}

// paquetVueA ecrit le bit de configuration `cfg`, la vue A `vueA`, son terminateur, puis la vue B
// `vueB` ; il rend le payload et E, le bit qui suit le terminateur.
func paquetVueA(cfg uint64, vueA, vueB func(*bitWriter)) ([]byte, int) {
	var w bitWriter
	w.bit(cfg)
	vueA(&w)
	w.bit(0)
	e := w.n
	vueB(&w)
	return w.buf, e
}

// vueBFermee ecrit une vue B d un DELTA du slot 123 sans composant, son terminateur, puis une vue C
// d une entree : la marche qui part de son debut ferme le paquet.
func vueBFermee(w *bitWriter) {
	w.deltaMasque13(uint32(marchSignatureSlot))
	w.finDeVueB()
	w.vueCUneEntree()
}

// grammaireDeScript rend la grammaire d un film de classe `classe` (cardinal `genres`) dont le
// Script se lit sans prefixe.
func grammaireDeScript(classe classeDeLaVueA, genres int) grammaireDeLaVueA {
	g := grammaireDeTest(classe, genres)
	g.script = scriptSansPrefixe
	return g
}

// debutDuPaquet rend le debut de la vue B de la cuisson et celui des marches sous la grammaire `g`.
func debutDuPaquet(pay []byte, w *World, g grammaireDeLaVueA) (int, lecture.DebutDeVueB, int) {
	cfg := cadreDeCarte()
	a := lireLaVueA(pay, 1, cfg.Profil, g)
	d, comment := debutDeLaVueBDeCuisson(pay, &a, g.classe, w, cfg)
	m, _ := DebutDeLaVueB(pay, w, cfg, VueADuFilm{g: g})
	return d, comment, m
}

// TestLaFinDeLaVueADUnFilmRecentEstLeDebutDeLaVueB : la vue A porte un Script dont la charge
// contient une signature stricte du slot 123, precedee d un bit nul ; la recherche la trouve, AVANT
// la fin de la vue A (E > S). Sous un film a table EGALE, la cuisson et les marches partent de E : la
// signature est dans un message lu. MUTATION — E ignore : ROUGE.
func TestLaFinDeLaVueADUnFilmRecentEstLeDebutDeLaVueB(t *testing.T) {
	pay, e := paquetVueA(1, scriptPortantUneSignature, vueBFermee)
	w := mondeDeCarte(compHighFrequency)
	cfg := cadreDeCarte()
	s := marchLocateStrict(pay, w, cfg)
	if s < 0 || s >= e {
		t.Fatalf("signature %d, fin de la vue A %d : le vecteur doit porter une signature dans la vue A", s, e)
	}
	d, comment, m := debutDuPaquet(pay, w, grammaireDeScript(vueAEgale, GenresVueA))
	if d != e || comment != lecture.DebutParVueA || m != e {
		t.Errorf("cuisson (%d, %d), marches %d ; attendu %d par la vue A", d, comment, m, e)
	}
	if l := lectureDEssai(pay, w, cfg, e); !l.Fermee {
		t.Errorf("la marche depuis E ne ferme pas le paquet : %+v", l)
	}
}

// TestLaFinDeLaVueAPrimeQuandLaMarcheButeEnsuite : la vue B qui part de E commence par un DELTA du
// slot 124 dont le masque annonce un composant hors de l archetype (la marche s y arrete), puis, plus
// loin et derriere un bit nul, une signature stricte (E < S). Film a table EGALE : E, sans reprise a
// la signature — la reprise serait une convention. Film a table PREFIXE : la marche depuis E ne ferme
// pas, le paquet suit le localisateur a l identique. MUTATIONS — signature reprise apres l arret de la
// marche (film EGALE) ; preuve de fermeture retiree (film PREFIXE) : ROUGES.
func TestLaFinDeLaVueAPrimeQuandLaMarcheButeEnsuite(t *testing.T) {
	pay, e := paquetVueA(1, zoomCourt, func(w *bitWriter) {
		w.deltaMasque13(124, 5)
		w.bit(0)
		w.signature123()
		w.finDeVueB()
		w.vueCUneEntree()
	})
	w := mondeDeCarte(compHighFrequency)
	cfg := cadreDeCarte()
	s, commentS := localiserLaListe(pay, w, cfg)
	if s <= e || commentS != lecture.DebutParSignature {
		t.Fatalf("localisateur (%d, %d), fin de la vue A %d : le vecteur doit porter une signature apres E", s, commentS, e)
	}
	if l := lectureDEssai(pay, w, cfg, e); l.Fermee {
		t.Fatalf("la marche depuis E ferme le paquet : le vecteur doit la faire buter")
	}
	if d, comment, m := debutDuPaquet(pay, w, grammaireDeScript(vueAEgale, GenresVueA)); d != e ||
		comment != lecture.DebutParVueA || m != e {
		t.Errorf("film recent : cuisson (%d, %d), marches %d ; attendu %d par la vue A", d, comment, m, e)
	}
	mS, _ := LocaliserBoucleDeRecords(pay, w, cfg, SignaturePuisLargeurLibre)
	if d, comment, m := debutDuPaquet(pay, w, grammaireDeScript(vueAPrefixe, 121)); d != s ||
		comment != commentS || m != mS {
		t.Errorf("film ancien non prouve : cuisson (%d, %d), marches %d ; attendu le localisateur (%d, %d), %d",
			d, comment, m, s, commentS, mS)
	}
}

// TestUnFilmAncienPrendLaFinDeSaVueAQuandElleFermeLePaquet : sous un film a table PREFIXE, la fin de
// la vue A est le debut de la vue B quand la marche qui en part ferme le paquet sans regle de
// l ecrivain contredite.
func TestUnFilmAncienPrendLaFinDeSaVueAQuandElleFermeLePaquet(t *testing.T) {
	pay, e := paquetVueA(1, scriptPortantUneSignature, vueBFermee)
	w := mondeDeCarte(compHighFrequency)
	if d, comment, m := debutDuPaquet(pay, w, grammaireDeScript(vueAPrefixe, 121)); d != e ||
		comment != lecture.DebutParVueA || m != e {
		t.Errorf("cuisson (%d, %d), marches %d ; attendu %d par la vue A, prouvee", d, comment, m, e)
	}
}

// TestUneVueALueEnPartieNeDecideRien : une vue A que la lecture n atteint pas jusqu a son
// terminateur — bit de configuration a 0, film sans table, genre 85 dont la charge n est pas
// portee — ne decide de rien : le paquet suit le localisateur a l identique, ici la signature
// trouvee dans la vue A. MUTATIONS — E pris malgre le bit de configuration a 0 ; E pris sur une
// lecture arretee : ROUGES.
func TestUneVueALueEnPartieNeDecideRien(t *testing.T) {
	cfg := cadreDeCarte()
	pay0, _ := paquetVueA(0, scriptPortantUneSignature, vueBFermee)
	pay85, _ := paquetVueA(1, func(w *bitWriter) {
		scriptPortantUneSignature(w)
		w.ecrireEnTeteDeMessage(85)
	}, vueBFermee)
	pay1, _ := paquetVueA(1, scriptPortantUneSignature, vueBFermee)
	for _, c := range []struct {
		nom string
		pay []byte
		g   grammaireDeLaVueA
	}{
		{"bit de configuration a 0", pay0, grammaireDeScript(vueAEgale, GenresVueA)},
		{"genre 85 non porte", pay85, grammaireDeScript(vueAEgale, GenresVueA)},
		{"film sans table", pay1, grammaireDeLaVueA{}},
	} {
		w := mondeDeCarte(compHighFrequency)
		s, commentS := localiserLaListe(c.pay, w, cfg)
		mS, _ := LocaliserBoucleDeRecords(c.pay, w, cfg, SignaturePuisLargeurLibre)
		if s < 0 || commentS != lecture.DebutParSignature {
			t.Fatalf("%s : localisateur (%d, %d), attendu la signature de la vue A", c.nom, s, commentS)
		}
		if d, comment, m := debutDuPaquet(c.pay, w, c.g); d != s || comment != commentS || m != mS {
			t.Errorf("%s : cuisson (%d, %d), marches %d ; attendu le localisateur (%d, %d), %d", c.nom, d,
				comment, m, s, commentS, mS)
		}
	}
}

// TestUnVraiPaquetPartDeLaFinDeSaVueA : la trame 1:204 de `bcb6d393` (HI_1_12_0, table EGALE),
// quarante Script : la vue B commence au bit 5605, qui suit le terminateur.
func TestUnVraiPaquetPartDeLaFinDeSaVueA(t *testing.T) {
	pay, err := os.ReadFile("testdata/vue_a_bcb6d393_1_204.bin")
	if err != nil {
		t.Fatal(err)
	}
	id, err := ReadFilmIdentity(bobineChunk00(t, "bcb6d393"))
	if err != nil {
		t.Fatal(err)
	}
	g := grammaireDeScript(classeDesGenres(id.TypeVersions))
	if id.SimulationDeLEnregistreur != simulationDistClient {
		g.script = scriptAvecPrefixe
	}
	cfg := cadreDeCarte()
	cfg.Profil.Grammaire.ControleDeCorruption = id.ControleDeCorruption
	a := lireLaVueA(pay, 1, cfg.Profil, g)
	if g.classe != vueAEgale {
		t.Fatalf("classe %d, attendu EGALE", g.classe)
	}
	if d, comment := debutDeLaVueBDeCuisson(pay, &a, g.classe, mondeDeCarte(), cfg); d != 5605 ||
		comment != lecture.DebutParVueA {
		t.Errorf("debut (%d, %d), attendu (5605, par la vue A)", d, comment)
	}
}

// TestLaTeteDeLaVueBSuitUnBitNul : le jeu n ecrit un bit nul que devant le PREMIER record de la vue
// B (le terminateur de la vue A). Un candidat NEW de tete precede d un bit a 1 n est la tete ni
// par la chaine ni par la fermeture. MUTATION — le bit nul retire des candidats : ROUGE.
func TestLaTeteDeLaVueBSuitUnBitNul(t *testing.T) {
	var bw bitWriter
	bw.bits(0x1f, 5) // la fin d un message de la vue A, SANS terminateur
	neuf := bw.n
	bw.neuf13(300, 2)
	bw.delta13(122)
	debut := bw.n
	bw.delta13(123)
	bw.bit(0)
	bw.bits(0, 2)
	w := mondeDeTete()
	cands := candidatsDeTete(bw.buf, debut, w)
	if len(cands) != 1 || cands[0] != neuf {
		t.Fatalf("candidats %v, attendu [%d]", cands, neuf)
	}
	if got, ok := debutParChaine(bw.buf, debut, cands, w, cadreDeTete); ok || got != debut {
		t.Errorf("chaine : debut %d (%v), attendu %d garde — un bit a 1 precede le NEW", got, ok, debut)
	}
	pay, faux, _ := deuxDebuts()
	pay[0] |= 0x80 // le bit qui precede `faux` passe a 1
	if got, rang := debutParFermetureRangee(pay, []int{faux}, mondeDeCarte(), cadreDeTete); rang != lecture.DebutNonLocalise ||
		got != -1 {
		t.Errorf("fermeture : debut %d (rang %d), attendu -1 non localise — un bit a 1 precede le candidat", got, rang)
	}
}

// TestLaMarcheDesMortsPartDeLaFinDeLaVueA : sur la bobine reelle a paquets delta, la marche des morts
// d objet ([ScanMarchFacts]) localise exactement les paquets que [DebutDeLaVueB] localise sous la
// grammaire de vue A du film, et plus que le localisateur seul : la fin de la vue A y decide.
// MUTATION — la marche des morts sans la grammaire de vue A du film : ROUGE.
func TestLaMarcheDesMortsPartDeLaFinDeLaVueA(t *testing.T) {
	film, err := source.LoadDir(bobineMarcheDir(), nil)
	if err != nil {
		t.Fatalf("bobine illisible : %v", err)
	}
	fc := NewFilmContext(film)
	f, err := ScanMarchFacts(fc)
	if err != nil {
		t.Fatalf("ScanMarchFacts : %v", err)
	}
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	kfs, deltas := marchPacketsOf(fc)
	cfg, _, _, _ := calibrateFrameConfig(reg, kfs, deltas, fc.CadreDeBalayage())
	tl := newMarchTimeline(reg, kfs)
	avecE, sansE := 0, 0
	for _, d := range deltas {
		w := tl.advanceTo(d.timestampUS)
		if _, ev, ok, _ := marchDebut(d.payload, w, cfg, VueADuFilm{g: fc.grammaireDeLaVueA()}); ev && ok {
			avecE++
		}
		if _, ev, ok, _ := marchDebut(d.payload, w, cfg, VueADuFilm{}); ev && ok {
			sansE++
		}
	}
	if f.Stats.LocatedPackets != avecE || avecE <= sansE {
		t.Errorf("paquets localises : marche %d, sous la vue A du film %d, localisateur seul %d", f.Stats.LocatedPackets,
			avecE, sansE)
	}
}
