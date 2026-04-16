// Package service — MatchHistoryService : pagination et enrichissement de l'historique.
//
// Port Go de match_history_service.py (apps/api/app/services/match_history_service.py).
package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// Outcomes mappés selon les codes Halo Infinite.
var outcomeLabels = map[int]string{
	1: "Égalité",
	2: "Victoire",
	3: "Défaite",
	4: "Abandon",
}

// availableSortFields est la liste des champs de tri autorisés.
var availableSortFields = []string{
	"start_time", "outcome_code", "performance_score_relative",
	"team_mmr", "delta_mmr", "win_rate_hist",
}

// availableColumns expose les colonnes disponibles dans MatchHistoryRow.
var availableColumns = []string{
	"match_id", "start_time", "outcome_label", "score_label",
	"map_ui", "mode_ui", "playlist_label",
	"team_mmr", "enemy_mmr", "delta_mmr",
	"win_rate_hist", "performance_score_relative",
	"average_life_mmss", "match_url",
}

// MatchHistoryService construit la réponse paginée pour la page historique.
type MatchHistoryService struct {
	repo           port.MatchHistoryRepository
	waypointPlayer string
}

// NewMatchHistoryService crée un MatchHistoryService.
func NewMatchHistoryService(repo port.MatchHistoryRepository, waypointPlayer string) *MatchHistoryService {
	return &MatchHistoryService{repo: repo, waypointPlayer: waypointPlayer}
}

// GetPage charge tous les matchs, applique filtres+pagination et retourne la réponse.
func (s *MatchHistoryService) GetPage(
	ctx context.Context,
	req domain.MatchHistoryQueryRequest,
) (domain.MatchHistoryPageResponse, error) {
	rawRows, err := s.repo.LoadAll(ctx)
	if err != nil {
		return domain.MatchHistoryPageResponse{}, fmt.Errorf("MatchHistoryService.GetPage: %w", err)
	}
	totalUnfiltered := len(rawRows)

	// Filtrage (réutilise la logique pure du FiltersService)
	filtered := filterMatchHistoryRows(rawRows, req.Filters)
	totalScoped := len(filtered)

	// Calcul win_rate par carte sur l'histotique complet (avant filtre)
	mapWinRates := computeMapWinRates(rawRows)

	// Enrichissement
	items := enrichRows(filtered, mapWinRates, s.waypointPlayer)

	// Tri
	sortItems(items, req.SortField, req.SortDir)

	// Pagination
	page, pageItems := paginate(items, req.Pagination)

	var exportHint *domain.ExportHint
	if req.IncludeExportHint && totalScoped > 0 {
		exportHint = &domain.ExportHint{
			FileName:      "levelup_matches.csv",
			EstimatedRows: totalScoped,
		}
	}

	return domain.MatchHistoryPageResponse{
		Summary: domain.MatchHistoryQuerySummary{
			TotalMatchesScoped:     totalScoped,
			TotalMatchesUnfiltered: totalUnfiltered,
			PeriodLabel:            buildPeriodLabel(req.Filters),
			ActiveFilterMode:       req.Filters.FilterMode,
		},
		Table: domain.MatchHistoryTable{
			Items:      pageItems,
			Pagination: page,
		},
		AvailableSortFields: availableSortFields,
		AvailableColumns:    availableColumns,
		ExportHint:          exportHint,
	}, nil
}

// ExportCSV charge les matchs filtrés et retourne les lignes pour export CSV.
// Pas de pagination — retourne tous les matchs de la requête.
func (s *MatchHistoryService) ExportCSV(
	ctx context.Context,
	req domain.MatchHistoryQueryRequest,
) ([]domain.MatchHistoryRow, error) {
	rawRows, err := s.repo.LoadAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchHistoryService.ExportCSV: %w", err)
	}

	filtered := filterMatchHistoryRows(rawRows, req.Filters)
	mapWinRates := computeMapWinRates(rawRows)
	items := enrichRows(filtered, mapWinRates, s.waypointPlayer)
	sortItems(items, req.SortField, req.SortDir)

	return items, nil
}

// ---------------------------------------------------------------------------
// Filtrage (conversion MatchHistoryRawRow → FilterMatchRow pour réutiliser la logique)
// ---------------------------------------------------------------------------

func filterMatchHistoryRows(rows []domain.MatchHistoryRawRow, f domain.FilterContextInput) []domain.MatchHistoryRawRow {
	filterRows := make([]domain.FilterMatchRow, len(rows))
	for i, r := range rows {
		filterRows[i] = toFilterMatchRow(r)
	}
	resolved := ResolveFiltersFromRows(filterRows, f)
	_ = resolved // les IDs filtrés sont obtenus différemment
	// Réappliquer les mêmes filtres pour préserver les objets complets
	filterRows = applyAllFilters(filterRows, f)
	keepIDs := make(map[string]struct{}, len(filterRows))
	for _, fr := range filterRows {
		keepIDs[fr.MatchID] = struct{}{}
	}
	out := rows[:0:0]
	for _, r := range rows {
		if _, ok := keepIDs[r.MatchID]; ok {
			out = append(out, r)
		}
	}
	return out
}

func applyAllFilters(rows []domain.FilterMatchRow, f domain.FilterContextInput) []domain.FilterMatchRow {
	f = normalizeInput(f)
	var temporal []domain.FilterMatchRow
	if f.FilterMode == "sessions" {
		temporal = applySessionFilter(rows, f.Sessions)
	} else {
		temporal = applyPeriodFilter(rows, f.Period)
	}
	return applyCascadeFilter(temporal, f.Cascade)
}

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
		if r.Outcome == 2 {
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
	mu := stripModeSuffix(coalesce(r.PairNameFR, r.PairName))
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

	return domain.MatchHistoryRow{
		MatchID:                  r.MatchID,
		StartTime:                startTime,
		StartTimeLabel:           label,
		OutcomeCode:              r.Outcome,
		OutcomeLabel:             outcomeLabel(r.Outcome),
		ScoreLabel:               "-",
		MapUI:                    ptrStr(mapU),
		ModeUI:                   ptrStr(mu),
		PlaylistLabel:            ptrStr(playlist),
		TeamMMR:                  r.TeamMMR,
		EnemyMMR:                 r.EnemyMMR,
		DeltaMMR:                 deltaMMR,
		WinRateHist:              winRate,
		WinRateHistTotal:         winRateTotal,
		PerformanceScoreRelative: nil,
		AverageLifeMMSS:          formatLifeSeconds(r.AverageLifeSeconds),
		MatchURL:                 matchURL,
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
	var pageItems []domain.MatchHistoryRow
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

func formatLifeSeconds(secs *float64) string {
	if secs == nil {
		return "0:00"
	}
	total := int(*secs)
	if total < 0 {
		return "0:00"
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
	if f.FilterMode == "sessions" {
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
