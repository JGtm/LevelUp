package mapper

import (
	"fmt"

	"levelup/go-api/internal/openspartan"
)

// MapParticipants emits one MatchParticipantRow per human player (PlayerType=1)
// of a match. Bots are skipped. When playerStats is non-empty, TeamMmr is
// joined onto the corresponding XUID.
//
// Returns ErrInvalidMatch when MatchID is empty or no human participant was
// found (an all-bot match is unexpected for an OpenSpartan import).
func MapParticipants(
	ms openspartan.MatchStats,
	playerStats []openspartan.PlayerMatchStatsValue,
	opts MapOptions,
) ([]MatchParticipantRow, error) {
	if ms.MatchID == "" {
		return nil, fmt.Errorf("%w: missing MatchId", ErrInvalidMatch)
	}

	mmrByXUID := make(map[string]float64, len(playerStats))
	for _, ps := range playerStats {
		if ps.Result == nil {
			continue
		}
		xuid := openspartan.ParseXUID(ps.ID)
		if xuid == "" {
			continue
		}
		mmrByXUID[xuid] = ps.Result.TeamMmr
	}

	rows := make([]MatchParticipantRow, 0, len(ms.Players))
	for _, p := range ms.Players {
		if p.PlayerType != 1 {
			continue
		}
		xuid := openspartan.ParseXUID(p.PlayerID)
		if xuid == "" {
			continue
		}
		if len(p.PlayerTeamStats) == 0 {
			continue
		}
		// Use the last PlayerTeamStats entry: if a player switched teams
		// mid-match, the final team is the canonical one we report on.
		pts := p.PlayerTeamStats[len(p.PlayerTeamStats)-1]
		cs := pts.Stats.CoreStats

		timePlayed, _ := DurationSeconds(p.ParticipationInfo.TimePlayed)
		// avg_life parse is best-effort; on malformed input we leave the
		// column NULL rather than failing the whole match.
		avgLife, avgLifeErr := DurationSecondsFloat(cs.AverageLifeDuration)

		row := MatchParticipantRow{
			MatchID:           ms.MatchID,
			XUID:              xuid,
			Gamertag:          opts.resolveGamertag(xuid),
			TeamID:            pts.TeamID,
			Outcome:           p.Outcome,
			Rank:              int16(p.Rank),
			Score:             cs.Score,
			Kills:             int16(cs.Kills),
			Deaths:            int16(cs.Deaths),
			Assists:           int16(cs.Assists),
			KDA:               cs.KDA,
			Accuracy:          cs.Accuracy,
			ShotsFired:        cs.ShotsFired,
			ShotsHit:          cs.ShotsHit,
			DamageDealt:       float64(cs.DamageDealt),
			DamageTaken:       float64(cs.DamageTaken),
			PersonalScore:     cs.PersonalScore,
			TimePlayedSeconds: timePlayed,
			HeadshotKills:     int16(cs.HeadshotKills),
			GrenadeKills:      int16(cs.GrenadeKills),
			MeleeKills:        int16(cs.MeleeKills),
			PowerWeaponKills:  int16(cs.PowerWeaponKills),
		}
		if avgLifeErr == nil && avgLife > 0 {
			v := avgLife
			row.AvgLifeSeconds = &v
		}
		if cs.MaxKillingSpree > 0 {
			ks := int16(cs.MaxKillingSpree)
			row.MaxKillingSpree = &ks
		}
		if mmr, ok := mmrByXUID[xuid]; ok {
			v := mmr
			row.TeamMMR = &v
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%w: no human participants in match %s", ErrInvalidMatch, ms.MatchID)
	}
	return rows, nil
}
