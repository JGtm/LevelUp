// Package sync — friends_recompute.go : recompute is_with_friends pour un joueur.
//
// §4 du plan Squad/Sessions overhaul. Suit le pattern session_recalc.go :
// acquisition de leases sur les deux DBs, ouverture séparée, résolution des
// XUIDs amis depuis xuid_aliases, UPDATE atomique, refresh aggregates.
//
// Sémantique CONVERGENTE (2026-06-19, project_convergent_sync_direction) :
// réconcilie is_with_friends vers l'état cible — TRUE pour les matchs où un ami
// COURANT a joué dans la même équipe, FALSE pour les matchs qui n'en ont plus
// (démotion). Idempotent : un 2e passage sans changement de friends → 0 ligne.
// Retirer un ami (y compris le dernier → liste vide) dé-flague rétroactivement
// les matchs concernés. ART-safe : UPDATE par batch IN(...) (pattern établi ici ;
// player_match_enrichment hors tables append-only protégées).
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
	MatchesPromoted     int64 // lignes player_match_enrichment passées de FALSE à TRUE
	MatchesDemoted      int64 // lignes passées de TRUE à FALSE (ami retiré — convergent)
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
	// Sémantique convergente : on NE court-circuite PLUS sur liste vide — une liste
	// vide signifie « plus aucun ami », donc tous les matchs TRUE doivent être
	// démotés. On acquiert les leases et on laisse Core réconcilier.
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

	// Garde défensive : sans player DB on ne peut rien réconcilier (cas unit-test /
	// config minimale).
	if playerDB == nil {
		return res, nil
	}

	// Calculer l'ensemble CIBLE : matchs où un ami COURANT a joué dans la même
	// équipe. Liste vide → cible vide → tout sera démoté (ami retiré / dernier ami).
	var targetIDs []string
	if len(friendGamertags) > 0 {
		if sharedDB == nil {
			// Impossible de résoudre la cible sans la shared DB : on s'abstient
			// (ne pas démoter à tort).
			return res, nil
		}
		friendXUIDs := LookupFriendXUIDs(ctx, sharedDB, friendGamertags)
		res.FriendXUIDsCount = len(friendXUIDs)
		if len(friendXUIDs) < len(friendGamertags) {
			slog.WarnContext(ctx, "friends recompute: some xuids unresolved",
				"player_xuid", playerXUID,
				"gamertags_provided", len(friendGamertags),
				"xuids_resolved", len(friendXUIDs),
			)
		}
		if len(friendXUIDs) > 0 {
			ids, err := loadMatchesWithFriends(ctx, sharedDB, playerXUID, friendXUIDs)
			if err != nil {
				return res, fmt.Errorf("RecomputeIsWithFriendsCore loadMatches: %w", err)
			}
			targetIDs = ids
		}
	}

	// Promotion : FALSE → TRUE pour la cible (no-op si cible vide).
	promoted, err := updateIsWithFriendsBatch(ctx, playerDB, targetIDs)
	if err != nil {
		return res, fmt.Errorf("RecomputeIsWithFriendsCore promote: %w", err)
	}
	res.MatchesPromoted = promoted

	// Démotion : TRUE → FALSE pour les matchs actuellement TRUE qui ne sont plus
	// dans la cible (ami retiré). Convergent.
	demoted, err := demoteStaleIsWithFriends(ctx, playerDB, targetIDs)
	if err != nil {
		return res, fmt.Errorf("RecomputeIsWithFriendsCore demote: %w", err)
	}
	res.MatchesDemoted = demoted

	if (promoted > 0 || demoted > 0) && refreshAggregates {
		if _, failed, err := RefreshAggregates(ctx, playerDB); err != nil {
			slog.WarnContext(ctx, "friends recompute: refresh aggregates failed",
				"player_xuid", playerXUID, "views_failed", failed, "err", err)
		} else {
			res.AggregatesRefreshed = true
		}
	}

	res.Duration = time.Since(start)
	slog.InfoContext(ctx, "friends recompute done",
		"player_xuid", playerXUID,
		"friend_xuids", res.FriendXUIDsCount,
		"target_matches", len(targetIDs),
		"matches_promoted", promoted,
		"matches_demoted", demoted,
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

// demoteStaleIsWithFriends remet is_with_friends à FALSE pour les matchs
// actuellement TRUE qui ne figurent PLUS dans targetIDs (ami retiré / dernier ami
// supprimé → targetIDs vide → démotion complète). Convergent.
func demoteStaleIsWithFriends(ctx context.Context, playerDB *sql.DB, targetIDs []string) (int64, error) {
	rows, err := playerDB.QueryContext(ctx,
		`SELECT match_id FROM player_match_enrichment WHERE COALESCE(is_with_friends, FALSE) = TRUE`)
	if err != nil {
		return 0, fmt.Errorf("demoteStale load current TRUE: %w", err)
	}
	defer rows.Close()
	var currentTrue []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("demoteStale scan: %w", err)
		}
		currentTrue = append(currentTrue, id)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(currentTrue) == 0 {
		return 0, nil
	}

	target := make(map[string]struct{}, len(targetIDs))
	for _, id := range targetIDs {
		target[id] = struct{}{}
	}
	var demote []string
	for _, id := range currentTrue {
		if _, ok := target[id]; !ok {
			demote = append(demote, id)
		}
	}
	if len(demote) == 0 {
		return 0, nil
	}
	return demoteIsWithFriendsBatch(ctx, playerDB, demote)
}

// demoteIsWithFriendsBatch fait passer is_with_friends de TRUE à FALSE pour les
// matchs fournis, batch par batch. Même pattern IN(...) que la promotion.
func demoteIsWithFriendsBatch(ctx context.Context, playerDB *sql.DB, matchIDs []string) (int64, error) {
	const batchSize = 500
	var total int64
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
			SET    is_with_friends = FALSE,
			       updated_at      = CURRENT_TIMESTAMP
			WHERE  COALESCE(is_with_friends, FALSE) = TRUE
			  AND  match_id IN (%s)
		`, strings.Join(placeholders, ","))
		result, err := playerDB.ExecContext(ctx, q, args...)
		if err != nil {
			return total, fmt.Errorf("demoteIsWithFriendsBatch batch %d-%d: %w", start, end, err)
		}
		n, _ := result.RowsAffected()
		total += n
	}
	return total, nil
}
