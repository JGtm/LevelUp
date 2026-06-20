// Package ingest transforme une timeline canonique Halo 5 (events live) en lignes
// persistables dans le warehouse shared (schéma hérité de Halo Infinite). Les
// mappers sont PURS : ils prennent des events canoniques + une fonction de
// résolution xuid injectée, et retournent des rows persist/domain — aucune IO,
// aucun accès DB. Le câblage (résolveur réel, lease, SharedPersister) vit dans la
// couche service/capture-on-fetch.
package ingest

import (
	"strconv"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/persist"
)

// MapMedalEvents projette les events de type `medal` d'une timeline en DEUX formes,
// comme Halo Infinite :
//   - aggregate : medals_earned (count par (xuid, medal_name_id)) → stats médailles.
//   - timeline  : highlight_events (1 ligne horodatée par médaille gagnée).
//
// resolveXUID(gamertag) retourne le xuid résolu ou "" si inconnu (la médaille est
// quand même persistée, xuid vide/NULL — l'identité reste dans le gamertag amont).
// Un RefID de médaille non numérique est ignoré (robustesse : le schéma medal_name_id
// est un BIGINT). L'ordre des MedalRow suit la 1re apparition (déterminisme des tests).
func MapMedalEvents(
	matchID string,
	events []canonical.MatchEvent,
	resolveXUID func(gamertag string) string,
) (aggregate []domain.MedalRow, timeline []persist.HighlightEventInsert) {
	type key struct {
		xuid  string
		medal int64
	}
	counts := make(map[key]int)
	var order []key
	eventType := string(canonical.MatchEventMedal)

	for i := range events {
		ev := events[i]
		if ev.Type != canonical.MatchEventMedal || ev.Player == nil || ev.RefID == nil {
			continue
		}
		medalID, err := strconv.ParseInt(*ev.RefID, 10, 64)
		if err != nil {
			continue // medal id non numérique → ignoré (medals_earned.medal_name_id = BIGINT)
		}
		xuid := resolveXUID(ev.Player.Gamertag)

		k := key{xuid: xuid, medal: medalID}
		if _, seen := counts[k]; !seen {
			order = append(order, k)
		}
		counts[k]++

		var xuidPtr *string
		if xuid != "" {
			x := xuid
			xuidPtr = &x
		}
		detail := *ev.RefID // medal id en string → coercé en type_hint INTEGER côté DDL
		timeline = append(timeline, persist.HighlightEventInsert{
			MatchID:     matchID,
			XUID:        xuidPtr,
			EventType:   eventType,
			TimeMS:      ev.TimeMs,
			DetailsJSON: &detail,
		})
	}

	aggregate = make([]domain.MedalRow, 0, len(order))
	for _, k := range order {
		aggregate = append(aggregate, domain.MedalRow{
			MatchID:     matchID,
			XUID:        k.xuid,
			MedalNameID: k.medal,
			Count:       counts[k],
		})
	}
	return aggregate, timeline
}
