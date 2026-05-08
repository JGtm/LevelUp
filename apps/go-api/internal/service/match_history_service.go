// Package service — MatchHistoryService : pagination et enrichissement de l'historique.
//
// Port Go de match_history_service.py (apps/api/app/services/match_history_service.py).
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
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
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
	"kda", "kills",
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
	// dataAdapter (optionnel, Phase C+ multi-titres) : point d'extension pour
	// router LoadMatchSummaries via la couche canonique quand elle évoluera
	// pour porter tous les champs MatchHistoryRawRow. À ce jour, le service
	// utilise le repo direct car canonical.MatchSummary ne couvre pas encore
	// la totalité du payload (delta_mmr, performance_score, etc.). Le hook
	// est en place pour permettre une bascule incrémentale.
	dataAdapter games.TitleDataAdapter
	// playerMatchesRepo (optionnel) : utilisé pour charger les canonical rows
	// nécessaires au calcul de BriefingKPIs (alimente <SessionBriefing> en haut
	// de page). Si non câblé, BriefingKPIs reste nil → briefing absent côté front.
	playerMatchesRepo port.PlayerMatchesRepository
	titleSlug         string
	gamertag          string
}

// NewMatchHistoryService crée un MatchHistoryService.
func NewMatchHistoryService(repo port.MatchHistoryRepository, waypointPlayer string) *MatchHistoryService {
	return &MatchHistoryService{repo: repo, waypointPlayer: waypointPlayer}
}

// WithDataAdapter injecte le DataAdapter multi-titres pour activer une
// future bascule LoadMatchSummaries. Dégradation gracieuse si nil.
func (s *MatchHistoryService) WithDataAdapter(a games.TitleDataAdapter) *MatchHistoryService {
	s.dataAdapter = a
	return s
}

