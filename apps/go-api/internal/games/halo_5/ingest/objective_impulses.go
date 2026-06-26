package ingest

import (
	"strconv"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/persist"
)

// objectiveImpulseIDs : impulses Halo 5 « objectif / leadership » comptés dans la
// courbe d'engagement (event_type=mode, comme l'objectif Infinite). Allowlist CURÉE
// suivant la taxonomie décidée : on INCLUT les contributions d'objectif POSITIVES,
// distinctes des kills. Sont EXCLUS (donc absents d'ici) :
//   - impulses KILL-dérivés (Kills / Enemy Player Kill / Headshot / Perfect Kill /
//     *Kill / *Takedown) → double-compteraient les kills synthétisés dans la courbe ;
//   - structurel / passif (Spawn / Death / Revived / Suicides) ;
//   - ticks de score (PlayerScoreImpulse) et bonus PvE/Warzone par round.
//
// IDs = catalogue officiel /metadata/h5/metadata/impulses (cf. .ai/H5_EXPLORATION/
// 02_impulses_catalogue.json). À ajuster si la narration produit l'exige.
var objectiveImpulseIDs = map[int64]bool{
	2944278681: true, // FlagCapturedImpulse
	1039658009: true, // Flag Pulls
	4191318012: true, // Flag Pickup
	1063951891: true, // impulse_flag_returned
	952111048:  true, // Point Victories (KOTH)
	2596382552: true, // Defender Wins
	991142198:  true, // Warzone Base Captured
	2344191790: true, // Warzone Core Destruction Victories
	2483589021: true, // Ball Held Duration (Oddball)
	3060032974: true, // Round Won
	2556889090: true, // Player Protected (support objectif)
}

// MapObjectiveImpulseEvents projette les events Impulse OBJECTIF (allowlist) en lignes
// highlight_events (event_type=mode, acteur=joueur). Alimente la dimension objectif de
// la courbe d'engagement (un porteur d'objectif à faibles frags ressort « meneur »).
// Les impulses hors allowlist (kills, structurel, score, PvE) sont ignorés.
// resolveXUID → "" toléré (xuid NULL, identité dans le gamertag amont).
func MapObjectiveImpulseEvents(
	matchID string,
	events []canonical.MatchEvent,
	resolveXUID func(gamertag string) string,
) []persist.HighlightEventInsert {
	const eventType = "mode" // convention highlight_events objectif (parité Infinite)
	var out []persist.HighlightEventInsert
	for i := range events {
		ev := events[i]
		if ev.Type != canonical.MatchEventImpulse || ev.Player == nil || ev.RefID == nil {
			continue
		}
		id, err := strconv.ParseInt(*ev.RefID, 10, 64)
		if err != nil || !objectiveImpulseIDs[id] {
			continue
		}
		var xuidPtr *string
		if xuid := resolveXUID(ev.Player.Gamertag); xuid != "" {
			x := xuid
			xuidPtr = &x
		}
		out = append(out, persist.HighlightEventInsert{
			MatchID:   matchID,
			XUID:      xuidPtr,
			EventType: eventType,
			TimeMS:    ev.TimeMs,
		})
	}
	return out
}
