package service

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// filterStatsMatchRows applique les filtres de contexte sur les StatsMatchRow.
func filterStatsMatchRows(rows []legacymatch.StatsMatchRow, f domain.FilterContextInput) []legacymatch.StatsMatchRow {
	filterRows := make([]domain.FilterMatchRow, len(rows))
	for i, r := range rows {
		// Value playlist des options = FR canonique (parité cascade filters_service) →
		// préférer le FR, retomber sur l'EN si le titre n'a pas de traduction FR.
		// Variable locale (bloc de boucle) : son adresse est distincte à chaque
		// itération, contrairement à un champ de r si le coalesce dépendait du row.
		playlist := r.PlaylistName
		if r.PlaylistNameFR != "" {
			playlist = r.PlaylistNameFR
		}
		filterRows[i] = domain.FilterMatchRow{
			MatchID:           r.MatchID,
			StartTime:         &r.StartTime,
			MapName:           &r.MapName,
			MapNameFR:         &r.MapNameFR,
			PairName:          &r.PairName,
			PairNameFR:        &r.PairNameFR,
			PlaylistName:      &playlist,
			GameVariantName:   &r.GameVariantName,
			GameVariantNameFR: &r.GameVariantNameFR,
			IsFirefight:       r.IsFirefight,
			IsRanked:          r.IsRanked,
			IsWithFriends:     r.IsWithFriends,
			SessionID:         r.SessionID,
			SessionLabel:      r.SessionLabel,
		}
	}

	filtered := applyAllFilters(filterRows, f)
	keepIDs := make(map[string]struct{}, len(filtered))
	for _, row := range filtered {
		keepIDs[row.MatchID] = struct{}{}
	}

	out := make([]legacymatch.StatsMatchRow, 0, len(filtered))
	for _, row := range rows {
		if _, ok := keepIDs[row.MatchID]; ok {
			out = append(out, row)
		}
	}
	return out
}