// WithPlayerMatchesRepo injecte le loader canonical-aware utilisé pour calculer
// BriefingKPIs (composant <SessionBriefing> en haut de la page Stats).
// Dégradation gracieuse si non câblé : BriefingKPIs reste nil.
func (s *MatchHistoryService) WithPlayerMatchesRepo(repo port.PlayerMatchesRepository, titleSlug, gamertag string) *MatchHistoryService {
	s.playerMatchesRepo = repo
	s.titleSlug = titleSlug
	s.gamertag = gamertag
	return s
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

	// Exclusions manuelles — avant tout autre filtrage
	rawRows = filterExcludedRows(rawRows)

	// Filtrage (réutilise la logique pure du FiltersService)
	filtered := filterMatchHistoryRows(rawRows, req.Filters)

	// §5 plan Squad/Sessions : filtre multi-sessions solo (post-FilterContext).
	if len(req.PickedSoloSessionLabels) > 0 {
		filtered = filterMatchHistoryRowsBySoloSessions(filtered, req.PickedSoloSessionLabels)
	}

	// Capture des rows post-cascade post-PickedSoloSessions, avant tout filtre
	// Explorer (ranked/outcome/skill/perf + Explorer-cascade). C'est la base pour
	// le calcul des "available_*" cascade-aware avec sémantique OR.
	baseForExplorerOptions := filtered

	// Filtres supplémentaires (ranked context, outcome, skill tier, perf tier).
	filtered = filterByRankedContext(filtered, req.RankedContext)
	filtered = filterByOutcome(filtered, req.OutcomeFilter)
	filtered = filterBySkillTier(filtered, req.SkillTiers, req.RankedContext)
	filtered = filterByPerfTiers(filtered, req.PerfTiers)

	// Options Explorer disponibles calculées AVANT les filtres Explorer additionnels.
	availExpTypes, availPlaylists, availMaps, availModes := computeExplorerAvailableOptions(filtered)

	// Options Explorer-spécifiques avec count cascade-aware (sémantique OR au sein
	// d'une dimension, AND entre dimensions). Calculées sur baseForExplorerOptions
	// pour que chaque dimension reflète "ce qu'on aurait si on cochait X".
	availOutcomes := computeAvailableOutcomes(baseForExplorerOptions, req)
	availPerfTiers := computeAvailablePerfTiers(baseForExplorerOptions, req)
	availSkillTiers := computeAvailableSkillTiers(baseForExplorerOptions, req)
	availRankedCtxs := computeAvailableRankedContexts(baseForExplorerOptions, req)
	availSquadScopes := computeAvailableSquadScopes(baseForExplorerOptions, req)

	// Filtres Explorer additionnels (date, experience, playlist, carte, mode, squad, match ID).
	filtered = applyExplorerMatchFilters(filtered, req)

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

	// §5 plan Squad/Sessions : sessions dispo dérivées de l'ensemble brut
	// (avant filtrage scope) pour permettre au front de proposer le picker
	// solo/squad indépendamment du filtre période courant.
	sessionLabels := buildMatchHistorySessionLabels(rawRows)

	// BriefingKPIs : alimente <SessionBriefing> en haut de la page Stats.
	// Charge les canonical rows et filtre par les match_id du scope filtré
	// (mêmes filtres que la table). Dégradation gracieuse si playerMatchesRepo
	// n'est pas câblé.
	var briefingKPIs *domain.KPIStats
	if s.playerMatchesRepo != nil && s.titleSlug != "" && s.gamertag != "" && len(filtered) > 0 {
		canonicalRows, cerr := s.playerMatchesRepo.LoadPlayerMatches(
			ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{},
		)
		if cerr != nil {
			slog.WarnContext(ctx, "match_history.briefing_kpis.load_failed", "err", cerr)
		} else {
			keep := make(map[string]struct{}, len(filtered))
			for _, r := range filtered {
				keep[r.MatchID] = struct{}{}
			}
			scoped := make([]canonical.PlayerMatchRow, 0, len(filtered))
			for _, c := range canonicalRows {
				if _, ok := keep[c.Summary.MatchID]; ok {
					scoped = append(scoped, c)
				}
			}
			if len(scoped) > 0 {
				kpis := analysis.ComputeKPIStats(scoped)
				briefingKPIs = &kpis
			}
		}
	}

	return domain.MatchHistoryPageResponse{
		Summary: domain.MatchHistoryQuerySummary{
			TotalMatchesScoped:       totalScoped,
			TotalMatchesUnfiltered:   totalUnfiltered,
			PeriodLabel:              buildPeriodLabel(req.Filters),
			ActiveFilterMode:         req.Filters.FilterMode,
			AvailableExperienceTypes: availExpTypes,
			AvailablePlaylists:       availPlaylists,
			AvailableMaps:            availMaps,
			AvailableModes:           availModes,
			AvailableOutcomes:        availOutcomes,
			AvailablePerfTiers:       availPerfTiers,
			AvailableSkillTiers:      availSkillTiers,
			AvailableRankedContexts:  availRankedCtxs,
			AvailableSquadScopes:     availSquadScopes,
		},
		Table: domain.MatchHistoryTable{
			Items:      pageItems,
			Pagination: page,
		},
		AvailableSortFields: availableSortFields,
		AvailableColumns:    availableColumns,
		ExportHint:          exportHint,
		SessionLabels:       sessionLabels,
		BriefingKPIs:        briefingKPIs,
	}, nil
}

// filterMatchHistoryRowsBySoloSessions garde les rows dont SessionLabel
// figure dans labels et qui sont des sessions solo (IsWithFriends=FALSE).
// Les rows sans label sont exclues. Comparaison case-sensitive (les labels
// sont des identifiants normalisés côté ingestion).
func filterMatchHistoryRowsBySoloSessions(
	rows []domain.MatchHistoryRawRow,
	labels []string,
) []domain.MatchHistoryRawRow {
	if len(labels) == 0 {
		return rows
	}
	allowed := make(map[string]struct{}, len(labels))
	for _, l := range labels {
		allowed[l] = struct{}{}
	}
	out := rows[:0:0]
	for _, r := range rows {
		if r.IsWithFriends || r.SessionLabel == nil {
			continue
		}
		if _, ok := allowed[*r.SessionLabel]; ok {
			out = append(out, r)
		}
	}
	return out
}

