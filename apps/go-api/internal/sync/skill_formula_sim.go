// Package sync — skill_formula_sim.go : simulation de variantes de formule LUSR.
//
// Outil diagnostique : compare 5 formules de score composite sur les N derniers
// matchs d'un joueur, en partant de InitialMU=1500 pour chaque variante.
// Sans écriture DB — lecture seule.
package sync

import (
	"context"
	"database/sql"
	"math"

	"levelup/go-api/internal/analysis"
)

// ── Types publics ─────────────────────────────────────────────────────────────

// FormulaVariant décrit une variante de formule de score composite.
type FormulaVariant struct {
	Name      string
	Weights   map[string]float64 // nil = CompositeWeights (baseline)
	KDAMode   bool               // remplace KvE+DvE par un KDA combiné
	DvEAdjust bool               // ajuste DE selon performance kills (piste-A)
}

// FormulaSimResult contient les MU finaux pour toutes les variantes sur une chaîne.
type FormulaSimResult struct {
	Chain       string
	MatchCount  int
	MUByVariant map[string]float64 // variant.Name → MU final
}

// FormulaSimReport regroupe les résultats par joueur.
type FormulaSimReport struct {
	XUID    string
	Results []FormulaSimResult
	LastN   int // matchs demandés (0 = tous)
}

// ── Variantes prédéfinies ─────────────────────────────────────────────────────

// simWeightsPisteC réduit DvE (0.24→0.12) et augmente KvE (0.27→0.36).
var simWeightsPisteC = map[string]float64{
	MetricKeyKillsVsExpected:  0.36,
	MetricKeyDeathsVsExpected: 0.12,
	MetricKeyWinFactor:        0.05,
	MetricKeyDamageEfficiency: 0.10,
	MetricKeyAccuracyDelta:    0.10,
	MetricKeyMedalExploit:     0.04,
	MetricKeyOffensiveConv:    0.16,
	MetricKeyDefensiveResist:  0.06,
}

// SimulationVariants est la liste des 5 variantes comparées.
var SimulationVariants = []FormulaVariant{
	{Name: "baseline"},
	{Name: "piste-C", Weights: simWeightsPisteC},
	{Name: "piste-A", DvEAdjust: true},
	{Name: "piste-A+C", DvEAdjust: true, Weights: simWeightsPisteC},
	{Name: "piste-B", KDAMode: true},
}

// ── Score composite avec variante ────────────────────────────────────────────

type simEntry struct{ score, weight float64 }

// computeCompositeForSim calcule le score composite [0,1] selon une variante.
func computeCompositeForSim(
	row *compositeMatchRow,
	enemyAvgKE, avgAcc, avgDmgEff, avgMedalExploit, avgOffConv, avgDefRes *float64,
	v FormulaVariant,
) float64 {
	w := CompositeWeights
	if v.Weights != nil {
		w = v.Weights
	}
	entries := make(map[string]simEntry, 8)
	simAddKillComponents(row, enemyAvgKE, w, v, entries)
	simAddSupportComponents(row, avgAcc, avgDmgEff, avgMedalExploit, avgOffConv, avgDefRes, w, entries)
	return simWeightedAvg(entries)
}

// simAddKillComponents ajoute les composantes KvE/DvE (ou KDA pour piste-B).
func simAddKillComponents(
	row *compositeMatchRow,
	enemyAvgKE *float64,
	w map[string]float64,
	v FormulaVariant,
	out map[string]simEntry,
) {
	if v.KDAMode {
		if row.KillsExpected > 0 && row.DeathsExpected > 0 {
			kdaReal := (row.Kills + row.Assists/3.0) / math.Max(1.0, row.Deaths)
			kdaExp := row.KillsExpected / math.Max(1.0, row.DeathsExpected)
			kdaW := w[MetricKeyKillsVsExpected] + w[MetricKeyDeathsVsExpected]
			out["kda"] = simEntry{sigmoidRatio(kdaReal, kdaExp), kdaW}
		}
		return
	}

	// KvE avec carry adjustment (identique à la formule de prod)
	if row.KillsExpected > 0 {
		score := sigmoidRatio(row.Kills, row.KillsExpected)
		if enemyAvgKE != nil && *enemyAvgKE > 0 {
			cr := row.KillsExpected / *enemyAvgKE
			adj := clampF(cr, 1.0, 2.0)
			if score > 0.5 {
				score = clampF(0.5+(score-0.5)/adj, 0.0, 1.0)
			}
		}
		out[MetricKeyKillsVsExpected] = simEntry{score, w[MetricKeyKillsVsExpected]}
	}

	// DvE — piste-A : ajuste DE proportionnellement à la performance kills.
	// Si le joueur fait 2× son KE, son DE effective double → plus de tolérance.
	if row.DeathsExpected > 0 {
		effDE := row.DeathsExpected
		if v.DvEAdjust && row.KillsExpected > 0 {
			effDE = math.Max(0.1, row.DeathsExpected*(row.Kills/row.KillsExpected))
		}
		out[MetricKeyDeathsVsExpected] = simEntry{
			sigmoidRatio(effDE, math.Max(1.0, row.Deaths)),
			w[MetricKeyDeathsVsExpected],
		}
	}
}

