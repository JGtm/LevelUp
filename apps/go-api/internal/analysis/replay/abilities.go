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

// AbilityPalette est une palette de capacités du titre : les rangs qui la SIGNENT, et les
// noms qu'elle donne à ceux d'entre eux qui sont établis. Elle vient du catalogue du titre
// (`config/titles/{slug}/mappings/replay_labels.toml`), jamais du code.
type AbilityPalette struct {
	// ID nomme la palette dans les journaux. Il ne sort jamais à l'écran.
	ID string
	// Markers sont les rangs dont l'observation signe cette palette.
	Markers []int
	// Ranks nomme les rangs établis. Partielle par nature.
	Ranks map[int]Label
}

// LE CLASSEMENT DE PALETTE — la règle, et les chiffres qui la fondent.
//
// POURQUOI UNE RÈGLE STATISTIQUE ET NON UNE LECTURE. On a d'abord cherché une DÉSIGNATION
// dans le film, et elle n'y est pas : le registre du chunk_00 est bit-à-bit identique d'un
// film à l'autre pour les noms et flags de composants, sa longueur ne suit pas les familles
// (1 973 120 octets aussi bien sur `00162144`, famille A, que sur les trois films de la
// famille B), et aucun marqueur de groupe de tags (`sofd`, `eqip`, `vcdd`, `uwfa`, `glpa`)
// n'apparaît dans aucun chunk. La palette se déduit donc de ce que le film MONTRE.
//
// CE QUE LA MESURE DONNE (7 films, i48, 748 lectures, zéro illisible) :
//
//	00ba2e1c   1:31 2:25 4:34 5:20 6:36 10:21 23:35                       -> 202/202 famille A
//	06dfe6d9   1:19 2:25 4:35 5:22 6:38 8:2 9:2 10:34 11:3 12:4 23:35     -> 219/219 famille A
//	00162144   2:14 4:9 9:2 10:10                                          ->   35/35  famille A
//	084a804d   1:13 4:48 5:11 6:18 8:10 9:8 10:2 19:4 23:15 44:1           ->  125/130 famille A
//	000d5950   19:18 20:22 21:26 22:16                                     ->   82/82  famille B
//	00502e52   19:22 20:17 21:8 22:18                                      ->   65/65  famille B
//	07aa428d   19:11 20:10 21:13 22:8                                      ->   42/42  famille B
//
// SIX FILMS SUR SEPT SONT PURS À 100 %, le septième à 96,2 %. La règle est donc une règle de
// MAJORITÉ, et le seuil n'est pas un réglage sensible : n'importe quelle valeur entre 50 % et
// 96 % donne le MÊME classement sur ce corpus. On prend 90 %.
//
// LES QUATRE LECTURES `19` ET L'UNIQUE `44` DE `084a804d` sont le bruit attendu d'un balayage
// bit à bit : le motif d'en-tête de record peut coïncider avec autre chose, et 44 est même
// hors de toute palette connue (un `sofd` compte ~27 entrées). Les compter contre la pureté
// plutôt que les ignorer est délibéré — c'est ce qui fait que la règle REFUSERAIT un film
// réellement mélangé.
const (
	// abilityPalettePurity : part minimale des lectures portant les marqueurs d'UNE palette.
	abilityPalettePurity = 0.90
	// abilityPaletteMinReads : en deçà, on ne classe pas. Le chiffre est DÉRIVÉ du seuil et
	// non choisi à part — c'est le plus petit n tel qu'UNE lecture parasite ne suffise pas à
	// disqualifier un film pur : (n−1)/n ≥ 0,90, donc n ≥ 10. En deçà, le test de pureté ne
	// mesurerait plus la palette mais le hasard du balayage. Le corpus est loin au-dessus :
	// le film le plus pauvre en transmet 35.
	abilityPaletteMinReads = 10
)

// classifyAbilityPalette choisit la palette du film d'après ce qu'il MONTRE, ou rend nil.
//
// UNE SIGNATURE AMBIGUË NE NOMME RIEN, et c'est le point de toute l'étape : un film dont les
// lectures se partagent entre deux palettes, ou qui en montre trop peu, sort AVEC SES RANGS
// et sans un seul nom. Nommer au jugé mettrait « grappin » sur un propulseur.
func classifyAbilityPalette(reads []AbilityRead, palettes []AbilityPalette) *AbilityPalette {
	if len(reads) < abilityPaletteMinReads || len(palettes) == 0 {
		return nil
	}
	counts := make([]int, len(palettes))
	for _, r := range reads {
		for i, p := range palettes {
			if containsRank(p.Markers, r.R) {
				counts[i]++
				break
			}
		}
	}
	for i := range palettes {
		if float64(counts[i])/float64(len(reads)) >= abilityPalettePurity {
			return &palettes[i]
		}
	}
	return nil
}

// paletteIDOrNone rend l'identifiant de la palette retenue, ou une mention explicite quand
// AUCUNE ne l'a ete — un journal muet se lirait comme « pas de capacite lue ».
func paletteIDOrNone(p *AbilityPalette) string {
	if p == nil {
		return "non classee"
	}
	return p.ID
}

func containsRank(marks []int, r int) bool {
	for _, m := range marks {
		if m == r {
			return true
		}
	}
	return false
}

// abilityLabelsUsed nomme les rangs de capacité que le document emploie RÉELLEMENT.
//
// Un rang hors table n'entre pas : il gardera son numéro à l'écran, marqué comme non
// interprétable. La table est partielle ET propre à une palette — le dire vaut mieux que
// combler.
func abilityLabelsUsed(reads []AbilityRead, palette *AbilityPalette) map[string]Label {
	if palette == nil {
		return nil
	}
	out := map[string]Label{}
	for _, r := range reads {
		if name, ok := palette.Ranks[r.R]; ok {
			out[strconv.Itoa(r.R)] = name
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
