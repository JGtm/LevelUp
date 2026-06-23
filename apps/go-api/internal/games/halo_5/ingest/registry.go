package ingest

import (
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// MatchRegistryRowFromSummary projette un résumé de match canonique en ligne
// match_registry (l'ANCRE du persist shared — SharedPersister.Persist no-op sans
// elle). Title-agnostique : remplit ce que le résumé fournit, laisse le reste à
// nil/zéro (l'API Halo n'expose pas toujours tout ; les colonnes sont nullables).
// firstSyncBy identifie le joueur dont la consultation a déclenché la capture.
func MatchRegistryRowFromSummary(s canonical.MatchSummary, firstSyncBy string) domain.MatchRegistryRow {
	playlistID, playlistName, playlistVer := assetRefParts(s.Playlist)
	mapID, mapName, mapVer := assetRefParts(s.Map)
	gvID, gvName, gvVer := assetRefParts(s.GameVariant)
	pairID, pairName, _ := assetRefParts(s.PairMode)

	row := domain.MatchRegistryRow{
		MatchID:              s.MatchID,
		StartTime:            s.StartedAtUTC,
		DurationSeconds:      s.DurationSeconds,
		PlaylistID:           playlistID,
		PlaylistName:         playlistName,
		PlaylistVersionID:    playlistVer,
		MapID:                mapID,
		MapName:              mapName,
		MapVersionID:         mapVer,
		GameVariantID:        gvID,
		GameVariantName:      gvName,
		GameVariantVersionID: gvVer,
		PairID:               pairID,
		PairName:             pairName,
		IsRanked:             derefBool(s.IsRanked),
		IsFirefight:          derefBool(s.IsPvE),
		FirstSyncBy:          firstSyncBy,
	}
	if s.PairMode != nil {
		row.ModeCategory = s.PairMode.DefaultLabel
	}
	// end_time = start + durée. L'API Halo 5 ne fournit pas d'horodatage de fin
	// explicite, mais la durée oui → end_time dérivable (le persister écrit aussi
	// end_time_utc). Sans lui, les consommateurs qui bornent [start, end]
	// (engagement loadMatchesForEngagement, corrélation média) excluent ces matchs.
	if s.DurationSeconds != nil && *s.DurationSeconds > 0 {
		end := s.StartedAtUTC.Add(time.Duration(*s.DurationSeconds) * time.Second)
		row.EndTime = &end
	}
	return row
}

func strptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// assetRefParts éclate une référence d'asset en (id, label, version) nullables.
func assetRefParts(a *canonical.AssetReference) (id, name, version *string) {
	if a == nil {
		return nil, nil, nil
	}
	return strptr(a.ID), strptr(a.DefaultLabel), strptr(a.VersionID)
}

func derefBool(b *bool) bool { return b != nil && *b }
