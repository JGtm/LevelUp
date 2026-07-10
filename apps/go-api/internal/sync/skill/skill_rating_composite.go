package skill

import (
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"math"
)

type compositeMatchRow struct {
	Kills               float64
	Deaths              float64
	Assists             float64
	KillsExpected       float64
	DeathsExpected      float64
	Outcome             *int
	DamageDealt         float64
	DamageTaken         float64
	Accuracy            float64
	MedalExploitScore   float64 // score brut ComputeMedalExploitScore, 0 si absent
	OffensiveConversion float64 // 225*(kills+assists/3)/damage_dealt, 0 si absent
	DefensiveResistance float64 // damage_taken/(225*deaths), 0 si absent
}

// computeCompositeScore calcule le score composite [0,1] d'un match. Wrapper
// rétro-compatible autour de computeCompositeScoreWithBreakdown qui ne retourne
// que le composite (utilisé par les tests existants).
//
// Les composantes manquantes (valeur 0 ou avg nil) sont ignorées et les poids
// renormalisés. 4 params réservés pour futures composantes du score composite
// (enemyAvgKE = carry adjustment vs adversaires ; avgMedalExploit = bonus exploit ;
// avgOffConv = offensive conversion ; avgDefRes = defensive resistance).
//
//nolint:unparam // signature stable pour formule PerfTier roadmap, callers passent nil aujourd'hui.
func computeCompositeScore(
	row *compositeMatchRow,
	avgAccuracy *float64,
	enemyAvgKE *float64,
	avgDamageEff *float64,
	avgMedalExploit *float64,
	avgOffConv *float64,
	avgDefRes *float64,
) float64 {
	composite, _ := computeCompositeScoreWithBreakdown(row, avgAccuracy, enemyAvgKE, avgDamageEff, avgMedalExploit, avgOffConv, avgDefRes)
	return composite
}

// computeCompositeScoreWithBreakdown calcule composite + breakdown détaillé.
// Le breakdown contient les valeurs des composantes effectivement calculées
// (clés = noms canoniques de CompositeWeights). Les composantes absentes du
// match (donnée manquante, poids 0) ne figurent pas dans la map — leur lecture
// retourne 0 côté Go, ce qui simplifie l'INSERT en batch.
//
// Utilisé par computeSkillRatingsBatch (commit V2-1) pour persister
// `lusr_component_history` en plus de `match_skill_rank`.
func computeCompositeScoreWithBreakdown(
	row *compositeMatchRow,
	avgAccuracy *float64,
	enemyAvgKE *float64,
	avgDamageEff *float64,
	avgMedalExploit *float64,
	avgOffConv *float64,
	avgDefRes *float64,
) (float64, map[string]float64) {
	w := CompositeWeights
	type entry struct {
		key    string
		score  float64
		weight float64
	}
	var valid []entry

	// 1. kills_vs_expected
	// Carry adjustment asymétrique : compresse le bonus quand les adversaires
	// sont faibles (playerKE >> enemyAvgKE), mais ne touche pas aux pénalités.
	// Référence : enemyAvgKE (difficulté réelle des adversaires), pas les
	// coéquipiers — évite de pénaliser un carry pour la faiblesse de son équipe.
	// Floor carryAdj à 1.0 : pas d'amplification si les adversaires sont plus forts.
	if row.KillsExpected > 0 {
		score := sigmoidRatio(row.Kills, row.KillsExpected)
		if enemyAvgKE != nil && *enemyAvgKE > 0 {
			carryRatio := row.KillsExpected / *enemyAvgKE
			carryAdj := ClampF(carryRatio, 1.0, 2.0)
			if score > 0.5 {
				score = ClampF(0.5+(score-0.5)/carryAdj, 0.0, 1.0)
			}
			// score ≤ 0.5 : pénalité pleine, non modifiée
		}
		valid = append(valid, entry{MetricKeyKillsVsExpected, score, w[MetricKeyKillsVsExpected]})
	}

	// 2. deaths_vs_expected (inversé)
	if row.DeathsExpected > 0 {
		score := sigmoidRatio(row.DeathsExpected, math.Max(1.0, row.Deaths))
		valid = append(valid, entry{MetricKeyDeathsVsExpected, score, w[MetricKeyDeathsVsExpected]})
	}

	// 3. win_factor
	if row.Outcome != nil {
		var winScore float64
		switch *row.Outcome {
		case domain.OutcomeWin:
			winScore = 1.0
		case domain.OutcomeDraw:
			winScore = 0.5
		case domain.OutcomeLoss:
			winScore = 0.0
		case domain.OutcomeDNF:
			winScore = 0.15
		default:
			winScore = 0.5
		}
		valid = append(valid, entry{MetricKeyWinFactor, winScore, w[MetricKeyWinFactor]})
	}

	// 4. damage_efficiency
	total := row.DamageDealt + row.DamageTaken
	if total > 0 {
		rawEff := ClampF(row.DamageDealt/total, 0.0, 1.0)
		scoreEff := rawEff
		if avgDamageEff != nil && *avgDamageEff > 0 {
			scoreEff = sigmoidRatio(rawEff, *avgDamageEff)
		}
		valid = append(valid, entry{MetricKeyDamageEfficiency, scoreEff, w[MetricKeyDamageEfficiency]})
	}

	// 5. accuracy_delta
	if row.Accuracy > 0 && avgAccuracy != nil && *avgAccuracy > 0 {
		score := sigmoidRatio(row.Accuracy, *avgAccuracy)
		valid = append(valid, entry{MetricKeyAccuracyDelta, score, w[MetricKeyAccuracyDelta]})
	}

	// 6. medal_exploit (optional — 0 si médailles absentes)
	if row.MedalExploitScore > 0 {
		ref := 5.0
		if avgMedalExploit != nil && *avgMedalExploit > 1e-9 {
			ref = *avgMedalExploit
		}
		valid = append(valid, entry{MetricKeyMedalExploit, sigmoidRatio(row.MedalExploitScore, ref), w[MetricKeyMedalExploit]})
	}

	// 7. offensive_conversion (optional — 0 si damage_dealt absent)
	if row.OffensiveConversion > 0 {
		ref := analysis.OffensiveConversionP80
		if avgOffConv != nil && *avgOffConv > 1e-9 {
			ref = *avgOffConv
		}
		valid = append(valid, entry{MetricKeyOffensiveConv, sigmoidRatio(row.OffensiveConversion, ref), w[MetricKeyOffensiveConv]})
	}

	// 8. defensive_resistance (optional — 0 si deaths=0)
	if row.DefensiveResistance > 0 {
		ref := analysis.DefensiveResistanceP80
		if avgDefRes != nil && *avgDefRes > 1e-9 {
			ref = *avgDefRes
		}
		valid = append(valid, entry{MetricKeyDefensiveResist, sigmoidRatio(row.DefensiveResistance, ref), w[MetricKeyDefensiveResist]})
	}

	if len(valid) == 0 {
		return 0.5, nil
	}
	totalWeight := 0.0
	for _, e := range valid {
		totalWeight += e.weight
	}
	if totalWeight < 1e-12 {
		return 0.5, nil
	}
	sum := 0.0
	breakdown := make(map[string]float64, len(valid))
	for _, e := range valid {
		sum += e.score * e.weight
		breakdown[e.key] = e.score
	}
	return ClampF(sum/totalWeight, 0.0, 1.0), breakdown
}

