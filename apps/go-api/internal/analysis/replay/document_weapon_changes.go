package replay

// document_weapon_changes.go — LES PRISES ET LES LÂCHERS D'ARME, datés à la milliseconde.
//
// CE QUE C'EST. Le composant `weapon-state-type-info` du bipède n'entre au masque du flux delta
// que lorsque l'identité d'un emplacement d'arme CHANGE. Chaque émission est donc une prise, un
// lâcher ou un échange, à l'instant du paquet — pas dans un intervalle de vingt secondes.
//
// CE QUE ÇA CORRIGE. Jusqu'ici le document ne portait que `padPickups` : « ce socle s'est vidé
// quelque part dans cet intervalle », sans le joueur (`PadPickup.XUID` vaut `null` partout).
// Le présent calque donne QUI, QUAND et QUELLE ARME.
//
// CE QUE ÇA NE DONNE PAS, ET IL FAUT LE LIRE AVANT D'Y TOUCHER :
//
//   - le SOCLE d'origine d'une prise. Trois hypothèses de lien vers l'objet du monde ont été
//     mesurées et les trois sont réfutées : la suppression de l'entité (1/71 sur six films,
//     SOUS son propre témoin), son attachement au porteur (1/21), et l'appariement par les
//     armes (5 à 12 % de gagnants nets contre 70 % exigés, avec 20 à 28 % de contradictions
//     physiques). Le lâcher, LUI, fait naître une entité arme-au-sol dans le MÊME paquet.
//   - la FIN DE VIE réelle de l'arme lâchée. Voir `WeaponChange.Until` : c'est une convention,
//     pas une mesure.
//   - la COMPLÉTUDE. Le canal est JUSTE — sur 5 627 tirs de trois films, il ne retire jamais
//     une arme encore utilisée — mais rien ne prouve qu'il voit TOUTES les prises. Les oracles
//     hors ligne disponibles sont soit trop grossiers (images-clés, 20 s), soit saturés (l'union
//     des inventaires plafonne à 98-100 % avant même d'appliquer le canal).
//
// PLAUSIBILITÉ MESURÉE, puisque la complétude ne l'est pas. Sur deux CTF Arena, une fois les
// DRAPEAUX écartés (ils occupent un emplacement d'arme et dominent le volume), il reste 22 et 21
// ramassages d'arme par match, composés de Gravity Hammer, S7 Sniper, M41 SPNKr, Pulse Carbine,
// BR75 — des armes de socle et de râtelier, jamais des armes de départ. Les journaux du même
// assemblage comptent 10 et 13 socles sur ces cartes : environ deux prises par socle et par
// match, cohérent avec un temps de recharge.

import (
	"fmt"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/weaponv3"
)

// WeaponChangeKind qualifie un changement d'arme en main, tel que le document le publie.
type WeaponChangeKind string

const (
	// WeaponTaken : l'emplacement était vide, ou l'arme était absente du loadout de spawn.
	WeaponTaken WeaponChangeKind = "taken"
	// WeaponDropped : l'emplacement passe à vide. C'est le cas NON AMBIGU, et c'est celui qui
	// porte l'affichage de l'arme au sol.
	WeaponDropped WeaponChangeKind = "dropped"
	// WeaponSwapped : l'emplacement passe d'une arme à une autre.
	WeaponSwapped WeaponChangeKind = "swapped"
)

