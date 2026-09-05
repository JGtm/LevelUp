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
// # DEUX RÈGLES, DANS CET ORDRE, ET JAMAIS L'INVERSE
//
// La RÈGLE PRIMAIRE lit le GESTE DE POSE ; le REPLI ne lit que la PRÉSENCE. Ce ne sont pas deux
// façons interchangeables de nommer le même acteur : la première observe la bombe quitter les
// mains à l'instant attendu, la seconde constate seulement que personne d'autre ne la tenait.
// La force de preuve diffère, donc l'ordre est structurel (deux passes, cf. `bombArmsByXUID`)
// et l'événement publié DIT laquelle l'a nommé (`BombEvent.ActorSource`).
//
// ## RÈGLE PRIMAIRE — le lâcher (mesurée en E2, 2026-09-04)
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
// ## REPLI — le porteur ACTIF à l'instant armé (E2-bis, arbitré par l'utilisateur 2026-09-04)
//
// CE QUE LA RÈGLE PRIMAIRE NE COUVRE PAS, ET QUI EST MESURÉ : 3 armements sur 13 n'ont AUCUN
// lâcher dans la fenêtre parce que LE PORTEUR TRAVERSE LA POSE — sa période couvre l'instant
// armé et ne se ferme pas là. Sur `35b75a31` à 299 176 ms, la période du porteur
// ([290 194, 308 258], horloge du match) ne se ferme que 4 245 ms APRÈS l'explosion : sur cette
// pose-là, le canal des armes tenues n'émet pas de lâcher exploitable. Même figure sur
// `1c01e34f` (395 764) et `69b16f5d` (273 746).
//
//	À DÉFAUT de règle primaire, l'ARMEUR est le porteur dont une période de portage FERMÉE
//	  A. couvre l'instant armé (DebutMS <= t_arm <= FinMS) ;
//	  B. est la SEULE dans ce cas — deux porteurs candidats laissent l'armement SANS acteur ;
//	  C. n'a pas déjà servi à un autre armement.
//
// La règle n'est pas neuve : c'est la branche « période active à t » de `b2PorteurA`, le juge
// de B2/B3, validé sur les cinq films du corpus. Ce qui est neuf, c'est de la porter en
// production DERRIÈRE le lâcher, jamais à sa place.
//
// CONDITION B — LE REPLI NE TRANCHE JAMAIS ENTRE DEUX. Le lâcher, lui, désigne un GESTE et sait
// donc départager (condition 4 : le plus proche gagne) ; la présence ne désigne rien. Si deux
// périodes couvrent l'instant armé — y compris quand l'une est au slot NON PONTÉ, car un
// porteur anonyme reste un porteur — l'armement est publié sans acteur (`ArmingsAmbiguous`).
// Mieux vaut un armement anonyme qu'un armeur faux.
//
// LES PÉRIODES RESTÉES OUVERTES sont HORS candidature, ici comme dans `CarryMSByXUID` : leur
// `FinMS` n'est pas une mesure mais la sentinelle `HeldObjectOpenEndMS`, et s'en servir comme
// borne de couverture ferait d'une période ouverte la candidate de TOUT armement postérieur —
// le repli dégénérerait en « le dernier qui a ramassé la bombe ». Une période fermée par la
// MORT du porteur, elle, EST candidate : sa fin est datée par le fil des morts, donc « il la
// tenait à cet instant » y est une mesure.
//
// # LE RECALAGE D'HORLOGE : DEUX SOURCES, DEUX ZÉROS
//
// C'est le piège de ce lot, et il ne se voit dans AUCUN test pur qui l'oublierait. Il vaut pour
// les DEUX règles : le repli compare lui aussi un instant du FILM à des périodes du MATCH.
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
// le terme saute — pour la règle primaire ET pour le repli.
//
// # CE QUI N'EST PAS DEVINÉ
//
// Un armement qu'aucune des deux règles ne résout, ou dont la période retenue est au slot NON
// PONTÉ, est PUBLIÉ SANS ACTEUR : l'instant est vrai, le nom manque, et inventer le nom serait
// la seule vraie faute. Les TROIS raisons se comptent séparément (`ArmingsNoCarrier`,
// `ArmingsNoBridge`, `ArmingsAmbiguous`) et le contrôle de cohérence publié tient par
// construction : somme des `bomb_arms` == `ArmingsAttributed` == `ArmingsByDrop` +
// `ArmingsByActiveCarry` <= `Armings`.