// filterByRankedContext restreint aux matchs ranked ("ranked") ou non-ranked ("unranked").
// Valeur vide ou "all" = pas de filtre.
func filterByRankedContext(rows []domain.MatchHistoryRawRow, ctx string) []domain.MatchHistoryRawRow {
	if ctx == "" || ctx == "all" {
		return rows
	}
	want := ctx == "ranked"
	out := rows[:0:0]
	for _, r := range rows {
		if r.IsRanked == want {
			out = append(out, r)
		}
	}
	return out
}

// filterByOutcome garde les rows dont l'outcome figure dans la liste.
// Liste vide = pas de filtre.
func filterByOutcome(rows []domain.MatchHistoryRawRow, outcomes []int) []domain.MatchHistoryRawRow {
	if len(outcomes) == 0 {
		return rows
	}
	set := make(map[int]struct{}, len(outcomes))
	for _, o := range outcomes {
		set[o] = struct{}{}
	}
	out := rows[:0:0]
	for _, r := range rows {
		if _, ok := set[r.Outcome]; ok {
			out = append(out, r)
		}
	}
	return out
}

// filterBySkillTier filtre par tier ranked (CSR) ou LUSR selon rankedContext.
// Ignoré si SkillTiers est vide ou si rankedContext est vide (évite le mélange CSR/LUSR).
// Les tiers sont comparés case-insensitive sur le champ EN ("Diamond", "Onyx"…).
func filterBySkillTier(rows []domain.MatchHistoryRawRow, tiers []string, rankedContext string) []domain.MatchHistoryRawRow {
	if len(tiers) == 0 || rankedContext == "" || rankedContext == "all" {
		return rows
	}
	tierSet := make(map[string]struct{}, len(tiers))
	for _, t := range tiers {
		tierSet[strings.ToLower(t)] = struct{}{}
	}
	out := rows[:0:0]
	for _, r := range rows {
		if r.SkillTier == nil {
			continue
		}
		if _, ok := tierSet[strings.ToLower(*r.SkillTier)]; ok {
			out = append(out, r)
		}
	}
	return out
}

