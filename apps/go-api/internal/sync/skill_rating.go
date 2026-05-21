// Package sync — skill_rating.go : algorithme LUSR (TrueSkill 2 adapté).
//
// Portage de src/analysis/skill_rating.py + _trueskill_math.py + _composite.py.
// Calcule un rating de compétence absolu par match en traitant les matchs
// séquentiellement dans l'ordre chronologique.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"time"

	"levelup/go-api/internal/analysis"
)

// ── PlayerState — état TrueSkill 2 entre deux matchs ────────────────────────

// PlayerState contient l'état mu/sigma d'un joueur pour un playlist_group.
type PlayerState struct {
	MU                   float64
	Sigma                float64
	MatchCount           int
	LastMatchTime        *time.Time
	AccuracyHistory      []float64
	DamageEffHistory     []float64
	MedalExploitHistory  []float64
	OffConversionHistory []float64
	DefResistanceHistory []float64
}

// NewPlayerState crée un état initial.
func NewPlayerState() *PlayerState {
	return &PlayerState{MU: InitialMU, Sigma: InitialSigma}
}

// ── Fonctions gaussiennes TrueSkill 2 ──────────────────────────────────────

func standardNormalPDF(x float64) float64 {
	return math.Exp(-0.5*x*x) / math.Sqrt(2.0*math.Pi)
}

func standardNormalCDF(x float64) float64 {
	return 0.5 * (1.0 + math.Erf(x/math.Sqrt(2.0)))
}

func vWin(t, eps float64) float64 {
	x := t - eps
	denom := standardNormalCDF(x)
	if denom < 1e-10 {
		return -x
	}
	return standardNormalPDF(x) / denom
}

func wWin(t, eps float64) float64 {
	v := vWin(t, eps)
	x := t - eps
	return v * (v + x)
}

// ── TrueSkill update ────────────────────────────────────────────────────────

// trueskillUpdate met à jour (mu, sigma) après un match.
// Mu : formule Elo-style continue (K_ELO × (score - 0.5) × wf).
// Sigma : réduction TrueSkill à t=0.
func trueskillUpdate(mu, sigma, muOpp, sigmaOpp, actualScore, weightFactor float64) (float64, float64) { //nolint:unparam // muOpp réservé pour TrueSkill 2 complet
	deltaMU := KElo * (actualScore - 0.5) * weightFactor
	newMU := math.Max(MinRating, mu+deltaMU)

	c2 := 2.0*Beta*Beta + sigma*sigma + sigmaOpp*sigmaOpp
	c := math.Sqrt(c2)
	eps := drawMargin(Beta)

	sigma2 := sigma * sigma
	w := wWin(0.0, eps/c)
	deltaSigma2 := sigma2 * (sigma2 / c2) * w * weightFactor

	newSigma2 := math.Max(MinSigma*MinSigma, sigma2-deltaSigma2)
	newSigma := math.Sqrt(newSigma2)
	newSigma = math.Min(math.Sqrt(newSigma*newSigma+Tau*Tau), MaxSigma)

	return newMU, newSigma
}

// applyInactivityDecay augmente sigma proportionnellement à l'inactivité.
func applyInactivityDecay(sigma, daysInactive float64) float64 {
	capped := math.Min(daysInactive, float64(MaxInactivityDays))
	if capped <= InactivityThresholdDay {
		return sigma
	}
	added := InactivitySigmaPerDay * (capped - InactivityThresholdDay)
	return clampF(sigma+added, MinSigma, MaxSigma)
}

// ── Score composite ─────────────────────────────────────────────────────────

