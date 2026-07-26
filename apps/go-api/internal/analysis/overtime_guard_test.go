package analysis

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// Garde-rail (CLAUDE.md n°6) : la marge overtime est une SOURCE UNIQUE
// (analysis.OvertimeMarginSeconds, overtime.go). Une factorisation sans
// garde-rail re-diverge — ce test échoue si :
//
//  1. un autre fichier redéfinit un symbole « overtimeMargin » (const/var/champ) ;
//  2. un littéral de marge est recalculé à la main à côté d'un temps
//     réglementaire / d'un overtime (ex. `regulationSeconds + 40`).
//
// Seul overtime.go (et ses tests) porte la valeur.
var (
	reMarginRedefinition = regexp.MustCompile(`(?i)(const|var)\s+\w*overtimeMargin\w*\s*[:=]`)
	reHandRolledMargin   = regexp.MustCompile(`(?i)(regulation|overtime)\w*\s*[+\-]\s*\d+`)
)

func TestOvertimeMarginHasSingleSource(t *testing.T) {
	t.Parallel()
	_, thisFile, _, _ := runtime.Caller(0)
	goAPIRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	canonical := map[string]bool{
		"overtime.go":            true,
		"overtime_test.go":       true,
		"overtime_guard_test.go": true,
	}

	var offenders []string
	err := filepath.Walk(goAPIRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "testdata" || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".go") || canonical[info.Name()] {
			return nil
		}
		src, readErr := os.ReadFile(path) //nolint:gosec // chemin dérivé du repo
		if readErr != nil {
			return readErr
		}
		for _, line := range strings.Split(string(src), "\n") {
			code := strings.TrimSpace(line)
			if code == "" || strings.HasPrefix(code, "//") {
				continue // les commentaires peuvent citer la mesure
			}
			if reMarginRedefinition.MatchString(code) || reHandRolledMargin.MatchString(code) {
				offenders = append(offenders, path+" : "+code)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(offenders) > 0 {
		t.Errorf("marge overtime recopiée hors de analysis/overtime.go — utiliser "+
			"analysis.OvertimeMarginSeconds / analysis.ComputeOvertime :\n%s",
			strings.Join(offenders, "\n"))
	}
}
