// Package sync — skill_rating.go : algorithme LUSR (TrueSkill 2 adapté).
//
// Portage de src/analysis/skill_rating.py + _trueskill_math.py + _composite.py.
// Calcule un rating de compétence absolu par match en traitant les matchs
// séquentiellement dans l'ordre chronologique.
package sync

import (
	"database/sql"
	"fmt"
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

// lusrMatchData contient les données d'un match pour le calcul LUSR.
type lusrMatchData struct {
	MatchID        string
	StartTime      time.Time
	PlaylistName   *string
	PairName       *string
	Outcome        *int
	Kills          float64
	Deaths         float64
	KillsExpected  float64
	DeathsExpected float64
	DamageDealt    float64
	DamageTaken    float64
	Accuracy       float64
	TeamID         *int
}

// lusrParticipant contient les données d'un participant pour le calcul LUSR.
type lusrParticipant struct {
	MatchID       string
	XUID          string
	TeamID        *int
	KillsExpected float64
}

// lusrResult contient le résultat du calcul LUSR pour un match.
type lusrResult struct {
	MatchID         string
	RatingValue     float64
	RatingDeviation float64
	PlaylistGroup   string
}

const lusrMaxDelta = 100.0 // Guard-rail : cap ±100 pts par match

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

// ── Helpers SQL ──────────────────────────────────────────────────────────────

func loadLUSRMatchData(sharedDB *sql.DB, xuid string) ([]lusrMatchData, error) {
	rows, err := sharedDB.Query(`
		SELECT
			mr.match_id, mr.start_time, mr.playlist_name, mr.pair_name,
			mp.outcome, COALESCE(mp.kills, 0), COALESCE(mp.deaths, 0),
			COALESCE(mp.kills_expected, 0), COALESCE(mp.deaths_expected, 0),
			COALESCE(mp.damage_dealt, 0), COALESCE(mp.damage_taken, 0),
			COALESCE(mp.accuracy, 0), mp.team_id
		FROM match_registry mr
		JOIN match_participants mp ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?
		  AND COALESCE(mr.is_ranked, FALSE) = FALSE
		  AND COALESCE(mr.is_firefight, FALSE) = FALSE
		  AND mr.start_time IS NOT NULL
		  AND (mr.duration_seconds IS NULL OR mr.duration_seconds >= 30)
		ORDER BY mr.start_time ASC`, xuid)
	if err != nil {
		return nil, fmt.Errorf("loadLUSRMatchData: %w", err)
	}
	defer rows.Close()

	var result []lusrMatchData
	for rows.Next() {
		var m lusrMatchData
		var outcome sql.NullInt64
		var teamID sql.NullInt64
		var plName, pairName sql.NullString
		if err := rows.Scan(
			&m.MatchID, &m.StartTime, &plName, &pairName,
			&outcome, &m.Kills, &m.Deaths,
			&m.KillsExpected, &m.DeathsExpected,
			&m.DamageDealt, &m.DamageTaken,
			&m.Accuracy, &teamID,
		); err != nil {
			continue
		}
		if plName.Valid {
			m.PlaylistName = &plName.String
		}
		if pairName.Valid {
			m.PairName = &pairName.String
		}
		if outcome.Valid {
			v := int(outcome.Int64)
			m.Outcome = &v
		}
		if teamID.Valid {
			v := int(teamID.Int64)
			m.TeamID = &v
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func loadLUSRParticipants(sharedDB *sql.DB, matchIDs []string) (map[string][]lusrParticipant, error) {
	result := make(map[string][]lusrParticipant)
	if len(matchIDs) == 0 {
		return result, nil
	}

	// Build IN clause with placeholders.
	query := "SELECT match_id, xuid, team_id, COALESCE(kills_expected, 0) FROM match_participants WHERE match_id IN ("
	args := make([]interface{}, len(matchIDs))
	for i, id := range matchIDs {
		if i > 0 {
			query += ","
		}
		query += "?"
		args[i] = id
	}
	query += ")"

	rows, err := sharedDB.Query(query, args...)
	if err != nil {
		return result, fmt.Errorf("loadLUSRParticipants: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var p lusrParticipant
		var teamID sql.NullInt64
		if err := rows.Scan(&p.MatchID, &p.XUID, &teamID, &p.KillsExpected); err != nil {
			continue
		}
		if teamID.Valid {
			v := int(teamID.Int64)
			p.TeamID = &v
		}
		result[p.MatchID] = append(result[p.MatchID], p)
	}
	return result, rows.Err()
}

func loadExistingRatingIDs(playerDB *sql.DB, ratingType string) map[string]bool {
	result := make(map[string]bool)
	rows, err := playerDB.Query("SELECT match_id FROM match_skill_rank WHERE rating_type = ?", ratingType)
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var mid string
		if rows.Scan(&mid) == nil {
			result[mid] = true
		}
	}
	return result
}

func loadExistingLUSRStates(playerDB *sql.DB) map[string]*PlayerState {
	states := make(map[string]*PlayerState)
	rows, err := playerDB.Query(`
		SELECT msr.playlist_group, msr.rating_value, msr.rating_deviation
		FROM match_skill_rank msr
		JOIN (
			SELECT playlist_group, MAX(start_time) AS max_st
			FROM match_skill_rank
			WHERE rating_type = 'LUSR'
			GROUP BY playlist_group
		) last ON msr.playlist_group = last.playlist_group
		       AND msr.start_time = last.max_st
		WHERE msr.rating_type = 'LUSR'`)
	if err != nil {
		return states
	}
	defer rows.Close()
	for rows.Next() {
		var pg string
		var rv, rd sql.NullFloat64
		if rows.Scan(&pg, &rv, &rd) != nil {
			continue
		}
		s := NewPlayerState()
		if rv.Valid {
			s.MU = rv.Float64
		}
		if rd.Valid {
			s.Sigma = rd.Float64
		}
		states[pg] = s
	}
	return states
}

func computeMatchKEStats(participants []lusrParticipant) (float64, float64) {
	var keValues []float64
	for _, p := range participants {
		if p.KillsExpected > 0 {
			keValues = append(keValues, p.KillsExpected)
		}
	}
	if len(keValues) == 0 {
		return InitialMU, 1.0
	}
	n := float64(len(keValues))
	sum := 0.0
	for _, k := range keValues {
		sum += k
	}
	avg := sum / n
	if len(keValues) < 2 {
		return avg, 1.0
	}
	variance := 0.0
	for _, k := range keValues {
		variance += (k - avg) * (k - avg)
	}
	variance /= n
	std := math.Sqrt(variance)
	if std == 0 {
		std = 1.0
	}
	return avg, std
}

func splitParticipantKEs(playerTeamID *int, participants []lusrParticipant) ([]float64, []float64) {
	var teammateKEs, enemyKEs []float64
	if playerTeamID == nil {
		for _, p := range participants {
			if p.KillsExpected > 0 {
				enemyKEs = append(enemyKEs, p.KillsExpected)
			}
		}
		return teammateKEs, enemyKEs
	}
	for _, p := range participants {
		if p.TeamID != nil && *p.TeamID == *playerTeamID {
			if p.KillsExpected > 0 {
				teammateKEs = append(teammateKEs, p.KillsExpected)
			}
		} else {
			if p.KillsExpected > 0 {
				enemyKEs = append(enemyKEs, p.KillsExpected)
			}
		}
	}
	return teammateKEs, enemyKEs
}

func upsertLUSRRatings(
	playerDB *sql.DB,
	results []lusrResult,
	existingCSR, existingLUSR map[string]bool,
	seedRatings map[string]float64,
) (int, error) {
	now := time.Now().UTC()
	prevRating := make(map[string]float64)
	for pg, r := range seedRatings {
		prevRating[pg] = r
	}

	updated := 0
	for _, r := range results {
		if existingCSR[r.MatchID] {
			continue
		}
		if existingLUSR[r.MatchID] {
			continue
		}

		ratingValue := r.RatingValue
		var delta *float64
		if prev, ok := prevRating[r.PlaylistGroup]; ok {
			rawDelta := ratingValue - prev
			if math.Abs(rawDelta) > lusrMaxDelta {
				if rawDelta > 0 {
					rawDelta = lusrMaxDelta
				} else {
					rawDelta = -lusrMaxDelta
				}
				ratingValue = prev + rawDelta
			}
			delta = &rawDelta
		}
		prevRating[r.PlaylistGroup] = ratingValue

		tier, sub := GetTierForRating(ratingValue)
		var tierName, tierFR, tierLabel *string
		if tier != nil {
			tierName = &tier.Name
			tierFR = &tier.NameFR
			label := FormatTierLabel(ratingValue)
			tierLabel = &label
		}

		_, err := playerDB.Exec(`
			INSERT INTO match_skill_rank
				(match_id, rating_type, rating_value, rating_deviation,
				 tier, tier_fr, sub_tier, tier_label,
				 rating_delta, playlist_group, created_at, updated_at)
			VALUES (?, 'LUSR', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (match_id) DO UPDATE SET
				rating_type      = 'LUSR',
				rating_value     = EXCLUDED.rating_value,
				rating_deviation = EXCLUDED.rating_deviation,
				tier             = EXCLUDED.tier,
				tier_fr          = EXCLUDED.tier_fr,
				sub_tier         = EXCLUDED.sub_tier,
				tier_label       = EXCLUDED.tier_label,
				rating_delta     = EXCLUDED.rating_delta,
				playlist_group   = EXCLUDED.playlist_group,
				updated_at       = EXCLUDED.updated_at`,
			r.MatchID, ratingValue, r.RatingDeviation,
			tierName, tierFR, sub, tierLabel,
			delta, r.PlaylistGroup, now, now)
		if err != nil {
			continue
		}
		updated++
	}
	return updated, nil
}
