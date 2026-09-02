// no_film_reread_test.go — LE FILM SE CHARGE UNE FOIS, ET RIEN NE DOIT ROUVRIR LA PORTE.
//
// # CE QUE CE GARDE-RAIL PROTEGE, ET CE QU'IL A COUTE DE L'OBTENIR
//
// Lot 1 de PLAN_CUISSON_PERF (2026-09-02). Un artefact de rejeu relisait et redecompressait le
// film ENTIER une trentaine de fois : chaque `ScanFilm*(dir)` ouvrait le repertoire de chunks
// pour son propre compte, avec TROIS inflates et TROIS marcheurs de paquets divergents dans le
// depot. Le decodage pesait ~94 % du temps de cuisson (mesure 0.8). Le lot a fait passer tous
// les balayages a un `*filmsource.Film` charge UNE fois par `replaybuild.BuildBytes`.
//
// Rien de tout cela ne tient si un balayage rajoute demain un `os.ReadFile` « juste pour lire un
// petit chunk », ou si un appelant de production reprend une enveloppe `ScanFilmXxx(dir)` parce
// qu'elle est plus courte a ecrire. Les trois regles ci-dessous ferment ces trois portes, et
// elles portent une DATE : elles tomberont avec les enveloppes, quand plus aucun appelant hors
// production n'en aura besoin (lot 6, au plus tot).
//
// # LES TROIS REGLES
//
//  1. `zlib.NewReader` est INTERDIT dans `filmdec` et `replay` (hors _test) — l'unique
//     decompresseur de la chaine de cuisson est `filmsource.Inflate`. Liste des sites
//     survivants ailleurs dans le depot : item 1.9 du plan.
//  2. `os.ReadFile` / `os.ReadDir` / `os.Open` sont INTERDITS dans `filmdec` (hors _test) sauf
//     dans les fichiers de l'allowlist datee ci-dessous : les chargeurs de CATALOGUE (qui ne
//     lisent pas de film) et les enveloppes D2 declarees.
//  3. Aucun appel d'ENVELOPPE `dir` depuis les sites de PRODUCTION (`analysis/replay` et
//     `replaybuild`, hors _test) : la production passe un film deja charge. La liste des
//     enveloppes est FERMEE et ecrite ici.
//
// # POURQUOI go/ast ET PAS UN GREP
//
// Ces paquets CITENT les noms interdits dans leurs commentaires — abondamment, puisque c'est la
// migration qu'ils documentent. Un test grep rougirait sur la documentation du garde-rail
// lui-meme. Le test parse donc les fichiers et ne regarde que les APPELS et les IMPORTS.
package archlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// apiRootDepuisIci rend la racine d'apps/go-api depuis ce fichier.
func apiRootDepuisIci(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(thisFile))) // internal/archlint -> internal -> apps/go-api
}

// fichiersGoNonTest rend les fichiers .go non-test d'un paquet, avec leur AST.
func fichiersGoNonTest(t *testing.T, pkgDir string) map[string]*ast.File {
	t.Helper()
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		t.Fatalf("paquet %s introuvable (%v) : s'il a DEMENAGE, deplacer ce garde-rail avec lui", pkgDir, err)
	}
	fset := token.NewFileSet()
	out := map[string]*ast.File{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(pkgDir, name), nil, 0)
		if err != nil {
			t.Fatalf("analyse de %s : %v", name, err)
		}
		out[name] = f
	}
	if len(out) == 0 {
		t.Fatalf("aucun fichier non-test dans %s : ce garde-rail n'aurait plus d'objet", pkgDir)
	}
	return out
}

// paquetsSansInflate : les deux paquets de la chaine de cuisson qui n'ont plus le droit de
// decompresser eux-memes, relatifs a apps/go-api.
var paquetsSansInflate = []string{
	"internal/analysis/filmdec",
	"internal/analysis/replay",
}

