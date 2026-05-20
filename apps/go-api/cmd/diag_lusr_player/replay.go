//go:build cgo

package main

import (
	"math"
	"time"

	"levelup/go-api/internal/analysis"
	lusync "levelup/go-api/internal/sync"
)

// replayRow capture l'état d'un match pendant le replay TrueSkill (un joueur, un mode).
type replayRow struct {
	matchID       string
	startTime     time.Time
	pairName      string
	chain         string
	teamID        int // -1 si NULL
	teamIDValid   bool
	kills         float64
	deaths        float64
	killsExpected float64
	deathsExp     float64
	teammateAvgKE float64
	teammateCount int
	enemyCount    int
	outcome       int // -1 si NULL
	scoreKVE      float64
	scoreKVERaw   float64
	composite     float64
	breakdown     map[string]float64
	muBefore      float64
	muAfter       float64
	sigmaBefore   float64
	sigmaAfter    float64
	deltaMU       float64
}

// playerState — équivalent local de sync.PlayerState (champs privés).
type playerState struct {
	mu                  float64
	sigma               float64
	matchCount          int
	lastMatchTime       *time.Time
	accuracyHistory     []float64
	damageEffHistory    []float64
	medalExploitHistory []float64
	offConvHistory      []float64
	defResHistory       []float64
}

func newPlayerState() *playerState {
	return &playerState{mu: lusync.InitialMU, sigma: lusync.InitialSigma}
}

// replay rejoue séquentiellement le LUSR sur tous les matchs du joueur.
// useCarryAdj=true → comportement prod actuel. false → score kve nu.
// medal_exploit_score est ignoré (poids 4 %, négligeable pour le diag).
func replay(matches []matchData, partsByMatch map[string][]participantData, useCarryAdj bool) []replayRow {
	states := make(map[string]*playerState)
	rows := make([]replayRow, 0, len(matches))

	for _, m := range matches {
		chain := lusync.GetLUSRChain(m.pairName)
		if chain == "" {
			continue
		}
		st, ok := states[chain]
		if !ok {
			st = newPlayerState()
			states[chain] = st
		}
		if st.lastMatchTime != nil {
			days := m.startTime.Sub(*st.lastMatchTime).Hours() / 24.0
			st.sigma = applyInactivityDecay(st.sigma, days)
		}

		parts := partsByMatch[m.matchID]
		matchAvgKE, matchStdKE := computeMatchKEStats(parts)
		teammateKEs, enemyKEs := splitKEs(m.teamID, parts)
		muOpp, sigmaOpp := computeEnemyStrength(enemyKEs, matchAvgKE, matchStdKE, st.mu)

		var teammateAvg *float64
		if len(teammateKEs) > 0 {
			s := 0.0
			for _, ke := range teammateKEs {
				s += ke
			}
			a := s / float64(len(teammateKEs))
			teammateAvg = &a
		}

		muBefore, sigmaBefore := st.mu, st.sigma

		scoreKVERaw, composite, breakdown := computeCompositeFull(m, st, teammateAvg, useCarryAdj)

		newMU, newSigma := trueskillUpdate(st.mu, st.sigma, muOpp, sigmaOpp, composite, 1.0)
		deltaMU := newMU - muBefore
		st.mu = newMU
		st.sigma = newSigma
		st.matchCount++
		t := m.startTime
		st.lastMatchTime = &t

		updateHistories(st, m)

		tmAvg := 0.0
		if teammateAvg != nil {
			tmAvg = *teammateAvg
		}
		// scoreKVE après carry-adj (ce qui rentre dans le composite)
		scoreKVE := scoreKVERaw
		if useCarryAdj && teammateAvg != nil && *teammateAvg > 0 && m.killsExpected > 0 {
			carryAdj := clampF(m.killsExpected/(*teammateAvg), 0.5, 2.0)
			scoreKVE = clampF(scoreKVERaw*(1.0/carryAdj)+0.5*(1.0-1.0/carryAdj), 0.0, 1.0)
		}

		row := replayRow{
			matchID:       m.matchID,
			startTime:     m.startTime,
			pairName:      m.pairName,
			chain:         chain,
			kills:         m.kills,
			deaths:        m.deaths,
			killsExpected: m.killsExpected,
			deathsExp:     m.deathsExpected,
			teammateAvgKE: tmAvg,
			teammateCount: len(teammateKEs),
			enemyCount:    len(enemyKEs),
			outcome:       -1,
			scoreKVE:      scoreKVE,
			scoreKVERaw:   scoreKVERaw,
			composite:     composite,
			breakdown:     breakdown,
			muBefore:      muBefore,
			muAfter:       newMU,
			sigmaBefore:   sigmaBefore,
			sigmaAfter:    newSigma,
			deltaMU:       deltaMU,
		}
		if m.teamID != nil {
			row.teamID = *m.teamID
			row.teamIDValid = true
		}
		if m.outcome != nil {
			row.outcome = *m.outcome
		}
		rows = append(rows, row)
	}
	return rows
}