// WeaponChange est UN changement d'arme en main.
//
// LES « DÉJÀ PORTÉES » NE SONT PAS PUBLIÉES, et c'est la décision la plus importante de ce
// fichier. Le flux ré-annonce parfois une arme que le joueur avait déjà à sa réapparition
// (changement d'emplacement, pas ramassage) : 10 cas sur 229 et 10 sur 100 dans les mesures.
// Les publier gonflerait le compte des prises d'environ 8 % avec des événements qui n'ont pas
// eu lieu.
type WeaponChange struct {
	// T est l'index de frame, sur le même axe que Point.T.
	T int `json:"t"`
	// Slot est le slot du bipède : il désigne la Track concernée, donc une VIE.
	Slot uint32 `json:"slot"`
	// Kind qualifie le changement.
	Kind WeaponChangeKind `json:"kind"`
	// W est l'identifiant de FAMILLE d'arme (high-32) en hexadécimal 8 chiffres, même
	// convention que `Loadout.W`. Vide sur un lâcher : l'emplacement n'a plus d'arme.
	W string `json:"w,omitempty"`
	// From est la famille précédente, quand elle est connue. Vide sinon.
	From string `json:"from,omitempty"`
	// Until, sur un LÂCHER SEULEMENT, est la frame jusqu'à laquelle le client peut montrer
	// l'arme au sol.
	//
	// C'EST UNE CONVENTION, PAS UNE MESURE, et le champ ne doit jamais être lu autrement.
	// Le jeu n'applique aucune minuterie inconditionnelle : le compte à rebours ne démarre
	// que lorsqu'un joueur s'éloigne sans regarder l'arme, il se gèle s'il revient, et un
	// joueur qui reste à proximité empêche la disparition INDÉFINIMENT
	// (cf. .ai/V7.5/reference/DESPAWN_ARMES_HALO_INFINITE.md). Mesure à l'appui : seules 5 à
	// 14 % des armes au sol reçoivent un événement de disparition dans le film, et les durées
	// observées n'ont aucune cohérence d'un match à l'autre — ce n'est pas un défaut de
	// lecture, c'est le comportement du jeu.
	//
	// `Until` applique donc la durée PUBLIÉE par arme (10, 20 ou 30 s) comme une borne
	// d'affichage raisonnable. Elle sera parfois trop courte — l'arme était encore là. Elle
	// n'invente rien qu'on aurait pu mesurer : la mesure n'existe pas.
	Until int `json:"until,omitempty"`
}

// weaponDespawnSeconds donne la durée de despawn PUBLIÉE par le jeu, par nom canonique d'arme.
//
// Source : guide communautaire relevé le 2026-08-30
// (.ai/V7.5/reference/DESPAWN_ARMES_HALO_INFINITE.md). NON OFFICIELLE — repère d'affichage,
// jamais vérité-terrain pour valider un décodage. Les variantes cosmétiques partagent la durée
// de leur canon, et c'est cohérent avec notre clé : on publie la FAMILLE, pas la variante.
var weaponDespawnSeconds = map[string]int{
	// 30 s — les armes de puissance.
	"S7 Sniper": 30, "Skewer": 30, "M41 SPNKr": 30, "Cindershot": 30,
	"Gravity Hammer": 30, "Energy Sword": 30, "Ravager": 30, "Needler": 30,
	// 20 s.
	"Heatwave": 20, "MLRS-2 Hydra": 20,
	// 10 s — les armes courantes.
	"Shock Rifle": 10, "Stalker Rifle": 10, "Mangler": 10, "CQS48 Bulldog": 10,
	"Sentinel Beam": 10, "BR75": 10, "VK78 Commando": 10, "MA40 AR": 10,
	"MA5K Avenger": 10, "Mk51 Sidekick": 10, "Disruptor": 10, "Pulse Carbine": 10,
	"Plasma Pistol": 10,
}

// weaponDespawnDefaultSeconds est la durée retenue pour une famille hors table : la plus
// COURTE des durées publiées. Une arme inconnue s'efface donc tôt plutôt que de rester
// affichée sur une carte où elle n'est peut-être plus.
const weaponDespawnDefaultSeconds = 10

// despawnSecondsFor rend la durée d'affichage d'une arme lâchée, par sa famille.
func despawnSecondsFor(family uint32) int {
	if name, ok := weaponv3.KnownWeaponHigh32[family]; ok {
		if s, ok := weaponDespawnSeconds[name]; ok {
			return s
		}
	}
	return weaponDespawnDefaultSeconds
}

