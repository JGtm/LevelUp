package analysis

import (
	"math"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

const (
	initialMu                  = 1500.0
	initialSigma               = 350.0
	minSigma                   = 60.0
	maxSigma                   = 350.0
	betaTS                     = 200.0
	tauTS                      = 25.0
	kElo                       = 32.0
	drawProbability            = 0.06
	inactivitySigmaPerDay      = 1.0
	inactivityThresholdDays    = 1.0
	maxInactivityDays          = 14.0
	individualMuAlpha          = 150.0
	defaultOpponentSigma       = 150.0
	accuracyHistorySize        = 50
	minMatchesForAccuracyDelta = 5
)

var compositeWeights = map[string]float64{
	"kills_vs_expected":  0.31,
	"deaths_vs_expected": 0.28,
	"win_factor":         0.05,
	"damage_efficiency":  0.23,
	"accuracy_delta":     0.13,
}

var winFactors = map[int]float64{
	2: 1.0,
	1: 0.5,
	3: 0.0,
	4: 0.15,
}

var playlistGroupPrefixes = map[string]string{
	"ranked":           "ranked",
	"ranked challenge": "ranked",
	"big team":         "big_team_battle",
	"firefight":        "firefight",
	"custom":           "custom",
}

type playerState struct {
	mu               float64
	sigma            float64
	matchCount       int
	lastMatchTime    *time.Time
	accuracyHistory  []float64
	damageEffHistory []float64
}

func newPlayerState() *playerState {
	return &playerState{
		mu:    initialMu,
		sigma: initialSigma,
	}
}

// ComputeSkillRatingsBatch calcule les ratings LUSR pour tous les matchs d'un joueur.
// Port de compute_skill_ratings_batch (skill_rating.py).
func ComputeSkillRatingsBatch(
	rows []domain.StatsMatchRow,
	participants []domain.ParticipantRow,
) []domain.LUSRMatchRating {
	if len(rows) == 0 {
		return nil
	}

	sorted := make([]domain.StatsMatchRow, len(rows))
	copy(sorted, rows)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartTime.Before(sorted[j].StartTime)
	})

	participantsByMatch := indexParticipants(participants)
	states := make(map[string]*playerState)
	results := make([]domain.LUSRMatchRating, 0, len(sorted))

	for _, row := range sorted {
		group := resolvePlaylistGroup(row.PlaylistName, row.PairName)
		state := getOrCreateState(states, group)

		if state.lastMatchTime != nil {
			days := row.StartTime.Sub(*state.lastMatchTime).Hours() / 24.0
			state.sigma = applyInactivityDecay(state.sigma, days)
		}

		matchParts := participantsByMatch[row.MatchID]
		avgKE, _ := computeMatchKEStats(matchParts)

		teammates, enemies := splitParticipants(row.TeamID, matchParts)

		muOpp, sigmaOpp := computeEnemyStrength(enemies, avgKE, state.mu)

		teammateAvgKE := avgKEForGroup(teammates)
		enemyAvgKE := avgKEForGroup(enemies)
		avgAcc := computeRollingAvg(state.accuracyHistory)
		avgDamageEff := computeRollingAvg(state.damageEffHistory)
		composite := computeCompositeScore(row, avgAcc, teammateAvgKE, enemyAvgKE, avgDamageEff)

		state.mu, state.sigma = trueskillUpdate(
			state.mu, state.sigma,
			muOpp, sigmaOpp,
			composite, 1.0,
		)
		state.matchCount++
		t := row.StartTime
		state.lastMatchTime = &t

		if row.Accuracy != nil {
			state.accuracyHistory = appendRolling(state.accuracyHistory, *row.Accuracy, accuracyHistorySize)
		}
		if row.DamageDealt != nil && row.DamageTaken != nil {
			total := *row.DamageDealt + *row.DamageTaken
			if total > 0 {
				de := *row.DamageDealt / total
				state.damageEffHistory = appendRolling(state.damageEffHistory, de, accuracyHistorySize)
			}
		}

		results = append(results, domain.LUSRMatchRating{
			MatchID:         row.MatchID,
			RatingValue:     state.mu,
			RatingDeviation: state.sigma,
			PlaylistGroup:   group,
		})
	}
	return results
}

