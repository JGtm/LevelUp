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

	// Fail-fast (fix 2026-05-31) : si shared est read-only, NE PAS boucler sur N
	// matchs en tentant N écritures vouées à l'échec (et comptées `failed`).
	// Détecter une fois, remonter une erreur nommée VISIBLE. C'est l'absence de
	// ce garde-fou qui a transformé un incident shared-RO en 31h de panne
	// silencieuse (cf. .ai/HANDOFF_sync_combat_completion.md).
	if rwErr := assertSharedWritable(ctx, sharedDB); rwErr != nil {
		slog.ErrorContext(ctx, "healEvents: shared non-writable — complétion combat SKIPPÉE (fail-fast)",
			"err", rwErr, "pending", len(matchIDs),
			"hint", "corruption ART / FATAL invalidated ? cf .ai/HANDOFF_sync_combat_completion.md")
		return 0, 0, fmt.Errorf("healEvents: %w", rwErr)
	}

	// Parallélisation 2026-05-22 : les film downloads sont des opérations
	// long-tail (CDN + zlib decompress) qui se prêtent bien au parallélisme.
	// DuckDB sérialise les writes côté DB donc pas de risque de corruption,
	// le gain vient du réseau. mu protège uniquement les compteurs.
	// Phase 3.6 : healParallelismNetworkOnly=24 — writes append-only sur
	// highlight_events, throttle réel par rate limiter HTTP du pool.
	var mu sync.Mutex
	// failed : échecs RÉELS d'écriture/fetch (shared en read-only, réseau,
	// parse zlib). Historiquement comptés en no_film → mensonge qui a masqué
	// 31h de panne (shared RO) en mai 2026, cf.
	// .ai/HANDOFF_sync_combat_completion.md. Désormais agrégés en WARN distinct.
	var failed int
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
			case procErr != nil:
				// Échec réel (write shared RO, réseau, parse) — surtout PAS
				// compté en no_film. C'est cette confusion qui a rendu le bug
				// shared-RO invisible pendant 31h (no_film=2 affiché alors que
				// l'INSERT échouait). Déjà loggé par match ci-dessus ; agrégé
				// en WARN après Wait().
				failed++
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
	if failed > 0 {
		// Visibilité ops : un échec d'écriture shared (read-only, lock) doit être
		// bruyant, jamais noyé dans no_film. Cf. règle "complétion = fonction
		// cœur, pas un best-effort silencieux".
		slog.WarnContext(ctx, "healEvents: écritures shared échouées (NON comptées en no_film)",
			"failed", failed, "healed", healed, "no_film", noFilm,
			"hint", "shared en read-only / lock ? cf. .ai/HANDOFF_sync_combat_completion.md")
	}
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

	// Fail-fast (fix 2026-05-31) : voir healEventsForRecentMatches. Shared RO →
	// erreur nommée immédiate plutôt que N échecs d'écriture silencieux.
	if rwErr := assertSharedWritable(ctx, sharedDB); rwErr != nil {
		slog.ErrorContext(ctx, "healWeaponKills: shared non-writable — complétion SKIPPÉE (fail-fast)",
			"err", rwErr, "pending", len(matchIDs),
			"hint", "corruption ART / FATAL invalidated ? cf .ai/HANDOFF_sync_combat_completion.md")
		return 0, 0, fmt.Errorf("healWeaponKills: %w", rwErr)
	}

	// Parallélisation idem healEvents : downloads CDN + parse parallèles, mu
	// protège les compteurs. errgroup ne propage pas l'erreur (best-effort).
	// Phase 3.6 : healParallelismNetworkOnly=24 — writes append-only sur
	// weapon_kills, throttle réel par rate limiter HTTP du pool.
	var mu sync.Mutex
	// failed : échecs réels (write shared RO, réseau). Avant, un bfErr était
	// loggé puis le match disparaissait des compteurs (ni healed ni no_film) —
	// invisible en agrégat. Désormais compté + agrégé en WARN. Même rationale
	// que healEvents (cf. .ai/HANDOFF_sync_combat_completion.md).
	var failed int
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
				mu.Lock()
				failed++
				mu.Unlock()
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
	if failed > 0 {
		slog.WarnContext(ctx, "healWeaponKills: écritures shared échouées (NON comptées en no_film)",
			"failed", failed, "healed", healed, "no_film", noFilm,
			"hint", "shared en read-only / lock ? cf. .ai/HANDOFF_sync_combat_completion.md")
	}
	return healed, noFilm, nil
}
