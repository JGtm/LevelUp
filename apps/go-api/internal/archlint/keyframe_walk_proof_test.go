package archlint

// keyframe_walk_proof_test.go — UNE MARCHE D'IMAGE-CLE DE PRODUCTION EST CELLE DU FILM (lot D-fix des
// retours du rejeu, 2026-09-24).
//
// # POURQUOI
//
// La marche d'ancres de la table d'image-cle ELIT son record suivant quand aucun voisin ne suit ;
// une fausse ancre de slot bas, elue, effacait les vrais records qui la precedaient (le joueur gere
// de l'index 0 de `bcb6d393` et `fb1a1a72`, que le lot M2 lisait alors ARRIVE plus tard). Depuis le
// lot D-fix, la marche DU FILM (`grammar.FilmContext.MarcheDImageCle`) refuse l'elu qu'un record
// prouve par la grammaire du film contredit. La preuve demande le registre du film : la marche SANS
// preuve (`grammar.WalkKeyframeWorld` et ses formes derivees) reste celle des INSTRUMENTS.
//
// Deux balayages d'un meme payload qui ne marcheraient pas pareil liraient deux tables differentes
// — un occupant present pour l'un, absent pour l'autre. CE GARDE-RAIL TIENT L'UNIFORMITE : dans les
// paquets de la chaine de cuisson, une forme SANS preuve ne s'appelle que depuis l'allowlist
// ci-dessous (instruments, enveloppes D2, et la chaine de precision par arme, a revision propre),
// et la valeur ZERO de `MarcheDImageCle` ne s'ecrit NULLE PART : les deux entrees des instruments
// marchent par `walkKeyframeWorldStats`, qui ne prend pas de preuve.
//
// VERIFIE DANS LES DEUX SENS : un appel en trop echoue, une entree MORTE de l'allowlist aussi.

