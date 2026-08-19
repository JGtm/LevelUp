package replay

// map_weapon_pads.go — LES EMPLACEMENTS DE SOCLE DE LA CARTE, CROISÉS AVEC LE MATCH, tels
// que l'API les sert AVEC le document de rejeu (ReplayDocument.MapWeaponPads).
//
// LA DÉCISION PRODUIT, mot pour mot (utilisateur, 2026-08-19) : « on ne les affiche que si
// allumés ». Elle tranche le piège que la mesure avait laissé ouvert.
//
// LE PIÈGE, ET IL EST SÉRIEUX. Le fichier de carte POSE les socles ; le mode les ALLUME.
// Cliffhanger porte DIX-SEPT emplacements au fichier : dix sont servis en CTF, et ZÉRO en
// Super Fiesta — l'artefact de ce match n'a même pas de clé `weaponPads`. Publier le
// catalogue tel quel afficherait dix-sept socles fantômes sur ce rejeu-là.
//
// LA RÈGLE QUI EN DÉCOULE, et elle est simple à énoncer : un emplacement du catalogue ne
// part au client QUE si un socle PUBLIÉ DU MATCH le confirme, à moins d'un mètre. Les
// autres sont ÉTIQUETÉS inactifs et restent au serveur. Le compte du catalogue, lui, part
// (`catalogN`) : publier dix emplacements sans dire que la carte en porte dix-sept
// laisserait croire à l'exhaustivité — c'est la même règle que `Coverage`.
//
// CE QUE LE CROISEMENT APPORTE, ET CE QU'IL N'APPORTE PAS :
//
//	APPORTE   la POSITION DU SPAWNER, connue dès la première image et au centimètre, là où
//	          le film ne donne que le centroïde des apparitions qu'il a vues. Et un
//	          emplacement CONFIRMÉ, donc dessinable sans attendre quoi que ce soit.
//	N'APPORTE PAS la PRÉSENCE. Ce qui est sur le socle, et quand, reste la mesure du film :
//	          le catalogue ne sait pas ce qui apparaît (un même objet porte l'épée ou le
//	          marteau selon le match) ni à quel instant. Les états plein / incertain / vide
//	          et le compte à rebours ne changent pas d'un iota.
//
// CE N'EST PAS UN CHAMP DE L'ARTEFACT, exactement comme `mapObjectives` et pour la même
// raison : l'artefact est décodé des seuls chunks du film, qui ne nomment ni la carte ni le
// mode. Le champ se remplit À LA REQUÊTE, au service, et le SchemaVersion de l'artefact ne
// bouge pas — rien n'a changé dans l'artefact.

import "levelup/go-api/internal/analysis/replay/mapvar"

// MapWeaponPadMatchM est le rayon de confirmation, en mètres. Le même seuil que la mesure
// du plan, écrit AVANT elle : 32 positions d'oracle sur trois cartes, 32 appariées, médiane
// 0,01 m. À cette résolution, un mètre n'est pas une tolérance, c'est une marge.
const MapWeaponPadMatchM = 1.0

// MapWeaponPads est le calque des emplacements de socle servi avec le rejeu.
type MapWeaponPads struct {
	// Pads : les emplacements ALLUMÉS, c'est-à-dire ceux qu'un socle du match confirme.
	// Jamais les autres.
	Pads []MapWeaponPadDTO `json:"pads"`
	// CatalogN est le nombre d'emplacements que la carte porte AU FICHIER, confirmés ou
	// non. Il dit ce que le calque n'affiche pas : Cliffhanger en CTF sert dix Pads pour
	// un CatalogN de dix-sept.
	CatalogN int `json:"catalogN"`
}

// MapWeaponPadDTO est UN emplacement allumé : la position du fichier de carte, et le socle
// du match qui l'a confirmé.
//
// LE LIEN VERS LE SOCLE DU MATCH EST L'ESSENTIEL. Il dit au client quel socle de
// `weaponPads` dessiner À CETTE POSITION-LÀ : la présence, la famille d'arme, le cycle et
// les états viennent tous du match. Un socle de `weaponPads` qu'AUCUN emplacement ne cite
// reste publié tel quel et se dessine comme avant — le film fait foi, le catalogue
// complète.
type MapWeaponPadDTO struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z,omitempty"`
	// Pad est l'index du socle confirmant dans `weaponPads`.
	Pad int `json:"pad"`
}

// BuildMapWeaponPads croise les emplacements d'une carte avec les socles d'un match.
//
// UN SOCLE DU MATCH NE CONFIRME QU'UN SEUL EMPLACEMENT (le plus proche l'emporte, et il
// est ensuite pris) : deux emplacements du fichier à moins d'un mètre l'un de l'autre ne
// doivent pas se réclamer du même socle et se dessiner deux fois au même endroit.
//
// Rend nil quand rien n'est allumé — le champ du document reste absent, et le client
// retombe sur son comportement d'avant : dessiner les socles du film. C'est la dégradation
// par absence, la même que partout ailleurs dans ce document.
func BuildMapWeaponPads(e MapWeaponPadsEntry, pads []WeaponPad) *MapWeaponPads {
	out := &MapWeaponPads{CatalogN: len(e.Pads)}
	pris := make([]bool, len(pads))
	for _, spot := range e.Pads {
		i := confirmePar(spot.Pos, pads, pris)
		if i < 0 {
			continue // ÉTEINT : le film ne l'a pas vu, il ne part pas au client.
		}
		pris[i] = true
		out.Pads = append(out.Pads, MapWeaponPadDTO{
			X: float32(spot.Pos.X), Y: float32(spot.Pos.Y), Z: float32(spot.Pos.Z), Pad: i,
		})
	}
	if len(out.Pads) == 0 {
		return nil
	}
	return out
}

// confirmePar rend l'index du socle de match le plus proche d'une position, à moins de
// MapWeaponPadMatchM et non encore pris ; -1 sinon.
func confirmePar(pos mapvar.Vec3, pads []WeaponPad, pris []bool) int {
	best, bestD := -1, MapWeaponPadMatchM
	for i, p := range pads {
		if pris[i] {
			continue
		}
		d := mapvar.Dist3(pos, mapvar.Vec3{X: float64(p.X), Y: float64(p.Y), Z: float64(p.Z)})
		if d < bestD {
			best, bestD = i, d
		}
	}
	return best
}
