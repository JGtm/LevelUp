// Package sync — transforms.go : transformation JSON API Halo → structs DB.
//
// Portage de transformers/_match.py + transformers/_medals.py (Python).
// Toutes les fonctions sont pures (stateless, sans accès DB).
//
// Structures produites :
//
//	MatchRegistryRow  → match_registry (shared)
//	ParticipantRow    → match_participants (shared)
//	MedalRow          → medals_earned (shared)
//
// Helpers privés (parseurs, regex, accésseurs primitifs) → transforms_helpers.go
package sync

import (
	"fmt"
	"time"
)

// ──────────────────────────────────────────────────────────────────────────────
// Structs de sortie
// ──────────────────────────────────────────────────────────────────────────────

// MatchRegistryRow représente une ligne dans match_registry (shared).
type MatchRegistryRow struct {
	MatchID                 string
	StartTime               time.Time
	EndTime                 *time.Time
	PlaylistID              *string
	PlaylistName            *string
	PlaylistVersionID       *string
	MapID                   *string
	MapName                 *string
	MapVersionID            *string
	PairID                  *string
	PairName                *string
	PairVersionID           *string
	GameVariantID           *string
	GameVariantName         *string
	GameVariantVersionID    *string
	ModeCategory            string
	IsRanked                bool
	IsFirefight             bool
	DurationSeconds         *int
	PlayableDurationSeconds *int
	RealStartTime           *time.Time
	Team0Score              *int
	Team1Score              *int
	FirstSyncBy             string
}

// ParticipantRow représente une ligne dans match_participants (shared).
type ParticipantRow struct {
	MatchID           string
	XUID              string
	Gamertag          *string
	TeamID            *int
	Outcome           *int
	Rank              *int
	Score             *int
	Kills             *int
	Deaths            *int
	Assists           *int
	ShotsFired        *int
	ShotsHit          *int
	DamageDealt       *float64
	DamageTaken       *float64
	KDA               *float64
	Accuracy          *float64
	PersonalScore     *int
	TimePlayedSeconds *int
	AvgLifeSeconds    *float64
	KillsExpected     *float64
	DeathsExpected    *float64
	KillsStddev       *float64
	TeamMMR           *float64
	EnemyMMR          *float64
	HeadshotKills     *int
}

// MedalRow représente une ligne dans medals_earned (shared).
type MedalRow struct {
	MatchID     string
	XUID        string
	MedalNameID int64
	Count       int
}

// ──────────────────────────────────────────────────────────────────────────────
// ExtractRegistry
// ──────────────────────────────────────────────────────────────────────────────

