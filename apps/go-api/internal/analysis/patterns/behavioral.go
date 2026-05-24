package patterns

import (
	"fmt"
	"math"
	"slices"

	"levelup/go-api/internal/analysis/temporal"
)

// behavioral.go — détection des patterns comportementaux.
//
// Cinq détecteurs :
//   - tilt : suite de défaites avec chute de KDA
//   - session_fatigue : dégradation KDA au sein d'une session
//   - engagement_drop : chute combinée EngageScore + ResidualBrut
//   - accuracy_plateau : précision stable mais basse
//   - perf_ceiling : plafond de performance LOWESS-détecté

// analyzeBehavior détecte les patterns comportementaux sur les rows.
func analyzeBehavior(rows []MatchRow, cfg PatternConfig) []BehavioralPattern {
	var out []BehavioralPattern
	if p, ok := detectTilt(rows, cfg); ok {
		out = append(out, p)
	}
	if p, ok := detectSessionFatigue(rows, cfg); ok {
		out = append(out, p)
	}
	if p, ok := detectEngagementDrop(rows, cfg); ok {
		out = append(out, p)
	}
	if p, ok := detectAccuracyPlateau(rows, cfg); ok {
		out = append(out, p)
	}
	if p, ok := detectPerfCeiling(rows, cfg); ok {
		out = append(out, p)
	}
	return out
}

// detectTilt détecte une suite de défaites avec chute de KDA.
func detectTilt(rows []MatchRow, cfg PatternConfig) (BehavioralPattern, bool) {
	// Trouver la plus longue suite de défaites
	bestRun, bestStart := 0, 0
	cur := 0
	curStart := 0
	for i, r := range rows {
		if r.Outcome == 3 { // LOSS
			if cur == 0 {
				curStart = i
			}
			cur++
			if cur > bestRun {
				bestRun = cur
				bestStart = curStart
			}
		} else {
			cur = 0
		}
	}
	if bestRun < cfg.TiltLossRun {
		return BehavioralPattern{}, false
	}

	tiltRows := rows[bestStart : bestStart+bestRun]
	// KDA pendant le tilt
	kdaDuring := meanKDA(tiltRows)
	// KDA hors du tilt
	var outsideRows []MatchRow
	outsideRows = append(outsideRows, rows[:bestStart]...)
	outsideRows = append(outsideRows, rows[bestStart+bestRun:]...)
	if len(outsideRows) == 0 {
		return BehavioralPattern{}, false
	}
	kdaOutside := meanKDA(outsideRows)
	if kdaOutside == 0 {
		return BehavioralPattern{}, false
	}
	kdaDrop := (kdaOutside - kdaDuring) / kdaOutside
	if kdaDrop <= cfg.TiltKDADropPct {
		return BehavioralPattern{}, false
	}

	sev := SeverityMedium
	if kdaDrop > 0.35 {
		sev = SeverityHigh
	}

	evidence := fmt.Sprintf("KDA chute de %.2f → %.2f", kdaOutside, kdaDuring)
	// Enrichir avec EngageScore si disponible
	if engDrop := meanEngageDrop(tiltRows, outsideRows); engDrop > 0 {
		evidence += fmt.Sprintf(" ; engagement −%.0f%%", engDrop*100)
	}

	return BehavioralPattern{
		Type:      BehaviorTilt,
		Trigger:   fmt.Sprintf("%d+ défaites consécutives", bestRun),
		Evidence:  evidence,
		Severity:  sev,
		Confirmed: false,
	}, true
}

// meanEngageDrop calcule la chute relative de EngageScore entre deux groupes.
// Retourne 0 si les données sont insuffisantes.
func meanEngageDrop(tiltRows, outsideRows []MatchRow) float64 {
	var tiltEng, outEng []float64
	for _, r := range tiltRows {
		if r.EngageScore != nil {
			tiltEng = append(tiltEng, *r.EngageScore)
		}
	}
	for _, r := range outsideRows {
		if r.EngageScore != nil {
			outEng = append(outEng, *r.EngageScore)
		}
	}
	if len(tiltEng) == 0 || len(outEng) == 0 {
		return 0
	}
	mOut := meanFloat(outEng)
	if mOut == 0 {
		return 0
	}
	mTilt := meanFloat(tiltEng)
	drop := (mOut - mTilt) / mOut
	if drop <= 0 {
		return 0
	}
	return drop
}

