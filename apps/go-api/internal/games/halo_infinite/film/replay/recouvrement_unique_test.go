package replay

// recouvrement_unique_test.go — LE RECOUVREMENT PISTE <-> VIE N'A QU'UNE MESURE : [recouvrementInclus].
//
// CLAUDE.md regle 6 : a la seconde divergence d'un meme calcul, un helper ET un garde-rail qui
// interdit l'ancien calcul. La mesure `min(fin) - max(debut)` etait ecrite en trois endroits du
// nommage, sous deux conventions : bornes incluses pour les xuids et les pistes deduites, bornes
// exclusives pour les bots — et une vie de bot d'un seul echantillon ne nommait pas sa piste. Ce
// fichier ne mesure pas un comportement : il mesure que la SOURCE reste unique.
//
// # CE QU'IL RECONNAIT (sources de production du paquet, hors du corps de `recouvrementInclus`)
//
//	F1  `min(..) - max(..)` et ses comparaisons (`<`, `<=`, `>`, `>=`), dans les deux ordres, avec
//	    `min`/`minI64` et `max`/`maxI64` ;
//	F2  la meme forme quand un cote est une VARIABLE de la fonction affectee d'un tel appel portant
//	    une borne de vie en argument (`mn := minI64(a, l.to)` puis `mn < maxI64(de, l.from)`) ;
//	F3  les bornes croisees : `X.from <= B && A <= X.to` (intersection non vide) et sa negation
//	    `B < X.from || X.to < A` (intersection vide), tous sens de comparaison, quand `A` et `B`
//	    different — `X.from <= t && t <= X.to` est l'appartenance d'un INSTANT, pas un recouvrement.
//
// # CE QU'IL NE RECONNAIT PAS
//
// Il est SYNTAXIQUE : une mesure ecrite autrement (variables de bornes intermediaires pour F3,
// helper local qui enveloppe min/max, comparaison sur des champs nommes autrement que `from`/`to`)
// lui echappe. Il lit les sources SUR LE DISQUE (`go/parser`) : une substitution passee par
// `go test -overlay` ne l'atteint pas — une mutation du garde-rail se joue en modifiant le fichier.

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

// chevauchementsDeViesAdmis : fonctions ou F3 est admis. Chaque entree porte sa raison.
var chevauchementsDeViesAdmis = map[string]string{
	// Vie <-> vie (exclusion de candidats), pas piste <-> vie : le predicat d'intersection a bornes
	// incluses, sans mesure. Meme convention que recouvrementInclus.
	"seChevauchent": "vie <-> vie, predicat a bornes incluses",
}

var (
	nomsMin = []string{"minI64", "min"}
	nomsMax = []string{"maxI64", "max"}
)

// estAppelA dit si l'expression est un appel a l'une des fonctions nommees.
func estAppelA(e ast.Expr, noms ...string) bool {
	c, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	id, ok := c.Fun.(*ast.Ident)
	return ok && contientNom(noms, id.Name)
}

func contientNom(noms []string, n string) bool {
	for _, x := range noms {
		if x == n {
			return true
		}
	}
	return false
}

// porteUneBorneDeVie dit si l'un des arguments de l'appel est un champ `from` ou `to`.
func porteUneBorneDeVie(e ast.Expr) bool {
	c, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	for _, a := range c.Args {
		if s, ok := a.(*ast.SelectorExpr); ok && (s.Sel.Name == "from" || s.Sel.Name == "to") {
			return true
		}
	}
	return false
}

// liaisonsMinMax releve, dans une fonction, les variables affectees d'un appel min (resp. max)
// sur une borne de vie (F2). Les bornages d'autres intervalles (frames d'une rafale, periodes
// d'une colline) n'y entrent pas : ce ne sont pas des recouvrements piste <-> vie.
func liaisonsMinMax(corps *ast.BlockStmt) (mins, maxs map[string]bool) {
	mins, maxs = map[string]bool{}, map[string]bool{}
	lier := func(nom *ast.Ident, v ast.Expr) {
		if !porteUneBorneDeVie(v) {
			return
		}
		switch {
		case estAppelA(v, nomsMin...):
			mins[nom.Name] = true
		case estAppelA(v, nomsMax...):
			maxs[nom.Name] = true
		}
	}
	ast.Inspect(corps, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.AssignStmt:
			if len(s.Lhs) == len(s.Rhs) {
				for i, g := range s.Lhs {
					if id, ok := g.(*ast.Ident); ok {
						lier(id, s.Rhs[i])
					}
				}
			}
		case *ast.ValueSpec:
			if len(s.Names) == len(s.Values) {
				for i, id := range s.Names {
					lier(id, s.Values[i])
				}
			}
		}
		return true
	})
	return mins, maxs
}

// formeMinMax reconnait F1 et F2.
func formeMinMax(b *ast.BinaryExpr, mins, maxs map[string]bool) bool {
	switch b.Op {
	case token.SUB, token.GEQ, token.GTR, token.LEQ, token.LSS:
	default:
		return false
	}
	lie := func(e ast.Expr, liees map[string]bool, noms []string) bool {
		if id, ok := e.(*ast.Ident); ok {
			return liees[id.Name]
		}
		return estAppelA(e, noms...)
	}
	estMin := func(e ast.Expr) bool { return lie(e, mins, nomsMin) }
	estMax := func(e ast.Expr) bool { return lie(e, maxs, nomsMax) }
	return (estMin(b.X) && estMax(b.Y)) || (estMax(b.X) && estMin(b.Y))
}

