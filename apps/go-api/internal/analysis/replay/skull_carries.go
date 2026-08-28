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

// skullCarryCtx regroupe l'axe de temps et le calage d'horloge film <-> match (memes conventions
// que la couronne VIP).
type skullCarryCtx struct {
	origin, step  uint64
	frames        int
	deathOffsetMS int64
}

// frameOfMatchMS pose un instant du MATCH sur l'axe de frames.
func (c skullCarryCtx) frameOfMatchMS(matchMS int) int {
	if c.step == 0 {
		return -1
	}
	filmUS := (int64(matchMS) + c.deathOffsetMS) * 1000
	if filmUS < int64(c.origin) {
		return -1
	}
	return int((uint64(filmUS) - c.origin) / c.step)
}

// openSlackFrames traduit le trou de fermeture d'un train en frames : un portage dont le dernier
// tic est a moins d'un trou de la fin de l'axe n'a ete ferme par aucun fait — il court jusqu'au
// bout (le film s'arrete pendant le portage). Au-dela, une fermeture est un vrai fait (une chute
// suivie d'une reprise, ou une fin de manche).
func (c skullCarryCtx) openSlackFrames() int {
	if c.step == 0 {
		return 0
	}
	return int((uint64(skullTickGapMS)*1000 + c.step - 1) / c.step)
}

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
func buildSkullCarries(scan SkullCarryScan, ctx skullCarryCtx) ([]SkullCarry, *SkullCarriesCoverage) {
	if !scan.Scanned {
		return nil, nil
	}
	cov := &SkullCarriesCoverage{SkullFilm: true, Grabs: skullGrabCount(scan.Records)}
	raws := skullCarryIntervals(scan.Records, scan.Identity)
	cov.Trains = len(raws)
	openThreshold := ctx.frames - 1 - ctx.openSlackFrames()
	out := make([]SkullCarry, 0, len(raws))
	for _, r := range raws {
		if r.xuid == "" {
			cov.NoBridge++
			continue
		}
		f0 := ctx.frameOfMatchMS(r.t0MS)
		if f0 < 0 || f0 >= ctx.frames {
			cov.OutOfWindow++
			continue
		}
		f1 := clampFrame(ctx.frameOfMatchMS(r.t1MS), ctx.frames)
		if f1 < f0 {
			f1 = f0
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
	carries, cov := buildSkullCarries(scan, skullCarryCtx{
		origin: clock.origin, step: clock.step, frames: clock.frames,
		deathOffsetMS: own.DeathOffsetMS,
	})
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
		"sansPont", cov.NoBridge, "horsFenetre", cov.OutOfWindow)
}
