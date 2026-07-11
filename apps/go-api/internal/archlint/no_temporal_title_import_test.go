// Package archlint — no_temporal_title_import_test.go : garde-rail F7 (§4).
//
// Le moteur d'engagement (internal/analysis/temporal) doit rester title-AGNOSTIC :
// il reçoit ses inputs (events, poids, signaux) et ne connaît AUCUN titre. Ce test
// interdit tout import d'un package games « titre » (racine internal/games, ou un
// sous-package de titre halo_*/synthetic_*) depuis temporal. Seul internal/games/
// canonical (la lingua franca inter-titres, sans logique de titre) est autorisé.
//
// Calque de no_data_path_join_test : un titre futur (Halo 7) s'active en fournissant
// son ingest + ses coefficients, ZÉRO modification du moteur temporal.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// gamesImportRE capture un import de package games. La branche canonical est
// explicitement tolérée (voir filtre dans le test).
var gamesImportRE = regexp.MustCompile(`"levelup/go-api/internal/games(/[a-z0-9_/]+)?"`)

func TestTemporalImportsNoTitlePackage(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile))
	temporalDir := filepath.Join(internalRoot, "analysis", "temporal")

	var violations []string
	err := filepath.WalkDir(temporalDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		// On scanne aussi les tests : un test qui importerait un package titre
		// trahirait un couplage (fixtures title-specific dans le moteur pur).
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel(filepath.Dir(internalRoot), path)
		rel = filepath.ToSlash(rel)
		for i, line := range strings.Split(string(data), "\n") {
			m := gamesImportRE.FindString(line)
			if m == "" {
				continue
			}
			// canonical = lingua franca inter-titres, autorisée.
			if strings.Contains(m, "internal/games/canonical") {
				continue
			}
			violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk temporal/: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("le moteur temporal doit rester title-agnostic (F7 §4) — "+
			"import d'un package games titre interdit (seul internal/games/canonical est toléré) :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
