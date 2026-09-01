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
	// PickupGrenade : une GRENADE (classe 2). Établi le 2026-09-01 par les NOMS et non par
	// corrélation : une fois les identifiants résolus dans le manifeste d'objets d'équipement
	// du titre (chaîne `sofd -> sofa -> {string_id, eqip}` des fichiers du jeu), la classe 2
	// est grenade dans 100,0 % de ses événements et la classe 3 dans 0,0 %, sur les deux films
	// de référence, sans un seul identifiant réparti sur les deux classes.
	PickupGrenade PickupKind = "grenade"
	// PickupEquipment : un objet d'ÉQUIPEMENT (classe 3) — mur, capteur, grappin, propulseur.
	// Même mesure que ci-dessus, prise par l'autre bout : 0,0 % de grenade.
	PickupEquipment PickupKind = "equipment"
	// PickupItem : autre chose qu'une arme, sans plus de précision.
	//
	// CETTE VALEUR N'EST PAS MORTE, ET ELLE N'A PAS ÉTÉ RENOMMÉE. Jusqu'au schéma 30 elle
	// couvrait TOUTES les classes non-arme ; depuis le schéma 31 les classes 2 et 3 ont leur
	// valeur propre et `item` reste le REPLI des classes non-arme que rien n'établit. Le R(3)
	// porte huit valeurs possibles et seules quatre sont observées sur le corpus : une
	// cinquième ne doit pas se publier comme une grenade parce qu'elle n'est pas une arme.
	PickupItem PickupKind = "item"
)