// trueskillUpdate met a jour mu/sigma apres un match.
func trueskillUpdate(mu, sigma, muOpp, sigmaOpp, actualScore, weightFactor float64) (float64, float64) { //nolint:unparam // muOpp réservé pour TrueSkill 2 complet
	deltaMu := kElo * (actualScore - 0.5) * weightFactor
	newMu := math.Max(minSigma, mu+deltaMu)

	c2 := 2*betaTS*betaTS + sigma*sigma + sigmaOpp*sigmaOpp
	c := math.Sqrt(c2)
	eps := drawMarginFromProbability(drawProbability, betaTS)
	w := wWin(0.0, eps/c)
	deltaSigma2 := sigma * sigma * (sigma * sigma / c2) * w * weightFactor
	newSigma := math.Max(minSigma, math.Sqrt(sigma*sigma-deltaSigma2))
	newSigma = math.Min(math.Sqrt(newSigma*newSigma+tauTS*tauTS), maxSigma)

	return newMu, newSigma
}

// applyInactivityDecay augmente sigma si le joueur est inactif.
func applyInactivityDecay(sigma, days float64) float64 {
	if days <= inactivityThresholdDays {
		return sigma
	}
	capped := math.Min(days, maxInactivityDays)
	sigma += inactivitySigmaPerDay * (capped - inactivityThresholdDays)
	return clampF(sigma, minSigma, maxSigma)
}

func drawMarginFromProbability(drawProb, betaVal float64) float64 {
	return normInvCDF((1.0+drawProb)/2.0) * math.Sqrt(2.0) * betaVal
}

func wWin(t, eps float64) float64 {
	vResult := vWin(t, eps)
	return vResult * (vResult + t - eps)
}

func vWin(t, eps float64) float64 {
	denom := normCDF(t - eps)
	if denom < 1e-10 {
		return -t + eps
	}
	return normPDF(t-eps) / denom
}

// computeCompositeScore calcule le score composite [0,1] pour un match.
func computeCompositeScore( //nolint:unparam // voir ci-dessous
	row domain.StatsMatchRow,
	avgAcc float64,
	teammateAvgKE, enemyAvgKE float64, //nolint:unparam // teammateAvgKE réservé pour future formule synergie
	avgDamageEff float64,
) float64 {
	components := make(map[string]float64)

	if row.KillsExpected != nil && *row.KillsExpected > 0 {
		kve := float64(row.Kills) / *row.KillsExpected
		components["kills_vs_expected"] = sigmoidRatio(kve, 1.0)
	}

	if row.DeathsExpected != nil && *row.DeathsExpected > 0 {
		dve := *row.DeathsExpected / math.Max(1.0, float64(row.Deaths))
		components["deaths_vs_expected"] = sigmoidRatio(dve, 1.0)
	}

	if row.Outcome != nil {
		if factor, ok := winFactors[*row.Outcome]; ok {
			components["win_factor"] = factor
		}
	}

	if row.DamageDealt != nil && row.DamageTaken != nil {
		total := *row.DamageDealt + *row.DamageTaken
		if total > 0 {
			de := *row.DamageDealt / total
			components["damage_efficiency"] = sigmoidRatio(de, math.Max(1e-9, avgDamageEff))
		}
	}

	if row.Accuracy != nil && avgAcc > 0 {
		delta := *row.Accuracy - avgAcc
		components["accuracy_delta"] = clampF(0.5+delta*2.0, 0.0, 1.0)
	}

	totalWeight := 0.0
	weightedSum := 0.0
	for key, val := range components {
		w := compositeWeights[key]
		weightedSum += val * w
		totalWeight += w
	}
	if totalWeight < 1e-9 {
		return 0.5
	}
	return clampF(weightedSum/totalWeight, 0.0, 1.0)
}

func sigmoidRatio(num, denom float64) float64 {
	if denom < 1e-9 {
		return 0.5
	}
	ratio := num / denom
	return clampF(ratio/(1.0+ratio), 0.0, 1.0)
}

func indexParticipants(participants []domain.ParticipantRow) map[string][]domain.ParticipantRow {
	idx := make(map[string][]domain.ParticipantRow, len(participants))
	for _, p := range participants {
		idx[p.MatchID] = append(idx[p.MatchID], p)
	}
	return idx
}

