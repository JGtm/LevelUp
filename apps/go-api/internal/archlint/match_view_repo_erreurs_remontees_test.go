// Package archlint — match_view_repo_erreurs_remontees_test.go : les lectures de la vue match
// (`platform/duckdb/match_view_repo*.go`) ne transforment jamais une défaillance en « pas de
// donnée ».
//
// Une lecture qui échoue puis rend `nil, nil` (ou une structure vide) est indiscernable d'un
// match sans donnée : la vue sert alors une section vide et le web ne dit rien de la panne. La
// panne REMONTE (le service la classe : raison dédiée ou `section_unavailable`), l'absence
// légitime se CONSTATE (`sql.ErrNoRows` testé avant `err != nil`, sonde `relationExists`,
// capability). Ce ratchet interdit, dans une branche `if err != nil` de ces fichiers :
//
//   - un `return` dont la dernière valeur (l'erreur) est `nil`, quand la fonction renvoie un `error` ;
//   - un `continue` sans appel `slog.*` ni compteur (`x++`, `x += n`) dans la même branche.
//
// L'allowlist est VIDE et le reste : une exception est une panne de plus qui se lit comme un vide.
package archlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// nomDErreurRE : un identifiant qui porte une erreur (`err`, `scanErr`, `iterErr`, `perr`).
var nomDErreurRE = regexp.MustCompile(`(?i)err`)

// repliSilencieux : un manquement relevé dans un fichier.
type repliSilencieux struct {
	Ligne int
	Motif string
}

// repliSilencieuxDuFichier relève, dans src, les replis silencieux d'une branche `if err != nil`.
func repliSilencieuxDuFichier(t *testing.T, nom string, src any) []repliSilencieux {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, nom, src, 0)
	if err != nil {
		t.Fatalf("%s : %v", nom, err)
	}
	var out []repliSilencieux
	var visiter func(n ast.Node, renvoieErreur bool)
	visiter = func(n ast.Node, renvoieErreur bool) {
		ast.Inspect(n, func(c ast.Node) bool {
			switch x := c.(type) {
			case *ast.FuncDecl:
				if x.Body != nil {
					visiter(x.Body, derniereValeurEstUneErreur(x.Type))
				}
				return false
			case *ast.FuncLit:
				visiter(x.Body, derniereValeurEstUneErreur(x.Type))
				return false
			case *ast.IfStmt:
				if estTestErreurNonNulle(x.Cond) {
					out = append(out, manquementsDeLaBranche(fset, x.Body, renvoieErreur)...)
				}
			}
			return true
		})
	}
	for _, d := range f.Decls {
		visiter(d, false)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ligne < out[j].Ligne })
	return out
}

func derniereValeurEstUneErreur(ft *ast.FuncType) bool {
	if ft.Results == nil || len(ft.Results.List) == 0 {
		return false
	}
	id, ok := ft.Results.List[len(ft.Results.List)-1].Type.(*ast.Ident)
	return ok && id.Name == "error"
}

// estTestErreurNonNulle : la condition est `<erreur> != nil`, seule ou en tête d'un `&&`.
func estTestErreurNonNulle(cond ast.Expr) bool {
	switch c := cond.(type) {
	case *ast.ParenExpr:
		return estTestErreurNonNulle(c.X)
	case *ast.BinaryExpr:
		if c.Op == token.LAND {
			return estTestErreurNonNulle(c.X) || estTestErreurNonNulle(c.Y)
		}
		if c.Op != token.NEQ {
			return false
		}
		id, ok := c.X.(*ast.Ident)
		nul, okNul := c.Y.(*ast.Ident)
		return ok && okNul && nul.Name == "nil" && nomDErreurRE.MatchString(id.Name)
	}
	return false
}

// manquementsDeLaBranche relève, dans le corps d'une branche d'erreur, les `return ..., nil` et les
// `continue` muets. Les fonctions littérales imbriquées ont leur propre branche d'erreur, relue à part.
func manquementsDeLaBranche(fset *token.FileSet, body *ast.BlockStmt, renvoieErreur bool) []repliSilencieux {
	var out []repliSilencieux
	journalise := brancheJournaliseOuCompte(body)
	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.ReturnStmt:
			if renvoieErreur && len(x.Results) > 0 {
				if id, ok := x.Results[len(x.Results)-1].(*ast.Ident); ok && id.Name == "nil" {
					out = append(out, repliSilencieux{fset.Position(x.Pos()).Line,
						"return avec erreur nil dans une branche `if err != nil` : l'erreur doit remonter, ou l'absence se constater avant (sql.ErrNoRows, relationExists)"})
				}
			}
		case *ast.BranchStmt:
			if x.Tok == token.CONTINUE && !journalise {
				out = append(out, repliSilencieux{fset.Position(x.Pos()).Line,
					"continue sans slog ni compteur dans une branche `if err != nil` : une ligne illisible se dit"})
			}
		}
		return true
	})
	return out
}

// brancheJournaliseOuCompte : la branche appelle `slog.*` ou incrémente un compteur.
func brancheJournaliseOuCompte(body *ast.BlockStmt) bool {
	vu := false
	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
				if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "slog" {
					vu = true
				}
			}
		case *ast.IncDecStmt:
			vu = true
		case *ast.AssignStmt:
			if x.Tok == token.ADD_ASSIGN {
				vu = true
			}
		}
		return !vu
	})
	return vu
}

func TestLecturesDeLaVueMatch_AucunRepliSilencieuxSurErreur(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	fichiers, err := filepath.Glob(filepath.Join(goAPIRoot, "internal", "platform", "duckdb", "match_view_repo*.go"))
	if err != nil {
		t.Fatalf("glob : %v", err)
	}
	lus := 0
	for _, f := range fichiers {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		lus++
		for _, m := range repliSilencieuxDuFichier(t, f, nil) {
			t.Errorf("%s:%d : %s", filepath.ToSlash(f), m.Ligne, m.Motif)
		}
	}
	if lus < 8 {
		t.Fatalf("ratchet vacant : %d fichier(s) lu(s)", lus)
	}
}

// Le détecteur lui-même : il rougit sur les deux replis et laisse passer les formes saines.
func TestRepliSilencieux_Detecteur(t *testing.T) {
	const mauvais = `package p
import "log/slog"
func a() ([]int, error) {
	x, err := f()
	if err != nil {
		slog.Warn("x")
		return nil, nil
	}
	return x, nil
}
func b(rows R) {
	for rows.Next() {
		if err := rows.Scan(); err != nil {
			continue
		}
	}
}
`
	if got := repliSilencieuxDuFichier(t, "mauvais.go", mauvais); len(got) != 2 {
		t.Fatalf("attendu 2 manquements, obtenu %v", got)
	}
	const sain = `package p
import ("errors"; "database/sql"; "log/slog")
func a() ([]int, error) {
	x, err := f()
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return x, nil
}
func b(rows R) (n int) {
	for rows.Next() {
		if err := rows.Scan(); err != nil {
			slog.Error("x", "err", err)
			continue
		}
		if err := rows.Scan(); err != nil {
			n++
			continue
		}
	}
	return n
}
func c() *int {
	if err := g(); err != nil {
		return nil
	}
	return nil
}
`
	if got := repliSilencieuxDuFichier(t, "sain.go", sain); len(got) != 0 {
		t.Fatalf("formes saines refusées : %v", got)
	}
}
