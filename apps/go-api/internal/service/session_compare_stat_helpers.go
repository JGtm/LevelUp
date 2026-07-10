// Package service — helpers arithmétiques pour session_compare_service.go.
// Calculs purs : winRate, avgKD, compareMetric, etc.
package service

import (
	"fmt"
	"math"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

func averageOC(matches []legacymatch.StatsMatchRow) *float64 {
	var sum float64
	var count int
	for _, m := range matches {
		if m.OffensiveConversion != nil {
			sum += *m.OffensiveConversion
			count++
		}
	}
	if count == 0 {
		return nil
	}
	v := math.Round(sum/float64(count)*100) / 100
	return &v
}

func averageDR(matches []legacymatch.StatsMatchRow) *float64 {
	var sum float64
	var count int
	for _, m := range matches {
		if m.DefensiveResistance != nil {
			sum += *m.DefensiveResistance
			count++
		}
	}
	if count == 0 {
		return nil
	}
	v := math.Round(sum/float64(count)*100) / 100
	return &v
}

func averagePerformanceScore(matches []legacymatch.StatsMatchRow) *float64 {
	count := 0
	total := 0.0
	for _, match := range matches {
		if match.PerfScoreComputed == nil {
			continue
		}
		count++
		total += *match.PerfScoreComputed
	}
	if count == 0 {
		return nil
	}
	value := math.Round((total/float64(count))*10) / 10
	return &value
}

// effectiveKDA retourne le KDA pré-calculé ou, à défaut, le dérive en FDA NET
// per-match ((kills + assists/3) − deaths) — JAMAIS un quotient par les morts.
// Utilisé par session_compare_service et session_page_service.
func effectiveKDA(match legacymatch.StatsMatchRow) *float64 {
	if match.KDA != nil {
		return match.KDA
	}
	value := math.Round((float64(match.Kills)+float64(match.Assists)/3.0-float64(match.Deaths))*100) / 100
	return &value
}

func derefFloat64(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func winRate(matches []legacymatch.StatsMatchRow) float64 {
	if len(matches) == 0 {
		return 0
	}
	wins := 0
	for _, m := range matches {
		if m.Outcome != nil && *m.Outcome == analysis.OutcomeWin {
			wins++
		}
	}
	// TODO(expiry:2026-12-31) P4 ADR 0006 : retirer *100 (convention API canonique 0..1).
	return analysis.WinRate(wins, len(matches)) * 100
}

func avgKD(matches []legacymatch.StatsMatchRow) float64 {
	if len(matches) == 0 {
		return 0
	}
	k, d := 0, 0
	for _, m := range matches {
		k += m.Kills
		d += m.Deaths
	}
	if d == 0 {
		return float64(k)
	}
	return float64(k) / float64(d)
}

func killsPerGame(matches []legacymatch.StatsMatchRow) float64 {
	if len(matches) == 0 {
		return 0
	}
	total := 0
	for _, m := range matches {
		total += m.Kills
	}
	return float64(total) / float64(len(matches))
}

func deathsPerGame(matches []legacymatch.StatsMatchRow) float64 {
	if len(matches) == 0 {
		return 0
	}
	total := 0
	for _, m := range matches {
		total += m.Deaths
	}
	return float64(total) / float64(len(matches))
}

func compareMetric(key, label string, a, b float64, format string) domain.SessionCompareMetricRow {
	va := fmt.Sprintf(format, a)
	vb := fmt.Sprintf(format, b)
	delta := fmt.Sprintf(format, a-b)
	winner := determineWinner(a, b)
	return domain.SessionCompareMetricRow{
		Key: key, Label: label, ValueA: va, ValueB: vb,
		Delta: &delta, Winner: winner,
	}
}

func compareMetricInverse(key, label string, a, b float64, format string) domain.SessionCompareMetricRow {
	va := fmt.Sprintf(format, a)
	vb := fmt.Sprintf(format, b)
	delta := fmt.Sprintf(format, a-b)
	winner := determineWinner(b, a) // lower is better
	return domain.SessionCompareMetricRow{
		Key: key, Label: label, ValueA: va, ValueB: vb,
		Delta: &delta, Winner: winner,
	}
}

func determineWinner(a, b float64) *string {
	const epsilon = 0.001
	if math.Abs(a-b) < epsilon {
		w := "tie"
		return &w
	}
	if a > b {
		w := "a"
		return &w
	}
	w := "b"
	return &w
}

func averageAccuracy(matches []legacymatch.StatsMatchRow) *float64 {
	var sum float64
	var count int
	for _, m := range matches {
		if m.Accuracy != nil {
			sum += *m.Accuracy
			count++
		}
	}
	if count == 0 {
		return nil
	}
	// StatsMatchRow.Accuracy provient brut de match_participants.accuracy, stockée en
	// pourcentage 0..100 (cf. player_matches_repo.go / Q23StatsMatchesShared). Le contrat
	// AvgAccuracy est 0..1 (ADR 0006) — le frontend multiplie par 100 à l'affichage. On
	// normalise donc ici, sinon double × 100 → précision affichée en milliers.
	v := math.Round(sum/float64(count)/100.0*1000) / 1000
	return &v
}
