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

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
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
		      -- Sonde de présence basculée sur la canonique le 2026-08-03 : c'est elle que
		      -- les lecteurs servent désormais, donc c'est son absence qui rend un match
		      -- « cassé ». Sonder l'ancienne table déclarerait sains des matchs que l'écran
		      -- affiche vides.
		      AND NOT EXISTS (SELECT 1 FROM match_kill_events_latest kvp WHERE kvp.match_id = mr.match_id)
		    )
		  )
		ORDER BY `+analysis.SQLStartTimeCanonical("mr")+` DESC NULLS LAST
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

// Statuts canoniques retournés par ReplayHighlightEventsForMatches via le
// callback ReplayProgressFn. Utilisés aussi par les tests d'intégration pour
// asserter le résultat.
const (
	replayStatusHealed       = "healed"
	replayStatusNoFilm       = "no_film"
	replayStatusParseAnomaly = "parse_anomaly"
	replayStatusError        = "error"
)

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
// `progressFn` peut être nil.
func ReplayHighlightEventsForMatches(
	ctx context.Context,
	client HaloClient,
	sharedDB *sql.DB,
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

		if err := clearEventsLoaded(ctx, sharedDB, matchID); err != nil {
			res.Errors++
			slog.WarnContext(ctx, "ReplayHighlightEvents: clearEventsLoaded échoué",
				"match_id", matchID, "err", err)
			if progressFn != nil {
				progressFn(i+1, len(matchIDs), matchID, "error")
			}
			continue
		}

		dummy := &domain.SyncResult{}
		err := ProcessHighlightEvents(ctx, client, sharedDB, matchID, dummy)

		var status string
		switch {
		case err != nil:
			res.Errors++
			status = replayStatusError
			slog.WarnContext(ctx, "ReplayHighlightEvents: ProcessHighlightEvents échoué",
				"match_id", matchID, "err", err)
		case dummy.EventsInserted > 0:
			res.Healed++
			res.EventsInserted += dummy.EventsInserted
			status = replayStatusHealed
		case len(dummy.Warnings) > 0:
			res.ParseAnomaly++
			status = replayStatusParseAnomaly
		default:
			res.NoFilm++
			status = replayStatusNoFilm
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
func clearEventsLoaded(ctx context.Context, db *sql.DB, matchID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE match_registry
		SET events_loaded      = FALSE,
		    backfill_completed = COALESCE(backfill_completed, 0) & ~?
		WHERE match_id = ?`, MBitEvents, matchID)
	if err != nil {
		return fmt.Errorf("clearEventsLoaded(%s): %w", matchID, err)
	}
	return nil
}

// BackfillEventsForMatches est la méthode `*SyncEngine` qui orchestre le
// replay highlight events sur une liste de matchs. Acquiert le lease shared,
// ouvre la DB, construit un HaloAPIClient depuis les tokens du moteur, puis
// délègue à `ReplayHighlightEventsForMatches`.
//
// Si `includeBroken` est true, les matchs détectés par
// `FindBrokenHighlightEventMatches` (events_loaded=TRUE mais cassés) sont
// ajoutés à la liste — utile quand l'appelant active `--force-events`.
//
// Pendant à `BackfillWeaponKillsForMatches` ; même contrat de lease.
func (e *SyncEngine) BackfillEventsForMatches(
	ctx context.Context,
	matchIDs []string,
	includeBroken bool,
	progressFn ReplayProgressFn,
) (ReplayResult, error) {
	if e.tokens == nil || e.tokens.SpartanToken == "" {
		return ReplayResult{}, fmt.Errorf("BackfillEventsForMatches: tokens Halo absents")
	}

	// Sprint B1 commit 13a : acquireSharedWriter centralise lease + open
	// (Provider en B-swap coordonne avec les readers HTTP).
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctxkeys.WithDBWriterLabel(ctx, "backfill_events_replay"))
	if err != nil {
		return ReplayResult{}, fmt.Errorf("BackfillEventsForMatches: %w", err)
	}
	defer releaseShared()

	ids := matchIDs
	if includeBroken {
		broken, findErr := FindBrokenHighlightEventMatches(ctx, sharedDB, 5000)
		if findErr != nil {
			slog.WarnContext(ctx, "BackfillEventsForMatches: FindBrokenHighlightEventMatches échoué",
				"err", findErr)
		} else {
			ids = UnionMatchIDs(ids, broken)
		}
	}
	if len(ids) == 0 {
		return ReplayResult{}, nil
	}

	client := NewHaloAPIClient(e.tokens.SpartanToken, e.tokens.ClearanceToken, 3)

	return ReplayHighlightEventsForMatches(ctx, client, sharedDB, ids, progressFn)
}
