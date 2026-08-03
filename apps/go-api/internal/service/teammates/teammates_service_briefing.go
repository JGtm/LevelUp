// Package service - teammates_service_briefing.go : briefing header +
// loadTeammatesCanonicalParallel + filtres synthesis (cascade, period,
// picked sessions, session, experience labels). Decoupe de
// teammates_service.go (god-file split, refactor 2026-05-27).
package teammates

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/squadagg"
)

func (s *TeammatesService) buildBriefingHeaderForTeammatesPage(
	ctx context.Context,
	mainFiltered []canonical.PlayerMatchRow,
	selectedGamertags []string,
	filters *domain.FilterContextInput,
	sessionMatchIDs map[string]bool,
	compFilter *exactCompositionFilter,
) *domain.SquadHeader {
	// Mode solo : SoloKPIs uniquement (pas de verdict squad).
	// Egalement le cas si squadLoader pas cable (degradation gracieuse).
	if len(selectedGamertags) == 0 || s.squadLoader == nil {
		if len(mainFiltered) == 0 {
			return nil
		}
		kpis := analysis.ComputeKPIStats(mainFiltered, games.EffectiveHpToKill(s.titleSlug))
		return &domain.SquadHeader{SoloKPIs: &kpis}
	}

	// Mode squad : charge canonical rows par teammate en parallele.
	teammateRows, err := s.loadTeammatesCanonicalParallel(ctx, selectedGamertags)
	if err != nil {
		slog.WarnContext(ctx, "teammates_briefing.load_failed",
			"err", err, "selected_count", len(selectedGamertags))
		// Degradation : juste SoloKPIs.
		if len(mainFiltered) == 0 {
			return nil
		}
		kpis := analysis.ComputeKPIStats(mainFiltered, games.EffectiveHpToKill(s.titleSlug))
		return &domain.SquadHeader{SoloKPIs: &kpis}
	}

	// Construire perPlayer : main + chaque teammate (filtres appliques).
	perPlayer := map[string][]canonical.PlayerMatchRow{s.gamertag: mainFiltered}
	for gt, rows := range teammateRows {
		filtered := rows
		if filters != nil {
			c := filters.Cascade
			filtered = squadagg.FilterRowsByCascade(filtered, c.ExperienceTypes, c.Playlists, c.Maps, c.Modes)
		}
		if len(sessionMatchIDs) > 0 {
			kept := make([]canonical.PlayerMatchRow, 0, len(filtered))
			for _, r := range filtered {
				if sessionMatchIDs[r.Summary.MatchID] {
					kept = append(kept, r)
				}
			}
			filtered = kept
		}
		perPlayer[gt] = filtered
	}

	squadOrder := squadagg.BuildSquadOrder(s.gamertag, selectedGamertags)
	gtToXUID := squadagg.ExtractSquadXUIDs(squadOrder, perPlayer)
	sharedMatches := squadagg.IntersectByMatchID(perPlayer)
	// Composition EXACTE (maillon 2) : écarte du briefing les matchs où un
	// coéquipier connu hors sélection était sur l'équipe du main. Désactivé
	// (no-op) si le chargement de l'équipe alliée a échoué en amont.
	sharedMatches = compFilter.applyShared(sharedMatches)

	header := squadagg.BuildSquadHeader(ctx, s.gamertag, gtToXUID, sharedMatches)
	// Mode degrade : si aucun match partage, le briefing repasse en mode solo
	// pour rester utile (sinon SoloKPIs serait nil et la section disparaitrait).
	if header.SoloKPIs == nil && len(mainFiltered) > 0 {
		kpis := analysis.ComputeKPIStats(mainFiltered, games.EffectiveHpToKill(s.titleSlug))
		header.SoloKPIs = &kpis
	}
	return header
}

