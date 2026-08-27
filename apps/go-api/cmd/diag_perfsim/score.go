package main

// score.go — réplique fidèle du moteur de note relative
// (internal/sync/performance.go + performance_helpers.go), étendue par :
//   - la scission ranked par famille (D-A) ;
//   - la métrique ospm `objective_participation` (D-C) ;
//   - des profils de poids par chaîne.
//
// Les constantes et fonctions partagées avec la production sont IMPORTÉES
// (skill.RelativeWeights, skill.MetricKey*, skill.ComputeCombatYield,
// skillchain.ClassifyLUSRChain, analysis.CombatEfficiency/NormalizeModeLabel)
// plutôt que recopiées : toute dérive du code produit casse la simulation.

import (
	"math"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games/halo_infinite/skillchain"
	"levelup/go-api/internal/sync/skill"
)

const (
	chainArenaSlayer    = "arena_slayer"
	chainArenaObjectif  = "arena_objectif"
	chainRanked         = "ranked"
	chainRankedSlayer   = "ranked_slayer"
	chainRankedObjectif = "ranked_objectif"
	chainFirefight      = "firefight"

	metricKeyOSPM  = "objective_participation"
	keyRankPerfDif = "rank_perf_diff"
)

// objectiveSubModes — COPIE LOCALE de la liste de classify.go:78-80. La
// factorisation en helper partagé unique est le lot B1.1 du plan : l'outil de
// lot 0 ne touche à aucun fichier produit.
var objectiveSubModes = map[string]bool{
	"ctf": true, "capture the flag": true, "neutral flag ctf": true,
	"one flag ctf": true, "covert one flag": true, "strongholds": true,
	"oddball": true, "king of the hill": true, "total control": true,
	"land grab": true, "extraction": true, "stockpile": true,
}

func isObjectiveSubMode(pairName string) bool {
	return objectiveSubModes[strings.ToLower(analysis.NormalizeModeLabel(pairName))]
}

func isObjectiveChain(chain string) bool {
	return chain == chainArenaObjectif || chain == chainRankedObjectif
}

// chainCurrent réplique skill.GetPerformanceChain (skill_config.go:185) pour le
// titre Halo Infinite (classifier par défaut).
func chainCurrent(m *matchRow) string {
	if m.IsRanked {
		return chainRanked
	}
	if m.IsFirefight {
		return chainFirefight
	}
	if c := skillchain.ClassifyLUSRChain(m.PairName); c != "" {
		return c
	}
	return chainArenaSlayer
}

// chainSplit applique la scission ranked par famille (D-A). Fallback documenté :
// pair_name NULL/inconnu sur un match ranked → ranked_slayer.
func chainSplit(m *matchRow) string {
	if m.IsRanked {
		if isObjectiveSubMode(m.PairName) {
			return chainRankedObjectif
		}
		return chainRankedSlayer
	}
	if m.IsFirefight {
		return chainFirefight
	}
	if c := skillchain.ClassifyLUSRChain(m.PairName); c != "" {
		return c
	}
	return chainArenaSlayer
}

// ── Profils de poids ────────────────────────────────────────────────────────

func weightsCurrentFor(string) map[string]float64 { return skill.RelativeWeights }

// weightsMinusDamageFor sert de sonde de sensibilité : il retire une métrique de
// poids 0.06 (dpm_damage) pour mesurer empiriquement l'effet de l'absence d'une
// métrique de même poids — c'est le cas de medal_exploit dans cette simulation.
var weightsMinusDamage = func() map[string]float64 {
	out := make(map[string]float64, len(skill.RelativeWeights))
	for k, v := range skill.RelativeWeights {
		if k == skill.MetricKeyDPMDamage {
			continue
		}
		out[k] = v
	}
	return out
}()

func weightsMinusDamageFor(string) map[string]float64 { return weightsMinusDamage }

