// Package archlint — garde-fou : DDL de match_objective_stats_latest centralisé.
//
// no_inline_objective_latest_view_test.go : interdit tout DDL inline
// `CREATE [OR REPLACE] VIEW match_objective_stats_latest` hors de la source unique
// migration.MatchObjectiveStatsLatestViewSQL (internal/migration/objective_stats_view.go).
//
// Contexte (2026-07-26, régression v7.2 scoreboard_empty) : le DDL existait en 4
// copies (2 steps de migration + 2 fixtures de test) et une 5e allait naître dans la
// lecture snapshot (ops/snapshot_read.go) — la vue manquait du schéma snapshot
// reconstruit et Q12 échouait en Catalog Error → scoreboard vide → bandeau « match
// partiel » sur TOUS les matchs du cut. Règle projet « ≤ 2 copies d'un même
// pattern » : helper canonique + ce ratchet. Contrairement aux autres tests archlint,
// celui-ci scanne AUSSI les _test.go : deux des copies historiques étaient des
// fixtures de test.
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

// objectiveLatestViewAllowlist : seuls fichiers (chemin relatif depuis internal/)
// autorisés à porter le littéral DDL de la vue.
var objectiveLatestViewAllowlist = map[string]bool{
	"migration/objective_stats_view.go":                true, // source unique (MatchObjectiveStatsLatestViewSQL)
	"archlint/no_inline_objective_latest_view_test.go": true, // ce garde-rail (vecteurs de self-test)
}

// objectiveLatestViewRE matche un DDL de création de la vue (CREATE VIEW /
// CREATE OR REPLACE VIEW, insensible à la casse). Les LECTURES de la vue
// (`FROM match_objective_stats_latest`) restent libres.
var objectiveLatestViewRE = regexp.MustCompile(`(?i)VIEW\s+match_objective_stats_latest`)

func TestNoInlineObjectiveLatestViewDDL(t *testing.T) {
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
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, _ := filepath.Rel(internalRoot, path)
		rel = filepath.ToSlash(rel)
		if objectiveLatestViewAllowlist[rel] {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
				continue // commentaire : mention tolérée
			}
			if objectiveLatestViewRE.MatchString(line) {
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("DDL inline de match_objective_stats_latest interdit — utiliser "+
			"migration.MatchObjectiveStatsLatestViewSQL (source unique, "+
			"internal/migration/objective_stats_view.go) :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// TestObjectiveLatestViewRE_Vectors verrouille la portée de la regex : elle DOIT
// attraper les formes DDL (avec ou sans OR REPLACE, source qualifiée ou non) et NE
// DOIT PAS attraper une lecture de la vue.
func TestObjectiveLatestViewRE_Vectors(t *testing.T) {
	mustMatch := []string{
		"CREATE OR REPLACE VIEW match_objective_stats_latest AS",
		"CREATE VIEW match_objective_stats_latest AS SELECT * FROM shared.match_objective_stats",
		"create or replace view\tmatch_objective_stats_latest as",
	}
	for _, s := range mustMatch {
		if !objectiveLatestViewRE.MatchString(s) {
			t.Errorf("regex devrait matcher (DDL) : %q", s)
		}
	}
	mustNotMatch := []string{
		"SELECT * FROM match_objective_stats_latest WHERE match_id = ?",
		"LEFT JOIN match_objective_stats_latest o ON o.match_id = p.match_id",
	}
	for _, s := range mustNotMatch {
		if objectiveLatestViewRE.MatchString(s) {
			t.Errorf("regex ne devrait PAS matcher (lecture) : %q", s)
		}
	}
}