// compositeMatchRow contient les champs nécessaires au score composite.
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
// renormalisés.
//
// (teammateAvgKE = synergie escouade ; avgMedalExploit = bonus exploit ; avgOffConv
// = offensive conversion ; avgDefRes = defensive resistance). Aujourd'hui tous nil,
// activés via PerfTier roadmap. Signature stable pour éviter les refactos cascade.
//
//nolint:unparam // 4 params réservés pour futures composantes du score composite
func computeCompositeScore(
	row *compositeMatchRow,
	avgAccuracy *float64,
	teammateAvgKE *float64,
	avgDamageEff *float64,
	avgMedalExploit *float64,
	avgOffConv *float64,
	avgDefRes *float64,
) float64 {
	composite, _ := computeCompositeScoreWithBreakdown(row, avgAccuracy, teammateAvgKE, avgDamageEff, avgMedalExploit, avgOffConv, avgDefRes)
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
	teammateAvgKE *float64,
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
	if row.KillsExpected > 0 {
		score := sigmoidRatio(row.Kills, row.KillsExpected)
		if teammateAvgKE != nil && *teammateAvgKE > 0 && row.KillsExpected > 0 {
			carryRatio := row.KillsExpected / *teammateAvgKE
			carryAdj := clampF(carryRatio, 0.5, 2.0)
			score = clampF(score*(1.0/carryAdj)+0.5*(1.0-1.0/carryAdj), 0.0, 1.0)
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
		case 2:
			winScore = 1.0
		case 1:
			winScore = 0.5
		case 3:
			winScore = 0.0
		case 4:
			winScore = 0.15
		default:
			winScore = 0.5
		}
		valid = append(valid, entry{MetricKeyWinFactor, winScore, w[MetricKeyWinFactor]})
	}

	// 4. damage_efficiency
	total := row.DamageDealt + row.DamageTaken
	if total > 0 {
		rawEff := clampF(row.DamageDealt/total, 0.0, 1.0)
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
	return clampF(sum/totalWeight, 0.0, 1.0), breakdown
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
	sigma := clampF(math.Sqrt(variance)+MinSigma, MinSigma, DefaultOpponentSigma)
	return avgMU, sigma
}

// ── Batch compute LUSR ──────────────────────────────────────────────────────
// Structs lusrMatchData, lusrParticipant, lusrResult et loaders SQL → skill_rating_loaders.go.
// Constante LUSRMaxDelta → skill_config.go.

// batchComputeLUSR calcule le LUSR pour tous les matchs non classés.
// medalExploitByMatch : match_id → score brut d'exploit médailles (nil = pas de données).
// force : si true, recalcule même les matchs déjà présents (utile après changement de formule).
// Retourne le nombre de matchs mis à jour.
func batchComputeLUSR(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string, medalExploitByMatch map[string]float64, force bool) (int, error) {
	// 1. Charger les matchs non classés, non-firefight, triés chronologiquement.
	matches, err := loadLUSRMatchData(ctx, sharedDB, xuid)
	if err != nil {
		return 0, err
	}
	if len(matches) == 0 {
		return 0, nil
	}

	// Filtrer les matchs marqués `is_excluded` côté playerDB : ils ne doivent ni
	// alimenter la cascade TrueSkill ni recevoir de rating LUSR.
	excluded, err := loadExcludedMatchIDs(ctx, playerDB)
	if err != nil {
		return 0, fmt.Errorf("batchComputeLUSR: %w", err)
	}
	if len(excluded) > 0 {
		before := len(matches)
		filtered := matches[:0]
		for _, m := range matches {
			if excluded[m.MatchID] {
				continue
			}
			filtered = append(filtered, m)
		}
		matches = filtered
		slog.Debug("batchComputeLUSR: matchs exclus filtrés",
			"xuid", xuid, "filtered", before-len(matches), "remaining", len(matches))
		if len(matches) == 0 {
			return 0, nil
		}
	}

	// 2. Charger les participants pour calcul de force adverse.
	matchIDs := make([]string, len(matches))
	for i, m := range matches {
		matchIDs[i] = m.MatchID
	}
	participantsByMatch, err := loadLUSRParticipants(ctx, sharedDB, matchIDs)
	if err != nil {
		return 0, err
	}

	// 3. Charger les matchs déjà classés CSR (protéger) et LUSR (pour mode incrémental).
	existingCSR, err := loadExistingRatingIDs(ctx, playerDB, "CSR")
	if err != nil {
		return 0, fmt.Errorf("batchComputeLUSR: %w", err)
	}
	existingLUSR, err := loadExistingRatingIDs(ctx, playerDB, "LUSR")
	if err != nil {
		return 0, fmt.Errorf("batchComputeLUSR: %w", err)
	}
	// En mode force, on ne filtre pas les LUSR existants à l'upsert — ON CONFLICT DO UPDATE écrase.
	existingLUSRForUpsert := existingLUSR
	if force {
		existingLUSRForUpsert = make(map[string]bool)
	}

	// 4. En mode force : recalcul depuis zéro (état vierge, pas de seed).
	//    En mode incrémental : reprendre depuis le dernier état persisté.
	var states map[string]*PlayerState
	seedRatings := make(map[string]float64)
	if force {
		states = make(map[string]*PlayerState)
	} else {
		states = loadExistingLUSRStates(ctx, playerDB)
		for pg, st := range states {
			seedRatings[pg] = st.MU
		}
	}

	// 5. En mode normal : filtrer les matchs déjà calculés.
	//    En mode force : tout recalculer (upsertLUSRRatings écrase via ON CONFLICT).
	toProcess := matches
	if !force {
		toProcess = make([]lusrMatchData, 0, len(matches))
		for _, m := range matches {
			if existingCSR[m.MatchID] || existingLUSR[m.MatchID] {
				continue
			}
			toProcess = append(toProcess, m)
		}
		if len(toProcess) == 0 {
			return 0, nil
		}
	}

	// 6. Calculer les ratings via TrueSkill 2 séquentiel.
	results := computeSkillRatingsBatch(toProcess, participantsByMatch, states, medalExploitByMatch)
	if len(results) == 0 {
		return 0, nil
	}

	// 7. Écrire les résultats.
	return upsertLUSRRatings(ctx, playerDB, results, existingCSR, existingLUSRForUpsert, seedRatings)
}

// computeSkillRatingsBatch calcule mu/sigma pour chaque match séquentiellement.
// medalExploitByMatch : map optionnelle match_id → score brut médailles.
func computeSkillRatingsBatch(
	matches []lusrMatchData,
	participantsByMatch map[string][]lusrParticipant,
	states map[string]*PlayerState,
	medalExploitByMatch map[string]float64,
) []lusrResult {
	if states == nil {
		states = make(map[string]*PlayerState)
	}
	results := make([]lusrResult, 0, len(matches))

	for _, match := range matches {
		pairName := ""
		if match.PairName != nil {
			pairName = *match.PairName
		}
		chain := GetLUSRChain(pairName)
		if chain == "" {
			continue // exclu : Ranked (→ CSR) ou Firefight (→ PvE)
		}

		state, exists := states[chain]
		if !exists {
			state = NewPlayerState()
			states[chain] = state
		}

		// Inactivité decay
		if state.LastMatchTime != nil {
			delta := match.StartTime.Sub(*state.LastMatchTime)
			days := delta.Hours() / 24.0
			state.Sigma = applyInactivityDecay(state.Sigma, days)
		}

		// Participants du match
		allParts := participantsByMatch[match.MatchID]
		matchAvgKE, matchStdKE := computeMatchKEStats(allParts)

		// Séparer coéquipiers et adversaires
		teammateKEs, enemyKEs := splitParticipantKEs(match.TeamID, allParts)

		// Force adversaire (ancrée sur state.MU)
		muOpp, sigmaOpp := computeEnemyStrength(enemyKEs, matchAvgKE, matchStdKE, state.MU)

		// Moyennes historiques (nil = pas assez de données → composante ignorée)
		avgAcc := rollingAvgPtr(state.AccuracyHistory)
		avgDmgEff := rollingAvgPtr(state.DamageEffHistory)
		avgMedalExploit := rollingAvgPtr(state.MedalExploitHistory)
		avgOffConv := rollingAvgPtr(state.OffConversionHistory)
		avgDefRes := rollingAvgPtr(state.DefResistanceHistory)

		// Teammate avg KE
		var teammateAvgKE *float64
		if len(teammateKEs) > 0 {
			sum := 0.0
			for _, ke := range teammateKEs {
				sum += ke
			}
			avg := sum / float64(len(teammateKEs))
			teammateAvgKE = &avg
		}

		// Guard : match sans outcome
		if match.Outcome == nil {
			state.MatchCount++
			t := match.StartTime
			state.LastMatchTime = &t
			results = append(results, lusrResult{
				MatchID:         match.MatchID,
				RatingValue:     math.Round(state.MU*10) / 10,
				RatingDeviation: math.Round(state.Sigma*10) / 10,
				PlaylistGroup:   chain,
			})
			continue
		}

		// Calcul des métriques dérivées
		offConv, defRes := computeCombatYield(match)
		medalScore := medalExploitByMatch[match.MatchID]

		// Score composite
		cRow := &compositeMatchRow{
			Kills:               match.Kills,
			Deaths:              match.Deaths,
			Assists:             match.Assists,
			KillsExpected:       match.KillsExpected,
			DeathsExpected:      match.DeathsExpected,
			Outcome:             match.Outcome,
			DamageDealt:         match.DamageDealt,
			DamageTaken:         match.DamageTaken,
			Accuracy:            match.Accuracy,
			MedalExploitScore:   medalScore,
			OffensiveConversion: offConv,
			DefensiveResistance: defRes,
		}
		composite, breakdown := computeCompositeScoreWithBreakdown(cRow, avgAcc, teammateAvgKE, avgDmgEff, avgMedalExploit, avgOffConv, avgDefRes)

		// Update TrueSkill
		newMU, newSigma := trueskillUpdate(state.MU, state.Sigma, muOpp, sigmaOpp, composite, 1.0)
		state.MU = newMU
		state.Sigma = newSigma
		state.MatchCount++
		t := match.StartTime
		state.LastMatchTime = &t

		// Mise à jour des historiques glissants
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

		results = append(results, lusrResult{
			MatchID:         match.MatchID,
			RatingValue:     math.Round(state.MU*10) / 10,
			RatingDeviation: math.Round(state.Sigma*10) / 10,
			PlaylistGroup:   chain,
			Components:      breakdown,
		})
	}
	return results
}

// rollingAvgPtr retourne la moyenne d'une slice si elle a au moins
// MinMatchesForAccuracyDelta éléments, nil sinon. Le seuil est toujours
// MinMatchesForAccuracyDelta (constant unique) — inline pour eviter le bruit
// unparam.
func rollingAvgPtr(hist []float64) *float64 {
	if len(hist) < MinMatchesForAccuracyDelta {
		return nil
	}
	sum := 0.0
	for _, v := range hist {
		sum += v
	}
	avg := sum / float64(len(hist))
	return &avg
}

// appendToHistory ajoute v à *hist et tronque a AccuracyHistorySize elements.
// La taille max est toujours AccuracyHistorySize (constant unique) — inline
// pour eviter le bruit unparam.
func appendToHistory(hist *[]float64, v float64) {
	*hist = append(*hist, v)
	if len(*hist) > AccuracyHistorySize {
		*hist = (*hist)[len(*hist)-AccuracyHistorySize:]
	}
}

// computeCombatYield calcule offensive_conversion et defensive_resistance depuis un match.
func computeCombatYield(m lusrMatchData) (offConv, defRes float64) {
	if m.DamageDealt > 0 {
		offConv = 225.0 * (m.Kills + m.Assists/3.0) / m.DamageDealt
	}
	if m.DamageTaken > 0 && m.Deaths > 0 {
		defRes = m.DamageTaken / (225.0 * m.Deaths)
	}
	return
}
