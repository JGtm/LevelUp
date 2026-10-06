package grammar

// debut_par_vue_a_test.go — LA FIN DE LA VUE A FIXE LE DEBUT DE LA VUE B (lot VA, etape V2 ;
// decisions de l utilisateur du 2026-10-04).
//
// Vecteurs ecrits d apres l ecrivain du tick (`FUN_142f2c3b0` : vue A, un bit 0, vue B, bout a
// bout) et d apres les lecteurs des messages de la vue A (`vue_a_lecture.go`).

import (
	"os"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
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

// debutDuPaquet rend le debut de la vue B de la cuisson et celui de la marche de killsource sous la
// grammaire `g`. Quand la vue A decide, la marche de killsource ne compte aucun repli a largeur
// libre ; sinon elle rend le verdict du localisateur, a l identique (le compte du repli
// `repli_localisation_largeur_libre` est la preuve de la verification (d) de la representation
// intermediaire).
func debutDuPaquet(t *testing.T, pay []byte, w *World, g grammaireDeLaVueA) (int, lecture.DebutDeVueB, int) {
	t.Helper()
	cfg := cadreDeCarte()
	a := lireLaVueA(pay, 1, cfg.Profil, g)
	d, comment := debutDeLaVueBDeCuisson(pay, &a, g.classe, w, cfg)
	m, libre := DebutDeLaVueB(pay, w, cfg, VueADuFilm{g: g})
	if debutParLaVueA(pay, &a, g.classe, w, cfg) >= 0 {
		if libre {
			t.Errorf("debut %d par la vue A compte comme un repli a largeur libre", m)
		}
	} else if _, libreS := LocaliserBoucleDeRecords(pay, w, cfg, SignaturePuisLargeurLibre); libre != libreS {
		t.Errorf("repli a largeur libre %v, le localisateur rend %v", libre, libreS)
	}
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
	d, comment, m := debutDuPaquet(t, pay, w, grammaireDeScript(vueAEgale, GenresVueA))
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
	if d, comment, m := debutDuPaquet(t, pay, w, grammaireDeScript(vueAEgale, GenresVueA)); d != e ||
		comment != lecture.DebutParVueA || m != e {
		t.Errorf("film recent : cuisson (%d, %d), marches %d ; attendu %d par la vue A", d, comment, m, e)
	}
	mS, _ := LocaliserBoucleDeRecords(pay, w, cfg, SignaturePuisLargeurLibre)
	if d, comment, m := debutDuPaquet(t, pay, w, grammaireDeScript(vueAPrefixe, 121)); d != s ||
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
	if d, comment, m := debutDuPaquet(t, pay, w, grammaireDeScript(vueAPrefixe, 121)); d != e ||
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
		if d, comment, m := debutDuPaquet(t, c.pay, w, c.g); d != s || comment != commentS || m != mS {
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

// TestUnFilmAncienNePrendPasUneFinDeVueAFermeeAuBitSeulement : le juge de la preuve PREFIXE est
// « fermee sans regle de l ecrivain contredite » (decision (2) de l utilisateur du 2026-10-04), pas
// « fermee au bit pres ». La vue B qui part de E lit un DELTA du slot 124 puis un DELTA du slot 123
// et ferme le paquet au bit pres, mais dans un ordre que l ecrivain n ecrit pas
// ([InvariantOrdreVueB]). Film a table EGALE : E (decision (1)). Film a table PREFIXE : E n est pas
// prouve, le paquet suit le localisateur a l identique. MUTATION — le juge affaibli en
// `FermeeAuBit` dans [debutParLaVueA] : ROUGE.
func TestUnFilmAncienNePrendPasUneFinDeVueAFermeeAuBitSeulement(t *testing.T) {
	pay, e := paquetVueA(1, zoomCourt, func(w *bitWriter) {
		w.deltaMasque13(124)
		w.deltaMasque13(uint32(marchSignatureSlot))
		w.finDeVueB()
		w.bit(0) // vue C vide
	})
	w := mondeDeCarte()
	cfg := cadreDeCarte()
	if l := lectureDEssai(pay, w, cfg, e); !l.FermeeAuBit || l.Fermee || l.Invariant != InvariantOrdreVueB {
		t.Fatalf("lecture depuis E = %d : %+v, attendu fermee au bit, ordre de l ecrivain contredit", e, l)
	}
	if d, comment, m := debutDuPaquet(t, pay, w, grammaireDeScript(vueAEgale, GenresVueA)); d != e ||
		comment != lecture.DebutParVueA || m != e {
		t.Errorf("film recent : cuisson (%d, %d), marches %d ; attendu %d par la vue A", d, comment, m, e)
	}
	s, commentS := localiserLaListe(pay, w, cfg)
	mS, _ := LocaliserBoucleDeRecords(pay, w, cfg, SignaturePuisLargeurLibre)
	d, comment, m := debutDuPaquet(t, pay, w, grammaireDeScript(vueAPrefixe, 121))
	if d == e || m == e || d != s || comment != commentS || m != mS {
		t.Errorf("film ancien : cuisson (%d, %d), marches %d ; attendu le localisateur (%d, %d), %d, et pas E = %d",
			d, comment, m, s, commentS, mS, e)
	}
}

// TestKillsourceLitLaVueASousLaCarteDuMatch : la marche de killsource prend la grammaire de vue A du
// film SOUS LA CARTE DU MATCH ([VueADuFilmSousCarte]) : un impact (genre 6) dont la position porte
// l index de la region jouee se lit sur les tables de cette region, la vue A atteint son terminateur,
// et sa fin est le debut de la vue B (`fb1a1a72`, HI_1_13_0, table EGALE). Le meme film sans carte
// ne connait pas la region jouee : l impact arrete la lecture, et le paquet suit le localisateur.
// MUTATION — la carte ignoree par [VueADuFilmSousCarte] : ROUGE.
func TestKillsourceLitLaVueASousLaCarteDuMatch(t *testing.T) {
	carte := carteDeTest()
	bornes := [3][2]float32{{-100, 100}, {-50, 50}, {-10, 10}}
	pay, e := paquetVueA(1, func(w *bitWriter) {
		w.ecrireEnTeteDeMessage(6)
		w.bit(1)
		w.bits(0, 7+7+0x13) // variante, deux scalaires, direction
		w.ecrirePositionDeNiveau(false, uint64(carte.Region), profile.LargeursAxeDuNiveau(bornes, 0xc))
		w.bits(0, 0x13+9+1)
	}, func(w *bitWriter) {
		w.deltaMasque13(124)
		w.finDeVueB()
		w.vueCUneEntree()
	})
	w := mondeDeCarte(compHighFrequency)
	cfg := cadreDeCarte()
	film := bobineFilm(t, "fb1a1a72")
	sousCarte, sansCarte := VueADuFilmSousCarte(film, &carte), VueADuFilmSousCarte(film, nil)
	if sousCarte.g.classe != vueAEgale {
		t.Fatalf("classe %d, attendu EGALE", sousCarte.g.classe)
	}
	if d, libre := DebutDeLaVueB(pay, w, cfg, sousCarte); d != e || libre {
		t.Errorf("sous la carte : (%d, %v), attendu (%d, faux) par la vue A", d, libre, e)
	}
	s, libreS := LocaliserBoucleDeRecords(pay, w, cfg, SignaturePuisLargeurLibre)
	if s == e {
		t.Fatalf("le localisateur rend E = %d : le vecteur ne distingue pas les deux chemins", e)
	}
	if d, libre := DebutDeLaVueB(pay, w, cfg, sansCarte); d != s || libre != libreS {
		t.Errorf("sans carte : (%d, %v), attendu le localisateur (%d, %v)", d, libre, s, libreS)
	}
}