// meanKDA calcule la KDA moyenne sur un groupe de rows.
func meanKDA(rows []MatchRow) float64 {
	if len(rows) == 0 {
		return 0
	}
	sum := 0.0
	for _, r := range rows {
		sum += r.KDA
	}
	return sum / float64(len(rows))
}

// detectSessionFatigue détecte la dégradation de KDA au sein des sessions.
func detectSessionFatigue(rows []MatchRow, cfg PatternConfig) (BehavioralPattern, bool) {
	sessions := groupBySessions(rows, cfg)
	if len(sessions) == 0 {
		return BehavioralPattern{}, false
	}
	negSlopeCount := 0
	for _, sess := range sessions {
		if len(sess) < cfg.FatigueMinSession {
			continue
		}
		// Extraire la KDA par position
		kdas := make([]float64, len(sess))
		for i, r := range sess {
			kdas[i] = r.KDA
		}
		slope := linearSlope(kdas)
		if slope < 0 {
			negSlopeCount++
		}
	}
	// Filtrer les sessions valides (>= FatigueMinSession)
	validSessions := 0
	for _, sess := range sessions {
		if len(sess) >= cfg.FatigueMinSession {
			validSessions++
		}
	}
	if validSessions == 0 {
		return BehavioralPattern{}, false
	}
	cov := float64(negSlopeCount) / float64(validSessions)
	if cov < cfg.FatigueSessionCovPct {
		return BehavioralPattern{}, false
	}
	return BehavioralPattern{
		Type:      BehaviorSessionFatigue,
		Trigger:   fmt.Sprintf("Sessions de %d+ matchs", cfg.FatigueMinSession),
		Evidence:  fmt.Sprintf("KDA diminue sur %.0f%% des sessions analysées", cov*100),
		Severity:  SeverityMedium,
		Confirmed: false,
	}, true
}

// groupBySessions regroupe les rows par SessionID. Fallback : rows dans les
// 30 minutes les unes des autres si SessionID est vide.
func groupBySessions(rows []MatchRow, _ PatternConfig) [][]MatchRow {
	// Tenter d'abord par SessionID
	hasSessions := false
	for _, r := range rows {
		if r.SessionID != "" {
			hasSessions = true
			break
		}
	}
	if hasSessions {
		return groupBySessionID(rows)
	}
	return groupByTimeProximity(rows, 30)
}

// groupBySessionID regroupe les rows par valeur de SessionID.
func groupBySessionID(rows []MatchRow) [][]MatchRow {
	order := []string{}
	seen := map[string]bool{}
	groups := map[string][]MatchRow{}
	for _, r := range rows {
		key := r.SessionID
		if key == "" {
			key = "_nosession_"
		}
		groups[key] = append(groups[key], r)
		if !seen[key] {
			seen[key] = true
			order = append(order, key)
		}
	}
	out := make([][]MatchRow, 0, len(groups))
	for _, k := range order {
		out = append(out, groups[k])
	}
	return out
}

// groupByTimeProximity regroupe les rows par proximité temporelle (windowMin).
func groupByTimeProximity(rows []MatchRow, windowMin int) [][]MatchRow {
	if len(rows) == 0 {
		return nil
	}
	var sessions [][]MatchRow
	current := []MatchRow{rows[0]}
	for i := 1; i < len(rows); i++ {
		gap := rows[i].PlayedAt.Sub(rows[i-1].PlayedAt).Minutes()
		if gap < 0 {
			gap = -gap
		}
		if gap <= float64(windowMin) {
			current = append(current, rows[i])
		} else {
			sessions = append(sessions, current)
			current = []MatchRow{rows[i]}
		}
	}
	if len(current) > 0 {
		sessions = append(sessions, current)
	}
	return sessions
}

