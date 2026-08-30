// Package service - match_history_service_filters.go : filtres applicables
// aux MatchHistoryRawRow (sessions, ranked context, outcomes, tiers,
// explorer date/types/playlists/maps/modes/squad/whitelist) + sessions
// labels + filterMatchHistoryRows global + ExportCSV. Decoupe de
// match_history_service.go (god-file split, refactor 2026-05-27).
package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"
	"unicode"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

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
	want := ctx == scopeRanked
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
		return expTypePVE
	}
	if r.IsRanked {
		return expTypePVPRanked
	}
	return expTypePVPUnranked
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
		if _, ok := set[coalesceStr(r.MapNameFR, r.MapName)]; ok {
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
		// Pair prioritaire, sinon fallback game_variant (titres sans pair, ex. H5) —
		// même convention que le chokepoint filters_service.modeUI et les options.
		label := ""
		if v := analysis.ResolveModeUIWithVariant(r.PairName, r.PairNameFR, r.GameVariantName, r.GameVariantNameFR); v != nil {
			label = *v
		}
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
	case scopeSolo:
		out := rows[:0:0]
		for _, r := range rows {
			if !r.IsWithFriends {
				out = append(out, r)
			}
		}
		return out
	case scopeSquad:
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
//
// LES BLANCS SONT RETIRÉS AVANT COMPARAISON, TOUS, pas seulement ceux des bords. Un match ID
// est un GUID : il n'en contient jamais un seul, donc aucun blanc de la requête ne peut être
// significatif. Un identifiant collé depuis un log, une URL ou un message en ramène pourtant —
// aux extrémités le plus souvent, à l'intérieur quand la source a replié la ligne — et sans ce
// nettoyage la recherche ne rendait RIEN alors que l'identifiant saisi était le bon : le pire
// des échecs, celui qui se lit « ce match n'existe pas ».
//
// CONSÉQUENCE ASSUMÉE : une requête qui se réduit à du blanc redevient une requête VIDE, donc
// pas de filtre — c'est le seul sens qu'on puisse lui donner sans inventer un critère.
func filterByExplorerMatchIDSearch(rows []domain.MatchHistoryRawRow, query string) []domain.MatchHistoryRawRow {
	q := strings.ToLower(stripAllSpaces(query))
	if q == "" {
		return rows
	}
	out := rows[:0:0]
	for _, r := range rows {
		if strings.Contains(strings.ToLower(r.MatchID), q) {
			out = append(out, r)
		}
	}
	return out
}

// stripAllSpaces retire tout caractère d'espacement Unicode (espace, tabulation, saut de
// ligne, espace insécable — celle que ramènent les collages depuis un navigateur).
func stripAllSpaces(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

// filterByMatchIDsWhitelist garde uniquement les rows dont MatchID ∈ ids.
// Liste vide = pas de filtre. Comparaison exacte (case-sensitive).
func filterByMatchIDsWhitelist(rows []domain.MatchHistoryRawRow, ids []string) []domain.MatchHistoryRawRow {
	if len(ids) == 0 {
		return rows
	}
	set := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	out := rows[:0:0]
	for _, r := range rows {
		if _, ok := set[r.MatchID]; ok {
			out = append(out, r)
		}
	}
	return out
}

// applyExplorerMatchFilters applique tous les filtres Explorer additionnels en séquence.
//
// replays : ensemble des matchs ayant un artefact de rejeu 2D, résolu UNE FOIS par
// requête par l'appelant (nil = présence non résolue → le filtre replay_scope « avec
// rejeu » ne garde rien).
func applyExplorerMatchFilters(
	rows []domain.MatchHistoryRawRow, req domain.MatchHistoryQueryRequest, replays port.ReplayAvailability,
) []domain.MatchHistoryRawRow {
	rows = filterByMatchIDsWhitelist(rows, req.MatchIDs)
	rows = filterByExplorerDateRange(rows, req.MatchStartDate, req.MatchEndDate)
	rows = filterByExplorerExperienceTypes(rows, req.ExperienceTypes)
	rows = filterByExplorerPlaylists(rows, req.Playlists)
	rows = filterByExplorerMapNames(rows, req.MapNames)
	rows = filterByExplorerModeNames(rows, req.ModeNames)
	rows = filterByExplorerSquadScope(rows, req.SquadScope)
	rows = filterByExplorerReplayScope(rows, req.ReplayScope, replays)
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
		if m := coalesceStr(r.MapNameFR, r.MapName); m != "" {
			mapSet[m] = struct{}{}
		}
		// Pair prioritaire, sinon fallback game_variant (titres sans pair, ex. H5) —
		// même convention que filterByExplorerModeNames et les options du filtre.
		if v := analysis.ResolveModeUIWithVariant(r.PairName, r.PairNameFR, r.GameVariantName, r.GameVariantNameFR); v != nil && *v != "" {
			modeSet[*v] = struct{}{}
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
	// Rejeu 2D : l'export CSV ne porte pas de colonne « Rejeu » (cf. availableColumns)
	// — inutile de lister le dossier d'artefacts pour un fichier qui ne l'affiche pas.
	items := enrichRows(filtered, mapWinRates, s.rowFormatters(nil))
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
	// match_context appliqué tôt — symétrie avec ResolveFiltersFromRowsAt pour
	// que /pages/timeseries et /filters/resolve restent cohérents (même
	// dataset post-context avant scope temporel).
	rows = applyMatchContextFilter(rows, f.MatchContext)
	// Scope temporel : sessions OU période. On filtre par session dès que
	// picked_sessions n'est pas vide (peu importe filter_mode) — défense en
	// profondeur contre un client qui aurait oublié de propager filter_mode.
	var temporal []domain.FilterMatchRow
	if hasPickedSessions(f.Sessions) {
		temporal = applySessionFilter(rows, f.Sessions)
	} else {
		temporal = applyPeriodFilter(rows, f.Period)
	}
	return applyCascadeFilter(temporal, f.Cascade)
}

// hasPickedSessions retourne true si au moins une session est sélectionnée
// dans n'importe quel champ supporté (legacy single-pick ou multi-select).
func hasPickedSessions(s domain.SessionsFilter) bool {
	if len(s.PickedSessions) > 0 {
		return true
	}
	if s.PickedSessionLabel != nil && *s.PickedSessionLabel != "" {
		return true
	}
	if s.PickedSoloSessionLabel != nil && *s.PickedSoloSessionLabel != "" {
		return true
	}
	if s.PickedSquadSessionLabel != nil && *s.PickedSquadSessionLabel != "" {
		return true
	}
	return false
}
