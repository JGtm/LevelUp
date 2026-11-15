//go:build research && cgo

package sync

// performance_objective_simulation_report_research_test.go — le RAPPORT de la simulation : un
// tableau des témoins (une ligne par joueur suivi), puis le résumé de V4 par famille, la
// distribution par chaîne, la note moyenne par rôle et la queue négative. Voir
// performance_objective_simulation_research_test.go.

import (
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/narrative"
)

// simTiers : les paliers de note de l'interface.
var simTiers = []float64{35, 50, 65, 80}

func simTier(v float64) int {
	n := 0
	for _, t := range simTiers {
		if v >= t {
			n++
		}
	}
	return n
}

func simQuantile(xs []float64, q float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	k := float64(len(s)-1) * q
	lo := int(k)
	hi := min(lo+1, len(s)-1)
	return s[lo] + (s[hi]-s[lo])*(k-float64(lo))
}

func simMean(xs []float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	var s float64
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

func simStd(xs []float64) float64 {
	m := simMean(xs)
	var s float64
	for _, x := range xs {
		s += (x - m) * (x - m)
	}
	return math.Sqrt(s / float64(len(xs)))
}

func simFamilyOf(r *simRecord) string {
	if r.Share.Family == "" {
		return "(sans donnée d'objectif)"
	}
	return map[narrative.ObjectiveFamily]string{
		narrative.FamilyCTF: "Drapeau", narrative.FamilyZonesStrongholds: "Bases", narrative.FamilyZonesKOTH: "Colline",
		narrative.FamilyOddball: "Oddball", narrative.FamilyVIP: "VIP", narrative.FamilyStockpile: "Stockpile",
		narrative.FamilyExtraction: "Extraction",
	}[r.Share.Family]
}

func simGroupBy(recs []simRecord, key func(*simRecord) string) ([]string, map[string][]simRecord) {
	g := map[string][]simRecord{}
	for i := range recs {
		k := key(&recs[i])
		g[k] = append(g[k], recs[i])
	}
	keys := make([]string, 0, len(g))
	for k := range g {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, g
}

func simScores(recs []simRecord, variant string) []float64 {
	xs := make([]float64, 0, len(recs))
	for i := range recs {
		if variant == "" {
			xs = append(xs, recs[i].Old.Score)
		} else {
			xs = append(xs, recs[i].By[variant].Score)
		}
	}
	return xs
}

// simWriteReport assemble le rapport complet.
func simWriteReport(t *testing.T, env *simEnv, recs []simRecord) string {
	t.Helper()
	var b strings.Builder
	fmt.Fprintf(&b, "# Simulation de la note à objectif (%d matchs)\n\n", len(recs))
	same, n := 0, 0
	for i := range recs {
		if recs[i].HasStored {
			n++
			if math.Abs(recs[i].Stored-recs[i].Old.Score) <= 0.1001 {
				same++
			}
		}
	}
	fmt.Fprintf(&b, "Contrôle : note actuelle recalculée = note stockée (à 0,1 près) sur %d matchs sur %d.\n\n", same, n)
	simReportWitnesses(&b, env, recs)
	simReportFamilies(&b, recs)
	simReportChains(&b, recs)
	simReportRoles(&b, recs)
	simReportTail(&b, recs)
	return b.String()
}

// simReportFamilies : V4 contre la note actuelle, par famille.
func simReportFamilies(b *strings.Builder, recs []simRecord) {
	keys, g := simGroupBy(recs, func(r *simRecord) string { return simFamilyOf(r) })
	b.WriteString("## V4 par famille de mode (V4 moins note actuelle)\n\n| Famille | Matchs | Écart médian | Écart absolu médian | 90e centile de l'écart absolu | Paliers qui changent |\n|---|---|---|---|---|---|\n")
	for _, k := range append(keys, "Toutes") {
		rs := g[k]
		if k == "Toutes" {
			rs = recs
		}
		var d, ad []float64
		changed := 0
		for i := range rs {
			v := rs[i].By["V4"].Score
			delta := v - rs[i].Old.Score
			d = append(d, delta)
			ad = append(ad, math.Abs(delta))
			if simTier(v) != simTier(rs[i].Old.Score) {
				changed++
			}
		}
		fmt.Fprintf(b, "| %s | %d | %+.1f | %.1f | %.1f | %.0f %% |\n", k, len(rs), simQuantile(d, .5), simQuantile(ad, .5),
			simQuantile(ad, .9), 100*float64(changed)/float64(len(rs)))
	}
	b.WriteString("\n")
}

func simReportChains(b *strings.Builder, recs []simRecord) {
	keys, g := simGroupBy(recs, func(r *simRecord) string { return r.Chain })
	b.WriteString("## Distribution par chaîne (moyenne / écart-type)\n\n| Chaîne | Matchs | Actuelle | V1 | V2 | V3 | V4 |\n|---|---|---|---|---|---|---|\n")
	for _, k := range keys {
		rs := g[k]
		old := simScores(rs, "")
		fmt.Fprintf(b, "| %s | %d | %.1f / %.1f |", k, len(rs), simMean(old), simStd(old))
		for _, v := range []string{"V1", "V2", "V3", "V4"} {
			xs := simScores(rs, v)
			fmt.Fprintf(b, " %.1f / %.1f |", simMean(xs), simStd(xs))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

// simReportRoles : porteurs, escortes et tueurs, note moyenne par variante.
func simReportRoles(b *strings.Builder, recs []simRecord) {
	type role struct {
		label string
		pick  func(*simRecord) bool
	}
	roles := []role{
		{"Porteurs (part >= 35 % et <= 8 frags)", func(r *simRecord) bool { return r.Share.Share >= .35 && r.Kills <= 8 }},
		{"Escortes (>= 60 % des points en soutien, >= 4 points)", func(r *simRecord) bool {
			return r.Share.Points >= 4 && r.Share.SupportPoints/r.Share.Points >= .6
		}},
		{"Tueurs (part <= 10 % et >= 15 frags)", func(r *simRecord) bool { return r.Share.Share <= .10 && r.Kills >= 15 }},
	}
	b.WriteString("## Note moyenne par rôle\n\n| Rôle | Matchs | Actuelle | V1 | V2 | V3 | V4 |\n|---|---|---|---|---|---|---|\n")
	for _, ro := range roles {
		var rs []simRecord
		for i := range recs {
			if recs[i].Share.Family != "" && ro.pick(&recs[i]) {
				rs = append(rs, recs[i])
			}
		}
		fmt.Fprintf(b, "| %s | %d | %.1f |", ro.label, len(rs), simMean(simScores(rs, "")))
		for _, v := range []string{"V1", "V2", "V3", "V4"} {
			fmt.Fprintf(b, " %.1f |", simMean(simScores(rs, v)))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

// simReportTail chiffre la queue négative de V4 (écart V4 moins actuelle), par famille.
func simReportTail(b *strings.Builder, recs []simRecord) {
	keys, g := simGroupBy(recs, func(r *simRecord) string { return simFamilyOf(r) })
	b.WriteString("## Queue négative de V4 (V4 moins note actuelle)\n\n| Famille | Pire écart | Matchs sous -8 | Matchs sous -5 |\n|---|---|---|---|\n")
	for _, k := range keys {
		worst, u8, u5 := 0.0, 0, 0
		for i := range g[k] {
			d := g[k][i].By["V4"].Score - g[k][i].Old.Score
			worst = math.Min(worst, d)
			if d < -8 {
				u8++
			}
			if d < -5 {
				u5++
			}
		}
		fmt.Fprintf(b, "| %s | %.1f | %d sur %d | %d |\n", k, worst, u8, len(g[k]), u5)
	}
	b.WriteString("\n")
}

// simWitnessIDs : les témoins, par préfixe d'identifiant de match (l'identifiant ne sort pas du
// rapport : date, carte et mode).
var simWitnessIDs = []string{"c349da69", "3fa60771", "2acc306f", "b8c1b220", "4a93f0e2", "d4230304", "b5d130ad", "cae8471d", "7665e832", "51ebbc0f"}

// simReportWitnesses : UN tableau, une ligne par joueur suivi de chaque témoin.
func simReportWitnesses(b *strings.Builder, env *simEnv, recs []simRecord) {
	b.WriteString("## Témoins\n\n| Match | Joueur (frags, part de l'objectif, part du soutien) | Actuelle | V1 | V2 | V3 | V4 |\n|---|---|---|---|---|---|---|\n")
	byID := map[string][]*simRecord{}
	for i := range recs {
		byID[recs[i].MatchID] = append(byID[recs[i].MatchID], &recs[i])
	}
	for _, short := range simWitnessIDs {
		var mid string
		for id := range byID {
			if strings.HasPrefix(id, short) {
				mid = id
			}
		}
		if mid == "" {
			continue
		}
		mine := byID[mid]
		sort.Slice(mine, func(i, j int) bool { return mine[i].Share.Share > mine[j].Share.Share })
		label := simMatchLabel(env, mine[0])
		for _, r := range mine {
			fmt.Fprintf(b, "| %s | %s (%.0f, %.0f %%, %.0f %%) | %.1f | %.1f | %.1f | %.1f | %.1f |\n", label, r.Player, r.Kills,
				100*r.Share.Share, 100*r.Share.SupportShare, r.Old.Score, r.By["V1"].Score, r.By["V2"].Score, r.By["V3"].Score, r.By["V4"].Score)
		}
	}
	b.WriteString("\n")
}

// simMatchLabel : date, carte, mode et score (du point de vue du camp du joueur).
func simMatchLabel(env *simEnv, r *simRecord) string {
	score := ""
	var t0, t1 sql.NullInt64
	if err := env.shared.QueryRow(`SELECT team_0_score, team_1_score FROM match_registry WHERE match_id = ?`, r.MatchID).Scan(&t0, &t1); err == nil {
		a, c := t0.Int64, t1.Int64
		if r.Team == 1 {
			a, c = c, a
		}
		score = fmt.Sprintf(", %d-%d", a, c)
	}
	return fmt.Sprintf("%s, %s, %s%s", r.Start[:10], r.Map, r.Pair, score)
}
