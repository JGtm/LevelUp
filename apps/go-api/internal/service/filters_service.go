// Package service — FiltersService : résolution du contexte de filtres.
//
// Port Go du filter_service.py Python (apps/api/app/services/filter_service.py).
// Les données brutes sont chargées par le repo ; ce service applique la logique pure.
package service

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// Constantes expérience (ordre affiché dans l'UI).
var experienceLabels = []string{"PVP non classé", "PVP classé", "PVE"}

// stripOnRe retire " on NomCarte" et les suffixes Forge/Ranked des modes.
var stripOnRe = regexp.MustCompile(`(?i) on .+$`)
var stripForgeRe = regexp.MustCompile(`(?i)\s*-\s*Forge\b`)
var stripRankedRe = regexp.MustCompile(`(?i)\s*-\s*Ranked\b`)

// FiltersService calcule FilterContextResolved depuis les données du repo.
type FiltersService struct {
	repo port.FiltersRepository
}

// NewFiltersService crée un FiltersService.
func NewFiltersService(repo port.FiltersRepository) *FiltersService {
	return &FiltersService{repo: repo}
}

// Resolve charge les matchs du joueur et retourne le contexte résolu.
func (s *FiltersService) Resolve(
	ctx context.Context,
	input domain.FilterContextInput,
) (domain.FilterContextResolved, error) {
	rows, err := s.repo.LoadMatchesForFilters(ctx)
	if err != nil {
		return domain.FilterContextResolved{}, err
	}
	return ResolveFiltersFromRows(rows, input), nil
}

// ResolveFiltersFromRows est la fonction pure testable sans repo.
func ResolveFiltersFromRows(
	rows []domain.FilterMatchRow,
	input domain.FilterContextInput,
) domain.FilterContextResolved {
	totalBefore := len(rows)
	sessionOpts := buildSessionOptions(rows)
	effective := normalizeInput(input)

	if totalBefore == 0 {
		return emptyResolved(effective, sessionOpts)
	}

	// 1. Filtre temporel
	var temporal []domain.FilterMatchRow
	if effective.FilterMode == "sessions" {
		temporal = applySessionFilter(rows, effective.Sessions)
	} else {
		temporal = applyPeriodFilter(rows, effective.Period)
	}

	// 2. Options disponibles (avant cascade)
	available := buildAvailableOptions(temporal, effective.Cascade)

	// 3. Cascade
	filtered := applyCascadeFilter(temporal, effective.Cascade)

	return domain.FilterContextResolved{
		Effective:        effective,
		AvailableOptions: available,
		SessionOptions:   sessionOpts,
		Counts: domain.FilterCounts{
			TotalMatchesBeforeFilters: totalBefore,
			TotalMatchesAfterFilters:  len(filtered),
		},
	}
}

// ---------------------------------------------------------------------------
// Helpers d'enrichissement
// ---------------------------------------------------------------------------