// ── Estimation μ individuel ─────────────────────────────────────────────────

func estimateIndividualMU(killsExpected, matchAvgKE, matchStdKE, baseMU float64) float64 {
	if matchStdKE < 1e-6 {
		return baseMU
	}
	z := (killsExpected - matchAvgKE) / matchStdKE
	return baseMU + IndividualMUAlpha*z
}

// computeEnemyStrength calcule la force estimée de l'adversaire (muOpp, sigmaOpp).
func computeEnemyStrength(enemyKEs []float64, matchAvgKE, matchStdKE, playerMU float64) (float64, float64) { //nolint:unparam // sigmaOpp actuellement fixe à DefaultOpponentSigma, extensible
	if len(enemyKEs) == 0 {
		return playerMU, DefaultOpponentSigma
	}
	muEstimates := make([]float64, 0, len(enemyKEs))
	for _, ke := range enemyKEs {
		est := estimateIndividualMU(ke, matchAvgKE, matchStdKE, playerMU)
		muEstimates = append(muEstimates, est)
	}
	avgMU := 0.0
	for _, m := range muEstimates {
		avgMU += m
	}
	avgMU /= float64(len(muEstimates))

	if len(muEstimates) <= 1 {
		return avgMU, DefaultOpponentSigma
	}
	variance := 0.0
	for _, m := range muEstimates {
		variance += (m - avgMU) * (m - avgMU)
	}
	variance /= float64(len(muEstimates))
	sigma := ClampF(math.Sqrt(variance)+MinSigma, MinSigma, DefaultOpponentSigma)
	return avgMU, sigma
}

// ── Batch compute LUSR ──────────────────────────────────────────────────────
// Structs LusrMatchData, lusrParticipant, lusrResult et loaders SQL → skill_rating_loaders.go.
// Constante LUSRMaxDelta → skill_config.go.

// BatchComputeLUSR est le wrapper public de BatchComputeLUSRWithMedals. Utilisé par
// RecomputeAfterARTRebuild (phase 4.4) et tout caller hors-package qui doit
// recompute la cascade LUSR (typiquement après un rebuild ART, un changement
// de formule, ou un backfill manuel). medal exploit map laissée nil — les
// callers qui veulent l'override exploit-aware (sync engine) utilisent encore
// le helper privé BatchComputeLUSRWithMedals.
