package ingest

import (
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/persist"
)

// MapKillPositions projette les positions monde (Vec3) des events `kill` en lignes
// kill_positions, jointes au kill par (match_id, killer_xuid, time_ms). Halo 5 les
// fournit nativement (KillerLoc/VictimLoc) ; un kill sans aucune position est
// ignoré, et chaque coordonnée absente reste nil (kill avec seulement la position
// du tueur, p.ex.).
func MapKillPositions(
	matchID string,
	events []canonical.MatchEvent,
	resolveXUID func(gamertag string) string,
) []persist.KillPositionInsert {
	var out []persist.KillPositionInsert
	for i := range events {
		ev := events[i]
		if ev.Type != canonical.MatchEventKill || ev.Killer == nil {
			continue
		}
		if ev.KillerLoc == nil && ev.VictimLoc == nil {
			continue // aucune position → rien à persister
		}
		row := persist.KillPositionInsert{
			MatchID:    matchID,
			KillerXUID: resolveXUID(ev.Killer.Gamertag),
			TimeMS:     ev.TimeMs,
		}
		if ev.KillerLoc != nil {
			kx, ky, kz := ev.KillerLoc.X, ev.KillerLoc.Y, ev.KillerLoc.Z
			row.KillerX, row.KillerY, row.KillerZ = &kx, &ky, &kz
		}
		if ev.VictimLoc != nil {
			vx, vy, vz := ev.VictimLoc.X, ev.VictimLoc.Y, ev.VictimLoc.Z
			row.VictimX, row.VictimY, row.VictimZ = &vx, &vy, &vz
		}
		out = append(out, row)
	}
	return out
}
