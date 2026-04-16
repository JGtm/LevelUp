// Package service — SessionCompareService : POST /pages/session-compare.
//
// Sprint 33 : compare deux sessions de jeu.
// Charge les matchs + sessions, filtre par label, calcule les métriques A vs B.
package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// SessionCompareService compare deux sessions de jeu d'un joueur.
type SessionCompareService struct {
	sessionsRepo port.SessionsRepository
	statsRepo    port.StatsRepository
}

// NewSessionCompareService crée un SessionCompareService.
func NewSessionCompareService(
	sessionsRepo port.SessionsRepository,
	statsRepo port.StatsRepository,
) *SessionCompareService {
	return &SessionCompareService{
		sessionsRepo: sessionsRepo,
		statsRepo:    statsRepo,
	}
}

// Compare exécute la comparaison entre deux sessions.
func (s *SessionCompareService) Compare(
	ctx context.Context,
	req domain.SessionCompareRequest,
) (domain.SessionCompareResponse, error) {
	// 1. Charger les matchs stats (avec session_label).
	matches, err := s.statsRepo.LoadStatsMatches(ctx)
	if err != nil {
		return domain.SessionCompareResponse{}, fmt.Errorf("SessionCompare load: %w", err)
	}

	// 2. Identifier les sessions disponibles.
	sessionLabels := extractSessionLabels(matches)

	if len(sessionLabels) < 2 {
		return domain.SessionCompareResponse{
			AvailableSessions: sessionLabels,
			Metrics:           []domain.SessionCompareMetricRow{},
			MapsTable:         []map[string]interface{}{},
			ModesTable:        []map[string]interface{}{},
		}, nil
	}

	// Sélection automatique : dernière et avant-dernière sessions.
	labelA := lastOrNil(sessionLabels, req.SessionA)
	labelB := secondLastOrNil(sessionLabels, req.SessionB)

	// 3. Filtrer les matchs par session.
	matchesA := filterBySession(matches, labelA)
	matchesB := filterBySession(matches, labelB)

	// 4. Calculer les entries et métriques.
	entryA := buildCompareEntry(matchesA, labelA)
	entryB := buildCompareEntry(matchesB, labelB)
	metrics := buildCompareMetrics(matchesA, matchesB)

	return domain.SessionCompareResponse{
		SessionA:          entryA,
		SessionB:          entryB,
		AvailableSessions: sessionLabels,
		Metrics:           metrics,
		MapsTable:         []map[string]interface{}{},
		ModesTable:        []map[string]interface{}{},
	}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func extractSessionLabels(matches []domain.StatsMatchRow) []string {
	seen := make(map[string]bool)
	var labels []string
	for _, m := range matches {
		if m.SessionLabel != nil && !seen[*m.SessionLabel] {
			seen[*m.SessionLabel] = true
			labels = append(labels, *m.SessionLabel)
		}
	}
	// Trier chronologiquement (les labels contiennent des dates).
	sort.Strings(labels)
	return labels
}

func lastOrNil(labels []string, override *string) string {
	if override != nil && *override != "" {
		return *override
	}
	if len(labels) == 0 {
		return ""
	}
	return labels[len(labels)-1]
}

func secondLastOrNil(labels []string, override *string) string {
	if override != nil && *override != "" {
		return *override
	}
	if len(labels) < 2 {
		return ""
	}
	return labels[len(labels)-2]
}

func filterBySession(matches []domain.StatsMatchRow, label string) []domain.StatsMatchRow {
	if label == "" {
		return nil
	}
	var out []domain.StatsMatchRow
	for _, m := range matches {
		if m.SessionLabel != nil && *m.SessionLabel == label {
			out = append(out, m)
		}
	}
	return out
}

func buildCompareEntry(matches []domain.StatsMatchRow, label string) *domain.SessionCompareEntry {
	if len(matches) == 0 || label == "" {
		return nil
	}
	wins, losses := 0, 0
	totalKills, totalDeaths := 0, 0
	var minTime, maxTime time.Time
	for i, m := range matches {
		if i == 0 {
			minTime = m.StartTime
			maxTime = m.StartTime
		}
		if m.StartTime.Before(minTime) {
			minTime = m.StartTime
		}
		if m.StartTime.After(maxTime) {
			maxTime = m.StartTime
		}
		if m.Outcome != nil {
			if *m.Outcome == analysis.OutcomeWin {
				wins++
			} else if *m.Outcome == analysis.OutcomeLoss {
				losses++
			}
		}
		totalKills += m.Kills
		totalDeaths += m.Deaths
	}

	var kda *float64
	if totalDeaths > 0 {
		v := math.Round(float64(totalKills)/float64(totalDeaths)*100) / 100
		kda = &v
	}

	start := minTime.Format(time.RFC3339)
	end := maxTime.Format(time.RFC3339)

	return &domain.SessionCompareEntry{
		SessionLabel: label,
		StartTime:    &start,
		EndTime:      &end,
		TotalMatches: len(matches),
		Wins:         wins,
		Losses:       losses,
		KDA:          kda,
	}
}

func buildCompareMetrics(a, b []domain.StatsMatchRow) []domain.SessionCompareMetricRow {
	metrics := make([]domain.SessionCompareMetricRow, 0, 6)

	wra := winRate(a)
	wrb := winRate(b)
	metrics = append(metrics, compareMetric("win_rate", "Win Rate", wra, wrb, "%.1f%%"))

	kda := avgKD(a)
	kdb := avgKD(b)
	metrics = append(metrics, compareMetric("kd_ratio", "K/D", kda, kdb, "%.2f"))

	kpga := killsPerGame(a)
	kpgb := killsPerGame(b)
	metrics = append(metrics, compareMetric("kills_per_game", "Kills/game", kpga, kpgb, "%.1f"))

	dpga := deathsPerGame(a)
	dpgb := deathsPerGame(b)
	metrics = append(metrics, compareMetricInverse("deaths_per_game", "Deaths/game", dpga, dpgb, "%.1f"))

	return metrics
}

func winRate(matches []domain.StatsMatchRow) float64 {
	if len(matches) == 0 {
		return 0
	}
	wins := 0
	for _, m := range matches {
		if m.Outcome != nil && *m.Outcome == analysis.OutcomeWin {
			wins++
		}
	}
	return float64(wins) / float64(len(matches)) * 100
}

func avgKD(matches []domain.StatsMatchRow) float64 {
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

func killsPerGame(matches []domain.StatsMatchRow) float64 {
	if len(matches) == 0 {
		return 0
	}
	total := 0
	for _, m := range matches {
		total += m.Kills
	}
	return float64(total) / float64(len(matches))
}

func deathsPerGame(matches []domain.StatsMatchRow) float64 {
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
