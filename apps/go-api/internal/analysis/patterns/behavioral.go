package patterns

import (
	"fmt"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// behavioral.go — détection des patterns comportementaux (tilt + fatigue).
//
// Les détecteurs engagement_drop, accuracy_plateau et perf_ceiling sont dans
// behavioral_engagement.go pour respecter la limite de 500 lignes par fichier.

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
	// Plus longue série de défaites consécutives + son index de départ (le start
	// est nécessaire pour découper tiltRows / outsideRows).
	bestRun, bestStart := analysis.LongestRun(rows, func(r MatchRow) bool {
		return r.Outcome == domain.OutcomeLoss
	})
	if bestRun < cfg.TiltLossRun {
		return BehavioralPattern{}, false
	}

	tiltRows := rows[bestStart : bestStart+bestRun]
	kdaDuring := meanKDA(tiltRows)
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
		kdas := make([]float64, len(sess))
		for i, r := range sess {
			kdas[i] = r.KDA
		}
		if linearSlope(kdas) < 0 {
			negSlopeCount++
		}
	}
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
	for _, r := range rows {
		if r.SessionID != "" {
			return groupBySessionID(rows)
		}
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
