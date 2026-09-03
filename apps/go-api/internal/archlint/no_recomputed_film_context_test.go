// no_recomputed_film_context_test.go — LA BANDE, LE DECOUPAGE ET LE REGISTRE SE CALCULENT UNE FOIS.
//
// # CE QUE CE GARDE-RAIL PROTEGE
//
// Lot 2 de PLAN_CUISSON_PERF (2026-09-03). Le lot 1 avait supprime les ~36 relectures du film ;
// restait le second etage du meme defaut — sur le film DEJA CHARGE, chaque balayage recalculait
// pour son propre compte les trois memes derivations, qui ne dependent pourtant que du film :
//
//	`bipedSlotBand`       la bande de slots bipede — HUIT releves par cuisson (positions,
//	                      ramassages natifs, et les six canaux delta), chacun une marche de
//	                      l'image-cle de tete de chaque chunk ;
//	`DetectI0LayoutOf`    le decoupage d'i0 — SIX detections, chacune six chunks marches bit a
//	                      bit, plus sa propre bande ;
//	`ParseRegistryChunk`  le registre ECS de `chunk_00` — une DOUZAINE d'analyses, une par
//	                      accesseur d'archetype.
//
// `filmdec.FilmContext` les porte desormais, construit une fois par `replay.BuildFromFilm` et
// passe aux balayages. Rien de tout cela ne tient si un balayage rajoute demain « juste un
// `bipedSlotBand` » parce que c'est plus court a ecrire que de faire descendre le contexte : la
// regression serait invisible (memes sorties, exactement) et ne se verrait qu'au chronometre.
//
// # LES DEUX REGLES
//
//  1. Dans `filmdec` (hors _test), les trois calculs ne s'appellent QUE depuis l'allowlist
//     ci-dessous — le contexte, et les deux sites qui calculent une valeur DIFFERENTE, ecrite.
//  2. Hors `filmdec`, aucun paquet de PRODUCTION de la chaine de cuisson n'analyse le registre
//     lui-meme (`filmdec.ParseRegistryChunk`), sauf l'allowlist datee.
//
// LES DEUX SONT VERIFIEES DANS LES DEUX SENS : un site en trop echoue, une entree MORTE de
// l'allowlist aussi. Une allowlist qui garde des entrees mortes ne mesure plus ce que le depot
// fait, et le lot suivant ne saurait plus ce qu'il reste a fermer.
//
// # POURQUOI go/ast ET PAS UN GREP
//
// `filmdec` CITE ces trois noms dans ses commentaires — abondamment, puisque c'est la migration
// qu'il documente. Un test grep rougirait sur la documentation du garde-rail lui-meme. Le test
// parse donc les fichiers et ne regarde que les APPELS, en nommant la FONCTION ENGLOBANTE : une
// allowlist par fichier laisserait passer un second appel ajoute dans le meme fichier.
package archlint

