package main

// confront2.go — l'alignement d'horloge (offset unique) et le detail intervalle par intervalle,
// plus la trace brute du signal de score de la manche 1.

import (
	"fmt"
	"sort"
	"strings"
)

// instantsBySlot rend, par slot de joueur, les instants (ms) ou l'emplacement s'est incremente
// dans la manche `round`.
func instantsBySlot(td *filmDump, key emplKey, round int) map[int][]int {
	out := map[int][]int{}
	for slot, byRound := range td.series {
		if pts := byRound[round][key]; len(pts) > 0 {
			out[slot] = incrementInstants(pts)
		}
	}
	return out
}

// gamertagOf rend le gamertag lu dans le film pour le xuid d'un slot, ou le xuid a defaut.
func gamertagOf(td *filmDump, xuid string) string {
	if xuid == "" {
		return "?"
	}
	if gt, ok := td.gamertags[atou(xuid)]; ok {
		return gt
	}
	return xuid
}

// normName normalise un gamertag pour l'appariement (minuscules, sans espaces).
func normName(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, " ", ""))
}

// bestOffset balaye l'offset d'horloge et retient celui qui aligne le PLUS de prises terrain sur
// un debut de periode de portage reconstruite DU MEME joueur (offset unique pour la manche).
func bestOffset(td *filmDump, intervals []carryInterval) (int, int) {
	bestOff, bestHits := 0, -1
	for off := -60000; off <= 60000; off += 500 {
		hits := 0
		for _, g := range terrainGrabsM1 {
			target := g.sec*1000 + off
			if matchesGrab(td, intervals, g.player, target) {
				hits++
			}
		}
		if hits > bestHits {
			bestHits, bestOff = hits, off
		}
	}
	return bestOff, bestHits
}

// matchesGrab dit si une periode de portage du joueur `player` DEBUTE pres de `targetMS`.
func matchesGrab(td *filmDump, intervals []carryInterval, player string, targetMS int) bool {
	want := normName(player)
	for _, iv := range intervals {
		if normName(gamertagOf(td, iv.xuid)) != want {
			continue
		}
		if abs(iv.start-targetMS) <= alignTolMS {
			return true
		}
	}
	return false
}

// confrontIntervals confronte, apres alignement, les prises et les porteurs d'intervalle. Rend le
// detail lisible et les deux comptes a (prises) et b (porteurs).
func confrontIntervals(td *filmDump, intervals []carryInterval, tickBySlot map[int][]int, offset int) (string, int, int) {
	var bb strings.Builder
	bb.WriteString("=== CONFRONTATION MANCHE 1 (d9781168), intervalle par intervalle ===\n")
	fmt.Fprintf(&bb, "offset d'horloge applique : %+d ms (film = terrain + offset)\n\n", offset)
	a, b := 0, 0
	for i, g := range terrainGrabsM1 {
		endSec := roundEndSec
		if i+1 < len(terrainGrabsM1) {
			endSec = terrainGrabsM1[i+1].sec
		}
		fromMS, toMS := g.sec*1000+offset, endSec*1000+offset
		grabOK := matchesGrab(td, intervals, g.player, fromMS)
		carrier, ticks := dominantCarrier(td, tickBySlot, fromMS, toMS)
		carrierOK := normName(carrier) == normName(g.player)
		if grabOK {
			a++
		}
		if carrierOK {
			b++
		}
		fmt.Fprintf(&bb, "[%3d-%3ds] terrain=%-16s | prise decodee=%-5v | porteur reconstruit=%-16s (%d tics) | prise=%s porteur=%s\n",
			g.sec, endSec, g.player, grabOK, carrier, ticks, okMark(grabOK), okMark(carrierOK))
	}
	fmt.Fprintf(&bb, "\nBILAN : prises a=%d/9, porteurs b=%d/9, gate=min=%d/9 (seuil 8/9)\n", a, b, min(a, b))
	return bb.String(), a, b
}

// roundEndSec borne le dernier intervalle de la manche 1 (large : la fin de manche ferme de toute
// facon les trains de tics).
const roundEndSec = 400

