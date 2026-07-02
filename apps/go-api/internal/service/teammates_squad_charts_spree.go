// Package service - teammates_squad_charts_spree.go : chargement des events
// kill/death par match pour le calcul-fallback du max killing spree escouade
// (analysis.ComputeMaxKillingSpree quand la valeur native est absente). Découpe
// de teammates_squad_charts_weapons_perf.go (dépassement limite 500 L). Le calcul
// native-ou-computed reste inline dans buildSquadPerformanceSeries (weapons_perf.go).
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/games/canonical"
)

// loadCanonicalKillEventsByMatch charge les events kill/death des matchs partagés et
// les regroupe par match_id sous forme canonical.HighlightEvent (XUID = acteur). Le
// repo applique déjà le fallback de synthèse kvPairs → kill/death pour les titres dont
// highlight_events ne porte pas les kills (Halo 5), donc ces events sont disponibles
// pour tout titre qui expose des kills horodatés. Sert au calcul-fallback du max
// killing spree (analysis.ComputeMaxKillingSpree) quand la valeur native est absente.
// PAS de correction T0 : la spree est order-based (invariante par décalage T0).
func (s *TeammatesService) loadCanonicalKillEventsByMatch(
	ctx context.Context,
	matchIDs []string,
) map[string][]canonical.HighlightEvent {
	if s.repo == nil || len(matchIDs) == 0 {
		return nil
	}
	rows, err := s.repo.LoadImpactEvents(ctx, matchIDs)
	if err != nil {
		slog.WarnContext(ctx, "teammates_perf_spree_events_load_failed",
			"err", err, "n_matches", len(matchIDs))
		return nil
	}
	out := make(map[string][]canonical.HighlightEvent, len(matchIDs))
	for _, r := range rows {
		out[r.MatchID] = append(out[r.MatchID], canonical.HighlightEvent{
			MatchID:   r.MatchID,
			EventType: r.EventType,
			TimeMS:    r.TimeMS,
			XUID:      r.XUID,
		})
	}
	return out
}
