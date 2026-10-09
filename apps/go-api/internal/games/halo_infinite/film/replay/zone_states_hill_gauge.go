package replay

// zone_states_hill_gauge.go — LA JAUGE DE CAPTURE D UNE COLLINE : la serie `ZoneState.Gauge` et ses
// segments `ZoneState.GaugeRamps`, lus sur le BLOC de l objet de mode KOTH (designateur, proprietaire,
// pousseur, jauge — zone_states_hill.go).
//
// CE QUE LE CANAL DIT SUR UNE COLLINE (mesure sur les films a colline du parc, journal du plan
// `.ai/PLAN_KOTH_REJEU_2026-10-09.md`) :
//
//	prise      un camp seul dans une colline libre : la jauge monte de 0 a ~1 en environ une
//	           seconde, le POUSSEUR nomme ce camp ; a l arrivee la jauge retombe a 0 et le
//	           PROPRIETAIRE bascule ;
//	vidange    une colline tenue que son camp perd : la jauge saute pres de 1 puis se vide en
//	           environ une seconde, pousseur au NEUTRE ; a 0 le proprietaire passe au neutre ;
//	contestee  une prise interrompue se fige ou redescend, puis reprend.
//
// Une variante a prise instantanee (le classe) n emet aucune jauge : rien n est publie.
//
// LA SERIE est l allegement de Bastion (`appendGaugeThinned`, memes seuils, meme echelle) applique
// aux emissions de la jauge pendant chaque periode ou la colline est active — sans le filtre « rien
// hors rampe » de Bastion, puisque la jauge d une colline redescend aussi pas a pas.
//
// LES SEGMENTS decoupent les emissions non nulles a POUSSEUR constant, chacun ferme par le retour a
// zero qui le suit : pousseur = camp connu -> la prise de ce camp (`CapturingTeam`) ; pousseur
// neutre et jauge qui ne remonte pas -> `Draining` ; tout autre cas -> segment sans camp.

import (
	"cmp"
	"slices"
	"sort"
)

// hillGaugeInput porte les deux canaux du bloc que la jauge lit, et ce dont la publication a besoin.
type hillGaugeInput struct {
	gauge  []zoneSample
	pusher []zoneSample
	teams  map[uint64]bool
	gap    int
}

// hillGaugeInputOf rend la jauge et le pousseur du bloc de l objet de mode : par le NOM (le bloc
// dont le designateur est la cle), ou par le VOISINAGE quand le designateur l a ete (le bloc tient
// sur quatre slots consecutifs : designateur, proprietaire, pousseur, jauge).
func hillGaugeInputOf(ser zoneSeries, d hillDesignator, teams map[uint64]bool, c zoneCtx) hillGaugeInput {
	in := hillGaugeInput{teams: teams, gap: zoneGaugeGapFrames(c.intervalMS)}
	pous, jauge := d.slot+2, d.slot+3
	if b, ok := zoneBlocDuSlot(d.slot, ser.noms, func(b zoneBlocNomme) uint32 { return b.cle }); ok {
		var okP, okJ bool
		pous, okP = ser.noms.parNom[b.pousseur]
		jauge, okJ = ser.noms.parNom[b.jauge]
		if !okP || !okJ {
			return in
		}
	}
	in.gauge, in.pusher = ser.gauge[jauge], ser.owner[pous]
	return in
}

// attachHillGauges pose sur chaque colline publiee la jauge de ses periodes actives.
func attachHillGauges(states []ZoneState, periods []hillPeriod, in hillGaugeInput) {
	if len(in.gauge) == 0 {
		return
	}
	idx := map[int]int{}
	for i, s := range states {
		idx[s.ZoneRef] = i
	}
	for _, p := range periods {
		i, ok := idx[p.ref]
		if !ok {
			continue
		}
		states[i].Gauge, _ = appendGaugeThinned(states[i].Gauge, in.gauge, p.t0, p.t1, in.gap)
		states[i].GaugeRamps = append(states[i].GaugeRamps, hillGaugeSegments(in, p.t0, p.t1)...)
	}
	for i := range states {
		slices.SortStableFunc(states[i].GaugeRamps, func(a, b ZoneGaugeRamp) int { return cmp.Compare(a.T0, b.T0) })
	}
}

// hillSegClass est la classe d une emission non nulle : le camp qui pousse, la vidange, ou rien.
type hillSegClass struct {
	team     int
	known    bool
	draining bool
}

// hillGaugeSegments decoupe les emissions de [t0, t1] en segments a pousseur constant.
func hillGaugeSegments(in hillGaugeInput, t0, t1 int) []ZoneGaugeRamp {
	var out []ZoneGaugeRamp
	var cur *ZoneGaugeRamp
	var first, last uint64
	for _, s := range in.gauge {
		if s.t < t0 || s.t > t1 {
			continue
		}
		if s.v <= zoneGaugeQuantZero {
			if cur != nil {
				cur.T1 = s.t
				out = closeHillSegment(out, cur, first, last)
				cur = nil
			}
			continue
		}
		cl := hillSegClassAt(in, s.t)
		if cur != nil && !sameHillClass(*cur, cl) {
			out = closeHillSegment(out, cur, first, last)
			cur = nil
		}
		if cur == nil {
			cur, first = &ZoneGaugeRamp{T0: s.t, Draining: cl.draining}, s.v
			if cl.known {
				cur.CapturingTeam = new(cl.team)
			}
		}
		cur.T1, last = s.t, s.v
	}
	if cur != nil {
		out = closeHillSegment(out, cur, first, last)
	}
	return out
}

// hillSegClassAt classe l emission de l instant `t` par la valeur du pousseur a cet instant.
func hillSegClassAt(in hillGaugeInput, t int) hillSegClass {
	v, ok := zoneSampleAt(in.pusher, t)
	switch {
	case !ok:
		return hillSegClass{}
	case v == zoneNeutralOwner:
		return hillSegClass{draining: true}
	}
	team, known := zoneOwnerTeam(v, in.teams)
	if !known || team == nil {
		return hillSegClass{}
	}
	return hillSegClass{team: *team, known: true}
}

// sameHillClass dit si l emission prolonge le segment en cours.
func sameHillClass(r ZoneGaugeRamp, cl hillSegClass) bool {
	if r.Draining != cl.draining || (r.CapturingTeam != nil) != cl.known {
		return false
	}
	return !cl.known || *r.CapturingTeam == cl.team
}

// closeHillSegment ferme un segment : une « vidange » dont la jauge finit plus haut qu elle n a
// commence n en est pas une (le pousseur neutre ne dit alors rien), elle perd sa classe.
func closeHillSegment(out []ZoneGaugeRamp, r *ZoneGaugeRamp, first, last uint64) []ZoneGaugeRamp {
	if r.Draining && last > first {
		r.Draining = false
	}
	return append(out, *r)
}

// zoneSampleAt rend la derniere valeur d une serie a l instant `t` (ou avant).
func zoneSampleAt(ss []zoneSample, t int) (uint64, bool) {
	i := sort.Search(len(ss), func(k int) bool { return ss[k].t > t })
	if i == 0 {
		return 0, false
	}
	return ss[i-1].v, true
}