import (
	"go/ast"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// paquetsDeLaMarche : les paquets de la chaine de cuisson qui marchent des images-cles.
var paquetsDeLaMarche = []string{
	"internal/games/halo_infinite/film/internal/grammar",
	"internal/games/halo_infinite/film/internal/facts/killsource",
	"internal/games/halo_infinite/film/internal/facts/objectives",
	"internal/games/halo_infinite/film/replay",
}

// formesSansPreuve : les entrees qui marchent une image-cle SANS la preuve du film.
var formesSansPreuve = map[string]bool{
	"WalkKeyframeWorld":        true,
	"WalkKeyframeWorldStats":   true,
	"walkKeyframeWorldFenetre": true,
	"walkKeyframeWorldStats":   true,
	"marcherLaTable":           true,
	"keyframeBornesToutes":     true,
	"keyframeBornes":           true,
	"familiesByRecord":         true,
	"keyframeInventories":      true,
	"invRecordSpans":           true,
}

// appelsSansPreuveAutorises : L'ALLOWLIST FERMEE (2026-09-24). Cle : `fichier.go/fonction -> appele`.
var appelsSansPreuveAutorises = map[string]string{
	"keyframe_world_marche.go/(MarcheDImageCle).Marcher -> marcherLaTable": "LA marche du film : elle passe sa preuve " +
		"au corps du balayeur",
	"keyframe_world_marche.go/walkKeyframeWorldStats -> marcherLaTable": "le corps SANS preuve, " +
		"reserve aux instruments (fenetre explicite)",
	"keyframe_world_marche.go/walkKeyframeWorldFenetre -> walkKeyframeWorldStats": "instrument a " +
		"fenetre explicite (lots 5.20.1 et M3.1)",
	"keyframe_world_marche.go/WalkKeyframeWorld -> walkKeyframeWorldFenetre": "l'entree des " +
		"instruments",
	"keyframe_world_marche.go/WalkKeyframeWorldStats -> walkKeyframeWorldStats": "l'entree des " +
		"instruments, avec leurs decisions",
	"keyframe_closure.go/keyframeBornesToutes -> WalkKeyframeWorld": "la forme INSTRUMENT des bornes ; " +
		"la production borne les records de la marche de son film (`keyframeBornesDe`)",
	"keyframe_closure.go/keyframeBornes -> keyframeBornesToutes": "les bornes fermables des instruments",
	"keyframe_loadout.go/familiesByRecord -> WalkKeyframeWorld": "forme instrument ; la production passe " +
		"par `familiesByRecordRecs` sur les records de la marche du film",
	"keyframe_ground_weapons.go/keyframeGroundWeapons -> familiesByRecord": "enveloppe D2 " +
		"`ScanFilmKeyframeGroundWeapons` et tests : aucun appelant de production (2026-09-24)",
	"keyframe_entity_queue.go/MeasureKeyframeAnchors -> WalkKeyframeWorld": "instrument de mesure, " +
		"aucun appelant de production (2026-09-24)",
	"keyframe_record_spans.go/KeyframeRecordSpans -> WalkKeyframeWorld": "instrument (emprise du lot " +
		"V5), aucun appelant de production (2026-09-24)",
	"keyframe_world.go/WorldFromKeyframe -> WalkKeyframeWorld": "aucun appelant de production (2026-09-24)",
	"weapon_hits.go/ScanFilmWeaponDamages -> WalkKeyframeWorld": "chaine de PRECISION PAR ARME " +
		"(`sync/killcollector/hits.go`) : revision propre `WeaponHitDistanceDecoderRev`, cadre par " +
		"defaut, film relu depuis le disque. Y brancher la preuve change ses lignes en base — decision " +
		"de backfill hors de la campagne des retours du rejeu (decouverte du lot D-fix, 2026-09-24)",
	"inventory_decode.go/keyframeInventories -> invRecordSpans": "forme instrument ; la cuisson passe par " +
		"`keyframeInventoriesDe` et `invRecordSpansDe` sur les records de la marche du film",
	"inventory_decode.go/invRecordSpans -> WalkKeyframeWorld": "forme instrument (cf. ci-dessus)",
}

// marchesZeroAutorisees : les sites ou la valeur ZERO de `MarcheDImageCle` s'ecrit — AUCUN (2026-09-24).
// Une marche de production sans preuve qui se deguiserait en `MarcheDImageCle{}` echapperait a
// l'allowlist des formes sans preuve : elle echoue ici.
var marchesZeroAutorisees = map[string]bool{}

func TestMarcheDImageCleDeProductionEstCelleDuFilm(t *testing.T) {
	racine := apiRootDepuisIci(t)
	vus, zeros := map[string]bool{}, map[string]bool{}
	for _, pkg := range paquetsDeLaMarche {
		for nom, f := range fichiersGoNonTest(t, filepath.Join(racine, filepath.FromSlash(pkg))) {
			for _, d := range f.Decls {
				fn, ok := d.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				porteur := nom + "/" + nomDeFonction(fn)
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					switch x := n.(type) {
					case *ast.CallExpr:
						if appele := nomAppele(x.Fun); formesSansPreuve[appele] {
							vus[porteur+" -> "+appele] = true
						}
					case *ast.CompositeLit:
						if nomAppele(x.Type) == "MarcheDImageCle" && len(x.Elts) == 0 {
							zeros[porteur] = true
						}
					}
					return true
				})
			}
		}
	}
	verifierAllowlist(t, vus, appelsSansPreuveAutorises, "une marche d'image-cle SANS preuve")
	zerosAutorises := map[string]string{}
	for k := range marchesZeroAutorisees {
		zerosAutorises[k] = "entree des instruments"
	}
	verifierAllowlist(t, zeros, zerosAutorises, "la valeur ZERO de `MarcheDImageCle`")
}

// verifierAllowlist compare les sites trouves a l'allowlist, dans les deux sens.
func verifierAllowlist(t *testing.T, vus map[string]bool, allow map[string]string, quoi string) {
	t.Helper()
	var enTrop, morts []string
	for cle := range vus {
		if _, ok := allow[cle]; !ok {
			enTrop = append(enTrop, cle)
		}
	}
	for cle := range allow {
		if !vus[cle] {
			morts = append(morts, cle)
		}
	}
	sort.Strings(enTrop)
	sort.Strings(morts)
	if len(enTrop) > 0 {
		t.Errorf("%s en production, hors allowlist :\n  %s\n"+
			"Un balayage de la cuisson marche par le contexte de son film : "+
			"`fc.MarcheDImageCle().Records(pay)` (ou `.Marcher(pay)`), jamais par la forme sans "+
			"preuve — deux balayages d'un meme payload doivent lire la meme table.", quoi,
			strings.Join(enTrop, "\n  "))
	}
	if len(morts) > 0 {
		t.Errorf("entrees MORTES de l'allowlist (%s) :\n  %s\nLes retirer.", quoi, strings.Join(morts, "\n  "))
	}
}