// linearSlope calcule la pente d'une série par régression linéaire simple.
func linearSlope(vals []float64) float64 {
	n := float64(len(vals))
	if n < 2 {
		return 0
	}
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	for i, v := range vals {
		x := float64(i)
		sumX += x
		sumY += v
		sumXY += x * v
		sumX2 += x * x
	}
	denom := n*sumX2 - sumX*sumX
	if denom == 0 {
		return 0
	}
	return (n*sumXY - sumX*sumY) / denom
}

// detectEngagementDrop détecte une chute combinée EngageScore + ResidualBrut.
func detectEngagementDrop(rows []MatchRow, _ PatternConfig) (BehavioralPattern, bool) {
	const minRows = 5
	const recentWindow = 10
	const dropThresholdHigh = 10

	// Extraire les rows avec les deux métriques non nil
	var fullRows []MatchRow
	for _, r := range rows {
		if r.EngageScore != nil && r.ResidualBrut != nil {
			fullRows = append(fullRows, r)
		}
	}
	if len(fullRows) < minRows {
		return BehavioralPattern{}, false
	}

	// Calculer P25 de EngageScore et ResidualBrut sur tous les rows valides
	p25Engage := percentile(collectEngageScores(fullRows), 0.25)
	p25Residual := percentile(collectResidualBrut(fullRows), 0.25)

	// Prendre les 10 plus récents
	recent := fullRows
	if len(recent) > recentWindow {
		recent = recent[:recentWindow]
	}
	if len(recent) < minRows {
		return BehavioralPattern{}, false
	}

	// Compter les rows sous les deux seuils simultanément
	dropCount := 0
	for _, r := range recent {
		if *r.EngageScore <= p25Engage && *r.ResidualBrut <= p25Residual {
			dropCount++
		}
	}
	if dropCount < minRows {
		return BehavioralPattern{}, false
	}

	sev := SeverityMedium
	if dropCount >= dropThresholdHigh {
		sev = SeverityHigh
	}

	// Calcul pour l'evidence
	avgEngage := meanNonNilEngageScore(recent)
	avgEngageAll := meanNonNilEngageScore(fullRows)
	engagePct := 100.0
	if avgEngageAll > 0 {
		engagePct = avgEngage / avgEngageAll * 100
	}
	avgResidual := meanNonNilResidualBrut(recent)
	avgResidualAll := meanNonNilResidualBrut(fullRows)
	residualPct := 100.0
	if avgResidualAll > 0 {
		residualPct = avgResidual / avgResidualAll * 100
	}

	return BehavioralPattern{
		Type:      BehaviorEngagementDrop,
		Trigger:   fmt.Sprintf("%d matchs récents sous les seuils habituels", dropCount),
		Evidence:  fmt.Sprintf("Engagement perso : %.0f%% vs habitude ; %.0f%% vs le lobby", engagePct, residualPct),
		Severity:  sev,
		Confirmed: false,
	}, true
}

// detectAccuracyPlateau détecte une précision stable mais basse.
func detectAccuracyPlateau(rows []MatchRow, cfg PatternConfig) (BehavioralPattern, bool) {
	const plateauWindow = 30
	window := rows
	if len(window) > plateauWindow {
		window = window[:plateauWindow]
	}
	if len(window) == 0 {
		return BehavioralPattern{}, false
	}
	accs := make([]float64, len(window))
	for i, r := range window {
		accs[i] = r.Accuracy
	}
	m := meanFloat(accs)
	std := stddev(accs, m)
	if std >= cfg.AccuracyPlateauStd || m >= cfg.AccuracyPlateauMax {
		return BehavioralPattern{}, false
	}
	sev := SeverityMedium
	if m > 0.35 {
		sev = SeverityLow
	} else if m < 0.25 {
		sev = SeverityHigh
	}
	return BehavioralPattern{
		Type:      BehaviorAccuracyPlateau,
		Trigger:   "30 matchs sans progression",
		Evidence:  fmt.Sprintf("Précision stable à %.1f%% (σ=%.3f)", m*100, std),
		Severity:  sev,
		Confirmed: false,
	}, true
}