import (
	"go/ast"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// calculsDuContexteFilm : les trois derivations que `filmdec.FilmContext` memorise, plus
// l'enveloppe D2 du decoupage (`DetectI0Layout(dir)`, qui charge un film entier pour le detecter).
var calculsDuContexteFilm = map[string]bool{
	"bipedSlotBand":      true,
	"DetectI0LayoutOf":   true,
	"DetectI0Layout":     true,
	"ParseRegistryChunk": true,
}

// appelsAutorisesDuContexte : L'ALLOWLIST FERMEE de la regle 1 (2026-09-03, lot 2). Clef :
// `fichier.go/fonction -> calcul`. Les methodes portent leur recepteur, `(*FilmContext).BipedSlots`.
//
//	film_context.go       LE CONTEXTE LUI-MEME : les trois calculs y ont lieu une fois par film,
//	                      a la premiere demande, et y sont memorises. C'est le seul site qui
//	                      calcule POUR TOUT LE MONDE.
//	i0_layout.go          `DetectI0LayoutOf` releve SA PROPRE bande, sur les SIX PREMIERS chunks
//	                      seulement (`detectMaxChunks`) : ce n'est PAS la bande du contexte, qui
//	                      couvre tous les chunks de donnees — deux valeurs differentes, deux
//	                      calculs, et les partager changerait la detection. `DetectI0Layout(dir)`
//	                      est l'enveloppe D2, hors production (regle 3 de no_film_reread_test.go).
//	offline_biped.go      `ScanBipedPositions` releve sa bande sur `opt.Chunks` (une SOUS-LISTE
//	                      quand l'appelant la restreint) et ne detecte le decoupage que si
//	                      `opt.Layout` est nil — en cuisson il vient du CATALOGUE, donc la
//	                      detection n'a pas lieu. Ses deux valeurs sont conditionnelles a des
//	                      options : les brancher sur le contexte demanderait de trancher quand
//	                      la sous-liste diverge, ce qui n'est pas un refacto pur. HORS PERIMETRE
//	                      DU LOT 2, note au §8 du plan ; RETRAIT CIBLE : lot 4, ou la bande
//	                      change de representation (tableau indexe, item 4.1).
//
//	weapon_hit_distance_resolver.go
//	                      ARRIVE PAR L'AMONT (merge de `feat/v75` du 2026-09-03, chantier
//	                      « precision par arme » remise le 2026-09-01). `DetectFilmWorldRange`
//	                      resout les bornes monde d'une carte par la SIGNATURE de largeurs d'axe
//	                      du decoupage i0 : il n'a ni film charge ni contexte, seulement un
//	                      repertoire, et son unique appelant (`sync/killcollector/hits.go`) est
//	                      une passe DESACTIVEE en production (capability `match.weapon.accuracy`
//	                      NON declaree par Infinite depuis la remise, et `ConfigureFilmAccuracy`
//	                      sans appelant de production). Entree posee par la RECONCILIATION, pas
//	                      par un lot : la migrer vers la forme film demande des formes
//	                      `Scan*(film)` pour trois balayages neufs de l'amont — hors perimetre du
//	                      merge, consigne au registre des reports. RETRAIT CIBLE : le lot qui
//	                      rallume la precision par arme, ou celui qui migre `hits.go`.
var appelsAutorisesDuContexte = map[string]string{
	"film_context.go/(*FilmContext).BipedSlots -> bipedSlotBand":    "le releve unique de la bande du film",
	"film_context.go/(*FilmContext).I0Layout -> DetectI0LayoutOf":   "la detection unique du decoupage d'i0",
	"film_context.go/(*FilmContext).Registry -> ParseRegistryChunk": "l'analyse unique du registre chunk_00",
	"i0_layout.go/DetectI0LayoutOf -> bipedSlotBand":                "bande REDUITE aux 6 premiers chunks : autre valeur",
	"i0_layout.go/DetectI0Layout -> DetectI0LayoutOf":               "enveloppe D2, hors production",
	"offline_biped.go/ScanBipedPositions -> bipedSlotBand":          "bande sur opt.Chunks : hors perimetre du lot 2",
	"offline_biped.go/ScanBipedPositions -> DetectI0LayoutOf":       "repli quand opt.Layout est nil : hors perimetre du lot 2",
	"weapon_hit_distance_resolver.go/DetectFilmWorldRange -> DetectI0Layout": "amont 2026-09-03 : " +
		"signature de largeurs d'axe depuis un repertoire, passe de precision par arme desactivee",
}

// TestContexteFilmCalculeUneFois — REGLE 1.
func TestContexteFilmCalculeUneFois(t *testing.T) {
	pkgDir := filepath.Join(apiRootDepuisIci(t), filepath.FromSlash("internal/analysis/filmdec"))
	vus := map[string]bool{}
	var enTrop []string
	for nom, f := range fichiersGoNonTest(t, pkgDir) {
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			porteur := nom + "/" + nomDeFonction(fn)
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				appele := nomAppele(call.Fun)
				if !calculsDuContexteFilm[appele] {
					return true
				}
				cle := porteur + " -> " + appele
				vus[cle] = true
				if _, autorise := appelsAutorisesDuContexte[cle]; !autorise {
					enTrop = append(enTrop, cle)
				}
				return true
			})
		}
	}
	var morts []string
	for cle := range appelsAutorisesDuContexte {
		if !vus[cle] {
			morts = append(morts, cle)
		}
	}
	sort.Strings(enTrop)
	sort.Strings(morts)
	if len(enTrop) > 0 {
		t.Errorf("un balayage recalcule ce que `filmdec.FilmContext` porte deja :\n  %s\n"+
			"La bande de slots bipede, le decoupage d'i0 et le registre chunk_00 se calculent UNE "+
			"fois par film (lot 2 de PLAN_CUISSON_PERF) : faire descendre le `*FilmContext` "+
			"jusqu'a ce balayage, et les lire par `fc.BipedSlots()` / `fc.I0Layout()` / "+
			"`fc.Registry()`. Une enveloppe D2 (`ScanFilmXxx(dir)`) ouvre son propre contexte : "+
			"`ScanXxx(NewFilmContext(film))`. Si ce site calcule vraiment une valeur DIFFERENTE, "+
			"l'ajouter a `appelsAutorisesDuContexte` en ecrivant EN QUOI elle differe, avec sa date.",
			strings.Join(enTrop, "\n  "))
	}
	if len(morts) > 0 {
		t.Errorf("entrees MORTES de l'allowlist (l'appel n'existe plus) :\n  %s\n"+
			"Les retirer : une allowlist qui garde des entrees mortes ne mesure plus ce que le "+
			"depot fait.", strings.Join(morts, "\n  "))
	}
}

