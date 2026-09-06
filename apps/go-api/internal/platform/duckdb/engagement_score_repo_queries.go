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

	"levelup/go-api/internal/analysis"
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

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadMatchEngagementContext: shared reader: %w", err)
	}
	defer release()

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
			COALESCE(mr.map_name_fr, mr.map_name),
			mr.map_id
		FROM match_registry mr
		JOIN match_participants mp ON mr.match_id = mp.match_id
		WHERE mr.match_id = ? AND mp.xuid = ?
	`
	var mctx port.MatchEngagementContext
	var mapID sql.NullString
	err = sharedDB.QueryRowContext(ctx, q, matchID, xuid).Scan(
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
		&mapID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("LoadMatchEngagementContext: %w", err)
	}

	// map_name_fr de match_registry est systematiquement NULL -> le COALESCE
	// retombe sur l'EN. On resout le nom FR via metadata.asset_translations
	// (meme source canonique que applyMapFRTranslations). Best-effort.
	if mapID.Valid && mapID.String != "" {
		if fr, ok := r.resolveMapNameFR(ctx, mapID.String); ok {
			mctx.MapName = &fr
		}
	}

	// Charger NTeam et NHumansLobby separement (bots = xuid LIKE 'bid(%').
	var sizeQ = `
		SELECT
			SUM(CASE WHEN team_id = ? AND ` + analysis.SQLIsNotBotCol("xuid") + ` THEN 1 ELSE 0 END),
			SUM(CASE WHEN ` + analysis.SQLIsNotBotCol("xuid") + ` THEN 1 ELSE 0 END)
		FROM match_participants WHERE match_id = ?
	`
	var nTeam, nLobby sql.NullInt64
	_ = sharedDB.QueryRowContext(ctx, sizeQ, mctx.TargetTeamID, matchID).Scan(&nTeam, &nLobby)
	mctx.NTeam = int(nTeam.Int64)
	mctx.NHumansLobby = int(nLobby.Int64)
	mctx.IsTeamMode = mctx.NTeam > 1

	return &mctx, nil
}

// resolveMapNameFR resout le nom FR d'une map depuis metadata.asset_translations
// par asset_id (= map_id). match_registry.map_name_fr etant toujours NULL, c'est
// la seule source FR fiable (cf. reference_asset_translations_fr + filters_repo
// applyMapFRTranslations). Best-effort : ("", false) si Metadata absent, pas de
// ligne FR ou erreur — l'appelant garde alors le nom EN.
func (r *EngagementScoreRepo) resolveMapNameFR(ctx context.Context, mapID string) (string, bool) {
	if r.pdb == nil {
		return "", false
	}
	return mapNameFRFromAssetTranslations(ctx, r.pdb.Metadata, mapID)
}

// mapNameFRFromAssetTranslations est LE corps de la resolution ci-dessus, promu en
// fonction de paquet le 2026-09-06 (correction R3) pour que le lecteur tactique
// s'en serve au lieu d'en ecrire une TROISIEME copie — il y en avait deja deux dans
// ce paquet (celle-ci et `FiltersRepo.applyMapFRTranslations`, qui resout par NOM
// EN faute de map_id sous la main).
//
// Best-effort assume : une metadata absente, une carte sans traduction ou une
// erreur de lecture rendent ("", false), et l'appelant garde le nom EN. Un nom de
// carte manquant n'est pas une panne d'affichage.
func mapNameFRFromAssetTranslations(ctx context.Context, meta *DB, mapID string) (string, bool) {
	if meta == nil || mapID == "" {
		return "", false
	}
	const q = `
		SELECT name FROM asset_translations
		WHERE asset_type = 'map' AND asset_id = ? AND lang IN ('fr-FR', 'fr')
		ORDER BY CASE WHEN lang = 'fr-FR' THEN 0 ELSE 1 END
		LIMIT 1
	`
	rows, err := meta.QueryRecovered(ctx, q, mapID)
	if err != nil {
		return "", false
	}
	defer rows.Close()
	if rows.Next() {
		var name string
		if rows.Scan(&name) == nil && name != "" {
			return name, true
		}
	}
	return "", false
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

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadEventsForMatch: shared reader: %w", err)
	}
	defer release()

	const q = `
		SELECT match_id, event_type, COALESCE(time_ms, 0), COALESCE(xuid, '')
		FROM highlight_events
		WHERE match_id = ?
		ORDER BY time_ms ASC
	`
	rows, err := sharedDB.QueryContext(ctx, q, matchID)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Title-agnostic : si highlight_events ne porte aucun kill/death (ex. Halo 5
	// = médailles seules), on synthétise les events kill/death depuis
	// killer_victim_pairs (kills horodatés, time_ms relatif au match) et on les
	// fusionne aux médailles, triés par TimeMS. Sans ça la courbe d'engagement
	// mesurerait la cadence des médailles, pas celle des kills. Sur Infinite
	// (kills dans highlight_events) ce chemin n'est jamais pris.
	if !analysis.HasCanonicalKillOrDeath(out) {
		synth, serr := r.loadSyntheticKillEvents(ctx, sharedDB, matchID)
		if serr == nil && len(synth) > 0 {
			out = analysis.MergeAndSortCanonicalEvents(out, synth)
		}
	}
	return out, nil
}

// loadSyntheticKillEvents charge killer_victim_pairs d'un match et synthétise
// des events canoniques kill/death (1 paire → 1 kill + 1 death) via le helper
// partagé analysis.SynthesizeKillEventsFromKVPairs. Best-effort : retourne nil
// sans erreur si la table/colonnes sont absentes.
func (r *EngagementScoreRepo) loadSyntheticKillEvents(
	ctx context.Context,
	sharedDB *sql.DB,
	matchID string,
) ([]canonical.HighlightEvent, error) {
	// Source canonique depuis le 2026-08-03. Bots (xuid NULL) écartés au SQL et non
	// normalisés en chaîne vide — même raison qu'en Q32c (cf. Q32cSquadKVPairsTemplate).
	const q = `
		SELECT feed_killer_xuid, victim_xuid, time_ms
		FROM ` + KillEventsCanonicalTable + `
		WHERE match_id = ?
		  AND feed_killer_xuid IS NOT NULL
		  AND victim_xuid      IS NOT NULL
		ORDER BY time_ms ASC
	`
	rows, err := sharedDB.QueryContext(ctx, q, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	inputs := make([]analysis.KVSyntheticInput, 0)
	for rows.Next() {
		var in analysis.KVSyntheticInput
		if err := rows.Scan(&in.KillerXUID, &in.VictimXUID, &in.TimeMS); err != nil {
			continue
		}
		inputs = append(inputs, in)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return analysis.SynthesizeKillEventsFromKVPairs(inputs, matchID), nil
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

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadTeamXUIDs: shared reader: %w", err)
	}
	defer release()

	var q = `
		SELECT xuid FROM match_participants
		WHERE match_id = ?
		  AND team_id = ?
		  AND ` + analysis.SQLIsNotBotCol("xuid") + `
		  AND xuid <> ?
	`
	rows, err := sharedDB.QueryContext(ctx, q, matchID, teamID, targetXUID)
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

	// coef_team_share non lu (D5, colonne inerte).
	const q = `
		SELECT xuid, mode_category, coef_lobby_share,
		       n_matches, last_updated
		FROM engagement_coefficients
		WHERE xuid = ?
		ORDER BY mode_category
	`
	rows, err := r.pdb.ReadDB().QueryRecovered(ctx, q, xuid)
	if err != nil {
		return nil, fmt.Errorf("LoadAllCoefficients: %w", err)
	}
	defer rows.Close()

	out := make([]domain.EngagementCoefficient, 0)
	for rows.Next() {
		var c domain.EngagementCoefficient
		if err := rows.Scan(
			&c.XUID, &c.ModeCategory, &c.CoefLobbyShare,
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

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListRecentPvPMatchIDs: shared reader: %w", err)
	}
	defer release()

	q := `
		SELECT mr.match_id
		FROM match_registry mr
		JOIN match_participants mp ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?
		  AND mr.start_time IS NOT NULL
		  AND COALESCE(mr.is_firefight, FALSE) = FALSE
		ORDER BY mr.start_time DESC
		LIMIT ?
	`
	rows, err := sharedDB.QueryContext(ctx, q, xuid, limit)
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