// ExtractRegistry extrait les données de match_registry depuis le JSON brut.
// Portage de extract_match_registry_data() (Python transformers/_match.py).
func ExtractRegistry(matchJSON map[string]any, syncBy string) (*MatchRegistryRow, error) {
	matchID, _ := matchJSON["MatchId"].(string)
	if matchID == "" {
		return nil, ErrMissingField("MatchId")
	}

	matchInfo, ok := matchJSON["MatchInfo"].(map[string]any)
	if !ok {
		return nil, ErrMissingField("MatchInfo")
	}

	startTime, err := parseISO(asString(matchInfo["StartTime"]))
	if err != nil {
		return nil, fmt.Errorf("StartTime: %w", err)
	}

	row := &MatchRegistryRow{
		MatchID:      matchID,
		StartTime:    startTime,
		ModeCategory: "Other",
		FirstSyncBy:  syncBy,
	}

	// Assets
	row.PlaylistID = strPtr(extractAssetID(matchInfo, "Playlist"))
	row.PlaylistName = strPtr(extractPublicName(matchInfo, "Playlist"))
	row.PlaylistVersionID = strPtr(extractVersionID(matchInfo, "Playlist"))
	row.MapID = strPtr(extractAssetID(matchInfo, "MapVariant"))
	row.MapName = strPtr(extractPublicName(matchInfo, "MapVariant"))
	row.MapVersionID = strPtr(extractVersionID(matchInfo, "MapVariant"))
	row.PairID = strPtr(extractAssetID(matchInfo, "PlaylistMapModePair"))
	row.PairName = strPtr(extractPublicName(matchInfo, "PlaylistMapModePair"))
	row.PairVersionID = strPtr(extractVersionID(matchInfo, "PlaylistMapModePair"))
	row.GameVariantID = strPtr(extractAssetID(matchInfo, "UgcGameVariant"))
	row.GameVariantName = strPtr(extractPublicName(matchInfo, "UgcGameVariant"))
	row.GameVariantVersionID = strPtr(extractVersionID(matchInfo, "UgcGameVariant"))

	// Fallback nom → ID
	row.PlaylistName = coalesceStrPtr(row.PlaylistName, row.PlaylistID)
	row.MapName = coalesceStrPtr(row.MapName, row.MapID)
	row.PairName = coalesceStrPtr(row.PairName, row.PairID)
	row.GameVariantName = coalesceStrPtr(row.GameVariantName, row.GameVariantID)

	// Flags
	row.IsRanked = isRankedPlaylist(matchInfo)
	row.IsFirefight = isFirefightMatch(matchInfo)
	if row.PairName != nil {
		row.ModeCategory = determineModeCategory(*row.PairName)
	}

	// Durées
	row.DurationSeconds = parsePTDuration(asString(matchInfo["Duration"]))
	row.PlayableDurationSeconds = parsePTDuration(asString(matchInfo["PlayableDuration"]))

	// end_time
	if et, err2 := parseISO(asString(matchInfo["EndTime"])); err2 == nil {
		row.EndTime = &et
	} else if row.DurationSeconds != nil {
		t := startTime.Add(time.Duration(*row.DurationSeconds) * time.Second)
		row.EndTime = &t
	}

	// real_start_time
	if row.DurationSeconds != nil && row.PlayableDurationSeconds != nil {
		countdown := *row.DurationSeconds - *row.PlayableDurationSeconds
		if countdown >= 0 {
			rst := startTime.Add(time.Duration(countdown) * time.Second)
			row.RealStartTime = &rst
		}
	}

	// Team scores
	t0, t1 := extractTeamScoresByID(matchJSON)
	row.Team0Score = t0
	row.Team1Score = t1

	return row, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// ExtractParticipants
// ──────────────────────────────────────────────────────────────────────────────

// ExtractParticipants extrait tous les participants d'un match.
// Portage de extract_participants() (Python transformers/_match.py).
func ExtractParticipants(matchJSON map[string]any) []ParticipantRow {
	matchID, _ := matchJSON["MatchId"].(string)
	if matchID == "" {
		return nil
	}

	players, _ := matchJSON["Players"].([]any)
	var rows []ParticipantRow //nolint:prealloc
	seen := map[string]bool{}

	for _, p := range players {
		player, ok := p.(map[string]any)
		if !ok {
			continue
		}
		xuid := extractXUID(asString(player["PlayerId"]))
		if xuid == "" || seen[xuid] {
			continue
		}
		seen[xuid] = true

		row := ParticipantRow{
			MatchID: matchID,
			XUID:    xuid,
		}

		// team_id, outcome, rank
		if v, ok := player["LastTeamId"].(float64); ok {
			n := int(v)
			row.TeamID = &n
		}
		if v, ok := player["Outcome"].(float64); ok {
			n := int(v)
			row.Outcome = &n
		}
		if v, ok := player["Rank"].(float64); ok {
			n := int(v)
			row.Rank = &n
		}

		// CoreStats depuis PlayerTeamStats[0]
		core := findCoreStats(player)
		if core != nil {
			row.Score = intPtrFrom(core, "PersonalScore")
			row.Kills = intPtrFrom(core, "Kills")
			row.Deaths = intPtrFrom(core, "Deaths")
			row.Assists = intPtrFrom(core, "Assists")
			row.ShotsFired = intPtrFrom(core, "ShotsFired")
			row.ShotsHit = intPtrFrom(core, "ShotsHit")
			row.DamageDealt = floatPtrFrom(core, "DamageDealt")
			row.DamageTaken = floatPtrFrom(core, "DamageTaken")
			row.PersonalScore = intPtrFrom(core, "PersonalScore")
			row.AvgLifeSeconds = floatPtrFrom(core, "AverageLifeDuration")
			row.HeadshotKills = intPtrFrom(core, "HeadshotKills")

			// KDA dérivé
			if row.Kills != nil && row.Deaths != nil && row.Assists != nil {
				k, d, a := float64(*row.Kills), float64(*row.Deaths), float64(*row.Assists)
				if d == 0 {
					d = 1
				}
				kda := (k + a) / d
				row.KDA = &kda
			}

			// Accuracy dérivée
			if row.ShotsFired != nil && *row.ShotsFired > 0 && row.ShotsHit != nil {
				acc := float64(*row.ShotsHit) / float64(*row.ShotsFired) * 100.0
				row.Accuracy = &acc
			}
		}

		// Gamertag
		gt := asString(player["Gamertag"])
		if gt == "" {
			gt = asString(player["PlayerName"])
		}
		if gt != "" {
			row.Gamertag = &gt
		}

		// time_played_seconds (depuis MatchInfo ou PlayerTeamStats duration)
		if dur := parsePTDuration(asString(player["ParticipationInfo.TimePlayed"])); dur != nil {
			row.TimePlayedSeconds = dur
		}

		rows = append(rows, row)
	}
	return rows
}

// ──────────────────────────────────────────────────────────────────────────────
// ExtractMedals
// ──────────────────────────────────────────────────────────────────────────────

// ExtractMedals extrait les médailles de TOUS les joueurs d'un match.
// Portage de extract_all_medals() (Python transformers/_medals.py).
func ExtractMedals(matchJSON map[string]any) []MedalRow {
	matchID, _ := matchJSON["MatchId"].(string)
	if matchID == "" {
		return nil
	}

	players, _ := matchJSON["Players"].([]any)
	var rows []MedalRow //nolint:prealloc

	for _, p := range players {
		player, ok := p.(map[string]any)
		if !ok {
			continue
		}
		xuid := extractXUID(asString(player["PlayerId"]))
		if xuid == "" {
			continue
		}
		core := findCoreStats(player)
		if core == nil {
			continue
		}
		medals, _ := core["Medals"].([]any)
		for _, m := range medals {
			medal, ok := m.(map[string]any)
			if !ok {
				continue
			}
			nameID := int64From(medal, "NameId")
			count := intFrom(medal, "Count")
			if nameID == 0 || count == 0 {
				continue
			}
			rows = append(rows, MedalRow{
				MatchID:     matchID,
				XUID:        xuid,
				MedalNameID: nameID,
				Count:       count,
			})
		}
	}
	return rows
}
