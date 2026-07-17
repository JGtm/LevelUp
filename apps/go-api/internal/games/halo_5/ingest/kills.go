package ingest

import (
	"strconv"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/persist"
)

// MapKillEvents projette les events `kill` d'une timeline Halo 5 en :
//   - killer_victim_pairs : forme PAR-KILL (1 row/kill, killer→victim + time_ms),
//     le grain de référence choisi (l'agrégat par-paire en dérive).
//   - weapon_kills : l'arme par kill (clé killer xuid + time_ms), native Halo 5
//     (confidence "native", aucune réconciliation d'attribution comme le film HI).
//
// Contrairement à Halo Infinite (analysis.ComputeKillerVictimPairs qui reconstruit
// les paires par appariement temporel kill/death), la timeline Halo 5 porte
// killer ET victim NATIVEMENT — aucune reconstruction. Les kills sans attaquant
// (environnement/suicide → Killer nil) ne forment pas de paire et sont ignorés.
func MapKillEvents(
	matchID string,
	events []canonical.MatchEvent,
	resolveXUID func(gamertag string) string,
) (pairs []persist.KillerVictimInsert, weapons []persist.WeaponKillInsert) {
	for i := range events {
		ev := events[i]
		if ev.Type != canonical.MatchEventKill || ev.Killer == nil || ev.Victim == nil {
			continue
		}
		killerXUID := resolveXUID(ev.Killer.Gamertag)
		victimXUID := resolveXUID(ev.Victim.Gamertag)

		pairs = append(pairs, persist.KillerVictimInsert{
			MatchID:        matchID,
			KillerXUID:     killerXUID,
			KillerGamertag: ev.Killer.Gamertag,
			VictimXUID:     victimXUID,
			VictimGamertag: ev.Victim.Gamertag,
			Count:          1,
			TimeMS:         int64(ev.TimeMs),
		})

		// Arme par kill (native Halo 5) → weapon_kills, jointe au kill par
		// (match_id, killer xuid, time_ms). ev.Weapon.ID = stock id décimal.
		if ev.Weapon != nil {
			if wid, err := strconv.ParseUint(ev.Weapon.ID, 10, 64); err == nil && wid != 0 {
				w := wid
				weapons = append(weapons, persist.WeaponKillInsert{
					MatchID:         matchID,
					XUID:            killerXUID,
					TimeMS:          ev.TimeMs,
					WeaponID:        &w,
					Confidence:      "native",
					AttributionPath: "h5_native",
					// Mecanique du kill deja derivee en amont (h5KillKind). canonical.KillKind
					// EST une string stable (weapon/melee/groundpound/shoulderbash) → cast
					// direct ; vide (Kind absent) => NULL en base via nullableStr cote persist.
					KillKind: string(ev.Kind),
				})
			}
		}
	}
	return pairs, weapons
}
