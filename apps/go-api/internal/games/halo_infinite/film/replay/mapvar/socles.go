package mapvar

// socles.go — LES EMPLACEMENTS DE SOCLE d'une variante de carte, et la famille que leur
// type_id trahit.
//
// CE QUE MESURE LE PLAN `.ai/V7.5/replay2d/PLAN_SOCLES_MVAR.md` (2026-08-19), et rien de
// plus : trois type_id du fichier de carte portent les socles d'arme et de power-up. Sur
// trois cartes et 32 positions d'oracle tirées d'artefacts de rejeu cuits, 32 sont
// appariées à moins d'un mètre, médiane 0,01 m — le centimètre. Le témoin négatif (tirages
// aléatoires dans l'emprise) rend 0,4 à 5,6 % sur les cartes DEV.
//
// CE QUE LE FICHIER NE DIT PAS, ET QU'ON NE DEVINE PAS :
//
//	L'ARME.   Un MÊME objet porte l'épée ou le marteau selon le match. L'arme appartient au
//	          match, le socle appartient à la carte. Aucun champ de l'objet ne la nomme.
//	L'ÉTAT.   Le fichier POSE les socles ; le mode les ALLUME. Cliffhanger porte 17 socles
//	          au fichier, en rend 10 en CTF et ZÉRO en Super Fiesta — et rien dans le
//	          `.mvar` d'une carte DEV n'explique l'extinction (ces objets ne portent aucun
//	          label, donc aucun filtre `*_include` / `*_exclude`). Publier ce catalogue tel
//	          quel afficherait 17 socles fantômes sur un rejeu de Fiesta : le croisement
//	          avec le film est OBLIGATOIRE en aval (cf. replay/map_weapon_pads.go).
//
// LA FAMILLE EST UNE INFÉRENCE, mesurée par corrélation avec les armes observées sur trois
// cartes et huit films. Elle se publie À CÔTÉ du type_id brut, jamais à sa place : le jour
// où elle est infirmée, on recalcule sans ré-extraire un seul fichier.

import (
	"math"
	"sort"
)

// PadFamily est la famille d'un socle, DÉRIVÉE du type_id — jamais lue dans le fichier.
type PadFamily string

const (
	// PadFamilyPower — les armes de POUVOIR : épée, marteau, SPNKr (M41 et Fuel Rod),
	// Cindershot, S7 Sniper (mesuré sur les trois films de Catalyst).
	PadFamilyPower PadFamily = "power"
	// PadFamilyRack — les armes de RÂTELIER : Bulldog, Disruptor, Mangler, Commando,
	// Vestige Carbine, BR75, Sentinel Beam.
	PadFamilyRack PadFamily = "rack"
	// PadFamilyPowerup — le socle de POWER-UP (surbouclier, camouflage). Seul de son type
	// sur Catalyst, et c'est celui que la voie `ti=37` du film a fini par retrouver.
	PadFamilyPowerup PadFamily = "powerup"
)

// padFamilies — LES TROIS TYPE_ID DE SOCLE, en hexadécimal parce que c'est ainsi qu'ils se
// reconnaissent d'un dump à l'autre. Ils ne figurent dans AUCUN index du dépôt : ils ont été
// isolés par appariement contre l'oracle des artefacts, pas lus dans une table du jeu.
//
// L'ÉCRITURE DÉCIMALE EST DONNÉE EN REGARD parce que l'inventaire du plan les nomme ainsi.
var padFamilies = map[int32]PadFamily{
	0x5F379533: PadFamilyPower,   // 1597478195
	0x6253CFC0: PadFamilyRack,    // 1649659840
	0x5E86D110: PadFamilyPowerup, // 1585893648
}

// PadFamilyOf rend la famille d'un type_id, et FAUX quand ce type n'est pas un socle.
// Un type inconnu n'est jamais promu socle par défaut : sur une carte Forge, 222 type_id
// cohabitent et « ressembler à un socle » ne veut rien dire.
func PadFamilyOf(typeID int32) (PadFamily, bool) {
	f, ok := padFamilies[typeID]
	return f, ok
}

// PadSpotMergeM est le rayon sous lequel deux objets de socle sont LE MÊME emplacement.
//
// POURQUOI IL FAUT REGROUPER, et le compte honnête en dépend : Catalyst déclare TREIZE
// objets de socle pour ONZE emplacements — deux d'entre eux sont à 4,7 cm et 9 mm d'un
// autre, soit le même emplacement déclaré deux fois, pas deux socles de plus. Le mètre est
// le même seuil que celui de l'appariement, écrit au plan AVANT la mesure.
const PadSpotMergeM = 1.0

