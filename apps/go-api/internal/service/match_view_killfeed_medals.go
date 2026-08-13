// Package service — match_view_killfeed_medals.go : L'IDENTITÉ DES MÉDAILLES DU FEED.
//
// CE QUE CE FICHIER FAIT. Les events `medal` de highlight_events arrivent du repo avec
// leur nom ANGLAIS (raw_json.medal_name — la quantité mesurée, lue du film). Ce fichier
// les résout contre le référentiel du jeu (medal_definitions, metadata.duckdb) : label
// et description dans la locale de requête, medal_name_id, et le visuel via l'adapter
// d'assets du titre. Même modèle que l'arme du kill : une décoration APRÈS l'assemblage,
// jamais une entrée du calcul.
//
// LE REPLI EST LE NOM BRUT, jamais un visuel voisin : une médaille absente du
// référentiel garde son nom anglais en toutes lettres (le front l'écrit tel quel).
// Mesuré sur 000d5950 : 21 noms distincts, 21 résolus, 44 occurrences couvertes.
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// decorateMedalEvents pose l'identité résolue sur les events `medal` du feed.
//
// Modifie la tranche EN PLACE. Best-effort de bout en bout : repo nil, référentiel
// indisponible ou nom inconnu → l'event garde son seul nom brut. L'échec du lookup est
// loggé (jamais avalé), puis la réponse part quand même — le feed dégrade, il ne casse
// pas.
func decorateMedalEvents(
	ctx context.Context,
	events []domain.MatchHighlightEvent,
	repo port.MatchViewRepository,
	assetURL games.TitleAssetURLAdapter,
) {
	names := make([]string, 0, 8)
	for i := range events {
		if events[i].EventType == analysis.EventTypeMedal && events[i].MedalName != "" {
			names = append(names, events[i].MedalName)
		}
	}
	if len(names) == 0 || repo == nil {
		return
	}
	metas, err := repo.LookupMedalMetaByName(ctx, names)
	if err != nil {
		slog.WarnContext(ctx, "match_view: résolution des médailles du feed en échec",
			"medal_events", len(names), "err", err)
		return
	}
	resolved := 0
	for i := range events {
		e := &events[i]
		if e.EventType != analysis.EventTypeMedal || e.MedalName == "" {
			continue
		}
		m, ok := metas[e.MedalName]
		if !ok {
			continue
		}
		id := m.MedalNameID
		e.MedalNameID = &id
		e.MedalLabel = m.Label
		e.MedalDescription = m.Description
		if assetURL != nil {
			e.MedalImageURL = assetURL.MedalImageURL(uint64(id))
		}
		resolved++
	}
	slog.DebugContext(ctx, "match_view: couverture identité médailles du feed",
		"medal_events", len(names), "resolues", resolved)
}
