package replay

// ground_weapon_pads_releve.go — UN SOCLE D ARME DONT L ARME APPARAIT HORS DE LA CARTE SE RELEVE
// LA OU ELLE EST PRISE.
//
// # CE QUE LE FILM ECRIT, ET CE QU IL N ECRIT PAS
//
// Un socle est le lieu ou des armes de meme famille reapparaissent, c est-a-dire la position de leur
// record de CREATION (ground_weapon_pads.go). Sur une carte Forge, le script peut creer l arme loin
// sous le niveau puis la poser sur son socle : le film ecrit la creation, pas la pose. Aucune piste
// delta ne suit alors l objet, et les images-cles ne portent pas la position d une arme au sol
// (keyframe_ground_weapons.go). Le socle publie tombe hors de la carte, et aucun emplacement de la
// reference ne le confirme ([BuildMapWeaponPads]).
//
// Ce que le film ecrit, c est la PRISE : l evenement natif `biped_pickup`, qui date une occupation du
// socle et nomme son ramasseur (pad_pickup_dating.go), et la position de ce ramasseur a cet instant.
//
// # LA REGLE (`repli_socle_hors_emprise_au_lieu_des_prises`)
//
// Un socle d ARME dont la position tombe hors de l emprise jouee du film (emprise_jouee.go, la meme
// que celle des positions et des vehicules) est releve au centroide des positions de ses ramasseurs,
// pris aux occupations que l evenement natif date, quand :
//   - au moins [gwPadMinHits] occupations datees le localisent — la recurrence qui fait un socle ;
//   - toutes tombent a moins de [originDropMaxDist] de ce centroide — le rayon ou un joueur prend
//     un objet, celui qui date deja la disparition d une arme au sol.
//
// Sinon le socle reste ou le film l a mis, et la raison se compte : trop peu de prises datees, ou des
// prises en PLUSIEURS lieux — un meme point de creation sert alors plusieurs socles de la carte, et un
// seul releve mentirait sur les autres. Rien d autre ne bouge : apparitions, presence, occupations et
// cycle restent ceux du socle.

import (
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// entreesDuReleve : ce que la regle consomme, deja lu ailleurs — jamais relu.
type entreesDuReleve struct {
	// natifs : le canal natif BRUT, pour l horodatage exact et le slot du ramasseur.
	natifs []types.BipedPickup
	// position rend la position d un ramasseur a un instant ([pickupOriginJudge.positionDe]).
	position func(slot uint32, tsUS uint64) (x, y, z float32, ok bool)
	emprise  empriseJouee
	clock    replayClock
}

// releverLesSoclesHorsEmprise applique la regle a chaque socle d arme et pose ses comptes sur la
// couverture. Modifie `pads` en place ; rend le nombre de socles releves.
func releverLesSoclesHorsEmprise(pads []WeaponPad, picks []PadPickup, in entreesDuReleve,
	cov *GroundWeaponCoverage,
) int {
	for p := range pads {
		famille, arme := PadWeaponFamilyKey(pads[p].Weapon)
		if !arme || !in.emprise.rejette(pads[p].X, pads[p].Y, pads[p].Z) {
			continue
		}
		cov.HorsEmprise++
		lieux := lieuxDesPrisesDatees(p, famille, picks, in)
		if len(lieux) < gwPadMinHits {
			continue
		}
		c, unSeulLieu := centroideDUnSeulLieu(lieux)
		if !unSeulLieu {
			cov.PlusieursLieux++
			continue
		}
		pads[p].X, pads[p].Y, pads[p].Z = c[0], c[1], c[2]
		cov.Releves++
	}
	return cov.Releves
}

// lieuxDesPrisesDatees rend la position du ramasseur de chaque occupation DATEE du socle `p`. Le
// ramassage natif qui l a datee se retrouve par sa famille et sa frame ; s il n est pas unique, ou si
// le ramasseur n a pas de position assez proche dans le temps, l occupation ne localise rien.
func lieuxDesPrisesDatees(p int, famille string, picks []PadPickup, in entreesDuReleve) [][3]float32 {
	var lieux [][3]float32
	for _, k := range picks {
		if k.Pad != p || k.T == nil {
			continue
		}
		r, ok := ramassageNatifA(*k.T, famille, in)
		if !ok {
			continue
		}
		if x, y, z, ok := in.position(r.Slot, r.TimestampUS); ok {
			lieux = append(lieux, [3]float32{x, y, z})
		}
	}
	return lieux
}

// ramassageNatifA rend le ramassage natif d ARME de la famille `famille` tombe a la frame `t`, quand
// il est unique.
func ramassageNatifA(t int, famille string, in entreesDuReleve) (types.BipedPickup, bool) {
	var trouve types.BipedPickup
	n := 0
	for _, r := range in.natifs {
		if !grammar.BipedPickupIsWeaponClass(r.Class) || fmt.Sprintf("%08x", r.CatalogID) != famille ||
			frameOf(r.TimestampUS, in.clock.origin, in.clock.step) != t {
			continue
		}
		trouve, n = r, n+1
	}
	return trouve, n == 1
}

// centroideDUnSeulLieu rend le centroide des lieux, et vrai quand tous tombent a moins de
// [originDropMaxDist] de lui.
func centroideDUnSeulLieu(lieux [][3]float32) ([3]float32, bool) {
	var s [3]float64
	for _, l := range lieux {
		s[0], s[1], s[2] = s[0]+float64(l[0]), s[1]+float64(l[1]), s[2]+float64(l[2])
	}
	n := float64(len(lieux))
	c := [3]float32{float32(s[0] / n), float32(s[1] / n), float32(s[2] / n)}
	for _, l := range lieux {
		if dist3(l, c) >= originDropMaxDist {
			return c, false
		}
	}
	return c, true
}
