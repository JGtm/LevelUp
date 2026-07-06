// Package sync — skill_formula_sim.go : simulation de variantes de formule LUSR.
//
// Outil diagnostique : compare 5 formules de score composite sur les N derniers
// matchs d'un joueur, en partant de InitialMU=1500 pour chaque variante.
// Sans écriture DB — lecture seule.
package skill

import (
	"context"
	"database/sql"
	"math"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// ── Types publics ─────────────────────────────────────────────────────────────

// FormulaVariant décrit une variante de formule de score composite.
type FormulaVariant struct {
	Name        string
	Weights     map[string]float64 // nil = CompositeWeights (baseline)
	KNEMode     bool               // remplace KvE+DvE par kills*DE / deaths*KE (rendement net)
	KDAMode     bool               // remplace KvE+DvE par un KDA combiné
	DvEAdjust   bool               // ajuste DE selon performance kills (piste-A)
	DvEFloor    bool               // DvE ≥ 0.5 : supprime la pénalité morts, garde la récompense
	KvMMode     bool               // remplace KvE+DvE par kills vs moyenne ennemis réels (pas d'expected)
	KvMatchMode bool               // remplace KvE+DvE par kills vs moyenne de TOUS les joueurs du match
	AccFloor    bool               // accuracy ≥ 0.5 : tirs de couverture non pénalisants
	DmgVsMatch  bool               // remplace DamageEfficiency par damage_dealt vs moyenne du match
	LastN       int                // 0 = tous les matchs ; > 0 = fenêtre glissante de N matchs
}

// simEnemy regroupe les stats adverses et globales calculées depuis les participants réels du match.
type simEnemy struct {
	AvgKE        *float64 // moyenne KE ennemis (carry adj des variantes classiques)
	AvgKills     *float64 // moyenne kills réels ennemis (KvM)
	AvgDeaths    *float64 // moyenne deaths réels ennemis (KvM)
	AvgDamage    *float64 // moyenne damage_dealt de tous les joueurs du match (DmgVsMatch / no-expected)
	AllAvgKills  *float64 // moyenne kills de TOUS les joueurs du match (no-expected)
	AllAvgDeaths *float64 // moyenne deaths de TOUS les joueurs du match (no-expected)
}

// FormulaSimResult contient les MU finaux pour toutes les variantes sur une chaîne.
type FormulaSimResult struct {
	Chain          string
	MatchCount     int
	MUByVariant    map[string]float64 // variant.Name → MU final
	SigmaByVariant map[string]float64 // variant.Name → Sigma final
}

// FormulaSimReport regroupe les résultats par joueur.
type FormulaSimReport struct {
	XUID    string
	Results []FormulaSimResult
	LastN   int // matchs demandés (0 = tous)
}

// ── Variantes prédéfinies ─────────────────────────────────────────────────────

// simWeightsCsrAligned : inspiré du CSR Halo — morts non pénalisantes (DvE réduit),
// damage dealing primordial (DmgEff ×2), kills conservés.
var simWeightsCsrAligned = map[string]float64{
	MetricKeyKillsVsExpected:  0.30,
	MetricKeyDeathsVsExpected: 0.08,
	MetricKeyWinFactor:        0.08,
	MetricKeyDamageEfficiency: 0.22,
	MetricKeyAccuracyDelta:    0.10,
	MetricKeyMedalExploit:     0.04,
	MetricKeyOffensiveConv:    0.12,
	MetricKeyDefensiveResist:  0.06,
}

// simWeightsDmgFirst : damage vs match avg comme signal principal, KvE/DvE avec expected réduits.
var simWeightsDmgFirst = map[string]float64{
	MetricKeyKillsVsExpected:  0.28,
	MetricKeyDeathsVsExpected: 0.08,
	MetricKeyWinFactor:        0.08,
	MetricKeyDamageEfficiency: 0.26,
	MetricKeyAccuracyDelta:    0.05,
	MetricKeyMedalExploit:     0.04,
	MetricKeyOffensiveConv:    0.15,
	MetricKeyDefensiveResist:  0.06,
}

// simWeightsDDEff : zéro dépendance aux KE/DE Microsoft.
// Signal primaire : DD/K+A (OffConv) + damage_dealt vs avg match.
// Inspiré du benchmark communautaire CompetitiveHalo : DD/K+A ≤ 200 = métrique #1.
// KvE/DvE à 0 — kills/morts capturés indirectement via OffConv et DmgVsMatch.
var simWeightsDDEff = map[string]float64{
	MetricKeyKillsVsExpected:  0.00,
	MetricKeyDeathsVsExpected: 0.00,
	MetricKeyWinFactor:        0.10,
	MetricKeyDamageEfficiency: 0.35, // damage_dealt vs avg match (primaire)
	MetricKeyAccuracyDelta:    0.05, // floor — reward-only
	MetricKeyMedalExploit:     0.08,
	MetricKeyOffensiveConv:    0.35, // DD/K+A — benchmark Reddit #1
	MetricKeyDefensiveResist:  0.07,
}

// SimulationVariants liste toutes les variantes à comparer.
var SimulationVariants = []FormulaVariant{
	{Name: "baseline"},
	{Name: "csr-aligned", DvEFloor: true, Weights: simWeightsCsrAligned},
	{Name: "dmg-first-w200", DvEFloor: true, AccFloor: true, DmgVsMatch: true, Weights: simWeightsDmgFirst, LastN: 200},
	{Name: "dd-eff-w50", DvEFloor: true, AccFloor: true, DmgVsMatch: true, Weights: simWeightsDDEff, LastN: 50},
	{Name: "dd-eff-w200", DvEFloor: true, AccFloor: true, DmgVsMatch: true, Weights: simWeightsDDEff, LastN: 200},
}

// ── Score composite avec variante ────────────────────────────────────────────

type simEntry struct{ score, weight float64 }

// computeCompositeForSim calcule le score composite [0,1] selon une variante.
func computeCompositeForSim(
	row *compositeMatchRow,
	enemy simEnemy,
	avgAcc, avgDmgEff, avgMedalExploit, avgOffConv, avgDefRes *float64,
	v FormulaVariant,
) float64 {
	w := CompositeWeights
	if v.Weights != nil {
		w = v.Weights
	}
	entries := make(map[string]simEntry, 8)
	simAddKillComponents(row, enemy, w, v, entries)
	simAddSupportComponents(row, enemy, avgAcc, avgDmgEff, avgMedalExploit, avgOffConv, avgDefRes, w, v, entries)
	return simWeightedAvg(entries)
}

// simAddKillComponents ajoute les composantes KvE/DvE selon la variante.
func simAddKillComponents(
	row *compositeMatchRow,
	enemy simEnemy,
	w map[string]float64,
	v FormulaVariant,
	out map[string]simEntry,
) {
	kneW := w[MetricKeyKillsVsExpected] + w[MetricKeyDeathsVsExpected]

	// KNE : rendement net kills×DE / deaths×KE — fusionne les métriques corrélées.
	// Neutre (0.5) quand kills/deaths == KE/DE, récompense si on surpasse le ratio.
	if v.KNEMode {
		if row.KillsExpected > 0 && row.DeathsExpected > 0 {
			kne := sigmoidRatio(row.Kills*row.DeathsExpected,
				math.Max(1.0, row.Deaths)*row.KillsExpected)
			out["kne"] = simEntry{kne, kneW}
		}
		return
	}

	// KDA : (kills+assists/3)/deaths vs KE/DE — mesure l'efficacité combinée.
	if v.KDAMode {
		if row.KillsExpected > 0 && row.DeathsExpected > 0 {
			kdaReal := (row.Kills + row.Assists/3.0) / math.Max(1.0, row.Deaths)
			kdaExp := row.KillsExpected / math.Max(1.0, row.DeathsExpected)
			out["kda"] = simEntry{sigmoidRatio(kdaReal, kdaExp), kneW}
		}
		return
	}

	// KvM : kills vs moyenne réelle des ennemis — sans dépendance Microsoft KE/DE.
	if v.KvMMode {
		if enemy.AvgKills != nil && *enemy.AvgKills > 0 {
			out[MetricKeyKillsVsExpected] = simEntry{sigmoidRatio(row.Kills, *enemy.AvgKills), w[MetricKeyKillsVsExpected]}
		}
		if enemy.AvgDeaths != nil && *enemy.AvgDeaths > 0 && w[MetricKeyDeathsVsExpected] > 0 {
			dvm := sigmoidRatio(*enemy.AvgDeaths, math.Max(1.0, row.Deaths))
			if v.DvEFloor && dvm < 0.5 {
				dvm = 0.5
			}
			out[MetricKeyDeathsVsExpected] = simEntry{dvm, w[MetricKeyDeathsVsExpected]}
		}
		return
	}

	// KvMatch : kills vs moyenne de TOUS les joueurs du match (pas d'expected Microsoft).
	// Plus stable que KvM car la moyenne match est moins biaisée par la composition d'équipe.
	if v.KvMatchMode {
		if enemy.AllAvgKills != nil && *enemy.AllAvgKills > 0 {
			out[MetricKeyKillsVsExpected] = simEntry{sigmoidRatio(row.Kills, *enemy.AllAvgKills), w[MetricKeyKillsVsExpected]}
		}
		if enemy.AllAvgDeaths != nil && *enemy.AllAvgDeaths > 0 && w[MetricKeyDeathsVsExpected] > 0 {
			dvm := sigmoidRatio(*enemy.AllAvgDeaths, math.Max(1.0, row.Deaths))
			if v.DvEFloor && dvm < 0.5 {
				dvm = 0.5
			}
			out[MetricKeyDeathsVsExpected] = simEntry{dvm, w[MetricKeyDeathsVsExpected]}
		}
		return
	}

	// Chemin normal : KvE + DvE indépendants.

	// KvE avec carry adjustment (identique à la formule de prod).
	if row.KillsExpected > 0 {
		score := sigmoidRatio(row.Kills, row.KillsExpected)
		if enemy.AvgKE != nil && *enemy.AvgKE > 0 {
			cr := row.KillsExpected / *enemy.AvgKE
			adj := ClampF(cr, 1.0, 2.0)
			if score > 0.5 {
				score = ClampF(0.5+(score-0.5)/adj, 0.0, 1.0)
			}
		}
		out[MetricKeyKillsVsExpected] = simEntry{score, w[MetricKeyKillsVsExpected]}
	}

	// DvE — piste-A : scale DE par le ratio kills/KE (plus de tolérance si bons kills).
	// DvE-floor : clamp à 0.5 minimum — les morts ne pénalisent jamais, récompensent seulement.
	if row.DeathsExpected > 0 && w[MetricKeyDeathsVsExpected] > 0 {
		effDE := row.DeathsExpected
		if v.DvEAdjust && row.KillsExpected > 0 {
			effDE = math.Max(0.1, row.DeathsExpected*(row.Kills/row.KillsExpected))
		}
		dve := sigmoidRatio(effDE, math.Max(1.0, row.Deaths))
		if v.DvEFloor && dve < 0.5 {
			dve = 0.5
		}
		out[MetricKeyDeathsVsExpected] = simEntry{dve, w[MetricKeyDeathsVsExpected]}
	}
}

// simAddSupportComponents ajoute les 6 composantes communes (win, dmg, acc, medal, off, def).
func simAddSupportComponents(
	row *compositeMatchRow,
	enemy simEnemy,
	avgAcc, avgDmgEff, avgMedalExploit, avgOffConv, avgDefRes *float64,
	w map[string]float64,
	v FormulaVariant,
	out map[string]simEntry,
) {
	if row.Outcome != nil {
		var s float64
		switch *row.Outcome {
		case domain.OutcomeWin:
			s = 1.0
		case domain.OutcomeDraw:
			s = 0.5
		case domain.OutcomeLoss:
			s = 0.0
		case domain.OutcomeDNF:
			s = 0.15
		default:
			s = 0.5
		}
		out[MetricKeyWinFactor] = simEntry{s, w[MetricKeyWinFactor]}
	}
	// DmgVsMatch : damage_dealt vs moyenne du match (signal absolu, contextuel).
	// Fallback sur le ratio dealt/total si avgDamage indisponible.
	if v.DmgVsMatch && enemy.AvgDamage != nil && *enemy.AvgDamage > 0 && row.DamageDealt > 0 {
		out[MetricKeyDamageEfficiency] = simEntry{sigmoidRatio(row.DamageDealt, *enemy.AvgDamage), w[MetricKeyDamageEfficiency]}
	} else {
		total := row.DamageDealt + row.DamageTaken
		if total > 0 {
			rawEff := ClampF(row.DamageDealt/total, 0, 1)
			se := rawEff
			if avgDmgEff != nil && *avgDmgEff > 0 {
				se = sigmoidRatio(rawEff, *avgDmgEff)
			}
			out[MetricKeyDamageEfficiency] = simEntry{se, w[MetricKeyDamageEfficiency]}
		}
	}
	// AccFloor : tirs de couverture non pénalisants — accuracy ≥ 0.5 seulement.
	if row.Accuracy > 0 && avgAcc != nil && *avgAcc > 0 {
		acc := sigmoidRatio(row.Accuracy, *avgAcc)
		if v.AccFloor && acc < 0.5 {
			acc = 0.5
		}
		out[MetricKeyAccuracyDelta] = simEntry{acc, w[MetricKeyAccuracyDelta]}
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
	return ClampF(sum/totalW, 0.0, 1.0)
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
	excluded, err := LoadExcludedMatchIDs(ctx, playerDB)
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

	// Fenêtre par variante : startByVariant[name] = premier index à traiter.
	startByVariant := make(map[string]int, len(SimulationVariants))
	for _, v := range SimulationVariants {
		if v.LastN > 0 && len(matches) > v.LastN {
			startByVariant[v.Name] = len(matches) - v.LastN
		}
	}

	chainCount := make(map[string]int)
	for i, match := range matches {
		if excluded[match.MatchID] {
			continue
		}
		pairName := ""
		if match.PairName != nil {
			pairName = *match.PairName
		}
		// Title-aware (C6) : cohérent avec computeSkillRatingsBatch (le simulateur
		// doit classer les modes comme le calcul réel). ""/halo_infinite → défaut.
		chain := GetLUSRChainForTitle(ctxkeys.TitleSlug(ctx), pairName)
		if chain == "" {
			continue
		}
		chainCount[chain]++
		if match.Outcome == nil {
			continue
		}
		simProcessMatch(i, startByVariant, match, parts[match.MatchID], medalByMatch[match.MatchID], chain, states)
	}
	return buildFormulaSimReport(xuid, lastN, chainCount, states), nil
}

// simProcessMatch calcule le composite selon chaque variante et met à jour l'état TrueSkill.
// matchIdx est la position dans la slice globale ; startByVariant permet à chaque variante
// d'ignorer les matchs antérieurs à sa fenêtre (v.LastN).
func simProcessMatch(
	matchIdx int,
	startByVariant map[string]int,
	match LusrMatchData,
	allParts []lusrParticipant,
	medalScore float64,
	chain string,
	states map[simStateKey]*PlayerState,
) {
	matchAvgKE, matchStdKE := computeMatchKEStats(allParts)
	_, enemyKEs := splitParticipantKEs(match.TeamID, allParts)

	var enemy simEnemy
	if len(enemyKEs) > 0 {
		sum := 0.0
		for _, ke := range enemyKEs {
			sum += ke
		}
		avg := sum / float64(len(enemyKEs))
		enemy.AvgKE = &avg
	}
	// Calcul des stats réelles des ennemis (KvM) et de la moyenne match (DmgVsMatch / no-expected).
	{
		var sumK, sumD, sumDmg, sumAllK, sumAllD float64
		nEnemy, nAll := 0, 0
		for _, p := range allParts {
			sumDmg += p.DamageDealt
			sumAllK += p.Kills
			sumAllD += p.Deaths
			nAll++
			if match.TeamID != nil && p.TeamID != nil && *p.TeamID == *match.TeamID {
				continue
			}
			sumK += p.Kills
			sumD += p.Deaths
			nEnemy++
		}
		if nEnemy > 0 {
			avgK := sumK / float64(nEnemy)
			avgD := sumD / float64(nEnemy)
			enemy.AvgKills = &avgK
			enemy.AvgDeaths = &avgD
		}
		if nAll > 0 {
			if sumDmg > 0 {
				avgDmg := sumDmg / float64(nAll)
				enemy.AvgDamage = &avgDmg
			}
			avgAllK := sumAllK / float64(nAll)
			avgAllD := sumAllD / float64(nAll)
			enemy.AllAvgKills = &avgAllK
			enemy.AllAvgDeaths = &avgAllD
		}
	}
	offConv, defRes := ComputeCombatYield(match)
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
		// Fenêtre glissante : ignorer les matchs antérieurs à v.LastN.
		if start, ok := startByVariant[v.Name]; ok && matchIdx < start {
			continue
		}
		state := states[simStateKey{v.Name, chain}]
		avgAcc := rollingAvgPtr(state.AccuracyHistory)
		avgDmgEff := rollingAvgPtr(state.DamageEffHistory)
		avgMedalExploit := rollingAvgPtr(state.MedalExploitHistory)
		avgOffConv := rollingAvgPtr(state.OffConversionHistory)
		avgDefRes := rollingAvgPtr(state.DefResistanceHistory)

		muOpp, sigmaOpp := computeEnemyStrength(enemyKEs, matchAvgKE, matchStdKE, state.MU)
		composite := computeCompositeForSim(cRow, enemy, avgAcc, avgDmgEff, avgMedalExploit, avgOffConv, avgDefRes, v)
		state.MU, state.Sigma = trueskillUpdate(state.MU, state.Sigma, muOpp, sigmaOpp, composite, 1.0)

		appendToHistory(&state.AccuracyHistory, match.Accuracy)
		totalDmg := match.DamageDealt + match.DamageTaken
		if totalDmg > 0 {
			appendToHistory(&state.DamageEffHistory, ClampF(match.DamageDealt/totalDmg, 0, 1))
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
			Chain:          chain,
			MatchCount:     count,
			MUByVariant:    make(map[string]float64, len(SimulationVariants)),
			SigmaByVariant: make(map[string]float64, len(SimulationVariants)),
		}
		for _, v := range SimulationVariants {
			st := states[simStateKey{v.Name, chain}]
			r.MUByVariant[v.Name] = math.Round(st.MU*10) / 10
			r.SigmaByVariant[v.Name] = math.Round(st.Sigma*10) / 10
		}
		results = append(results, r)
	}
	return &FormulaSimReport{XUID: xuid, Results: results, LastN: lastN}
}
