package mapper

import (
	"fmt"

	"levelup/go-api/internal/openspartan"
)

// MapMedals walks every PlayerTeamStats CoreStats.Medals slice and produces
// one MedalEarnedRow per (xuid, medal_name_id), aggregating counts across
// multiple PlayerTeamStats entries (mid-match team switches).
//
// Returns nil rows (not error) when the match has no medals — many short
// matches end without anyone earning a medal.
func MapMedals(ms openspartan.MatchStats) ([]MedalEarnedRow, error) {
	if ms.MatchID == "" {
		return nil, fmt.Errorf("%w: missing MatchId", ErrInvalidMatch)
	}
	type key struct {
		xuid    string
		medalID int64
	}
	counts := make(map[key]int)
	for _, p := range ms.Players {
		if p.PlayerType != 1 {
			continue
		}
		xuid := openspartan.ParseXUID(p.PlayerID)
		if xuid == "" {
			continue
		}
		for _, pts := range p.PlayerTeamStats {
			for _, m := range pts.Stats.CoreStats.Medals {
				if m.NameID == 0 || m.Count == 0 {
					continue
				}
				counts[key{xuid: xuid, medalID: int64(m.NameID)}] += m.Count
			}
		}
	}
	if len(counts) == 0 {
		return nil, nil
	}
	rows := make([]MedalEarnedRow, 0, len(counts))
	for k, c := range counts {
		rows = append(rows, MedalEarnedRow{
			MatchID:     ms.MatchID,
			XUID:        k.xuid,
			MedalNameID: k.medalID,
			Count:       int16(c),
		})
	}
	return rows, nil
}
