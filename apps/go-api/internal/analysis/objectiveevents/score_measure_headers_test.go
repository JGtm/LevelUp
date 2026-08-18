package objectiveevents

import (
	"fmt"
	"sort"
)

// score_measure_headers_test.go — l'hypothese de la phase 0-ter : les deux en-tetes de 5 bits d'un
// compteur statborg portent-ils le NUMERO DE MANCHE ?
//
// Ce que la mesure doit trancher, et par quoi :
//
//	positif  sur un match a plusieurs manches, des enregistrements a en-tete != 0 apparaissent
//	         APRES la fin de la premiere manche, et la somme des manches (une valeur finale par
//	         en-tete) egale l'oracle ;
//	negatif  sur un match a UNE manche, aucun enregistrement a en-tete != 0 — sinon l'en-tete est
//	         autre chose (ou pas seulement une manche), et il faut l'ecrire.
//
// Le controle negatif compte autant que le positif : un en-tete relache qui ferait apparaitre des
// « manches » sur un match a une seule manche signalerait de simples faux positifs.

// writeExtHeaders publie la distribution des deux en-tetes d'un compteur, globalement puis par
// slot et par en-tete (avec les bornes temporelles et la valeur finale du segment).
func writeExtHeaders(m *measureRows, recs []statRecordExt, comp int) {
	pairs := map[string]int{}
	for _, r := range recs {
		if v, ok := r.Cur[comp]; ok {
			pairs[fmt.Sprintf("h1=%d,h2=%d", v.H1, v.H2)]++
		}
	}
	for _, k := range sortedStrKeys(pairs) {
		m.row("hdr", comp, k, pairs[k])
	}
	for _, side := range []bool{false, true} {
		for _, g := range headerGroups(recs, comp, side) {
			m.row("hdr_slot", comp, headerName(side), g.Slot, g.Header,
				g.Count, g.FirstMS, g.LastMS, g.Last)
		}
	}
}

// headerGroup est la suite des emissions d'un compteur pour un slot et une valeur d'en-tete.
type headerGroup struct {
	Slot, Header, Count int
	FirstMS, LastMS     int
	Last                int64
	Kept                int
}

// headerGroups regroupe les emissions par (slot, en-tete) et filtre chaque groupe par la meme
// plus longue suite croissante que la production. `useH2` choisit le second en-tete.
func headerGroups(recs []statRecordExt, comp int, useH2 bool) []headerGroup {
	type key struct{ slot, hdr int }
	series := map[key][]ScorePoint{}
	for _, r := range recs {
		v, ok := r.Cur[comp]
		if !ok || v.A < 0 {
			continue
		}
		h := v.H1
		if useH2 {
			h = v.H2
		}
		k := key{r.Slot, h}
		series[k] = append(series[k], ScorePoint{TimeMS: r.TimeMS, Slot: r.Slot, Value: v.A})
	}
	var out []headerGroup
	for k, pts := range series {
		sort.SliceStable(pts, func(i, j int) bool { return pts[i].TimeMS < pts[j].TimeMS })
		kept := longestRun(pts, true)
		if len(kept) == 0 {
			continue
		}
		out = append(out, headerGroup{Slot: k.slot, Header: k.hdr, Count: len(pts),
			FirstMS: kept[0].TimeMS, LastMS: kept[len(kept)-1].TimeMS,
			Last: kept[len(kept)-1].Value, Kept: len(kept)})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Slot != out[j].Slot {
			return out[i].Slot < out[j].Slot
		}
		return out[i].Header < out[j].Header
	})
	return out
}

// headerTotals somme, par slot, la valeur finale de chaque en-tete — la lecture « un en-tete = une
// manche ». Un groupe trop court pour etre une manche est ignore ([roundSegmentMin]).
func headerTotals(recs []statRecordExt, comp int, useH2 bool) map[int]int64 {
	out := map[int]int64{}
	for _, g := range headerGroups(recs, comp, useH2) {
		if g.Kept < roundSegmentMin {
			continue
		}
		out[g.Slot] += g.Last
	}
	return out
}

// writeExtHeaderVerdict confronte les deux lectures possibles (en-tete 1 ou en-tete 2 = manche) a
// l'oracle, pour le score d'equipe et pour la somme des frags des joueurs.
func writeExtHeaderVerdict(m *measureRows, recs []statRecordExt, or oracleMatch) {
	for _, side := range []bool{false, true} {
		tot := headerTotals(recs, modeScoreComp, side)
		m.row("hdr_score", headerName(side), verdictPair(tot[6], tot[8], or.Team0, or.Team1),
			fmt.Sprintf("%d/%d", tot[6], tot[8]), fmt.Sprintf("%d/%d", or.Team0, or.Team1))

		kills := headerTotals(recs, coreKillsComp, side)
		var sum int64
		for slot, v := range kills {
			if !IsTeamSlot(slot) {
				sum += v
			}
		}
		var oracleSum int64
		for _, l := range or.Lines {
			oracleSum += int64(l.Kills)
		}
		verdict := "ecart"
		if sum == oracleSum {
			verdict = "exact"
		}
		m.row("hdr_frags", headerName(side), verdict, sum, oracleSum)
	}
}

// writeExtSerieH publie la serie du score d'equipe AVEC ses en-tetes : c'est elle qui dit si une
// valeur est atteinte par une rampe d'increments ou par un saut isole (item 0t.2).
func writeExtSerieH(m *measureRows, recs []statRecordExt) {
	type pt struct {
		t      int
		v      int64
		h1, h2 int
	}
	bySlot := map[int][]pt{}
	for _, r := range recs {
		if !IsTeamSlot(r.Slot) {
			continue
		}
		if v, ok := r.Cur[modeScoreComp]; ok && v.A >= 0 {
			bySlot[r.Slot] = append(bySlot[r.Slot], pt{r.TimeMS, v.A, v.H1, v.H2})
		}
	}
	for _, slot := range sortedSlots(bySlot) {
		pts := bySlot[slot]
		sort.SliceStable(pts, func(i, j int) bool { return pts[i].t < pts[j].t })
		for _, p := range pts {
			m.row("serieh", slot, p.t, p.v, p.h1, p.h2)
		}
	}
}

// verdictPair dit si le couple du film egale celui de l'oracle, dans un ordre ou dans l'autre.
func verdictPair(a, b int64, t0, t1 int) string {
	film := []int64{a, b}
	oracle := []int64{int64(t0), int64(t1)}
	sort.Slice(film, func(i, j int) bool { return film[i] < film[j] })
	sort.Slice(oracle, func(i, j int) bool { return oracle[i] < oracle[j] })
	if film[0] == oracle[0] && film[1] == oracle[1] {
		return "exact"
	}
	return "ecart"
}

func headerName(useH2 bool) string {
	if useH2 {
		return "h2"
	}
	return "h1"
}

func sortedStrKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
