// Package service — career_service_highlights.go : section "Matchs marquants"
// de la page Carrière (GetHighlightMatchIDs + cascade counts + i18n FR).
// Découpé de career_service.go (god-file split, refactor 2026-05-27).
package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// GetHighlightMatchIDs retourne les match_ids triés (best d'abord, worst
// ensuite) des matchs marquants — 15 + 15 — avec les cascade counts pour
// les dropdowns Expérience / Saisons. Le handler enrichit ensuite les IDs
// via MatchHistoryService pour produire des ExplorerMatchesRow complets.
func (s *CareerService) GetHighlightMatchIDs(ctx context.Context, input domain.HighlightFilterInput) (domain.HighlightMatchesData, error) {
	// Résout les SeasonIDs sélectionnés en fenêtres temporelles via le catalog.
	catalog := s.loadSeasonCatalog(ctx)
	selectedRanges, _ := resolveSeasonRanges(catalog, input.SeasonIDs)

	experience := normalizeExperience(input.Experience)

	// Pool complet d'abord — sert (a) au calcul des cascade counts et
	// (b) à l'expansion des labels affichés (sélection utilisateur) en raw
	// sources COALESCE(...) pour la clause SQL du Q9b.
	pool, perr := s.repo.GetHighlightPool(ctx)
	if perr != nil {
		slog.WarnContext(ctx, "career.highlight.pool_load_failed", "err", perr)
	}

	// Override des labels EN → FR depuis les tables metadata (best-effort).
	applyHighlightPoolFRTranslations(ctx, s.repo, pool)

	filters := domain.CareerHighlightFilters{
		Experience:       experience,
		SeasonRanges:     selectedRanges,
		ModeRawSources:   expandModeUIsToRawSources(pool, input.ModeUIs),
		PlaylistNamesRaw: expandPlaylistNamesToRaw(pool, input.PlaylistNames),
	}

	rows, err := s.repo.GetHighlightMatchIDs(ctx, filters)
	if err != nil {
		return domain.HighlightMatchesData{}, fmt.Errorf("CareerService.GetHighlightMatchIDs: %w", err)
	}

	return domain.HighlightMatchesData{
		Rows:                rows,
		AvailableExperience: computeHighlightAvailableExperience(pool, selectedRanges, input.ModeUIs, input.PlaylistNames),
		AvailableSeasons:    computeHighlightAvailableSeasons(pool, experience, catalog, input.ModeUIs, input.PlaylistNames),
		AvailableModes:      computeHighlightAvailableModes(pool, experience, selectedRanges, input.PlaylistNames),
		AvailablePlaylists:  computeHighlightAvailablePlaylists(pool, experience, selectedRanges, input.ModeUIs),
	}, nil
}

// applyHighlightPoolFRTranslations enrichit en place les labels FR du pool :
//   - ModeUI : si la valeur normalisée a une traduction dans mode_name_tr (lang='fr'),
//     remplace la valeur EN par la FR.
//   - PlaylistName : si playlist_id a un nom FR dans asset_translations, le préfère
//     à la valeur brute (qui peut être EN si playlist_name_fr était NULL en DB).
//
// Best-effort : silencieux si Metadata absent. Pattern aligné sur home_repo.
func applyHighlightPoolFRTranslations(ctx context.Context, repo port.CareerRepository, pool []domain.HighlightMatchPoolRow) {
	if len(pool) == 0 {
		return
	}

	// Modes : collecter les ModeUI distincts (post-NormalizeModeLabel), charger
	// les traductions FR, override en place.
	modeENSet := make(map[string]struct{})
	for _, m := range pool {
		if m.ModeUI != "" {
			modeENSet[m.ModeUI] = struct{}{}
		}
	}
	if len(modeENSet) > 0 {
		modeENs := make([]string, 0, len(modeENSet))
		for k := range modeENSet {
			modeENs = append(modeENs, k)
		}
		if modeFR, err := repo.LoadModeTranslationsFR(ctx, modeENs); err == nil && len(modeFR) > 0 {
			for i := range pool {
				if fr, ok := modeFR[pool[i].ModeUI]; ok && fr != "" {
					pool[i].ModeUI = fr
				}
			}
		} else if err != nil {
			slog.WarnContext(ctx, "career.highlight.mode_fr_load_failed", "err", err)
		}
	}

	// Playlists : collecter les playlist_ids distincts, charger les traductions
	// asset_translations FR, override le label si la valeur brute est manquante
	// ou identique à l'EN (placeholder COALESCE).
	playlistIDSet := make(map[string]struct{})
	for _, m := range pool {
		if m.PlaylistID != "" {
			playlistIDSet[m.PlaylistID] = struct{}{}
		}
	}
	if len(playlistIDSet) > 0 {
		ids := make([]string, 0, len(playlistIDSet))
		for k := range playlistIDSet {
			ids = append(ids, k)
		}
		if plFR, err := repo.LoadPlaylistAssetTranslationsFR(ctx, ids); err == nil && len(plFR) > 0 {
			for i := range pool {
				fr := strings.TrimSpace(plFR[pool[i].PlaylistID])
				if fr == "" {
					continue
				}
				// Override si raw est vide OU identique au FR (donc l'EN n'a pas
				// vraiment de FR distinct).
				if pool[i].PlaylistName == "" || strings.EqualFold(pool[i].PlaylistName, fr) {
					pool[i].PlaylistName = fr
					continue
				}
				// Override aussi si la valeur brute est l'EN (heuristique : la
				// valeur brute est en EN si elle n'est pas déjà la FR de l'asset).
				// Comme on n'a pas l'EN séparément ici, on force le FR par asset
				// (source de vérité côté metadata).
				pool[i].PlaylistName = fr
			}
		} else if err != nil {
			slog.WarnContext(ctx, "career.highlight.playlist_fr_load_failed", "err", err)
		}
	}
}