// weightsObjective — profil objectif (D-C), poids ospm paramétrable.
func weightsObjective(ospm float64) map[string]float64 {
	return map[string]float64{
		metricKeyOSPM:                   ospm,
		skill.MetricKeyKPM:              0.10,
		skill.MetricKeyKDA:              0.09,
		skill.MetricKeyAccuracy:         0.03,
		skill.MetricKeyPSPM:             0.08,
		skill.MetricKeyDPMDeaths:        0.10,
		skill.MetricKeyAPM:              0.06,
		skill.MetricKeyDPMDamage:        0.06,
		skill.MetricKeyRankPerf:         0.04,
		skill.MetricKeyKillsVsExpected:  0.09,
		skill.MetricKeyDeathsVsExpected: 0.07,
		skill.MetricKeyMedalExploit:     0.06,
		skill.MetricKeyOffensiveConv:    0.09,
		skill.MetricKeyDefensiveResist:  0.05,
	}
}

// weightsFor rend la fonction chaîne → profil du régime NOUVEAU.
func weightsFor(ospm float64) func(string) map[string]float64 {
	obj := weightsObjective(ospm)
	return func(chain string) map[string]float64 {
		if isObjectiveChain(chain) {
			return obj
		}
		return skill.RelativeWeights
	}
}

// ── Métriques ───────────────────────────────────────────────────────────────

type metrics struct {
	KPM                 float64
	DPMDeaths           float64
	APM                 float64
	KDA                 float64
	Accuracy            *float64
	PSPM                *float64
	DPMDamage           *float64
	Rank                *float64
	TeamMMR             *float64
	EnemyMMR            *float64
	KillsVsExpected     *float64
	DeathsVsExpected    *float64
	OffensiveConversion *float64
	DefensiveResistance *float64
	MedalExploit        *float64 // TOUJOURS nil en simulation (cf. rapport)
	OSPM                *float64
}

func posPtr(v float64) *float64 {
	if v > 0 {
		return &v
	}
	return nil
}

// extract réplique extractMatchMetrics (performance.go:78) et ajoute ospm.
func extract(m *matchRow) *metrics {
	duration := m.TimePlayedSeconds
	if duration <= 0 {
		duration = 600.0
	}
	minutes := duration / 60.0
	out := &metrics{
		KPM:       m.Kills / minutes,
		DPMDeaths: m.Deaths / minutes,
		APM:       m.Assists / minutes,
		KDA:       analysis.CombatEfficiency(int(m.Kills), int(m.Assists), int(m.Deaths)),
	}
	out.Accuracy = posPtr(m.Accuracy)
	out.Rank = posPtr(m.Rank)
	out.TeamMMR = posPtr(m.TeamMMR)
	out.EnemyMMR = posPtr(m.EnemyMMR)
	out.OffensiveConversion = posPtr(m.OffensiveConversion)
	out.DefensiveResistance = posPtr(m.DefensiveResistance)
	if m.PersonalScore > 0 {
		out.PSPM = posPtr(m.PersonalScore / minutes)
	}
	if m.DamageDealt > 0 {
		out.DPMDamage = posPtr(m.DamageDealt / minutes)
	}
	if m.KillsExpected > 0 {
		out.KillsVsExpected = posPtr(m.Kills / m.KillsExpected)
	}
	if m.DeathsExpected > 0 && m.Deaths > 0 {
		out.DeathsVsExpected = posPtr(m.DeathsExpected / math.Max(1.0, m.Deaths))
	}
	// ospm : présent SI ET SEULEMENT SI le match a une couverture
	// personal_score_awards. 0 point objectif est une valeur LÉGITIME (le joueur
	// n'a rien fait à l'objectif) ; une absence de données n'est PAS un zéro.
	if m.PSACovered {
		v := m.ObjectiveScore / minutes
		out.OSPM = &v
	}
	return out
}

// standardMetrics — métriques « plus = mieux » (ordre de performance.go:201).
var standardMetrics = []string{
	skill.MetricKeyKPM, skill.MetricKeyAPM, skill.MetricKeyKDA, skill.MetricKeyAccuracy,
	skill.MetricKeyPSPM, skill.MetricKeyDPMDamage, skill.MetricKeyKillsVsExpected,
	skill.MetricKeyDeathsVsExpected, skill.MetricKeyOffensiveConv,
	skill.MetricKeyDefensiveResist, skill.MetricKeyMedalExploit, metricKeyOSPM,
}

