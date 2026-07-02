// Package sync — session_recalc.go : recalcul des sessions pour un joueur.
//
// RecalculatePlayerSessions ouvre les deux DBs, charge les matchs depuis
// shared, calcule les sessions et écrit session_id / session_label dans
// player_match_enrichment.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// sessionMatchesSQL charge les matchs d'un joueur depuis shared DB.
// Params : ?1 = xuid (exclusion subquery), ?2 = xuid (filtre WHERE).
// Note : sans préfixe "shared." — query exécutée directement sur la shared DB.
const sessionMatchesSQL = `
SELECT
    mp.match_id,
    r.start_time,
    (SELECT string_agg(t.xuid, ',' ORDER BY t.xuid)
     FROM match_participants t
     WHERE t.match_id = mp.match_id AND t.team_id = mp.team_id AND t.xuid <> ?
    ) AS teammates_sig,
    COALESCE(r.is_ranked, FALSE) AS is_ranked,
    mp.time_played_seconds,
    CASE WHEN mp.time_played_seconds IS NOT NULL
         THEN r.start_time + INTERVAL (mp.time_played_seconds || ' seconds')
         ELSE NULL
    END AS end_time
FROM match_participants mp
JOIN match_registry r ON r.match_id = mp.match_id
WHERE mp.xuid = ?
ORDER BY r.start_time ASC`

// loadSessionMatchRowsDirect charge les matchs depuis shared DB (sans préfixe "shared.").
func loadSessionMatchRowsDirect(ctx context.Context, sharedDB *sql.DB, xuid string) ([]domain.SessionMatchRow, error) {
	rows, err := sharedDB.QueryContext(ctx, sessionMatchesSQL, xuid, xuid)
	if err != nil {
		return nil, fmt.Errorf("loadSessionMatchRowsDirect: %w", err)
	}
	defer rows.Close()

	var results []domain.SessionMatchRow
	for rows.Next() {
		var m domain.SessionMatchRow
		var endTime *time.Time
		if err := rows.Scan(
			&m.MatchID,
			&m.StartTime,
			&m.TeammatesSig,
			&m.IsRanked,
			&m.TimePlayedSecs,
			&endTime,
		); err != nil {
			return nil, fmt.Errorf("loadSessionMatchRowsDirect scan: %w", err)
		}
		m.EndTime = endTime
		results = append(results, m)
	}
	return results, rows.Err()
}

// LookupFriendXUIDs retourne les XUIDs des gamertags depuis xuid_aliases.
// Les gamertags non trouvés sont silencieusement ignorés.
func LookupFriendXUIDs(ctx context.Context, sharedDB *sql.DB, gamertags []string) []string {
	if len(gamertags) == 0 {
		return nil
	}
	var xuids []string
	for _, gt := range gamertags {
		var xuid string
		if err := sharedDB.QueryRowContext(ctx,
			"SELECT xuid FROM xuid_aliases WHERE LOWER(gamertag) = LOWER(?)", gt,
		).Scan(&xuid); err == nil && xuid != "" {
			xuids = append(xuids, xuid)
		}
	}
	return xuids
}

// recalculateSessionsInline calcule les sessions de façon incrémentale sur des DBs
// déjà ouvertes (les leases sont détenues par le caller). Délègue à appendSessionsInline
// qui ne recalcule et ne réécrit que ce qui a changé (nouveaux matchs + labels si étendus).
//
// Retourne (matchs_mis_à_jour, error).
func recalculateSessionsInline(
	ctx context.Context,
	playerDB, sharedDB *sql.DB,
	xuid string,
	opts domain.SessionComputeOptions,
	friendGamertags []string,
) (int, error) {
	if len(friendGamertags) > 0 {
		opts.FriendsXUIDs = LookupFriendXUIDs(ctx, sharedDB, friendGamertags)
	}
	return appendSessionsInline(ctx, playerDB, sharedDB, xuid, opts)
}

// RecalculatePlayerSessions recalcule les sessions pour un joueur.
// Acquiert les leases, ouvre les DBs, calcule et écrit dans player_match_enrichment.
// friendGamertags : gamertags des amis à résoudre en XUIDs (utilisé si TeamChangeMode = "friends").
// provider (sprint B1 commit 13b) : si non-nil, route les accès shared via
// Provider.AcquireWriter pour coordonner avec readers HTTP. Si nil, fallback
// legacy (dblease + OpenSharedDB direct).
// Retourne le nombre de matchs mis à jour.
func RecalculatePlayerSessions(
	ctx context.Context,
	provider sharedprovider.Provider,
	playerDBPath, sharedDBPath, xuid string,
	opts domain.SessionComputeOptions,
	friendGamertags []string,
) (int, error) {
	slog.InfoContext(ctx, "sessions: recalcul démarré",
		"xuid", xuid,
		"gap_minutes", opts.GapMinutes,
		"team_change_mode", opts.TeamChangeMode,
	)
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, playerDBPath, dblease.KindPlayer)
	if err != nil {
		return 0, fmt.Errorf("RecalculatePlayerSessions lease player: %w", err)
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("RecalculatePlayerSessions OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	// Sprint B1 commit 13b : helper standalone (Provider en B-swap, legacy sinon).
	sharedDB, releaseShared, err := AcquireSharedWriterStandalone(ctxkeys.WithDBWriterLabel(ctx, "sessions_recalc"), provider, sharedDBPath)
	if err != nil {
		return 0, fmt.Errorf("RecalculatePlayerSessions: %w", err)
	}
	defer releaseShared()

	// Résoudre les XUIDs des amis si nécessaire.
	if len(friendGamertags) > 0 {
		opts.FriendsXUIDs = LookupFriendXUIDs(ctx, sharedDB, friendGamertags)
	}

	matchRows, err := loadSessionMatchRowsDirect(ctx, sharedDB, xuid)
	if err != nil {
		return 0, fmt.Errorf("RecalculatePlayerSessions loadMatches: %w", err)
	}
	if len(matchRows) == 0 {
		slog.InfoContext(ctx, "sessions: aucun match trouvé, recalcul ignoré", "xuid", xuid)
		return 0, nil
	}

	assignments := analysis.ComputeSessionsWithContext(matchRows, opts)
	groups := analysis.BuildSessionGroups(matchRows, assignments)
	assignments = analysis.MergeSessionLabels(assignments, groups)

	n, err := WriteSessionAssignments(ctx, playerHandle.SQLDb(), assignments)
	if err != nil {
		return 0, fmt.Errorf("RecalculatePlayerSessions write: %w", err)
	}
	slog.InfoContext(ctx, "sessions: recalcul terminé",
		"xuid", xuid,
		"matches_loaded", len(matchRows),
		"matches_updated", n,
	)
	return n, nil
}
