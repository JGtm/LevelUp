// Package sync — events_replay.go : détection et replay des matchs où le
// pipeline highlight events a échoué silencieusement.
//
// Deux cas couverts :
//
//  1. `events_loaded=TRUE` mais aucune ligne dans `highlight_events` —
//     cassure primaire historique due à l'ancien parser byte-aligné (avant
//     le fix bit-aligné de mai 2026).
//
//  2. `highlight_events` présents (kills détectés) mais `killer_victim_pairs`
//     vide — cassure secondaire due à `InsertKillerVictimPairsFromEvents`
//     avec `INSERT OR IGNORE` rejeté par DuckDB (table sans PK) avant le fix
//     du même mois.
//
// Le helper extrait une boucle clear+process idempotente partagée par :
//   - `cmd/levelup replay-events` (sous-commande CLI)
//   - `internal/api/handlers/backfill.go` (HTTP `/backfill/start` avec
//     `events:true` et `force_events:true`)
//
// Il consomme `ProcessHighlightEvents` (engine.go), donc bénéficie du parser
// bit-aligné corrigé et de la détection d'anomalie expvar
// `highlight_events_parse_anomaly_total`.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sort"

	"levelup/go-api/internal/domain"
)

// ReplayResult agrège les compteurs d'un cycle de replay highlight events.
type ReplayResult struct {
	Total          int
	Healed         int
	NoFilm         int
	ParseAnomaly   int
	Errors         int
	EventsInserted int
}

