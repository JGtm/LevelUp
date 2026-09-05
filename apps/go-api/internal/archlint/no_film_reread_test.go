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
// # LES QUATRE REGLES
//
//  1. `zlib.NewReader` est INTERDIT dans les paquets de la chaine de cuisson (hors _test) —
//     l'unique decompresseur y est `filmsource.Inflate`.
//  2. Les lectures de disque (`os.ReadFile` / `ReadDir` / `Open` / `OpenFile` / `Stat`, et
//     `filepath.Glob` — cf. [lecturesDisque]) sont INTERDITES dans `filmdec` (hors _test) sauf
//     dans les fichiers de l'allowlist datee ci-dessous : les chargeurs de CATALOGUE (qui ne
//     lisent pas de film) et les enveloppes D2 declarees.
//  3. Aucun appel d'ENVELOPPE `dir` depuis les sites de PRODUCTION de D2 (hors _test) : la
//     production passe un film deja charge. La liste des enveloppes est FERMEE et ecrite ici,
//     celle des paquets de production reprend D2 au site pres.
//  4. L'allowlist des importateurs de `compress/zlib` du DEPOT ENTIER est FERMEE (regle 1 par
//     paquet, regle 4 par depot) : un inflate qui reapparait n'importe ou se voit.
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
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
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

