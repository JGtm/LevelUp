package ingest

import (
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/persist"
)

// MapAssistEvents projette les ASSISTS de la timeline Halo 5 (Death.Assistants[])
// en lignes highlight_events (1 par assistant, event_type=assist, acteur=assistant).
// Alimente le signal « support » de la courbe d'engagement (parité Halo Infinite,
// dont la branche assist existe mais reste vide faute de timeline). Le compteur de
// rythme partitionne par xuid → l'assist crédite l'activité de l'ASSISTANT (acteur
// distinct du tueur → aucun double-comptage avec les kills synthétisés).
//
// resolveXUID(gamertag) → xuid résolu ou "" (assist quand même persistée, xuid NULL,
// identité dans le gamertag amont). Assistants sans gamertag ignorés.
func MapAssistEvents(
	matchID string,
	events []canonical.MatchEvent,
	resolveXUID func(gamertag string) string,
) []persist.HighlightEventInsert {
	eventType := string(canonical.MatchEventAssist)
	var out []persist.HighlightEventInsert
	for i := range events {
		ev := events[i]
		if ev.Type != canonical.MatchEventKill || len(ev.Assists) == 0 {
			continue
		}
		for _, a := range ev.Assists {
			if a.Gamertag == "" {
				continue
			}
			var xuidPtr *string
			if xuid := resolveXUID(a.Gamertag); xuid != "" {
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
	}
	return out
}
