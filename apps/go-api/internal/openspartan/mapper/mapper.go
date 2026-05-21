package mapper

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/openspartan"
)

// MapMatch is the top-level entry point. Given a ParsedMatch yielded by the
// openspartan.Reader, it produces every row the v6 schema expects from one
// match payload (registry + participants + medals).
//
// Highlight events and xuid_aliases come from sibling OpenSpartan tables and
// are mapped via MapHighlight / MapXuidAliasFromRow respectively.
func MapMatch(pm *openspartan.ParsedMatch, opts MapOptions) (MappedMatch, error) {
	if pm == nil {
		return MappedMatch{}, fmt.Errorf("%w: nil ParsedMatch", ErrInvalidMatch)
	}
	if strings.TrimSpace(pm.MatchID) == "" {
		return MappedMatch{}, fmt.Errorf("%w: missing MatchId", ErrInvalidMatch)
	}
	if len(pm.Stats.Players) == 0 {
		return MappedMatch{}, fmt.Errorf("%w: empty Players for match %s", ErrInvalidMatch, pm.MatchID)
	}

	reg, err := MapRegistry(pm.Stats, opts)
	if err != nil {
		return MappedMatch{}, fmt.Errorf("map registry %s: %w", pm.MatchID, err)
	}
	parts, err := MapParticipants(pm.Stats, pm.PlayerStats, opts)
	if err != nil {
		return MappedMatch{}, fmt.Errorf("map participants %s: %w", pm.MatchID, err)
	}
	medals, err := MapMedals(pm.Stats)
	if err != nil {
		return MappedMatch{}, fmt.Errorf("map medals %s: %w", pm.MatchID, err)
	}
	reg.PlayerCount = int16(len(parts))
	return MappedMatch{Registry: reg, Participants: parts, Medals: medals}, nil
}

// MapRegistry projects a MatchStats payload into the match_registry row shape.
// It tolerates missing optional pieces (Playlist, PlaylistMapModePair, Teams)
// and leaves them nil.
func MapRegistry(ms openspartan.MatchStats, opts MapOptions) (MatchRegistryRow, error) {
	now := opts.now()
	if !ms.MatchInfo.StartTime.IsZero() && ms.MatchInfo.StartTime.After(now.Add(24*time.Hour)) {
		return MatchRegistryRow{}, fmt.Errorf("%w: start_time %s", ErrFutureMatch, ms.MatchInfo.StartTime)
	}
	durSec, err := DurationSeconds(ms.MatchInfo.Duration)
	if err != nil {
		return MatchRegistryRow{}, err
	}
	playableSec, err := DurationSeconds(ms.MatchInfo.PlayableDuration)
	if err != nil {
		return MatchRegistryRow{}, err
	}

	row := MatchRegistryRow{
		MatchID:                 ms.MatchID,
		StartTime:               ms.MatchInfo.StartTime,
		StartTimeUTC:            ms.MatchInfo.StartTime.UTC(),
		DurationSeconds:         durSec,
		PlayableDurationSeconds: playableSec,
		BackfillCompleted:       1,
		ParticipantsLoaded:      true,
		EventsLoaded:            true,
		MedalsLoaded:            true,
		FirstSyncBy:             opts.source(),
		FirstSyncAt:             now,
		LastUpdatedAt:           now,
	}
	assignRegistryEndTime(&row, ms.MatchInfo.EndTime)
	assignRegistryAssetIDs(&row, ms)
	assignRegistryTeamScores(&row, ms.Teams)
	return row, nil
}

// assignRegistryEndTime renseigne EndTime + EndTimeUTC si non-zéro.
func assignRegistryEndTime(row *MatchRegistryRow, endTime time.Time) {
	if endTime.IsZero() {
		return
	}
	row.EndTime = &endTime
	endUTC := endTime.UTC()
	row.EndTimeUTC = &endUTC
}