// pickupKindOfClass rend la nature d'un ramassage à partir de son R(3).
//
// LA TABLE EST ICI ET NULLE PART AILLEURS : le décodeur (`filmdec`) est title-agnostic et ne
// sait rien de ce qu'une classe désigne ; le document est la couche qui l'interprète.
func pickupKindOfClass(c uint8) PickupKind {
	switch {
	case filmdec.BipedPickupIsWeaponClass(c):
		return PickupWeapon
	case c == 2:
		return PickupGrenade
	case c == 3:
		return PickupEquipment
	default:
		return PickupItem
	}
}

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
	// Family est le SLUG de l'objet ramassé, résolu au moment du build par les catalogues du
	// TITRE (schéma 31). Deux sources, selon la nature :
	//
	//	arme (classes 0/1)          `LabelCatalog.Keys` : famille -> weapon_key, ex. `hinf_ma40_ar`
	//	non-arme (classes 2/3)      `LabelCatalog.EquipmentFamilies` : GlobalID `eqip` -> famille
	//	                            du manifeste, ex. `grenade_frag`, `thruster`, `wall`
	//
	// C'EST UN SLUG, JAMAIS UN LIBELLÉ. Aucune chaîne FR/EN ne descend du Go (règle
	// multi-titre) : le client joint ce slug à ses propres tables. `weaponLabels` nomme déjà
	// les weapon_key ; les familles d'équipement attendent leur table côté web.
	//
	// ABSENT quand aucun catalogue ne connaît l'identifiant — et c'est publié comme tel plutôt
	// que rempli d'un `unknown` qui se lirait comme un nom. Le manifeste du titre ne déclare
	// que 21 objets d'équipement, ceux du corpus qui l'a bâti ; les trous doivent SE VOIR, et
	// `coverage.pickups.unknownFamilies` les compte.
	Family string `json:"family,omitempty"`
	// Kind dit si c'est une arme, une grenade, un équipement, ou autre chose. C'est le champ
	// sur lequel un client branche.
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
// LA RÉSOLUTION DU NOM SE FAIT SUR L'ENTIER, PAS SUR LA CHAÎNE — et c'est délibéré.
//
// Le chantier précédent a payé un P0 pour avoir comparé deux ÉCRITURES d'une même famille
// (`%08x` nu contre `"0x"` + majuscules) : la jointure était morte et publiait un zéro qui se
// lisait comme une mesure. La leçon retenue ici va plus loin que « normaliser au point de
// jointure » : on ne fabrique pas la chaîne du tout. Les deux catalogues du titre
// (`LabelCatalog.Keys` et `LabelCatalog.EquipmentFamilies`) sont keyés par `uint32`, et
// `BipedPickup.CatalogID` EST cet `uint32`. La classe entière de bogues n'existe donc pas sur
// ce chemin, et aucun helper de normalisation n'est nécessaire.
//
// (Vérifié sur pièces le 2026-09-01 : les 21 entrées de `[[equipment_objects]]` s'écrivent
// `"0x"` + minuscules, et `tagGlobalID32` les parse en `uint32` au chargement du manifeste.
// La casse du fichier n'atteint jamais cette jointure.)
func buildPickups(
	pickups []filmdec.BipedPickup, clk replayClock, slotXUID map[uint32]uint64,
	st filmdec.BipedPickupStats, weaponKeys map[uint32]string,
) ([]Pickup, PickupCoverage) {
	cov := PickupCoverage{
		Decoded:    len(pickups),
		MultiEvent: st.MultiEvent,
		Refused:    st.RefusedNoRef + st.RefusedNoCatalog + st.RefusedOffBand,
	}
	if len(pickups) == 0 || clk.step == 0 {
		return nil, cov
	}
	out := make([]Pickup, 0, len(pickups))
	for _, p := range pickups {
		if p.TimestampUS < clk.origin {
			cov.BeforeOrigin++
			continue
		}
		k := pickupKindOfClass(p.Class)
		e := Pickup{
			T:      int((p.TimestampUS - clk.origin) / clk.step),
			Slot:   p.Slot,
			W:      fmt.Sprintf("%08x", p.CatalogID),
			Family: pickupFamily(p.CatalogID, k, clk.families, weaponKeys),
			Kind:   k,
			Class:  int(p.Class),
		}
		if x, ok := slotXUID[p.Slot]; ok {
			e.XUID = strconv.FormatUint(x, 10)
			cov.Named++
		}
		if e.Family == "" {
			cov.UnknownFamilies++
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

// pickupFamily résout le slug d'un objet ramassé dans le catalogue qui LE concerne.
//
// LE CATALOGUE EST CHOISI PAR LA NATURE, jamais essayé « au cas où ». Chercher un identifiant
// d'équipement dans la table des armes, ou l'inverse, ne rendrait rien de bon même en cas de
// succès : les deux espaces sont DISJOINTS (mesuré le 2026-09-01, 0 identifiant commun dans
// les deux sens sur les deux films de référence), donc une résolution croisée serait la preuve
// d'une erreur, pas un repli. Un `item` de repli n'interroge aucun des deux : sa nature n'est
// pas établie, donc aucun catalogue ne le concerne.
func pickupFamily(id uint32, k PickupKind, equipment, weapons map[uint32]string) string {
	switch k {
	case PickupWeapon:
		return weapons[id]
	case PickupGrenade, PickupEquipment:
		return equipment[id]
	default:
		return ""
	}
}

// PickupCoverage dit ce que le canal a vu, ce qu'il a écarté et ce qu'il ne PEUT PAS voir.
type PickupCoverage struct {
	// Decoded est le nombre de ramassages rendus par le décodeur.
	Decoded int `json:"decoded"`
	// Published est le nombre publié dans le document.
	Published int `json:"published"`
	// Named est le nombre dont le ramasseur porte un XUID.
	Named int `json:"named"`
	// Weapons / Items ventilent les publiés par nature. `Items` compte TOUS les non-armes —
	// grenades comprises — et garde donc exactement le sens qu'il avait au schéma 30, même
	// depuis que `Kind` les distingue. Le détail par nature se lit sur les événements.
	Weapons int `json:"weapons"`
	Items   int `json:"items"`
	// UnknownFamilies compte les ramassages publiés SANS `family` : aucun catalogue du titre ne
	// connaît leur identifiant.
	//
	// C'EST UN DÉNOMINATEUR, PAS UNE ANOMALIE. Le manifeste `[[equipment_objects]]` ne déclare
	// que les 21 objets du corpus qui l'a bâti, et le catalogue d'armes ne couvre pas tout le
	// jeu. Sans ce compteur, un artefact où rien ne se résout se lirait comme un artefact où
	// tout va bien : `family` étant `omitempty`, son absence est invisible à la lecture.
	// Mesuré à 0 sur les deux films de référence (118 non-armes, 100 % résolus).
	UnknownFamilies int `json:"unknownFamilies"`
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
