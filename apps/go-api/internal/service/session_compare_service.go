// Package service â€” SessionCompareService : POST /pages/session-compare.
//
// Sprint 33 : compare deux sessions de jeu.
// Charge les matchs + sessions, filtre par label, calcule les mÃ©triques A vs B.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

// SessionCompareService compare deux sessions de jeu d'un joueur.
type SessionCompareService struct {
	sessionsRepo port.SessionsRepository
	statsRepo    port.StatsRepository
	// playerMatchesRepo (P4.1, ADR 0011) : loader canonical-aware optionnel.
	// TODO P4.3 : retirer le converter quand les analyses session_compare
	// (extractSessionLabels, buildCompareEntry, etc.) consommeront canonical.
	playerMatchesRepo port.PlayerMatchesRepository
	titleSlug         string
	gamertag          string
}

// NewSessionCompareService crÃ©e un SessionCompareService.
func NewSessionCompareService(
	sessionsRepo port.SessionsRepository,
	statsRepo port.StatsRepository,
) *SessionCompareService {
	return &SessionCompareService{
		sessionsRepo: sessionsRepo,
		statsRepo:    statsRepo,
	}
}

// WithPlayerMatchesRepo (P4.1, ADR 0011) injecte le loader canonical-aware.
func (s *SessionCompareService) WithPlayerMatchesRepo(repo port.PlayerMatchesRepository, titleSlug, gamertag string) *SessionCompareService {
	s.playerMatchesRepo = repo
	s.titleSlug = titleSlug
	s.gamertag = gamertag
	return s
}

// Compare exÃ©cute la comparaison entre deux sessions.
func (s *SessionCompareService) Compare(
	ctx context.Context,
	req domain.SessionCompareRequest,
) (domain.SessionCompareResponse, error) {
	// 1. Charger les matchs stats (avec session_label).
	// P4.3 finale (ADR 0011) : path canonical exclusif.
	if s.playerMatchesRepo == nil || s.titleSlug == "" || s.gamertag == "" {
		return domain.SessionCompareResponse{}, fmt.Errorf("SessionCompare: PlayerMatchesRepo non cÃ¢blÃ© (P4.3 finale exige le wiring DI)")
	}
	canonicalRows, err := s.playerMatchesRepo.LoadPlayerMatches(
		ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{},
	)
	if err != nil {
		slog.ErrorContext(ctx, "session_compare: échec chargement canonical", "gamertag", s.gamertag, "err", err)
		return domain.SessionCompareResponse{}, fmt.Errorf("SessionCompare: %w", err)
	}
	matches := filterStatsMatchRows(analysis.StatsMatchRowsFromCanonical(canonicalRows), req.Filters)
	slog.DebugContext(ctx, "session_compare: rows chargés", "gamertag", s.gamertag, "canonical", len(canonicalRows), "filtered", len(matches))

	// 2. Identifier les sessions disponibles.
	sessionLabels := extractSessionLabels(matches)

	if len(sessionLabels) < 2 {
		slog.InfoContext(ctx, "session_compare: sessions insuffisantes", "gamertag", s.gamertag, "sessions", len(sessionLabels))
		return domain.SessionCompareResponse{
			AvailableSessions: sessionLabels,
			Metrics:           []domain.SessionCompareMetricRow{},
			MapsTable:         []map[string]interface{}{},
			ModesTable:        []map[string]interface{}{},
		}, nil
	}

	// SÃ©lection automatique : derniÃ¨re et avant-derniÃ¨re sessions.
	labelA := lastOrNil(sessionLabels, req.SessionA)
	labelB := secondLastOrNil(sessionLabels, req.SessionB)
	slog.DebugContext(ctx, "session_compare: sélection sessions", "gamertag", s.gamertag, "session_a", labelA, "session_b", labelB, "available", len(sessionLabels))

	// 3. Filtrer les matchs par session.
	matchesA := filterBySession(matches, labelA)
	matchesB := filterBySession(matches, labelB)

	// 4. Calculer les entries et mÃ©triques.
	entryA := buildCompareEntry(matchesA, labelA)
	entryB := buildCompareEntry(matchesB, labelB)
	metrics := buildCompareMetrics(matchesA, matchesB)

	slog.InfoContext(ctx, "session_compare: comparaison terminée",
		"gamertag", s.gamertag,
		"matches_a", len(matchesA), "matches_b", len(matchesB),
		"metrics", len(metrics))

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

func extractSessionLabels(matches []legacymatch.StatsMatchRow) []string {
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

func filterBySession(matches []legacymatch.StatsMatchRow, label string) []legacymatch.StatsMatchRow {
	if label == "" {
		return nil
	}
	var out []legacymatch.StatsMatchRow
	for _, m := range matches {
		if m.SessionLabel != nil && *m.SessionLabel == label {
			out = append(out, m)
		}
	}
	return out
}

func buildCompareEntry(matches []legacymatch.StatsMatchRow, label string) *domain.SessionCompareEntry {
	if len(matches) == 0 || label == "" {
		return nil
	}
	wins, losses := 0, 0
	totalKills, totalDeaths := 0, 0
	var minTime, maxTime time.Time
	var ocSum, drSum float64
	var ocCount, drCount int
	var residualSum float64
	var residualCount int
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
			switch *m.Outcome {
			case analysis.OutcomeWin:
				wins++
			case analysis.OutcomeLoss:
				losses++
			}
		}
		totalKills += m.Kills
		totalDeaths += m.Deaths
		if m.OffensiveConversion != nil {
			ocSum += *m.OffensiveConversion
			ocCount++
		}
		if m.DefensiveResistance != nil {
			drSum += *m.DefensiveResistance
			drCount++
		}
		if m.EngagementScoreBrut != nil {
			residualSum += *m.EngagementScoreBrut
			residualCount++
		}
	}

	var kda *float64
	if totalDeaths > 0 {
		v := math.Round(float64(totalKills)/float64(totalDeaths)*100) / 100
		kda = &v
	}

	var avgOC, avgDR *float64
	if ocCount > 0 {
		v := math.Round(ocSum/float64(ocCount)*100) / 100
		avgOC = &v
	}
	if drCount > 0 {
		v := math.Round(drSum/float64(drCount)*100) / 100
		avgDR = &v
	}
	var avgResidualBrut *float64
	if residualCount > 0 {
		v := math.Round(residualSum/float64(residualCount)*100) / 100
		avgResidualBrut = &v
	}

	start := minTime.Format(time.RFC3339)
	end := maxTime.Format(time.RFC3339)

	return &domain.SessionCompareEntry{
		SessionLabel:     label,
		StartTime:        &start,
		EndTime:          &end,
		TotalMatches:     len(matches),
		Wins:             wins,
		Losses:           losses,
		KDA:              kda,
		PerformanceScore: averagePerformanceScore(matches),
		DominantCategory: dominantSessionCategoryPtr(matches),
		AvgOC:            avgOC,
		AvgDR:            avgDR,
		AvgResidualBrut:  avgResidualBrut,
	}
}

