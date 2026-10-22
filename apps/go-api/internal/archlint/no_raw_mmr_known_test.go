// Package archlint — no_raw_mmr_known_test.go : ratchet de la règle « un MMR à 0 est une valeur
// absente ».
//
// La règle s'écrit UNE fois (analysis/mmr_known.go : MMRKnown, KnownMMR, KnownMMRValue,
// SQLKnownMMR). Un match classé abandonné revient de l'API avec un MMR à 0 : lu comme un niveau,
// il tire les moyennes, les écarts et le régresseur des assistances attendues. Deux manières de
// la contourner sont interdites hors du helper :
//   - comparer à la main un MMR à 0 (`TeamMMR > 0`) : la règle diverge d'une copie à l'autre ;
//   - projeter les colonnes brutes `<alias>.team_mmr` / `<alias>.enemy_mmr` dans une lecture SQL
//     (platform/, sync/) : la ligne servie porterait le 0.
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

var (
	// rawMMRComparison : un champ MMR comparé à 0 (`TeamMMR > 0`, `enemyMMR >= 0`).
	rawMMRComparison = regexp.MustCompile(`\b(Team|Enemy|team|enemy)MMR\s*(>|>=)\s*0\b`)
	// rawMMRProjection : la colonne brute projetée (`    p.team_mmr,`, `COALESCE(mp.enemy_mmr, 0)`),
	// hors prédicat `IS [NOT] NULL` qui ne lit pas la valeur.
	rawMMRProjection = regexp.MustCompile(`(^|[^_A-Za-z0-9"])[a-z][a-z0-9]*\.(team|enemy)_mmr\b`)
)

func walkGoSources(t *testing.T, subs []string, visit func(rel string, lines []string)) {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	for _, sub := range subs {
		err := filepath.WalkDir(filepath.Join(goAPIRoot, sub), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "migrations" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			visit(filepath.ToSlash(rel), strings.Split(string(data), "\n"))
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s/: %v", sub, err)
		}
	}
}

func isCommentLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "--")
}

func TestNoRawMMRZeroComparison(t *testing.T) {
	var violations []string
	walkGoSources(t, []string{"internal", "cmd"}, func(rel string, lines []string) {
		if rel == "internal/analysis/mmr_known.go" {
			return
		}
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if isCommentLine(trimmed) {
				continue
			}
			if rawMMRComparison.MatchString(line) {
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
			}
		}
	})
	if len(violations) > 0 {
		t.Errorf("comparaison brute d'un MMR à 0 interdite — utiliser analysis.MMRKnown / "+
			"KnownMMR / KnownMMRValue :\n  %s", strings.Join(violations, "\n  "))
	}
}

func TestNoRawMMRColumnProjection(t *testing.T) {
	var violations []string
	walkGoSources(t, []string{"internal/platform", "internal/sync"}, func(rel string, lines []string) {
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if isCommentLine(trimmed) {
				continue
			}
			loc := rawMMRProjection.FindStringIndex(line)
			if loc == nil {
				continue
			}
			rest := strings.ToUpper(line[loc[1]:])
			if strings.HasPrefix(strings.TrimSpace(rest), "IS NULL") || strings.HasPrefix(strings.TrimSpace(rest), "IS NOT NULL") {
				continue
			}
			// Le fragment du helper lui-même : `analysis.SQLKnownMMR("p.team_mmr")`.
			if strings.Contains(line, "SQLKnownMMR(") {
				continue
			}
			violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
		}
	})
	if len(violations) > 0 {
		t.Errorf("colonne team_mmr / enemy_mmr projetée brute — l'envelopper dans "+
			"analysis.SQLKnownMMR(col) (un MMR à 0 est une valeur absente) :\n  %s", strings.Join(violations, "\n  "))
	}
}
