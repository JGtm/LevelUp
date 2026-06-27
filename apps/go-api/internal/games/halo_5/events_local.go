// Package halo_5 — events_local.go : source LOCALE (DuckDB synchronisé) du kill-feed
// d'un match, pour servir la timeline hors-ligne (démo, aucun token → l'API cryptum
// /events échouerait). Le kill-feed natif h5 est persisté (killer_victim_pairs +
// weapon_kills + kill_positions) → reconstructible sans live.
package halo_5

import (
	"context"

	"levelup/go-api/internal/games/canonical"
)

// MatchEventsLocalSource lit le kill-feed d'un match depuis le substrat LOCAL et le
// projette en canonical.MatchEventTimeline. Implémentée STRUCTURELLEMENT par
// platform/duckdb.Halo5MatchEventsSource (retour canonical, aucun import croisé → pas
// de cycle ; même pattern que MatchHistorySource). Injectée via WithMatchEventsSource.
type MatchEventsLocalSource interface {
	GetMatchEvents(ctx context.Context, matchID string, opts canonical.MatchEventOptions) (*canonical.MatchEventTimeline, error)
}
