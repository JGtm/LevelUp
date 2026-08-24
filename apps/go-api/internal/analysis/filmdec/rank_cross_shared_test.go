package filmdec

// rank_cross_shared_test.go — LE TABLEAU D'EXCLUSIVITÉ PAR RANG, partagé par les
// instruments qui cherchent un canal DÉDIÉ à une capacité (item 0.4 et 0.5 du plan
// .ai/V7.5/replay2d/PLAN_CAPACITES_ACTIVES.md).
//
// POURQUOI IL EST ICI ET PAS RECOPIÉ. Deux canaux sont interrogés avec EXACTEMENT le même
// protocole — `i56` (énergie de capacité, chutes de charge) et `i51` (minuteur EMP,
// déclenchements). Recopier le tableau ferait diverger les deux mesures le jour où l'une
// est corrigée, et la règle du dépôt est de centraliser avant la troisième copie.
//
// LE DÉNOMINATEUR EST LA MOITIÉ DE LA MESURE. Une exclusivité ne se juge pas sur un compte
// d'événements : un rang sans événement peut n'avoir aucune LECTURE du canal. Le tableau
// publie donc, par rang, les vies identifiées, celles ayant au moins une lecture, les
// lectures, puis seulement les événements — et le taux se prend sur les vies LUES.

import (
	"fmt"
	"sort"
	"testing"
)

// xrGroup est un groupe de rangs jugé ensemble (une capacité, éventuellement portée par
// deux palettes différentes).
type xrGroup struct {
	name  string
	ranks []int
}

// xrSpec décrit ce que l'instrument mesure : le nom de l'événement compté, et les groupes
// de rangs cibles, évalués DANS L'ORDRE (le premier qui matche gagne — les groupes sont
// disjoints à l'affichage).
type xrSpec struct {
	event  string
	groups []xrGroup
}

// xrRow agrège un groupe de vies. `lives` compte les vies IDENTIFIÉES, `withRead` celles qui
// ont au moins une lecture du canal : c'est sur ces dernières que le taux se prend.
type xrRow struct {
	lives, withRead, reads, withEvent, events int
}

func (r *xrRow) add(reads, events int) {
	r.lives++
	r.reads += reads
	r.events += events
	if reads > 0 {
		r.withRead++
	}
	if events > 0 {
		r.withEvent++
	}
}

func (r xrRow) String() string {
	per := 0.0
	if r.withRead > 0 {
		per = float64(r.events) / float64(r.withRead)
	}
	return fmt.Sprintf("%3d vies · %3d avec lecture · %4d lectures · %3d avec événement · "+
		"%4d événements · %.2f par vie-lue", r.lives, r.withRead, r.reads, r.withEvent, r.events, per)
}

// xrTable publie le tableau événements x rang, puis les groupes disjoints par capacité.
func xrTable(t *testing.T, spec xrSpec, slotRanks map[uint32][]int, reads, events map[uint32]int) {
	t.Helper()
	rows := map[int]*xrRow{}
	for sl, ranks := range slotRanks {
		for _, r := range eaRankSet(ranks) {
			if rows[r] == nil {
				rows[r] = &xrRow{}
			}
			rows[r].add(reads[sl], events[sl])
		}
	}
	keys := make([]int, 0, len(rows))
	for r := range rows {
		keys = append(keys, r)
	}
	sort.Ints(keys)
	t.Logf("== TABLEAU %s x RANG i48 (une vie multi-rangs compte dans chaque rang) ==", spec.event)
	for _, r := range keys {
		t.Logf("  rang %-2d : %s%s", r, rows[r], xrTag(spec, r))
	}
	xrGroups(t, spec, slotRanks, reads, events)
	t.Logf("RAPPEL du critère : la quasi-totalité de la masse de %s doit tomber sur les vies "+
		"du rang CIBLE, et 0 ou quasi-0 sur les autres rangs LUS. Un rang sans lecture ne "+
		"prouve rien. Verdict PAR CAPACITÉ, jamais groupé.", spec.event)
}

// xrTag marque les rangs cibles dans le tableau.
func xrTag(spec xrSpec, r int) string {
	for _, g := range spec.groups {
		for _, x := range g.ranks {
			if x == r {
				return "  <- " + g.name
			}
		}
	}
	return ""
}

// xrGroups publie les groupes DISJOINTS, plus les deux témoins qui décident : les autres
// rangs identifiés, et les vies SANS identité `i48`. Ce dernier groupe est le plus sévère —
// si des vies sans identité montrent le même taux que la cible, le canal ne distingue rien.
func xrGroups(t *testing.T, spec xrSpec, slotRanks map[uint32][]int, reads, events map[uint32]int) {
	t.Helper()
	rows := make([]xrRow, len(spec.groups))
	var gOther, gNoID xrRow
	all := map[uint32]bool{}
	for sl := range slotRanks {
		all[sl] = true
	}
	for sl := range reads {
		all[sl] = true
	}
	for sl := range all {
		ranks, hasID := slotRanks[sl]
		placed := false
		for gi, g := range spec.groups {
			for _, x := range g.ranks {
				if !placed && eaHasRank(ranks, x) {
					rows[gi].add(reads[sl], events[sl])
					placed = true
				}
			}
		}
		switch {
		case placed:
		case hasID:
			gOther.add(reads[sl], events[sl])
		default:
			gNoID.add(reads[sl], events[sl])
		}
	}
	for gi, g := range spec.groups {
		t.Logf("  GROUPE %-28s : %s", g.name, rows[gi])
	}
	t.Logf("  GROUPE %-28s : %s", "autres rangs identifiés", gOther)
	t.Logf("  GROUPE %-28s : %s", "sans identité i48 (TÉMOIN)", gNoID)
}