// loadTeammatesCanonicalParallel charge les canonical PlayerMatchRow pour
// chaque gamertag en parallele via errgroup. Capability absente est ignoree
// silencieusement (le teammate sera juste absent du resultat).
//
// Utilise squadLoader.LoadFor (resolution dynamique par gamertag) plutot que
// playerMatchesRepo (qui est bound au main et ignore l'arg gamertag).
// Si squadLoader est nil, retourne une map vide → mode solo dans le briefing.
func (s *TeammatesService) loadTeammatesCanonicalParallel(
	ctx context.Context,
	gamertags []string,
) (map[string][]canonical.PlayerMatchRow, error) {
	if s.squadLoader == nil {
		return map[string][]canonical.PlayerMatchRow{}, nil
	}
	g, gctx := errgroup.WithContext(ctx)
	var mu sync.Mutex
	out := make(map[string][]canonical.PlayerMatchRow, len(gamertags))
	for _, gt := range gamertags {
		gt := gt
		g.Go(func() error {
			rows, err := s.squadLoader.LoadFor(gctx, s.titleSlug, gt, port.PlayerMatchFilters{})
			if err != nil {
				if errors.Is(err, games.ErrCapabilityNotSupported) {
					return nil
				}
				return fmt.Errorf("LoadFor(%s): %w", gt, err)
			}
			mu.Lock()
			out[gt] = rows
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return out, nil
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

// extractSynthesisSessionLabels collecte les sessions uniques en sÃƒÂ©parant solo / escouade,
// calcule les bornes temporelles, agrÃƒÂ¨ge les expÃƒÂ©riences et playlists prÃƒÂ©sentes, et trie par StartedAt DESC.
func extractSynthesisSessionLabels(matches []legacymatch.SynthesisMatchRow) domain.SessionLabelsList {
	type meta struct {
		startedAt   time.Time
		endedAt     time.Time
		experiences map[string]struct{}
		playlists   map[string]struct{}
		// count : matchs de la session dans le scope (solo OU escouade). Sans
		// coéquipier sélectionné, c'est ce compte que le sélecteur de sessions
		// affiche — même règle « ce que je vois est ce qui est compté ».
		count int
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
		entry.count++
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
				MatchCount:  entry.count,
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

// synthesisExperienceLabel dÃƒÂ©rive le label d'expÃƒÂ©rience d'un match (miroir de filters_service.go).
func synthesisExperienceLabel(m legacymatch.SynthesisMatchRow) string {
	if m.IsFirefight {
		return squadagg.ExpTypePVE
	}
	if m.IsRanked {
		return squadagg.ExpTypePVPRanked
	}
	return squadagg.ExpTypePVPUnranked
}

// filterSynthesisByCascade applique les filtres experience_types et playlists sur les matchs.
func filterSynthesisByCascade(matches []legacymatch.SynthesisMatchRow, c domain.CascadeFilter) []legacymatch.SynthesisMatchRow {
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

// filterSynthesisBySession filtre les matchs selon les sessions sÃƒÂ©lectionnÃƒÂ©es (union des labels).
// Slices vides Ã¢â€ â€™ tous les matchs retournÃƒÂ©s sans filtre.
// filterSynthesisByPeriodInput filtre les matchs selon une fenetre temporelle (start/end inclus).
// Le rail periode et le PeriodePill du FilterOmnibar ecrivent dans req.Filters.Period ;
// teammates_service doit appliquer ce filtre pour que la nav periode ait un effet sur
// l'ecran Escouade (charts, tableaux, KPIs). Aucune valeur posee = no-op.
func filterSynthesisByPeriodInput(matches []legacymatch.SynthesisMatchRow, p domain.PeriodInput) []legacymatch.SynthesisMatchRow {
	if p.StartDate == nil && p.EndDate == nil {
		return matches
	}
	out := matches[:0:0]
	for _, m := range matches {
		t := m.StartTime
		if p.StartDate != nil && t.Before(*p.StartDate) {
			continue
		}
		if p.EndDate != nil {
			end := p.EndDate.Add(24*time.Hour - time.Second)
			if t.After(end) {
				continue
			}
		}
		out = append(out, m)
	}
	return out
}

// filterSynthesisByPickedSessions filtre par labels presents dans
// req.Filters.Sessions.PickedSessions (rail nav, FilterOmnibar SessionPill,
// applySessionLabels squad). SynthesisMatchRow ne porte que SessionLabel ;
// les valeurs envoyees par le frontend doivent donc etre des labels (cf. fix
// goToPrevSession qui ecrit target.label, applySessionLabels qui propage les
// labels du SessionMultiSelect). Slice vide = no-op.
func filterSynthesisByPickedSessions(matches []legacymatch.SynthesisMatchRow, pickedSessions []string) []legacymatch.SynthesisMatchRow {
	if len(pickedSessions) == 0 {
		return matches
	}
	keep := make(map[string]struct{}, len(pickedSessions))
	for _, lbl := range pickedSessions {
		keep[lbl] = struct{}{}
	}
	out := matches[:0:0]
	for _, m := range matches {
		if m.SessionLabel == nil {
			continue
		}
		if _, ok := keep[*m.SessionLabel]; ok {
			out = append(out, m)
		}
	}
	return out
}

func filterSynthesisBySession(
	matches []legacymatch.SynthesisMatchRow,
	pickedSolo []string,
	pickedSquad []string,
) []legacymatch.SynthesisMatchRow {
	if len(pickedSolo) == 0 && len(pickedSquad) == 0 {
		return matches
	}
	filtered := make([]legacymatch.SynthesisMatchRow, 0, len(matches))
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