// updateHistories alimente les rolling histories (identiques entre old/new).
func updateHistories(st *playerState, m matchData) {
	appendHist(&st.accuracyHistory, m.accuracy)
	total := m.damageDealt + m.damageTaken
	if total > 0 {
		appendHist(&st.damageEffHistory, clampF(m.damageDealt/total, 0, 1))
	}
	if m.damageDealt > 0 {
		offConv := 225.0 * (m.kills + m.assists/3.0) / m.damageDealt
		if offConv > 0 {
			appendHist(&st.offConvHistory, offConv)
		}
	}
	if m.damageTaken > 0 && m.deaths > 0 {
		defRes := m.damageTaken / (225.0 * m.deaths)
		if defRes > 0 {
			appendHist(&st.defResHistory, defRes)
		}
	}
}

// computeCompositeFull retourne (scoreKVE_brut, composite_final, breakdown).
// Le breakdown contient les valeurs des 8 composantes calculées (clés = noms
// canoniques), utile pour diagnostiquer ce qui tire le composite vers le bas.
func computeCompositeFull(m matchData, st *playerState, teammateAvgKE *float64, useCarryAdj bool) (float64, float64, map[string]float64) {
	w := lusync.CompositeWeights
	type entry struct {
		key           string
		score, weight float64
	}
	var valid []entry
	avgAcc := rollingAvg(st.accuracyHistory)
	avgDmgEff := rollingAvg(st.damageEffHistory)
	avgOffConv := rollingAvg(st.offConvHistory)
	avgDefRes := rollingAvg(st.defResHistory)

	scoreKVERaw := 0.0
	if m.killsExpected > 0 {
		scoreKVERaw = sigmoidRatio(m.kills, m.killsExpected)
		score := scoreKVERaw
		if useCarryAdj && teammateAvgKE != nil && *teammateAvgKE > 0 {
			carryAdj := clampF(m.killsExpected/(*teammateAvgKE), 0.5, 2.0)
			score = clampF(score*(1.0/carryAdj)+0.5*(1.0-1.0/carryAdj), 0.0, 1.0)
		}
		valid = append(valid, entry{lusync.MetricKeyKillsVsExpected, score, w[lusync.MetricKeyKillsVsExpected]})
	}
	if m.deathsExpected > 0 {
		s := sigmoidRatio(m.deathsExpected, math.Max(1.0, m.deaths))
		valid = append(valid, entry{lusync.MetricKeyDeathsVsExpected, s, w[lusync.MetricKeyDeathsVsExpected]})
	}
	if m.outcome != nil {
		var s float64
		switch *m.outcome {
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
		valid = append(valid, entry{lusync.MetricKeyWinFactor, s, w[lusync.MetricKeyWinFactor]})
	}
	total := m.damageDealt + m.damageTaken
	if total > 0 {
		raw := clampF(m.damageDealt/total, 0.0, 1.0)
		s := raw
		if avgDmgEff != nil && *avgDmgEff > 0 {
			s = sigmoidRatio(raw, *avgDmgEff)
		}
		valid = append(valid, entry{lusync.MetricKeyDamageEfficiency, s, w[lusync.MetricKeyDamageEfficiency]})
	}
	if m.accuracy > 0 && avgAcc != nil && *avgAcc > 0 {
		s := sigmoidRatio(m.accuracy, *avgAcc)
		valid = append(valid, entry{lusync.MetricKeyAccuracyDelta, s, w[lusync.MetricKeyAccuracyDelta]})
	}
	if m.damageDealt > 0 {
		offConv := 225.0 * (m.kills + m.assists/3.0) / m.damageDealt
		if offConv > 0 {
			ref := analysis.OffensiveConversionP80
			if avgOffConv != nil && *avgOffConv > 1e-9 {
				ref = *avgOffConv
			}
			valid = append(valid, entry{lusync.MetricKeyOffensiveConv, sigmoidRatio(offConv, ref), w[lusync.MetricKeyOffensiveConv]})
		}
	}
	if m.damageTaken > 0 && m.deaths > 0 {
		defRes := m.damageTaken / (225.0 * m.deaths)
		if defRes > 0 {
			ref := analysis.DefensiveResistanceP80
			if avgDefRes != nil && *avgDefRes > 1e-9 {
				ref = *avgDefRes
			}
			valid = append(valid, entry{lusync.MetricKeyDefensiveResist, sigmoidRatio(defRes, ref), w[lusync.MetricKeyDefensiveResist]})
		}
	}

	breakdown := make(map[string]float64, len(valid))
	for _, e := range valid {
		breakdown[e.key] = e.score
	}
	if len(valid) == 0 {
		return scoreKVERaw, 0.5, breakdown
	}
	tw, sum := 0.0, 0.0
	for _, e := range valid {
		tw += e.weight
		sum += e.score * e.weight
	}
	if tw < 1e-12 {
		return scoreKVERaw, 0.5, breakdown
	}
	return scoreKVERaw, clampF(sum/tw, 0.0, 1.0), breakdown
}

func rollingAvg(h []float64) *float64 {
	if len(h) < lusync.MinMatchesForAccuracyDelta {
		return nil
	}
	s := 0.0
	for _, v := range h {
		s += v
	}
	a := s / float64(len(h))
	return &a
}

func appendHist(h *[]float64, v float64) {
	*h = append(*h, v)
	if len(*h) > lusync.AccuracyHistorySize {
		*h = (*h)[len(*h)-lusync.AccuracyHistorySize:]
	}
}

func applyInactivityDecay(sigma, days float64) float64 {
	capped := math.Min(days, float64(lusync.MaxInactivityDays))
	if capped <= lusync.InactivityThresholdDay {
		return sigma
	}
	added := lusync.InactivitySigmaPerDay * (capped - lusync.InactivityThresholdDay)
	return clampF(sigma+added, lusync.MinSigma, lusync.MaxSigma)
}

func computeMatchKEStats(parts []participantData) (avg, stdDev float64) {
	if len(parts) == 0 {
		return 0, 0
	}
	sum := 0.0
	count := 0
	for _, p := range parts {
		if p.killsExpected > 0 {
			sum += p.killsExpected
			count++
		}
	}
	if count == 0 {
		return 0, 0
	}
	avg = sum / float64(count)
	variance := 0.0
	for _, p := range parts {
		if p.killsExpected > 0 {
			d := p.killsExpected - avg
			variance += d * d
		}
	}
	stdDev = math.Sqrt(variance / float64(count))
	return
}

func splitKEs(playerTeamID *int, parts []participantData) (teammates, enemies []float64) {
	for _, p := range parts {
		if p.killsExpected <= 0 {
			continue
		}
		if playerTeamID != nil && p.teamID != nil && *p.teamID == *playerTeamID {
			// Inclut le joueur lui-même → on l'exclut en filtrant team mais on
			// ne peut pas le distinguer ici (pas de xuid). On garde tous les
			// teammates → l'avg sera légèrement biaisée mais c'est OK pour
			// l'objectif du diag (compare deux modes sur même biais).
			teammates = append(teammates, p.killsExpected)
		} else if playerTeamID != nil && p.teamID != nil {
			enemies = append(enemies, p.killsExpected)
		}
	}
	return
}

func computeEnemyStrength(enemyKEs []float64, matchAvgKE, matchStdKE, playerMU float64) (float64, float64) {
	if len(enemyKEs) == 0 {
		return playerMU, lusync.DefaultOpponentSigma
	}
	estimates := make([]float64, 0, len(enemyKEs))
	for _, ke := range enemyKEs {
		var z float64
		if matchStdKE >= 1e-6 {
			z = (ke - matchAvgKE) / matchStdKE
		}
		estimates = append(estimates, playerMU+lusync.IndividualMUAlpha*z)
	}
	avg := 0.0
	for _, m := range estimates {
		avg += m
	}
	avg /= float64(len(estimates))
	if len(estimates) <= 1 {
		return avg, lusync.DefaultOpponentSigma
	}
	v := 0.0
	for _, m := range estimates {
		v += (m - avg) * (m - avg)
	}
	v /= float64(len(estimates))
	sig := clampF(math.Sqrt(v)+lusync.MinSigma, lusync.MinSigma, lusync.DefaultOpponentSigma)
	return avg, sig
}

// ── TrueSkill maths (copie locale, équivalent du package sync) ──────────────

func trueskillUpdate(mu, sigma, _, sigmaOpp, score, wf float64) (float64, float64) {
	deltaMU := lusync.KElo * (score - 0.5) * wf
	newMU := math.Max(lusync.MinRating, mu+deltaMU)
	c2 := 2.0*lusync.Beta*lusync.Beta + sigma*sigma + sigmaOpp*sigmaOpp
	c := math.Sqrt(c2)
	eps := drawMargin(lusync.Beta)
	w := wWin(0.0, eps/c)
	deltaSig2 := sigma * sigma * (sigma * sigma / c2) * w * wf
	newSig := math.Sqrt(math.Max(lusync.MinSigma*lusync.MinSigma, sigma*sigma-deltaSig2))
	newSig = math.Min(math.Sqrt(newSig*newSig+lusync.Tau*lusync.Tau), lusync.MaxSigma)
	return newMU, newSig
}

func drawMargin(beta float64) float64 {
	const drawProb = 0.06
	p := (drawProb + 1.0) / 2.0
	if p >= 1.0 {
		return 8.0 * beta
	}
	return math.Sqrt(-2.0*math.Log(1.0-p)) * beta
}

func wWin(t, eps float64) float64 {
	v := vWin(t, eps)
	x := t - eps
	return v * (v + x)
}

func vWin(t, eps float64) float64 {
	x := t - eps
	denom := 0.5 * (1.0 + math.Erf(x/math.Sqrt(2.0)))
	if denom < 1e-10 {
		return -x
	}
	pdf := math.Exp(-0.5*x*x) / math.Sqrt(2.0*math.Pi)
	return pdf / denom
}

func sigmoidRatio(num, denom float64) float64 {
	if denom <= 0 {
		return 0.5
	}
	r := num / denom
	return clampF(r/(1.0+r), 0.0, 1.0)
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
