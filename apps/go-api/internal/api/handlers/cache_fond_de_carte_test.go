package handlers

// cache_fond_de_carte_test.go — garde-rail anti-duplication (CLAUDE.md règle n°6) du Cache-Control des
// routes de fond de carte : les quatre routes qui servent des octets d'un fond (image du fond, plan,
// relief, données du relief) passent par la constante `cacheFondDeCarte`. Le littéral
// `private, max-age=3600` ne s'écrit que dans son fichier de définition.

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// cacheFondDeCarteDefFile : SEUL fichier de production du paquet autorisé à écrire le littéral du
// Cache-Control des fonds de carte. Allowlist VIDE et datée : aucune autre exemption au 2026-11-18.
const cacheFondDeCarteDefFile = "background_variante.go"

// reCacheFondLitteral : le littéral du Cache-Control des fonds, quel que soit l'espacement.
var reCacheFondLitteral = regexp.MustCompile(`private,\s*max-age=3600`)

func TestCacheFondDeCarteSansLitteralHorsDeSaDefinition(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	dir := filepath.Dir(thisFile)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var offenders []string
	for _, e := range entries {
		nom := e.Name()
		if !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") || nom == cacheFondDeCarteDefFile {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, nom))
		if err != nil {
			t.Fatalf("lecture de %s : %v", nom, err)
		}
		if reCacheFondLitteral.Match(src) {
			offenders = append(offenders, nom)
		}
	}
	if len(offenders) > 0 {
		t.Errorf("littéral du Cache-Control des fonds hors de %s — utiliser cacheFondDeCarte :\n  %s",
			cacheFondDeCarteDefFile, strings.Join(offenders, "\n  "))
	}
}

// Le garde-rail MORD sur le littéral et ne mord QUE sur lui, et son fichier de définition le porte
// toujours : sans quoi l'exemption serait un trou.
func TestCacheFondDeCarteGuardIsDiscriminant(t *testing.T) {
	for _, src := range []string{
		`servirBlobAvecETag(w, r, blob, ct, "private, max-age=3600")`,
		"servirBlobAvecETag(w, r, blob, ct, `private,  max-age=3600`)",
	} {
		if !reCacheFondLitteral.MatchString(src) {
			t.Errorf("littéral NON détecté (garde-rail aveugle) : %q", src)
		}
	}
	for _, src := range []string{
		`servirBlobAvecETag(w, r, blob, ct, cacheFondDeCarte)`,
		`"public, max-age=3600"`,
		`"private, max-age=60"`,
	} {
		if reCacheFondLitteral.MatchString(src) {
			t.Errorf("FAUX POSITIF sur usage légitime : %q", src)
		}
	}
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	src, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), cacheFondDeCarteDefFile))
	if err != nil {
		t.Fatalf("lecture de %s : %v", cacheFondDeCarteDefFile, err)
	}
	if !reCacheFondLitteral.Match(src) {
		t.Errorf("%s ne porte plus le littéral : son exemption est devenue un trou", cacheFondDeCarteDefFile)
	}
}