// FindBrokenHighlightEventMatches retourne les match_ids où le pipeline
// highlight events a échoué silencieusement. Triés par start_time décroissant
// (les plus récents d'abord, qui ont le plus de chances d'avoir leur film
// encore disponible côté CDN Halo).
//
// Critères (OR) :
//   - `events_loaded=TRUE` mais aucun row dans `highlight_events`
//   - `highlight_events` contient des kills mais `killer_victim_pairs` est vide
//
// Filtré sur les matchs réels (présence d'au moins un participant).
func FindBrokenHighlightEventMatches(ctx context.Context, db *sql.DB, limit int) ([]string, error) {
	if db == nil {
		return nil, fmt.Errorf("FindBrokenHighlightEventMatches: db nil")
	}
	if limit <= 0 {
		limit = 200
	}
	rows, err := db.QueryContext(ctx, `
		SELECT mr.match_id
		FROM match_registry mr
		WHERE COALESCE(mr.events_loaded, FALSE) = TRUE
		  AND EXISTS (SELECT 1 FROM match_participants mp WHERE mp.match_id = mr.match_id)
		  AND (
		    NOT EXISTS (SELECT 1 FROM highlight_events he WHERE he.match_id = mr.match_id)
		    OR (
		      EXISTS (SELECT 1 FROM highlight_events he WHERE he.match_id = mr.match_id AND he.event_type = 'kill')
		      AND NOT EXISTS (SELECT 1 FROM killer_victim_pairs kvp WHERE kvp.match_id = mr.match_id)
		    )
		  )
		ORDER BY COALESCE(mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC') DESC NULLS LAST
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("FindBrokenHighlightEventMatches query: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("FindBrokenHighlightEventMatches scan: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ReplayProgressFn est un callback optionnel appelé pour chaque match traité.
// `done` et `total` permettent au caller d'afficher une progression.
// `status` est l'un de "healed", "no_film", "parse_anomaly", "error" — utile
// pour structurer un log côté caller.
type ReplayProgressFn func(done, total int, matchID, status string)

// ReplayHighlightEventsForMatches rejoue le parsing highlight events pour la
// liste de match_ids fournie. Pour chaque match :
//
//  1. Reset `events_loaded=FALSE` + clear bit `MBitEvents` dans
//     `match_registry.backfill_completed` (pour que `ProcessHighlightEvents`
//     traite le match comme à faire).
//  2. Appelle `ProcessHighlightEvents` (parser bit-aligné + insert events +
//     insert killer_victim_pairs honnête).
//
// Idempotent : peut être ré-exécuté sans dommage. Cancellable via `ctx`.
//
// `progressFn` peut être nil. `globalDB` peut être nil (les xuid_aliases
// globaux ne seront pas mis à jour, mais la shared DB l'est toujours).
func ReplayHighlightEventsForMatches(
	ctx context.Context,
	client HaloClient,
	sharedDB, globalDB *sql.DB,
	matchIDs []string,
	progressFn ReplayProgressFn,
) (ReplayResult, error) {
	if sharedDB == nil {
		return ReplayResult{}, fmt.Errorf("ReplayHighlightEventsForMatches: sharedDB nil")
	}
	if client == nil {
		return ReplayResult{}, fmt.Errorf("ReplayHighlightEventsForMatches: client nil")
	}

	res := ReplayResult{Total: len(matchIDs)}
	for i, matchID := range matchIDs {
		if err := ctx.Err(); err != nil {
			return res, err
		}

		if err := clearEventsLoaded(sharedDB, matchID); err != nil {
			res.Errors++
			slog.WarnContext(ctx, "ReplayHighlightEvents: clearEventsLoaded échoué",
				"match_id", matchID, "err", err)
			if progressFn != nil {
				progressFn(i+1, len(matchIDs), matchID, "error")
			}
			continue
		}

		dummy := &domain.SyncResult{}
		err := ProcessHighlightEvents(ctx, client, sharedDB, globalDB, matchID, dummy)

		var status string
		switch {
		case err != nil:
			res.Errors++
			status = "error"
			slog.WarnContext(ctx, "ReplayHighlightEvents: ProcessHighlightEvents échoué",
				"match_id", matchID, "err", err)
		case dummy.EventsInserted > 0:
			res.Healed++
			res.EventsInserted += dummy.EventsInserted
			status = "healed"
		case len(dummy.Warnings) > 0:
			res.ParseAnomaly++
			status = "parse_anomaly"
		default:
			res.NoFilm++
			status = "no_film"
		}

		if progressFn != nil {
			progressFn(i+1, len(matchIDs), matchID, status)
		}
	}
	return res, nil
}

// UnionMatchIDs retourne l'union ordonnée stable de deux listes de match_ids.
// L'ordre des éléments uniques de `a` est préservé en premier, puis ceux de
// `b` qui n'apparaissent pas dans `a`. Helper pour la Phase 2 (HTTP handler)
// qui combine `missing` (events_loaded=FALSE) et le résultat de
// `FindBrokenHighlightEventMatches` (events_loaded=TRUE mais cassé).
func UnionMatchIDs(a, b []string) []string {
	if len(a) == 0 {
		return append([]string(nil), b...)
	}
	if len(b) == 0 {
		return append([]string(nil), a...)
	}
	seen := make(map[string]struct{}, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, id := range a {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, id := range b {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// SortedMatchIDs retourne une copie triée lexicographiquement des match_ids.
// Utile pour rendre les tests déterministes et pour des outputs CLI lisibles.
func SortedMatchIDs(ids []string) []string {
	out := append([]string(nil), ids...)
	sort.Strings(out)
	return out
}

// clearEventsLoaded remet `events_loaded=FALSE` et clear le bit `MBitEvents`
// dans `match_registry.backfill_completed`, pour que le pipeline de re-parse
// traite le match comme à faire.
func clearEventsLoaded(db *sql.DB, matchID string) error {
	_, err := db.Exec(`
		UPDATE match_registry
		SET events_loaded      = FALSE,
		    backfill_completed = COALESCE(backfill_completed, 0) & ~?
		WHERE match_id = ?`, MBitEvents, matchID)
	if err != nil {
		return fmt.Errorf("clearEventsLoaded(%s): %w", matchID, err)
	}
	return nil
}