import (
	"sort"
	"strconv"
)

// BombEventArmed est la valeur `event_type` d'un armement de bombe daté, sous l'objectif
// `objectiveevents.ObjectiveTypeBomb` — le pendant de BombEventDetonated.
const BombEventArmed = "bomb_armed"

// BombActorSourceDrop / BombActorSourceActiveCarry disent LAQUELLE des deux règles a nommé
// l'acteur d'un fait daté. Ce n'est pas un détail de journal : les deux n'ont pas la même force
// de preuve (un geste observé contre une présence constatée), et un lecteur qui les
// confondrait surestimerait ce que la mesure établit. La persistance les portera dans la
// colonne `details` de `match_objective_events` — `source` y est déjà prise par la provenance
// du DÉCODAGE (vocabulaire `objectiveevents.Source*`), qui est une autre question.
const (
	// BombActorSourceDrop : la bombe a QUITTÉ les mains dans la fenêtre de l'armement.
	BombActorSourceDrop = "carry_drop"
	// BombActorSourceActiveCarry : personne d'autre ne la tenait à cet instant.
	BombActorSourceActiveCarry = "carry_active"
)

// bombArmDropWindowMS est la demi-fenêtre de la règle PRIMAIRE : écart maximal accepté entre la
// fermeture par lâcher d'une période de portage et l'instant armé, une fois les deux sur
// l'horloge du MATCH. Cf. l'en-tête pour les trois grandeurs qui la bornent. Le repli n'a pas
// de fenêtre : il exige une COUVERTURE stricte de l'instant armé.
const bombArmDropWindowMS = 2500

// bombArmCandidate est une période de portage FERMÉE, réduite à ce que la jointure lit. Les
// périodes ouvertes n'en sont pas (cf. l'en-tête) : leur fin est une sentinelle, pas une mesure.
type bombArmCandidate struct {
	// XUID du porteur, 0 quand le pont n'a pas nommé le slot.
	XUID uint64
	// DebutMS / FinMS sur l'horloge du MATCH.
	DebutMS, FinMS int
	// ParLacher : la période s'est fermée sur une transition DEPUIS la famille bombe — le
	// geste de pose. Faux = fermée par la mort du porteur, qui n'est pas un geste.
	ParLacher bool
}

// bombArmVerdict est ce que les deux passes ont statué pour UN armement : l'index de la période
// retenue (-1 si aucune), la règle qui l'a retenue, et le cas particulier de l'ambiguïté du
// repli — deux porteurs candidats, aucun acteur publié.
type bombArmVerdict struct {
	periode int
	source  string
	ambigu  bool
}

// bombArmsByXUID attribue chaque armement daté à un poseur quand l'une des deux règles le
// permet, et rend les faits datés correspondants — sur l'horloge du FILM, comme les explosions.
//
// DEUX PASSES, ET C'EST LA STRUCTURE QUI GARANTIT LA PRIORITÉ : la passe 1 sert la règle du
// lâcher sur TOUS les armements avant que la passe 2 ne serve le repli sur les restants. Un
// seul parcours chronologique mêlant les deux laisserait un repli précoce consommer une période
// qu'un lâcher ultérieur réclamait — le second recours volerait la source primaire.
//
// La consommation des périodes est exclusive, dans les deux passes : deux armements ne peuvent
// pas se réclamer du même porteur.
func bombArmsByXUID(in BombStatsInput, cov *BombStatsCoverage) (map[string]int, []BombEvent) {
	if !in.ArmingsRead {
		return nil, nil
	}
	cov.Armings = len(in.Armings)
	armings := append([]BombArming(nil), in.Armings...)
	sort.SliceStable(armings, func(i, j int) bool { return armings[i].TimeMS < armings[j].TimeMS })
	cands := bombArmCandidates(in)
	used := make([]bool, len(cands))
	verdicts := make([]bombArmVerdict, len(armings))

	// PASSE 1 — la source primaire : le lâcher.
	for i, a := range armings {
		j := bombPoseurOf(cands, used, a.TimeMS+in.FilmToMatchOffsetMS)
		verdicts[i] = bombArmVerdict{periode: j}
		if j >= 0 {
			used[j] = true
			verdicts[i].source = BombActorSourceDrop
		}
	}
	// PASSE 2 — le repli, sur les seuls armements que la passe 1 n'a pas statués.
	for i, a := range armings {
		if verdicts[i].periode >= 0 {
			continue
		}
		j, ambigu := bombPorteurActifOf(cands, used, a.TimeMS+in.FilmToMatchOffsetMS)
		verdicts[i] = bombArmVerdict{periode: j, ambigu: ambigu}
		if j >= 0 {
			used[j] = true
			verdicts[i].source = BombActorSourceActiveCarry
		}
	}
	return bombArmSynthese(armings, verdicts, cands, cov)
}

