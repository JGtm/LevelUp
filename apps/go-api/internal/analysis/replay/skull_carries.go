package replay

import (
	"log/slog"
	"sort"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// skull_carries.go — LA REGLE : de quoi est faite une periode de portage du CRANE d'Oddball.
//
// # Le principe, en une phrase
//
// On ne decode PAS le crane : on lit QUI le PORTE. Le porteur est le joueur dont les TICS DE
// SCORE DE MODE montent — en Oddball, `comp 0 A` (le score de mode par joueur) compte les tics de
// possession (`skull_scoring_ticks`). Un TRAIN de tics d'un meme joueur EST une periode de
// portage ; le crane est a la position de ce joueur, et le client le pose sur sa piste deja
// publiee, comme la couronne VIP et le drapeau porte.
//
// # Ce qu'il a fallu pour l'etablir, et pourquoi ce canal
//
// Le portage a resiste a CINQ campagnes (proximite, traversee, score PERSONNEL) : negatifs, biais
// des longs portages. Le canal des TICS de score de MODE, lui, tient : gate oracle porteur
// PRINCIPAL correct 7/7 films, gate terrain manche 1 de d9781168 prises 9/9 et porteurs
// d'intervalle 8/9 (seuil 8/9). L'emplacement (`comp 0 A`) est identifie par l'oracle films
// confondus, PAS ajuste au film terrain. Detail : `ODDBALL_PORTEUR_PROTOCOLE.md` +
// `TERRAIN_*.log`.
//
// # PAR MANCHE, et c'est structurel
//
// Les tics sont lus MANCHE PAR MANCHE (`SeriesByRound`), et le porteur d'un train est nomme par
// l'identite de SA manche ([objectiveevents.RoundIdentity.AtRound]) : le slot d'entite est
// reattribue d'une manche a l'autre. Une bascule de manche NE FERME PAS un portage par une fausse
// prise — les trains sont bornes par les seuls TROUS DE TICS, et chaque manche est un parcours
// distinct, donc le dernier train d'une manche et le premier de la suivante ne se melangent pas.
//
// # Ce qui n'est PAS decide ici, et c'est delibere
//
// Le MODE. `comp 0 A` est le score de mode de N'IMPORTE quel mode. La garde est chez l'APPELANT
// (`replaybuild`, qui connait `game_variant_name`), comme la couronne VIP et la colline de KOTH :
// ce paquet consomme un `SkullInput` et ne devine aucun mode. Un film non-Oddball ne fournit pas
// de `SkullInput.Scanned`, et le calque reste vide.

// skullTickGapMS : au-dela de ce trou entre deux tics d'un meme joueur, la periode de portage se
// ferme. Les tics tombent a ~1 Hz pendant le portage ; trois secondes separent nettement deux
// periodes distinctes sans couper une periode continue (meme valeur que l'instrument du gate).
const skullTickGapMS = 3000

// SkullInput est CE QUE L'APPELANT FOURNIT du crane. Entree de DONNEES, comme `Flag` et `Vip`.
//
// LA GARDE DE MODE EST ICI, chez l'appelant : `comp 0 A` est le score de mode de tout mode, donc
// seul un appelant qui SAIT que le match est Oddball (par `game_variant_name`) doit poser
// `Scanned`. `Scanned` faux = ni calque ni couverture.
type SkullInput struct {
	// Scanned dit que l'appelant a RECONNU un film Oddball et fournit de quoi lire.
	Scanned bool
	// Records sont les enregistrements d'entite du film — les MEMES que la courbe de score, le
	// drapeau et la couronne : ils portent les tics de score de mode (`comp 0 A`), les prises
	// (`comp 21 B`) et les progressions du compteur de morts qui identifient les slots. Aucun fait
	// de match n'entre : le porteur se nomme par les instants de mort, et le calque est donc
	// publiable hors ligne.
	Records []objectiveevents.StatRecord
}

// SkullCarryScan porte ce que le film rend du porteur. Les lectures voyagent ensemble, et
// `Scanned` dit qu'elles ont abouti : une liste vide sans lui serait indistinguable d'un film
// non-Oddball.
type SkullCarryScan struct {
	Scanned bool
	// Records : les enregistrements d'entite (tics de score de mode + prises).
	Records []objectiveevents.StatRecord
	// Identity est le pont slot statborg -> xuid PAR MANCHE (par les instants de mort).
	Identity objectiveevents.RoundIdentity
}

// L'AXE DE TEMPS est le `matchClock` partagé (match_clock.go) : la conversion match -> frames
// était la même que celle du drapeau et de la couronne, elle n'est plus écrite qu'une fois.
// Le trou de fermeture d'un train se traduit en frames par `slackFrames(skullTickGapMS)` : un
// portage dont le dernier tic est a moins d'un trou de la fin de l'axe n'a ete ferme par aucun
// fait — il court jusqu'au bout (le film s'arrete pendant le portage). Au-dela, une fermeture
// est un vrai fait (une chute suivie d'une reprise, ou une fin de manche).

// skullRawCarry est une periode de portage reconstruite : un train de tics d'un meme slot dans une
// manche, en horloge du MATCH.
type skullRawCarry struct {
	xuid       string
	round      int
	t0MS, t1MS int
}

// buildSkullCarries rend les periodes de portage du crane en FRAMES et la couverture du calque.
//
// Rend (nil, nil) quand rien n'a ete balaye (film non-Oddball), et (nil, couverture) quand le film
// est Oddball mais qu'aucune periode ne sort — la couverture dit alors POURQUOI.
//
// `presence` est l'index des vies bipedes publiees (cf. [carrierPresence]) et decide, par
// [carrierPresence.gate], de ce qui sort : un portage dont les pistes publiees prouvent que le
// porteur etait AILLEURS est un FANTOME et part en `CarrierAbsent` ; un portage qui deborde d'une
// vie NOMMEE du porteur est ROGNE a elle. Partout ailleurs — porteur jamais nomme, ou vie ANONYME
// couvrant l'intervalle — le gate S'ABSTIENT : on ne rejette pas l'inconnu. `presence` zero
// (tests) laisse donc passer tous les trains.
func buildSkullCarries(scan SkullCarryScan, ctx matchClock, presence carrierPresence) ([]SkullCarry, *SkullCarriesCoverage) {
	if !scan.Scanned {
		return nil, nil
	}
	cov := &SkullCarriesCoverage{SkullFilm: true, Grabs: skullGrabCount(scan.Records)}
	raws := skullCarryIntervals(scan.Records, scan.Identity)
	cov.Trains = len(raws)
	openThreshold := ctx.frames - 1 - ctx.slackFrames(skullTickGapMS)
	out := make([]SkullCarry, 0, len(raws))
	for _, r := range raws {
		if r.xuid == "" {
			cov.NoBridge++
			continue
		}
		f0 := ctx.frameOfMatchMS(int64(r.t0MS))
		if f0 < 0 || f0 >= ctx.frames {
			cov.OutOfWindow++
			continue
		}
		f1 := clampFrame(ctx.frameOfMatchMS(int64(r.t1MS)), ctx.frames)
		if f1 < f0 {
			f1 = f0
		}
		// Gate de PRESENCE : le porteur doit etre sur la carte pendant le portage.
		var ok bool
		if f0, f1, ok = presence.gate(r.xuid, f0, f1); !ok {
			cov.CarrierAbsent++
			continue
		}
		closed := f1 < openThreshold
		out = append(out, SkullCarry{XUID: r.xuid, T0: f0, T1: f1, Closed: closed})
		if closed {
			cov.Closed++
		} else {
			cov.Open++
		}
	}
	cov.Carries = len(out)
	return out, cov
}

// presenceSpan est une fenetre [f0,f1] fermee sur l'axe de frames publie.
type presenceSpan struct{ f0, f1 int }

// carrierPresence est l'index de PRESENCE des porteurs sur l'axe de frames publie — et, ce qui
// compte autant, ce qu'il NE SAIT PAS.
//
// POURQUOI DEUX CHAMPS ET PAS UNE SEULE MAP. Le gate de presence (2026-08-30) a d'abord indexe
// les seules vies NOMMEES, et lu « aucune vie nommee de X ne couvre l'intervalle » comme « X est
// ABSENT de la carte ». C'est un faux syllogisme : le pont d'identite laisse des vies ANONYMES
// (18 slots sur 142 sur `d9781168`), et une vie anonyme est une PRESENCE SANS IDENTITE, pas une
// absence. Mesure du 2026-09-06, chaine independante : en Oddball le score EST le temps de
// portage, et la feuille de match donne 191 s / 196 s par equipe sur `d9781168` ; le gate publiait
// 60,1 s / 147,4 s. Il ecartait deux tiers du temps de portage d'une equipe, en croyant ecarter
// des fantomes. Le champ `unnamed` est ce qui rend l'ignorance VISIBLE au gate.
type carrierPresence struct {
	// named : les vies publiees et NOMMEES, groupees par xuid.
	named map[string][]presenceSpan
	// unnamed : les vies publiees que le pont n'a identifiees NI par xuid NI comme bot.
	// Quelqu'un est la, on ne sait pas qui — donc on ne peut RIEN affirmer sur l'absence d'un
	// joueur a cet instant.
	unnamed []presenceSpan
}

// carrierPresenceOf indexe les vies bipedes PUBLIEES (`doc.Tracks`) : les nommees par xuid, les
// non identifiees a part. Meme axe de frames que les portages (les deux passent par le meme
// `origin`/`step`), donc directement comparables. Sert le crane ET la bombe.
//
// UNE VIE DE BOT N'ENTRE NULLE PART, et c'est voulu. Elle n'a pas de xuid (seul cas ou une vie
// est nommee sans en avoir un, cf. [Track.Bot]), donc elle ne peut pas porter un portage ; mais
// elle est IDENTIFIEE, donc elle ne cree aucun doute sur ou se trouve un joueur. La ranger avec
// les anonymes ferait abstenir le gate sur les 20 films a bots du parc sans raison.
func carrierPresenceOf(tracks []Track) carrierPresence {
	p := carrierPresence{named: map[string][]presenceSpan{}}
	for _, t := range tracks {
		span := presenceSpan{t.StartFrame, t.EndFrame}
		switch {
		case t.XUID != "":
			p.named[t.XUID] = append(p.named[t.XUID], span)
		case t.Bot == "":
			p.unnamed = append(p.unnamed, span)
		}
	}
	return p
}

// gate applique la regle de PRESENCE a un portage [f0,f1] attribue a `xuid`. Il rend les bornes
// a publier et `false` quand le portage est un FANTOME (a ecarter, `CarrierAbsent`).
//
// TROIS CAS D'ABSTENTION, tous ramenes au meme principe : ON NE REJETTE PAS L'INCONNU.
//  1. `xuid` n'a AUCUNE vie nommee (jamais ponte, ou `named` vide en test) : rien a opposer.
//  2. Une vie ANONYME recouvre l'intervalle : la presence y est INCONNUE, pas nulle. Ni rejet ni
//     rognage — rogner reviendrait a affirmer que le porteur n'etait pas la ou une vie sans nom
//     dit que quelqu'un l'etait.
//  3. Une vie nommee de `xuid` recouvre l'intervalle : le portage est publie, ROGNE a la vie qui
//     le recouvre le plus (le crane n'est porte que tant que son porteur est present).
//
// Le rejet ne subsiste donc que quand les pistes publiees rendent COMPTE de tout l'intervalle et
// que le porteur n'y est pas — le seul cas ou « absent » est une mesure et non une ignorance.
func (p carrierPresence) gate(xuid string, f0, f1 int) (int, int, bool) {
	spans := p.named[xuid]
	if len(spans) == 0 {
		return f0, f1, true
	}
	// L'IGNORANCE PASSE AVANT LE ROGNAGE, et c'est la moitie la plus couteuse du correctif : sur
	// `d9781168`, le rejet coutait 32,6 s de portage et le rognage 91,2 s. Rogner un portage a une
	// vie nommee alors qu'une vie SANS NOM couvre le reste, c'est affirmer une absence que rien
	// n'etablit.
	if _, unknown := bestOverlap(p.unnamed, f0, f1); unknown {
		return f0, f1, true
	}
	if span, ok := bestOverlap(spans, f0, f1); ok {
		if f0 < span.f0 {
			f0 = span.f0
		}
		if f1 > span.f1 {
			f1 = span.f1
		}
		return f0, f1, true
	}
	return f0, f1, false
}

// bestOverlap rend la fenetre de presence qui recouvre le plus [f0,f1], et si un recouvrement
// existe. Sans recouvrement (le porteur n'est present a AUCUN instant du portage), (presenceSpan{},
// false) — le portage est un fantome.
func bestOverlap(spans []presenceSpan, f0, f1 int) (presenceSpan, bool) {
	best := presenceSpan{}
	bestOv := 0
	for _, s := range spans {
		lo, hi := f0, f1
		if s.f0 > lo {
			lo = s.f0
		}
		if s.f1 < hi {
			hi = s.f1
		}
		if ov := hi - lo + 1; ov > bestOv {
			bestOv = ov
			best = s
		}
	}
	return best, bestOv > 0
}

// skullCarryIntervals reconstruit les periodes de portage : les trains de tics de score de mode,
// PAR MANCHE et par slot, chaque train nomme par l'identite de sa manche. Ordre TOTAL (instant,
// puis manche, puis xuid) : sans lui le parcours de map rendrait une sortie differente a chaque
// execution.
func skullCarryIntervals(recs []objectiveevents.StatRecord, identity objectiveevents.RoundIdentity) []skullRawCarry {
	bySlot := objectiveevents.SeriesByRound(recs, objectiveevents.SkullTicksComponent, false)
	var out []skullRawCarry
	for slot, byRound := range bySlot {
		for round, pts := range byRound {
			inst := skullTickInstants(pts)
			if len(inst) == 0 {
				continue
			}
			xuid := identity.AtRound(round, slot)
			start, last := inst[0], inst[0]
			for _, t := range inst[1:] {
				if t-last > skullTickGapMS {
					out = append(out, skullRawCarry{xuid: xuid, round: round, t0MS: start, t1MS: last})
					start = t
				}
				last = t
			}
			out = append(out, skullRawCarry{xuid: xuid, round: round, t0MS: start, t1MS: last})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].t0MS != out[j].t0MS {
			return out[i].t0MS < out[j].t0MS
		}
		if out[i].round != out[j].round {
			return out[i].round < out[j].round
		}
		return out[i].xuid < out[j].xuid
	})
	return out
}