// TestPasDeZlibDansLaChaineDeCuisson — REGLE 1.
func TestPasDeZlibDansLaChaineDeCuisson(t *testing.T) {
	racine := apiRootDepuisIci(t)
	var violations []string
	for _, pkg := range paquetsSansInflate {
		for nom, f := range fichiersGoNonTest(t, filepath.Join(racine, filepath.FromSlash(pkg))) {
			for _, imp := range f.Imports {
				if strings.Trim(imp.Path.Value, `"`) == "compress/zlib" {
					violations = append(violations, pkg+"/"+nom)
				}
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("`compress/zlib` importe dans la chaine de cuisson :\n  %s\n"+
			"Le film est decompresse UNE fois, par `filmsource` (lot 1 de PLAN_CUISSON_PERF). "+
			"Un chunk isole se lit par `filmsource.Inflate` ou `filmdec.ReadFilmChunk` ; un film "+
			"entier par `filmsource.LoadDir`. Trois inflates divergents ont deja coute une mesure "+
			"sur 1 378 films pour prouver qu'ils voyaient les memes octets.",
			strings.Join(violations, "\n  "))
	}
}

// fichiersFilmdecLisantLeDisque : l'ALLOWLIST DATEE de la regle 2 (2026-09-02, lot 1).
//
//	map_bounds.go            chargeur de CATALOGUE (map_quant_bounds.json) — ne lit aucun film.
//	film_packets.go          enveloppes D2 `ReadFilmChunk` / `CountFilmChunks` : les lecteurs
//	                         d'un chunk ISOLE (instruments, tests) et rien d'autre.
//	keyframe_entity_queue.go `FindPackets`, instrument de reconciliation qui balaye une RACINE
//	                         de films (pas un film) : il n'a aucun film a charger.
//
// RETRAIT CIBLE : lot 6, quand les enveloppes D2 n'auront plus d'appelant. Critere mesurable :
// `grep -r 'ScanFilm[A-Za-z]*(' --include=*.go` ne rend plus que leurs definitions.
var fichiersFilmdecLisantLeDisque = map[string]bool{
	"map_bounds.go":            true,
	"film_packets.go":          true,
	"keyframe_entity_queue.go": true,
}

// lecturesDisque : les appels `os.*` qui touchent le systeme de fichiers.
var lecturesDisque = map[string]bool{"ReadFile": true, "ReadDir": true, "Open": true, "Stat": true}

// TestFilmdecNeLitPasLeDisqueHorsAllowlist — REGLE 2.
func TestFilmdecNeLitPasLeDisqueHorsAllowlist(t *testing.T) {
	pkgDir := filepath.Join(apiRootDepuisIci(t), filepath.FromSlash("internal/analysis/filmdec"))
	var violations []string
	for nom, f := range fichiersGoNonTest(t, pkgDir) {
		if fichiersFilmdecLisantLeDisque[nom] {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			id, ok := sel.X.(*ast.Ident)
			if ok && id.Name == "os" && lecturesDisque[sel.Sel.Name] {
				violations = append(violations, nom+" -> os."+sel.Sel.Name)
			}
			return true
		})
	}
	if len(violations) > 0 {
		t.Fatalf("`filmdec` relit le disque hors allowlist :\n  %s\n"+
			"Les balayages recoivent un `*filmsource.Film` DEJA CHARGE (lot 1) : un `os.ReadFile` "+
			"ici, c'est le film relu une fois de plus — l'exact defaut que le lot a supprime. "+
			"Si le fichier lu n'est PAS un film (catalogue, manifeste), l'ajouter a "+
			"`fichiersFilmdecLisantLeDisque` avec sa justification et sa date.",
			strings.Join(violations, "\n  "))
	}
}

// enveloppesInterditesEnProduction : la LISTE FERMEE des enveloppes `dir` du lot 1, que la
// production ne doit jamais appeler (regle D2 du plan). Chacune charge un film ENTIER pour un
// seul balayage ; les appeler depuis la cuisson annulerait le lot.
//
// La liste couvre les enveloppes exportees de `filmdec` et de `replay`. Les formes film
// (`ScanXxx(film, ...)`) ne sont PAS ici : ce sont elles que la production appelle.
var enveloppesInterditesEnProduction = []string{
	// filmdec
	"ScanFilmAbilityRanks", "ScanFilmBipedPickups", "ScanFilmBipedPositions", "ScanFilmCamoStates",
	"ScanFilmCarrierMarks", "ScanFilmEquipmentChanges", "ScanFilmEquipmentCreations",
	"ScanFilmEquipmentCreationsForBand", "ScanFilmEquipmentPlacements", "ScanFilmEquipmentState",
	"ScanFilmFireEvents", "ScanFilmGrappleReads", "ScanFilmGrenadeThrows",
	"ScanFilmGroundWeaponCreations", "ScanFilmGroundWeaponCreationsForBand",
	"ScanFilmHeldWeaponChanges", "ScanFilmInventoryDeltas", "ScanFilmKeyframeGroundWeapons",
	"ScanFilmKeyframeLoadouts", "ScanFilmManagedProperties", "ScanFilmNavpointRadial",
	"ScanFilmObjectives", "ScanFilmProjectiles", "ScanFilmUnitEquipment",
	"ScanFilmWorldObjectKeyframes", "ScanFilmWorldObjects", "ScanFilmWorldObjectsForBand",
	"ScanFilmZoomEvents",
	"DetectI0Layout", "EquipmentArchetype", "CalibrateMPPWidths",
	"GroundWeaponSlotBand", "GroundWeaponPositions", "WorldObjectPositionsForBand",
	"ReadFilmChunk", "CountFilmChunks",
	// replay
	"ScanFilmDeaths", "ScanFilmClockOrigin", "ScanFilmPlayerIndices", "ScanFilmKeyframeInventory",
}

// paquetsDeProduction : les paquets ou l'appel d'une enveloppe est interdit.
var paquetsDeProduction = []string{
	"internal/analysis/replay",
	"internal/replaybuild",
}

// TestProductionNAppellePasLesEnveloppes — REGLE 3.
//
// Les DEFINITIONS des enveloppes de `replay` vivent dans ces memes fichiers : le test ne compte
// que les APPELS (`ast.CallExpr`), jamais les declarations, et une enveloppe qui se contente de
// deleguer a sa forme film ne s'appelle pas elle-meme.
func TestProductionNAppellePasLesEnveloppes(t *testing.T) {
	interdites := map[string]bool{}
	for _, n := range enveloppesInterditesEnProduction {
		interdites[n] = true
	}
	racine := apiRootDepuisIci(t)
	var violations []string
	for _, pkg := range paquetsDeProduction {
		for nom, f := range fichiersGoNonTest(t, filepath.Join(racine, filepath.FromSlash(pkg))) {
			ast.Inspect(f, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if appele := nomAppele(call.Fun); interdites[appele] {
					violations = append(violations, pkg+"/"+nom+" -> "+appele)
				}
				return true
			})
		}
	}
	if len(violations) > 0 {
		t.Fatalf("la production appelle une enveloppe `dir` :\n  %s\n"+
			"Ces enveloppes chargent un film ENTIER par appel (regle D2 de PLAN_CUISSON_PERF) : "+
			"elles existent pour les tests et les instruments de recherche. La cuisson recoit un "+
			"`*filmsource.Film` deja charge et appelle la forme film (`ScanXxx(film, ...)`).",
			strings.Join(violations, "\n  "))
	}
}

// nomAppele rend le nom de la fonction appelee : `Xxx(...)` ou `pkg.Xxx(...)`.
func nomAppele(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	}
	return ""
}
