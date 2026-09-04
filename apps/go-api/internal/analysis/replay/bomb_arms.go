package replay

// bomb_arms.go — QUI A ARMÉ LA BOMBE : la jointure entre un armement DATÉ et un porteur NOMMÉ.
//
// # POURQUOI UNE JOINTURE, ET PAS UNE LECTURE
//
// L'armement est daté par l'anneau `ti=12 i14` (bomb_armings.go), qui est un MARQUEUR D'ÉCRAN :
// il donne un instant, jamais un acteur. Et ce n'est pas un manque de notre décodage — c'est le
// moteur qui ne porte pas l'acteur : le Lua `primitive_carriable_arming_base` (tag `25af9c45`)
// émet six événements d'armement dont le seul porteur d'identité est l'ÉQUIPE
// (`activatingTeam`, via `Item_GetInventoryUnit` -> `Player_GetMultiplayerTeam`) ; il n'existe
// aucun `activatingPlayer` (mesure Ghidra du 2026-09-04). Nommer un armeur ne peut donc se
// faire QUE par recoupement avec un autre canal — celui des armes tenues.
//
// # LA RÈGLE DE JOINTURE, ÉCRITE AVANT LA MESURE
//
// L'armement est une INTERACTION TENUE au site (`Device_GetInteractionHoldTime`) : le porteur
// tient la bombe, la montée de l'anneau court pendant le hold, et à la fin de la montée la
// bombe QUITTE SES MAINS — elle est posée. Le canal des armes tenues voit exactement ce
// départ : une transition DEPUIS la famille bombe, c'est-à-dire la FERMETURE PAR LÂCHER d'une
// période de portage. Le lâcher EST le geste de pose.
//
//	L'ARMEUR d'un armement daté à t_arm est le porteur dont une période de portage :
//	  1. a commencé AVANT t_arm            — on n'arme pas une bombe qu'on n'a pas encore ;
//	  2. se ferme par un LÂCHER            — ni par la mort du porteur, ni par la fin du film ;
//	  3. se ferme dans ±2 500 ms de t_arm  — la fenêtre, cf. ci-dessous ;
//	  4. est la plus PROCHE de t_arm parmi les candidates, et n'a pas déjà servi à un autre
//	     armement (une période de portage ne pose qu'une bombe).
//
// LE SENS DU GESTE ÉTAIT DÉDUIT, IL EST MAINTENANT MESURÉ. B2 (2026-09-01) le déduisait de
// deux médianes : lâcher -> explosion 4 804 ms contre une mèche de 4 930 ms, soit un lâcher
// ~126 ms APRÈS l'instant armé. La mesure DIRECTE de la jointure (gate du 2026-09-04, quatre
// films, 10 appariements) donne l'écart lâcher − armement :
//
//	35b75a31  +247, +247        3d58eb37  +256, +257, +259
//	1c01e34f  +252, +253, +254  69b16f5d  +253, +258
//
// — TOUS positifs, étendue de 12 ms sur 10 observations. L'anneau se remplit, PUIS la bombe
// quitte les mains ~255 ms plus tard. La fenêtre reste néanmoins SYMÉTRIQUE : dix observations
// sur quatre films ne justifient pas de coder un signe dans la règle.
//
// LA FENÊTRE (`bombArmDropWindowMS` = 2 500 ms) est bornée par trois grandeurs mesurées, pas
// choisie : elle couvre ~10 fois l'écart observé de ~255 ms, absorbe très largement le résidu
// d'horloge du recalage ci-dessous (33-114 ms mesurés sur les cinq films du gate, plafonnés à
// 1 000 ms par `originControlToleranceMS`), et reste sous la MOITIÉ de la séparation minimale
// de deux armements réels (hold le plus court mesuré ~0,9 s + mèche 4,93 s, soit >= ~6 s) —
// deux fenêtres consécutives ne peuvent donc pas se disputer un même lâcher.
//
// CE QUE LA RÈGLE NE COUVRE PAS, ET QUI EST MESURÉ AUSSI : 3 armements sur 13 n'ont AUCUN
// lâcher dans la fenêtre. Sur `35b75a31` à 299 176 ms, la période du porteur COUVRE l'armement
// et ne se ferme que 4 245 ms APRÈS l'explosion : sur cette pose-là le canal n'émet pas de
// lâcher. Ces armements sont publiés SANS ACTEUR (`ArmingsNoDrop`), jamais attribués au
// porteur actif « faute de mieux ».
//
// # LE RECALAGE D'HORLOGE : DEUX SOURCES, DEUX ZÉROS
//
// C'est le piège de ce lot, et il ne se voit dans AUCUN test pur qui l'oublierait.
//
//	BombArming.TimeMS       horloge du FILM — l'anneau est daté par le MANIFESTE
//	                        (`chunkStartMS`, cf. filmdec/navpoint_radial_scan.go) ;
//	HeldObjectPeriod.*MS    horloge du MATCH — `matchMS = TimestampUS/1000 − deathOffsetMS`
//	                        (cf. l'en-tête de bomb_carries.go).
//
// Le pont est une SOUSTRACTION DE DEUX LECTURES DU MÊME FILM, déjà écrite en production dans
// `resolveOriginMs` (origin.go) sous le nom d'`ecart` :
//
//	horlogeMatch(ms) = horlogeFilm(ms) + premierPaquetDuFilmUS/1000 − deathOffsetMS
//
// — parce que la frame 0 se date des deux façons, `(firstPosUS − filmClockUS)/1000` sur
// l'horloge du manifeste et `firstPosUS/1000 − deathOffsetMS` sur celle du match. Leur
// différence est exactement ce décalage. Elle est MESURÉE à 16-81 ms sur les quatre films
// témoins d'origin.go, et à 33-114 ms sur les cinq films du gate de ce lot (35b75a31 : 114 ;
// 9f57c612 : 88 ; 1c01e34f : 50 ; 3d58eb37 : 33 ; 69b16f5d : 62) — même ordre de grandeur, et
// au-delà de 1 000 ms la production ne publie plus d'origine du tout.
//
// Ce noyau ne lit aucun film : le décalage lui est FOURNI (`BombStatsInput.FilmToMatchOffsetMS`)
// et l'appelant le calcule des deux entrées ci-dessus. `TestBombArmsRecalageHorloge` échoue si
// le terme saute.
//
// # CE QUI N'EST PAS DEVINÉ
//
// Un armement dont aucune période ne tient les quatre conditions, ou dont la période retenue
// est au slot NON PONTÉ, est PUBLIÉ SANS ACTEUR : l'instant est vrai, le nom manque, et
// inventer le nom serait la seule vraie faute. Les deux raisons se comptent séparément
// (`ArmingsNoDrop`, `ArmingsNoBridge`) et le contrôle de cohérence publié tient par
// construction : somme des `bomb_arms` == `ArmingsAttributed` <= `Armings`.

