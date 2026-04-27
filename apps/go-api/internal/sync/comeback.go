// Package sync — comeback.go : backfill du dominance_flag par match.
//
// Port de src/data/comeback_backfill.py.
//
// Ordre de priorité Python (reproduit ici) :
//  1. DOMINATION   : médaille Steaktacular (ID 1169390319) gagnée par mon équipe.
//  2. HUMILIATION  : médaille Steaktacular gagnée par l'équipe adverse.
//     3-5. REMONTADA / DÉBÂCLE / CONTRE-REMONTADA : score curve depuis kill events,
//     uniquement si le mode est de type Slayer (game_variant_name like '%slayer%').
//
// Les valeurs 3-5 utilisent l'algo ComputeDominanceFlag (analysis/comeback.go)
// avec la sensibilité "standard".
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/analysis"
)

// BackfillDominanceFlags calcule et persiste le dominance_flag pour une liste de matchs.
//
// sharedDB : connexion à shared_matches_v2.duckdb (medals, events, participants, registry).
// playerDB : connexion à stats.duckdb du joueur (écriture player_match_enrichment).
// xuid     : identifiant Xbox du joueur.
// matchIDs : liste des match_id à traiter.
func BackfillDominanceFlags(
	ctx context.Context,
	sharedDB, playerDB *sql.DB,
	xuid string,
	matchIDs []string,
) error {
	for _, matchID := range matchIDs {
		flag, err := computeMatchDominanceFlag(ctx, sharedDB, xuid, matchID)
		if err != nil {
			slog.Warn("BackfillDominanceFlags: compute", "match_id", matchID, "err", err)
			continue
		}
		if err := writeDominanceFlag(ctx, playerDB, matchID, flag); err != nil {
			slog.Warn("BackfillDominanceFlags: write", "match_id", matchID, "err", err)
		}
	}
	return nil
}

// computeMatchDominanceFlag calcule le flag pour un match.
func computeMatchDominanceFlag(ctx context.Context, db *sql.DB, xuid, matchID string) (int, error) {
	myTeamID, outcome, err := loadMyTeamAndOutcome(ctx, db, matchID, xuid)
	if err != nil {
		return 0, fmt.Errorf("team/outcome: %w", err)
	}

	// 1. Vérifier médaille Steaktacular (DOMINATION / HUMILIATION).
	steakByTeam, err := loadSteaktacularByTeam(ctx, db, matchID)
	if err != nil {
		return 0, fmt.Errorf("steaktacular: %w", err)
	}
	if steakByTeam[myTeamID] > 0 {
		return analysis.DominanceFlagDomination, nil
	}
	for teamID := range steakByTeam {
		if teamID != myTeamID && steakByTeam[teamID] > 0 {
			return analysis.DominanceFlagHumiliation, nil
		}
	}

	// 2. Comeback — uniquement pour les modes Slayer.
	gameVariant, err := loadGameVariant(ctx, db, matchID)
	if err != nil {
		return 0, nil // non critique
	}
	if !strings.Contains(strings.ToLower(gameVariant), "slayer") {
		return 0, nil
	}

	// 3. Construire la courbe de score depuis kill events.
	events, err := loadKillEventsWithTeam(ctx, db, matchID)
	if err != nil {
		return 0, nil // non critique
	}
	snapshots := analysis.BuildScoreSnapshots(events)
	return analysis.ComputeDominanceFlag(
		snapshots, myTeamID, outcome,
		"standard", false, false, matchID,
	), nil
}

// loadMyTeamAndOutcome charge team_id et outcome du joueur pour un match.
func loadMyTeamAndOutcome(ctx context.Context, db *sql.DB, matchID, xuid string) (int, int, error) {
	row := db.QueryRowContext(ctx, `
SELECT COALESCE(team_id, 0) AS team_id, COALESCE(outcome, 0) AS outcome
FROM match_participants
WHERE match_id = ? AND xuid = ?
LIMIT 1`, matchID, xuid)

	var teamID, outcome int
	if err := row.Scan(&teamID, &outcome); err != nil {
		return 0, 0, err
	}
	return teamID, outcome, nil
}

// loadSteaktacularByTeam retourne team_id → nombre de Steaktacular gagnées dans le match.
func loadSteaktacularByTeam(ctx context.Context, db *sql.DB, matchID string) (map[int]int, error) {
	rows, err := db.QueryContext(ctx, `
SELECT mp.team_id, SUM(me.count) AS total
FROM medals_earned me
JOIN match_participants mp
    ON mp.match_id = me.match_id AND mp.xuid = me.xuid
WHERE me.match_id = ? AND me.medal_name_id = ?
GROUP BY mp.team_id`, matchID, analysis.MedalSteaktacularID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]int)
	for rows.Next() {
		var teamID, total int
		if err := rows.Scan(&teamID, &total); err != nil {
			return nil, err
		}
		result[teamID] = total
	}
	return result, rows.Err()
}

// loadGameVariant charge le nom du game variant depuis match_registry.
func loadGameVariant(ctx context.Context, db *sql.DB, matchID string) (string, error) {
	row := db.QueryRowContext(ctx, `
SELECT COALESCE(game_variant_name, '') FROM match_registry WHERE match_id = ? LIMIT 1`, matchID)
	var name string
	return name, row.Scan(&name)
}

// loadKillEventsWithTeam charge les kill events avec le team_id de l'acteur.
func loadKillEventsWithTeam(ctx context.Context, db *sql.DB, matchID string) ([]analysis.KillEvent, error) {
	rows, err := db.QueryContext(ctx, `
SELECT he.time_ms, COALESCE(mp.team_id, 0) AS team_id
FROM highlight_events he
JOIN match_participants mp
    ON mp.match_id = he.match_id AND mp.xuid = he.xuid
WHERE he.match_id = ?
  AND he.event_type = 'kill'
  AND he.xuid IS NOT NULL
ORDER BY he.time_ms ASC`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []analysis.KillEvent
	for rows.Next() {
		var e analysis.KillEvent
		if err := rows.Scan(&e.TimeMS, &e.TeamID); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// writeDominanceFlag écrit ou met à jour le dominance_flag dans player_match_enrichment.
func writeDominanceFlag(ctx context.Context, db *sql.DB, matchID string, flag int) error {
	_, err := db.ExecContext(ctx, `
INSERT INTO player_match_enrichment (match_id, dominance_flag)
VALUES (?, ?)
ON CONFLICT (match_id) DO UPDATE SET dominance_flag = excluded.dominance_flag`,
		matchID, flag)
	return err
}
