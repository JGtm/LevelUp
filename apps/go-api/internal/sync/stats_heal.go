// Package sync — stats_heal.go : self-healing pour les matchs où des colonnes
// `match_participants` ou `match_registry` sont absentes (sync joué avec un
// ancien binaire qui ne les extrayait pas : max_killing_spree, grenade_kills,
// melee_kills, power_weapon_kills, time_played_seconds, avg_life_seconds,
// gamertag, team_X_ps_score).
//
// Stratégie : détecter les matchs avec un proxy NULL (max_killing_spree),
// re-fetcher GetMatchStats, re-extraire participants + registry, UPSERT.
// L'UPSERT préserve les valeurs non-NULL (COALESCE) — pas de risque de
// régression sur les colonnes déjà correctes.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"

	"golang.org/x/sync/errgroup"
)

// healStatsForRecentMatches détecte les matchs récents où le joueur a
// max_killing_spree NULL (proxy "ancien binaire") ET re-fetch pour combler
// les nouvelles colonnes (max_spree, grenade/melee/power, time_played,
// avg_life, gamertag) + recalcule team_X_ps_score.
//
// Coût : 1 GetMatchStats par match (pas de film). Marque le match comme traité
// via INSERT-or-UPDATE — la prochaine sync ne re-touchera pas ces matchs si
// max_killing_spree est désormais non-NULL.
func healStatsForRecentMatches(
	ctx context.Context,
	sharedDB *sql.DB,
	client HaloClient,
	xuid, selfGamertag string,
	limit int,
) (healed int, err error) {
	if limit <= 0 {
		limit = 30
	}
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT mr.match_id
		FROM match_registry mr
		JOIN match_participants mp ON mp.match_id = mr.match_id
		WHERE mp.xuid = ?
		  AND mp.max_killing_spree IS NULL
		ORDER BY mr.start_time DESC
		LIMIT ?
	`, xuid, limit)
	if err != nil {
		return 0, fmt.Errorf("healStats query: %w", err)
	}
	var matchIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		matchIDs = append(matchIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(matchIDs) == 0 {
		return 0, nil
	}

	// Parallélisation 2026-05-22 : 1 GetMatchStats par match (pas de film),
	// les calls peuvent tourner en parallèle. mu protège le compteur. errgroup
	// ne propage pas l'erreur (best-effort par-match).
	var mu sync.Mutex
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(healParallelism)
	for _, matchID := range matchIDs {
		matchID := matchID
		eg.Go(func() error {
			if egCtx.Err() != nil {
				return egCtx.Err()
			}
			matchJSON, gErr := client.GetMatchStats(egCtx, matchID)
			if gErr != nil {
				slog.WarnContext(egCtx, "healStats: GetMatchStats échoué",
					"match_id", matchID, "err", gErr)
				return nil
			}
			// Re-extract participants — les nouvelles colonnes (max_spree,
			// grenade/melee/power, time_played, avg_life, gamertag) sont
			// remplies par ExtractParticipants ; UPSERT préserve l'existant
			// via COALESCE.
			participants := ExtractParticipants(matchJSON)
			ensureGamertagForSelf(participants, xuid, selfGamertag)
			if pErr := InsertParticipants(egCtx, sharedDB, participants); pErr != nil {
				slog.WarnContext(egCtx, "healStats: upsert participants échoué",
					"match_id", matchID, "err", pErr)
				return nil
			}
			// Re-extract registry pour combler team_0_ps_score / team_1_ps_score.
			// UPSERT registry préserve l'existant et écrase team_X_ps_score
			// si les nouveaux ne sont pas NULL.
			reg, rErr := ExtractRegistry(matchJSON, "heal")
			if rErr == nil && reg != nil {
				if insErr := InsertRegistryIfNotExists(egCtx, sharedDB, *reg); insErr != nil {
					slog.DebugContext(egCtx, "healStats: upsert registry skipped",
						"match_id", matchID, "err", insErr)
				}
			}
			mu.Lock()
			healed++
			mu.Unlock()
			return nil
		})
	}
	_ = eg.Wait()
	return healed, nil
}
