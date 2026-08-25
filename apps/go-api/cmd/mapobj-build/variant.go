// cmd/mapobj-build — variant.go : choix de la variante `.mvar` d'une carte.
//
// POURQUOI CE FICHIER EXISTE. Un asset de carte expose plusieurs `.mvar`. Le choix se
// faisait au nombre d'objectifs, et ce critère se retourne contre lui-même sur les cartes
// bâties dans Forge : le canevas de base livré avec le jeu contient le RACK des objets de
// mode — un exemplaire de chaque, rangé hors du terrain — pendant que la carte réellement
// jouée vit dans l'autre fichier.
//
// Mesuré le 2026-08-08 sur Vagabond (asset 105f5d84) :
//
//	fo08_wetland.mvar :  100 objets,  20 objectifs, tous à z = 50,50 exactement,
//	                     étalés sur 8,20 m   (rack)
//	map.mvar          : 4709 objets,   4 objectifs (3 zones de Bastion + 1 crâne),
//	                     étalés sur 22,30 m  (terrain)
//
// Le critère du nombre retenait le rack : trois « zones de Bastion » de 1 m de rayon, à 2 m
// l'une de l'autre, au même millimètre d'altitude. Publier ça, c'est publier un objectif
// au mauvais endroit — ce que l'en-tête de fetch.go dit vouloir éviter. Ce que le rack et
// le terrain ne partagent pas, c'est l'ÉCHELLE de leurs objectifs : voir
// parkedAbsoluteSpreadM, qui porte la règle et l'histoire de ses deux calibrations.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// parkedAbsoluteSpreadM : sous cette emprise ABSOLUE, les objectifs d'une variante sont
// RANGÉS (rack du canevas) ; au-dessus, ils sont POSÉS sur un terrain.
//
// POURQUOI UN CRITÈRE ABSOLU (mesuré le 2026-08-20). Le garde-fou d'origine (2026-08-08)
// comparait l'emprise des objectifs à celle de TOUS les objets de la variante — un RAPPORT,
// seuil 5 %. Sur une carte bâtie dans Forge, cet ensemble n'est PAS le terrain : c'est le
// CANEVAS entier, décor lointain et volumes de bord compris. Le rapport répond alors à
// « les objectifs sont-ils petits devant le monde constructible ? », qui n'est pas la
// question que ce garde-fou pose. Les deux variantes d'Empyrean (asset d035fc3e) le
// montrent :
//
//	map.mvar (la carte JOUÉE)  : 5 297 objets sur 1 061,64 m, 10 objectifs sur 38,32 m
//	                             -> 3,609 %  ->  ÉCARTÉE À TORT par le rapport
//	fo11_blank.mvar (le rack)  :   100 objets sur   356,10 m, 25 objectifs sur 13,30 m
//	                             -> 3,735 %  ->  écartée, à raison
//
// Les deux rapports sont à 0,13 point l'un de l'autre : AUCUN recalibrage du seuil relatif
// ne les sépare. Ce qui les sépare est l'emprise ABSOLUE : un rack range un exemplaire de
// chaque objet de mode côte à côte, à l'échelle de l'OBJET ; un terrain place ses objectifs
// à l'échelle du JEU.
//
// POURQUOI LE RAPPORT A ÉTÉ RETIRÉ (mesuré le 2026-08-25, lot catalogue). Le correctif du
// 2026-08-20 avait gardé le rapport EN PLUS du plancher, en ET, par prudence : « le ET ne
// peut que relâcher le garde-fou ». C'est vrai, et c'est le défaut — le rapport garde un
// droit de VETO sur une détection de rack correcte. Il l'exerce sur un canevas ROGNÉ :
//
//	Dynasty  (cfd90b63) fo08_wetland.mvar :  82 objets sur 34,40 m, 25 objectifs sur 13,30 m
//	Kaiketsu (98a83f87) fo05_desert.mvar  :  82 objets sur 34,40 m, 25 objectifs sur 13,30 m
//
// Le rack est le MÊME (13,30 m, collines à z = 50,50, un exemplaire de chaque objet de
// mode), mais le canevas ne fait plus 356 m : il en fait 34,40. Le seuil relatif tombe donc
// à 1,72 m, le rack de 13,30 m passe au-dessus, et le ET le déclare POSÉ. Les deux cartes
// retenaient ainsi leur rack (25 objectifs) contre leur carte jouée (`map.mvar`, 5 540 et
// 5 497 objets, 13 et 16 objectifs) — cinq fausses collines et de fausses zones de Bastion
// d'un mètre de rayon chacune.
//
// Le plancher seul tranche le corpus ENTIER sans exception : sur les 73 entrées du
// catalogue, exactement deux tombent sous 17,0 m — les deux racks ci-dessus, à 13,30 m — et
// la suivante est à 21,21 m. Les quatre témoins de TestGrandCanevasNEcartePasLaCarteJouee
// (38,32 / 13,30 / 8,20 / 21,21 m) gardent tous leur verdict. Le rapport n'était donc plus
// défendu par aucune mesure, et il coûtait deux cartes : il est supprimé.
//
// Calibration : sur les entrées RETENUES du catalogue, la plus petite emprise d'objectifs
// réellement posés est 21,21 m (`cliffside_map`, 9 objectifs), puis 22,20 et 22,73 m ; le
// plus grand rack connu est 13,30 m (Empyrean, Dynasty, Kaiketsu), le plus petit 8,20 m
// (Vagabond). Le plancher est posé entre les deux (moyenne géométrique 16,79 m), avec un
// facteur ~1,27 de marge de part et d'autre. Une carte dont TOUTES les variantes tombent
// sous le plancher échoue bruyamment (« aucune variante exploitable », ingestRemote) : le
// faux positif est visible, jamais silencieux.
const parkedAbsoluteSpreadM = 17.0

