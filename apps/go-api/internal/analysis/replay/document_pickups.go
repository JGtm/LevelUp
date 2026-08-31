package replay

// document_pickups.go — LES RAMASSAGES NATIFS, datés à la milliseconde et ATTRIBUÉS.
//
// CE QUE C'EST. L'événement `biped_pickup` de la bobine (type 9 de la liste d'événements d'un
// paquet delta, cf. filmdec/biped_pickups.go). Le moteur l'écrit quand un bipède ramasse
// quelque chose : il donne l'instant, le RAMASSEUR et l'identifiant de CATALOGUE de l'objet.
//
// CE QUE ÇA APPORTE, PAR RAPPORT AUX DEUX CANAUX EXISTANTS :
//
//   - `weaponChanges` (i43..i46) est précis mais son rappel est partiel : sur le film de
//     référence, les images-clés révèlent 14 arrivées d'arme dont 7 qu'il n'explique pas. Le
//     canal natif en nomme 5 sur 7 (puis 3 sur 3 sur le second film), contre un plancher de
//     hasard mesuré à 9-14 %. Et là où les deux canaux voient la même prise, ils s'accordent :
//     21/21 et 11/12 appariements arme nommée à moins de 500 ms, témoin décalé à 4,8 % et 0 %.
//   - `padPickups` ne publiait qu'un intervalle de vingt secondes SANS joueur. Le canal natif
//     donne l'instant exact et le ramasseur — c'est exactement l'« oracle plus rapproché que
//     20 s » que le contrat de `PadPickup.XUID` désignait comme condition de levée.
//
// CE QUE ÇA NE DONNE PAS :
//
//   - le SOCLE d'origine. L'événement porte l'identifiant de CATALOGUE de l'objet, pas un
//     handle du monde : l'hypothèse « la référence désigne l'objet » a été mesurée et RÉFUTÉE
//     (`512 + index` vaut le slot du RAMASSEUR sur 32/32 paires de vérité terrain). Le
//     rapprochement avec un socle reste l'affaire de la chaîne spatiale (`padPickups`).
//   - la COMPLÉTUDE. Le balayage ne voit que les événements EN TÊTE de leur liste ; un type 9
//     en deuxième position d'une liste ouverte par une autre famille lui échappe. C'est une
//     borne INFÉRIEURE, et la couverture la publie (`multiEvent`).

import (
	"fmt"
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
)

// PickupKind qualifie ce qui a été ramassé.
type PickupKind string

const (
	// PickupWeapon : une ARME. La classe le dit, et la séparation est MESURÉE : les classes 0
	// et 1 portent une famille d'arme connue du canal i43..i46 dans 63 à 72 % des cas.
	PickupWeapon PickupKind = "weapon"
	// PickupItem : autre chose qu'une arme — équipement, grenade, consommable. Les classes 2
	// et 3 portent une famille d'arme dans 0,0 % des cas, sur 118 événements de deux films.
	PickupItem PickupKind = "item"
)

// Pickup est UN ramassage : quand, qui, quoi.
type Pickup struct {
	// T est l'index de frame, sur le même axe que Point.T.
	T int `json:"t"`
	// Slot est le slot du bipède ramasseur : il désigne la Track concernée, donc une VIE.
	// Même convention que WeaponChange.Slot.
	Slot uint32 `json:"slot"`
	// XUID est l'identité du ramasseur, en décimal — même forme que Track.XUID. Absent quand
	// le pont slot -> joueur ne nomme pas cette vie ; l'événement reste publié, daté et
	// rattaché à sa Track, parce qu'un ramassage anonyme vaut mieux qu'un ramassage effacé.
	XUID string `json:"xuid,omitempty"`
	// W est l'identifiant de CATALOGUE de l'objet, en hexadécimal 8 chiffres MINUSCULES et
	// SANS préfixe — la convention de `WeaponChange.W`, et elle seule.
	//
	// PAS CELLE DE `Loadout.W`, ET LA NUANCE A COÛTÉ UN BOGUE. Le commentaire de ce champ
	// affirmait « même convention que `Loadout.W` et `WeaponChange.W` » : c'était faux de
	// moitié. `Loadout.W` et `WeaponPad.Weapon` passent par `formatWeaponFamily`, qui écrit
	// `"0x"` + huit MAJUSCULES. La datation des occupations de socle comparait les deux
	// espaces directement et ne trouvait donc JAMAIS rien (revue adversariale du 2026-08-31).
	// La jointure normalise désormais au point de comparaison (`padFamilyKey`) ; les formes
	// publiées, elles, ne bougent pas — des clients les lisent déjà.
	//
	// L'ESPACE DE VALEURS, lui, est bien commun : mesuré, 100 % des familles vues par i43..i46
	// figurent dans l'ensemble des identifiants du canal natif. Sur un objet non-arme, c'est un
	// identifiant que le catalogue d'armes ne nomme pas — publié quand même, brut : le nommer
	// viendra, l'effacer serait perdre.
	W string `json:"w"`
	// Kind dit si c'est une arme ou autre chose. C'est le champ sur lequel un client branche.
	Kind PickupKind `json:"kind"`
	// Class est le R(3) BRUT de la charge. Il est publié EN PLUS de Kind, et ce n'est pas une
	// redondance : ce qui distingue la classe 0 de la 1, et la 2 de la 3, n'est PAS établi. Le
	// jour où ce sera lu, les artefacts déjà cuits porteront la valeur — sinon il faudrait
	// tout recuire pour une information qui était là.
	Class int `json:"class"`
}