// bombArmSynthese transforme les verdicts en comptes par joueur, en faits datés et en
// couverture. Chaque armement daté sort un fait — avec acteur quand il est nommé, sans acteur
// sinon, et la raison de l'absence est comptée à part.
func bombArmSynthese(
	armings []BombArming, verdicts []bombArmVerdict, cands []bombArmCandidate,
	cov *BombStatsCoverage,
) (map[string]int, []BombEvent) {
	counts := map[string]int{}
	events := make([]BombEvent, 0, len(armings))
	for i, a := range armings {
		ev := BombEvent{Type: BombEventArmed, TimeMS: a.TimeMS}
		v := verdicts[i]
		switch {
		case v.periode < 0 && v.ambigu:
			cov.ArmingsAmbiguous++
		case v.periode < 0:
			cov.ArmingsNoCarrier++
		case cands[v.periode].XUID == 0:
			cov.ArmingsNoBridge++
		default:
			ev.XUID = strconv.FormatUint(cands[v.periode].XUID, 10)
			ev.ActorSource = v.source
			counts[ev.XUID]++
			cov.ArmingsAttributed++
			if v.source == BombActorSourceDrop {
				cov.ArmingsByDrop++
			} else {
				cov.ArmingsByActiveCarry++
			}
		}
		events = append(events, ev)
	}
	return counts, events
}

// bombArmCandidates retient les périodes FERMÉES, en marquant celles qui portent un GESTE DE
// POSE. Une période fermée par la MORT du porteur n'est pas une pose (le canal n'émet rien à la
// mort, c'est le fil des morts qui la ferme) — elle reste candidate au REPLI, qui ne lit que la
// présence, jamais à la règle primaire. Une période restée OUVERTE à la fin du film est écartée
// des deux : elle n'a pas de lâcher, et sa fin n'est pas une mesure.
//
// Triées par instant de fin puis par xuid — un ordre TOTAL, donc une sortie reproductible.
func bombArmCandidates(in BombStatsInput) []bombArmCandidate {
	if !in.CarryRead {
		return nil
	}
	out := make([]bombArmCandidate, 0, len(in.Carry.Periods))
	for _, p := range in.Carry.Periods {
		if p.Ouverte {
			continue
		}
		out = append(out, bombArmCandidate{
			XUID: p.XUID, DebutMS: p.DebutMS, FinMS: p.FinMS, ParLacher: !p.FinParMort,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].FinMS != out[j].FinMS {
			return out[i].FinMS < out[j].FinMS
		}
		return out[i].XUID < out[j].XUID
	})
	return out
}

// bombPoseurOf est la RÈGLE PRIMAIRE : l'index de la période fermée PAR LÂCHER la plus proche
// de l'instant armé (horloge du MATCH), ou -1 quand aucune ne tient les conditions 1 à 4 de
// l'en-tête. À écart égal, la première dans l'ordre trié gagne — un ordre total, donc une
// sortie reproductible.
func bombPoseurOf(cands []bombArmCandidate, used []bool, armMatchMS int) int {
	best, bestEcart := -1, 0
	for i, c := range cands {
		if used[i] || !c.ParLacher || c.DebutMS > armMatchMS {
			continue
		}
		ecart := c.FinMS - armMatchMS
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

// bombPorteurActifOf est le REPLI : l'index de l'UNIQUE période qui COUVRE l'instant armé
// (conditions A à C de l'en-tête). Rend (-1, true) dès qu'une DEUXIÈME période le couvre — la
// présence ne départage pas, et le repli ne tranche jamais entre deux porteurs. Rend
// (-1, false) quand aucune ne le couvre.
func bombPorteurActifOf(cands []bombArmCandidate, used []bool, armMatchMS int) (int, bool) {
	best := -1
	for i, c := range cands {
		if used[i] || c.DebutMS > armMatchMS || c.FinMS < armMatchMS {
			continue
		}
		if best >= 0 {
			return -1, true
		}
		best = i
	}
	return best, false
}
