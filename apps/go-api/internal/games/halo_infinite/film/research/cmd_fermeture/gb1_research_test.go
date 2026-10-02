//go:build research

package main

// gb1_research_test.go — LE MODE `gb1` DE BOUT EN BOUT, sur les deux mini-bobines VERSIONNEES de
// `killsource` a trames delta (jamais un film de `data/`) : `minibobine_000d5950` (prefixe
// contigu, chunks 00 a 05 + un chunk highlight) et `minibobine_e5adf7b2` (Big Team Battle,
// version 40 : registre, premier chunk de donnees, un chunk highlight). Rapport dans un
// repertoire temporaire.
//
//	go test -tags=research ./internal/games/halo_infinite/film/research/cmd_fermeture/ -run GB1 -v

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// bobinesGB1 : les deux mini-bobines a trames delta, mesurees dans cet ordre.
var bobinesGB1 = []string{"minibobine_000d5950", "minibobine_e5adf7b2"}

// TestModeGB1EcritSesSortiesEtSesInvariants : les deux bobines mesurees en mode `gb1` seul ; les
// trois TSV et la section du resume portent leurs lignes, et les invariants de la mesure tiennent.
func TestModeGB1EcritSesSortiesEtSesInvariants(t *testing.T) {
	dir := t.TempDir()
	rap, err := ouvrirRapport(dir, tableECS{}, modes{gb1: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range bobinesGB1 {
		if err := mesurerUnFilm(racineKillsource, id, 4, rap); err != nil {
			t.Fatalf("%s : %v", id, err)
		}
	}
	lignes := rap.gb1.lignes
	if err := rap.terminer(10); err != nil {
		t.Fatal(err)
	}
	if rap.mesures != len(bobinesGB1) || len(lignes) != len(bobinesGB1) {
		t.Fatalf("%d film(s) mesure(s), %d ligne(s) de resume, attendu %d", rap.mesures, len(lignes), len(bobinesGB1))
	}
	for _, l := range lignes {
		verifierInvariantsGB1(t, l)
	}
	attendus := map[string]string{
		"gb1_films.tsv":       "minibobine_e5adf7b2\t",
		"gb1_vies.tsv":        "minibobine_000d5950\t",
		"gb1_orphelins.tsv":   "film\tslot\ttag",
		"fermeture_resume.md": "## GB-1",
	}
	for nom, marque := range attendus {
		brut, err := os.ReadFile(filepath.Join(dir, nom)) //nolint:gosec // repertoire du test
		if err != nil {
			t.Fatalf("%s : %v", nom, err)
		}
		if !strings.Contains(string(brut), marque) {
			t.Errorf("%s ne porte pas %q :\n%s", nom, marque, brut)
		}
		t.Logf("%s :\n%s", nom, premieresLignes(string(brut), 8))
	}
	if _, err := os.Stat(filepath.Join(dir, "fermeture_films.tsv")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("fermeture_films.tsv cree en mode gb1 seul")
	}
}

// statutsAttendusGB1 : `minibobine_e5adf7b2` ne garde, en donnees, que le PREMIER chunk du film
// (avant toute apparition : aucun `ti=35` aux images-cles, mesure du 2026-09-26) — le mode doit
// l ecrire comme tel, sans echouer le film.
var statutsAttendusGB1 = map[string]string{
	"minibobine_000d5950": statutMesure,
	"minibobine_e5adf7b2": statutSansBipede,
}

// verifierInvariantsGB1 : ce qui doit tenir sur tout film, par construction de la mesure.
func verifierInvariantsGB1(t *testing.T, l ligneResumeGB1) {
	t.Helper()
	if l.statut != statutsAttendusGB1[l.id] {
		t.Errorf("%s : statut %q, attendu %q", l.id, l.statut, statutsAttendusGB1[l.id])
	}
	s := l.s
	if s.dureeFilm <= 0 {
		t.Errorf("%s : duree du film %d ms", l.id, s.dureeFilm)
	}
	if l.statut != statutMesure {
		return
	}
	if s.vies == 0 || s.entetes == 0 || s.dureePubliee <= 0 {
		t.Errorf("%s : %d vie(s), %d en-tete(s) brut(s), durationMs %d — la bobine porte des bipedes",
			l.id, s.vies, s.entetes, s.dureePubliee)
	}
	// LE CONSTAT GB-1 PAR CONSTRUCTION : le filtre de production ne rattache rien a une vie de
	// generation >= 2.
	if s.gen2SansProd != s.viesGen2 {
		t.Errorf("%s : %d vie(s) gen >= 2 sans position prod sur %d", l.id, s.gen2SansProd, s.viesGen2)
	}
	if s.vies != s.creationSeule+s.imageCleSeule+s.deuxSources {
		t.Errorf("%s : partition des sources des vies rompue", l.id)
	}
	if s.orphelinsGardes > s.orphelins {
		t.Errorf("%s : %d orphelins gardes pour %d orphelins", l.id, s.orphelinsGardes, s.orphelins)
	}
	// MESURE DU 2026-09-26 SUR `minibobine_000d5950` (17 vies, toutes de generation 1) : filtre
	// desarme, le marcheur n ancre AUCUN en-tete de plus, et chaque vie a ses positions. Un tag relu
	// au mauvais decalage y ferait apparaitre des orphelins et des vies sans position vivante.
	if l.id == "minibobine_000d5950" && (s.orphelins != 0 || s.sansVivante != 0 || s.viesGen2 != 0) {
		t.Errorf("%s : %d orphelin(s), %d vie(s) sans position vivante, %d vie(s) gen >= 2 ; attendu 0, 0, 0",
			l.id, s.orphelins, s.sansVivante, s.viesGen2)
	}
}

// TestDeuxModesSurUnFilm : `-mode fermeture,gb1` mesure le film une fois, ecrit les sorties des
// deux modes et les deux familles de sections du resume.
func TestDeuxModesSurUnFilm(t *testing.T) {
	dir := t.TempDir()
	tab, err := lireTable(cheminTable)
	if err != nil {
		t.Fatal(err)
	}
	rap, err := ouvrirRapport(dir, tab, modes{fermeture: true, gb1: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := mesurerUnFilm(racineKillsource, "minibobine_000d5950", 4, rap); err != nil {
		t.Fatal(err)
	}
	if err := rap.terminer(5); err != nil {
		t.Fatal(err)
	}
	if rap.mesures != 1 {
		t.Errorf("%d film(s) mesure(s), attendu 1", rap.mesures)
	}
	resume, err := os.ReadFile(filepath.Join(dir, "fermeture_resume.md")) //nolint:gosec // repertoire du test
	if err != nil {
		t.Fatal(err)
	}
	for _, marque := range []string{"## Par build", "## Causes d arret", "## GB-1"} {
		if !strings.Contains(string(resume), marque) {
			t.Errorf("le resume ne porte pas %q", marque)
		}
	}
	for _, nom := range []string{"fermeture_films.tsv", "gb1_films.tsv"} {
		if _, err := os.Stat(filepath.Join(dir, nom)); err != nil {
			t.Errorf("%s : %v", nom, err)
		}
	}
}

// TestLireModes : `-mode` accepte les deux modes, seuls ou ensemble, et refuse le reste.
func TestLireModes(t *testing.T) {
	cas := map[string]modes{
		"fermeture":      {fermeture: true},
		"gb1":            {gb1: true},
		"fermeture, gb1": {fermeture: true, gb1: true},
		"v2":             {fermeture: true, v2: true},
		"v2,gb1":         {fermeture: true, gb1: true, v2: true},
	}
	for v, attendu := range cas {
		md, err := lireModes(v)
		if err != nil || md != attendu {
			t.Errorf("lireModes(%q) = %+v, %v ; attendu %+v", v, md, err, attendu)
		}
	}
	for _, v := range []string{"", "gb2", "gb1,autre"} {
		if _, err := lireModes(v); err == nil {
			t.Errorf("lireModes(%q) accepte", v)
		}
	}
}

// premieresLignes rend les n premieres lignes d un texte (journal du test).
func premieresLignes(s string, n int) string {
	l := strings.SplitN(s, "\n", n+1)
	if len(l) > n {
		l = l[:n]
	}
	return strings.Join(l, "\n")
}

// TestRattachementSynthetique : les chemins que les deux bobines n exercent pas (aucune vie de
// generation >= 2, aucun orphelin sur `minibobine_000d5950`) — vie de generation 2 positionnee
// par la generation vivante, orphelin isole ecarte, orphelin garde, vie sans position.
func TestRattachementSynthetique(t *testing.T) {
	const s = uint64(1_000_000)
	m := mesureGB1{statut: statutMesure, vies: map[cleDeVie]*vieGB1{}, orphelins: map[cleDeVie]*orphelinGB1{}}
	m.vie(cleDeVie{10, 1}).creations = 1
	m.vie(cleDeVie{10, 2}).creations = 1
	m.vie(cleDeVie{11, 1}).imagesCles = 1
	pos := func(slot uint32, ts uint64) grammar.BipedPosition {
		return grammar.BipedPosition{Slot: slot, TimestampUS: ts}
	}
	brut := []grammar.BipedPosition{pos(10, 1*s), pos(10, 2*s), pos(10, 3*s), pos(10, 20*s), pos(10, 21*s),
		pos(11, 5*s), pos(12, 30*s), pos(12, 31*s)}
	tags := []uint32{1, 1, 1, 2, 2, 3, 0, 0}
	m.rattacherBrut(brut, tags)
	m.delta = bornes{debut: 1 * s, fin: 40 * s}
	if v := m.vies[cleDeVie{10, 2}]; v.brut != 2 || v.vives != 2 || v.premierUS != 20*s || v.dernierUS != 21*s {
		t.Errorf("vie (10, 2) : %+v", *v)
	}
	if m.orphelinsParTag[3] != 1 || m.gardesParTag[3] != 0 || m.orphelinsParTag[0] != 2 || m.gardesParTag[0] != 2 {
		t.Errorf("orphelins par tag %v, gardes %v", m.orphelinsParTag, m.gardesParTag)
	}
	syn := synthetiser(m)
	attendu := syntheseGB1{vies: 3, viesGen2: 1, slotsMultiVies: 1, creationSeule: 2, imageCleSeule: 1,
		sansProd: 3, gen2SansProd: 1, sansVivante: 1, gen2AvecVivante: 1, entetes: 8, orphelins: 3,
		orphelinsGardes: 2, dureeDelta: 39_000, dureeVivante: 20_100, finTronqueeVivante: 19_000}
	if syn != attendu {
		t.Errorf("synthese\n  %+v\nattendu\n  %+v", syn, attendu)
	}
}