// buildWeaponChanges projette les changements lus dans le film sur l'axe de frames du document.
//
// Les « déjà portées » sont ÉCARTÉES (cf. le contrat de WeaponChange) et les événements
// antérieurs à l'origine du document le sont aussi — un rejeu ne montre pas ce qui précède sa
// première frame.
func buildWeaponChanges(
	changes []filmdec.HeldWeaponChange, origin uint64, step uint64, endFrame int,
) ([]WeaponChange, WeaponChangeCoverage) {
	var cov WeaponChangeCoverage
	cov.Decoded = len(changes)
	if len(changes) == 0 || step == 0 {
		return nil, cov
	}
	out := make([]WeaponChange, 0, len(changes))
	for _, c := range changes {
		if c.Kind == filmdec.HeldWeaponRestated {
			cov.Restated++
			continue
		}
		if c.TimestampUS < origin {
			cov.BeforeOrigin++
			continue
		}
		frame := int((c.TimestampUS - origin) / step)
		w := WeaponChange{T: frame, Slot: c.Slot, Kind: weaponChangeKindOf(c.Kind)}
		if c.Family != filmdec.NoWeaponVariant {
			w.W = fmt.Sprintf("%08x", c.Family)
		}
		if c.Previous != filmdec.NoWeaponVariant {
			w.From = fmt.Sprintf("%08x", c.Previous)
		}
		if w.Kind == WeaponDropped && c.Previous != filmdec.NoWeaponVariant {
			until := frame + int(uint64(despawnSecondsFor(c.Previous))*1_000_000/step)
			if endFrame > 0 && until > endFrame {
				until = endFrame
			}
			w.Until = until
		}
		out = append(out, w)
		cov.Published++
		switch w.Kind {
		case WeaponTaken:
			cov.Taken++
		case WeaponDropped:
			cov.Dropped++
		case WeaponSwapped:
			cov.Swapped++
		}
	}
	if len(out) == 0 {
		return nil, cov
	}
	return out, cov
}

// weaponChangeKindOf traduit la nature lue par le décodeur en nature publiée.
func weaponChangeKindOf(k filmdec.HeldWeaponChangeKind) WeaponChangeKind {
	switch k {
	case filmdec.HeldWeaponDropped:
		return WeaponDropped
	case filmdec.HeldWeaponSwapped:
		return WeaponSwapped
	default:
		return WeaponTaken
	}
}

// WeaponChangeCoverage dit ce que le calque a vu et ce qu'il a écarté, pour qu'un lecteur
// puisse juger sans relire le film.
type WeaponChangeCoverage struct {
	// Decoded est le nombre de changements rendus par le décodeur.
	Decoded int `json:"decoded"`
	// Published est le nombre publié dans le document.
	Published int `json:"published"`
	// Restated compte les ré-annonces d'une arme déjà portée au spawn — écartées.
	Restated int `json:"restated"`
	// BeforeOrigin compte les changements antérieurs à la première frame — écartés.
	BeforeOrigin int `json:"beforeOrigin"`
	// Taken / Dropped / Swapped ventilent les publiés.
	Taken   int `json:"taken"`
	Dropped int `json:"dropped"`
	Swapped int `json:"swapped"`
}

// spawnSetFrom construit le prédicat « cette famille était-elle portée au dernier relevé
// d'image-clé qui précède ? », à partir des loadouts déjà balayés par l'assemblage.
//
// IL SERT À DISTINGUER UNE PRISE D'UNE RÉ-ANNONCE, et c'est sa seule raison d'être : la
// PREMIÈRE émission d'un emplacement n'a pas d'état précédent dans le flux, son état de départ
// vient du spawn. Sans ce prédicat, chaque première émission serait comptée comme une prise.
func spawnSetFrom(loadouts []filmdec.KeyframeLoadout) func(uint32, uint64) (map[uint32]bool, bool) {
	if len(loadouts) == 0 {
		return nil
	}
	bySlot := map[uint32][]filmdec.KeyframeLoadout{}
	for _, l := range loadouts {
		bySlot[l.Slot] = append(bySlot[l.Slot], l)
	}
	return func(slot uint32, at uint64) (map[uint32]bool, bool) {
		list := bySlot[slot]
		if len(list) == 0 {
			return nil, false
		}
		pick := list[0]
		for _, l := range list {
			if l.TimestampUS <= at {
				pick = l
			}
		}
		set := make(map[uint32]bool, len(pick.Families))
		for _, f := range pick.Families {
			set[f] = true
		}
		return set, true
	}
}
