// Package duckdb — engagement_score_repo_queries.go : methodes loaders Phase 4
// extraites de engagement_score_repo.go pour respecter limite 500L.
//
// Couvre : load match metadata, events, team xuids, coefficients all-modes,
// liste matchs PvP recents.
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

func (r *EngagementScoreRepo) LoadMatchEngagementContext(
	ctx context.Context,
	matchID, xuid string,
) (*port.MatchEngagementContext, error) {
	if matchID == "" || xuid == "" {
		return nil, errors.New("EngagementScoreRepo.LoadMatchEngagementContext: matchID and xuid required")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	const q = `
		SELECT
			mr.match_id,
			COALESCE(EPOCH_MS(mr.start_time_utc), EPOCH_MS(mr.start_time)),
			COALESCE(EPOCH_MS(mr.end_time_utc), EPOCH_MS(mr.end_time)),
			COALESCE(mr.is_ranked, FALSE),
			COALESCE(mr.is_firefight, FALSE),
			COALESCE(mp.team_id, 0),
			COALESCE(mp.personal_score, 0),
			COALESCE(mp.kills, 0),
			COALESCE(mp.assists, 0),
			mr.map_name
		FROM shared.match_registry mr
		JOIN shared.match_participants mp ON mr.match_id = mp.match_id
		WHERE mr.match_id = ? AND mp.xuid = ?
	`
	var mctx port.MatchEngagementContext
	err := r.pdb.ReadDB().QueryRow(ctx, q, matchID, xuid).Scan(
		&mctx.MatchID,
		&mctx.StartTimeMS,
		&mctx.EndTimeMS,
		&mctx.IsRanked,
		&mctx.IsPvE,
		&mctx.TargetTeamID,
		&mctx.PersonalScore,
		&mctx.Kills,
		&mctx.Assists,
		&mctx.MapName,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("LoadMatchEngagementContext: %w", err)
	}

	// Charger NTeam et NHumansLobby separement (bots = xuid LIKE 'bid(%').
	const sizeQ = `
		SELECT
			SUM(CASE WHEN team_id = ? AND xuid NOT LIKE 'bid(%' THEN 1 ELSE 0 END),
			SUM(CASE WHEN xuid NOT LIKE 'bid(%' THEN 1 ELSE 0 END)
		FROM shared.match_participants WHERE match_id = ?
	`
	var nTeam, nLobby sql.NullInt64
	_ = r.pdb.ReadDB().QueryRow(ctx, sizeQ, mctx.TargetTeamID, matchID).Scan(&nTeam, &nLobby)
	mctx.NTeam = int(nTeam.Int64)
	mctx.NHumansLobby = int(nLobby.Int64)
	mctx.IsTeamMode = mctx.NTeam > 1

	return &mctx, nil
}

// LoadEventsForMatch charge tous les events highlight_events d'un match.
func (r *EngagementScoreRepo) LoadEventsForMatch(
	ctx context.Context,
	matchID string,
) ([]canonical.HighlightEvent, error) {
	if matchID == "" {
		return nil, errors.New("EngagementScoreRepo.LoadEventsForMatch: matchID required")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	const q = `
		SELECT match_id, event_type, COALESCE(time_ms, 0), COALESCE(xuid, '')
		FROM shared.highlight_events
		WHERE match_id = ?
		ORDER BY time_ms ASC
	`
	rows, err := r.pdb.ReadDB().Query(ctx, q, matchID)
	if err != nil {
		return nil, fmt.Errorf("LoadEventsForMatch: %w", err)
	}
	defer rows.Close()

	out := make([]canonical.HighlightEvent, 0)
	for rows.Next() {
		var ev canonical.HighlightEvent
		if err := rows.Scan(&ev.MatchID, &ev.EventType, &ev.TimeMS, &ev.XUID); err != nil {
			continue
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

// LoadTeamXUIDs charge les XUIDs des coequipiers humains (joueur cible exclu).
func (r *EngagementScoreRepo) LoadTeamXUIDs(
	ctx context.Context,
	matchID string,
	teamID int,
	targetXUID string,
) (map[string]bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const q = `
		SELECT xuid FROM shared.match_participants
		WHERE match_id = ?
		  AND team_id = ?
		  AND xuid NOT LIKE 'bid(%'
		  AND xuid <> ?
	`
	rows, err := r.pdb.ReadDB().Query(ctx, q, matchID, teamID, targetXUID)
	if err != nil {
		return nil, fmt.Errorf("LoadTeamXUIDs: %w", err)
	}
	defer rows.Close()

	out := make(map[string]bool)
	for rows.Next() {
		var x string
		if err := rows.Scan(&x); err == nil {
			out[x] = true
		}
	}
	return out, rows.Err()
}

// LoadAllCoefficients charge tous les coefficients du joueur, toutes categories.
func (r *EngagementScoreRepo) LoadAllCoefficients(
	ctx context.Context,
	xuid string,
) ([]domain.EngagementCoefficient, error) {
	if xuid == "" {
		return nil, errors.New("EngagementScoreRepo.LoadAllCoefficients: xuid required")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if !r.coefficientsTableExists(ctx) {
		return nil, port.ErrEngagementUnavailable
	}

	const q = `
		SELECT xuid, mode_category, coef_team_share, coef_lobby_share,
		       n_matches, last_updated
		FROM engagement_coefficients
		WHERE xuid = ?
		ORDER BY mode_category
	`
	rows, err := r.pdb.ReadDB().Query(ctx, q, xuid)
	if err != nil {
		return nil, fmt.Errorf("LoadAllCoefficients: %w", err)
	}
	defer rows.Close()

	out := make([]domain.EngagementCoefficient, 0)
	for rows.Next() {
		var c domain.EngagementCoefficient
		if err := rows.Scan(
			&c.XUID, &c.ModeCategory, &c.CoefTeamShare, &c.CoefLobbyShare,
			&c.NMatches, &c.LastUpdated,
		); err != nil {
			continue
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListRecentPvPMatchIDs liste les match_ids PvP du joueur, ordre chronologique
// croissant. Utilise par le service Timeseries (Mock 11).
func (r *EngagementScoreRepo) ListRecentPvPMatchIDs(
	ctx context.Context,
	xuid string,
	limit int,
) ([]string, error) {
	if xuid == "" || limit <= 0 {
		return nil, errors.New("ListRecentPvPMatchIDs: xuid and limit > 0 required")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	q := `
		SELECT mr.match_id
		FROM shared.match_registry mr
		JOIN shared.match_participants mp ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?
		  AND mr.start_time IS NOT NULL
		  AND COALESCE(mr.is_firefight, FALSE) = FALSE
		ORDER BY mr.start_time DESC
		LIMIT ?
	`
	rows, err := r.pdb.ReadDB().Query(ctx, q, xuid, limit)
	if err != nil {
		return nil, fmt.Errorf("ListRecentPvPMatchIDs: %w", err)
	}
	defer rows.Close()

	out := make([]string, 0, limit)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			out = append(out, id)
		}
	}
	// Inverser pour ordre chronologique croissant (oldest first).
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, rows.Err()
}