// filterByPerfTiers garde uniquement les rows dont le palier de performance
// (calculé depuis PerformanceScore via analysis.PerfTier) figure dans tiers.
// Liste vide = pas de filtre. Rows sans score toujours exclues quand un filtre est actif.
func filterByPerfTiers(rows []domain.MatchHistoryRawRow, tiers []int) []domain.MatchHistoryRawRow {
	if len(tiers) == 0 {
		return rows
	}
	tierSet := make(map[int]struct{}, len(tiers))
	for _, t := range tiers {
		tierSet[t] = struct{}{}
	}
	out := rows[:0:0]
	for _, r := range rows {
		if r.PerformanceScore == nil {
			continue
		}
		t := int(analysis.PerfTier(*r.PerformanceScore))
		if _, ok := tierSet[t]; ok {
			out = append(out, r)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Filtres Explorer additionnels (date, experience, playlist, map, mode, squad, match ID)
// ---------------------------------------------------------------------------

// explorerExperienceType dérive le type d'expérience d'un MatchHistoryRawRow.
func explorerExperienceType(r domain.MatchHistoryRawRow) string {
	if r.IsFirefight {
		return "PVE"
	}
	if r.IsRanked {
		return "PVP classé"
	}
	return "PVP non classé"
}

// filterByExplorerDateRange garde les rows dont StartTime est dans [start, end].
// nil = pas de borne. Plage inclusive.
func filterByExplorerDateRange(rows []domain.MatchHistoryRawRow, start, end *time.Time) []domain.MatchHistoryRawRow {
	if start == nil && end == nil {
		return rows
	}
	out := rows[:0:0]
	for _, r := range rows {
		if r.StartTime == nil {
			continue
		}
		t := *r.StartTime
		if start != nil && t.Before(*start) {
			continue
		}
		if end != nil && t.After(*end) {
			continue
		}
		out = append(out, r)
	}
	return out
}

// filterByExplorerExperienceTypes garde les rows dont le type d'expérience figure dans types.
// Liste vide = pas de filtre.
func filterByExplorerExperienceTypes(rows []domain.MatchHistoryRawRow, types []string) []domain.MatchHistoryRawRow {
	if len(types) == 0 {
		return rows
	}
	set := stringSliceToSet(types)
	out := rows[:0:0]
	for _, r := range rows {
		if _, ok := set[explorerExperienceType(r)]; ok {
			out = append(out, r)
		}
	}
	return out
}

// filterByExplorerPlaylists garde les rows dont PlaylistName figure dans playlists.
// Liste vide = pas de filtre.
func filterByExplorerPlaylists(rows []domain.MatchHistoryRawRow, playlists []string) []domain.MatchHistoryRawRow {
	if len(playlists) == 0 {
		return rows
	}
	set := stringSliceToSet(playlists)
	out := rows[:0:0]
	for _, r := range rows {
		if _, ok := set[derefStr(r.PlaylistName)]; ok {
			out = append(out, r)
		}
	}
	return out
}

// filterByExplorerMapNames garde les rows dont le label carte (FR > EN) figure dans maps.
// Liste vide = pas de filtre.
func filterByExplorerMapNames(rows []domain.MatchHistoryRawRow, maps []string) []domain.MatchHistoryRawRow {
	if len(maps) == 0 {
		return rows
	}
	set := stringSliceToSet(maps)
	out := rows[:0:0]
	for _, r := range rows {
		if _, ok := set[coalesce(r.MapNameFR, r.MapName)]; ok {
			out = append(out, r)
		}
	}
	return out
}

// filterByExplorerModeNames garde les rows dont le label mode normalisé (FR > EN) figure dans modes.
// Liste vide = pas de filtre.
func filterByExplorerModeNames(rows []domain.MatchHistoryRawRow, modes []string) []domain.MatchHistoryRawRow {
	if len(modes) == 0 {
		return rows
	}
	set := stringSliceToSet(modes)
	out := rows[:0:0]
	for _, r := range rows {
		label := analysis.NormalizeModeLabel(coalesce(r.PairNameFR, r.PairName))
		if _, ok := set[label]; ok {
			out = append(out, r)
		}
	}
	return out
}

// filterByExplorerSquadScope filtre selon le contexte squad :
// "solo" = !IsWithFriends, "squad" = IsWithFriends, sinon noop.
func filterByExplorerSquadScope(rows []domain.MatchHistoryRawRow, scope string) []domain.MatchHistoryRawRow {
	switch scope {
	case "solo":
		out := rows[:0:0]
		for _, r := range rows {
			if !r.IsWithFriends {
				out = append(out, r)
			}
		}
		return out
	case "squad":
		out := rows[:0:0]
		for _, r := range rows {
			if r.IsWithFriends {
				out = append(out, r)
			}
		}
		return out
	}
	return rows
}

// filterByExplorerMatchIDSearch garde les rows dont MatchID contient query (insensible à la casse).
// query vide = pas de filtre.
func filterByExplorerMatchIDSearch(rows []domain.MatchHistoryRawRow, query string) []domain.MatchHistoryRawRow {
	if query == "" {
		return rows
	}
	q := strings.ToLower(query)
	out := rows[:0:0]
	for _, r := range rows {
		if strings.Contains(strings.ToLower(r.MatchID), q) {
			out = append(out, r)
		}
	}
	return out
}

// applyExplorerMatchFilters applique tous les filtres Explorer additionnels en séquence.
func applyExplorerMatchFilters(rows []domain.MatchHistoryRawRow, req domain.MatchHistoryQueryRequest) []domain.MatchHistoryRawRow {
	rows = filterByExplorerDateRange(rows, req.MatchStartDate, req.MatchEndDate)
	rows = filterByExplorerExperienceTypes(rows, req.ExperienceTypes)
	rows = filterByExplorerPlaylists(rows, req.Playlists)
	rows = filterByExplorerMapNames(rows, req.MapNames)
	rows = filterByExplorerModeNames(rows, req.ModeNames)
	rows = filterByExplorerSquadScope(rows, req.SquadScope)
	rows = filterByExplorerMatchIDSearch(rows, req.MatchIDSearch)
	return rows
}

// computeExplorerAvailableOptions calcule les valeurs distinctes triées pour les 4 dimensions
// Explorer (experience, playlist, carte, mode) depuis les rows AVANT les filtres Explorer.
func computeExplorerAvailableOptions(rows []domain.MatchHistoryRawRow) (expTypes, playlists, maps, modes []string) {
	expSet := make(map[string]struct{})
	plSet := make(map[string]struct{})
	mapSet := make(map[string]struct{})
	modeSet := make(map[string]struct{})
	for _, r := range rows {
		expSet[explorerExperienceType(r)] = struct{}{}
		if pl := derefStr(r.PlaylistName); pl != "" {
			plSet[pl] = struct{}{}
		}
		if m := coalesce(r.MapNameFR, r.MapName); m != "" {
			mapSet[m] = struct{}{}
		}
		if mo := analysis.NormalizeModeLabel(coalesce(r.PairNameFR, r.PairName)); mo != "" {
			modeSet[mo] = struct{}{}
		}
	}
	expTypes = sortedKeys(expSet)
	playlists = sortedKeys(plSet)
	maps = sortedKeys(mapSet)
	modes = sortedKeys(modeSet)
	return
}

// sortedKeys retourne les clés d'un map[string]struct{} triées alphabétiquement.
func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// buildMatchHistorySessionLabels convertit les MatchHistoryRawRow en
// SessionLabelInput puis délègue à BuildSessionLabelsList. Skip les rows
// sans StartTime (ne devrait jamais arriver en prod, mais protège).
func buildMatchHistorySessionLabels(rows []domain.MatchHistoryRawRow) domain.SessionLabelsList {
	inputs := make([]SessionLabelInput, 0, len(rows))
	for _, r := range rows {
		if r.SessionLabel == nil || *r.SessionLabel == "" || r.StartTime == nil {
			continue
		}
		inputs = append(inputs, SessionLabelInput{
			Label:         *r.SessionLabel,
			StartTime:     *r.StartTime,
			IsWithFriends: r.IsWithFriends,
		})
	}
	return BuildSessionLabelsList(inputs)
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

	rawRows = filterExcludedRows(rawRows)
	filtered := filterMatchHistoryRows(rawRows, req.Filters)
	filtered = filterByRankedContext(filtered, req.RankedContext)
	filtered = filterByOutcome(filtered, req.OutcomeFilter)
	filtered = filterBySkillTier(filtered, req.SkillTiers, req.RankedContext)
	filtered = filterByPerfTiers(filtered, req.PerfTiers)
	mapWinRates := computeMapWinRates(rawRows)
	items := enrichRows(filtered, mapWinRates, s.waypointPlayer)
	sortItems(items, req.SortField, req.SortDir)

	return items, nil
}

// ---------------------------------------------------------------------------
// Filtrage (conversion MatchHistoryRawRow → FilterMatchRow pour réutiliser la logique)
// ---------------------------------------------------------------------------

// filterExcludedRows retire les matchs marqués is_excluded par l'utilisateur.
func filterExcludedRows(rows []domain.MatchHistoryRawRow) []domain.MatchHistoryRawRow {
	out := rows[:0:0]
	excluded := 0
	for _, r := range rows {
		if r.IsExcluded {
			excluded++
		} else {
			out = append(out, r)
		}
	}
	if excluded > 0 {
		slog.Debug("match history: excluded rows filtered",
			"excluded_count", excluded,
			"remaining", len(out),
		)
	}
	return out
}

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
	mu := analysis.NormalizeModeLabel(coalesce(r.PairNameFR, r.PairName))
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
		PerformanceScoreRelative: perfScore,
		PerfTier:                 perfTier,
		KDA:                      kda,
		Kills:                    r.Kills,
		Deaths:                   r.Deaths,
		Assists:                  r.Assists,
		SkillTierLabel:           r.SkillTierLabel,
		AverageLifeMMSS:          formatLifeSeconds(r.AverageLifeSeconds),
		MatchURL:                 matchURL,
		IsExcluded:               r.IsExcluded,
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
	case "kills":
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
