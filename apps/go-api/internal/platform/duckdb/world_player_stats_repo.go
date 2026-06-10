// Package duckdb — world_player_stats_repo.go : stats joueur du classement
// mondial par saison CSR x playlist (Phase B du plan PLAN_WORLD_LEADERBOARD_ENRICHED.md).
//
// Écriture : InsertPlayerSeasonStats (INSERT pur, append-only — règle ART).
// Lecture  : GetWorldPlayerSeasonStats lit la vue _latest, dérive les ratios
//
//	(win_rate / kda / kills_per_min) et l'indicateur inter-saison via
//	LAG() (comparaison à la saison précédente OÙ la même playlist existe ;
//	l'ORDER BY season_id saute naturellement les saisons manquantes).
//
// Compteurs BRUTS stockés ; ratios JAMAIS stockés (dérivés ici). Cf. § schéma du plan.
//
// NB ordre des saisons : ORDER BY season_id (lexicographique). Correct pour les
// season_id à numéro >= 10 (ex. csrseason12-1 < csrseason13-1) — les seuls
// présents dans les snapshots scrapés. Si des saisons à 1 chiffre entraient, il
// faudrait ordonner par date (csr_season_calendars) — non requis aujourd'hui.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// InsertPlayerSeasonStats persiste un lot de stats joueur-saison-playlist en
// INSERT pur (append-only, règle ART — jamais d'UPDATE/UPSERT) dans
// world_player_season_stats. `db` est une connexion shared en écriture (cron ou
// backfill, writer unique). Tout le lot dans une transaction (atomicité +
// written_at cohérent → la vue _latest groupe par batch). Retourne le nb inséré.
func InsertPlayerSeasonStats(ctx context.Context, db *sql.DB, stats []domain.WorldPlayerSeasonStats) (int, error) {
	if len(stats) == 0 {
		return 0, nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("InsertPlayerSeasonStats: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op si Commit a réussi

	const ins = `
		INSERT INTO world_player_season_stats
			(title_slug, gamertag, season_id, playlist_id,
			 match_count, win_count, loss_count, tie_count, dnf_count,
			 kills, deaths, assists, playtime_s, medal_count, computed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	now := time.Now()
	for _, s := range stats {
		title := s.TitleSlug
		if title == "" {
			title = "halo_infinite"
		}
		if _, err := tx.ExecContext(ctx, ins,
			title, s.Gamertag, s.SeasonID, s.PlaylistID,
			s.MatchCount, s.WinCount, s.LossCount, s.TieCount, s.DnfCount,
			s.Kills, s.Deaths, s.Assists, s.PlaytimeSec, s.MedalCount, now,
		); err != nil {
			return 0, fmt.Errorf("InsertPlayerSeasonStats (%s/%s/%s): %w", s.Gamertag, s.SeasonID, s.PlaylistID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("InsertPlayerSeasonStats: commit: %w", err)
	}
	return len(stats), nil
}

// worldPlayerStatsQuery : compteurs bruts + ratios dérivés + comparaison
// inter-saison (LAG sur la saison précédente avec la même playlist) pour une
// (saison, playlist) données. Lit la vue _latest. Paramètres : title, season, playlist.
const worldPlayerStatsQuery = `
WITH hist AS (
	SELECT
		gamertag, season_id, playlist_id,
		match_count, win_count, loss_count, tie_count, dnf_count,
		kills, deaths, assists, playtime_s, medal_count,
		LAG(season_id)   OVER w AS prev_season_id,
		LAG(match_count) OVER w AS prev_match_count,
		LAG(win_count)   OVER w AS prev_win_count,
		LAG(kills)       OVER w AS prev_kills,
		LAG(deaths)      OVER w AS prev_deaths,
		LAG(assists)     OVER w AS prev_assists
	FROM world_player_season_stats_latest
	WHERE title_slug = ?
	WINDOW w AS (PARTITION BY title_slug, gamertag, playlist_id ORDER BY season_id)
)
SELECT
	gamertag, season_id, playlist_id,
	match_count, win_count, loss_count, tie_count, dnf_count,
	kills, deaths, assists, playtime_s, medal_count,
	win_count::DOUBLE / NULLIF(match_count, 0)                   AS win_rate,
	(kills + assists / 3.0) / NULLIF(deaths, 0)                  AS kda,
	kills::DOUBLE / NULLIF(playtime_s / 60.0, 0)                 AS kills_per_min,
	prev_season_id,
	prev_win_count::DOUBLE / NULLIF(prev_match_count, 0)         AS prev_win_rate,
	(prev_kills + prev_assists / 3.0) / NULLIF(prev_deaths, 0)   AS prev_kda,
	CASE
		WHEN prev_season_id IS NULL THEN NULL
		WHEN (kills + assists/3.0)/NULLIF(deaths,0) > (prev_kills + prev_assists/3.0)/NULLIF(prev_deaths,0) THEN 'up'
		WHEN (kills + assists/3.0)/NULLIF(deaths,0) < (prev_kills + prev_assists/3.0)/NULLIF(prev_deaths,0) THEN 'down'
		ELSE 'stable'
	END AS kda_trend,
	CASE
		WHEN prev_season_id IS NULL THEN NULL
		WHEN win_count::DOUBLE/NULLIF(match_count,0) > prev_win_count::DOUBLE/NULLIF(prev_match_count,0) THEN 'up'
		WHEN win_count::DOUBLE/NULLIF(match_count,0) < prev_win_count::DOUBLE/NULLIF(prev_match_count,0) THEN 'down'
		ELSE 'stable'
	END AS win_rate_trend
FROM hist
WHERE season_id = ? AND playlist_id = ?
ORDER BY gamertag`

// GetWorldPlayerSeasonStats retourne les stats enrichies (compteurs bruts +
// ratios dérivés + indicateur inter-saison) pour une (saison, playlist) du titre
// courant. Lit world_player_season_stats_latest (shared RO). Vide si aucune donnée.
func (r *LeaderboardRepo) GetWorldPlayerSeasonStats(
	ctx context.Context, season, playlist string,
) ([]domain.WorldPlayerSeasonStats, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetWorldPlayerSeasonStats: shared reader: %w", err)
	}
	defer release()

	rows, err := sharedDB.QueryContext(ctx, worldPlayerStatsQuery, defaultLeaderboardTitleSlug, season, playlist)
	if err != nil {
		return nil, fmt.Errorf("GetWorldPlayerSeasonStats: query: %w", err)
	}
	defer rows.Close()

	var out []domain.WorldPlayerSeasonStats
	for rows.Next() {
		var s domain.WorldPlayerSeasonStats
		var winRate, kda, kpm, prevWinRate, prevKDA sql.NullFloat64
		var prevSeason, kdaTrend, wrTrend sql.NullString
		if err := rows.Scan(
			&s.Gamertag, &s.SeasonID, &s.PlaylistID,
			&s.MatchCount, &s.WinCount, &s.LossCount, &s.TieCount, &s.DnfCount,
			&s.Kills, &s.Deaths, &s.Assists, &s.PlaytimeSec, &s.MedalCount,
			&winRate, &kda, &kpm, &prevSeason, &prevWinRate, &prevKDA, &kdaTrend, &wrTrend,
		); err != nil {
			return nil, fmt.Errorf("GetWorldPlayerSeasonStats: scan: %w", err)
		}
		s.TitleSlug = defaultLeaderboardTitleSlug
		s.WinRate = nullFloat(winRate)
		s.KDA = nullFloat(kda)
		s.KillsPerMin = nullFloat(kpm)
		s.PrevSeasonID = nullStr(prevSeason)
		s.PrevWinRate = nullFloat(prevWinRate)
		s.PrevKDA = nullFloat(prevKDA)
		s.KDATrend = nullStr(kdaTrend)
		s.WinRateTrend = nullStr(wrTrend)
		out = append(out, s)
	}
	return out, rows.Err()
}

// defaultLeaderboardTitleSlug : titre courant pour le classement mondial (V1 mono-titre).
const defaultLeaderboardTitleSlug = "halo_infinite"

// nullFloat convertit un sql.NullFloat64 en *float64 (nil si NULL).
func nullFloat(n sql.NullFloat64) *float64 {
	if !n.Valid {
		return nil
	}
	v := n.Float64
	return &v
}

// nullStr convertit un sql.NullString en *string (nil si NULL ou vide).
func nullStr(n sql.NullString) *string {
	if !n.Valid || n.String == "" {
		return nil
	}
	v := n.String
	return &v
}
