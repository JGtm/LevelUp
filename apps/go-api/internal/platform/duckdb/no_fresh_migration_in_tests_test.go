// Package duckdb — no_fresh_migration_in_tests_test.go : GARDE-RAIL « pas de migration neuve dans
// un test du paquet ».
//
// Une migration complète coûte de 0,6 s à 5 s ; un test qui n'a besoin que du schéma livré copie
// un gabarit déjà migré (migrated_template_test.go : MigratedFileCopy, MigratedTempFile,
// FillMigratedMemory, OpenMigratedMemory). Ce test refuse tout appel à migration.RunForDB ou
// RunForTitleDB dans un *_test.go du paquet, hors de freshMigrationAllowlist.
//
// Une entrée de l'allowlist se justifie par l'OBJET du test (il porte sur la migration elle-même),
// jamais par la commodité, et sa raison commence par une date. L'allowlist ne fait que rétrécir :
// une entrée qui n'appelle plus rien fait échouer le test, elle se supprime.
//
// Hors périmètre du garde-rail (lecture lexicale des appels directs) : les migrations neuves
// indirectes (duckdb.GetOrOpen, migration.ForTarget + ApplySchema, migration.ExecScript).
package duckdb

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// freshMigrationAllowlist : fichiers de test du paquet autorisés à migrer à neuf, avec la raison.
var freshMigrationAllowlist = map[string]string{
	"migrated_template_test.go": "2026-11-17 : construit les gabarits (une migration neuve par cible) et vérifie leur fidélité à une migration neuve.",

	"career_progression_fresh_migrations_test.go": "2026-11-17 : l'objet du test est la promesse « une player DB reconstruite par la seule chaîne de migrations sert le chemin de persistance carrière » ; une copie de gabarit ne la prouverait pas.",

	"csr_pipeline_e2e_integration_test.go": "2026-11-17 : le test de bout en bout exerce l'ordre réel de création d'une base joueur, schéma de synchronisation (EnsurePlayerSchema) puis migrations ; la copie d'un gabarit migré donne une base dont le schéma diffère (colonnes NOT NULL, défauts, types), et ce que prouvent les lecteurs CSR glisserait d'une forme de base à l'autre.",

	"pool_shared_social_recovery_concurrent_test.go": "2026-11-17 : l'objet du test est la réapplication idempotente des migrations shared_social après réouverture (RunForDB rejoué sur la même base) ; le test pose lui-même le premier passage.",
}

// freshMigrationCalls rend, par fichier de test du dossier, les lignes des appels à
// migration.RunForDB / RunForTitleDB (lus dans l'AST : un commentaire ou une chaîne ne comptent pas).
func freshMigrationCalls(t *testing.T, dir string) map[string][]int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", dir, err)
	}
	out := map[string][]int{}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (sel.Sel.Name != "RunForDB" && sel.Sel.Name != "RunForTitleDB") {
				return true
			}
			out[name] = append(out[name], fset.Position(call.Pos()).Line)
			return true
		})
	}
	return out
}

func freshMigrationDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	return filepath.Dir(thisFile)
}

// TestNoFreshMigrationInTests : aucun test du paquet ne migre à neuf hors de l'allowlist, et
// chaque entrée de l'allowlist appelle encore RunForDB ou RunForTitleDB.
func TestNoFreshMigrationInTests(t *testing.T) {
	calls := freshMigrationCalls(t, freshMigrationDir(t))

	var violations []string
	for name, lines := range calls {
		if _, ok := freshMigrationAllowlist[name]; !ok {
			violations = append(violations, name+" (lignes "+joinLines(lines)+")")
		}
	}
	sort.Strings(violations)
	for _, v := range violations {
		t.Errorf("migration neuve dans un test : %s — copier un gabarit (MigratedFileCopy, MigratedTempFile, "+
			"FillMigratedMemory ou OpenMigratedMemory), ou, si le test porte sur la migration elle-même, "+
			"l'ajouter à freshMigrationAllowlist avec une raison datée", v)
	}
	for name := range freshMigrationAllowlist {
		if len(calls[name]) == 0 {
			t.Errorf("%s : l'entrée de freshMigrationAllowlist n'appelle plus RunForDB ni RunForTitleDB — la supprimer", name)
		}
	}
}

// TestFreshMigrationAllowlistDatee : la raison de chaque entrée commence par une date AAAA-MM-JJ
// et la clé est un nom de fichier *_test.go qui existe dans le paquet.
func TestFreshMigrationAllowlistDatee(t *testing.T) {
	forme := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\b`)
	dir := freshMigrationDir(t)
	for name, raison := range freshMigrationAllowlist {
		if !forme.MatchString(raison) {
			t.Errorf("%s : la raison ne commence pas par une date AAAA-MM-JJ : %q", name, raison)
		}
		if !strings.HasSuffix(name, "_test.go") || strings.Contains(name, "/") {
			t.Errorf("%s : la clé doit être un nom de fichier *_test.go du paquet", name)
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s : fichier introuvable dans le paquet : %v", name, err)
		}
	}
}

func joinLines(lines []int) string {
	parts := make([]string, len(lines))
	for i, l := range lines {
		parts[i] = strconv.Itoa(l)
	}
	return strings.Join(parts, ", ")
}

// TestNoFreshMigration_LectureDeLAST : le détecteur voit un appel réel (RunForDB comme
// RunForTitleDB), pas une mention en commentaire ni dans une chaîne, et ignore les fichiers qui
// ne sont pas des tests.
func TestNoFreshMigration_LectureDeLAST(t *testing.T) {
	dir := t.TempDir()
	write := func(name, src string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write("appel_test.go", `package x

func f() {
	migration.RunForDB(nil, "shared")
	migration.RunForTitleDB(nil, "s", "shared")
}
`)
	write("mention_test.go", `package x

// migration.RunForDB(db, t) en commentaire
var s = "migration.RunForDB("
`)
	write("prod.go", `package x

func g() { migration.RunForDB(nil, "shared") }
`)

	got := freshMigrationCalls(t, dir)
	if len(got) != 1 || len(got["appel_test.go"]) != 2 {
		t.Errorf("appels vus = %v, want 2 appels dans appel_test.go et rien d'autre", got)
	}
}