// skullTickInstants rend un instant par UNITE gagnee par le compteur de tics (deroulage par
// valeur : la meme valeur reemise ne rajoute rien, si bien que chaque tic est date a sa PREMIERE
// emission).
func skullTickInstants(pts []objectiveevents.ScorePoint) []int {
	var out []int
	prev := int64(0)
	for _, p := range pts {
		for ; prev < p.Value; prev++ {
			out = append(out, p.TimeMS)
		}
	}
	return out
}

// skullGrabCount rend le nombre total de PRISES du crane (`comp 21 B`), toutes manches — le
// denominateur de couverture, independant des trains de tics.
func skullGrabCount(recs []objectiveevents.StatRecord) int {
	total := 0
	for _, byRound := range objectiveevents.SeriesByRound(recs, objectiveevents.SkullGrabsComponent, false) {
		for _, pts := range byRound {
			if n := len(pts); n > 0 {
				total += int(pts[n-1].Value)
			}
		}
	}
	return total
}

// attachSkullCarries pose les periodes de portage du crane sur le document, avec leur couverture.
//
// LE PONT D'IDENTITE (slot statborg -> xuid) SE FAIT ICI, comme pour la couronne et le drapeau,
// par les seuls INSTANTS DE MORT et PAR MANCHE — aucune base. `own.DeathOffsetMS` cale l'horloge
// des enregistrements (meme horloge que le fil des morts) sur l'axe des frames.
func attachSkullCarries(doc *ReplayDocument, opt Options, own OwnerReport, clock replayClock) {
	in := opt.Skull
	if !in.Scanned {
		return
	}
	scan := SkullCarryScan{
		Scanned:  true,
		Records:  in.Records,
		Identity: objectiveevents.ResolveRoundIdentity(in.Records, deathInstantsOf(opt.Deaths)),
	}
	carries, cov := buildSkullCarries(scan, matchClock{
		origin: clock.origin, step: clock.step, frames: clock.frames,
		deathOffsetMS: own.DeathOffsetMS,
	}, carrierPresenceOf(doc.Tracks))
	doc.SkullCarries = carries
	if doc.Coverage != nil {
		doc.Coverage.SkullCarries = cov
	}
	logSkullCarriesCoverage(cov)
}

// logSkullCarriesCoverage journalise ce que le calque publie — et ce qu'il ecarte.
func logSkullCarriesCoverage(cov *SkullCarriesCoverage) {
	if cov == nil {
		return
	}
	slog.Info("rejeu : portage du crane d'Oddball",
		"prises", cov.Grabs, "trains", cov.Trains, "portages", cov.Carries,
		"fermes", cov.Closed, "ouverts", cov.Open,
		"sansPont", cov.NoBridge, "horsFenetre", cov.OutOfWindow,
		"porteurAbsent", cov.CarrierAbsent)
}
