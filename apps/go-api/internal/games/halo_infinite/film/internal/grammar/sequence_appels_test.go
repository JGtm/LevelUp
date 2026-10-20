package grammar

// sequence_appels_test.go — LA SUITE DES APPELS d une fonction de production, l instrument commun
// des garde-rails de lecteur unique (`lecteur_minuteur_guard_test.go`,
// `lecteur_jeu_darmes_guard_test.go`). Un garde-rail qui cherche une sequence de lectures dans le
// TEXTE depend de la mise en page (une ligne par lecture) : la meme copie ecrite sur une ligne lui
// echappe. Celui-ci la cherche dans l arbre syntaxique : l ordre des appels est celui du source,
// les blancs, retours a la ligne et commentaires n y comptent pas.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// appelLu : un appel, reduit au nom appele (l identifiant, ou le selecteur final : `ReadBits`
// pour `br.ReadBits`), au texte de ses arguments, blancs retires, et a son bloc. Un appel dont la
// fonction n a pas de nom (litteral de fonction, indexation) porte le nom `?`. Deux appels de meme
// `bloc` sont dans la meme liste d instructions : une suite de lectures en ligne y tient, une
// suite qui sort d un `if` (lectures sous porte, puis lecture inconditionnelle) n y tient pas.
type appelLu struct {
	nom, args string
	bloc      int
}

// fichierDAppels : les suites d appels d un fichier de production, une par fonction declaree.
type fichierDAppels struct {
	nom       string
	fonctions [][]appelLu
}

// appelsDeLaProduction rend les suites d appels de chaque fichier de production du paquet (les
// `_test.go` exclus), dans l ordre du source. Les conversions vers un type predeclare
// (`uint32(x)`) ne sont pas des appels.
func appelsDeLaProduction(t *testing.T) []fichierDAppels {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob : %v", err)
	}
	var out []fichierDAppels
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		out = append(out, fichierDAppels{nom: f, fonctions: appelsDuFichier(t, f)})
	}
	if len(out) == 0 {
		t.Fatal("aucun fichier Go de production vu : le garde-rail ne garde rien")
	}
	return out
}

// appelsDuFichier rend la suite des appels de chaque fonction declaree de `f`.
func appelsDuFichier(t *testing.T, f string) [][]appelLu {
	t.Helper()
	src, err := os.ReadFile(f) //nolint:gosec // fichiers du paquet lui-meme
	if err != nil {
		t.Fatalf("lecture de %s : %v", f, err)
	}
	return appelsDuSource(t, f, src)
}

// appelsDuSource rend la suite des appels de chaque fonction declaree du source `src` (nomme `f`
// dans les messages) : la meme analyse que [appelsDuFichier], pour les vecteurs des garde-rails.
func appelsDuSource(t *testing.T, f string, src []byte) [][]appelLu {
	t.Helper()
	fset := token.NewFileSet()
	fichier, err := parser.ParseFile(fset, f, src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("analyse de %s : %v", f, err)
	}
	var out [][]appelLu
	for _, d := range fichier.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		out = append(out, appelsDuCorps(fset, src, fd.Body))
	}
	return out
}

// appelsDuCorps rend la suite des appels de `corps` dans l ordre du source, chacun marque de son
// bloc (le plus proche `{ ... }` ou `case` qui l enclot, numerote a l entree).
func appelsDuCorps(fset *token.FileSet, src []byte, corps *ast.BlockStmt) []appelLu {
	var seq []appelLu
	var pile []ast.Node
	var blocs []int
	numero := 0
	ast.Inspect(corps, func(n ast.Node) bool {
		if n == nil {
			switch pile[len(pile)-1].(type) {
			case *ast.BlockStmt, *ast.CaseClause, *ast.CommClause:
				blocs = blocs[:len(blocs)-1]
			}
			pile = pile[:len(pile)-1]
			return true
		}
		pile = append(pile, n)
		switch n := n.(type) {
		case *ast.BlockStmt, *ast.CaseClause, *ast.CommClause:
			numero++
			blocs = append(blocs, numero)
		case *ast.CallExpr:
			if a, garde := appelDe(fset, src, n); garde {
				a.bloc = blocs[len(blocs)-1]
				seq = append(seq, a)
			}
		}
		return true
	})
	return seq
}

// appelDe reduit `c` a un [appelLu] ; faux pour une conversion vers un type predeclare.
func appelDe(fset *token.FileSet, src []byte, c *ast.CallExpr) (appelLu, bool) {
	nom := "?"
	switch fn := c.Fun.(type) {
	case *ast.Ident:
		if _, estType := types.Universe.Lookup(fn.Name).(*types.TypeName); estType {
			return appelLu{}, false
		}
		nom = fn.Name
	case *ast.SelectorExpr:
		nom = fn.Sel.Name
	}
	args := make([]string, len(c.Args))
	for i, a := range c.Args {
		texte := string(src[fset.Position(a.Pos()).Offset:fset.Position(a.End()).Offset])
		args[i] = strings.Join(strings.Fields(texte), "")
	}
	return appelLu{nom: nom, args: strings.Join(args, ",")}, true
}

// estAppel dit si `a` appelle `nom` avec l un des arguments `args` (tous si aucun n est donne).
func estAppel(a appelLu, nom string, args ...string) bool {
	if a.nom != nom {
		return false
	}
	if len(args) == 0 {
		return true
	}
	for _, x := range args {
		if a.args == x {
			return true
		}
	}
	return false
}

// memeBloc dit si les appels `seq` sont tous dans le meme bloc.
func memeBloc(seq ...appelLu) bool {
	for _, a := range seq {
		if a.bloc != seq[0].bloc {
			return false
		}
	}
	return true
}
