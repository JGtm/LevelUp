package archlint

// film_depottest_tests_seulement_test.go — `depottest` EST UNE AIDE DE TEST, JAMAIS DE PRODUCTION.
//
// # LA REGLE
//
// `libs/film/internal/depottest` lit l'environnement (la racine des donnees de test) : l'exception de
// `film_sans_exploitation_test.go` n'est admissible que parce que ce paquet n'est importe que par des
// fichiers `_test.go` de la brique. Le test refuse tout fichier `.go` de `libs/film` qui n'est pas un
// `_test.go` (instruments `research` compris) et qui importe `libs/film/internal/depottest`.
//
// `depottest` est sous `internal/` : aucun autre module ne peut l'importer, ce garde-rail n'a donc a
// balayer que la brique. Allowlist vide.
//
// # LA MUTATION QUI DOIT LE FAIRE ROUGIR
//
// Importer `libs/film/internal/depottest` depuis un fichier non-test de `libs/film/replay`.

import (
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// cheminDeDepottest : le chemin d'import de l'aide de test.
const cheminDeDepottest = prefixeBriques + "film/internal/depottest"

func TestDepottestNEstImporteQueParDesTests(t *testing.T) {
	var violations []string
	vus := 0
	fichiersGo(t, surDisque(t, racineBriqueFilm), func(rel string, data []byte) {
		if strings.HasSuffix(rel, "_test.go") {
			return
		}
		vus++
		f, err := parser.ParseFile(token.NewFileSet(), rel, data, parser.ImportsOnly)
		if err != nil {
			t.Errorf("%s : %v", rel, err)
			return
		}
		for _, imp := range f.Imports {
			if chemin, uerr := strconv.Unquote(imp.Path.Value); uerr == nil && chemin == cheminDeDepottest {
				violations = append(violations, rel)
			}
		}
	})
	if vus < 100 {
		t.Fatalf("balayage muet : %d fichiers hors test vus sous %s", vus, racineBriqueFilm)
	}
	if len(violations) > 0 {
		t.Errorf("depottest importe hors d'un _test.go (c'est une aide de test, elle lit l'environnement) :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