// simAddSupportComponents ajoute les 6 composantes communes (win, dmg, acc, medal, off, def).
func simAddSupportComponents(
	row *compositeMatchRow,
	avgAcc, avgDmgEff, avgMedalExploit, avgOffConv, avgDefRes *float64,
	w map[string]float64,
	out map[string]simEntry,
) {
	if row.Outcome != nil {
		var s float64
		switch *row.Outcome {
		case 2:
			s = 1.0
		case 1:
			s = 0.5
		case 3:
			s = 0.0
		case 4:
			s = 0.15
		default:
			s = 0.5
		}
		out[MetricKeyWinFactor] = simEntry{s, w[MetricKeyWinFactor]}
	}
	total := row.DamageDealt + row.DamageTaken
	if total > 0 {
		rawEff := clampF(row.DamageDealt/total, 0, 1)
		se := rawEff
		if avgDmgEff != nil && *avgDmgEff > 0 {
			se = sigmoidRatio(rawEff, *avgDmgEff)
		}
		out[MetricKeyDamageEfficiency] = simEntry{se, w[MetricKeyDamageEfficiency]}
	}
	if row.Accuracy > 0 && avgAcc != nil && *avgAcc > 0 {
		out[MetricKeyAccuracyDelta] = simEntry{sigmoidRatio(row.Accuracy, *avgAcc), w[MetricKeyAccuracyDelta]}
	}
	if row.MedalExploitScore > 0 {
		ref := 5.0
		if avgMedalExploit != nil && *avgMedalExploit > 1e-9 {
			ref = *avgMedalExploit
		}
		out[MetricKeyMedalExploit] = simEntry{sigmoidRatio(row.MedalExploitScore, ref), w[MetricKeyMedalExploit]}
	}
	if row.OffensiveConversion > 0 {
		ref := analysis.OffensiveConversionP80
		if avgOffConv != nil && *avgOffConv > 1e-9 {
			ref = *avgOffConv
		}
		out[MetricKeyOffensiveConv] = simEntry{sigmoidRatio(row.OffensiveConversion, ref), w[MetricKeyOffensiveConv]}
	}
	if row.DefensiveResistance > 0 {
		ref := analysis.DefensiveResistanceP80
		if avgDefRes != nil && *avgDefRes > 1e-9 {
			ref = *avgDefRes
		}
		out[MetricKeyDefensiveResist] = simEntry{sigmoidRatio(row.DefensiveResistance, ref), w[MetricKeyDefensiveResist]}
	}
}

func simWeightedAvg(entries map[string]simEntry) float64 {
	if len(entries) == 0 {
		return 0.5
	}
	totalW, sum := 0.0, 0.0
	for _, e := range entries {
		totalW += e.weight
		sum += e.score * e.weight
	}
	if totalW < 1e-12 {
		return 0.5
	}
	return clampF(sum/totalW, 0.0, 1.0)
}

// ── Simulation runner ─────────────────────────────────────────────────────────

type simStateKey struct{ variant, chain string }

