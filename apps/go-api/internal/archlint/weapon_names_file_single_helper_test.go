// Package archlint — weapon_names_file_single_helper_test.go : garde-rail « le fichier des noms
// d'armes d'un titre ne se nomme que par son helper ».
//
// POURQUOI. Le fichier `weapon_names.toml` est la source unique des noms d'armes affichés (table
// `weapon_name_labels` seedée depuis lui). Quatre sites l'avaient chacun nommé et chargé à la main
// (`boot_migrations`, `provision_title`, le catalogue de libellés du rejeu, le référentiel des
// pictogrammes) : une quatrième copie contournait la source unique sans qu'aucun test ne le voie.
//
// INTERDIT dans le code du module (hors _test.go) : le littéral de chaîne "weapon_names.toml",
// hors de internal/games/mappings/loader_weapon_names.go, qui porte [mappings.WeaponNamesPath] et
// [mappings.LoadWeaponNamesInDir]. Un commentaire qui cite le fichier n'est pas un littéral.
package archlint

import (
	"go/ast"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	weaponNamesHelperFile = "internal/games/mappings/loader_weapon_names.go"
	weaponNamesLiteral    = "weapon_names.toml"
)

var weaponNamesRoots = []string{"cmd", "internal", "pkg"}

func weaponNamesLiteralProblems(sources []goSource) []string {
	var out []string
	helperSites := 0
	for _, src := range sources {
		ast.Inspect(src.File, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			v, err := strconv.Unquote(lit.Value)
			if err != nil || v != weaponNamesLiteral {
				return true
			}
			if src.Rel == weaponNamesHelperFile {
				helperSites++
				return true
			}
			out = append(out, src.Rel+":"+strconv.Itoa(src.Fset.Position(lit.Pos()).Line)+
				" : le fichier des noms d'armes est nommé hors de mappings.WeaponNamesPath / LoadWeaponNamesInDir")
			return true
		})
	}
	if helperSites != 1 {
		out = append(out, weaponNamesHelperFile+" : "+strconv.Itoa(helperSites)+
			" littéral(aux) du fichier des noms d'armes, attendu exactement 1 (le helper)")
	}
	sort.Strings(out)
	return out
}

func TestLeFichierDesNomsDArmesNEstNommeQueParSonHelper(t *testing.T) {
	problems := weaponNamesLiteralProblems(goSourcesUnder(t, weaponNamesRoots...))
	if len(problems) > 0 {
		t.Errorf("fichier des noms d'armes nommé hors du helper :\n  %s", strings.Join(problems, "\n  "))
	}
}

func TestLeGardeRailDesNomsDArmesDetecteUneCopie(t *testing.T) {
	copie := parseGoSnippet(t, "internal/service/copie.go",
		"package service\nvar p = \"weapon_names.toml\"\n")
	helper := parseGoSnippet(t, weaponNamesHelperFile,
		"package mappings\nconst weaponNamesFile = \"weapon_names.toml\"\n")
	if got := weaponNamesLiteralProblems([]goSource{helper, copie}); len(got) != 1 ||
		!strings.Contains(got[0], "internal/service/copie.go:2") {
		t.Fatalf("la copie hors helper doit être signalée seule : %v", got)
	}
	if got := weaponNamesLiteralProblems([]goSource{helper}); len(got) != 0 {
		t.Fatalf("le helper seul est conforme : %v", got)
	}
	if got := weaponNamesLiteralProblems([]goSource{copie}); len(got) != 2 {
		t.Fatalf("sans helper, le littéral manquant doit aussi être signalé : %v", got)
	}
}