func buildCompareMetrics(a, b []legacymatch.StatsMatchRow) []domain.SessionCompareMetricRow {
	metrics := make([]domain.SessionCompareMetricRow, 0, 6)

	wra := winRate(a)
	wrb := winRate(b)
	metrics = append(metrics, compareMetric("win_rate", "Win Rate", wra, wrb, "%.1f%%"))

	kda := avgKD(a)
	kdb := avgKD(b)
	metrics = append(metrics, compareMetric("kd_ratio", "K/D", kda, kdb, "%.2f"))

	kpga := killsPerGame(a)
	kpgb := killsPerGame(b)
	metrics = append(metrics, compareMetric("kills_per_match", "Kills/match", kpga, kpgb, "%.1f"))

	dpga := deathsPerGame(a)
	dpgb := deathsPerGame(b)
	metrics = append(metrics, compareMetricInverse("deaths_per_match", "Deaths/match", dpga, dpgb, "%.1f"))

	spa := averagePerformanceScore(a)
	spb := averagePerformanceScore(b)
	if spa != nil || spb != nil {
		metrics = append(metrics, compareMetric(
			"score",
			"Score perf.",
			derefFloat64(spa),
			derefFloat64(spb),
			"%.1f",
		))
	}

	// OC/DR : uniquement si au moins une session a des données dégâts.
	ocA := averageOC(a)
	ocB := averageOC(b)
	if ocA != nil || ocB != nil {
		metrics = append(metrics, compareMetric("offensive_conversion", "Conversion off.", derefFloat64(ocA), derefFloat64(ocB), "%.2f"))
	}
	drA := averageDR(a)
	drB := averageDR(b)
	if drA != nil || drB != nil {
		metrics = append(metrics, compareMetric("defensive_resistance", "Résistance déf.", derefFloat64(drA), derefFloat64(drB), "%.2f"))
	}

	return metrics
}

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

func dominantSessionCategoryPtr(matches []legacymatch.StatsMatchRow) *string {
	category := dominantSessionCategory(matches)
	if category == "" {
		return nil
	}
	return &category
}

func dominantSessionCategory(matches []legacymatch.StatsMatchRow) string {
	if len(matches) == 0 {
		return ""
	}
	counts := map[string]int{}
	for _, match := range matches {
		category := classifySessionCategory(match)
		counts[category]++
	}
	bestLabel := ""
	bestCount := -1
	for label, count := range counts {
		if count > bestCount {
			bestLabel = label
			bestCount = count
		}
	}
	return bestLabel
}

func sessionIsRanked(matches []legacymatch.StatsMatchRow) bool {
	if len(matches) == 0 {
		return false
	}
	ranked := 0
	for _, match := range matches {
		if match.IsRanked {
			ranked++
		}
	}
	return ranked*2 >= len(matches)
}

// CatÃ©gories de session retournÃ©es par classifySessionCategory.
const (
	sessionCategoryFirefight = "Firefight"
	sessionCategoryRanked    = "Ranked"
	sessionCategoryBTB       = "BTB"
	sessionCategoryArena     = "Arena"
)

func classifySessionCategory(match legacymatch.StatsMatchRow) string {
	lower := strings.ToLower(match.PlaylistName + " " + match.PairName)
	switch {
	case strings.Contains(lower, "firefight"):
		return sessionCategoryFirefight
	case match.IsRanked || strings.Contains(lower, "ranked") || strings.Contains(lower, "classÃ©"):
		return sessionCategoryRanked
	case strings.Contains(lower, "btb") || strings.Contains(lower, "big team"):
		return sessionCategoryBTB
	default:
		return sessionCategoryArena
	}
}

func effectiveKDA(match legacymatch.StatsMatchRow) *float64 {
	if match.KDA != nil {
		return match.KDA
	}
	if match.Deaths == 0 {
		value := float64(match.Kills)
		return &value
	}
	value := math.Round((float64(match.Kills)/float64(match.Deaths))*100) / 100
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
	// TODO P4 ADR 0006 : retirer *100 (convention API canonique 0..1).
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
