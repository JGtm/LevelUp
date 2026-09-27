// Package archlint — no_local_sql_splitter_test.go : un seul découpeur de script SQL
// (backlog 2026-09-26, lot B2 ; CLAUDE.md règle n°6).
//
// Le découpeur canonique vit dans internal/migration/helpers.go (splitSQL, exporté en
// migration.SplitSQL) et l'exécuteur de script dans le même fichier (execScriptContext,
// exporté en migration.ExecScriptContext / migration.ExecScript). Avant ce lot, QUATRE
// découpeurs exécutaient du SQL : migration, sync/schema.go, sync/skill (copie de test) et
// cmd/diag_exec. Ils n'avaient pas la même sémantique : la copie de sync passait à DuckDB
// un fragment fait uniquement de commentaires (« empty query », CI du 2026-09-20).
//
// Interdit, hors internal/migration/helpers*.go :
//   - toute fonction nommée comme un découpeur ou un exécuteur de script SQL
//     (splitSQL, splitStatements, execScript, runSQLScript…), sauf un délégué d'une ligne
//     `return migration.X(...)` (cas de sync.execScript, qui garde ses appels de test) ;
//   - dans un fichier qui exécute du SQL : `strings.Split*(x, ";")`, et toute comparaison
//     à l'octet `';'` (boucle de découpage faite main).
package archlint

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

// decoupeurSQLCanonique : fichiers où vit le cœur (et ses tests).
var decoupeurSQLCanonique = regexp.MustCompile(`^internal/migration/helpers[^/]*\.go$`)

// decoupeurSQLDispenses : fichiers dispensés, avec date et raison.
var decoupeurSQLDispenses = map[string]string{
	// 2026-09-26 : analyse un DDL d'un garde-rail de schéma sans jamais l'exécuter (la
	// découpe grossière sur « ; » est voulue : le texte a déjà perdu ses commentaires).
	"internal/migration/arbitration_clocks_utc_guard_test.go": "garde-rail d'analyse, n'exécute pas de SQL",
}

// nomDeDecoupeurSQL : noms de fonctions qui déclarent un découpeur ou un exécuteur de script.
var nomDeDecoupeurSQL = regexp.MustCompile(`(?i)^(split(sql|statements?|script)|(exec|run|apply)(sql)?script)`)

// methodesQuiExecutentDuSQL : méthodes de database/sql (ou équivalents) qui exécutent du SQL.
var methodesQuiExecutentDuSQL = map[string]bool{
	"Exec": true, "ExecContext": true, "QueryContext": true,
	"QueryRow": true, "QueryRowContext": true, "Query": true,
}

var fonctionsDeDecoupeDeChaine = map[string]bool{
	"Split": true, "SplitN": true, "SplitAfter": true, "SplitAfterN": true,
}

func TestNoLocalSQLSplitter(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	var violations []string
	lus, canoniqueVu := 0, false
	for _, sub := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(goAPIRoot, sub), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			lus++
			if decoupeurSQLCanonique.MatchString(rel) {
				canoniqueVu = canoniqueVu || rel == "internal/migration/helpers.go"
				return nil
			}
			if _, dispense := decoupeurSQLDispenses[rel]; dispense {
				return nil
			}
			bas := strings.ToLower(string(data))
			if !strings.Contains(bas, "split") && !strings.Contains(bas, "script") && !strings.Contains(bas, "';'") {
				return nil
			}
			fset := token.NewFileSet()
			f, perr := parser.ParseFile(fset, path, data, 0)
			if perr != nil {
				t.Errorf("%s : analyse impossible : %v", rel, perr)
				return nil
			}
			violations = append(violations, decoupeursLocaux(fset, f, rel)...)
			return nil
		})
		if err != nil {
			t.Fatalf("parcours de %s/ : %v", sub, err)
		}
	}
	if lus == 0 {
		t.Fatal("aucun fichier .go lu sous internal/ et cmd/ : le garde-rail ne voit rien")
	}
	for rel := range decoupeurSQLDispenses {
		if _, err := os.Stat(filepath.Join(goAPIRoot, filepath.FromSlash(rel))); err != nil {
			t.Errorf("dispense périmée : %s n'existe plus (%v), la retirer de decoupeurSQLDispenses", rel, err)
		}
	}
	if !canoniqueVu {
		t.Fatal("internal/migration/helpers.go introuvable : le découpeur canonique a bougé, mettre à jour ce garde-rail")
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Errorf("découpeur de script SQL local interdit (un seul découpeur : migration.SplitSQL / "+
			"migration.ExecScriptContext, backlog B2) :\n  %s", strings.Join(violations, "\n  "))
	}
}

// decoupeursLocaux rend les violations d'un fichier : définitions nommées hors délégation,
// puis découpes sur « ; » dans un fichier qui exécute du SQL.
func decoupeursLocaux(fset *token.FileSet, f *ast.File, rel string) []string {
	var out []string
	site := func(n ast.Node, quoi string) {
		out = append(out, rel+":"+strconv.Itoa(fset.Position(n.Pos()).Line)+"  "+quoi)
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && nomDeDecoupeurSQL.MatchString(fn.Name.Name) && !delegueAMigration(fn) {
			site(fn, "définition de "+fn.Name.Name)
		}
	}
	if !executeDuSQL(f) {
		return out
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if decoupeSurPointVirgule(x) {
				site(x, "strings.Split*(…, \";\") dans un fichier qui exécute du SQL")
			}
		case *ast.BinaryExpr:
			if (x.Op == token.EQL || x.Op == token.NEQ) && (estOctetPointVirgule(x.X) || estOctetPointVirgule(x.Y)) {
				site(x, "comparaison à ';' dans un fichier qui exécute du SQL")
			}
		case *ast.CaseClause:
			for _, e := range x.List {
				if estOctetPointVirgule(e) {
					site(e, "case ';' dans un fichier qui exécute du SQL")
				}
			}
		}
		return true
	})
	return out
}

// delegueAMigration : le corps est exactement `return migration.X(...)`.
func delegueAMigration(fn *ast.FuncDecl) bool {
	if fn.Body == nil || len(fn.Body.List) != 1 {
		return false
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return false
	}
	call, ok := ret.Results[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "migration"
}

// executeDuSQL : le fichier appelle une méthode qui exécute du SQL (Query sans argument,
// comme url.URL.Query(), n'en est pas une).
func executeDuSQL(f *ast.File) bool {
	trouve := false
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || trouve {
			return !trouve
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && methodesQuiExecutentDuSQL[sel.Sel.Name] && len(call.Args) > 0 {
			trouve = true
		}
		return !trouve
	})
	return trouve
}

func decoupeSurPointVirgule(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !fonctionsDeDecoupeDeChaine[sel.Sel.Name] || len(call.Args) < 2 {
		return false
	}
	if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "strings" {
		return false
	}
	lit, ok := call.Args[1].(*ast.BasicLit)
	return ok && lit.Kind == token.STRING && lit.Value == `";"`
}

func estOctetPointVirgule(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	return ok && lit.Kind == token.CHAR && lit.Value == `';'`
}
