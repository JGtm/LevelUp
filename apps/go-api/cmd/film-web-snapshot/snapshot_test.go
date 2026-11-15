package main

import (
	"strings"
	"testing"

	"levelup/go-api/internal/testutil"
)

// TestInstantaneEstFrais : l'instantane que lisent les tests web est egal, octet pour octet (fins de ligne
// LF), a ce que la brique du decodeur de film produit aujourd'hui. Sans lui, un test web garderait une
// copie perimee de la source qu'il croit confronter.
func TestInstantaneEstFrais(t *testing.T) {
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	attendu, err := generer()
	if err != nil {
		t.Fatalf("generation depuis la brique : %v", err)
	}
	diff, err := ecarts(racine, attendu)
	if err != nil {
		t.Fatalf("comparaison : %v", err)
	}
	if len(diff) > 0 {
		t.Fatalf("l'instantane des fichiers de la brique lus par les tests web est perime :\n  %s\n"+
			"regenerer : go run ./cmd/film-web-snapshot (depuis apps/go-api), puis relancer les tests web",
			strings.Join(diff, "\n  "))
	}
}

// TestEntetesDeChroniqueNeGardeQueLesEntetes : les trois formes d'en-tete sont gardees dans l'ordre, tout le
// reste est ecarte, y compris une ligne qui ressemble a un en-tete sans en avoir la forme.
func TestEntetesDeChroniqueNeGardeQueLesEntetes(t *testing.T) {
	src := "// intro\n// v3 (2026-01-01) : x\ncode()\n// SCHEMA 4 : y\n// CE QUE LA VERSION 5 : z\n" +
		"// CE QUE LE SCHEMA 6 : w\n// vers 7\n  // v8 (indente)\n"
	got := string(entetesDeChronique([]byte(src)))
	want := "// v3 (2026-01-01) : x\n// SCHEMA 4 : y\n// CE QUE LA VERSION 5 : z\n// CE QUE LE SCHEMA 6 : w\n"
	if got != want {
		t.Fatalf("entetes = %q, attendu %q", got, want)
	}
}

// TestLInstantaneNeContientQueDesFinsDeLigneLF : un contenu genere n'a jamais de retour chariot, et il y a un
// fichier par source plus le manifeste.
func TestLInstantaneNeContientQueDesFinsDeLigneLF(t *testing.T) {
	attendu, err := generer()
	if err != nil {
		t.Fatal(err)
	}
	for nom, contenu := range attendu {
		if strings.Contains(string(contenu), "\r") {
			t.Errorf("%s contient un retour chariot", nom)
		}
	}
	if len(attendu) != len(sources)+1 {
		t.Errorf("%d fichiers generes, %d attendus (sources + manifeste)", len(attendu), len(sources)+1)
	}
}
