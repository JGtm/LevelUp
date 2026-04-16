// Package sync — skill_rating.go : algorithme LUSR (TrueSkill 2 adapté).
//
// Portage de src/analysis/skill_rating.py + _trueskill_math.py + _composite.py.
// Calcule un rating de compétence absolu par match en traitant les matchs
// séquentiellement dans l'ordre chronologique.
package sync

import (
	"database/sql"
	"math"
	"time"
)

// ── PlayerState — état TrueSkill 2 entre deux matchs ────────────────────────

// PlayerState contient l'état mu/sigma d'un joueur pour un playlist_group.
type PlayerState struct {
	MU               float64
	Sigma            float64
	MatchCount       int
	LastMatchTime    *time.Time
	AccuracyHistory  []float64
	DamageEffHistory []float64
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
func trueskillUpdate(mu, sigma, muOpp, sigmaOpp, actualScore, weightFactor float64) (float64, float64) {
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
	Kills          float64
	Deaths         float64
	KillsExpected  float64
	DeathsExpected float64
	Outcome        *int
	DamageDealt    float64
	DamageTaken    float64
	Accuracy       float64
}

// computeCompositeScore calcule le score composite [0,1] d'un match.
func computeCompositeScore(
	row *compositeMatchRow,
	avgAccuracy *float64,
	teammateAvgKE *float64,
	avgDamageEff *float64,
) float64 {
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
		valid = append(valid, entry{"kills_vs_expected", score, w["kills_vs_expected"]})
	}

	// 2. deaths_vs_expected (inversé)
	if row.DeathsExpected > 0 {
		score := sigmoidRatio(row.DeathsExpected, math.Max(1.0, row.Deaths))
		valid = append(valid, entry{"deaths_vs_expected", score, w["deaths_vs_expected"]})
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
		valid = append(valid, entry{"win_factor", winScore, w["win_factor"]})
	}

	// 4. damage_efficiency
	total := row.DamageDealt + row.DamageTaken
	if total > 0 {
		rawEff := clampF(row.DamageDealt/total, 0.0, 1.0)
		scoreEff := rawEff
		if avgDamageEff != nil && *avgDamageEff > 0 {
			scoreEff = sigmoidRatio(rawEff, *avgDamageEff)
		}
		valid = append(valid, entry{"damage_efficiency", scoreEff, w["damage_efficiency"]})
	}

	// 5. accuracy_delta
	if row.Accuracy > 0 && avgAccuracy != nil && *avgAccuracy > 0 {
		score := sigmoidRatio(row.Accuracy, *avgAccuracy)
		valid = append(valid, entry{"accuracy_delta", score, w["accuracy_delta"]})
	}

	if len(valid) == 0 {
		return 0.5
	}
	totalWeight := 0.0
	for _, e := range valid {
		totalWeight += e.weight
	}
	if totalWeight < 1e-12 {
		return 0.5
	}
	sum := 0.0
	for _, e := range valid {
		sum += e.score * e.weight
	}
	return clampF(sum/totalWeight, 0.0, 1.0)
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
func computeEnemyStrength(enemyKEs []float64, matchAvgKE, matchStdKE, playerMU float64) (float64, float64) {
	if len(enemyKEs) == 0 {
		return playerMU, DefaultOpponentSigma
	}
	var muEstimates []float64
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
// Retourne le nombre de matchs mis à jour.
func batchComputeLUSR(playerDB, sharedDB *sql.DB, xuid string) (int, error) {
	// 1. Charger les matchs non classés, non-firefight, triés chronologiquement.
	matches, err := loadLUSRMatchData(sharedDB, xuid)
	if err != nil {
		return 0, err
	}
	if len(matches) == 0 {
		return 0, nil
	}

	// 2. Charger les participants pour calcul de force adverse.
	matchIDs := make([]string, len(matches))
	for i, m := range matches {
		matchIDs[i] = m.MatchID
	}
	participantsByMatch, err := loadLUSRParticipants(sharedDB, matchIDs)
	if err != nil {
		return 0, err
	}

	// 3. Charger les matchs déjà classés CSR (protéger).
	existingCSR := loadExistingRatingIDs(playerDB, "CSR")
	existingLUSR := loadExistingRatingIDs(playerDB, "LUSR")

	// 4. Charger les états existants (mode incrémental).
	states := loadExistingLUSRStates(playerDB)
	seedRatings := make(map[string]float64)
	for pg, st := range states {
		seedRatings[pg] = st.MU
	}

	// 5. Filtrer les matchs déjà calculés.
	var newMatches []lusrMatchData
	for _, m := range matches {
		if existingCSR[m.MatchID] || existingLUSR[m.MatchID] {
			continue
		}
		newMatches = append(newMatches, m)
	}
	if len(newMatches) == 0 {
		return 0, nil
	}

	// 6. Calculer les ratings via TrueSkill 2 séquentiel.
	results := computeSkillRatingsBatch(newMatches, participantsByMatch, states)
	if len(results) == 0 {
		return 0, nil
	}

	// 7. Écrire les résultats.
	return upsertLUSRRatings(playerDB, results, existingCSR, existingLUSR, seedRatings)
}

// computeSkillRatingsBatch calcule mu/sigma pour chaque match séquentiellement.
func computeSkillRatingsBatch(
	matches []lusrMatchData,
	participantsByMatch map[string][]lusrParticipant,
	states map[string]*PlayerState,
) []lusrResult {
	if states == nil {
		states = make(map[string]*PlayerState)
	}
	var results []lusrResult

	for _, match := range matches {
		group := GetPlaylistGroup(match.PlaylistName, match.PairName)
		pgCfg, ok := PlaylistGroups[group]
		weightFactor := 1.0
		if ok {
			weightFactor = pgCfg.WeightFactor
		}

		state, exists := states[group]
		if !exists {
			state = NewPlayerState()
			states[group] = state
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

		// Précision moyenne historique
		var avgAcc *float64
		if len(state.AccuracyHistory) >= MinMatchesForAccuracyDelta {
			sum := 0.0
			for _, a := range state.AccuracyHistory {
				sum += a
			}
			avg := sum / float64(len(state.AccuracyHistory))
			avgAcc = &avg
		}

		// Efficacité dégâts moyenne historique
		var avgDmgEff *float64
		if len(state.DamageEffHistory) >= MinMatchesForAccuracyDelta {
			sum := 0.0
			for _, d := range state.DamageEffHistory {
				sum += d
			}
			avg := sum / float64(len(state.DamageEffHistory))
			avgDmgEff = &avg
		}

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
				PlaylistGroup:   group,
			})
			continue
		}

		// Score composite
		cRow := &compositeMatchRow{
			Kills:          match.Kills,
			Deaths:         match.Deaths,
			KillsExpected:  match.KillsExpected,
			DeathsExpected: match.DeathsExpected,
			Outcome:        match.Outcome,
			DamageDealt:    match.DamageDealt,
			DamageTaken:    match.DamageTaken,
			Accuracy:       match.Accuracy,
		}
		composite := computeCompositeScore(cRow, avgAcc, teammateAvgKE, avgDmgEff)

		// Update TrueSkill
		newMU, newSigma := trueskillUpdate(state.MU, state.Sigma, muOpp, sigmaOpp, composite, weightFactor)
		state.MU = newMU
		state.Sigma = newSigma
		state.MatchCount++
		t := match.StartTime
		state.LastMatchTime = &t

		// Historique précision
		if match.Accuracy > 0 {
			state.AccuracyHistory = append(state.AccuracyHistory, match.Accuracy)
			if len(state.AccuracyHistory) > AccuracyHistorySize {
				state.AccuracyHistory = state.AccuracyHistory[len(state.AccuracyHistory)-AccuracyHistorySize:]
			}
		}

		// Historique efficacité dégâts
		totalDmg := match.DamageDealt + match.DamageTaken
		if totalDmg > 0 {
			eff := clampF(match.DamageDealt/totalDmg, 0.0, 1.0)
			state.DamageEffHistory = append(state.DamageEffHistory, eff)
			if len(state.DamageEffHistory) > AccuracyHistorySize {
				state.DamageEffHistory = state.DamageEffHistory[len(state.DamageEffHistory)-AccuracyHistorySize:]
			}
		}

		results = append(results, lusrResult{
			MatchID:         match.MatchID,
			RatingValue:     math.Round(state.MU*10) / 10,
			RatingDeviation: math.Round(state.Sigma*10) / 10,
			PlaylistGroup:   group,
		})
	}
	return results
}
