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

	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
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
//
// provider : si non-nil, passe par Provider.AcquireWriter pour coordonner avec
// le pool joueur et les readers HTTP. Si nil, fallback legacy (dblease +
// OpenSharedDB direct, risque "different configuration"). Sprint B1 commit 13b.
//
// Cette variante acquiert ses propres leases + ouvre les DBs. Pour appel
// inline depuis le sync engine (qui détient déjà les leases), utiliser
// RecomputeIsWithFriendsCore.
func RecomputeIsWithFriends(
	ctx context.Context,
	provider sharedprovider.Provider,
	playerDBPath, sharedDBPath, playerXUID string,
	friendGamertags []string,
) (FriendsRecomputeResult, error) {
	if len(friendGamertags) == 0 {
		slog.InfoContext(ctx, "friends recompute skipped (no friends)", "player_xuid", playerXUID)
		return FriendsRecomputeResult{XUID: playerXUID}, nil
	}

	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, playerDBPath, dblease.KindPlayer)
	if err != nil {
		return FriendsRecomputeResult{XUID: playerXUID}, fmt.Errorf("RecomputeIsWithFriends lease player: %w", err)
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(playerDBPath)
	if err != nil {
		return FriendsRecomputeResult{XUID: playerXUID}, fmt.Errorf("RecomputeIsWithFriends OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	// Sprint B1 commit 13b : helper standalone (Provider en B-swap, dblease +
	// OpenSharedDB en legacy).
	sharedDB, releaseShared, err := AcquireSharedWriterStandalone(ctx, provider, sharedDBPath)
	if err != nil {
		return FriendsRecomputeResult{XUID: playerXUID}, fmt.Errorf("RecomputeIsWithFriends: %w", err)
	}
	defer releaseShared()

	return RecomputeIsWithFriendsCore(ctx, playerHandle.SQLDb(), sharedDB, playerXUID, friendGamertags, true)
}

// RecomputeIsWithFriendsCore exécute la logique recompute sur des handles
// déjà ouverts (leases déjà acquises par l'appelant). Utilisé par l'engine
// post-sync pour éviter le double-acquire de leases (deadlock).
//
// Si refreshAggregates=false, le caller est responsable de refresh les vues
// après (typiquement quand l'engine refresh aggregates dans son propre step
// post-sync, on évite le double refresh).
func RecomputeIsWithFriendsCore(
	ctx context.Context,
	playerDB, sharedDB *sql.DB,
	playerXUID string,
	friendGamertags []string,
	refreshAggregates bool,
) (FriendsRecomputeResult, error) {
	start := time.Now()
	res := FriendsRecomputeResult{XUID: playerXUID}

	if len(friendGamertags) == 0 {
		return res, nil
	}

	// Résoudre les gamertags amis → XUIDs via xuid_aliases.
	friendXUIDs := LookupFriendXUIDs(sharedDB, friendGamertags)
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
	matchIDs, err := loadMatchesWithFriends(ctx, sharedDB, playerXUID, friendXUIDs)
	if err != nil {
		return res, fmt.Errorf("RecomputeIsWithFriendsCore loadMatches: %w", err)
	}
	if len(matchIDs) == 0 {
		res.Duration = time.Since(start)
		return res, nil
	}

	// UPDATE batché : SET is_with_friends = TRUE WHERE FALSE AND match_id IN (...).
	rowsAffected, err := updateIsWithFriendsBatch(ctx, playerDB, matchIDs)
	if err != nil {
		return res, fmt.Errorf("RecomputeIsWithFriendsCore update: %w", err)
	}
	res.MatchesPromoted = rowsAffected

	if rowsAffected > 0 && refreshAggregates {
		if _, err := RefreshAggregates(playerDB); err != nil {
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
	// p2.xuid != p1.xuid : exclut la self-join quand le joueur est aussi dans
	// friendXUIDs (setup multi-joueurs où tous les membres du groupe sont trackés).
	// Sans ce guard, chaque match du joueur satisferait le JOIN via sa propre ligne.
	q := fmt.Sprintf(`
		SELECT DISTINCT p1.match_id
		FROM match_participants p1
		JOIN match_participants p2
		    ON p2.match_id = p1.match_id
		    AND p2.team_id = p1.team_id
		    AND p2.xuid IN (%s)
		    AND p2.xuid != p1.xuid
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
		// COALESCE pour couvrir les lignes héritées où is_with_friends est NULL :
		// sans DEFAULT au schéma initial, les inserts du sync écrivaient NULL et
		// `WHERE is_with_friends = FALSE` ne matchait pas (logique 3-valeurs SQL),
		// donc le badge "Solo" persistait après ajout d'un ami. Cf. thought_log 2026-05-08.
		q := fmt.Sprintf(`
			UPDATE player_match_enrichment
			SET    is_with_friends = TRUE,
			       updated_at      = CURRENT_TIMESTAMP
			WHERE  COALESCE(is_with_friends, FALSE) = FALSE
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
