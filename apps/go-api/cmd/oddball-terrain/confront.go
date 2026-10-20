package main

// confront.go — LE GATE CENTRAL : le canal de score reproduit-il la sequence des porteurs de la
// manche 1 de d9781168 ? On identifie d'abord les emplacements Oddball par l'oracle (general),
// puis on aligne l'horloge sur les prises observees et on confronte porteur par porteur.

import (
	"fmt"
	"sort"
	"strings"
)

// tickGapMS : au-dela de ce trou entre deux tics de score d'un meme joueur, on ferme la periode
// de portage. Les tics tombent ~1/s pendant le portage ; trois secondes separent nettement deux
// periodes distinctes du meme joueur sans couper une periode continue.
const tickGapMS = 3000

// alignTolMS : tolerance d'appariement entre une prise decodee et une prise terrain.
const alignTolMS = 5000

// gt est une prise de la verite terrain : instant de lecture (s) et joueur.
type gt struct {
	sec    int
	player string
}

// terrainGrabsM1 : la sequence FIGEE des prises de la manche 1 (cf. TERRAIN_PROTOCOLE.md §1).
var terrainGrabsM1 = []gt{
	{48, "SHROOM GOD3261"}, {65, "scuderiasven"}, {124, "LadyJezz"}, {130, "L0UDEN13"},
	{155, "scuderiasven"}, {160, "LadyJezz"}, {180, "L0UDEN13"}, {189, "DinoR00"},
	{219, "SHROOM GOD3261"},
}

// carryInterval est une periode de portage reconstruite : un train de tics d'un meme slot.
type carryInterval struct {
	slot       int
	xuid       string
	start, end int
	ticks      int
}

// identifyEmplacements retient l'emplacement `skull_grabs` et l'emplacement `skull_scoring_ticks`
// par confrontation A L'ORACLE, films confondus. Rend les cles et un rapport.
func identifyEmplacements(dumps map[string]*filmDump, oracle map[string]map[string]oracleStat, films []string) (emplKey, emplKey, string) {
	var b strings.Builder
	b.WriteString("=== IDENTIFICATION DES EMPLACEMENTS ODDBALL (par l'oracle, films confondus) ===\n")
	grabsKey, gr := bestEmplacement(dumps, oracle, films, func(o oracleStat) int { return o.Grabs })
	ticksKey, tr := bestEmplacement(dumps, oracle, films, func(o oracleStat) int { return o.Ticks })
	fmt.Fprintf(&b, "skull_grabs           -> comp %d cote %s\n%s", grabsKey.comp, grabsKey.side, gr)
	fmt.Fprintf(&b, "skull_scoring_ticks   -> comp %d cote %s\n%s", ticksKey.comp, ticksKey.side, tr)
	return grabsKey, ticksKey, b.String()
}

// bestEmplacement classe les emplacements par accord exact avec la colonne d'oracle `col`,
// priorite aux joueurs dont l'oracle est NON NUL (un emplacement toujours nul matcherait tous les
// zeros). Rend la meilleure cle et le detail des cinq premieres.
func bestEmplacement(dumps map[string]*filmDump, oracle map[string]map[string]oracleStat, films []string, col func(oracleStat) int) (emplKey, string) {
	type score struct {
		key               emplKey
		matchNZ, matchAll int
		filmsNZ           int
	}
	agg := map[emplKey]*score{}
	for comp := 0; comp <= sweepMaxComp; comp++ {
		for _, side := range []string{"A", "B"} {
			key := emplKey{comp: comp, side: side}
			s := &score{key: key}
			for _, id := range films {
				d := dumps[id]
				if d == nil {
					continue
				}
				got := emplTotalByXUID(d, key)
				filmHitNZ := false
				for xuid, ov := range oracle[id] {
					want := col(ov)
					if got[xuid] == want {
						s.matchAll++
						if want > 0 {
							s.matchNZ++
							filmHitNZ = true
						}
					}
				}
				if filmHitNZ {
					s.filmsNZ++
				}
			}
			agg[key] = s
		}
	}
	ranked := make([]*score, 0, len(agg))
	for _, s := range agg {
		ranked = append(ranked, s)
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].matchNZ != ranked[j].matchNZ {
			return ranked[i].matchNZ > ranked[j].matchNZ
		}
		return ranked[i].matchAll > ranked[j].matchAll
	})
	var b strings.Builder
	for i := 0; i < len(ranked) && i < 5; i++ {
		s := ranked[i]
		fmt.Fprintf(&b, "    comp %2d %s : accords non-nuls=%d, accords totaux=%d, films non-nuls=%d\n",
			s.key.comp, s.key.side, s.matchNZ, s.matchAll, s.filmsNZ)
	}
	return ranked[0].key, b.String()
}

