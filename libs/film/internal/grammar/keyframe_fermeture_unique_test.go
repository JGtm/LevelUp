package grammar

// keyframe_fermeture_unique_test.go — UNE SEULE DEFINITION DE LA FERMETURE D UN RECORD D IMAGE-CLE
// (regle 6 du depot ; plan `.ai/archive/V7.5/PLAN_RI_RESIDU_IMAGES_CLES_2026-10-14.md`, R1.2).
//
// La fermeture s ecrivait `tr.EndBit == b.Want` en trois endroits (preuve d etat complet de la phase des
// images-cles, compteurs `KeyClosed` des balayages de la bombe et des objectifs). Depuis Rslot, elle
// lit aussi les entrees que la table porte entre deux ancres ([fermeSurLAncreSuivante]) : une copie qui
// comparerait encore la fin de traversee a l ancre suivante rendrait une autre population de records
// fermes.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"
)

// fichierDeLaFermeture est le seul fichier de production qui compare une fin de traversee a l ancre
// suivante.
const fichierDeLaFermeture = "keyframe_entrees.go"

// TestLaFermetureDImageCleEstDefinieUneFois : dans le code de production du paquet, seule
// [fermeSurLAncreSuivante] compare un `EndBit` a un `Want`. MUTATION — `tr.EndBit == b.Want` remis dans
// `objective_scan.go` : ROUGE.
func TestLaFermetureDImageCleEstDefinieUneFois(t *testing.T) {
	ents, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var horsDuHelper []string
	dansLeHelper := 0
	for _, e := range ents {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, n, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(x ast.Node) bool {
			be, ok := x.(*ast.BinaryExpr)
			if !ok || (be.Op != token.EQL && be.Op != token.NEQ) || !compareFinEtAncre(be) {
				return true
			}
			if n == fichierDeLaFermeture {
				dansLeHelper++
			} else {
				horsDuHelper = append(horsDuHelper, fset.Position(be.Pos()).String())
			}
			return true
		})
	}
	if dansLeHelper == 0 {
		t.Fatalf("%s ne compare plus de fin de traversee a l ancre suivante : le garde-rail ne garde plus rien",
			fichierDeLaFermeture)
	}
	sort.Strings(horsDuHelper)
	if len(horsDuHelper) > 0 {
		t.Errorf("fermeture d image-cle recopiee hors de %s : %s — passer par fermeSurLAncreSuivante",
			fichierDeLaFermeture, strings.Join(horsDuHelper, ", "))
	}
}

// compareFinEtAncre dit si la comparaison oppose un champ `EndBit` a un champ `Want`.
func compareFinEtAncre(be *ast.BinaryExpr) bool {
	champ := func(e ast.Expr) string {
		if s, ok := e.(*ast.SelectorExpr); ok {
			return s.Sel.Name
		}
		return ""
	}
	g, d := champ(be.X), champ(be.Y)
	return (g == "EndBit" && d == "Want") || (g == "Want" && d == "EndBit")
}
