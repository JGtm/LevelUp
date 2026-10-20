// film_types_leaf_test.go — `internal/games/halo_infinite/film/types` EST UNE FEUILLE, ET C EST
// CE QUI AUTORISE TOUT LE MONDE A L IMPORTER (lot 2.6.2, 2026-09-16).
//
// # POURQUOI CE GARDE-RAIL EXISTE, ET IL A UNE DATE
//
// Le paquet porte les types de DONNEES qui traversent les frontieres de couche du decodeur
// (ADR 0034 D-1). Les cinq couches le nomment, `film/source` COMPRIS — or `source` est elle-meme
// une feuille a zero import du depot, pour une raison qui n est pas un gout : un cycle VERIFIE
// (`filmsource_leaf_test.go`, lot 1 de PLAN_CUISSON_PERF). L exception accordee a `types` dans ce
// ratchet-la ne tient QUE si `types` n importe lui-meme rien du depot : une feuille importee par
// une feuille ne ferme aucun cycle. Ce test est la moitie manquante de cet argument.
//
// Le jour ou quelqu un ajoutera « juste un petit import » de `title`, `canonical` ou `grammar`
// dans `types`, il rouvrira la porte que le lot 1 avait fermee — et il l apprendra ici plutot
// qu a la premiere compilation cyclique d un consommateur, trois lots plus loin.
//
// # CE QU IL VERIFIE, ET COMMENT
//
// Il PARSE les imports (go/parser, ImportsOnly) des fichiers non-test du paquet : un test grep se
// ferait tromper par un chemin cite dans un commentaire, et ce paquet en cite plusieurs. Les
// `_test.go` sont HORS PERIMETRE, deliberement : `shapes_test.go` importe `source` et `facts`
// pour lire leurs REVISIONS, ce qui ne cree aucun cycle (un paquet de test externe n est importe
// par personne) et constitue le lien entre la forme figee et la revision qui la date.
package archlint

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// filmTypesLeafPkg : le paquet feuille, relatif a apps/go-api.
const filmTypesLeafPkg = "internal/games/halo_infinite/film/types"

func TestFilmTypesEstUneFeuille(t *testing.T) {
	pkgDir := filepath.Join(apiRootDepuisIci(t), filepath.FromSlash(filmTypesLeafPkg))
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		t.Fatalf("paquet %s introuvable (%v) : s il a DEMENAGE, deplacer ce garde-rail avec lui — "+
			"la contrainte de feuille tient au cycle que `film/source` ferme, pas au chemin",
			filmTypesLeafPkg, err)
	}
	fset := token.NewFileSet()
	fichiers := 0
	var violations []string
	for _, e := range entries {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		fichiers++
		f, perr := parser.ParseFile(fset, filepath.Join(pkgDir, nom), nil, parser.ImportsOnly)
		if perr != nil {
			t.Fatalf("analyse de %s : %v", nom, perr)
		}
		for _, imp := range f.Imports {
			chemin := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(chemin, prefixeModuleFilm) {
				violations = append(violations, nom+" -> "+chemin)
			}
		}
	}
	if fichiers == 0 {
		t.Fatalf("aucun fichier non-test dans %s : ce garde-rail n aurait plus d objet",
			filmTypesLeafPkg)
	}
	if len(violations) > 0 {
		t.Fatalf("%s n est plus une FEUILLE — %d import(s) du depot :\n  %s\n"+
			"Les cinq couches du decodeur nomment ce paquet, `film/source` comprise : un import du "+
			"depot ici rouvre le cycle filmdec/objectiveevents/filmcache que le lot 1 de "+
			"PLAN_CUISSON_PERF a ferme. Un type qui a besoin d un autre paquet n est pas un type "+
			"de CONTRAT : il reste dans la couche qui le produit.",
			filmTypesLeafPkg, len(violations), strings.Join(violations, "\n  "))
	}
}
