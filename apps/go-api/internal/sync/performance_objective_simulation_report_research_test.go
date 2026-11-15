//go:build research && cgo

package sync

// performance_objective_simulation_report_research_test.go — le RAPPORT de la simulation : les
// agrégations (écarts par famille, paliers, distributions par chaîne, vérifications, témoins)
// en tableaux Markdown. Voir performance_objective_simulation_research_test.go.

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

// simStat résume les écarts d'un réglage contre la note actuelle sur un groupe de matchs.
func simStat(recs []simRecord, variant string) string {
	var d, ad []float64
	changed := 0
	for i := range recs {
		v := recs[i].By[variant].Score
		delta := v - recs[i].Old.Score
		d = append(d, delta)
		ad = append(ad, math.Abs(delta))
		if simTier(v) != simTier(recs[i].Old.Score) {
			changed++
		}
	}
	if len(d) == 0 {
		return "-"
	}
	return fmt.Sprintf("méd. %+.1f ; |écart| méd. %.1f, p90 %.1f ; paliers %.0f %%",
		simQuantile(d, .5), simQuantile(ad, .5), simQuantile(ad, .9), 100*float64(changed)/float64(len(d)))
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

// simWriteReport assemble le rapport complet.
func simWriteReport(t *testing.T, env *simEnv, recs []simRecord) string {
	t.Helper()
	var b strings.Builder
	names := []string{"base", "soutien x1,5", "soutien x2", "part du soutien"}
	fmt.Fprintf(&b, "# Simulation de la note à objectif (%d matchs)\n\n", len(recs))
	simReportControl(&b, recs)
	simReportFamilies(&b, recs, names)
	simReportChains(&b, recs, append([]string{"base à 229"}, names...))
	simReportChecks(&b, recs)
	simReportRoles(&b, recs, names)
	simReportHP(&b, recs)
	simReportWitnesses(t, &b, env, recs, names)
	return b.String()
}

func simReportControl(b *strings.Builder, recs []simRecord) {
	same := 0
	n := 0
	for i := range recs {
		if recs[i].HasStored {
			n++
			if math.Abs(recs[i].Stored-recs[i].Old.Score) <= 0.1001 {
				same++
			}
		}
	}
	fmt.Fprintf(b, "## Contrôle\n\nNote actuelle recalculée = note stockée (à 0,1 près) : %d matchs sur %d.\n\n", same, n)
}

func simReportFamilies(b *strings.Builder, recs []simRecord, names []string) {
	keys, g := simGroupBy(recs, func(r *simRecord) string { return simFamilyOf(r) })
	fmt.Fprintf(b, "## Écart par famille de mode (réglage moins note actuelle)\n\n| Famille | Matchs |")
	for _, n := range names {
		fmt.Fprintf(b, " %s |", n)
	}
	b.WriteString("\n|---|---|")
	for range names {
		b.WriteString("---|")
	}
	b.WriteString("\n")
	for _, k := range append(keys, "Toutes") {
		rs := g[k]
		if k == "Toutes" {
			rs = recs
		}
		fmt.Fprintf(b, "| %s | %d |", k, len(rs))
		for _, n := range names {
			fmt.Fprintf(b, " %s |", simStat(rs, n))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

func simReportChains(b *strings.Builder, recs []simRecord, names []string) {
	keys, g := simGroupBy(recs, func(r *simRecord) string { return r.Chain })
	b.WriteString("## Distribution par chaîne (moyenne / écart-type)\n\n| Chaîne | Matchs | Actuelle |")
	for _, n := range names {
		fmt.Fprintf(b, " %s |", n)
	}
	b.WriteString("\n|---|---|---|")
	for range names {
		b.WriteString("---|")
	}
	b.WriteString("\n")
	for _, k := range keys {
		rs := g[k]
		var old []float64
		for i := range rs {
			old = append(old, rs[i].Old.Score)
		}
		fmt.Fprintf(b, "| %s | %d | %.1f / %.1f |", k, len(rs), simMean(old), simStd(old))
		for _, n := range names {
			var xs []float64
			for i := range rs {
				xs = append(xs, rs[i].By[n].Score)
			}
			fmt.Fprintf(b, " %.1f / %.1f |", simMean(xs), simStd(xs))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

func simReportChecks(b *strings.Builder, recs []simRecord) {
	b.WriteString("## Vérifications\n\n")
	for _, n := range []string{"base", "soutien x1,5", "soutien x2", "part du soutien"} {
		under, zero, zeroFloor, withObj := 0, 0, 0, 0
		var maxGap float64
		for i := range recs {
			o := recs[i].By[n]
			if o.Score < roundPerfScore(o.Combat)-1e-9 {
				under++
			}
			if !o.Has {
				continue
			}
			withObj++
			if o.Part == 0 {
				zero++
				if math.Abs(o.Score-roundPerfScore(o.Combat)) < 1e-9 {
					zeroFloor++
				} else {
					maxGap = math.Max(maxGap, o.Score-roundPerfScore(o.Combat))
				}
			}
		}
		fmt.Fprintf(b, "- **%s** : note sous le combat seul : %d matchs sur %d ; part nulle : %d matchs, dont %d à la note du combat seul (le reste dépasse le combat de %.1f point au plus).\n",
			n, under, withObj, zero, zeroFloor, maxGap)
	}
	b.WriteString("\n")
}

// simReportRoles : porteurs, escortes et tueurs, notes moyennes par réglage.
func simReportRoles(b *strings.Builder, recs []simRecord, names []string) {
	type role struct {
		label string
		pick  func(*simRecord) bool
	}
	roles := []role{
		{"porteurs (part >= 35 % et <= 8 frags)", func(r *simRecord) bool { return r.Share.Share >= .35 && r.Kills <= 8 }},
		{"escortes (>= 60 % des points en soutien, >= 4 points)", func(r *simRecord) bool {
			return r.Share.Points >= 4 && r.Share.SupportPoints/r.Share.Points >= .6
		}},
		{"tueurs (part <= 10 % et >= 15 frags)", func(r *simRecord) bool { return r.Share.Share <= .10 && r.Kills >= 15 }},
	}
	b.WriteString("## Rôles : note moyenne avant / après\n\n| Rôle | Matchs | Actuelle |")
	for _, n := range names {
		fmt.Fprintf(b, " %s |", n)
	}
	b.WriteString("\n|---|---|---|")
	for range names {
		b.WriteString("---|")
	}
	b.WriteString("\n")
	for _, ro := range roles {
		var rs []simRecord
		for i := range recs {
			if recs[i].Share.Family != "" && ro.pick(&recs[i]) {
				rs = append(rs, recs[i])
			}
		}
		var old []float64
		for i := range rs {
			old = append(old, rs[i].Old.Score)
		}
		fmt.Fprintf(b, "| %s | %d | %.1f |", ro.label, len(rs), simMean(old))
		for _, n := range names {
			var xs []float64
			for i := range rs {
				xs = append(xs, rs[i].By[n].Score)
			}
			fmt.Fprintf(b, " %.1f |", simMean(xs))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

// simReportHP : l'effet de 229 points de vie au lieu de 225 sur la note relative.
func simReportHP(b *strings.Builder, recs []simRecord) {
	diffOld, diffBase := 0, 0
	var maxOld, maxBase float64
	for i := range recs {
		d := math.Abs(recs[i].Old229.Score - recs[i].Old.Score)
		if d > 1e-9 {
			diffOld++
		}
		maxOld = math.Max(maxOld, d)
		db := math.Abs(recs[i].By["base à 229"].Score - recs[i].By["base"].Score)
		if db > 1e-9 {
			diffBase++
		}
		maxBase = math.Max(maxBase, db)
	}
	fmt.Fprintf(b, "## 229 points de vie au lieu de 225\n\nNote actuelle : %d matchs changent sur %d (écart max %.2f). Note à objectif (base) : %d matchs changent sur %d (écart max %.2f).\n\n",
		diffOld, len(recs), maxOld, diffBase, len(recs), maxBase)
}

// simWitnessIDs : les témoins, par préfixe d'identifiant de match (l'identifiant ne sort pas du
// rapport : date, carte et mode).
var simWitnessIDs = []string{"c349da69", "3fa60771", "2acc306f", "b8c1b220", "4a93f0e2", "d4230304", "b5d130ad", "cae8471d", "7665e832", "51ebbc0f"}

// simActionLabels : colonnes d'action montrées dans la table des témoins.
var simActionLabels = map[narrative.ObjectiveFamily][][2]string{
	narrative.FamilyCTF: {{"flag_captures", "captures"}, {"flag_steals", "vols"}, {"flag_capture_assists", "passes de capture"},
		{"flag_returns", "retours"}, {"flag_carriers_killed", "porteurs abattus"}, {"flag_returners_killed", "retourneurs abattus"}, {"flag_secures", "sécurisations"}},
	narrative.FamilyZonesKOTH:        {{"zone_captures", "captures de zone"}, {"zone_secures", "sécurisations"}, {"zone_offensive_kills", "frags offensifs"}, {"zone_defensive_kills", "frags défensifs"}},
	narrative.FamilyZonesStrongholds: {{"zone_captures", "captures de base"}, {"zone_secures", "sécurisations"}, {"zone_offensive_kills", "frags offensifs"}, {"zone_defensive_kills", "frags défensifs"}},
	narrative.FamilyOddball:          {{"skull_grabs", "prises du crâne"}, {"skull_carriers_killed", "porteurs abattus"}, {"skull_scoring_ticks", "points de tenue"}},
}

func simReportWitnesses(t *testing.T, b *strings.Builder, env *simEnv, recs []simRecord, names []string) {
	b.WriteString("## Témoins (note actuelle puis réglages)\n\n")
	byID := map[string]map[string]*simRecord{}
	for i := range recs {
		if byID[recs[i].MatchID] == nil {
			byID[recs[i].MatchID] = map[string]*simRecord{}
		}
		byID[recs[i].MatchID][recs[i].Player] = &recs[i]
	}
	tracked := map[string]string{}
	for gt, x := range env.xuids {
		tracked[x] = gt
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
		simWriteWitness(t, b, env, mid, byID[mid], tracked, names)
	}
}

func simWriteWitness(t *testing.T, b *strings.Builder, env *simEnv, mid string, mine map[string]*simRecord, tracked map[string]string, names []string) {
	var any *simRecord
	for _, r := range mine {
		any = r
		break
	}
	fam := any.Share.Family
	var rowsAll []narrative.ObjectiveTeamRow
	for gt := range mine {
		rowsAll = env.objRows[gt][mid]
		break
	}
	kills := map[string]int{}
	if rs, err := env.shared.Query(`SELECT xuid, COALESCE(kills, 0) FROM match_participants WHERE match_id = ?`, mid); err == nil {
		for rs.Next() {
			var x string
			var k int
			if rs.Scan(&x, &k) == nil {
				kills[x] = k
			}
		}
		_ = rs.Close()
	}
	score := ""
	var t0, t1 sql.NullInt64
	if err := env.shared.QueryRow(`SELECT team_0_score, team_1_score FROM match_registry WHERE match_id = ?`, mid).Scan(&t0, &t1); err == nil {
		a, c := t0.Int64, t1.Int64
		if any.Team == 1 {
			a, c = c, a
		}
		score = fmt.Sprintf("score %d-%d pour l'équipe, ", a, c)
	}
	fmt.Fprintf(b, "#### %s, %s, %s (%s%.1f min)\n\n| Joueur | Frags | Actions d'objectif | Part | Part du soutien | Actuelle |", any.Start[:10], any.Map, any.Pair, score, any.Minutes)
	for _, n := range names {
		fmt.Fprintf(b, " %s |", n)
	}
	b.WriteString("\n|---|---|---|---|---|---|")
	for range names {
		b.WriteString("---|")
	}
	b.WriteString("\n")
	anon := 0
	type line struct {
		share float64
		text  string
	}
	var lines []line
	for i := range rowsAll {
		r := &rowsAll[i]
		if r.TeamID != any.Team {
			continue
		}
		sh, ok := narrative.ComputeObjectiveTeamShare(rowsAll, r.XUID)
		if !ok {
			continue
		}
		name, rec := tracked[r.XUID], mine[tracked[r.XUID]]
		if name == "" {
			anon++
			name = fmt.Sprintf("coéquipier non suivi %d", anon)
		}
		var acts []string
		for _, c := range simActionLabels[fam] {
			if v := r.Values[c[0]]; v > 0 {
				acts = append(acts, fmt.Sprintf("%.0f %s", v, c[1]))
			}
		}
		if len(acts) == 0 {
			acts = []string{"aucune"}
		}
		text := fmt.Sprintf("| %s | %d | %s | %.0f %% | %.0f %% |", name, kills[r.XUID], strings.Join(acts, ", "), 100*sh.Share, 100*sh.SupportShare)
		if rec != nil {
			text += fmt.Sprintf(" %.1f |", rec.Old.Score)
			for _, n := range names {
				text += fmt.Sprintf(" %.1f |", rec.By[n].Score)
			}
		} else {
			text += " pas de note |"
			for range names {
				text += " |"
			}
		}
		lines = append(lines, line{sh.Share, text})
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].share > lines[j].share })
	for _, l := range lines {
		b.WriteString(l.text + "\n")
	}
	b.WriteString("\n")
}
