package archlint

// v_match_full_ddl_single_source_test.go — UN SEUL DDL DE v_match_full (CLAUDE.md règle 6).
//
// La source est la constante `migration.VMatchFullViewSQL` (internal/migration/steps_shared.go).
// Les chemins qui recréent la vue (migrations, schéma de secours du moteur, seed de la démo)
// l'importent. DuckDB fige l'étoile `mr.*` à la création : un second littéral qui diverge de la
// première (colonne explicite, filtre) donne deux définitions de la même vue selon le chemin
// qui l'a posée en dernier.
//
// RÈGLE. Dans le code non-test du module, aucun fichier hors de la source ne porte un
// `CREATE [OR REPLACE] VIEW v_match_full`. Les tests construisent leurs propres vues réduites
// (schémas minimaux) : ils ne sont pas lus.
//
// Mutation qui doit le faire rougir : remettre le littéral `CREATE OR REPLACE VIEW v_match_full
// AS SELECT mr.* FROM match_registry mr` dans internal/sync/schema.go ou internal/ops/seed_demo.go.

import (
	"regexp"
	"testing"
)

const vMatchFullSourceFile = "internal/migration/steps_shared.go"

// reCreateVMatchFull : la création de la vue, quelle que soit la casse ou la mise en forme.
var reCreateVMatchFull = regexp.MustCompile(`(?i)CREATE\s+(OR\s+REPLACE\s+)?VIEW\s+(IF\s+NOT\s+EXISTS\s+)?v_match_full\b`)

func TestVMatchFullDDLSingleSource(t *testing.T) {
	t.Parallel()
	sources := goSourcesUnder(t, "internal", "cmd")
	var hors []string
	vueDansLaSource := false
	for _, s := range sources {
		if !reCreateVMatchFull.Match(s.Src) {
			continue
		}
		if s.Rel == vMatchFullSourceFile {
			vueDansLaSource = true
			continue
		}
		hors = append(hors, s.Rel)
	}
	if !vueDansLaSource {
		t.Fatalf("%s ne porte plus le DDL de v_match_full : le garde-rail ne tient plus rien", vMatchFullSourceFile)
	}
	for _, rel := range hors {
		t.Errorf("%s : second DDL de v_match_full ; importer migration.VMatchFullViewSQL", rel)
	}
}

// TestVMatchFullDDLSingleSource_Preuve : le motif voit un littéral recopié, et ignore la vue
// d'un autre nom.
func TestVMatchFullDDLSingleSource_Preuve(t *testing.T) {
	t.Parallel()
	for _, src := range []string{
		"CREATE OR REPLACE VIEW v_match_full AS SELECT mr.* FROM match_registry mr",
		"create view v_match_full as select 1",
		"CREATE   OR\nREPLACE VIEW  v_match_full AS SELECT 1",
	} {
		if !reCreateVMatchFull.MatchString(src) {
			t.Errorf("le motif aurait dû voir : %q", src)
		}
	}
	for _, src := range []string{
		"CREATE OR REPLACE VIEW v_match_full_extra AS SELECT 1",
		"SELECT * FROM v_match_full",
	} {
		if reCreateVMatchFull.MatchString(src) {
			t.Errorf("le motif n'aurait pas dû voir : %q", src)
		}
	}
}
