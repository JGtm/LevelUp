// Package sync — events_heal.go : self-healing pour les matchs où
// `highlight_events` / `killer_victim_pairs` / `weapon_kills` sont absents.
//
// Cause typique : matchs synchronisés avant que `processHighlightEvents`
// (resp. weapon kills pipeline) ne soit câblé dans le sync. Le heal se
// limite aux N matchs récents pour éviter de spammer l'API film au premier
// déploiement (les vieux matchs ont rarement de film disponible — 404).
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/domain"
)

// healParallelism : nombre max de goroutines parallèles dans les heal loops
// qui WRITENT sur match_participants (skill_heal, stats_heal). Conservé à 8
// car ces UPSERTs sont protégés par singleflight (phase 2.3, commit aef47968)
// mais restent coûteux en CGO côté DuckDB. Empiriquement, 8 = pool_size×2
// saturent les 3 tokens sans pression mémoire ni race ART.
const healParallelism = 8

// healParallelismNetworkOnly : variante pour les heal loops dont les writes
// touchent UNIQUEMENT des tables append-only sans PK conflictuelle
// (highlight_events, weapon_kills) — pas de risque race ART, le seul throttle
// pertinent est le rate limiter HTTP du HaloAPIClient (~15 RPS effectif sur
// 3 tokens). Bump à 24 → plus de goroutines en attente du token pool, qui se
// résorbe naturellement quand le rate limiter libère un slot. Audit Agent 3
// estime gain skill_heal 8s → 4s, similaire sur events/weapon_kills.
//
// Plan stabilisation 2026-05-22 §3.6.
const healParallelismNetworkOnly = 24

// healEventsForRecentMatches détecte les matchs avec `events_loaded=FALSE`
// (registry bit absent) et tente de fetcher highlight_events + killer_victim
// via le pipeline existant `processHighlightEvents`.
//
// Limité aux N matchs les plus récents (limit) pour éviter le spam API.
// Best-effort : un match sans film 404/410 est marqué silencieusement, le
// sync continue.
func healEventsForRecentMatches(
	ctx context.Context,
	sharedDB, globalDB *sql.DB,
	client HaloClient,
	limit int,
) (healed, noFilm int, err error) {
	if limit <= 0 {
		limit = 30
	}
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT match_id FROM match_registry
		WHERE COALESCE(events_loaded, FALSE) = FALSE
		ORDER BY start_time DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return 0, 0, fmt.Errorf("healEvents query: %w", err)
	}
	var matchIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, 0, err
		}
		matchIDs = append(matchIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}
	if len(matchIDs) == 0 {
		return 0, 0, nil
	}

	// Parallélisation 2026-05-22 : les film downloads sont des opérations
	// long-tail (CDN + zlib decompress) qui se prêtent bien au parallélisme.
	// DuckDB sérialise les writes côté DB donc pas de risque de corruption,
	// le gain vient du réseau. mu protège uniquement les compteurs.
	// Phase 3.6 : healParallelismNetworkOnly=24 — writes append-only sur
	// highlight_events, throttle réel par rate limiter HTTP du pool.
	var mu sync.Mutex
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(healParallelismNetworkOnly)
	for _, matchID := range matchIDs {
		matchID := matchID
		eg.Go(func() error {
			if egCtx.Err() != nil {
				return egCtx.Err()
			}
			dummy := &domain.SyncResult{}
			eventsBefore := dummy.EventsInserted
			procErr := ProcessHighlightEvents(egCtx, client, sharedDB, globalDB, matchID, dummy)
			if procErr != nil {
				slog.WarnContext(egCtx, "healEvents: échec match", "match_id", matchID, "err", procErr)
			}

			// Fix Phase 1bis (mai 2026) : ne marquer events_loaded=TRUE que sur
			// un résultat HONNÊTE — soit le parse a inséré des events, soit le
			// film est définitivement absent (no_film, ProcessHighlightEvents
			// l'a déjà marqué). En cas d'erreur réseau OU de parse_anomaly
			// (chunk présent mais 0 event extrait, signe d'un format API qui
			// a évolué), on laisse events_loaded=FALSE pour retry au prochain
			// sync.
			eventsInserted := dummy.EventsInserted > eventsBefore
			hasAnomaly := len(dummy.Warnings) > 0
			if procErr == nil && !hasAnomaly {
				if markErr := MarkEventsLoaded(egCtx, sharedDB, matchID); markErr != nil {
					slog.DebugContext(egCtx, "healEvents: MarkEventsLoaded échoué",
						"match_id", matchID, "err", markErr)
				}
			}

			mu.Lock()
			switch {
			case eventsInserted:
				healed++
			case hasAnomaly:
				// parse_anomaly déjà loggé en WARN par ProcessHighlightEvents.
			default:
				noFilm++
			}
			mu.Unlock()
			return nil
		})
	}
	_ = eg.Wait()
	return healed, noFilm, nil
}

// healWeaponKillsForRecentMatches détecte les matchs où `weapon_kills` est vide
// pour ce joueur ET le bit MBitWeaponKills n'est pas set dans match_registry,
// et lance le pipeline film pour les peupler.
//
// Limité aux N matchs récents (films absents pour vieux matchs).
func healWeaponKillsForRecentMatches(
	ctx context.Context,
	sharedDB *sql.DB,
	client HaloClient,
	xuid string,
	limit int,
) (healed, noFilm int, err error) {
	if limit <= 0 {
		limit = 30
	}
	// Sélectionne les matchs où ce joueur a participé mais sans weapon_kills,
	// ET où le bit MBitWeaponKills n'est pas set (== pas encore traité).
	rows, err := sharedDB.QueryContext(ctx, fmt.Sprintf(`
		SELECT mr.match_id
		FROM match_registry mr
		JOIN match_participants mp ON mp.match_id = mr.match_id
		WHERE mp.xuid = ?
		  AND (COALESCE(mr.backfill_completed, 0) & %d) = 0
		  AND NOT EXISTS (
		    SELECT 1 FROM weapon_kills wk
		    WHERE wk.match_id = mr.match_id AND wk.xuid = mp.xuid
		  )
		ORDER BY mr.start_time DESC
		LIMIT ?
	`, MBitWeaponKills), xuid, limit)
	if err != nil {
		return 0, 0, fmt.Errorf("healWeaponKills query: %w", err)
	}
	var matchIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, 0, err
		}
		matchIDs = append(matchIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}
	if len(matchIDs) == 0 {
		return 0, 0, nil
	}

	// Parallélisation idem healEvents : downloads CDN + parse parallèles, mu
	// protège les compteurs. errgroup ne propage pas l'erreur (best-effort).
	// Phase 3.6 : healParallelismNetworkOnly=24 — writes append-only sur
	// weapon_kills, throttle réel par rate limiter HTTP du pool.
	var mu sync.Mutex
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(healParallelismNetworkOnly)
	for _, matchID := range matchIDs {
		matchID := matchID
		eg.Go(func() error {
			if egCtx.Err() != nil {
				return egCtx.Err()
			}
			found, bfErr := BackfillWeaponKillsForMatch(egCtx, client, sharedDB, matchID, xuid)
			if bfErr != nil {
				slog.WarnContext(egCtx, "healWeaponKills: échec match", "match_id", matchID, "err", bfErr)
				return nil
			}
			mu.Lock()
			if found {
				healed++
			} else {
				noFilm++
			}
			mu.Unlock()
			return nil
		})
	}
	_ = eg.Wait()
	return healed, noFilm, nil
}
