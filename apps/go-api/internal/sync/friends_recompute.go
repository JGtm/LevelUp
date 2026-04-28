// Package sync — friends_recompute.go : recompute is_with_friends pour un joueur.
//
// §4 du plan Squad/Sessions overhaul. Suit le pattern session_recalc.go :
// acquisition de leases sur les deux DBs, ouverture séparée, résolution des
// XUIDs amis depuis xuid_aliases, UPDATE atomique, refresh aggregates.
//
// Sémantique additive : SET is_with_friends = TRUE WHERE FALSE AND match_id IN (matchs avec amis).
// Idempotent (la garde FALSE rend le retry safe). Ne démote PAS les anciennes
// sessions squad si un ami est retiré — c'est intentionnel (un match resté
// historiquement squad le reste).
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// FriendsRecomputeResult agrège les résultats du recompute pour une player DB.
type FriendsRecomputeResult struct {
	XUID                string
	FriendXUIDsCount    int
	MatchesPromoted     int64 // nombre de lignes player_match_enrichment passées de FALSE à TRUE
	AggregatesRefreshed bool
	Duration            time.Duration
}

// RecomputeIsWithFriends recompute is_with_friends pour un joueur en flaggant
// à TRUE toutes les lignes player_match_enrichment dont match_id correspond à
// un match où au moins un xuid d'ami (résolu via xuid_aliases) a participé
// dans la même équipe que le joueur.
//
// friendGamertags : liste des amis (settings.friend_gamertags). Vide → no-op.
// Les gamertags non résolus dans xuid_aliases sont logués Warn et ignorés.
func RecomputeIsWithFriends(
	ctx context.Context,
	playerDBPath, sharedDBPath, playerXUID string,
	friendGamertags []string,
) (FriendsRecomputeResult, error) {
	start := time.Now()
	res := FriendsRecomputeResult{XUID: playerXUID}

	if len(friendGamertags) == 0 {
		slog.InfoContext(ctx, "friends recompute skipped (no friends)", "player_xuid", playerXUID)
		return res, nil
	}

	relPlayer, err := AcquireLeaseCtx(ctx, playerDBPath)
	if err != nil {
		return res, fmt.Errorf("RecomputeIsWithFriends lease player: %w", err)
	}
	defer relPlayer()

	relShared, err := AcquireLeaseCtx(ctx, sharedDBPath)
	if err != nil {
		return res, fmt.Errorf("RecomputeIsWithFriends lease shared: %w", err)
	}
	defer relShared()

	playerHandle, err := OpenPlayerDB(playerDBPath)
	if err != nil {
		return res, fmt.Errorf("RecomputeIsWithFriends OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	sharedHandle, err := OpenSharedDB(sharedDBPath)
	if err != nil {
		return res, fmt.Errorf("RecomputeIsWithFriends OpenSharedDB: %w", err)
	}
	defer sharedHandle.Close()

	// Résoudre les gamertags amis → XUIDs via xuid_aliases.
	friendXUIDs := LookupFriendXUIDs(sharedHandle.SQLDb(), friendGamertags)
	res.FriendXUIDsCount = len(friendXUIDs)
	if len(friendXUIDs) == 0 {
		slog.WarnContext(ctx, "friends recompute: no xuid resolved",
			"player_xuid", playerXUID, "gamertags_provided", len(friendGamertags))
		return res, nil
	}
	if len(friendXUIDs) < len(friendGamertags) {
		slog.WarnContext(ctx, "friends recompute: some xuids unresolved",
			"player_xuid", playerXUID,
			"gamertags_provided", len(friendGamertags),
			"xuids_resolved", len(friendXUIDs),
		)
	}

	// Charger les match_ids où au moins un ami a participé dans la même équipe que le joueur.
	matchIDs, err := loadMatchesWithFriends(ctx, sharedHandle.SQLDb(), playerXUID, friendXUIDs)
	if err != nil {
		return res, fmt.Errorf("RecomputeIsWithFriends loadMatches: %w", err)
	}
	if len(matchIDs) == 0 {
		slog.InfoContext(ctx, "friends recompute: no matches with friends",
			"player_xuid", playerXUID, "friend_xuids", len(friendXUIDs))
		res.Duration = time.Since(start)
		return res, nil
	}

	// UPDATE batché : SET is_with_friends = TRUE WHERE FALSE AND match_id IN (...).
	rowsAffected, err := updateIsWithFriendsBatch(ctx, playerHandle.SQLDb(), matchIDs)
	if err != nil {
		return res, fmt.Errorf("RecomputeIsWithFriends update: %w", err)
	}
	res.MatchesPromoted = rowsAffected

	if rowsAffected > 0 {
		if _, err := RefreshAggregates(playerHandle.SQLDb()); err != nil {
			slog.WarnContext(ctx, "friends recompute: refresh aggregates failed",
				"player_xuid", playerXUID, "err", err)
		} else {
			res.AggregatesRefreshed = true
		}
	}

	res.Duration = time.Since(start)
	slog.InfoContext(ctx, "friends recompute done",
		"player_xuid", playerXUID,
		"friend_xuids", len(friendXUIDs),
		"matches_in_shared", len(matchIDs),
		"matches_promoted", rowsAffected,
		"aggregates_refreshed", res.AggregatesRefreshed,
		"duration_ms", res.Duration.Milliseconds(),
	)
	return res, nil
}

// loadMatchesWithFriends récupère les match_ids où le joueur ET au moins un
// ami ont joué dans la même équipe. Query exécutée sur la shared DB.
func loadMatchesWithFriends(
	ctx context.Context,
	sharedDB *sql.DB,
	playerXUID string,
	friendXUIDs []string,
) ([]string, error) {
	if len(friendXUIDs) == 0 {
		return nil, nil
	}
	// Ordre des args : IN(...) d'abord (xuid amis), puis WHERE p1.xuid = ?.
	placeholders := make([]string, len(friendXUIDs))
	args := make([]any, 0, len(friendXUIDs)+1)
	for i, x := range friendXUIDs {
		placeholders[i] = "?"
		args = append(args, x)
	}
	args = append(args, playerXUID)
	q := fmt.Sprintf(`
		SELECT DISTINCT p1.match_id
		FROM match_participants p1
		JOIN match_participants p2
		    ON p2.match_id = p1.match_id
		    AND p2.team_id = p1.team_id
		    AND p2.xuid IN (%s)
		WHERE p1.xuid = ?
	`, strings.Join(placeholders, ","))

	rows, err := sharedDB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("loadMatchesWithFriends query: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("loadMatchesWithFriends scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// updateIsWithFriendsBatch fait passer is_with_friends de FALSE à TRUE pour
// les matchs fournis, batch par batch (pour rester sous la limite des
// placeholders DuckDB en cas de très gros volume).
func updateIsWithFriendsBatch(ctx context.Context, playerDB *sql.DB, matchIDs []string) (int64, error) {
	const batchSize = 500
	var totalAffected int64
	for start := 0; start < len(matchIDs); start += batchSize {
		end := start + batchSize
		if end > len(matchIDs) {
			end = len(matchIDs)
		}
		batch := matchIDs[start:end]
		placeholders := make([]string, len(batch))
		args := make([]any, len(batch))
		for i, id := range batch {
			placeholders[i] = "?"
			args[i] = id
		}
		q := fmt.Sprintf(`
			UPDATE player_match_enrichment
			SET    is_with_friends = TRUE,
			       updated_at      = CURRENT_TIMESTAMP
			WHERE  is_with_friends = FALSE
			  AND  match_id IN (%s)
		`, strings.Join(placeholders, ","))
		result, err := playerDB.ExecContext(ctx, q, args...)
		if err != nil {
			return totalAffected, fmt.Errorf("updateIsWithFriendsBatch batch %d-%d: %w", start, end, err)
		}
		n, _ := result.RowsAffected()
		totalAffected += n
	}
	return totalAffected, nil
}