// expandPlaylistNamesToRaw résout les labels affichés (FR ou EN) sélectionnés
// en l'ensemble des valeurs brutes COALESCE(playlist_name_fr, playlist_name) du
// pool. Sert au filtre SQL `COALESCE IN (...)` côté repo. Renvoie nil si vide.
func expandPlaylistNamesToRaw(pool []domain.HighlightMatchPoolRow, selected []string) []string {
	if len(selected) == 0 {
		return nil
	}
	wanted := make(map[string]struct{}, len(selected))
	for _, s := range selected {
		wanted[s] = struct{}{}
	}
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, m := range pool {
		if _, ok := wanted[m.PlaylistName]; !ok {
			continue
		}
		if m.PlaylistNameRaw == "" {
			continue
		}
		if _, dup := seen[m.PlaylistNameRaw]; dup {
			continue
		}
		seen[m.PlaylistNameRaw] = struct{}{}
		out = append(out, m.PlaylistNameRaw)
	}
	return out
}

// expandModeUIsToRawSources résout les labels normalisés sélectionnés
// (ex. "Slayer") en l'ensemble des valeurs brutes COALESCE(pair_name_fr,
// pair_name) du pool qui se normalisent vers ces labels. Sert au filtre SQL
// `COALESCE(NULLIF(r.pair_name_fr, ”), r.pair_name) IN (...)` côté repo.
// Renvoie nil si selection vide (= pas de filtre Modes).
func expandModeUIsToRawSources(pool []domain.HighlightMatchPoolRow, selected []string) []string {
	if len(selected) == 0 {
		return nil
	}
	wanted := make(map[string]struct{}, len(selected))
	for _, s := range selected {
		wanted[s] = struct{}{}
	}
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, m := range pool {
		if _, ok := wanted[m.ModeUI]; !ok {
			continue
		}
		if m.ModeUISource == "" {
			continue
		}
		if _, dup := seen[m.ModeUISource]; dup {
			continue
		}
		seen[m.ModeUISource] = struct{}{}
		out = append(out, m.ModeUISource)
	}
	return out
}

// loadSeasonCatalog charge le catalog via SeasonsCatalog injecté. Retourne nil
// si non câblé (dégradation gracieuse — pas de cascade saisons).
func (s *CareerService) loadSeasonCatalog(ctx context.Context) []SeasonCatalogEntry {
	if s.seasonsCatalog == nil || s.titleSlug == "" {
		return nil
	}
	return s.seasonsCatalog.Load(ctx, s.titleSlug)
}

// normalizeExperience clamp la valeur d'entrée sur les 3 valeurs autorisées.
// Toute autre valeur (vide, "tous", etc.) → "all" (= pas de filtre).
func normalizeExperience(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case scopeRanked:
		return scopeRanked
	case scopeUnranked:
		return scopeUnranked
	default:
		return "all"
	}
}

// resolveSeasonRanges projette les seasonIDs sélectionnés en SeasonTimeRange
// via le catalog. Retourne aussi la liste des IDs qui ont matché (pour debug).
// IDs inconnus du catalog silencieusement ignorés.
func resolveSeasonRanges(catalog []SeasonCatalogEntry, seasonIDs []string) ([]domain.SeasonTimeRange, []string) {
	if len(seasonIDs) == 0 || len(catalog) == 0 {
		return nil, nil
	}
	wanted := make(map[string]struct{}, len(seasonIDs))
	for _, id := range seasonIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			wanted[id] = struct{}{}
		}
	}
	ranges := make([]domain.SeasonTimeRange, 0, len(wanted))
	matched := make([]string, 0, len(wanted))
	for _, e := range catalog {
		if _, ok := wanted[e.ID]; !ok {
			continue
		}
		ranges = append(ranges, domain.SeasonTimeRange{Start: e.Start, End: e.End})
		matched = append(matched, e.ID)
	}
	return ranges, matched
}

// matchPassesStringList : true si val est dans list, ou si list est vide (= pas de filtre).
func matchPassesStringList(val string, list []string) bool {
	if len(list) == 0 {
		return true
	}
	for _, s := range list {
		if s == val {
			return true
		}
	}
	return false
}

