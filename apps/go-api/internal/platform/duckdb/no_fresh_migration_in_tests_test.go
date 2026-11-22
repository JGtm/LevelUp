// Package duckdb — no_fresh_migration_in_tests_test.go : GARDE-RAIL « pas de migration neuve dans
// un test du paquet ».
//
// Une migration complète coûte de 0,6 s à 5 s ; un test qui n'a besoin que du schéma livré copie
// un gabarit déjà migré (migrated_template_test.go : MigratedFileCopy, MigratedTempFile,
// FillMigratedMemory, OpenMigratedMemory). Ce test refuse, dans un *_test.go du paquet hors de
// freshMigrationAllowlist :
//
//   - tout usage de migration.RunForDB ou RunForTitleDB, appelé OU pris en valeur
//     (f := migration.RunForDB) : l'AST est lu sur les sélecteurs, pas sur les seuls appels ;
//   - tout usage de GetOrOpen dans un fichier qui ne copie aucun gabarit (MigratedFileCopy ou
//     MigratedTempFile) : GetOrOpen joue la chaîne de migrations player sur un fichier absent, et
//     la lecture ne décide pas si le chemin passé existe déjà. La règle porte sur le fichier : un
//     fichier qui copie le gabarit pour un chemin et en passe un autre à GetOrOpen n'est pas vu.
//
// Une entrée de l'allowlist se justifie par l'OBJET du test (il porte sur la migration elle-même),
// jamais par la commodité, et sa raison commence par une date. L'allowlist ne fait que rétrécir :
// une entrée qui n'utilise plus rien de ce qui précède fait échouer le test, elle se supprime.
//
// Hors périmètre, gain nul : les deux migration.ForTarget + ApplySchema
// (leaderboard_world_repo_test.go, world_player_stats_repo_test.go) et l'ExecScript
// (weapon_names_testutil_test.go) n'appliquent que quelques étapes ; une copie de gabarit ne
// ferait rien gagner de mesurable.
package duckdb

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// freshMigrationAllowlist : fichiers de test du paquet autorisés à migrer à neuf, avec la raison.
var freshMigrationAllowlist = map[string]string{
	"migrated_template_test.go": "2026-11-17 : construit les gabarits (une migration neuve par cible) et vérifie leur fidélité à une migration neuve.",

	"career_progression_fresh_migrations_test.go": "2026-11-17 : l'objet du test est la promesse « une player DB reconstruite par la seule chaîne de migrations sert le chemin de persistance carrière » ; une copie de gabarit ne la prouverait pas.",

	"pool_migration_test.go": "2026-11-18 : l'objet du test est la migration que GetOrOpen joue sur une base héritée fabriquée à la main (schéma partiel, colonne absente) ; une copie de gabarit, déjà à jour, ne laisserait rien à migrer.",

	"pool_shared_social_recovery_concurrent_test.go": "2026-11-17 : l'objet du test est la réapplication idempotente des migrations shared_social après réouverture (RunForDB rejoué sur la même base) ; le test pose lui-même le premier passage.",
}

// nomsDeMigration : les fonctions de migration que le garde-rail suit.
var nomsDeMigration = []string{"RunForDB", "RunForTitleDB"}

// nomsDOuverture : l'ouverture du pool joueur, qui migre un fichier absent.
var nomsDOuverture = []string{"GetOrOpen"}

// nomsDeCopie : les aides qui posent un gabarit migré sur un fichier.
var nomsDeCopie = []string{"MigratedFileCopy", "MigratedTempFile"}

// appelsDuPaquet : pour chaque fichier de test, les lignes où un nom suivi est utilisé.
type appelsDuPaquet struct {
	migrations map[string][]int // RunForDB, RunForTitleDB
	ouvertures map[string][]int // GetOrOpen
	copies     map[string][]int // MigratedFileCopy, MigratedTempFile
}