// RunFormulaSim simule les 5 variantes de formule sur les lastN derniers matchs
// éligibles LUSR, depuis InitialMU=1500 pour chaque variante.
// 0 → tous les matchs. Lecture seule — aucune écriture DB.
func RunFormulaSim(
	ctx context.Context,
	playerDB, sharedDB *sql.DB,
	xuid string,
	medalByMatch map[string]float64,
	lastN int,
) (*FormulaSimReport, error) {
	matches, err := loadLUSRMatchData(ctx, sharedDB, xuid)
	if err != nil {
		return nil, err
	}
	excluded, err := loadExcludedMatchIDs(ctx, playerDB)
	if err != nil {
		return nil, err
	}
	if lastN > 0 && len(matches) > lastN {
		matches = matches[len(matches)-lastN:]
	}
	if len(matches) == 0 {
		return &FormulaSimReport{XUID: xuid, LastN: lastN}, nil
	}
	matchIDs := make([]string, len(matches))
	for i, m := range matches {
		matchIDs[i] = m.MatchID
	}
	parts, err := loadLUSRParticipants(ctx, sharedDB, matchIDs)
	if err != nil {
		return nil, err
	}

	states := make(map[simStateKey]*PlayerState)
	for _, v := range SimulationVariants {
		for chain := range LUSRChains {
			states[simStateKey{v.Name, chain}] = NewPlayerState()
		}
	}
	chainCount := make(map[string]int)

	for _, match := range matches {
		if excluded[match.MatchID] {
			continue
		}
		pairName := ""
		if match.PairName != nil {
			pairName = *match.PairName
		}
		chain := GetLUSRChain(pairName)
		if chain == "" {
			continue
		}
		chainCount[chain]++
		if match.Outcome == nil {
			continue
		}
		simProcessMatch(match, parts[match.MatchID], medalByMatch[match.MatchID], chain, states)
	}
	return buildFormulaSimReport(xuid, lastN, chainCount, states), nil
}

// simProcessMatch calcule le composite selon chaque variante et met à jour l'état TrueSkill.
func simProcessMatch(
	match lusrMatchData,
	allParts []lusrParticipant,
	medalScore float64,
	chain string,
	states map[simStateKey]*PlayerState,
) {
	matchAvgKE, matchStdKE := computeMatchKEStats(allParts)
	_, enemyKEs := splitParticipantKEs(match.TeamID, allParts)

	var enemyAvgKE *float64
	if len(enemyKEs) > 0 {
		sum := 0.0
		for _, ke := range enemyKEs {
			sum += ke
		}
		avg := sum / float64(len(enemyKEs))
		enemyAvgKE = &avg
	}
	offConv, defRes := computeCombatYield(match)
	cRow := &compositeMatchRow{
		Kills: match.Kills, Deaths: match.Deaths, Assists: match.Assists,
		KillsExpected: match.KillsExpected, DeathsExpected: match.DeathsExpected,
		Outcome:             match.Outcome,
		DamageDealt:         match.DamageDealt,
		DamageTaken:         match.DamageTaken,
		Accuracy:            match.Accuracy,
		MedalExploitScore:   medalScore,
		OffensiveConversion: offConv,
		DefensiveResistance: defRes,
	}

	for _, v := range SimulationVariants {
		state := states[simStateKey{v.Name, chain}]
		avgAcc := rollingAvgPtr(state.AccuracyHistory)
		avgDmgEff := rollingAvgPtr(state.DamageEffHistory)
		avgMedalExploit := rollingAvgPtr(state.MedalExploitHistory)
		avgOffConv := rollingAvgPtr(state.OffConversionHistory)
		avgDefRes := rollingAvgPtr(state.DefResistanceHistory)

		muOpp, sigmaOpp := computeEnemyStrength(enemyKEs, matchAvgKE, matchStdKE, state.MU)
		composite := computeCompositeForSim(cRow, enemyAvgKE, avgAcc, avgDmgEff, avgMedalExploit, avgOffConv, avgDefRes, v)
		state.MU, state.Sigma = trueskillUpdate(state.MU, state.Sigma, muOpp, sigmaOpp, composite, 1.0)

		appendToHistory(&state.AccuracyHistory, match.Accuracy)
		totalDmg := match.DamageDealt + match.DamageTaken
		if totalDmg > 0 {
			appendToHistory(&state.DamageEffHistory, clampF(match.DamageDealt/totalDmg, 0, 1))
		}
		if medalScore > 0 {
			appendToHistory(&state.MedalExploitHistory, medalScore)
		}
		if offConv > 0 {
			appendToHistory(&state.OffConversionHistory, offConv)
		}
		if defRes > 0 {
			appendToHistory(&state.DefResistanceHistory, defRes)
		}
	}
}

func buildFormulaSimReport(
	xuid string,
	lastN int,
	chainCount map[string]int,
	states map[simStateKey]*PlayerState,
) *FormulaSimReport {
	results := make([]FormulaSimResult, 0, len(chainCount))
	for chain, count := range chainCount {
		r := FormulaSimResult{
			Chain:       chain,
			MatchCount:  count,
			MUByVariant: make(map[string]float64, len(SimulationVariants)),
		}
		for _, v := range SimulationVariants {
			r.MUByVariant[v.Name] = math.Round(states[simStateKey{v.Name, chain}].MU*10) / 10
		}
		results = append(results, r)
	}
	return &FormulaSimReport{XUID: xuid, Results: results, LastN: lastN}
}
