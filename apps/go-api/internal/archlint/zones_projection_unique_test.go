package archlint

// zones_projection_unique_test.go — garde-rail : une seule projection des zones nommées vers le type
// du paquet d'algo. `tactical.ZonesDepuisLePort` (internal/analysis/tactical) est la seule construction
// d'un `tactical.ZoneNommee` non vide hors de son paquet ; une seconde copie (le service Tactique et
// la commande du modèle de spawn en avaient chacun une) re-diverge au premier champ ajouté.
//
// Le garde-rail repère le littéral composé `tactical.ZoneNommee{...}` (non vide) dans le code non
// testé de `cmd/` et `internal/`, hors du paquet. Le zéro `tactical.ZoneNommee{}` (valeur de repli) n'est
// pas une projection et reste permis.

import (
	"go/ast"
	"strings"
	"testing"
)

// zonesProjectionPaquet : le seul paquet qui construit un ZoneNommee (chemin de garde-rail).
const zonesProjectionPaquet = "internal/analysis/tactical/"

// projectionsZonesHorsPaquet rend la position (fichier:ligne) de chaque littéral `tactical.ZoneNommee{...}`
// non vide d'un fichier source qui n'appartient pas au paquet.
func projectionsZonesHorsPaquet(src goSource) []string {
	if strings.HasPrefix(src.Rel, zonesProjectionPaquet) {
		return nil
	}
	var out []string
	ast.Inspect(src.File, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok || len(lit.Elts) == 0 {
			return true
		}
		sel, ok := lit.Type.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "ZoneNommee" {
			return true
		}
		if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "tactical" {
			pos := src.Fset.Position(lit.Pos())
			out = append(out, pos.Filename)
		}
		return true
	})
	return out
}

func TestZonesProjectionUnique_AucuneSecondeProjectionHorsDuPaquet(t *testing.T) {
	for _, src := range goSourcesUnder(t, "cmd", "internal") {
		for _, pos := range projectionsZonesHorsPaquet(src) {
			t.Errorf("%s : projection de zones hors de %s : utiliser tactical.ZonesDepuisLePort",
				pos, zonesProjectionPaquet)
		}
	}
}

// TestZonesProjectionUnique_DetecteLaCopie : auto-preuve, le détecteur voit une copie et laisse passer
// le zéro et le paquet lui-même.
func TestZonesProjectionUnique_DetecteLaCopie(t *testing.T) {
	copie := parseGoSnippet(t, "internal/service/copie.go", `package service
func f(zs []domain.ZoneNommee) []tactical.ZoneNommee {
	out := []tactical.ZoneNommee{}
	out = append(out, tactical.ZoneNommee{X: 1})
	return out
}
func zero() tactical.ZoneNommee { return tactical.ZoneNommee{} }
`)
	if got := projectionsZonesHorsPaquet(copie); len(got) != 1 {
		t.Fatalf("copie : %d littéraux détectés, attendu 1 (le zéro ne compte pas)", len(got))
	}
	dansLePaquet := parseGoSnippet(t, "internal/analysis/tactical/zones_port.go", `package tactical
func f() ZoneNommee { return ZoneNommee{X: 1} }
`)
	if got := projectionsZonesHorsPaquet(dansLePaquet); got != nil {
		t.Fatalf("le paquet tactical est exempté : %v", got)
	}
}