// lireLesAppels lit l'AST de chaque *_test.go du dossier (un commentaire ou une chaîne ne
// comptent pas). Un nom compte quand il est le sélecteur d'une expression (`pkg.Nom`, appelé ou
// pris en valeur) ou un identifiant seul qui n'est ni un sélecteur ni le nom d'une déclaration de
// fonction (appel dans le paquet lui-même).
func lireLesAppels(t *testing.T, dir string) appelsDuPaquet {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", dir, err)
	}
	out := appelsDuPaquet{map[string][]int{}, map[string][]int{}, map[string][]int{}}
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
		ecartes := map[*ast.Ident]bool{}
		note := func(id *ast.Ident, pos token.Pos) {
			ligne := fset.Position(pos).Line
			switch {
			case slices.Contains(nomsDeMigration, id.Name):
				out.migrations[name] = append(out.migrations[name], ligne)
			case slices.Contains(nomsDOuverture, id.Name):
				out.ouvertures[name] = append(out.ouvertures[name], ligne)
			case slices.Contains(nomsDeCopie, id.Name):
				out.copies[name] = append(out.copies[name], ligne)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				ecartes[x.Name] = true
			case *ast.SelectorExpr:
				ecartes[x.Sel] = true
				note(x.Sel, x.Sel.Pos())
			case *ast.Ident:
				if !ecartes[x] {
					note(x, x.Pos())
				}
			}
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

// violationsDeMigrationNeuve rend, triées, les fichiers hors allowlist qui migrent à neuf ou
// ouvrent le pool sans copier de gabarit, et les entrées de l'allowlist qui n'utilisent plus rien.
func violationsDeMigrationNeuve(a appelsDuPaquet, allowlist map[string]string) (violations, perimees []string) {
	for name, lines := range a.migrations {
		if _, ok := allowlist[name]; !ok {
			violations = append(violations, name+" : migration neuve (lignes "+joinLines(lines)+")")
		}
	}
	for name, lines := range a.ouvertures {
		if _, ok := allowlist[name]; ok || len(a.copies[name]) > 0 {
			continue
		}
		violations = append(violations, name+" : GetOrOpen sans copie de gabarit (lignes "+joinLines(lines)+")")
	}
	for name := range allowlist {
		if len(a.migrations[name]) == 0 && len(a.ouvertures[name]) == 0 {
			perimees = append(perimees, name)
		}
	}
	sort.Strings(violations)
	sort.Strings(perimees)
	return violations, perimees
}

// TestNoFreshMigrationInTests : aucun test du paquet ne migre à neuf ni n'ouvre le pool sans
// gabarit hors de l'allowlist, et chaque entrée de l'allowlist utilise encore ce qu'elle exempte.
func TestNoFreshMigrationInTests(t *testing.T) {
	violations, perimees := violationsDeMigrationNeuve(lireLesAppels(t, freshMigrationDir(t)), freshMigrationAllowlist)
	for _, v := range violations {
		t.Errorf("migration neuve dans un test : %s — copier un gabarit (MigratedFileCopy, MigratedTempFile, "+
			"FillMigratedMemory ou OpenMigratedMemory) avant d'ouvrir, ou, si le test porte sur la migration "+
			"elle-même, l'ajouter à freshMigrationAllowlist avec une raison datée", v)
	}
	for _, name := range perimees {
		t.Errorf("%s : l'entrée de freshMigrationAllowlist n'utilise plus RunForDB, RunForTitleDB ni GetOrOpen — la supprimer", name)
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

// TestNoFreshMigration_LectureDeLAST : le détecteur voit un usage réel (appelé ou pris en valeur,
// RunForDB comme RunForTitleDB, GetOrOpen qualifié ou non), pas une mention en commentaire ni dans
// une chaîne, ignore les fichiers qui ne sont pas des tests, et décide à partir de ce qu'il lit.
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
	write("valeur_test.go", `package x

var g = migration.RunForTitleDB

func h() {
	f := migration.RunForDB
	_ = f
}
`)
	write("mention_test.go", `package x

// migration.RunForDB(db, t) et duckdb.GetOrOpen(ctx, cfg) en commentaire
var s = "migration.RunForDB( GetOrOpen("
`)
	write("prod.go", `package x

func g() { migration.RunForDB(nil, "shared"); duckdb.GetOrOpen(nil, nil) }
`)
	write("ouvre_seul_test.go", `package x

func a() { duckdb.GetOrOpen(ctx, cfg) }
`)
	write("ouvre_dans_le_paquet_test.go", `package x

func a() { GetOrOpen(ctx, cfg) }

func GetOrOpenAutre() {}
`)
	write("ouvre_valeur_test.go", `package x

func a() { ouvrir := duckdb.GetOrOpen; _ = ouvrir }
`)
	write("ouvre_avec_copie_test.go", `package x

func a() {
	duckdb.MigratedFileCopy(t, target, path)
	duckdb.GetOrOpen(ctx, cfg)
}
`)
	write("ouvre_avec_copie_temp_test.go", `package x

func a() {
	p := duckdb.MigratedTempFile(t, target, "p.duckdb")
	duckdb.GetOrOpen(ctx, cfg(p))
}
`)
	write("ouvre_exempt_test.go", `package x

func a() { duckdb.GetOrOpen(ctx, cfg) }
`)
	write("declare_test.go", `package x

func MigratedFileCopy() {}

func GetOrOpen() {}
`)

	got := lireLesAppels(t, dir)
	if n := len(got.migrations["appel_test.go"]); n != 2 {
		t.Errorf("appels de migration vus dans appel_test.go = %d, want 2", n)
	}
	if n := len(got.migrations["valeur_test.go"]); n != 2 {
		t.Errorf("migrations prises en valeur vues dans valeur_test.go = %d, want 2", n)
	}
	for _, rien := range []string{"mention_test.go", "prod.go", "declare_test.go"} {
		if len(got.migrations[rien])+len(got.ouvertures[rien])+len(got.copies[rien]) != 0 {
			t.Errorf("%s : rien n'est un usage réel, vu migrations=%v ouvertures=%v copies=%v",
				rien, got.migrations[rien], got.ouvertures[rien], got.copies[rien])
		}
	}

	violations, perimees := violationsDeMigrationNeuve(got, map[string]string{
		"ouvre_exempt_test.go": "2026-11-18 : exemple",
		"absent_test.go":       "2026-11-18 : entrée qui ne lit plus rien",
	})
	var noms []string
	for _, v := range violations {
		noms = append(noms, strings.SplitN(v, " : ", 2)[0])
	}
	want := []string{"appel_test.go", "ouvre_dans_le_paquet_test.go", "ouvre_seul_test.go", "ouvre_valeur_test.go", "valeur_test.go"}
	if !slices.Equal(noms, want) {
		t.Errorf("fichiers en violation = %v, want %v", noms, want)
	}
	if !slices.Equal(perimees, []string{"absent_test.go"}) {
		t.Errorf("entrées périmées = %v, want [absent_test.go]", perimees)
	}
}
