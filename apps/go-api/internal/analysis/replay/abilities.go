package replay

import (
	"sort"
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
)

// abilities.go — LA CAPACITÉ D'ARMURE PORTÉE, portée à la grille du rejeu.
//
// UNE SEULE GRANDEUR, DEUX SOURCES — et c'est tout le sujet de ce fichier. Le film dit deux
// fois quelle capacité un joueur porte, par deux chemins sans rapport l'un avec l'autre :
//
//	i48  `biped-desired-ability-set-component`, dans les paquets DELTA. Il transmet le RANG
//	     COMPLET dans la palette du match (R(6) après une porte). Rare — 0,03 à 0,09 % des
//	     records, à peu près une fois par vie — mais lu à 100 % : 748 lectures sur 8 films,
//	     zéro illisible. Cf. filmdec/ability_rank.go.
//	kf   le champ de 3 bits ancré dans les IMAGES-CLÉS (inventory_decode.go, règle R1).
//	     Dense (une lecture par joueur et par image-clé) mais BORGNE : son motif d'ancrage
//	     se termine par `010`, qui sont les bits de POIDS FORT du rang. Il ne voit donc que
//	     la fenêtre 16..23 de la palette, et rien d'autre.
//
// POURQUOI LES DEUX, ET POURQUOI DANS LE MÊME CHAMP. Jusqu'au 2026-08-14, le canal
// d'image-clé publiait `rang − 16` sous le nom d'« index de capacité » : une grandeur
// DIFFÉRENTE du rang, sous un nom qui ne le disait pas. C'est la cause, enfin nommée, de
// trois anomalies — les valeurs ne sortaient jamais de 3-7, 21 films sur 40 ne rendaient
// aucune lecture (aucun joueur n'y portait un rang 16-23), et les films « où les huit
// joueurs portent le même équipement » étaient un artefact. La correction n'est pas de
// choisir un canal : c'est de les ramener à LA MÊME grandeur — le rang — et de dire, sur
// chaque lecture, d'où elle vient. Deux canaux qui rendent des grandeurs différentes sous le
// même nom, c'est le défaut qui a coûté ce chantier ; deux canaux qui rendent la même
// grandeur en le disant, c'est une couverture.
//
// CE QUI N'EST PAS ICI, et qui est la demande d'écran non satisfaite : l'ÉTAT ACTIF. Savoir
// qu'un joueur PORTE un camouflage ne dit pas qu'il l'a DÉCLENCHÉ. La source d'état (`i57`)
// est lue sur 0,82 % des records et s'associe aux épisodes d'`i54` à 72,2 % contre 34 % de
// témoin — une erreur sur quatre. Tant qu'elle n'est pas fiable, aucun effet plein-fiche ne
// se code : le document publie l'IDENTITÉ, pas l'ÉTAT.

// Sources de lecture publiées sur AbilityRead.Src. Elles ne sont pas décoratives : la
// fenêtre de visibilité des deux canaux n'est pas la même, et un lecteur qui voudrait juger
// une couverture doit pouvoir les séparer.
const (
	// AbilitySrcI48 : paquet delta, rang complet sur toute la palette.
	AbilitySrcI48 = "i48"
	// AbilitySrcKeyframe : image-clé, fenêtre 16..23 UNIQUEMENT.
	AbilitySrcKeyframe = "kf"
)

// AbilityRead est UNE lecture de la capacité d'armure portée par un slot.
type AbilityRead struct {
	// T est l'index de frame, sur le même axe que Point.T.
	T int `json:"t"`
	// Slot désigne la Track concernée — donc une VIE, pas un joueur.
	Slot uint32 `json:"slot"`
	// R est le RANG dans la palette du match. Ce n'est PAS un index de capacité universel :
	// le même rang désigne des capacités différentes d'une palette à l'autre, et c'est
	// AbilityLabels — construite pour la palette de CE film — qui le nomme, ou pas.
	R int `json:"r"`
	// Src dit par quel canal la lecture est arrivée (AbilitySrcI48 / AbilitySrcKeyframe).
	// Publié parce que les deux canaux ne voient pas la même chose : une lecture `kf` est
	// forcément dans 16..23, une lecture `i48` peut valoir n'importe quel rang.
	Src string `json:"src"`
}

// buildAbilityReads projette les deux canaux sur la grille de frames du rejeu.
//
// Les lectures ANTÉRIEURES à l'origine du rejeu sont écartées : elles n'ont pas de place sur
// l'axe, et leur en inventer une les poserait sur la première image comme si elles y avaient
// été mesurées.
//
// AUCUNE FUSION, AUCUN ARBITRAGE : les deux canaux disent la même grandeur, on publie les
// deux lectures telles quelles. Départager deux mesures concordantes n'apporterait rien ;
// les départager quand elles divergent supposerait de savoir laquelle a tort, ce qu'on ne
// sait pas — et le client, lui, prend simplement la plus récente.
func buildAbilityReads(
	ranks []filmdec.AbilityRank, inv []KeyframeInventory, origin, step uint64,
) []AbilityRead {
	out := make([]AbilityRead, 0, len(ranks)+len(inv))
	for _, r := range ranks {
		if r.TimestampUS < origin {
			continue
		}
		out = append(out, AbilityRead{
			T: int((r.TimestampUS - origin) / step), Slot: r.Slot, R: r.Rank, Src: AbilitySrcI48,
		})
	}
	for _, r := range inv {
		if r.TimestampUS < origin || r.AbilityRank < 0 {
			continue
		}
		out = append(out, AbilityRead{
			T: int((r.TimestampUS - origin) / step), Slot: r.Slot, R: r.AbilityRank,
			Src: AbilitySrcKeyframe,
		})
	}
	if len(out) == 0 {
		return nil
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].T != out[j].T {
			return out[i].T < out[j].T
		}
		if out[i].Slot != out[j].Slot {
			return out[i].Slot < out[j].Slot
		}
		return out[i].Src < out[j].Src
	})
	return out
}

// keepAbilitiesOfPublishedTracks écarte les lectures dont le slot n'a pas de trajectoire
// publiée : le client n'aurait aucune fiche où les poser.
func keepAbilitiesOfPublishedTracks(reads []AbilityRead, tracks []Track) []AbilityRead {
	return keepOfPublishedTracks(reads, tracks,
		func(a AbilityRead, published map[uint32]bool) bool { return published[a.Slot] })
}

// abilityLabelsUsed nomme les rangs de capacité que le document emploie RÉELLEMENT.
//
// Un rang hors table n'entre pas : il gardera son numéro à l'écran, marqué comme non
// interprétable. La table est partielle ET propre à une palette — le dire vaut mieux que
// combler.
func abilityLabelsUsed(reads []AbilityRead, catalog map[int]Label) map[string]Label {
	out := map[string]Label{}
	for _, r := range reads {
		if name, ok := catalog[r.R]; ok {
			out[strconv.Itoa(r.R)] = name
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
