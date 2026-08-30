package main

// gate.go — GATE ORACLE (generalisation) : reconstruire le portage de chaque film depuis le canal
// de score (emplacement des tics) et confronter au temps de portage de l'oracle. Le film n'apporte
// pas la valeur (l'oracle l'a) mais l'INSTANT ; ce gate verifie que le decode tient sur des films
// que la verite terrain n'a jamais vus (recouvrement par joueur + porteur principal).

import (
	"fmt"
	"sort"
	"strings"
)

// recoveryLo / recoveryHi bornent un recouvrement acceptable (>= 80 %, sans survol grossier).
const (
	recoveryLo = 0.80
	recoveryHi = 1.25
)

// gateOracle reconstruit le portage par film et rend le rapport chiffre du gate.
func gateOracle(dumps map[string]*filmDump, films []string, oracle map[string]map[string]oracleStat, ticksKey emplKey) string {
	var b strings.Builder
	fmt.Fprintf(&b, "=== GATE ORACLE : recouvrement du temps de portage (emplacement des tics = comp %d %s) ===\n",
		ticksKey.comp, ticksKey.side)
	b.WriteString("Reconstruction = duree cumulee des trains de tics par joueur (toutes manches).\n\n")
	principalOK, filmsMeasured, allRatios := 0, 0, []float64{}
	for _, id := range films {
		d := dumps[id]
		if d == nil {
			fmt.Fprintf(&b, "FILM %s : NON decode\n\n", id)
			continue
		}
		filmsMeasured++
		ok, ratios := gateFilm(&b, d, oracle[id], ticksKey)
		if ok {
			principalOK++
		}
		allRatios = append(allRatios, ratios...)
	}
	within := 0
	for _, r := range allRatios {
		if r >= recoveryLo && r <= recoveryHi {
			within++
		}
	}
	fmt.Fprintf(&b, "=== BILAN GATE ORACLE ===\n")
	fmt.Fprintf(&b, "porteur principal correct : %d/%d films (seuil >= 3/5)\n", principalOK, filmsMeasured)
	fmt.Fprintf(&b, "recouvrements dans [%.0f%%,%.0f%%] : %d/%d couples (film,joueur) porteurs\n",
		recoveryLo*100, recoveryHi*100, within, len(allRatios))
	return b.String()
}

// gateFilm reconstruit un film : duree de portage par joueur, recouvrement vs oracle, principal.
func gateFilm(b *strings.Builder, d *filmDump, oracleFilm map[string]oracleStat, ticksKey emplKey) (bool, []float64) {
	carry := carrySecondsByXUID(d, ticksKey)
	fmt.Fprintf(b, "FILM %s (tronque=%v) :\n", d.id, d.truncated)
	var ratios []float64
	var recXUID, oraXUID string
	var recMax, oraMax float64
	for _, xuid := range sortedStrKeys(oracleFilm) {
		o := oracleFilm[xuid]
		rec := carry[xuid]
		if rec > recMax {
			recMax, recXUID = rec, xuid
		}
		if o.Time > oraMax {
			oraMax, oraXUID = o.Time, xuid
		}
		if o.Time <= 0 {
			continue
		}
		ratio := rec / o.Time
		ratios = append(ratios, ratio)
		fmt.Fprintf(b, "  %-16s reconstruit=%5.1fs  oracle=%5.1fs  recouvrement=%3.0f%%\n",
			gamertagOf(d, xuid), rec, o.Time, ratio*100)
	}
	principalOK := recXUID == oraXUID && oraXUID != ""
	fmt.Fprintf(b, "  porteur principal : reconstruit=%s (%.1fs) oracle=%s (%.1fs) -> %s\n\n",
		gamertagOf(d, recXUID), recMax, gamertagOf(d, oraXUID), oraMax, okMark(principalOK))
	return principalOK, ratios
}

// carrySecondsByXUID reconstruit la duree de portage (s) par joueur : somme des durees des trains
// de tics sur toutes les manches reelles.
func carrySecondsByXUID(d *filmDump, ticksKey emplKey) map[string]float64 {
	out := map[string]float64{}
	for _, round := range d.rounds {
		for _, iv := range carryIntervals(d, ticksKey, round) {
			out[iv.xuid] += float64(iv.end-iv.start) / 1000
		}
	}
	return out
}

func sortedStrKeys(m map[string]oracleStat) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// abs local (le paquet objectiveevents a le sien, non exporte).
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
