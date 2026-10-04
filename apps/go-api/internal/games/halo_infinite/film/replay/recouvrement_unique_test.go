package replay

// recouvrement_unique_test.go — LE RECOUVREMENT PISTE <-> VIE N'A QU'UNE MESURE : [recouvrementInclus].
//
// CLAUDE.md regle 6 : a la seconde divergence d'un meme calcul, un helper ET un garde-rail qui
// interdit l'ancien calcul. La mesure `min(fin) - max(debut)` etait ecrite en trois endroits du
// nommage, sous deux conventions : bornes incluses pour les xuids et les pistes deduites, bornes
// exclusives pour les bots — et une vie de bot d'un seul echantillon ne nommait pas sa piste. Ce
// fichier ne mesure pas un comportement : il mesure que la SOURCE reste unique.
//
// PORTEE : les sources de production du paquet. Une soustraction ou une comparaison dont un cote
// est un appel a `minI64`/`min` et l'autre un appel a `maxI64`/`max` est la forme interdite, hors
// du corps de `recouvrementInclus`.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// estAppelA dit si l'expression est un appel a l'une des fonctions nommees.
func estAppelA(e ast.Expr, noms ...string) bool {
	c, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	id, ok := c.Fun.(*ast.Ident)
	if !ok {
		return false
	}
	for _, n := range noms {
		if id.Name == n {
			return true
		}
	}
	return false
}

// recouvrementEcritALaMain reconnait `min(..) - max(..)` et `min(..) >= max(..)` (et leurs
// variantes de comparaison), avec ou sans `+ 1` autour.
func recouvrementEcritALaMain(b *ast.BinaryExpr) bool {
	switch b.Op {
	case token.SUB, token.GEQ, token.GTR, token.LEQ, token.LSS:
	default:
		return false
	}
	mins, maxs := []string{"minI64", "min"}, []string{"maxI64", "max"}
	return (estAppelA(b.X, mins...) && estAppelA(b.Y, maxs...)) ||
		(estAppelA(b.X, maxs...) && estAppelA(b.Y, mins...))
}

func TestRecouvrementPisteVieAUneSeuleMesure(t *testing.T) {
	fichiers, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	vus := 0
	for _, f := range fichiers {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f) //nolint:gosec // sources du paquet
		if err != nil {
			t.Fatal(err)
		}
		af, err := parser.ParseFile(fset, f, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range af.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				b, ok := n.(*ast.BinaryExpr)
				if !ok || !recouvrementEcritALaMain(b) {
					return true
				}
				if fd.Name.Name == "recouvrementInclus" {
					vus++
					return true
				}
				t.Errorf("%s : recouvrement ecrit a la main dans %s — passer par recouvrementInclus",
					fset.Position(b.Pos()), fd.Name.Name)
				return true
			})
		}
	}
	if vus != 1 {
		t.Fatalf("recouvrementInclus porte %d mesure(s), attendu 1 : le garde-rail ne voit plus sa cible", vus)
	}
}

// TestRecouvrementInclusCompteUnInstantCommun : la borne commune d'un seul instant recouvre 1.
func TestRecouvrementInclusCompteUnInstantCommun(t *testing.T) {
	l := lifeSpan{from: 1_000, to: 1_000}
	if got := recouvrementInclus(900, 1_100, l); got != 1 {
		t.Errorf("vie d'un instant dans [900,1100] : %d, attendu 1", got)
	}
	if got := recouvrementInclus(1_000, 1_000, l); got != 1 {
		t.Errorf("piste et vie d'un meme instant : %d, attendu 1", got)
	}
	if got := recouvrementInclus(1_001, 1_100, l); got > 0 {
		t.Errorf("piste apres la vie : %d, attendu <= 0", got)
	}
}
