// Package service — TeammatesService : endpoint POST /pages/teammates (contrat FastAPI).
//
// Sprint 33 : adapte les données SquadRepository vers le contrat TeammatesPageResponse.
// Réutilise les mêmes queries Q29-Q31 que SquadService mais expose le format FastAPI.
package service

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// FriendGamertagsResolver retourne la liste courante des amis configurés
// (app_settings.friend_gamertags). Appelé à chaque requête pour refléter les
// PATCH settings sans redémarrage.
type FriendGamertagsResolver func(ctx context.Context) []string

// TeammatesService calcule les stats coéquipiers au format FastAPI.
type TeammatesService struct {
	repo            port.SquadRepository
	friendGamertags FriendGamertagsResolver
}

// NewTeammatesService crée un TeammatesService.
//
// friendGamertags : optionnel. Si nil, le filtre amis-only est désactivé
// (top retourné brut, ancien comportement). Quand fourni, le top dropdown
// est restreint aux amis configurés.
func NewTeammatesService(repo port.SquadRepository, friendGamertags FriendGamertagsResolver) *TeammatesService {
	return &TeammatesService{repo: repo, friendGamertags: friendGamertags}
}

// GetPage retourne la page Teammates avec options, comparaisons et solo ref.
func (s *TeammatesService) GetPage(
	ctx context.Context,
	playerXUID string,
	req domain.TeammatesQueryRequest,
) (domain.TeammatesPageResponse, error) {
	topRows, err := s.repo.LoadTopTeammates(ctx, playerXUID)
	if err != nil {
		return domain.TeammatesPageResponse{}, fmt.Errorf("TeammatesService: %w", err)
	}

	// §3 plan Squad/Sessions : filtre top dropdown aux amis configurés
	// (settings.friend_gamertags). Hors amis = exclus du dropdown mais
	// toujours requêtables explicitement via SelectedGamertags + alias.
	var friendGTs []string
	if s.friendGamertags != nil {
		friendGTs = s.friendGamertags(ctx)
	}
	dropdownRows := topRows
	if friendGTs != nil {
		dropdownRows = filterTopRowsToFriends(topRows, friendGTs)
	}

	// Options (liste des coéquipiers fréquents — limitée aux amis si configuré).
	options := buildTeammateOptions(dropdownRows)

	// LoadSynthesisMatches alimente sessionLabels (qu'on filtre derrière par
	// session escouade ; le code session_labels.solo est conservé pour
	// compat de DTO mais inutilisé par la page Escouade — la page Solo a
	// son propre endpoint).
	allMatches, err := s.repo.LoadSynthesisMatches(ctx, playerXUID)
	if err != nil {
		return domain.TeammatesPageResponse{}, fmt.Errorf("TeammatesService synthesis: %w", err)
	}

	// Extraire les session_labels disponibles (solo / escouade).
	sessionLabels := extractSynthesisSessionLabels(allMatches)

	// Filtrer les matchs selon les sessions sélectionnées.
	filteredMatches := filterSynthesisBySession(allMatches, req.PickedSoloSessions, req.PickedSquadSessions)

	// Appliquer les filtres cascade (experience_types, playlists) si présents.
	if req.Filters != nil {
		filteredMatches = filterSynthesisByCascade(filteredMatches, req.Filters.Cascade)
	}

	totalMatches := len(filteredMatches)

	// Calculs détaillés pour les gamertags sélectionnés.
	teammates := make([]domain.TeammateRow, 0, len(req.SelectedGamertags))
	var allSquadRows []domain.SquadMatchRow
	matchSeries := map[string][]domain.SquadMatchSeriesPoint{}

	for _, gt := range req.SelectedGamertags {
		row, squadMatches, err := s.buildTeammateRowWithMatches(ctx, playerXUID, gt, topRows, filteredMatches)
		if err != nil {
			slog.WarnContext(ctx, "teammates: erreur buildTeammateRow", "gamertag", gt, "err", err)
			continue // skip teammate on error
		}
		if row != nil {
			teammates = append(teammates, *row)
			allSquadRows = append(allSquadRows, squadMatches...)
			matchSeries[gt] = buildMatchSeries(squadMatches)
		}
	}

	// Timeseries + MapBreakdown sur l'union des matchs escouade.
	var timeseries []domain.SquadTimeseriesPoint
	var mapBreakdown []domain.MapBreakdownRow
	if len(allSquadRows) > 0 {
		timeseries = analysis.ComputeSquadTimeseries(allSquadRows, 20)
		mapBreakdown = computeMapBreakdown(allSquadRows)
	}

	return domain.TeammatesPageResponse{
		Options:       options,
		Teammates:     teammates,
		TotalMatches:  totalMatches,
		SessionLabels: sessionLabels,
		FriendsCount:  len(friendGTs),
		Timeseries:    timeseries,
		MapBreakdown:  mapBreakdown,
		MatchSeries:   matchSeries,
	}, nil
}