// dominantCarrier rend le joueur qui recoit le PLUS de tics dans la fenetre [fromMS,toMS), et son
// compte. C'est le porteur reconstruit de l'intervalle.
func dominantCarrier(td *filmDump, tickBySlot map[int][]int, fromMS, toMS int) (string, int) {
	best, bestSlot := 0, -1
	for slot, inst := range tickBySlot {
		n := 0
		for _, t := range inst {
			if t >= fromMS && t < toMS {
				n++
			}
		}
		if n > best {
			best, bestSlot = n, slot
		}
	}
	if bestSlot < 0 {
		return "-", 0
	}
	// Confrontation manche 1 : l'identite est celle de la manche 0 (slot reattribue par manche).
	return gamertagOf(td, identOf(td, 0, bestSlot)), best
}

func okMark(ok bool) string {
	if ok {
		return "OK"
	}
	return "X"
}

// terrainScoresLog trace le signal brut de la manche 1 : roster identifie + oracle, puis, par
// joueur, le score personnel, les prises et les periodes de portage reconstruites.
func terrainScoresLog(td *filmDump, oracleFilm map[string]oracleStat, grabsKey, ticksKey emplKey, round int, intervals []carryInterval) string {
	var b strings.Builder
	b.WriteString("=== TERRAIN_scores : signal brut de d9781168 (manche 1 = round 0) ===\n")
	fmt.Fprintf(&b, "manches reelles : %v ; enregistrements=%d ; tronque=%v\n\n", td.rounds, td.nrec, td.truncated)
	b.WriteString("ROSTER manche 1 (slot -> xuid -> gamertag) + oracle MATCH (temps portage s / grabs / tics) :\n")
	for _, slot := range sortedKeys(td.ident[round]) {
		xuid := td.ident[round][slot]
		o := oracleFilm[xuid]
		fmt.Fprintf(&b, "  slot %2d  %-18s  %-16s  oracle: %.1fs / %dgrabs / %dtics\n",
			slot, xuid, gamertagOf(td, xuid), o.Time, o.Grabs, o.Ticks)
	}
	b.WriteString("\nPERIODES DE PORTAGE reconstruites (manche 1, par train de tics) :\n")
	for _, iv := range intervals {
		fmt.Fprintf(&b, "  [%6d-%6d ms] %-16s  %d tics\n",
			iv.start, iv.end, gamertagOf(td, iv.xuid), iv.ticks)
	}
	b.WriteString("\nPRISES decodees (emplacement skull_grabs), manche 1 :\n")
	writeInstants(&b, td, instantsBySlot(td, grabsKey, round))
	b.WriteString("\nSCORE PERSONNEL (comp 1 B), sauts de la manche 1 :\n")
	writePersonalJumps(&b, td, round)
	return b.String()
}

// writeInstants imprime, par joueur (identite manche 1), les instants d'un emplacement.
func writeInstants(b *strings.Builder, td *filmDump, bySlot map[int][]int) {
	for _, slot := range sortedIntKeys(bySlot) {
		fmt.Fprintf(b, "  %-16s : %v\n", gamertagOf(td, identOf(td, 0, slot)), bySlot[slot])
	}
}

// writePersonalJumps imprime, par joueur, les sauts (delta) du score personnel de la manche.
func writePersonalJumps(b *strings.Builder, td *filmDump, round int) {
	key := emplKey{comp: 1, side: "B"}
	slots := make([]int, 0, len(td.series))
	for s := range td.series {
		slots = append(slots, s)
	}
	sort.Ints(slots)
	for _, slot := range slots {
		pts := td.series[slot][round][key]
		if len(pts) == 0 {
			continue
		}
		var jumps []string
		prev := int64(0)
		for _, p := range pts {
			if d := p.v - prev; d != 0 {
				jumps = append(jumps, fmt.Sprintf("%d:+%d", p.t, d))
			}
			prev = p.v
		}
		fmt.Fprintf(b, "  %-16s : %s\n", gamertagOf(td, identOf(td, 0, slot)), strings.Join(jumps, " "))
	}
}

func sortedIntKeys(m map[int][]int) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}
