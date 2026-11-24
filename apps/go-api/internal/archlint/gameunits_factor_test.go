// Package archlint — gameunits_factor_test.go : LE FACTEUR UNITÉS DU JEU → MÈTRES A UN SEUL FOYER.
//
// Une unité du jeu vaut 10 pieds, soit 3,048 m (`internal/analysis/gameunits`). Le facteur est écrit
// UNE fois côté Go, dans ce paquet, et UNE fois côté web (`lib/formatters/distance.ts`) ; une copie
// « 3.048 » ailleurs, ou « 3 » arrondi à l'œil, ferait diverger deux lectures de la même distance.
// Le test de parité lit les deux fichiers et exige la même valeur.
package archlint

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

const gameunitsFichier = "internal/analysis/gameunits/gameunits.go"

// littérauxFacteur rend la position de chaque littéral flottant 3.048 (ou 0.3048) d'une source.
func littérauxFacteur(src goSource) []string {
	var out []string
	ast.Inspect(src.File, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.FLOAT {
			return true
		}
		if v, err := strconv.ParseFloat(lit.Value, 64); err == nil && (v == 3.048 || v == 0.3048) {
			out = append(out, src.Fset.Position(lit.Pos()).String())
		}
		return true
	})
	return out
}

func TestGameUnits_ReconnaitLeFacteur(t *testing.T) {
	mal := parseGoSnippet(t, "internal/x.go", "package x\nvar a = d * 3.048\nvar b = d / 0.3048")
	if got := littérauxFacteur(mal); len(got) != 2 {
		t.Errorf("facteur non reconnu : %v", got)
	}
	sain := parseGoSnippet(t, "internal/y.go", "package y\nvar a = d * 3.0\nvar b = 30.48")
	if got := littérauxFacteur(sain); len(got) != 0 {
		t.Errorf("faux positif : %v", got)
	}
}

func TestGameUnits_LeFacteurNEstEcritQueDansSonPaquet(t *testing.T) {
	var fautifs []string
	vu := false
	for _, src := range goSourcesUnder(t, "internal", "cmd") {
		lits := littérauxFacteur(src)
		if src.Rel == gameunitsFichier {
			vu = len(lits) > 0
			continue
		}
		fautifs = append(fautifs, lits...)
	}
	if !vu {
		t.Errorf("%s ne porte plus le facteur : le garde-rail ne garderait rien", gameunitsFichier)
	}
	if len(fautifs) > 0 {
		t.Errorf("facteur unités du jeu → mètres écrit hors de %s (utiliser gameunits.UnitsToMeters / MetersToUnits) :\n  %v",
			gameunitsFichier, fautifs)
	}
}

// TestGameUnits_ParitéAvecLeWeb : le facteur du helper web est celui du paquet Go.
func TestGameUnits_ParitéAvecLeWeb(t *testing.T) {
	racine := apiModuleRoot(t)
	web, err := os.ReadFile(filepath.Join(racine, "..", "web", "src", "lib", "formatters", "distance.ts"))
	if err != nil {
		t.Fatalf("lecture du helper web : %v", err)
	}
	m := regexp.MustCompile(`METERS_PER_GAME_UNIT\s*=\s*([0-9.]+)`).FindSubmatch(web)
	if m == nil {
		t.Fatal("METERS_PER_GAME_UNIT introuvable dans distance.ts")
	}
	webFacteur, _ := strconv.ParseFloat(string(m[1]), 64)

	goSrc, err := os.ReadFile(filepath.Join(racine, filepath.FromSlash(gameunitsFichier)))
	if err != nil {
		t.Fatal(err)
	}
	g := regexp.MustCompile(`MetersPerUnit\s*=\s*([0-9.]+)`).FindSubmatch(goSrc)
	if g == nil {
		t.Fatal("MetersPerUnit introuvable dans gameunits.go")
	}
	goFacteur, _ := strconv.ParseFloat(string(g[1]), 64)
	if webFacteur != goFacteur {
		t.Errorf("facteur web %v ≠ facteur Go %v", webFacteur, goFacteur)
	}
}
