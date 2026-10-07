package wire

// registry_pages_tactical_wiring_test.go — LE CÂBLAGE DE L'ONGLET TACTIQUE (plan Tactique v2, L3.5).
//
// Chaque source du détail d'une zone retire son champ quand elle manque, sans erreur : un `With*`
// oublié ne rougit aucun test de service. Les options sont lues dans l'arbre syntaxique de la
// factory `Tactical` — leur argument et la porte `if` qui les entoure.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func appelsDansTactical(t *testing.T, methode string) []appelOptionCable {
	t.Helper()
	return appelsDansFactory(t, "registry_pages_tactical.go", "Tactical", methode)
}

// Chaque dépendance du service est câblée UNE fois, SANS condition, avec l'argument attendu. Le
// classificateur est nil sur un titre sans film.kill_source : c'est killSourceClassifierFor qui le
// décide, sur les capabilities du titre, pas une porte ici.
func TestTactical_CableChaqueDependanceSansCondition(t *testing.T) {
	attendus := map[string]string{
		"WithRasterStore":          "rasters",
		"WithCalloutsStore":        "callouts",
		"WithRetentionMois":        "r.retentionMoisRejeu",
		"WithRadarRange":           "r.radarRangeFor(pdb)",
		"WithPlayerMatches":        "r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag",
		"WithRoundsDecide":         "r.roundsDecideFor(pdb)",
		"WithKillSourceClassifier": "r.killSourceClassifierFor(pdb)",
		"WithWeaponLabels":         "duckdb.NewWeaponRangeRepo(pdb, r.killSourceClassifierFor(pdb))",
		"WithReplay":               "r.replayServiceFor(pdb)",
	}
	for methode, args := range attendus {
		appels := appelsDansTactical(t, methode)
		if len(appels) != 1 {
			t.Errorf("%d appel(s) à %s dans Tactical, attendu 1", len(appels), methode)
			continue
		}
		if got := strings.Join(appels[0].args, ", "); got != args {
			t.Errorf("%s(%s) : attendu %s(%s)", methode, got, methode, args)
		}
		if len(appels[0].portes) != 0 {
			t.Errorf("%s est sous condition (%v) : attendu un câblage inconditionnel", methode, appels[0].portes)
		}
	}
}

// La factory a quitté registry_pages.go (au-delà du seuil de taille du dépôt).
func TestTactical_FactoryHorsDeRegistryPages(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "registry_pages.go", nil, 0)
	if err != nil {
		t.Fatalf("lecture de registry_pages.go : %v", err)
	}
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv != nil && fn.Name.Name == "Tactical" {
			t.Error("la factory Tactical est encore déclarée dans registry_pages.go")
		}
	}
}
