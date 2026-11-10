// Package archlint — briques_chemin_unique_test.go : le dossier d'une brique se demande au résolveur
// `internal/briques`, il ne se construit jamais depuis la racine du dépôt.
//
// POURQUOI. Une brique (`libs/<brique>`, ADR 0041) est un module Go : son dossier est celui que l'outil Go
// désigne (`go list -m -f {{.Dir}}`), c'est-à-dire le dossier de l'espace de travail aujourd'hui et le cache
// de modules le jour où la brique est consommée en version publiée. Un chemin écrit `filepath.Join(racine,
// "libs", "film", ...)` marche tant que la brique est sous `libs/` et casse en silence (ou saute) quand elle n'y
// est plus. Des fixtures, des goldens, des registres et des sources balayées construisaient ce chemin à la main ;
// ils passent tous par `briques.Chemin`, `briques.SurDisque` ou les aides de test qui les enveloppent
// (`testutil.CheminBriqueFilm`, `surDisque`).
//
// CE QU'IL VÉRIFIE. Aucun appel `Join(...)` (`filepath` ou `path`) de `apps/go-api` ne porte en argument un
// littéral `"libs"` ou commençant par `"libs/"`. Un chemin de garde-rail `libs/<brique>/...` passé à une fonction
// qui le résout (`surDisque`, `racineDeclaree`, `briques.SurDisque`) n'est pas visé : c'est une clé, pas un chemin.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : écrire `filepath.Join(racine, "libs", "film", "go.mod")` dans un test.
package archlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// cheminsDeBriqueEnDurAutorises : fichiers qui construisent `libs/...` à la main, avec la raison datée. Entrée =
// « fichier : raison, date ». Une entrée dont le fichier ne construit plus rien est morte et fait échouer le test.
var cheminsDeBriqueEnDurAutorises = map[string]string{
	"apps/go-api/internal/archlint/dockerfile_go_version_test.go": "vérifie le DÉPÔT (go.mod de chaque brique contre le Dockerfile et dependabot.yml), pas une lecture de brique ; " +
		"supprimé avec `libs/` par l'étape 6.5 du plan 8.1.2 — 2026-11-10",
	"apps/go-api/cmd/replay-equiv/main.go": "références du harnais (`dossierEquivalence`) : des tests de la brique du décodeur " +
		"(`replay/golden_inputs_film_test.go`, `replay/equivalence_minifilm_test.go`, `internal/grammar/keyframe_closure_cuisson_ratchet_test.go`) " +
		"lisent le même dossier `replay/testdata/equivalence`, le déménagement prévu par l'item 6.2 est donc arrêté " +
		"en attente de décision du pilote — 2026-11-10",
}

// constructionsDeCheminDeBrique rend « fichier:ligne » de chaque `Join(...)` qui porte un littéral `"libs"` ou
// `"libs/..."`.
func constructionsDeCheminDeBrique(rel string, src []byte) []string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rel, src, parser.SkipObjectResolution)
	if err != nil {
		return []string{rel + " : source illisible : " + err.Error()}
	}
	var sites []string
	ast.Inspect(f, func(n ast.Node) bool {
		appel, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := appel.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Join" {
			return true
		}
		for _, arg := range appel.Args {
			lit, ok := arg.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			v, err := strconv.Unquote(lit.Value)
			if err == nil && (v == "libs" || strings.HasPrefix(v, "libs/")) {
				sites = append(sites, rel+":"+strconv.Itoa(fset.Position(appel.Pos()).Line))
				break
			}
		}
		return true
	})
	return sites
}

func TestCheminsDeBriqueParLeResolveur(t *testing.T) {
	vus := map[string]bool{}
	var problemes []string
	for _, f := range workspaceGoSources(t, "apps/go-api") {
		src, err := os.ReadFile(f.Chemin) //nolint:gosec // chemin issu du parcours du module
		if err != nil {
			t.Fatalf("lecture de %s : %v", f.Rel, err)
		}
		sites := constructionsDeCheminDeBrique(f.Rel, src)
		if len(sites) == 0 {
			continue
		}
		if _, permis := cheminsDeBriqueEnDurAutorises[f.Rel]; permis {
			vus[f.Rel] = true
			continue
		}
		problemes = append(problemes, sites...)
	}
	sort.Strings(problemes)
	if len(problemes) > 0 {
		t.Errorf("chemin de brique construit à la main (utiliser briques.Chemin, briques.SurDisque, "+
			"testutil.CheminBriqueFilm ou surDisque) :\n  %s", strings.Join(problemes, "\n  "))
	}
	for rel := range cheminsDeBriqueEnDurAutorises {
		if !vus[rel] {
			t.Errorf("%s : l'entrée de cheminsDeBriqueEnDurAutorises ne construit plus de chemin de brique, la retirer", rel)
		}
	}
}

// TestCheminsDeBriqueParLeResolveur_AutoPreuve : la forme interdite rougit sous `filepath.Join` et `path.Join`,
// un littéral qui n'est qu'une clé, un autre dossier nommé `libs` hors littéral et un `Join` sans `libs` passent.
func TestCheminsDeBriqueParLeResolveur_AutoPreuve(t *testing.T) {
	cas := map[string]int{
		`package p; var _ = filepath.Join(racine, "libs", "film")`:      1,
		`package p; var _ = path.Join(racine, "libs/film/go.mod")`:      1,
		`package p; var _ = filepath.Join(racine, "apps", "go-api")`:    0,
		`package p; var _ = surDisque(t, "libs/film/replay")`:           0,
		`package p; var _ = filepath.Join(racine, dossierLibs, "film")`: 0,
		`package p; var _ = filepath.Join(racine, "libsodium")`:         0,
	}
	for src, attendu := range cas {
		if got := len(constructionsDeCheminDeBrique("x.go", []byte(src))); got != attendu {
			t.Errorf("%s : %d site(s), attendu %d", src, got, attendu)
		}
	}
}