// buildPickups projette les ramassages lus dans le film sur l'axe de frames du document et
// pose l'identité du ramasseur.
//
// Les événements antérieurs à l'origine du document sont écartés — un rejeu ne montre pas ce
// qui précède sa première frame — et le compte de ces écarts est publié.
func buildPickups(
	pickups []filmdec.BipedPickup, origin uint64, step uint64, slotXUID map[uint32]uint64,
	st filmdec.BipedPickupStats,
) ([]Pickup, PickupCoverage) {
	cov := PickupCoverage{
		Decoded:    len(pickups),
		MultiEvent: st.MultiEvent,
		Refused:    st.RefusedNoRef + st.RefusedNoCatalog + st.RefusedOffBand,
	}
	if len(pickups) == 0 || step == 0 {
		return nil, cov
	}
	out := make([]Pickup, 0, len(pickups))
	for _, p := range pickups {
		if p.TimestampUS < origin {
			cov.BeforeOrigin++
			continue
		}
		k := PickupItem
		if filmdec.BipedPickupIsWeaponClass(p.Class) {
			k = PickupWeapon
		}
		e := Pickup{
			T:     int((p.TimestampUS - origin) / step),
			Slot:  p.Slot,
			W:     fmt.Sprintf("%08x", p.CatalogID),
			Kind:  k,
			Class: int(p.Class),
		}
		if x, ok := slotXUID[p.Slot]; ok {
			e.XUID = strconv.FormatUint(x, 10)
			cov.Named++
		}
		out = append(out, e)
		cov.Published++
		if k == PickupWeapon {
			cov.Weapons++
		} else {
			cov.Items++
		}
	}
	if len(out) == 0 {
		return nil, cov
	}
	return out, cov
}

// PickupCoverage dit ce que le canal a vu, ce qu'il a écarté et ce qu'il ne PEUT PAS voir.
type PickupCoverage struct {
	// Decoded est le nombre de ramassages rendus par le décodeur.
	Decoded int `json:"decoded"`
	// Published est le nombre publié dans le document.
	Published int `json:"published"`
	// Named est le nombre dont le ramasseur porte un XUID.
	Named int `json:"named"`
	// Weapons / Items ventilent les publiés par nature.
	Weapons int `json:"weapons"`
	Items   int `json:"items"`
	// BeforeOrigin compte les ramassages antérieurs à la première frame — écartés.
	BeforeOrigin int `json:"beforeOrigin"`
	// MultiEvent compte les listes d'événements qui portent un AUTRE événement après le
	// ramassage. C'EST LA MESURE DE CE QUE LE CANAL NE VOIT PAS : le balayage ne décode que
	// l'événement de tête, donc un ramassage en deuxième position lui échappe. Un lecteur qui
	// veut juger le rappel doit lire ce nombre.
	MultiEvent int `json:"multiEvent"`
	// Refused compte les événements que le décodeur a REFUSÉ de publier (référence absente,
	// identifiant absent, slot hors bande de bipèdes). Jamais non nul sur le corpus de
	// référence : une valeur non nulle signale une largeur de runtime inadaptée au film.
	Refused int `json:"refused"`
}