func computeMatchKEStats(parts []domain.ParticipantRow) (avg, stdDev float64) {
	if len(parts) == 0 {
		return 0, 0
	}
	sum := 0.0
	count := 0
	for _, p := range parts {
		if p.KillsExpected != nil {
			sum += *p.KillsExpected
			count++
		}
	}
	if count == 0 {
		return 0, 0
	}
	avg = sum / float64(count)
	variance := 0.0
	for _, p := range parts {
		if p.KillsExpected != nil {
			d := *p.KillsExpected - avg
			variance += d * d
		}
	}
	stdDev = math.Sqrt(variance / float64(count))
	return avg, stdDev
}

func splitParticipants(
	playerTeamID *int,
	parts []domain.ParticipantRow,
) (teammates, enemies []domain.ParticipantRow) {
	for _, p := range parts {
		if playerTeamID != nil && p.TeamID != nil && *p.TeamID == *playerTeamID {
			teammates = append(teammates, p)
		} else {
			enemies = append(enemies, p)
		}
	}
	return teammates, enemies
}

func computeEnemyStrength(
	enemies []domain.ParticipantRow,
	avgKE, playerMu float64,
) (muOpp, sigmaOpp float64) { //nolint:unparam // sigmaOpp actuellement fixe à defaultOpponentSigma, extensible
	if len(enemies) == 0 {
		return playerMu, defaultOpponentSigma
	}
	enemyAvgKE := avgKEForGroup(enemies)
	if avgKE > 0 && enemyAvgKE > 0 {
		ratio := enemyAvgKE / avgKE
		muOpp = playerMu * clampF(ratio, 0.5, 2.0)
	} else {
		muOpp = playerMu
	}
	return muOpp, defaultOpponentSigma
}

func avgKEForGroup(parts []domain.ParticipantRow) float64 {
	if len(parts) == 0 {
		return 0
	}
	sum := 0.0
	count := 0
	for _, p := range parts {
		if p.KillsExpected != nil {
			sum += *p.KillsExpected
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func resolvePlaylistGroup(playlistName, pairName string) string {
	lower := strings.ToLower(playlistName + " " + pairName)
	for prefix, group := range playlistGroupPrefixes {
		if strings.Contains(lower, prefix) {
			return group
		}
	}
	return "arena"
}

func getOrCreateState(states map[string]*playerState, group string) *playerState {
	if s, ok := states[group]; ok {
		return s
	}
	s := newPlayerState()
	states[group] = s
	return s
}

func appendRolling(slice []float64, value float64, maxSize int) []float64 {
	slice = append(slice, value)
	if len(slice) > maxSize {
		slice = slice[len(slice)-maxSize:]
	}
	return slice
}

func computeRollingAvg(slice []float64) float64 {
	if len(slice) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range slice {
		sum += v
	}
	return sum / float64(len(slice))
}

func normCDF(x float64) float64 {
	if x < -8.0 {
		return 0.0
	}
	if x > 8.0 {
		return 1.0
	}
	neg := x < 0
	if neg {
		x = -x
	}
	result := 1.0 / (1.0 + 0.2316419*x)
	poly := result * (0.319381530 + result*(-0.356563782+
		result*(1.781477937+result*(-1.821255978+result*1.330274429))))
	cdf := 1.0 - normPDF(x)*poly
	if neg {
		return 1.0 - cdf
	}
	return cdf
}

func normPDF(x float64) float64 {
	return math.Exp(-0.5*x*x) / math.Sqrt(2.0*math.Pi)
}

func normInvCDF(p float64) float64 {
	if p <= 0.0 || p >= 1.0 {
		return 0.0
	}
	if p < 0.5 {
		return -rationalApprox(math.Sqrt(-2.0 * math.Log(p)))
	}
	return rationalApprox(math.Sqrt(-2.0 * math.Log(1.0-p)))
}

func rationalApprox(t float64) float64 {
	c := []float64{2.515517, 0.802853, 0.010328}
	d := []float64{1.432788, 0.189269, 0.001308}
	num := c[0] + t*(c[1]+t*c[2])
	den := 1.0 + t*(d[0]+t*(d[1]+t*d[2]))
	return t - num/den
}