// filterTopRowsToFriends garde uniquement les lignes dont le gamertag matche
// (case-insensitive, trim) un ami de friendGamertags. Liste vide = aucune
// ligne (le user doit ajouter des amis pour peupler le dropdown).
func filterTopRowsToFriends(rows []domain.TopTeammateRow, friendGamertags []string) []domain.TopTeammateRow {
	if len(friendGamertags) == 0 {
		return nil
	}
	allowed := make(map[string]struct{}, len(friendGamertags))
	for _, gt := range friendGamertags {
		k := strings.ToLower(strings.TrimSpace(gt))
		if k != "" {
			allowed[k] = struct{}{}
		}
	}
	out := make([]domain.TopTeammateRow, 0, len(rows))
	for _, r := range rows {
		if _, ok := allowed[strings.ToLower(r.Gamertag)]; ok {
			out = append(out, r)
		}
	}
	return out
}

// extractSynthesisSessionLabels collecte les sessions uniques en séparant solo / escouade,
// calcule les bornes temporelles, agrège les expériences et playlists présentes, et trie par StartedAt DESC.
func extractSynthesisSessionLabels(matches []domain.SynthesisMatchRow) domain.SessionLabelsList {
	type meta struct {
		startedAt   time.Time
		endedAt     time.Time
		experiences map[string]struct{}
		playlists   map[string]struct{}
	}
	soloMap := map[string]*meta{}
	squadMap := map[string]*meta{}

	for _, m := range matches {
		if m.SessionLabel == nil || *m.SessionLabel == "" {
			continue
		}
		label := *m.SessionLabel
		t := m.StartTime
		var em map[string]*meta
		if m.IsWithFriends {
			em = squadMap
		} else {
			em = soloMap
		}
		entry, ok := em[label]
		if !ok {
			entry = &meta{
				startedAt:   t,
				endedAt:     t,
				experiences: map[string]struct{}{},
				playlists:   map[string]struct{}{},
			}
			em[label] = entry
		}
		if t.Before(entry.startedAt) {
			entry.startedAt = t
		}
		if t.After(entry.endedAt) {
			entry.endedAt = t
		}
		entry.experiences[synthesisExperienceLabel(m)] = struct{}{}
		if m.PlaylistName != "" {
			entry.playlists[m.PlaylistName] = struct{}{}
		}
	}

	toSlice := func(m map[string]*meta) []domain.SessionLabelEntry {
		out := make([]domain.SessionLabelEntry, 0, len(m))
		for label, entry := range m {
			exps := make([]string, 0, len(entry.experiences))
			for e := range entry.experiences {
				exps = append(exps, e)
			}
			slices.Sort(exps)
			pls := make([]string, 0, len(entry.playlists))
			for p := range entry.playlists {
				pls = append(pls, p)
			}
			slices.Sort(pls)
			out = append(out, domain.SessionLabelEntry{
				Label:       label,
				StartedAt:   entry.startedAt,
				EndedAt:     entry.endedAt,
				Experiences: exps,
				Playlists:   pls,
			})
		}
		slices.SortFunc(out, func(a, b domain.SessionLabelEntry) int {
			return cmp.Compare(b.StartedAt.Unix(), a.StartedAt.Unix())
		})
		return out
	}

	return domain.SessionLabelsList{
		Solo:  toSlice(soloMap),
		Squad: toSlice(squadMap),
	}
}