// computeHighlightAvailableExperience calcule les counts cascade-aware pour
// la dropdown Expérience : on respecte Saisons + Modes + Playlists, mais pas
// le filtre Expérience courant (sinon on n'aurait que le count de l'option active).
func computeHighlightAvailableExperience(pool []domain.HighlightMatchPoolRow, seasonRanges []domain.SeasonTimeRange, modeUIs, playlistNames []string) []domain.HighlightExperienceCount {
	counts := struct{ all, ranked, unranked int }{}
	for _, m := range pool {
		if !matchInSeasonRanges(m.StartTime, seasonRanges) {
			continue
		}
		if !matchPassesStringList(m.ModeUI, modeUIs) {
			continue
		}
		if !matchPassesStringList(m.PlaylistName, playlistNames) {
			continue
		}
		counts.all++
		if m.IsRanked {
			counts.ranked++
		} else {
			counts.unranked++
		}
	}
	return []domain.HighlightExperienceCount{
		{Value: "all", Count: counts.all},
		{Value: scopeRanked, Count: counts.ranked},
		{Value: scopeUnranked, Count: counts.unranked},
	}
}

// computeHighlightAvailableSeasons calcule les counts par saison du catalog
// en respectant Expérience + Modes + Playlists, mais pas le filtre Saisons
// (la dropdown affiche le count par saison si on coche cette saison).
func computeHighlightAvailableSeasons(pool []domain.HighlightMatchPoolRow, experience string, catalog []SeasonCatalogEntry, modeUIs, playlistNames []string) []domain.HighlightSeasonCount {
	if len(catalog) == 0 {
		return nil
	}
	out := make([]domain.HighlightSeasonCount, 0, len(catalog))
	for _, season := range catalog {
		count := 0
		for _, m := range pool {
			if !matchPassesExperience(m.IsRanked, experience) {
				continue
			}
			if !matchPassesStringList(m.ModeUI, modeUIs) {
				continue
			}
			if !matchPassesStringList(m.PlaylistName, playlistNames) {
				continue
			}
			if m.StartTime == nil {
				continue
			}
			if m.StartTime.Before(season.Start) {
				continue
			}
			if season.End != nil && !m.StartTime.Before(*season.End) {
				continue
			}
			count++
		}
		out = append(out, domain.HighlightSeasonCount{Value: season.ID, Count: count})
	}
	return out
}

// computeHighlightAvailableModes calcule les counts par mode (pair_name) en
// respectant Expérience + Saisons + Playlists, mais pas le filtre Modes courant.
// Trié par count DESC pour une UX cohérente.
func computeHighlightAvailableModes(pool []domain.HighlightMatchPoolRow, experience string, seasonRanges []domain.SeasonTimeRange, playlistNames []string) []domain.HighlightModeCount {
	counts := make(map[string]int)
	for _, m := range pool {
		if m.ModeUI == "" {
			continue
		}
		if !matchPassesExperience(m.IsRanked, experience) {
			continue
		}
		if !matchInSeasonRanges(m.StartTime, seasonRanges) {
			continue
		}
		if !matchPassesStringList(m.PlaylistName, playlistNames) {
			continue
		}
		counts[m.ModeUI]++
	}
	out := make([]domain.HighlightModeCount, 0, len(counts))
	for mode, c := range counts {
		out = append(out, domain.HighlightModeCount{Value: mode, Count: c})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	return out
}

// computeHighlightAvailablePlaylists calcule les counts par playlist en
// respectant Expérience + Saisons + Modes, mais pas le filtre Playlists courant.
// Trié par count DESC. Les matchs sans playlist (chaîne vide) sont ignorés.
func computeHighlightAvailablePlaylists(pool []domain.HighlightMatchPoolRow, experience string, seasonRanges []domain.SeasonTimeRange, modeUIs []string) []domain.HighlightPlaylistCount {
	counts := make(map[string]int)
	for _, m := range pool {
		if m.PlaylistName == "" {
			continue
		}
		if !matchPassesExperience(m.IsRanked, experience) {
			continue
		}
		if !matchInSeasonRanges(m.StartTime, seasonRanges) {
			continue
		}
		if !matchPassesStringList(m.ModeUI, modeUIs) {
			continue
		}
		counts[m.PlaylistName]++
	}
	out := make([]domain.HighlightPlaylistCount, 0, len(counts))
	for pl, c := range counts {
		out = append(out, domain.HighlightPlaylistCount{Value: pl, Count: c})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	return out
}

// matchPassesExperience : true si le match passe le filtre Expérience.
func matchPassesExperience(isRanked bool, experience string) bool {
	switch experience {
	case scopeRanked:
		return isRanked
	case "unranked":
		return !isRanked
	default: // "all" ou inconnu
		return true
	}
}

// matchInSeasonRanges : true si startTime tombe dans au moins une fenêtre
// de seasonRanges. Si seasonRanges est vide → true (pas de filtre).
func matchInSeasonRanges(startTime *time.Time, seasonRanges []domain.SeasonTimeRange) bool {
	if len(seasonRanges) == 0 {
		return true
	}
	if startTime == nil {
		return false
	}
	for _, w := range seasonRanges {
		if startTime.Before(w.Start) {
			continue
		}
		if w.End != nil && !startTime.Before(*w.End) {
			continue
		}
		return true
	}
	return false
}