// paquetsSansAnalyseDeRegistre : les paquets de la chaine de cuisson qui, hors `filmdec`, ne
// doivent pas analyser le registre eux-memes — la cuisson passe par le contexte du film.
var paquetsSansAnalyseDeRegistre = []string{
	"internal/analysis/replay",
	"internal/replaybuild",
	"internal/analysis/objectiveevents",
	"internal/games/halo_infinite/film/killsource",
	"internal/sync/killcollector",
	"internal/api/wire",
}

// analysesDeRegistreAutorisees : L'ALLOWLIST FERMEE de la regle 2 (2026-09-03, lot 2). Chemins
// relatifs a la racine du module, separateur `/`.
var analysesDeRegistreAutorisees = map[string]string{
	"internal/games/halo_infinite/film/killsource/world.go": "`World.Snapshot` analyse le " +
		"registre du film pour son propre monde. HORS PERIMETRE SANS CONDITION (decision D14 de " +
		"PLAN_CUISSON_PERF) : `killsource` n'est pas dans ce plan. Note §8 — c'est la DERNIERE " +
		"analyse de registre de la chaine de cuisson qui ne passe pas par `FilmContext`.",
	"internal/sync/killcollector/hits.go": "ARRIVE PAR L'AMONT (merge de `feat/v75` du " +
		"2026-09-03, chantier « precision par arme » remise le 2026-09-01). Cette passe rejoue le " +
		"film DEPUIS LE DISQUE (`ConfigureFilmAccuracy(dir, ...)`) et analyse chunk_00 pour ses " +
		"propres balayages ; elle est DESACTIVEE en production — Infinite ne declare pas la " +
		"capability `match.weapon.accuracy` et `ConfigureFilmAccuracy` n'a aucun appelant hors " +
		"tests. La migrer vers `FilmContext` exige de donner leur forme film a trois balayages " +
		"neufs de l'amont (`ScanFilmWeaponShots`, `ScanFilmWeaponDamages`, `BuildBipedTracks`) : " +
		"hors perimetre d'une reconciliation de branche, consigne au registre des reports. " +
		"RETRAIT CIBLE : le lot qui rallume la precision par arme, ou celui qui migre `hits.go`.",
}

// TestRegistreAnalyseParLeContexteSeul — REGLE 2.
func TestRegistreAnalyseParLeContexteSeul(t *testing.T) {
	racine := apiRootDepuisIci(t)
	trouves := map[string]bool{}
	for _, pkg := range paquetsSansAnalyseDeRegistre {
		for nom, f := range fichiersGoNonTest(t, filepath.Join(racine, filepath.FromSlash(pkg))) {
			ast.Inspect(f, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if nomAppele(call.Fun) == "ParseRegistryChunk" {
					trouves[pkg+"/"+nom] = true
				}
				return true
			})
		}
	}
	var enTrop, morts []string
	for site := range trouves {
		if _, ok := analysesDeRegistreAutorisees[site]; !ok {
			enTrop = append(enTrop, site)
		}
	}
	for site := range analysesDeRegistreAutorisees {
		if !trouves[site] {
			morts = append(morts, site)
		}
	}
	sort.Strings(enTrop)
	sort.Strings(morts)
	if len(enTrop) > 0 {
		t.Errorf("le registre du film est analyse hors du contexte :\n  %s\n"+
			"Le registre chunk_00 s'analyse UNE fois par film, dans `filmdec.FilmContext` "+
			"(lot 2 de PLAN_CUISSON_PERF) : lire `fc.Registry()` ou l'accesseur d'archetype qui "+
			"va avec, plutot que de refaire l'analyse.", strings.Join(enTrop, "\n  "))
	}
	if len(morts) > 0 {
		t.Errorf("entrees MORTES de l'allowlist (le fichier n'analyse plus le registre) :\n  %s\n"+
			"Les retirer.", strings.Join(morts, "\n  "))
	}
}

// nomDeFonction rend le nom d'une declaration de fonction, recepteur compris pour une methode :
// `ScanBipedPositions`, `(*FilmContext).BipedSlots`. Sans le recepteur, deux methodes homonymes
// de types differents se confondraient dans l'allowlist.
func nomDeFonction(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	return "(" + typeDuRecepteur(fn.Recv.List[0].Type) + ")." + fn.Name.Name
}

// typeDuRecepteur rend `*FilmContext` ou `FilmContext`.
func typeDuRecepteur(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return "*" + typeDuRecepteur(t.X)
	case *ast.Ident:
		return t.Name
	}
	return "?"
}