// synthesisExperienceLabel dérive le label d'expérience d'un match (miroir de filters_service.go).
func synthesisExperienceLabel(m domain.SynthesisMatchRow) string {
	if m.IsFirefight {
		return "PVE"
	}
	if m.IsRanked {
		return "PVP classé"
	}
	return "PVP non classé"
}

// filterSynthesisByCascade applique les filtres experience_types et playlists sur les matchs.
func filterSynthesisByCascade(matches []domain.SynthesisMatchRow, c domain.CascadeFilter) []domain.SynthesisMatchRow {
	if len(c.ExperienceTypes) == 0 && len(c.Playlists) == 0 {
		return matches
	}
	expSet := make(map[string]struct{}, len(c.ExperienceTypes))
	for _, e := range c.ExperienceTypes {
		expSet[e] = struct{}{}
	}
	plSet := make(map[string]struct{}, len(c.Playlists))
	for _, p := range c.Playlists {
		plSet[p] = struct{}{}
	}
	out := matches[:0:0]
	for _, m := range matches {
		if len(expSet) > 0 {
			if _, ok := expSet[synthesisExperienceLabel(m)]; !ok {
				continue
			}
		}
		if len(plSet) > 0 {
			if _, ok := plSet[m.PlaylistName]; !ok {
				continue
			}
		}
		out = append(out, m)
	}
	return out
}

