// Package archlint — garde-fous d'architecture vérifiés en test (ratchet).
//
// no_slug_comparison_test.go : interdit les NOUVEAUX gating de titre par
// comparaison de slug (`slug == "halo_infinite"`, `!= title.DefaultSlug`),
// conformément à ADR 0025 + master title-agnostic §7. Tout gating titre doit
// passer par une capability (HasCapability / FeatureChecker), jamais par le
// slug. Les 2 hard-gates connus du câblage multi-titre sont allowlistés (à
// retirer quand l'adapter du 2e titre est enregistré) ; toute occurrence hors
// allowlist fait échouer le test.
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

// slugCompareAllowlist : fichiers (chemin relatif depuis internal/) où un gating
// slug est toléré transitoirement. Ce sont les hard-gates du câblage DataAdapter
// multi-titre (cf. .ai/PLAN_TITLE_AGNOSTIC_TRACKER.md, gaps Phase 2) — à retirer
// quand un 2e titre enregistre son adapter via le resolver.
var slugCompareAllowlist = map[string]bool{
	"api/registry.go":        true, // dataAdapterForPDB : gate DefaultSlug
	"api/registry_career.go": true, // TitleDataAdapter factory : gate DefaultSlug
}

// slugCompareRE matche un gating de titre : une variable *Slug (ou le littéral
// "halo_infinite") comparée par == / != à "halo_infinite" ou title.DefaultSlug.
var slugCompareRE = regexp.MustCompile(
	`(?:[Ss]lug\s*[!=]=\s*(?:"halo_infinite"|(?:title\.)?DefaultSlug))` +
		`|(?:"halo_infinite"\s*[!=]=)`)

func TestNoNewSlugComparison(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile)) // .../internal

	var violations []string
	err := filepath.WalkDir(internalRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(internalRoot, path)
		rel = filepath.ToSlash(rel)
		if slugCompareAllowlist[rel] {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
				continue // ligne de commentaire : explication tolérée
			}
			if slugCompareRE.MatchString(line) {
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("gating de titre par slug interdit (ADR 0025) — passer par une capability "+
			"(HasCapability / FeatureChecker) ou allowlister un hard-gate transitoire :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
