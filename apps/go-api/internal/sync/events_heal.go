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

	"levelup/go-api/internal/domain"
)

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

	for _, matchID := range matchIDs {
		if ctx.Err() != nil {
			return healed, noFilm, ctx.Err()
		}
		dummy := &domain.SyncResult{}
		eventsBefore := dummy.EventsInserted
		err := ProcessHighlightEvents(ctx, client, sharedDB, globalDB, matchID, dummy)
		if err != nil {
			slog.WarnContext(ctx, "healEvents: échec match", "match_id", matchID, "err", err)
		}
		// Marquer events_loaded=TRUE même en erreur ou en no-film : sinon le
		// match revient à chaque sync. processHighlightEvents le fait déjà sur
		// no-film mais pas sur erreur réseau ; le faire ici garantit la
		// convergence (au pire on retentera lors d'un --force-events explicite).
		if markErr := MarkEventsLoaded(sharedDB, matchID); markErr != nil {
			slog.DebugContext(ctx, "healEvents: MarkEventsLoaded échoué",
				"match_id", matchID, "err", markErr)
		}
		if dummy.EventsInserted > eventsBefore {
			healed++
		} else {
			noFilm++
		}
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

	for _, matchID := range matchIDs {
		if ctx.Err() != nil {
			return healed, noFilm, ctx.Err()
		}
		found, err := BackfillWeaponKillsForMatch(ctx, client, sharedDB, matchID, xuid)
		if err != nil {
			slog.WarnContext(ctx, "healWeaponKills: échec match", "match_id", matchID, "err", err)
			continue
		}
		if found {
			healed++
		} else {
			noFilm++
		}
	}
	return healed, noFilm, nil
}