// filterSynthesisBySession filtre les matchs selon les sessions sélectionnées (union des labels).
// Slices vides → tous les matchs retournés sans filtre.
func filterSynthesisBySession(
	matches []domain.SynthesisMatchRow,
	pickedSolo []string,
	pickedSquad []string,
) []domain.SynthesisMatchRow {
	if len(pickedSolo) == 0 && len(pickedSquad) == 0 {
		return matches
	}
	filtered := make([]domain.SynthesisMatchRow, 0, len(matches))
	for _, m := range matches {
		label := ""
		if m.SessionLabel != nil {
			label = *m.SessionLabel
		}
		if len(pickedSolo) > 0 && !m.IsWithFriends && slices.Contains(pickedSolo, label) {
			filtered = append(filtered, m)
			continue
		}
		if len(pickedSquad) > 0 && m.IsWithFriends && slices.Contains(pickedSquad, label) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// buildTeammateOptions convertit les TopTeammateRow en TeammateOption.
func buildTeammateOptions(rows []domain.TopTeammateRow) []domain.TeammateOption {
	opts := make([]domain.TeammateOption, 0, len(rows))
	for _, r := range rows {
		xuid := r.XUID
		opts = append(opts, domain.TeammateOption{
			Gamertag:       r.Gamertag,
			XUID:           &xuid,
			EncounterCount: r.GamesTogether,
		})
	}
	return opts
}

// buildTeammateRowWithMatches construit les KPIs avec/sans pour un coéquipier et retourne aussi les matches escouade.
func (s *TeammatesService) buildTeammateRowWithMatches(
	ctx context.Context,
	playerXUID, gamertag string,
	topRows []domain.TopTeammateRow,
	allMatches []domain.SynthesisMatchRow,
) (*domain.TeammateRow, []domain.SquadMatchRow, error) {
	// Étape 1 : chercher le gamertag dans le top 50 escouade — case-insensitive
	// pour absorber les variations de casse entre la saisie user et la valeur en
	// DB (Halo API renvoie tantôt "Madina97294" tantôt "madina97294").
	var teammateXUID string
	var encounterCount int
	for _, r := range topRows {
		if strings.EqualFold(r.Gamertag, gamertag) {
			teammateXUID = r.XUID
			encounterCount = r.GamesTogether
			break
		}
	}

	// Étape 2 : fallback — résoudre via shared.xuid_aliases pour les gamertags
	// hors top 50 (utilisateur qui a 50+ coéquipiers réguliers OU saisie libre
	// dans la combobox). encounterCount reste 0 — recalculé depuis squadMatches
	// plus bas si on charge effectivement les matchs.
	if teammateXUID == "" {
		resolved, found, err := s.repo.LookupXUIDByGamertag(ctx, gamertag)
		if err != nil {
			slog.WarnContext(ctx, "teammates_gamertag_lookup_failed",
				"player_xuid", playerXUID,
				"gamertag", gamertag,
				"err", err.Error(),
			)
			return nil, nil, nil
		}
		if !found {
			// Vraiment inconnu de tous les aliases — on log et on drop.
			slog.WarnContext(ctx, "teammates_gamertag_not_found",
				"player_xuid", playerXUID,
				"gamertag", gamertag,
				"top_rows_count", len(topRows),
			)
			return nil, nil, nil
		}
		teammateXUID = resolved
	}

	// Charger les matchs communs.
	squadMatches, err := s.repo.LoadSquadMatches(ctx, playerXUID, teammateXUID)
	if err != nil {
		// Erreur DB : on log avec contexte (le warn générique
		// "teammates: erreur buildTeammateRow" du caller perd le détail
		// du XUID résolu, on conserve donc une trace ciblée ici).
		slog.ErrorContext(ctx, "teammates_load_squad_matches_failed",
			"player_xuid", playerXUID, "teammate_xuid", teammateXUID,
			"gamertag", gamertag, "err", err.Error())
		return nil, nil, fmt.Errorf("buildTeammateRowWithMatches LoadSquadMatches: %w", err)
	}

	withKPIs := computeKPIsFromSquadMatches(squadMatches)

	// KPIs "sans" = matchs qui ne sont PAS dans les matchs communs.
	commonIDs := make(map[string]bool, len(squadMatches))
	for _, m := range squadMatches {
		commonIDs[m.MatchID] = true
	}
	withoutKPIs := computeKPIsFromSynthesisExcluding(allMatches, commonIDs)

	xuid := teammateXUID
	var lastSeen *time.Time
	if len(squadMatches) > 0 {
		t := squadMatches[0].StartTime
		for _, m := range squadMatches {
			if m.StartTime.After(t) {
				t = m.StartTime
			}
		}
		lastSeen = &t
	}

	// Si encounterCount n'a pas été renseigné par le top 50 (fallback alias-only),
	// on le calcule depuis les matchs communs effectivement chargés.
	if encounterCount == 0 {
		encounterCount = len(squadMatches)
	}

	return &domain.TeammateRow{
		Gamertag:       gamertag,
		XUID:           &xuid,
		EncounterCount: encounterCount,
		LastSeenAt:     lastSeen,
		WithKPIs:       withKPIs,
		WithoutKPIs:    &withoutKPIs,
	}, squadMatches, nil
}

// computeKPIsFromSquadMatches calcule les KPIs depuis les matchs communs.
func computeKPIsFromSquadMatches(matches []domain.SquadMatchRow) domain.TeammateKPIs {
	n := len(matches)
	if n == 0 {
		return domain.TeammateKPIs{}
	}
	wins := 0
	totalKills, totalDeaths, totalAssists := 0, 0, 0
	totalHS, totalPK := 0, 0
	accSum, accCount := 0.0, 0
	for _, m := range matches {
		if m.Outcome == analysis.OutcomeWin {
			wins++
		}
		totalKills += m.Kills
		totalDeaths += m.Deaths
		totalAssists += m.Assists
		totalHS += m.HeadshotKills
		totalPK += m.PerfectKills
		if m.Accuracy != nil {
			accSum += *m.Accuracy
			accCount++
		}
	}
	kd := safeDiv(float64(totalKills), float64(totalDeaths))
	kpg := float64(totalKills) / float64(n)
	apg := float64(totalAssists) / float64(n)
	hspg := float64(totalHS) / float64(n)
	pkpg := float64(totalPK) / float64(n)
	var acc *float64
	if accCount > 0 {
		v := round2(accSum / float64(accCount) * 100)
		acc = &v
	}
	return domain.TeammateKPIs{
		MatchCount: n,
		Wins:       wins,
		KDRatio:    &kd,
		// TODO P4 ADR 0006 : retirer *100 (convention API canonique 0..1).
		WinRate:              round2(analysis.WinRate(wins, n) * 100),
		Accuracy:             acc,
		KillsPerGame:         &kpg,
		AssistsPerGame:       &apg,
		HeadshotKillsPerGame: &hspg,
		PerfectKillsPerGame:  &pkpg,
	}
}

// computeKPIsFromSynthesisExcluding calcule les KPIs en excluant certains matchs.
func computeKPIsFromSynthesisExcluding(
	matches []domain.SynthesisMatchRow,
	exclude map[string]bool,
) domain.TeammateKPIs {
	var filtered []domain.SynthesisMatchRow
	for _, m := range matches {
		if !exclude[m.MatchID] {
			filtered = append(filtered, m)
		}
	}
	n := len(filtered)
	if n == 0 {
		return domain.TeammateKPIs{}
	}
	wins := 0
	totalKills, totalDeaths := 0, 0
	for _, m := range filtered {
		if m.Outcome == analysis.OutcomeWin {
			wins++
		}
		totalKills += m.Kills
		totalDeaths += m.Deaths
	}
	kd := safeDiv(float64(totalKills), float64(totalDeaths))
	kpg := float64(totalKills) / float64(n)
	return domain.TeammateKPIs{
		MatchCount: n,
		Wins:       wins,
		KDRatio:    &kd,
		// TODO P4 ADR 0006 : retirer *100 (convention API canonique 0..1).
		WinRate:      round2(analysis.WinRate(wins, n) * 100),
		KillsPerGame: &kpg,
	}
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return a
	}
	return round2(a / b)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// computeMapBreakdown agrège les stats par carte depuis les matchs escouade.
func computeMapBreakdown(matches []domain.SquadMatchRow) []domain.MapBreakdownRow {
	type stats struct{ count, wins int }
	m := map[string]*stats{}
	for _, r := range matches {
		key := r.MapUI
		if key == "" {
			key = "Unknown"
		}
		if _, ok := m[key]; !ok {
			m[key] = &stats{}
		}
		m[key].count++
		if r.Outcome == analysis.OutcomeWin {
			m[key].wins++
		}
	}
	result := make([]domain.MapBreakdownRow, 0, len(m))
	for mapUI, s := range m {
		result = append(result, domain.MapBreakdownRow{
			MapUI:      mapUI,
			MatchCount: s.count,
			WinRate:    round2(float64(s.wins) / float64(s.count) * 100),
		})
	}
	return result
}

// buildMatchSeries construit la série temporelle des matchs pour un coéquipier.
func buildMatchSeries(matches []domain.SquadMatchRow) []domain.SquadMatchSeriesPoint {
	series := make([]domain.SquadMatchSeriesPoint, 0, len(matches))
	for _, m := range matches {
		series = append(series, domain.SquadMatchSeriesPoint{
			MatchID:          m.MatchID,
			StartTime:        m.StartTime.Format("2006-01-02T15:04:05Z"),
			Outcome:          m.Outcome,
			PerformanceScore: m.PerformanceScore,
			TeamMMRAvg:       m.TeamMMR,
			SessionLabel:     m.SessionLabel,
		})
	}
	return series
}
