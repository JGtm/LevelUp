// Package archlint — no_raw_combat_oc_threshold_test.go : garde-rail du seuil de
// référence OC (conversion offensive) du coach et des jalons combat. Jumeau
// strict de no_raw_combat_dr_threshold_test.go (seuil DR 1.59).
//
// Le littéral 0.83 avait été recopié dans 3 fichiers (coach/generator.go,
// api/wire/post_sync_progression_queries.go, ops/milestone_dates.go) : un
// ajustement du seuil devait donc être répliqué à la main sous peine de
// désaligner l'alerte coach de son jalon. La source unique est
// analysis.CombatOCReferenceThreshold (internal/analysis/combat_yield.go).
//
// CLAUDE.md règle 6 : une factorisation sans garde-rail re-diverge. Ce test
// interdit toute NOUVELLE occurrence du littéral hors du fichier canonique.
//
// Le motif ne cible que le nombre exact 0.83 en position de littéral Go : il
// n'attrape ni 0.833 / 10.83 (bornes numériques), ni la forme française « 0,83 »
// des commentaires, ni les lignes de commentaire (ignorées explicitement).
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

// combatOCCanonicalFile : seul fichier autorisé à porter la valeur littérale.
const combatOCCanonicalFile = "internal/analysis/combat_yield.go"

// rawCombatOCRE matche le littéral 0.83 non précédé/suivi d'un autre chiffre
// (ou d'un point), pour ne pas capturer 0.833, 10.83 ou une version « v0.83.1 ».
var rawCombatOCRE = regexp.MustCompile(`(^|[^\d.])0\.83([^\d]|$)`)

func TestNoRawCombatOCThresholdLiteral(t *testing.T) {
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
				if d.Name() == "migrations" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			if rel == combatOCCanonicalFile {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for i, line := range strings.Split(string(data), "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
					continue
				}
				if rawCombatOCRE.MatchString(line) {
					violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s/: %v", sub, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("littéral 0.83 interdit hors de %s — utiliser "+
			"analysis.CombatOCReferenceThreshold (seuil de référence OC partagé "+
			"coach / jalons combat) :\n  %s",
			combatOCCanonicalFile, strings.Join(violations, "\n  "))
	}
}
