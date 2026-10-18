package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// La ligne d'un film liste ses pistes sans identite (ni xuid ni bot) et les compteurs du pont.
func TestLigneParcViesListeLesPistesSansIdentite(t *testing.T) {
	var doc documentMesure
	doc.Tracks = []pisteMesuree{
		{Slot: 527, XUID: "2533", StartFrame: 0, EndFrame: 10},
		{Slot: 540, Bot: "343 Lacuna", StartFrame: 3, EndFrame: 9},
		{Slot: 541, StartFrame: 12, EndFrame: 18},
	}
	got := strings.Split(ligneParcVies("3f048bf3", "3f048bf3-x", "HI_1_13_0", "ok", "Slayer", &doc), "\t")
	want := []string{"3f048bf3", "3f048bf3-x", "HI_1_13_0", "ok", "Slayer", "3", "1", "0", "0", "0/0/0", "541:12-18"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("ligne = %q, attendu %q", got, want)
	}
	if n := len(strings.Split(ligneParcVies("x", "", "", "echec", "", nil), "\t")); n != len(want) {
		t.Fatalf("une ligne sans document doit porter %d colonnes, elle en porte %d", len(want), n)
	}
}

// La comparaison separe les vies nommees, perdues, changees et les equipes changees, et rapproche
// une piste redecoupee de celles du meme slot qui la recouvrent.
func TestComparerFilmClasseLesChangements(t *testing.T) {
	avant := []pisteRelue{
		{slot: 1, debut: 0, fin: 10, identite: "A"},
		{slot: 2, debut: 0, fin: 10, identite: ""},
		{slot: 3, debut: 0, fin: 10, identite: "C"},
		{slot: 4, debut: 0, fin: 10, identite: "D"},
		{slot: 5, debut: 0, fin: 10, team: 0, identite: "E"},
		{slot: 6, debut: 0, fin: 10, identite: "F"},
	}
	apres := []pisteRelue{
		{slot: 1, debut: 0, fin: 10, identite: "A"},
		{slot: 2, debut: 0, fin: 10, identite: "B"},
		{slot: 3, debut: 0, fin: 10, identite: ""},
		{slot: 4, debut: 0, fin: 10, identite: "X"},
		{slot: 5, debut: 0, fin: 10, team: 1, identite: "E"},
		{slot: 6, debut: 2, fin: 10, identite: "G"},
	}
	var b bilanComparaison
	comparerFilm(&b, "f", avant, apres)
	if b.nommees != 1 || b.perdues != 1 || b.changees != 1 || b.equipes != 1 || b.decoupe != 1 {
		t.Fatalf("bilan = %+v", b)
	}
	if b.sansNomAvant != 1 || b.sansNomApres != 1 {
		t.Fatalf("sans nom avant/apres = %d/%d", b.sansNomAvant, b.sansNomApres)
	}
	if !strings.Contains(strings.Join(b.lignes, "\n"), `redecoupee 6:0-10 "F" -> 6:2-10 "G"`) {
		t.Fatalf("la piste redecoupee au nom change doit etre listee : %q", b.lignes)
	}
}

// La reprise relit les films deja ecrits ; le fichier d'identites se relit tel qu'il s'ecrit.
func TestRepriseEtIdentitesSeRelisent(t *testing.T) {
	dir := t.TempDir()
	tsv := filepath.Join(dir, "films.tsv")
	if err := os.WriteFile(tsv, []byte("aaaa\tm\t\tok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	vus, err := idsDejaMesures(tsv)
	if err != nil || !vus["aaaa"] || len(vus) != 1 {
		t.Fatalf("reprise = %v, %v", vus, err)
	}
	chemin := filepath.Join(dir, "aaaa.tsv")
	var doc documentMesure
	doc.Tracks = []pisteMesuree{{Slot: 7, Team: 1, Bot: "343 Ritzy", StartFrame: 4, EndFrame: 9}, {Slot: 8}}
	ecrit := identitesDuDocument(doc)
	if !strings.HasSuffix(strings.Split(ecrit, "\n")[1], "\t\t") {
		t.Fatalf("une piste sans lien couvrant porte une provenance vide : %q", ecrit)
	}
	if err := os.WriteFile(chemin, []byte(ecrit), 0o600); err != nil {
		t.Fatal(err)
	}
	relues, err := lirePistes(chemin)
	if err != nil {
		t.Fatal(err)
	}
	if len(relues) != 2 || relues[0].identite != "bot:343 Ritzy" || relues[0].team != 1 || relues[1].identite != "" {
		t.Fatalf("pistes relues = %+v", relues)
	}
}