// petitGrand normalise une comparaison en (petit, grand) ; faux si ce n'en est pas une.
func petitGrand(e ast.Expr) (petit, grand ast.Expr, ok bool) {
	b, est := e.(*ast.BinaryExpr)
	if !est {
		return nil, nil, false
	}
	switch b.Op {
	case token.LEQ, token.LSS:
		return b.X, b.Y, true
	case token.GEQ, token.GTR:
		return b.Y, b.X, true
	}
	return nil, nil, false
}

// borne rend la base `X` quand l'expression est `X.<champ>`.
func borne(e ast.Expr, champ string) (string, bool) {
	s, ok := e.(*ast.SelectorExpr)
	if !ok || s.Sel.Name != champ {
		return "", false
	}
	return types.ExprString(s.X), true
}

// formeBornesCroisees reconnait F3 sur un `&&` (intersection non vide) ou un `||` (vide).
func formeBornesCroisees(b *ast.BinaryExpr) bool {
	if b.Op != token.LAND && b.Op != token.LOR {
		return false
	}
	p1, g1, ok1 := petitGrand(b.X)
	p2, g2, ok2 := petitGrand(b.Y)
	if !ok1 || !ok2 {
		return false
	}
	croise := func(pa, ga, pb, gb ast.Expr) bool {
		if b.Op == token.LAND { // X.from <= ga && pb <= X.to
			xa, okA := borne(pa, "from")
			xb, okB := borne(gb, "to")
			return okA && okB && xa == xb && types.ExprString(ga) != types.ExprString(pb)
		} // ga < X.from || X.to < pb
		xa, okA := borne(ga, "from")
		xb, okB := borne(pb, "to")
		return okA && okB && xa == xb && types.ExprString(pa) != types.ExprString(gb)
	}
	return croise(p1, g1, p2, g2) || croise(p2, g2, p1, g1)
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
			mins, maxs := liaisonsMinMax(fd.Body)
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				b, ok := n.(*ast.BinaryExpr)
				if !ok {
					return true
				}
				switch {
				case formeMinMax(b, mins, maxs) && fd.Name.Name == "recouvrementInclus":
					vus++
				case formeMinMax(b, mins, maxs):
					t.Errorf("%s : recouvrement ecrit a la main (min/max) dans %s — passer par recouvrementInclus",
						fset.Position(b.Pos()), fd.Name.Name)
				case formeBornesCroisees(b) && chevauchementsDeViesAdmis[fd.Name.Name] == "":
					t.Errorf("%s : recouvrement ecrit a la main (bornes croisees) dans %s — passer par "+
						"recouvrementInclus", fset.Position(b.Pos()), fd.Name.Name)
				}
				return true
			})
		}
	}
	if vus != 1 {
		t.Fatalf("recouvrementInclus porte %d mesure(s), attendu 1 : le garde-rail ne voit plus sa cible", vus)
	}
}

// TestRecouvrementGardeRailReconnaitSesFormes : chaque forme de l'en-tete est reconnue, et
// l'appartenance d'un instant ne l'est pas. Le garde-rail lit le disque : ce test tient ses
// formes sans passer par lui.
func TestRecouvrementGardeRailReconnaitSesFormes(t *testing.T) {
	cas := []struct {
		nom, src string
		want     bool
	}{
		{"F1", `func f() { _ = minI64(a, l.to) - maxI64(de, l.from) }`, true},
		{"F1-comparaison", `func f() { _ = max(de, l.from) <= min(a, l.to) }`, true},
		{"F2", `func f() { mn := minI64(a, l.to); _ = mn < maxI64(de, l.from) }`, true},
		{"F2-var", `func f() { var mx = max(de, l.from); _ = min(a, l.to) - mx }`, true},
		{"F3", `func f() { _ = l.from <= to && from <= l.to }`, true},
		{"F3-inverse", `func f() { _ = to >= l.from && l.to >= from }`, true},
		{"F3-vide", `func f() { _ = to < l.from || l.to < from }`, true},
		{"instant", `func f() { _ = l.from <= t && t <= l.to }`, false},
		{"instant-exclu", `func f() { _ = t < l.from || t > l.to }`, false},
		{"union", `func f() { de, a := min(t, l.from), max(t, l.to); _ = de < 0; _ = a }`, false},
		{"bornage-hors-vie", `func f() { t0, t1 := max(r.t0, p.t0), min(r.t1, p.t1); _ = t0 > t1 }`, false},
	}
	for _, c := range cas {
		af, err := parser.ParseFile(token.NewFileSet(), c.nom, "package p\n"+c.src, 0)
		if err != nil {
			t.Fatalf("%s : %v", c.nom, err)
		}
		fd := af.Decls[0].(*ast.FuncDecl)
		mins, maxs := liaisonsMinMax(fd.Body)
		got := false
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			if b, ok := n.(*ast.BinaryExpr); ok && (formeMinMax(b, mins, maxs) || formeBornesCroisees(b)) {
				got = true
			}
			return true
		})
		if got != c.want {
			t.Errorf("%s : reconnu %v, attendu %v", c.nom, got, c.want)
		}
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
