package replay

import (
	"sort"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// flag_carries_lives.go — DE PORTAGES ISOLES A LA VIE D'UN DRAPEAU.
//
// CE QUE CE FICHIER AJOUTE AUX PORTAGES : l'ENTRE-DEUX. Un portage dit « ce joueur l'a tenu de t0
// a t1 » ; il ne dit pas ou est le drapeau le reste du temps. La simulation chronologique
// ci-dessous le tient a jour, drapeau par drapeau, avec quatre etats et trois seules transitions :
//
//	prise    ->  `carried` a la position du porteur quand un fait DATE ferme le portage,
//	             `carried_open` sinon (rien ne le ferme : l'intervalle court jusqu'a la fin de
//	             l'axe, et c'est une borne haute publiee comme telle)
//	fin      ->  `home` si le portage s'est acheve sur une CAPTURE (le drapeau rentre a sa base),
//	             `dropped` sinon, a la derniere position connue du porteur
//	retour   ->  `home` sur un `flag_returns`, quand UN SEUL drapeau est au sol
//
// LE RETOUR NE NOMME PAS SON DRAPEAU. `flag_returns` est credite au joueur qui touche le drapeau
// de son equipe ; l'evenement ne porte ni l'objet ni l'equipe. On l'applique donc au seul drapeau
// AU SOL a cet instant, et on s'abstient quand il y en a zero ou deux — se taire vaut mieux que
// renvoyer le mauvais drapeau a une base ou il n'est pas. Le compte des abstentions est publie.
//
// LE RETOUR AUTOMATIQUE N'EST PAS SIMULE, ET C'EST UNE MESURE, PAS UN OUBLI. Halo renvoie un
// drapeau reste au sol au bout d'un delai ; le film n'en porte AUCUN evenement. Le delai a ete
// cherche sur les trois films CTF par l'ecart entre une fin de portage sans reprise et la prise
// suivante au socle (item 1.2 du plan) : la distribution est trop dispersee pour qu'un seuil s'en
// deduise sans se tromper, et le resultat est publie tel quel plutot qu'arrondi. Un drapeau au
// sol reste donc `dropped` jusqu'a sa reprise, son retour ou la fin du match — un etat trop long,
// jamais une position inventee.
//
// SANS SOCLE, PAS DE `home`. Une carte hors du catalogue d'objectifs ne donne aucune position de
// base : les etats `home` sont alors OMIS (leur position serait inventee), et la vie du drapeau
// se reduit a ses portages et a ses laches. La couverture publie `Spawns: 0`.

// flagStateUnknown est l ETAT ABSENT : une transition qui BORNE le span precedent sans rien
// affirmer. Seul emploi : une capture sur une carte dont le socle est inconnu — le drapeau rentre
// quelque part, et ce quelque part n est pas connu.
const flagStateUnknown = ""

// flagLifeEventKind ordonne les evenements simultanes : une fin LIBERE le drapeau avant qu'un
// retour ou une nouvelle prise ne le reclame.
type flagLifeEventKind int

const (
	flagLifeClose flagLifeEventKind = iota
	flagLifeReturn
	flagLifeOpen
)

// flagLifeEvent est un changement d'etat date, sur l'horloge du MATCH.
type flagLifeEvent struct {
	at   int64
	kind flagLifeEventKind
	// carry indexe `raws` pour les ouvertures et les fermetures ; -1 pour un retour.
	carry int
}

// flagTransition est un etat qui COMMENCE a une frame donnee.
type flagTransition struct {
	frame int
	state string
	xuid  *string
	x, y  float32
}

// assembleFlagLives rend la vie de chaque drapeau : des spans contigus, tries, sans recouvrement.
func assembleFlagLives(raws []flagCarryRaw, scan FlagCarryScan, ctx flagCarryCtx,
	cov *FlagCarriesCoverage) []FlagCarry {
	if ctx.frames <= 0 || len(raws) == 0 {
		return nil
	}
	n := len(scan.Spawns)
	if n == 0 {
		n = 1
	}
	trans := make([][]flagTransition, n)
	for f := 0; f < n && f < len(scan.Spawns); f++ {
		trans[f] = append(trans[f], flagTransition{
			frame: 0, state: FlagStateHome, x: scan.Spawns[f].X, y: scan.Spawns[f].Y,
		})
	}
	state := make([]string, n)
	for f := range state {
		state[f] = FlagStateHome
	}
	for _, ev := range flagLifeTimeline(raws, scan) {
		applyFlagLifeEvent(ev, raws, scan, flagLifeState{trans: trans, state: state, ctx: ctx, cov: cov})
	}
	return flagCarriesOf(trans, scan, ctx.frames)
}

// flagLifeState regroupe ce que l'application d'un evenement fait evoluer — sans elle,
// `applyFlagLifeEvent` prendrait sept parametres.
type flagLifeState struct {
	trans [][]flagTransition
	state []string
	ctx   flagCarryCtx
	cov   *FlagCarriesCoverage
}

// flagLifeTimeline rend la suite chronologique des evenements de vie des drapeaux.
func flagLifeTimeline(raws []flagCarryRaw, scan FlagCarryScan) []flagLifeEvent {
	out := make([]flagLifeEvent, 0, 2*len(raws))
	for i, r := range raws {
		out = append(out, flagLifeEvent{at: r.t0, kind: flagLifeOpen, carry: i})
		if r.closed {
			out = append(out, flagLifeEvent{at: r.t1, kind: flagLifeClose, carry: i})
		}
	}
	for _, t := range flagReturnTimes(scan) {
		out = append(out, flagLifeEvent{at: t, kind: flagLifeReturn, carry: -1})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].at != out[j].at {
			return out[i].at < out[j].at
		}
		return out[i].kind < out[j].kind
	})
	return out
}