// emplTotalByXUID rend le total de match (somme des manches) d'un emplacement, par xuid. La somme
// se fait PAR MANCHE avec l'identite de la manche : le slot etant reattribue d'une manche a
// l'autre, un meme slot peut porter deux joueurs sur le match.
func emplTotalByXUID(d *filmDump, key emplKey) map[string]int {
	out := map[string]int{}
	for slot, byRound := range d.series {
		for round, empls := range byRound {
			xuid := identOf(d, round, slot)
			if xuid == "" {
				continue
			}
			if pts := empls[key]; len(pts) > 0 {
				out[xuid] += int(pts[len(pts)-1].v)
			}
		}
	}
	return out
}

// confrontTerrain aligne l'horloge et confronte les porteurs de la manche 1. Rend le contenu de
// TERRAIN_scores.log (traces brutes), de TERRAIN_confrontation.log (le detail intervalle par
// intervalle), et un resume verdict imprime a l'ecran.
func confrontTerrain(td *filmDump, oracleFilm map[string]oracleStat, grabsKey, ticksKey emplKey) (string, string, string) {
	if td == nil {
		msg := "VERDICT GATE 3 : d9781168 NON decode — confrontation impossible.\n"
		return msg, msg, msg
	}
	const round0 = 0
	tickBySlot := instantsBySlot(td, ticksKey, round0)
	intervals := carryIntervals(td, ticksKey, round0)
	scoresLog := terrainScoresLog(td, oracleFilm, grabsKey, ticksKey, round0, intervals)

	offset, alignHits := bestOffset(td, intervals)
	confLog, a, b := confrontIntervals(td, intervals, tickBySlot, offset)
	verdict := fmt.Sprintf(
		"VERDICT GATE 3 (manche 1 de d9781168) : offset*=%+.1f s (aligne %d/%d prises) ; "+
			"prises reproduites a*=%d/9 ; porteurs d'intervalle b*=%d/9 ; gate=min(a,b)=%d/9 (seuil 8/9)\n",
		float64(offset)/1000, alignHits, len(terrainGrabsM1), a, b, min(a, b))
	return scoresLog, verdict + "\n" + confLog, verdict
}

// carryIntervals reconstruit les periodes de portage d'une manche : trains de tics par slot.
func carryIntervals(td *filmDump, ticksKey emplKey, round int) []carryInterval {
	var out []carryInterval
	for slot, byRound := range td.series {
		pts := byRound[round][ticksKey]
		if len(pts) == 0 {
			continue
		}
		inst := incrementInstants(pts)
		if len(inst) == 0 {
			continue
		}
		xuid := identOf(td, round, slot)
		start, last, n := inst[0], inst[0], 1
		for _, t := range inst[1:] {
			if t-last > tickGapMS {
				out = append(out, carryInterval{slot: slot, xuid: xuid, start: start, end: last, ticks: n})
				start, n = t, 0
			}
			last, n = t, n+1
		}
		out = append(out, carryInterval{slot: slot, xuid: xuid, start: start, end: last, ticks: n})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].start < out[j].start })
	return out
}

// incrementInstants rend un instant par UNITE gagnee par le compteur (les tics de score).
func incrementInstants(pts []point) []int {
	var out []int
	prev := int64(0)
	for _, p := range pts {
		for ; prev < p.v; prev++ {
			out = append(out, p.t)
		}
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
