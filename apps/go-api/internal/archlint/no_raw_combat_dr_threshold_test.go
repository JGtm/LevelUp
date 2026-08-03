// Package archlint — no_raw_combat_dr_threshold_test.go : garde-rail du seuil de
// référence DR (résistance défensive) du coach et des jalons combat.
//
// Le littéral 1.59 avait été recopié dans 4 fichiers (coach/generator.go,
// coach_advisor/signals.go, api/wire/post_sync_progression_queries.go,
// ops/milestone_dates.go) : un ajustement du seuil devait donc être répliqué à
// la main sous peine de désaligner l'alerte coach de son jalon. La source unique
// est analysis.CombatDRReferenceThreshold (internal/analysis/combat_yield.go).
//
// CLAUDE.md règle 6 : une factorisation sans garde-rail re-diverge. Ce test
// interdit toute NOUVELLE occurrence du littéral hors du fichier canonique.
//
// Le motif ne cible que le nombre exact 1.59 en position de littéral Go : il
// n'attrape ni 1.590x / 11.59 (bornes numériques), ni la forme française « 1,59 »
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

// combatDRCanonicalFile : seul fichier autorisé à porter la valeur littérale.
const combatDRCanonicalFile = "internal/analysis/combat_yield.go"

// rawCombatDRRE matche le littéral 1.59 non précédé/suivi d'un autre chiffre
// (ou d'un point), pour ne pas capturer 1.595, 21.59 ou une version « v1.59.2 ».
var rawCombatDRRE = regexp.MustCompile(`(^|[^\d.])1\.59([^\d]|$)`)

func TestNoRawCombatDRThresholdLiteral(t *testing.T) {
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
			if rel == combatDRCanonicalFile {
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
				if rawCombatDRRE.MatchString(line) {
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
		t.Errorf("littéral 1.59 interdit hors de %s — utiliser "+
			"analysis.CombatDRReferenceThreshold (seuil de référence DR partagé "+
			"coach / jalons combat) :\n  %s",
			combatDRCanonicalFile, strings.Join(violations, "\n  "))
	}
}