func metricValue(m *metrics, key string) (float64, bool) {
	switch key {
	case skill.MetricKeyKPM:
		return m.KPM, true
	case skill.MetricKeyDPMDeaths:
		return m.DPMDeaths, true
	case skill.MetricKeyAPM:
		return m.APM, true
	case skill.MetricKeyKDA:
		return m.KDA, true
	}
	return derefMetric(m, key)
}

// derefMetric rend les métriques OPTIONNELLES (pointeur nil = absente, poids
// redistribué). Les 4 métriques toujours présentes (kpm/dpm_deaths/apm/kda)
// retournent délibérément false ici : buildSeries les alimente explicitement.
func derefMetric(m *metrics, key string) (float64, bool) {
	var ptr *float64
	switch key {
	case skill.MetricKeyAccuracy:
		ptr = m.Accuracy
	case skill.MetricKeyPSPM:
		ptr = m.PSPM
	case skill.MetricKeyDPMDamage:
		ptr = m.DPMDamage
	case skill.MetricKeyKillsVsExpected:
		ptr = m.KillsVsExpected
	case skill.MetricKeyDeathsVsExpected:
		ptr = m.DeathsVsExpected
	case skill.MetricKeyOffensiveConv:
		ptr = m.OffensiveConversion
	case skill.MetricKeyDefensiveResist:
		ptr = m.DefensiveResistance
	case skill.MetricKeyMedalExploit:
		ptr = m.MedalExploit
	case metricKeyOSPM:
		ptr = m.OSPM
	}
	if ptr == nil {
		return 0, false
	}
	return *ptr, true
}

// buildSeries réplique prepareHistoryMetrics (performance_helpers.go:82) + ospm.
func buildSeries(history []matchRow) map[string][]float64 {
	result := make(map[string][]float64, len(standardMetrics)+2)
	for i := range history {
		m := extract(&history[i])
		result[skill.MetricKeyKPM] = append(result[skill.MetricKeyKPM], m.KPM)
		result[skill.MetricKeyDPMDeaths] = append(result[skill.MetricKeyDPMDeaths], m.DPMDeaths)
		result[skill.MetricKeyAPM] = append(result[skill.MetricKeyAPM], m.APM)
		result[skill.MetricKeyKDA] = append(result[skill.MetricKeyKDA], m.KDA)
		for _, key := range standardMetrics {
			v, ok := derefMetric(m, key)
			if !ok {
				continue
			}
			result[key] = append(result[key], v)
		}
		if m.Rank != nil && m.TeamMMR != nil && m.EnemyMMR != nil {
			delta := *m.TeamMMR - *m.EnemyMMR
			result[keyRankPerfDif] = append(result[keyRankPerfDif], (4.5-(delta/100.0)*0.5)-*m.Rank)
		}
	}
	for k, s := range result {
		sort.Float64s(s)
		result[k] = s
	}
	return result
}

// ── Percentiles ─────────────────────────────────────────────────────────────

func percentileRank(value float64, series []float64) float64 {
	if len(series) < 2 {
		return 50.0
	}
	count := 0
	for _, v := range series {
		if v <= value {
			count++
		}
	}
	return clamp(float64(count)/float64(len(series))*100.0, 0, 100)
}

func percentileRankInverse(value float64, series []float64) float64 {
	if len(series) < 2 {
		return 50.0
	}
	count := 0
	for _, v := range series {
		if v >= value {
			count++
		}
	}
	return clamp(float64(count)/float64(len(series))*100.0, 0, 100)
}

func clamp(v, lo, hi float64) float64 {
	return math.Min(math.Max(v, lo), hi)
}

// ── Note d'un match ─────────────────────────────────────────────────────────

