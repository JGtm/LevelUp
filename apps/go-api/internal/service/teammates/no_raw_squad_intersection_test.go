package teammates

// no_raw_squad_intersection_test.go — garde-rail (ratchet) : interdit de consommer
// l'intersection brute (intersectSquadRowsByMatchID) SANS la passer par le filtre
// de composition exacte (filterExactComposition). Sans ce filtre, la page Escouade
// ré-introduit le bug "coéquipier ajouté à une session qu'il n'a pas jouée" : une
// composition {A,B} inclut à tort les matchs {A,B,C} (C = autre coéquipier connu).
//
// Créé 2026-07-24 (fix bugs Escouade). Modèle : archlint/no_inline_expected_fda_test.go.

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestExactCompositionWiringPresent verrouille le câblage des 3 maillons Go :
//   - allSquadRows / allSquadRowsForTimeline passent par filterExactComposition ;
//   - le briefing header applique compFilter.applyShared.
func TestExactCompositionWiringPresent(t *testing.T) {
	dir := packageDir(t)
	mustContain(t, filepath.Join(dir, "teammates_service.go"),
		"allSquadRows = filterExactComposition(allSquadRows,",
		"allSquadRowsForTimeline = filterExactComposition(allSquadRowsForTimeline,")
	mustContain(t, filepath.Join(dir, "teammates_service_briefing.go"),
		"compFilter.applyShared(")
}

// TestNoRawIntersectionConsumption : tout fichier (hors test) qui appelle
// intersectSquadRowsByMatchID DOIT aussi référencer filterExactComposition — le
// résultat de l'intersection ne doit jamais être consommé brut.
func TestNoRawIntersectionConsumption(t *testing.T) {
	dir := packageDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		src := string(data)
		// La déclaration de la fonction elle-même (intersect.go) est tolérée.
		if strings.Contains(src, "func intersectSquadRowsByMatchID(") {
			continue
		}
		if strings.Contains(src, "intersectSquadRowsByMatchID(") &&
			!strings.Contains(src, "filterExactComposition") {
			t.Errorf("%s consomme intersectSquadRowsByMatchID sans filterExactComposition "+
				"(réintroduit le bug de composition non exclusive)", name)
		}
	}
}

func packageDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	return filepath.Dir(thisFile)
}

func mustContain(t *testing.T, path string, needles ...string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	src := string(data)
	for _, n := range needles {
		if !strings.Contains(src, n) {
			t.Errorf("%s doit contenir %q (câblage composition exacte)", filepath.Base(path), n)
		}
	}
}