// paquetsSansInflate : les paquets de la chaine de cuisson qui n'ont plus le droit de
// decompresser eux-memes, relatifs a apps/go-api.
//
// `objectiveevents` y est entre a l'item 1.5 (2026-09-02) : il portait le TROISIEME inflate et
// le TROISIEME marcheur de paquets du depot (`decompressChunk`, `walkFrames`), et c'est celui
// dont la grammaire divergeait le plus — il n'emettait que le type 0, s'arretait sur CHUNK_END
// sans l'emettre, bornait la taille SANS l'offset, et marchait le BRUT COMPRESSE quand
// l'inflate echouait. Ses neuf points d'entree prennent desormais un `*filmsource.Film`.
//
// `killsource` y est entre a l'item 1.9 (2026-09-02), apres que l'item 1.4 lui a retire son
// `inflate` (`io.ReadAll` sur un `zlib.NewReader`, `chunks.go:90`) en meme temps que
// `ChunkSource`, `MemoryChunks`, `DirChunks` et `splitPackets` : c'etait le PREMIER des trois
// inflates du depot, et son film arrive maintenant deja charge (`Decode(ctx, name, film, opts)`).
// Le reste du depot est couvert par l'allowlist FERMEE de la regle 4.
var paquetsSansInflate = []string{
	"internal/analysis/filmdec",
	"internal/analysis/objectiveevents",
	"internal/analysis/replay",
	"internal/games/halo_infinite/film/killsource",
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

// lecturesDisque : les appels `os.*` (et `filepath.Glob`) qui touchent le systeme de fichiers.
//
// `OpenFile` ET `Glob` AJOUTES AU LOT 6 (constat 5 de la revue de branche) : la liste ne portait
// que `ReadFile`, `ReadDir`, `Open` et `Stat`. Les deux portes ouvertes etaient reelles et
// triviales a franchir — `os.OpenFile(path, os.O_RDONLY, 0)` lit exactement ce que `os.Open` lit,
// et `filepath.Glob` enumere un repertoire de chunks aussi bien que `os.ReadDir`. Une allowlist
// qui laisse le meme geste passer sous un autre nom ne mesure plus rien.
//
// LA REGLE NE COUVRE QUE `internal/analysis/filmdec` : `filmsource` est HORS de son perimetre par
// construction (il n'est pas dans `pkgDir`), et c'est voulu — c'est LE paquet autorise a lire un
// film, l'unique chargeur de la chaine (D1).
var lecturesDisque = map[string]bool{
	"ReadFile": true, "ReadDir": true, "Open": true, "OpenFile": true, "Stat": true, "Glob": true,
}

// paquetsLecteursDeDisque : les identifiants de paquet dont un appel de [lecturesDisque] compte
// comme une lecture. `filepath` y est pour `Glob` seul — aucun autre nom de la liste n'existe
// dans ce paquet, l'union ne peut donc pas sur-detecter.
var paquetsLecteursDeDisque = map[string]bool{"os": true, "filepath": true}

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
			if ok && paquetsLecteursDeDisque[id.Name] && lecturesDisque[sel.Sel.Name] {
				violations = append(violations, nom+" -> "+id.Name+"."+sel.Sel.Name)
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
//
// ELLE NE PORTE QUE DES NOMS SANS HOMONYME, ET C'EST UNE CONDITION DE VALIDITE (lot 6, constat 4).
// Le test compare des NOMS d'appeles : il parse l'AST et ne type rien, donc `fc.Xxx()` et
// `filmdec.Xxx()` lui sont indiscernables. Une enveloppe homonyme d'une methode rendrait la regle
// NON DISCRIMINANTE — elle interdirait l'appel legitime. Un seul cas existait,
// `EquipmentArchetype` (enveloppe `dir`) contre `FilmContext.EquipmentArchetype` (la methode que
// la cuisson appelle) : l'enveloppe s'appelle desormais `EquipmentArchetypeDir`. Avant d'ajouter
// un nom ici, verifier qu'aucune methode ne le porte
// (`grep -rE '^func \([^)]+\) <Nom>\('`).
// QUATRE NOMS AJOUTES A L'INTEGRATION DU CHANTIER VEHICULES (2026-09-05, union de
// `feat/v75-vehicules-sons` et `wt/vehicule-deadstate`) : `ScanFilmBipedPositionsForBand`,
// `ScanFilmBipedAimOnly`, `ScanFilmVehicleEvents` et `ScanFilmVehicleCreations`. Les quatre
// balayages arrivaient en ANCIENNE forme, dont trois APPELES EN PRODUCTION par
// `replay/build_vehicles.go` ; ils ont recu leur forme film (`ScanBipedPositionsForBand(film,
// band, opt)`, `ScanBipedAimOnly(fc)`, `ScanVehicleEvents(fc)`, `ScanVehicleCreations(fc, wr)`),
// et ces quatre enveloppes `dir` survivent parce que des tests les appellent
// (`filmdec/offline_biped_test.go`, `filmdec/vehicle_creation_test.go`,
// `filmdec/vehicules_v11_scan_test.go`, `filmdec/event_list*_test.go`,
// `replay/vehicules_v*_test.go`). TROIS AUTRES enveloppes du chantier ont ete SUPPRIMEES au lieu
// d'etre inscrites ici, et PAS POUR LA MEME RAISON — la distinction compte, parce qu'elle dit ou
// est passee la fonctionnalite :
//
//	ScanFilmVehicleCreationsForBand  son APPELANT DE PRODUCTION A ETE MIGRE vers la forme film
//	                                 (`replay/build_vehicles.go` appelle desormais
//	                                 `filmdec.ScanVehicleCreationsForBand(fc, wr, band)`).
//	                                 L'enveloppe `dir` n'avait plus d'appelant DU TOUT : elle
//	                                 se supprime, elle ne s'interdit pas ;
//	ScanFilmKeyframeRecordSpans      arrivee SANS AUCUN APPELANT ;
//	ScanFilmVehicleOccupancy         idem — et c'etait l'unique consommateur de la precedente.
//
// Regle D2 dans les trois cas : une enveloppe sans appelant se supprime. Aucune methode ne porte
// ces quatre noms (verifie par
// `grep -rE '^func \([^)]+\) ScanFilm...\('` : aucune correspondance).
//
// TROIS NOMS AJOUTES A LA RECONCILIATION DU 2026-09-05 (merge de `feat/v75`, 65 commits) :
// `ScanFilmAbilityImpulses`, `ScanFilmAbilityCharges` et `ScanFilmTranslocatorTeleports`. Les
// trois balayages arrivaient de l'amont en ANCIENNE forme et etaient appeles EN PRODUCTION
// (`replay/build.go:308/457/473`) ; ils ont recu leur forme film (`ScanAbilityImpulses(fc)`,
// `ScanAbilityCharges(fc)`, `ScanTranslocatorTeleports(film, entry)`), et leurs enveloppes `dir`
// survivent parce que des tests les appellent (`replay/golden_inputs_test.go`,
// `filmdec/transloc_{exemption,positions}_film_test.go`). Aucune methode ne porte ces noms
// (verifie : `grep -rE '^func \([^)]+\) (ScanFilmAbilityImpulses|ScanFilmAbilityCharges|ScanFilmTranslocatorTeleports)\('`).
var enveloppesInterditesEnProduction = []string{
	// filmdec
	"ScanFilmAbilityCharges", "ScanFilmAbilityImpulses",
	"ScanFilmAbilityRanks", "ScanFilmBipedPickups", "ScanFilmBipedPositions", "ScanFilmCamoStates",
	"ScanFilmCarrierMarks", "ScanFilmEquipmentChanges", "ScanFilmEquipmentCreations",
	"ScanFilmEquipmentCreationsForBand", "ScanFilmEquipmentPlacements", "ScanFilmEquipmentState",
	"ScanFilmFireEvents", "ScanFilmGrappleReads", "ScanFilmGrenadeThrows",
	"ScanFilmGroundWeaponCreations", "ScanFilmGroundWeaponCreationsForBand",
	"ScanFilmHeldWeaponChanges", "ScanFilmInventoryDeltas", "ScanFilmKeyframeGroundWeapons",
	"ScanFilmKeyframeLoadouts", "ScanFilmManagedProperties", "ScanFilmNavpointRadial",
	"ScanFilmObjectives", "ScanFilmProjectiles", "ScanFilmTranslocatorTeleports",
	"ScanFilmUnitEquipment",
	"ScanFilmBipedAimOnly", "ScanFilmBipedPositionsForBand",
	"ScanFilmVehicleCreations", "ScanFilmVehicleEvents",
	"ScanFilmWorldObjectKeyframes", "ScanFilmWorldObjects", "ScanFilmWorldObjectsForBand",
	"ScanFilmZoomEvents",
	"DetectI0Layout", "EquipmentArchetypeDir", "CalibrateMPPWidths",
	"GroundWeaponSlotBand", "GroundWeaponPositions", "WorldObjectPositionsForBand",
	"ReadFilmChunk", "CountFilmChunks",
	// replay
	"ScanFilmDeaths", "ScanFilmClockOrigin", "ScanFilmPlayerIndices", "ScanFilmKeyframeInventory",
}

// paquetsDeProduction : les paquets ou l'appel d'une enveloppe est interdit. C'est la liste des
// SITES DE PRODUCTION de la regle D2 du plan, au paquet pres (item 1.9, complete le 2026-09-02 :
// les quatre derniers manquaient — ils ne peuvent pas appeler d'enveloppe aujourd'hui, et c'est
// precisement ce qu'un ratchet garde).
//
//	internal/analysis/replay        BuildFromFilm et les balayages du document
//	internal/replaybuild            la cuisson (BuildBytes / BuildMatch)
//	internal/analysis/objectiveevents  ses neuf points d'entree prennent un *filmsource.Film
//	internal/games/halo_infinite/film/killsource  Decode recoit le film deja charge
//	internal/sync/killcollector     positions.go : le pont disque a disparu a l'item 1.6
//	internal/api/wire               registry_replay_build.go : le cablage de l'API
//	cmd/zone-attribution            measure.go : charge le film UNE fois (item 1.6)
var paquetsDeProduction = []string{
	"internal/analysis/replay",
	"internal/replaybuild",
	"internal/analysis/objectiveevents",
	"internal/games/halo_infinite/film/killsource",
	"internal/sync/killcollector",
	"internal/api/wire",
	"cmd/zone-attribution",
}

// appelsDEnveloppeAutorises : L'ALLOWLIST FERMEE de la regle 3. Clef : `paquet/fichier.go ->
// enveloppe`. Elle est verifiee DANS LES DEUX SENS — un site en trop echoue, une entree MORTE
// aussi : une allowlist qui garde des entrees mortes ne mesure plus ce que le depot fait.
//
//	internal/sync/killcollector/hits.go
//	          ARRIVE PAR L'AMONT (merge de `feat/v75` du 2026-09-03). Le numerateur de precision
//	          par arme est DIR-BASE PAR CONCEPTION — son en-tete l'ecrit : il se greffe sur le
//	          film DEJA sur disque (`ConfigureFilmAccuracy(dir, ...)`, cache local), la ou la
//	          cuisson recoit des chunks en memoire. La passe est DESACTIVEE en production
//	          (Infinite ne declare pas `match.weapon.accuracy` depuis la remise du 2026-09-01, et
//	          `ConfigureFilmAccuracy` n'a aucun appelant hors tests) : aucune cuisson ne paie ces
//	          relectures aujourd'hui. Lui donner la forme film exige de creer les formes
//	          `Scan*(film)` de trois balayages neufs de l'amont — hors perimetre d'une
//	          reconciliation de branche. RETRAIT CIBLE : le lot qui rallume la precision par arme,
//	          ou celui qui migre `hits.go`. Consigne au registre des reports.
var appelsDEnveloppeAutorises = map[string]string{
	"internal/sync/killcollector/hits.go -> ReadFilmChunk":   "amont 2026-09-03, passe desactivee",
	"internal/sync/killcollector/hits.go -> CountFilmChunks": "amont 2026-09-03, passe desactivee",
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
	vus := map[string]bool{}
	var violations []string
	for _, pkg := range paquetsDeProduction {
		for nom, f := range fichiersGoNonTest(t, filepath.Join(racine, filepath.FromSlash(pkg))) {
			ast.Inspect(f, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				appele := nomAppele(call.Fun)
				if !interdites[appele] {
					return true
				}
				cle := pkg + "/" + nom + " -> " + appele
				vus[cle] = true
				if _, autorise := appelsDEnveloppeAutorises[cle]; !autorise {
					violations = append(violations, cle)
				}
				return true
			})
		}
	}
	var morts []string
	for cle := range appelsDEnveloppeAutorises {
		if !vus[cle] {
			morts = append(morts, cle)
		}
	}
	sort.Strings(violations)
	sort.Strings(morts)
	if len(violations) > 0 {
		t.Errorf("la production appelle une enveloppe `dir` :\n  %s\n"+
			"Ces enveloppes chargent un film ENTIER par appel (regle D2 de PLAN_CUISSON_PERF) : "+
			"elles existent pour les tests et les instruments de recherche. La cuisson recoit un "+
			"`*filmsource.Film` deja charge et appelle la forme film (`ScanXxx(film, ...)`).",
			strings.Join(violations, "\n  "))
	}
	if len(morts) > 0 {
		t.Errorf("entrees MORTES de `appelsDEnveloppeAutorises` (l'appel n'existe plus) :\n  %s\n"+
			"Les retirer.", strings.Join(morts, "\n  "))
	}
}

// sitesZlibAutorises : L'ALLOWLIST FERMEE des fichiers non-test qui importent `compress/zlib`
// dans tout `apps/go-api` (regle 4, item 1.9 du plan, 2026-09-02). Chemins relatifs a la racine
// du module, separateur `/`.
//
// POURQUOI UNE LISTE FERMEE ET PAS UNE INTERDICTION PAR PAQUET. Avant le lot 1, TROIS inflates
// divergents cohabitaient dans la chaine de cuisson (`filmdec`, `objectiveevents`,
// `killsource`) — et il a fallu une mesure sur 1 378 films pour prouver qu'ils voyaient les
// memes octets. Un quatrieme se serait ajoute sans bruit dans n'importe quel paquet : la regle
// par paquet ne l'aurait vu que si le paquet avait ete prevu. La liste ci-dessous est donc
// exhaustive et VERIFIEE DANS LES DEUX SENS — un site en trop echoue, un site disparu aussi
// (une allowlist qui garde des entrees mortes ne dit plus rien).
//
// LE SITE `cmd/replay-equiv/walkers.go` A DISPARU AU LOT 6 (2026-09-03), avec le mode `-walkers`
// lui-meme : il portait EN COPIE les trois marcheurs historiques et leur inflate pour la mesure
// 0.7, et leurs originaux n'existaient plus depuis les items 1.4/1.5 — la copie ne comparait
// donc plus qu'a elle-meme. La mesure reste figee au §2 de `MESURES_CUISSON_PERF.md`. Les entrees
// restantes sont PERMANENTES : aucune ne decompresse un film de cuisson.
var sitesZlibAutorises = map[string]string{
	"internal/analysis/filmsource/film.go": "L'UNIQUE inflate de la chaine de cuisson (D1). " +
		"Rend le PARTIEL sur flux tronque : un film Theater se termine parfois net.",
	"internal/analysis/highlight_event_parser.go": "Parseur autonome du fil des morts, appele " +
		"sur des blobs BRUTS ou zlib (sync/collect.go, engine_highlight_events.go, " +
		"convergence_backfill_events.go) : sa double tolerance date de l'incident du 2026-05-22.",
	"internal/sync/haloclient/halo_client_http.go": "Validation d'un chunk AU TELECHARGEMENT, " +
		"avant tout stockage : ce n'est pas un decodage de film.",
	"cmd/replay-worker/job.go": "Meme validation au telechargement, cote ouvrier.",
	"internal/hinavmesh/conteneur.go": "AUTRE DOMAINE (conteneurs de navmesh du jeu), " +
		"aucun rapport avec les chunks de film.",
	"cmd/fetch_film_chunks/main.go":    "Outil de RECHERCHE : telecharge et inspecte des chunks.",
	"cmd/diag_weapons_v3/positions.go": "Outil de DIAGNOSTIC des armes v3.",
	"cmd/rdata_weapon_scan/main.go":    "Outil de RECHERCHE (balayage de rdata).",
}

// dossiersIgnoresPourZlib : ce que le balayage du depot ne parse pas.
var dossiersIgnoresPourZlib = map[string]bool{
	"testdata": true, "node_modules": true, "vendor": true, ".git": true,
}

// TestAllowlistZlibFermee — REGLE 4.
func TestAllowlistZlibFermee(t *testing.T) {
	racine := apiRootDepuisIci(t)
	fset := token.NewFileSet()
	trouves := map[string]bool{}
	err := filepath.WalkDir(racine, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if dossiersIgnoresPourZlib[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if perr != nil {
			return perr
		}
		for _, imp := range f.Imports {
			if strings.Trim(imp.Path.Value, `"`) != "compress/zlib" {
				continue
			}
			rel, rerr := filepath.Rel(racine, path)
			if rerr != nil {
				return rerr
			}
			trouves[filepath.ToSlash(rel)] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("balayage du depot : %v", err)
	}
	var enTrop, disparus []string
	for site := range trouves {
		if _, ok := sitesZlibAutorises[site]; !ok {
			enTrop = append(enTrop, site)
		}
	}
	for site := range sitesZlibAutorises {
		if !trouves[site] {
			disparus = append(disparus, site)
		}
	}
	sort.Strings(enTrop)
	sort.Strings(disparus)
	if len(enTrop) > 0 {
		t.Errorf("NOUVEL inflate hors allowlist :\n  %s\n"+
			"Le film se decompresse UNE fois, par `filmsource` (lot 1 de PLAN_CUISSON_PERF). "+
			"Si ce site ne decompresse PAS un film (telechargement, autre domaine, outil de "+
			"recherche), l'ajouter a `sitesZlibAutorises` avec sa justification et sa date.",
			strings.Join(enTrop, "\n  "))
	}
	if len(disparus) > 0 {
		t.Errorf("entrees MORTES de l'allowlist (le fichier n'importe plus zlib) :\n  %s\n"+
			"Les retirer : une allowlist qui garde des entrees mortes n'est plus une mesure de "+
			"ce que le depot fait, et le lot 6 ne saurait plus ce qu'il reste a fermer.",
			strings.Join(disparus, "\n  "))
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
