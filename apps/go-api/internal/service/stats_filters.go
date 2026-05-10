package service

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// filterStatsMatchRows applique les filtres de contexte sur les StatsMatchRow.
func filterStatsMatchRows(rows []legacymatch.StatsMatchRow, f domain.FilterContextInput) []legacymatch.StatsMatchRow {
	filterRows := make([]domain.FilterMatchRow, len(rows))
	for i, r := range rows {
		filterRows[i] = domain.FilterMatchRow{
			MatchID:       r.MatchID,
			StartTime:     &r.StartTime,
			MapName:       &r.MapName,
			MapNameFR:     &r.MapNameFR,
			PairName:      &r.PairName,
			PairNameFR:    &r.PairNameFR,
			PlaylistName:  &r.PlaylistName,
			IsFirefight:   r.IsFirefight,
			IsRanked:      r.IsRanked,
			IsWithFriends: r.IsWithFriends,
			SessionID:     r.SessionID,
			SessionLabel:  r.SessionLabel,
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
