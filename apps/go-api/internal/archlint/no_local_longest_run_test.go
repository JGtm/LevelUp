// Package archlint — no_local_longest_run_test.go : ratchet F2 (revue 2026-07-17).
//
// Interdit toute nouvelle copie locale du balayage « plus longue série
// consécutive » (idiome `cur++` suivi d'un `if cur > best {` de mise à jour du
// max). La source unique est analysis.LongestRun[T] (internal/analysis). Le motif
// avait été recopié en quatre exemplaires (longestOutcomeRun,
// sliceBestWinStreakCanonical, synthesis winStreak/maxStreak, detectTilt) —
// leçon CLAUDE.md règle 6.
//
// ALLOWLIST (accumulateurs sémantiquement distincts, PAS un LongestRun binaire) :
//   - longest_run.go       : la source unique elle-même.
//   - max_killing_spree.go : accumulateur À TROIS ÉTATS (kill=+1 / death=reset /
//     autre event=IGNORÉ), non réductible à LongestRun(items, pred) qui reset sur
//     tout non-match. Distinct par construction — daté 2026-07-17.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// increment `x++` puis peu après `if y > z {` — signature du balayage best/cur.
var longestRunIdiomRE = regexp.MustCompile(`\+\+[ \t]*\n[ \t]*if\s+\w+\s*>\s*\w+\s*\{`)

var longestRunAllowed = map[string]bool{
	"longest_run.go":       true,
	"max_killing_spree.go": true,
}

func TestNoLocalLongestRun(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile))
	goAPIRoot := filepath.Dir(internalRoot)

	var violations []string
	for _, sub := range []string{"internal", "cmd"} {
		root := filepath.Join(goAPIRoot, sub)
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if longestRunAllowed[filepath.Base(path)] {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if longestRunIdiomRE.Match(data) {
				rel, _ := filepath.Rel(goAPIRoot, path)
				violations = append(violations, filepath.ToSlash(rel))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s/: %v", sub, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("balayage « plus longue série » local interdit (F2) — utiliser "+
			"analysis.LongestRun[T] (internal/analysis) :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
