// Package archlint — no_inline_expected_fda_test.go : interdit de réécrire la
// formule « FDA attendu » (kills_expected + assists_expected/3 − deaths_expected)
// ailleurs que dans le helper canonique internal/analysis/expected_fda.go
// (ExpectedFDA / FDADiff). Le ratchet flague toute ligne Go non-test, HORS
// internal/analysis/, qui combine un champ *Expected (KillsExpected /
// DeathsExpected / AssistsExpected) avec une division par 3 — signature de
// l'attendu FDA recodé inline. Créé le 2026-07-23 (plan expected-fda, A2).
//
// Pourquoi : centraliser le /3 (règle CLAUDE.md n°6 ≤2 copies + garde-rail) —
// une réécriture inline diverge du helper (ADR 0006 : le FDA attendu a UNE
// formule). Faible taux de faux positifs : les seuls autres usages de
// KillsExpected/DeathsExpected divisent PAR l'attendu (kill value expected) ou
// par math.Max, jamais par 3.
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

// inlineExpectedFDAAllowlist : fichiers (chemin relatif depuis internal/) où une
// arithmétique *Expected .../3 est TOLÉRÉE. Vide par construction : la formule
// vit uniquement dans internal/analysis/expected_fda.go (exclu du walk). Toute
// entrée doit être datée et justifiée.
var inlineExpectedFDAAllowlist = map[string]bool{}

var (
	expectedFieldRE = regexp.MustCompile(`\b(?:Kills|Deaths|Assists)Expected\b`)
	divByThreeRE    = regexp.MustCompile(`/\s*3(?:\.0+)?(?:\D|$)`)
)

func TestNoInlineExpectedFDA(t *testing.T) {
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
		// La formule canonique vit dans internal/analysis/ — hors périmètre du ratchet.
		if strings.HasPrefix(rel, "analysis/") || inlineExpectedFDAAllowlist[rel] {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
				continue // commentaire : explication tolérée
			}
			if expectedFieldRE.MatchString(line) && divByThreeRE.MatchString(line) {
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("FDA attendu recodé inline interdit — passer par analysis.ExpectedFDA "+
			"(source unique de kills_expected + assists_expected/3 − deaths_expected) :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// TestInlineExpectedFDARE_Discriminates verrouille la précision de la détection :
// la combinaison (champ *Expected + /3) DOIT matcher la formule FDA, et NE DOIT
// PAS matcher les usages légitimes (division PAR l'attendu, ou /math.Max).
func TestInlineExpectedFDARE_Discriminates(t *testing.T) {
	matches := func(s string) bool {
		return expectedFieldRE.MatchString(s) && divByThreeRE.MatchString(s)
	}
	mustMatch := []string{
		`exp := *row.KillsExpected + *row.AssistsExpected/3 - *row.DeathsExpected`,
		`v += *r.AssistsExpected / 3.0`,
	}
	for _, s := range mustMatch {
		if !matches(s) {
			t.Errorf("devrait matcher (FDA attendu inline) : %q", s)
		}
	}
	mustNotMatch := []string{
		`kve := float64(row.Kills) / *row.KillsExpected`,               // division PAR l'attendu
		`dve := *row.DeathsExpected / math.Max(1.0, deaths)`,           // /math.Max, pas /3
		`out.KillsExpected = r.Enrichment.SkillSnapshot.KillsExpected`, // simple affectation
		`kdaExp := analysis.ExpectedFDA(k, d, a)`,                      // appel au helper canonique
	}
	for _, s := range mustNotMatch {
		if matches(s) {
			t.Errorf("ne devrait PAS matcher : %q", s)
		}
	}
}
