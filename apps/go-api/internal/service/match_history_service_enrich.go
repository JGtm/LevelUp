// Package service - match_history_service_enrich.go : toFilterMatchRow +
// computeMapWinRates + enrichRows/enrichRow + sortItems/compareRows +
// paginate + helpers de format (outcomeLabel, formatDateFR,
// formatLifeSeconds, buildMatchURL, buildPeriodLabel, ptr/cmp helpers).
// Decoupe de match_history_service.go (god-file split, refactor 2026-05-27).
package service

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

func toFilterMatchRow(r domain.MatchHistoryRawRow) domain.FilterMatchRow {
	return domain.FilterMatchRow{
		MatchID:       r.MatchID,
		StartTime:     r.StartTime,
		MapName:       r.MapName,
		MapNameFR:     r.MapNameFR,
		PairName:      r.PairName,
		PairNameFR:    r.PairNameFR,
		PlaylistName:  r.PlaylistName,
		IsFirefight:   r.IsFirefight,
		IsRanked:      r.IsRanked,
		SessionID:     r.SessionID,
		SessionLabel:  r.SessionLabel,
		IsWithFriends: r.IsWithFriends,
	}
}

// ---------------------------------------------------------------------------
// Win rate par carte
// ---------------------------------------------------------------------------

func computeMapWinRates(rows []domain.MatchHistoryRawRow) map[string][2]int {
	m := make(map[string][2]int)
	for _, r := range rows {
		name := derefStr(r.MapName)
		if name == "" {
			continue
		}
		entry := m[name]
		entry[1]++ // total
		if r.Outcome == domain.OutcomeWin {
			entry[0]++ // wins
		}
		m[name] = entry
	}
	return m
}

// ---------------------------------------------------------------------------
// Enrichissement
// ---------------------------------------------------------------------------

func enrichRows(rows []domain.MatchHistoryRawRow, mapWR map[string][2]int, waypoint string) []domain.MatchHistoryRow {
	out := make([]domain.MatchHistoryRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, enrichRow(r, mapWR, waypoint))
	}
	return out
}

func enrichRow(r domain.MatchHistoryRawRow, mapWR map[string][2]int, waypoint string) domain.MatchHistoryRow {
	modeUI := analysis.ResolveModeUI(r.PairName, r.PairNameFR)
	// Fallback game_variant : les titres sans pair_name (Halo 5) portent leur mode dans
	// le game_variant. Sans ce repli, mode_ui resterait vide sur la liste Explorer/
	// Historique pour H5 (parité home/matchs-récents : analysis.modeLabels PairMode→GameVariant).
	if modeUI == nil {
		modeUI = analysis.ResolveModeUI(r.GameVariantName, r.GameVariantNameFR)
	}
	mapU := coalesce(r.MapNameFR, r.MapName)
	playlist := coalesce(r.PlaylistName, nil)

	var startTime time.Time
	if r.StartTime != nil {
		startTime = *r.StartTime
	}
	label := formatDateFR(startTime)

	var deltaMMR *float64
	if r.TeamMMR != nil && r.EnemyMMR != nil {
		v := *r.TeamMMR - *r.EnemyMMR
		deltaMMR = &v
	}

	var winRate *float64
	var winRateTotal *int
	if name := derefStr(r.MapName); name != "" {
		if entry, ok := mapWR[name]; ok && entry[1] > 0 {
			v := math.Round(float64(entry[0])/float64(entry[1])*100*10) / 10
			winRate = &v
			winRateTotal = &entry[1]
		}
	}

	matchURL := buildMatchURL(waypoint, r.MatchID)

	var perfScore *int
	var perfTier int
	if r.PerformanceScore != nil {
		v := int(math.Round(*r.PerformanceScore))
		perfScore = &v
		perfTier = int(analysis.PerfTier(*r.PerformanceScore))
	}

	var kda *float64
	if r.KDA != nil {
		kda = r.KDA
	}

	scoreLabel := "-"
	if r.MyTeamScore != nil && r.EnemyTeamScore != nil {
		scoreLabel = fmt.Sprintf("%d - %d", *r.MyTeamScore, *r.EnemyTeamScore)
	}

	return domain.MatchHistoryRow{
		MatchID:                  r.MatchID,
		StartTime:                startTime,
		StartTimeLabel:           label,
		OutcomeCode:              r.Outcome,
		OutcomeLabel:             outcomeLabel(r.Outcome),
		ScoreLabel:               scoreLabel,
		MapUI:                    ptrStr(mapU),
		ModeUI:                   modeUI,
		PlaylistLabel:            ptrStr(playlist),
		TeamMMR:                  r.TeamMMR,
		EnemyMMR:                 r.EnemyMMR,
		DeltaMMR:                 deltaMMR,
		WinRateHist:              winRate,
		WinRateHistTotal:         winRateTotal,
		PerformanceScoreRelative: perfScore,
		PerfTier:                 perfTier,
		KDA:                      kda,
		Kills:                    r.Kills,
		Deaths:                   r.Deaths,
		Assists:                  r.Assists,
		SkillTierLabel:           r.SkillTierLabel,
		SkillRatingType:          r.SkillRatingType,
		ExpectedWinProb:          r.SkillExpectedWinProb,
		PlacementDone:            r.PlacementDone,
		PlacementTotal:           r.PlacementTotal,
		AverageLifeMMSS:          formatLifeSeconds(r.AverageLifeSeconds),
		DurationSeconds:          r.TimePlayedSeconds,
		MatchURL:                 matchURL,
		IsExcluded:               r.IsExcluded,
		IsWithFriends:            r.IsWithFriends,
		ExperienceTypeLabel:      explorerExperienceType(r),
		DominanceFlag:            r.DominanceFlag,
	}
}