// scoreMatch réplique computeRelativePerformanceScore (performance.go:184), la
// liste des métriques retenues étant pilotée par le profil de poids de la chaîne.
func scoreMatch(cur *matchRow, window []matchRow, weights map[string]float64) (*float64, map[string]float64) {
	if len(window) < skill.MinMatchesForRelative {
		return nil, nil
	}
	mm := extract(cur)
	hist := buildSeries(window)
	pct := make(map[string]float64, len(weights))
	used := make(map[string]float64, len(weights))

	for _, key := range standardMetrics {
		w, hasW := weights[key]
		if !hasW || w == 0 {
			continue
		}
		val, ok := metricValue(mm, key)
		if !ok || len(hist[key]) == 0 {
			continue
		}
		pct[key] = percentileRank(val, hist[key])
		used[key] = w
	}
	if w, ok := weights[skill.MetricKeyDPMDeaths]; ok && w != 0 && len(hist[skill.MetricKeyDPMDeaths]) > 0 {
		pct[skill.MetricKeyDPMDeaths] = percentileRankInverse(mm.DPMDeaths, hist[skill.MetricKeyDPMDeaths])
		used[skill.MetricKeyDPMDeaths] = w
	}
	if w, ok := weights[skill.MetricKeyRankPerf]; ok && w != 0 &&
		mm.Rank != nil && mm.TeamMMR != nil && mm.EnemyMMR != nil && len(hist[keyRankPerfDif]) > 0 {
		diff := (4.5 - ((*mm.TeamMMR-*mm.EnemyMMR)/100.0)*0.5) - *mm.Rank
		pct[skill.MetricKeyRankPerf] = percentileRank(diff, hist[keyRankPerfDif])
		used[skill.MetricKeyRankPerf] = w
	}

	if len(pct) == 0 {
		return nil, nil
	}
	total := 0.0
	for _, w := range used {
		total += w
	}
	if total <= 0 {
		return nil, nil
	}
	score := 0.0
	for k, p := range pct {
		score += p * used[k]
	}
	score = math.Round((score/total)*10) / 10
	return &score, pct
}

// ── Exécution d'un régime ───────────────────────────────────────────────────

type simMatch struct {
	MatchID string
	Start   time.Time
	Pair    string
	Kills   float64
	Deaths  float64
	Chain   string
	Note    *float64
	Pct     map[string]float64
}

type chainStat struct {
	NTotal  int
	NScored int
	Notes   []float64
}

type regimeResult struct {
	Label   string
	Matches map[string]*simMatch
	Chains  map[string]*chainStat
}

// runRegime rejoue la boucle de batchComputePerformanceScores (performance.go:367)
// en mode force : fenêtre 50 par chaîne, seuil MinMatchesPerChainForRelative,
// le match courant entre TOUJOURS dans l'historique de sa chaîne.
func runRegime(label string, matches []matchRow, chainOf func(*matchRow) string,
	weightsOf func(string) map[string]float64,
) *regimeResult {
	res := &regimeResult{
		Label:   label,
		Matches: make(map[string]*simMatch, len(matches)),
		Chains:  map[string]*chainStat{},
	}
	history := map[string][]matchRow{}
	for i := range matches {
		m := &matches[i]
		ch := chainOf(m)
		st := res.Chains[ch]
		if st == nil {
			st = &chainStat{}
			res.Chains[ch] = st
		}
		st.NTotal++

		sm := &simMatch{
			MatchID: m.MatchID, Start: m.StartTime, Pair: m.PairName,
			Kills: m.Kills, Deaths: m.Deaths, Chain: ch,
		}
		win := history[ch]
		if len(win) >= skill.MinMatchesPerChainForRelative {
			start := len(win) - windowSize
			if start < 0 {
				start = 0
			}
			if note, pct := scoreMatch(m, win[start:], weightsOf(ch)); note != nil {
				sm.Note, sm.Pct = note, pct
				st.NScored++
				st.Notes = append(st.Notes, *note)
			}
		}
		history[ch] = append(win, *m)
		res.Matches[m.MatchID] = sm
	}
	return res
}