func stripModeSuffix(s string) string {
	s = stripOnRe.ReplaceAllString(s, "")
	s = stripForgeRe.ReplaceAllString(s, "")
	s = stripRankedRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func modeUI(row domain.FilterMatchRow) string {
	raw := derefStr(row.PairNameFR)
	if raw == "" {
		raw = derefStr(row.PairName)
	}
	return stripModeSuffix(raw)
}

func mapUI(row domain.FilterMatchRow) string {
	if v := derefStr(row.MapNameFR); v != "" {
		return v
	}
	return derefStr(row.MapName)
}

func playlistUI(row domain.FilterMatchRow) string {
	return derefStr(row.PlaylistName)
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ---------------------------------------------------------------------------
// Session options
// ---------------------------------------------------------------------------

func buildSessionOptions(rows []domain.FilterMatchRow) domain.SessionOptions {
	type aggEntry struct {
		count   int
		isSquad bool
	}
	agg := make(map[string]aggEntry)
	sessionID := make(map[string]string) // label → session_id

	for _, r := range rows {
		lbl := derefStr(r.SessionLabel)
		if lbl == "" {
			continue
		}
		e := agg[lbl]
		e.count++
		if r.IsWithFriends {
			e.isSquad = true
		}
		agg[lbl] = e
		if sid := derefStr(r.SessionID); sid != "" {
			sessionID[lbl] = sid
		}
	}

	labels := make([]string, 0, len(agg))
	for lbl := range agg {
		labels = append(labels, lbl)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(labels)))

	var all []domain.SessionOption
	var soloLabels, squadLabels []string
	for _, lbl := range labels {
		e := agg[lbl]
		sid := sessionID[lbl]
		if sid == "" {
			sid = lbl
		}
		opt := domain.SessionOption{
			Label:      lbl,
			SessionID:  sid,
			MatchCount: e.count,
			IsSquad:    e.isSquad,
		}
		all = append(all, opt)
		if e.isSquad {
			squadLabels = append(squadLabels, lbl)
		} else {
			soloLabels = append(soloLabels, lbl)
		}
	}
	return domain.SessionOptions{
		AllSessions: all,
		SoloLabels:  soloLabels,
		SquadLabels: squadLabels,
	}
}

// ---------------------------------------------------------------------------
// Filtres
// ---------------------------------------------------------------------------

func applyPeriodFilter(rows []domain.FilterMatchRow, p domain.PeriodInput) []domain.FilterMatchRow {
	if p.StartDate == nil && p.EndDate == nil {
		return rows
	}
	out := rows[:0:0]
	for _, r := range rows {
		if r.StartTime == nil {
			continue
		}
		t := *r.StartTime
		if p.StartDate != nil && t.Before(*p.StartDate) {
			continue
		}
		if p.EndDate != nil {
			end := p.EndDate.Add(24*time.Hour - time.Second)
			if t.After(end) {
				continue
			}
		}
		out = append(out, r)
	}
	return out
}

func applySessionFilter(rows []domain.FilterMatchRow, s domain.SessionsFilter) []domain.FilterMatchRow {
	keep := make(map[string]struct{})
	add := func(lbl string) {
		if lbl != "" {
			keep[lbl] = struct{}{}
		}
	}
	add(derefStr(s.PickedSessionLabel))
	add(derefStr(s.PickedSoloSessionLabel))
	add(derefStr(s.PickedSquadSessionLabel))
	for _, lbl := range s.PickedSessions {
		add(lbl)
	}
	if len(keep) == 0 {
		return rows
	}
	out := rows[:0:0]
	for _, r := range rows {
		if lbl := derefStr(r.SessionLabel); lbl != "" {
			if _, ok := keep[lbl]; ok {
				out = append(out, r)
			}
		}
	}
	return out
}

func applyCascadeFilter(rows []domain.FilterMatchRow, c domain.CascadeFilter) []domain.FilterMatchRow {
	rows = applyExperienceFilter(rows, c.ExperienceTypes)
	rows = filterBySet(rows, c.Playlists, playlistUI)
	rows = filterBySet(rows, c.Modes, modeUI)
	rows = filterBySet(rows, c.Maps, mapUI)
	return rows
}

func applyExperienceFilter(rows []domain.FilterMatchRow, types []string) []domain.FilterMatchRow {
	if len(types) == 0 || len(types) >= len(experienceLabels) {
		return rows
	}
	wantPVE, wantRanked, wantUnranked := false, false, false
	for _, t := range types {
		tl := strings.ToLower(t)
		switch {
		case strings.Contains(tl, "pve") || strings.Contains(tl, "firefight"):
			wantPVE = true
		case strings.Contains(tl, "classé") || strings.Contains(tl, "ranked"):
			wantRanked = true
		default:
			wantUnranked = true
		}
	}
	out := rows[:0:0]
	for _, r := range rows {
		switch {
		case r.IsFirefight && wantPVE:
			out = append(out, r)
		case !r.IsFirefight && r.IsRanked && wantRanked:
			out = append(out, r)
		case !r.IsFirefight && !r.IsRanked && wantUnranked:
			out = append(out, r)
		}
	}
	return out
}

func filterBySet(rows []domain.FilterMatchRow, values []string, fn func(domain.FilterMatchRow) string) []domain.FilterMatchRow {
	if len(values) == 0 {
		return rows
	}
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		set[v] = struct{}{}
	}
	out := rows[:0:0]
	for _, r := range rows {
		if _, ok := set[fn(r)]; ok {
			out = append(out, r)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Options disponibles
// ---------------------------------------------------------------------------

func buildAvailableOptions(rows []domain.FilterMatchRow, c domain.CascadeFilter) domain.AvailableFilterOptions {
	expOpts := make([]domain.LabelValue, len(experienceLabels))
	for i, lbl := range experienceLabels {
		expOpts[i] = domain.LabelValue{Label: lbl, Value: lbl}
	}

	rowsExp := applyExperienceFilter(rows, c.ExperienceTypes)
	playlistOpts := uniqueLabelValues(rowsExp, playlistUI)

	rowsPl := filterBySet(rowsExp, c.Playlists, playlistUI)
	modeOpts := uniqueLabelValues(rowsPl, modeUI)

	rowsMo := filterBySet(rowsPl, c.Modes, modeUI)
	mapOpts := uniqueLabelValues(rowsMo, mapUI)

	return domain.AvailableFilterOptions{
		ExperienceTypes: expOpts,
		Playlists:       playlistOpts,
		Modes:           modeOpts,
		Maps:            mapOpts,
	}
}

func uniqueLabelValues(rows []domain.FilterMatchRow, fn func(domain.FilterMatchRow) string) []domain.LabelValue {
	seen := make(map[string]struct{})
	for _, r := range rows {
		if v := fn(r); v != "" {
			seen[v] = struct{}{}
		}
	}
	vals := make([]string, 0, len(seen))
	for v := range seen {
		vals = append(vals, v)
	}
	sort.Strings(vals)
	out := make([]domain.LabelValue, len(vals))
	for i, v := range vals {
		out[i] = domain.LabelValue{Label: v, Value: v}
	}
	return out
}

// ---------------------------------------------------------------------------
// Helpers mineurs
// ---------------------------------------------------------------------------

func normalizeInput(in domain.FilterContextInput) domain.FilterContextInput {
	p := in.Period
	if p.StartDate != nil && p.EndDate != nil && p.EndDate.Before(*p.StartDate) {
		p.StartDate, p.EndDate = p.EndDate, p.StartDate
	}
	in.Period = p
	return in
}

func emptyResolved(effective domain.FilterContextInput, sess domain.SessionOptions) domain.FilterContextResolved {
	expOpts := make([]domain.LabelValue, len(experienceLabels))
	for i, lbl := range experienceLabels {
		expOpts[i] = domain.LabelValue{Label: lbl, Value: lbl}
	}
	return domain.FilterContextResolved{
		Effective:        effective,
		AvailableOptions: domain.AvailableFilterOptions{ExperienceTypes: expOpts},
		SessionOptions:   sess,
		Counts:           domain.FilterCounts{},
	}
}
