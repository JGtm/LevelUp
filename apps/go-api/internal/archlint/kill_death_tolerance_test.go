// Package archlint — kill_death_tolerance_test.go : UNE SEULE FENÊTRE D APPARIEMENT KILL <-> MORT.
//
// Cinq appelants de `analysis.ComputeKillerVictimPairs` portaient chacun leur constante de 5 ms
// (`creditToleranceMS`, `phantomPairToleranceMS`, `killPairToleranceMs`, deux `toleranceMS`
// locales). Une valeur qui diverge en un point rend un jeu de couples différent des autres sans
// que rien ne le signale. La fenêtre vit dans `analysis.KillDeathPairToleranceMS`, et ce ratchet
// exige que tout appel la passe.
package archlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestComputeKillerVictimPairsPrendLaFenetreCanonique — tout appel hors tests passe
// `analysis.KillDeathPairToleranceMS` en second argument.
//
// Mutation qui doit le faire rougir : remettre `const toleranceMS = int64(5)` dans
// `internal/sync/collect.go` et la passer à l appel.
func TestComputeKillerVictimPairsPrendLaFenetreCanonique(t *testing.T) {
	goAPIRoot := racineGoAPI(t)
	var fautifs []string
	appels := 0
	balayerSourcesGo(t, goAPIRoot, func(rel, texte string) {
		if strings.HasSuffix(rel, "_test.go") || !strings.Contains(texte, "ComputeKillerVictimPairs(") {
			return
		}
		f, err := parser.ParseFile(token.NewFileSet(), rel, texte, 0)
		if err != nil {
			t.Fatalf("%s : %v", rel, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || nomAppele(call.Fun) != "ComputeKillerVictimPairs" {
				return true
			}
			appels++
			if len(call.Args) != 2 || !estFenetreCanonique(call.Args[1]) {
				fautifs = append(fautifs, rel)
			}
			return true
		})
	})
	if appels == 0 {
		t.Fatal("aucun appel de ComputeKillerVictimPairs trouvé : le balayage ne voit plus rien")
	}
	if len(fautifs) > 0 {
		t.Errorf("appel(s) de ComputeKillerVictimPairs sans analysis.KillDeathPairToleranceMS :\n  %s",
			strings.Join(fautifs, "\n  "))
	}
}

// estFenetreCanonique : l argument est-il `analysis.KillDeathPairToleranceMS` (ou l identifiant nu,
// dans le paquet `analysis`) ?
func estFenetreCanonique(arg ast.Expr) bool {
	switch a := arg.(type) {
	case *ast.Ident:
		return a.Name == "KillDeathPairToleranceMS"
	case *ast.SelectorExpr:
		x, ok := a.X.(*ast.Ident)
		return ok && x.Name == "analysis" && a.Sel.Name == "KillDeathPairToleranceMS"
	}
	return false
}