import (
	"sort"
	"strconv"
)

// BombEventArmed est la valeur `event_type` d'un armement de bombe daté, sous l'objectif
// `objectiveevents.ObjectiveTypeBomb` — le pendant de BombEventDetonated.
const BombEventArmed = "bomb_armed"

// bombArmDropWindowMS est la demi-fenêtre de la jointure : écart maximal accepté entre la
// fermeture par lâcher d'une période de portage et l'instant armé, une fois les deux sur
// l'horloge du MATCH. Cf. l'en-tête pour les trois grandeurs qui la bornent.
const bombArmDropWindowMS = 2500

// bombDrop est une période de portage FERMÉE PAR UN LÂCHER, réduite à ce que la jointure lit.
type bombDrop struct {
	// XUID du porteur, 0 quand le pont n'a pas nommé le slot.
	XUID uint64
	// DebutMS / FinMS sur l'horloge du MATCH ; FinMS est l'instant du lâcher.
	DebutMS, FinMS int
}

// bombArmsByXUID attribue chaque armement daté à un poseur quand la jointure le permet, et
// rend les faits datés correspondants — sur l'horloge du FILM, comme les explosions.
//
// L'ordre de traitement est chronologique et la consommation des périodes est exclusive : deux
// armements ne peuvent pas se réclamer du même lâcher.
func bombArmsByXUID(in BombStatsInput, cov *BombStatsCoverage) (map[string]int, []BombEvent) {
	if !in.ArmingsRead {
		return nil, nil
	}
	cov.Armings = len(in.Armings)
	armings := append([]BombArming(nil), in.Armings...)
	sort.SliceStable(armings, func(i, j int) bool { return armings[i].TimeMS < armings[j].TimeMS })
	drops := bombDropPeriods(in)
	used := make([]bool, len(drops))
	counts := map[string]int{}
	events := make([]BombEvent, 0, len(armings))
	for _, a := range armings {
		ev := BombEvent{Type: BombEventArmed, TimeMS: a.TimeMS}
		i := bombPoseurOf(drops, used, a.TimeMS+in.FilmToMatchOffsetMS)
		switch {
		case i < 0:
			cov.ArmingsNoDrop++
		case drops[i].XUID == 0:
			used[i] = true
			cov.ArmingsNoBridge++
		default:
			used[i] = true
			ev.XUID = strconv.FormatUint(drops[i].XUID, 10)
			counts[ev.XUID]++
			cov.ArmingsAttributed++
		}
		events = append(events, ev)
	}
	return counts, events
}

// bombDropPeriods retient les seules périodes qui portent un GESTE DE POSE : fermées par un
// lâcher. Une période fermée par la MORT du porteur n'est pas une pose (le canal n'émet rien à
// la mort, c'est le fil des morts qui la ferme) ; une période restée OUVERTE à la fin du film
// n'a pas de lâcher du tout. Triées par instant de lâcher, l'ordre de la jointure.
func bombDropPeriods(in BombStatsInput) []bombDrop {
	if !in.CarryRead {
		return nil
	}
	out := make([]bombDrop, 0, len(in.Carry.Periods))
	for _, p := range in.Carry.Periods {
		if p.Ouverte || p.FinParMort {
			continue
		}
		out = append(out, bombDrop{XUID: p.XUID, DebutMS: p.DebutMS, FinMS: p.FinMS})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].FinMS != out[j].FinMS {
			return out[i].FinMS < out[j].FinMS
		}
		return out[i].XUID < out[j].XUID
	})
	return out
}

// bombPoseurOf rend l'index de la période candidate la plus proche de l'instant armé (horloge
// du MATCH), ou -1 quand aucune ne tient les conditions 1 à 4 de l'en-tête. À écart égal, la
// première dans l'ordre trié gagne — un ordre total, donc une sortie reproductible.
func bombPoseurOf(drops []bombDrop, used []bool, armMatchMS int) int {
	best, bestEcart := -1, 0
	for i, d := range drops {
		if used[i] || d.DebutMS > armMatchMS {
			continue
		}
		ecart := d.FinMS - armMatchMS
		if ecart < 0 {
			ecart = -ecart
		}
		if ecart > bombArmDropWindowMS {
			continue
		}
		if best < 0 || ecart < bestEcart {
			best, bestEcart = i, ecart
		}
	}
	return best
}