// detectPerfCeiling détecte un plafond de performance via LOWESS.
func detectPerfCeiling(rows []MatchRow, _ PatternConfig) (BehavioralPattern, bool) {
	const minNonNilForCeiling = 20
	const perfWindow = 30
	const lowessAlpha = 0.4
	const flatSlopeThresh = 2.0
	const topN = 10

	// Collecter les rows avec PerfScore non nil
	var perfRows []MatchRow
	for _, r := range rows {
		if r.PerfScore != nil {
			perfRows = append(perfRows, r)
		}
	}
	if len(perfRows) < minNonNilForCeiling {
		return BehavioralPattern{}, false
	}

	// Prendre les 30 derniers
	window := perfRows
	if len(window) > perfWindow {
		window = window[:perfWindow]
	}

	scores := make([]float64, len(window))
	for i, r := range window {
		scores[i] = *r.PerfScore
	}

	maxScore := scores[0]
	for _, s := range scores {
		if s > maxScore {
			maxScore = s
		}
	}

	// Top 10 des scores
	sorted := make([]float64, len(scores))
	copy(sorted, scores)
	slices.Sort(sorted)
	// Prendre les 10 plus grands (fin de la slice triée)
	top := sorted
	if len(top) > topN {
		top = top[len(top)-topN:]
	}
	meanTop10 := meanFloat(top)

	// Condition 1 : max - meanTop10 < 5
	if maxScore-meanTop10 >= 5 {
		return BehavioralPattern{}, false
	}

	// Condition 2 : pente LOWESS ≈ plate
	smoothed := temporal.LowessSmooth(scores, lowessAlpha)
	slope := estimateLowessSlope(smoothed)
	if math.Abs(slope) >= flatSlopeThresh {
		return BehavioralPattern{}, false
	}

	return BehavioralPattern{
		Type:      BehaviorPerfCeiling,
		Trigger:   "Progression LOWESS plate sur 30+ matchs",
		Evidence:  fmt.Sprintf("Plafond de performance à %.0f pts (top 10 moyen : %.0f)", maxScore, meanTop10),
		Severity:  SeverityMedium,
		Confirmed: false,
	}, true
}

// estimateLowessSlope estime la pente globale d'une série lissée.
func estimateLowessSlope(smoothed []float64) float64 {
	// Filtrer les NaN
	var valid []float64
	for _, v := range smoothed {
		if !math.IsNaN(v) {
			valid = append(valid, v)
		}
	}
	return linearSlope(valid)
}

// stddev calcule l'écart-type d'une slice.
func stddev(vals []float64, mean float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		d := v - mean
		sum += d * d
	}
	return math.Sqrt(sum / float64(len(vals)))
}

// percentile retourne le percentile p (0..1) d'une slice.
func percentile(vals []float64, p float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	slices.Sort(sorted)
	idx := int(p * float64(len(sorted)))
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// collectEngageScores extrait les EngageScore non nil.
func collectEngageScores(rows []MatchRow) []float64 {
	var out []float64
	for _, r := range rows {
		if r.EngageScore != nil {
			out = append(out, *r.EngageScore)
		}
	}
	return out
}

// collectResidualBrut extrait les ResidualBrut non nil.
func collectResidualBrut(rows []MatchRow) []float64 {
	var out []float64
	for _, r := range rows {
		if r.ResidualBrut != nil {
			out = append(out, *r.ResidualBrut)
		}
	}
	return out
}

// meanNonNilEngageScore calcule la moyenne des EngageScore non nil.
func meanNonNilEngageScore(rows []MatchRow) float64 {
	return meanFloat(collectEngageScores(rows))
}

// meanNonNilResidualBrut calcule la moyenne des ResidualBrut non nil.
func meanNonNilResidualBrut(rows []MatchRow) float64 {
	return meanFloat(collectResidualBrut(rows))
}