// flagReturnTimes rend les instants tries des `flag_returns` du film.
func flagReturnTimes(scan FlagCarryScan) []int64 {
	var out []int64
	for _, e := range scan.Events {
		if e.Stat == objectiveevents.StatFlagReturns {
			out = append(out, int64(e.TimeMS))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// applyFlagLifeEvent fait evoluer l'etat des drapeaux d'un evenement.
func applyFlagLifeEvent(ev flagLifeEvent, raws []flagCarryRaw, scan FlagCarryScan, st flagLifeState) {
	if ev.kind == flagLifeReturn {
		applyFlagReturn(ev, scan, st)
		return
	}
	r := raws[ev.carry]
	f := r.flagIndex
	if f < 0 || f >= len(st.state) {
		return
	}
	switch ev.kind {
	case flagLifeOpen:
		xuid := r.xuid
		// UN PORTAGE QUE RIEN NE FERME NE PORTE PAS LE MEME NOM. `closed` dit qu'un fait DATE a
		// mis fin au portage ; sans lui l'intervalle court jusqu'a la fin de l'axe, ce qui est
		// une BORNE HAUTE et non une mesure (le lacher volontaire n'est date par rien). L'etat
		// [FlagStateCarriedOpen] publie ce doute au lieu de le noyer dans la certitude.
		state := FlagStateCarriedOpen
		if r.closed {
			state = FlagStateCarried
		}
		st.state[f] = state
		st.trans[f] = append(st.trans[f], flagTransition{
			frame: st.ctx.frameOfMatchMS(r.t0), state: state, xuid: &xuid, x: r.x0, y: r.y0,
		})
	case flagLifeClose:
		next := flagTransition{frame: st.ctx.frameOfMatchMS(r.t1) + 1, state: FlagStateDropped, x: r.x1, y: r.y1}
		if r.captured {
			if f >= len(scan.Spawns) {
				// Socle inconnu : on n invente pas sa position. La transition SANS ETAT
				// borne le portage sans rien affirmer de la suite.
				next = flagTransition{frame: next.frame, state: flagStateUnknown}
			} else {
				next.state, next.x, next.y = FlagStateHome, scan.Spawns[f].X, scan.Spawns[f].Y
			}
		}
		st.state[f] = next.state
		st.trans[f] = append(st.trans[f], next)
	}
}

// applyFlagReturn renvoie a sa base le SEUL drapeau au sol, ou s'abstient et se compte.
func applyFlagReturn(ev flagLifeEvent, scan FlagCarryScan, st flagLifeState) {
	only, several := -1, false
	for f, s := range st.state {
		if s != FlagStateDropped {
			continue
		}
		if only >= 0 {
			several = true
			break
		}
		only = f
	}
	if several || only < 0 || only >= len(scan.Spawns) {
		st.cov.AmbiguousReturns++
		return
	}
	st.state[only] = FlagStateHome
	st.trans[only] = append(st.trans[only], flagTransition{
		frame: st.ctx.frameOfMatchMS(ev.at), state: FlagStateHome,
		x: scan.Spawns[only].X, y: scan.Spawns[only].Y,
	})
}

// flagCarriesOf transforme les transitions de chaque drapeau en spans contigus.
func flagCarriesOf(trans [][]flagTransition, scan FlagCarryScan, frames int) []FlagCarry {
	out := make([]FlagCarry, 0, len(trans))
	for f, ts := range trans {
		spans := spansOfTransitions(ts, frames)
		if len(spans) == 0 {
			continue
		}
		team := TeamNeutral
		if f < len(scan.Spawns) {
			team = scan.Spawns[f].Team
		}
		out = append(out, FlagCarry{Team: team, Spans: spans})
	}
	return out
}

// spansOfTransitions borne chaque transition par la suivante et jette celles de duree nulle.
//
// Une transition de duree nulle n'est pas une anomalie : une prise qui suit immediatement une fin
// (reprise au sol dans la meme frame) produit un `dropped` qui ne dure pas une frame. Le publier
// donnerait au client un etat a dessiner qui n'a jamais existe a l'ecran.
func spansOfTransitions(ts []flagTransition, frames int) []FlagSpan {
	if len(ts) == 0 {
		return nil
	}
	sort.SliceStable(ts, func(i, j int) bool { return ts[i].frame < ts[j].frame })
	out := make([]FlagSpan, 0, len(ts))
	for i, t := range ts {
		if t.state == flagStateUnknown {
			continue // borne le span precedent sans rien publier (socle inconnu)
		}
		t0 := clampFrame(t.frame, frames)
		t1 := frames - 1
		if i+1 < len(ts) {
			t1 = clampFrame(ts[i+1].frame, frames) - 1
		}
		if t1 < t0 {
			continue
		}
		if n := len(out); n > 0 && out[n-1].State == t.state && sameFlagPos(out[n-1], t) {
			out[n-1].T1 = t1
			continue
		}
		out = append(out, FlagSpan{State: t.state, T0: t0, T1: t1, XUID: t.xuid, X: t.x, Y: t.y})
	}
	return out
}

// sameFlagPos dit si une transition prolonge le span courant sans rien changer.
func sameFlagPos(s FlagSpan, t flagTransition) bool {
	if s.X != t.x || s.Y != t.y {
		return false
	}
	switch {
	case s.XUID == nil && t.xuid == nil:
		return true
	case s.XUID == nil || t.xuid == nil:
		return false
	}
	return *s.XUID == *t.xuid
}
