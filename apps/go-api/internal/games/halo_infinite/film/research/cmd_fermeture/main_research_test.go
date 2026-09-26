//go:build research

package main

// main_research_test.go — L OUTIL DE BOUT EN BOUT, sur les bobines VERSIONNEES du depot (jamais
// un film de `data/`) : la bobine contigue de `killsource` (registre + trames delta) et une
// mini-bobine du rejeu (images-cles seules : zero paquet delta). Rapport dans un repertoire
// temporaire.
//
//	go test -tags=research ./internal/games/halo_infinite/film/research/cmd_fermeture/

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// racinesDeTest : les repertoires qui portent les bobines, relatifs a ce paquet.
const (
	racineKillsource = "../../internal/facts/killsource/testdata"
	racineRejeu      = "../../replay/testdata"
	cheminTable      = "../../internal/grammar/testdata/ecs_table.tsv"
)

// TestOutilEcritSesQuatreSorties : deux films mesures, un par racine, et les quatre fichiers du
// rapport portent leurs lignes.
func TestOutilEcritSesQuatreSorties(t *testing.T) {
	dir := t.TempDir()
	tab, err := lireTable(cheminTable)
	if err != nil {
		t.Fatal(err)
	}
	rap, err := ouvrirRapport(dir, tab)
	if err != nil {
		t.Fatal(err)
	}
	if err := mesurerUnFilm(racineKillsource, "minibobine_000d5950", 4, rap); err != nil {
		t.Fatalf("bobine contigue : %v", err)
	}
	if err := mesurerUnFilm(racineRejeu, "minifilm_bcb6d393", 4, rap); err != nil {
		t.Fatalf("mini-bobine du rejeu : %v", err)
	}
	if err := rap.terminer(10); err != nil {
		t.Fatal(err)
	}
	attendus := map[string]string{
		"fermeture_films.tsv":      "minibobine_000d5950\t",
		"fermeture_archetypes.tsv": "minibobine_000d5950\t",
		"fermeture_bloquants.tsv":  "device-animation-layer-state-component",
		"fermeture_resume.md":      "## Causes d arret",
	}
	for nom, marque := range attendus {
		brut, err := os.ReadFile(filepath.Join(dir, nom)) //nolint:gosec // repertoire du test
		if err != nil {
			t.Fatalf("%s : %v", nom, err)
		}
		if !strings.Contains(string(brut), marque) {
			t.Errorf("%s ne porte pas %q :\n%s", nom, marque, brut)
		}
	}
	if rap.mesures != 2 {
		t.Errorf("%d film(s) mesure(s), attendu 2", rap.mesures)
	}
}

// TestSortieSousDataRefusee : un repertoire de rapport sous `data/` est refuse avant toute
// ecriture.
func TestSortieSousDataRefusee(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data", "rapport")
	if err := preparerSortie(dir); err == nil {
		t.Fatalf("%s accepte : le rapport doit s ecrire hors de data/", dir)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("%s a ete cree malgre le refus", dir)
	}
}

// TestLimiteBorneLaListe : `-limite` garde les premiers films, dans l ordre donne.
func TestLimiteBorneLaListe(t *testing.T) {
	ids := decouper(" a, b ,,c ")
	if got := strings.Join(borner(ids, 2), ","); got != "a,b" {
		t.Errorf("borner(2) = %q, attendu a,b", got)
	}
	if got := strings.Join(borner(ids, 0), ","); got != "a,b,c" {
		t.Errorf("borner(0) = %q, attendu a,b,c", got)
	}
}
