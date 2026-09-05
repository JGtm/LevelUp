// filmsource_leaf_test.go — `internal/analysis/filmsource` EST UNE FEUILLE, ET ELLE DOIT LE RESTER.
//
// # POURQUOI CE GARDE-RAIL EXISTE, ET IL A UNE DATE
//
// 2026-09-02, lot 1 de PLAN_CUISSON_PERF (§2 et §3 D1). La source du film (decompression + une
// grammaire de decoupage unique) devait vivre quelque part, et `filmdec` etait le candidat
// naturel. Il est INTERDIT, et l'argument est un cycle d'import VERIFIE, pas une preference :
//
//	`filmcache` importe `objectiveevents` (filmcache.go) ;
//	cinq tests INTERNES de `filmdec` importent `objectiveevents` ou `filmcache`
//	(sonde_registre_verdicts_test.go, navpoint_ti12_radial_test.go, objectif_ti11_minuteurs_test.go,
//	ti47_annonces_test.go, zone_census_report_test.go).
//
// Donc : faire importer `filmdec` par `objectiveevents` — ou `filmcache` par `filmdec` — ferme un
// cycle, en production ou en test. La seule position tenable est une FEUILLE que tout le monde
// peut importer : stdlib seule, zero import du depot. Le jour ou quelqu'un ajoutera « juste un
// petit import » de `title`, `canonical` ou `filmdec` dans ce paquet, il rouvrira la porte que ce
// lot a fermee — et il l'apprendra ici plutot qu'a la premiere compilation cyclique d'un
// consommateur, trois lots plus loin.
//
// # CE QU'IL VERIFIE, ET COMMENT
//
// Il PARSE les imports (go/parser, ImportsOnly) des fichiers non-test du paquet : un test grep se
// ferait tromper par un chemin cite dans un commentaire — et ce paquet en cite plusieurs, y
// compris ceux du cycle ci-dessus. Les `_test.go` sont HORS PERIMETRE, deliberement : le test
// EXTERNE `filmsource_test` importe `filmdec` pour comparer les deux marcheurs de paquets sur un
// film reel, ce qui ne cree aucun cycle (un paquet de test externe n'est importe par personne) et
// constitue la preuve d'equivalence de la grammaire.
package archlint

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// filmsourceLeafPkg : le paquet feuille, relatif a apps/go-api.
const filmsourceLeafPkg = "internal/analysis/filmsource"

// filmsourceLeafPrefix : le prefixe du module. Tout import qui commence par la est un import DU
// DEPOT, et il est interdit ici. La stdlib (et elle seule) est autorisee.
const filmsourceLeafPrefix = "levelup/go-api/"

func TestFilmsourceEstUneFeuille(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	apiRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile))) // internal/archlint -> internal -> apps/go-api
	pkgDir := filepath.Join(apiRoot, filepath.FromSlash(filmsourceLeafPkg))

	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		t.Fatalf("paquet %s introuvable (%v) : s'il a DEMENAGE, deplacer ce garde-rail avec lui — "+
			"la contrainte de feuille tient au cycle filmdec/objectiveevents/filmcache, pas au chemin", filmsourceLeafPkg, err)
	}

	fset := token.NewFileSet()
	fichiers := 0
	var violations []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fichiers++
		f, err := parser.ParseFile(fset, filepath.Join(pkgDir, name), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("analyse de %s : %v", name, err)
		}
		for _, imp := range f.Imports {
			chemin := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(chemin, filmsourceLeafPrefix) {
				violations = append(violations, name+" -> "+chemin)
			}
		}
	}

	if fichiers == 0 {
		t.Fatalf("aucun fichier non-test dans %s : ce garde-rail n'aurait plus d'objet", filmsourceLeafPkg)
	}
	if len(violations) > 0 {
		t.Fatalf("%s n'est plus une FEUILLE — %d import(s) du depot :\n  %s\n"+
			"Ce paquet est importe par filmdec, objectiveevents, killsource, filmcache et killcollector : "+
			"tout import du depot y rouvre le cycle que le lot 1 a ferme (cf. l'en-tete de ce fichier). "+
			"Ce dont il a besoin se PASSE en argument (ChunkMeta, Source), on ne l'importe pas.",
			filmsourceLeafPkg, len(violations), strings.Join(violations, "\n  "))
	}
}