// minObjectivesForSpread : à un seul objectif l'emprise vaut 0 et ne dit rien.
const minObjectivesForSpread = 2

// isParkedPalette dit si les objectifs d'une variante sont RANGÉS (rack du canevas) au
// lieu d'être POSÉS sur le terrain.
func isParkedPalette(v *mapvar.Variant) bool {
	objectives := v.Objectives()
	if len(objectives) < minObjectivesForSpread || len(v.Objects) == 0 {
		return false
	}
	gMinX, gMinY, gMaxX, gMaxY := inf, inf, -inf, -inf
	for _, o := range objectives {
		gMinX, gMaxX = min(gMinX, o.Pos.X), max(gMaxX, o.Pos.X)
		gMinY, gMaxY = min(gMinY, o.Pos.Y), max(gMaxY, o.Pos.Y)
	}
	return max(gMaxX-gMinX, gMaxY-gMinY) < parkedAbsoluteSpreadM
}

// inf sert de borne initiale aux min/max (pas de math.Inf pour rester en float64 littéral
// lisible dans les deux sens).
const inf = 1e18

// dumpedObject est la forme de DIAGNOSTIC d'un objet de variante.
//
// Ce n'est PAS le catalogue : `map_objectives.json` ne porte que les objets qui ont un
// RÔLE d'objectif. Sur une carte bâtie dans Forge, le décor lui-même est fait d'objets de
// variante (Vagabond : 4 709), et sans eux on ne peut ni relire la carte à l'oeil ni
// vérifier un repère de terrain — la revue visuelle du 2026-08-08 a buté exactement là.
// Le type_id n'est pas résolu en nom : cette table n'existe pas encore (question Q1 de la
// piste palette Forge), donc on publie l'entier brut plutôt qu'un libellé deviné.
//
// LES LABELS NON RÉSOLUS SONT PUBLIÉS EN BRUT (lot C-ter volet 2, 2026-08-19) : un dump qui
// ne montre que les labels que la table sait nommer cache exactement ce qu'un inventaire
// cherche — les objets de mode dont le label n'est PAS encore craqué (les collines de KOTH
// n'avaient aucun label résolu). Le hash brut se confronte ensuite à mapvar.LabelHash.
// L'instance_id voyage aussi : c'est la clé que le catalogue devra décider de porter.
type dumpedObject struct {
	Index      int           `json:"index"`
	TypeID     int32         `json:"type_id"`
	InstanceID int32         `json:"instance_id"`
	Pos        mapvar.Vec3   `json:"pos"`
	Up         mapvar.Vec3   `json:"up"`
	Forward    mapvar.Vec3   `json:"forward"`
	Team       int           `json:"team_index"`
	Labels     []string      `json:"labels,omitempty"`
	Unresolved []int32       `json:"unresolved_labels,omitempty"`
	Shape      *mapvar.Shape `json:"shape,omitempty"`
}

// dumpObjects écrit tous les objets d'une variante, pour inspection hors ligne.
func dumpObjects(path string, v *mapvar.Variant) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	blob, err := json.Marshal(dumpedObjectsOf(v))
	if err != nil {
		return err
	}
	return os.WriteFile(path, blob, 0o644)
}

// dumpedObjectsOf projette chaque objet de la variante en forme de diagnostic : labels
// résolus par leur nom, labels inconnus par leur hash brut — jamais tus.
func dumpedObjectsOf(v *mapvar.Variant) []dumpedObject {
	out := make([]dumpedObject, 0, len(v.Objects))
	for _, o := range v.Objects {
		d := dumpedObject{
			Index: o.Index, TypeID: o.TypeID, InstanceID: o.InstanceID, Pos: o.Pos, Up: o.Up,
			Forward: o.Forward, Team: o.TeamIndex, Shape: o.Shape(),
		}
		for _, h := range o.Labels {
			if n := mapvar.LabelName(h); n != "" {
				d.Labels = append(d.Labels, n)
				continue
			}
			d.Unresolved = append(d.Unresolved, h)
		}
		out = append(out, d)
	}
	return out
}