// assignRegistryAssetIDs renseigne MapID, GameVariantID, PlaylistID, PairID
// et leurs versions, en tolérant les champs optionnels manquants.
func assignRegistryAssetIDs(row *MatchRegistryRow, ms openspartan.MatchStats) {
	if id := strings.TrimSpace(ms.MatchInfo.MapVariant.AssetID); id != "" {
		row.MapID = &id
		row.MapVersionID = strPtrOrNil(ms.MatchInfo.MapVariant.VersionID)
	}
	if id := strings.TrimSpace(ms.MatchInfo.UgcGameVariant.AssetID); id != "" {
		row.GameVariantID = &id
		row.GameVariantVersionID = strPtrOrNil(ms.MatchInfo.UgcGameVariant.VersionID)
	}
	if ms.MatchInfo.Playlist != nil {
		if id := strings.TrimSpace(ms.MatchInfo.Playlist.AssetID); id != "" {
			row.PlaylistID = &id
			row.PlaylistVersionID = strPtrOrNil(ms.MatchInfo.Playlist.VersionID)
		}
	}
	if ms.MatchInfo.PlaylistMapModePair != nil {
		if id := strings.TrimSpace(ms.MatchInfo.PlaylistMapModePair.AssetID); id != "" {
			row.PairID = &id
			row.PairVersionID = strPtrOrNil(ms.MatchInfo.PlaylistMapModePair.VersionID)
		}
	}
	// SeasonID + IsRanked (Phase 9.5) : le payload OpenSpartan expose déjà ces
	// infos qui n'étaient pas exploitées historiquement (rows.go:29 "false by
	// default"). On les lit ici pour éviter que chaque import OpenSpartan
	// recrée le bug "0 ranked + season_id NULL" qui nécessite ensuite un
	// backfill migration.
	if sid := strings.TrimSpace(ms.MatchInfo.SeasonID); sid != "" {
		row.SeasonID = &sid
	}
	// Heuristique nom playlist (cohérente avec sync/transforms_helpers.go:isRankedPlaylist
	// + le fallback dans csr_backfill.go).
	if row.PlaylistName != nil && strings.Contains(strings.ToLower(*row.PlaylistName), "ranked") {
		row.IsRanked = true
	} else if ms.MatchInfo.Playlist != nil {
		// Si PlaylistName n'est pas encore résolu (peuplé par post-import recompute),
		// on essaie via PairName si déjà disponible (rare en import-time mais sûr).
		if row.PairName != nil && strings.HasPrefix(strings.ToLower(*row.PairName), "ranked:") {
			row.IsRanked = true
		}
	}
}

// assignRegistryTeamScores renseigne Team0/Team1 scores depuis le slice Teams.
func assignRegistryTeamScores(row *MatchRegistryRow, teams []openspartan.Team) {
	if len(teams) >= 1 {
		if s, ps, ok := teamScores(teams[0].Stats); ok {
			s16 := int16(s)
			row.Team0Score = &s16
			row.Team0PsScore = &ps
		}
	}
	if len(teams) >= 2 {
		if s, ps, ok := teamScores(teams[1].Stats); ok {
			s16 := int16(s)
			row.Team1Score = &s16
			row.Team1PsScore = &ps
		}
	}
}

// teamScores extracts Score and PersonalScore from a Teams[i].Stats raw
// payload. Returns ok=false when CoreStats is missing or malformed; the
// caller leaves the corresponding columns NULL in that case.
func teamScores(stats json.RawMessage) (score int, ps int, ok bool) {
	if len(stats) == 0 {
		return 0, 0, false
	}
	var probe struct {
		CoreStats struct {
			Score         int `json:"Score"`
			PersonalScore int `json:"PersonalScore"`
		} `json:"CoreStats"`
	}
	if err := json.Unmarshal(stats, &probe); err != nil {
		return 0, 0, false
	}
	return probe.CoreStats.Score, probe.CoreStats.PersonalScore, true
}

// strPtrOrNil returns nil when s is empty after trimming, &trimmed otherwise.
func strPtrOrNil(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