// ---------------------------------------------------------------------------
// Tri + pagination
// ---------------------------------------------------------------------------

func sortItems(items []domain.MatchHistoryRow, field, dir string) {
	if field == "" {
		field = "start_time"
	}
	if dir == "" {
		dir = "desc"
	}
	descending := dir == "desc"

	sort.SliceStable(items, func(i, j int) bool {
		less := compareMatchHistoryRows(items[i], items[j], field)
		if descending {
			return !less
		}
		return less
	})
}

func compareMatchHistoryRows(a, b domain.MatchHistoryRow, field string) bool {
	switch field {
	case "outcome_code":
		return a.OutcomeCode < b.OutcomeCode
	case "team_mmr":
		return cmpNullFloat(a.TeamMMR, b.TeamMMR)
	case "delta_mmr":
		return cmpNullFloat(a.DeltaMMR, b.DeltaMMR)
	case "win_rate_hist":
		return cmpNullFloat(a.WinRateHist, b.WinRateHist)
	case "performance_score_relative":
		return cmpNullInt(a.PerformanceScoreRelative, b.PerformanceScoreRelative)
	case "kda":
		return cmpNullFloat(a.KDA, b.KDA)
	case scopeKills:
		return a.Kills < b.Kills
	default: // start_time
		return a.StartTime.Before(b.StartTime)
	}
}

func paginate(items []domain.MatchHistoryRow, req domain.PaginationRequest) (domain.PaginationMeta, []domain.MatchHistoryRow) {
	total := len(items)
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 25
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}
	// Init à [] plutôt que nil : un slice nil sérialise en JSON `null` et crashe
	// le front sur .filter() / .map(). Cf. testutil.RequireNoNilSlicesWithoutOmitempty.
	pageItems := []domain.MatchHistoryRow{}
	if start < total {
		pageItems = items[start:end]
	}
	meta := domain.PaginationMeta{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasNext:  page < totalPages,
		HasPrev:  page > 1,
	}
	return meta, pageItems
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func outcomeLabel(code int) string {
	if lbl, ok := outcomeLabels[code]; ok {
		return lbl
	}
	return "-"
}

func formatDateFR(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("02/01/2006 15:04")
}

// zeroDuration est le format MM:SS retourné quand la durée est nulle ou invalide.
const zeroDuration = "0:00"

func formatLifeSeconds(secs *float64) string {
	if secs == nil {
		return zeroDuration
	}
	total := int(*secs)
	if total < 0 {
		return zeroDuration
	}
	mm := total / 60
	ss := total % 60
	return fmt.Sprintf("%d:%02d", mm, ss)
}

func buildMatchURL(waypoint, matchID string) string {
	wp := strings.TrimSpace(waypoint)
	if wp == "" {
		return ""
	}
	return "https://www.halowaypoint.com/halo-infinite/players/" + wp + "/matches/" + matchID
}

func buildPeriodLabel(f domain.FilterContextInput) *string {
	if f.FilterMode == scopeSessions {
		return nil
	}
	p := f.Period
	if p.StartDate == nil && p.EndDate == nil {
		return nil
	}
	var parts []string
	if p.StartDate != nil {
		parts = append(parts, p.StartDate.Format("02/01/2006"))
	}
	if p.EndDate != nil {
		parts = append(parts, p.EndDate.Format("02/01/2006"))
	}
	lbl := strings.Join(parts, " → ")
	return &lbl
}

func coalesce(a, b *string) string {
	if a != nil && *a != "" {
		return *a
	}
	if b != nil {
		return *b
	}
	return ""
}

func ptrStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func cmpNullFloat(a, b *float64) bool {
	if a == nil {
		return b != nil
	}
	if b == nil {
		return false
	}
	return *a < *b
}

func cmpNullInt(a, b *int) bool {
	if a == nil {
		return b != nil
	}
	if b == nil {
		return false
	}
	return *a < *b
}