// PadSpot est UN EMPLACEMENT de socle de la carte : une position, le type brut qui l'a
// désigné, la famille qu'on en déduit.
type PadSpot struct {
	// Pos est la position de l'objet REPRÉSENTATIF, en repère monde, mètres — le même
	// repère que les positions joueur du rejeu.
	Pos Vec3
	// TypeID est le type brut du représentant. Publié tel quel en aval, la famille à côté.
	TypeID int32
	// Family est l'inférence (cf. padFamilies).
	Family PadFamily
	// InstanceID est l'identifiant que le JEU donne au représentant — la seule identité
	// d'objet qui ne soit pas de notre invention (même règle que Zone.InstanceID).
	InstanceID int32
	// Objects est le nombre d'objets FUSIONNÉS dans cet emplacement. Deux sur les deux
	// doublons de Catalyst, un partout ailleurs sur les cartes DEV mesurées.
	Objects int
	// Mixed dit que des objets de familles DIFFÉRENTES ont été fusionnés ici. La famille
	// publiée est alors celle du représentant, et ce drapeau existe pour que le producteur
	// puisse le DIRE plutôt que de le taire — il n'a jamais été levé sur les cartes
	// mesurées, et c'est une mesure, pas une garantie.
	Mixed bool
}

// PadSpots rend les emplacements de socle d'une variante, dans un ordre DÉTERMINISTE.
//
// L'ORDRE EST SPATIAL (x, puis y, puis z, puis instance_id en dernier recours), le même que
// celui du catalogue d'objectifs : deux exécutions rendent le même fichier, et un diff git
// sur le catalogue se lit. L'ordre du fichier `.mvar`, lui, n'a aucun sens stable.
//
// LE REGROUPEMENT EST GLOUTON sur cet ordre : chaque objet rejoint le premier emplacement
// à moins de PadSpotMergeM, sinon il en ouvre un. Sur des socles espacés de plusieurs
// mètres (le cas mesuré partout sauf sur les doublons), le résultat ne dépend pas de la
// stratégie ; l'ordre déterministe garantit qu'il ne dépend pas non plus du fichier.
func PadSpots(v *Variant) []PadSpot {
	if v == nil {
		return nil
	}
	objs := make([]Object, 0, 16)
	for _, o := range v.Objects {
		if _, ok := PadFamilyOf(o.TypeID); ok {
			objs = append(objs, o)
		}
	}
	sort.SliceStable(objs, func(i, j int) bool {
		return lessPadSpot(objs[i], objs[j])
	})
	out := make([]PadSpot, 0, len(objs))
	for _, o := range objs {
		fam, _ := PadFamilyOf(o.TypeID)
		if i := nearestPadSpot(out, o.Pos); i >= 0 {
			out[i].Objects++
			out[i].Mixed = out[i].Mixed || out[i].Family != fam
			continue
		}
		out = append(out, PadSpot{
			Pos: o.Pos, TypeID: o.TypeID, Family: fam, InstanceID: o.InstanceID, Objects: 1,
		})
	}
	return out
}

// nearestPadSpot rend l'index du premier emplacement à moins de PadSpotMergeM, ou -1.
func nearestPadSpot(spots []PadSpot, p Vec3) int {
	for i := range spots {
		if Dist3(spots[i].Pos, p) < PadSpotMergeM {
			return i
		}
	}
	return -1
}

// lessPadSpot est l'ordre spatial des objets de socle (x, y, z, puis instance_id).
func lessPadSpot(a, b Object) bool {
	if a.Pos.X != b.Pos.X {
		return a.Pos.X < b.Pos.X
	}
	if a.Pos.Y != b.Pos.Y {
		return a.Pos.Y < b.Pos.Y
	}
	if a.Pos.Z != b.Pos.Z {
		return a.Pos.Z < b.Pos.Z
	}
	return a.InstanceID < b.InstanceID
}

// Dist3 est la distance euclidienne 3D entre deux points du repère monde, en mètres.
//
// TROIS DIMENSIONS ET NON DEUX, parce que c'est le critère de la mesure : l'appariement du
// plan compare les positions au centimètre en 3D, et deux socles superposés à des étages
// différents existent (Cliffhanger en porte). Un appariement XY seul les confondrait.
func Dist3(a, b Vec3) float64 {
	dx, dy, dz := a.X-b.X, a.Y-b.Y, a.Z-b.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}
