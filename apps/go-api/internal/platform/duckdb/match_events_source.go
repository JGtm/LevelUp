// Package duckdb — match_events_source.go : source DuckDB de la timeline
// canonique d'events (halo_infinite.EventsSource).
//
// Fournit à l'adapter canonique HI les 2 entrées de la reconstruction
// MatchEvent (Phase 2 du plan PLAN_CANONICAL_MATCH_EVENTS) :
//   - les highlight_events filmés bruts d'un match (réutilise HighlightEventsRepo) ;
//   - la MatchTimeline (durée + T0 countdown) pour la correction temporelle.
//
// Per-PlayerDB : la lecture passe par SharedReadDB().Get() (ADR 0016) — connexion
// directe à shared_matches_v2.duckdb, tables référencées en bare (PAS `shared.*`).
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// MatchEventsSource implémente halo_infinite.EventsSource (structurellement) :
// elle alimente la reconstruction de la timeline canonique d'un match HI.
type MatchEventsSource struct {
	pdb *PlayerDB
}

// NewMatchEventsSource crée une MatchEventsSource liée à un PlayerDB.
func NewMatchEventsSource(pdb *PlayerDB) *MatchEventsSource {
	return &MatchEventsSource{pdb: pdb}
}

// LoadHighlightEvents charge tous les events filmés d'un match (réutilise le
// loader unifié HighlightEventsRepo, filtre mono-match, ordre temporel).
func (s *MatchEventsSource) LoadHighlightEvents(
	ctx context.Context,
	matchID string,
) ([]canonical.HighlightEvent, error) {
	return NewHighlightEventsRepo(s.pdb).Load(ctx, port.HighlightEventFilters{
		MatchIDs: []string{matchID},
		OrderBy:  "time_ms ASC",
	})
}

// GetMatchTimeline lit la durée + l'offset T0 (countdown pré-match) du match
// depuis match_registry et construit la MatchTimeline (source unique de la
// correction T0). Match absent → MatchTimeline zéro-value (T0=0 → identité,
// pas d'erreur : un match sans ligne registry ne casse pas la timeline d'events).
func (s *MatchEventsSource) GetMatchTimeline(
	ctx context.Context,
	matchID string,
) (domain.MatchTimeline, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	db, release, err := s.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return domain.MatchTimeline{}, fmt.Errorf("MatchEventsSource.GetMatchTimeline: shared reader: %w", err)
	}
	defer release()

	var durationSeconds, t0Ms sql.NullInt64
	err = db.QueryRowContext(ctx, matchTimelineQuery, matchID).Scan(&durationSeconds, &t0Ms)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return domain.MatchTimeline{}, nil
	case err != nil:
		return domain.MatchTimeline{}, fmt.Errorf("MatchEventsSource.GetMatchTimeline: %w", err)
	}
	return domain.NewMatchTimeline(durationSeconds.Int64*1000, t0Ms.Int64), nil
}

// matchTimelineQuery lit durée (BIGINT castée, robuste si la colonne est DOUBLE)
// + T0 (countdown) d'un match. T0 = écart ms entre real_start_time et
// start_time_utc (identique à Q13MatchMeta) ; NULL → 0 (fallback runtime).
// Exécutée sur SharedReader (ADR 0016) — pas de préfixe `shared.`.
var matchTimelineQuery = `
SELECT
    CAST(r.duration_seconds AS BIGINT) AS duration_seconds,
    CASE
        WHEN r.real_start_time IS NOT NULL THEN
            epoch_ms(r.real_start_time AT TIME ZONE 'UTC')
            - epoch_ms(` + StartTimeCanonicalSQL("r") + `)
    END AS t0_ms
FROM match_registry r
WHERE r.match_id = ?`
