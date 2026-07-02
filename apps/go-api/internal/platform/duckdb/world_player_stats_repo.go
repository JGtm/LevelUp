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
	"strings"
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
			 kills, deaths, assists, playtime_s, medal_count,
			 kda, accuracy, damage_dealt, damage_taken, computed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	now := time.Now()
	for _, s := range stats {
		title := s.TitleSlug
		if title == "" {
			title = defaultLeaderboardTitleSlug
		}
		if _, err := tx.ExecContext(ctx, ins,
			title, s.Gamertag, s.SeasonID, s.PlaylistID,
			s.MatchCount, s.WinCount, s.LossCount, s.TieCount, s.DnfCount,
			s.Kills, s.Deaths, s.Assists, s.PlaytimeSec, s.MedalCount,
			s.KDA, s.Accuracy, s.DamageDealt, s.DamageTaken, now,
		); err != nil {
			return 0, fmt.Errorf("InsertPlayerSeasonStats (%s/%s/%s): %w", s.Gamertag, s.SeasonID, s.PlaylistID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("InsertPlayerSeasonStats: commit: %w", err)
	}
	return len(stats), nil
}

// WorldLeaderboardTopN borne l'enrichissement à la profondeur RÉELLEMENT affichée
// par le classement mondial : top 100 PAR playlist (cf. GetCSRWorldLeaderboard,
// `ORDER BY rank ASC LIMIT 100` ; service.defaultLeaderboardLimit). Enrichir les
// rangs 101+ serait du fetch jamais affiché (~moitié du coût). À garder synchronisé
// avec le display si la profondeur change.
const WorldLeaderboardTopN = 100

// WorldSeasonPlayers retourne les joueurs distincts (gamertag + xuid) présents dans
// les snapshots CSR mondiaux d'une saison, restreints au top `topN` PAR playlist
// (rank <= topN ; topN <= 0 = aucun cap, toutes playlists confondues). Sert à
// alimenter l'enrichissement (un joueur fetché une fois couvre toutes ses playlists —
// cf. insight Phase A/C). Le xuid vient du snapshot Waypoint (parsé par le scraper) :
// MAX(xuid) par gamertag préfère une valeur non-NULL quand certaines lignes de la
// saison sont antérieures à la persistance du xuid (B1). XUID vide = aucune ligne ne
// le portait → l'enrichissement retombera sur PeopleHub. `db` est un lecteur shared.
// Triés pour un ordre déterministe.
func WorldSeasonPlayers(ctx context.Context, db *sql.DB, season string, topN int) ([]domain.WorldPlayerRef, error) {
	q := `SELECT gamertag, COALESCE(MAX(xuid), '') AS xuid
		FROM world_csr_leaderboard_latest
		WHERE season_id = ? AND gamertag <> ''`
	args := []any{season}
	if topN > 0 {
		q += ` AND rank <= ?`
		args = append(args, topN)
	}
	q += ` GROUP BY gamertag ORDER BY gamertag`
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("WorldSeasonPlayers(%s): %w", season, err)
	}
	defer rows.Close()
	var out []domain.WorldPlayerRef
	for rows.Next() {
		var ref domain.WorldPlayerRef
		if err := rows.Scan(&ref.Gamertag, &ref.XUID); err != nil {
			return nil, fmt.Errorf("WorldSeasonPlayers scan: %w", err)
		}
		out = append(out, ref)
	}
	return out, rows.Err()
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
		kda, accuracy, damage_dealt, damage_taken,
		LAG(season_id)   OVER w AS prev_season_id,
		LAG(match_count) OVER w AS prev_match_count,
		LAG(win_count)   OVER w AS prev_win_count,
		LAG(kda)         OVER w AS prev_kda_raw
	FROM world_player_season_stats_latest
	WHERE title_slug = ?
	WINDOW w AS (PARTITION BY title_slug, gamertag, playlist_id ORDER BY season_id)
)
SELECT
	gamertag, season_id, playlist_id,
	match_count, win_count, loss_count, tie_count, dnf_count,
	kills, deaths, assists, playtime_s, medal_count,
	kda, accuracy, damage_dealt, damage_taken,
	win_count::DOUBLE / NULLIF(match_count, 0)                   AS win_rate,
	kills::DOUBLE / NULLIF(playtime_s / 60.0, 0)                 AS kills_per_min,
	prev_season_id,
	prev_win_count::DOUBLE / NULLIF(prev_match_count, 0)         AS prev_win_rate,
	prev_kda_raw                                                AS prev_kda,
	CASE
		WHEN prev_season_id IS NULL THEN NULL
		WHEN kda > prev_kda_raw THEN 'up'
		WHEN kda < prev_kda_raw THEN 'down'
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
	return queryWorldPlayerStats(ctx, sharedDB, season, playlist)
}

// queryWorldPlayerStats exécute worldPlayerStatsQuery sur une connexion shared
// DÉJÀ acquise (réutilisable par GetCSRWorldLeaderboard sans re-acquérir le
// reader → évite un Get imbriqué sur le SharedReader).
func queryWorldPlayerStats(ctx context.Context, db *sql.DB, season, playlist string) ([]domain.WorldPlayerSeasonStats, error) {
	rows, err := db.QueryContext(ctx, worldPlayerStatsQuery, defaultLeaderboardTitleSlug, season, playlist)
	if err != nil {
		return nil, fmt.Errorf("queryWorldPlayerStats: query: %w", err)
	}
	defer rows.Close()

	var out []domain.WorldPlayerSeasonStats
	for rows.Next() {
		var s domain.WorldPlayerSeasonStats
		var winRate, kpm, prevWinRate, prevKDA sql.NullFloat64
		var prevSeason, kdaTrend, wrTrend sql.NullString
		if err := rows.Scan(
			&s.Gamertag, &s.SeasonID, &s.PlaylistID,
			&s.MatchCount, &s.WinCount, &s.LossCount, &s.TieCount, &s.DnfCount,
			&s.Kills, &s.Deaths, &s.Assists, &s.PlaytimeSec, &s.MedalCount,
			&s.KDA, &s.Accuracy, &s.DamageDealt, &s.DamageTaken,
			&winRate, &kpm, &prevSeason, &prevWinRate, &prevKDA, &kdaTrend, &wrTrend,
		); err != nil {
			return nil, fmt.Errorf("GetWorldPlayerSeasonStats: scan: %w", err)
		}
		s.TitleSlug = defaultLeaderboardTitleSlug
		s.WinRate = nullFloat(winRate)
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

// loadPrevSeasonRanks retourne le rang de chaque gamertag à la saison PRÉCÉDENTE
// (la plus récente < season pour cette playlist) depuis world_csr_leaderboard_latest.
// Clé = gamertag en minuscules (matching case-insensitive). Vide si pas de saison
// antérieure. Sert au calcul de RankDelta (rang N vs N-1). `db` déjà acquise.
func loadPrevSeasonRanks(ctx context.Context, db *sql.DB, playlist, season string) (map[string]int, error) {
	// Saison précédente = plus grand rang NUMÉRIQUE strictement < la saison courante.
	// Un MAX(season_id) / season_id < ? SQL serait LEXICOGRAPHIQUE (csrseason6-1 >
	// csrseason13-2 car '6' > '1') → raterait par ex. prev(csrseason10-1) = csrseason6-1.
	// On choisit donc la saison précédente en Go via worldSeasonRank.
	seasons, err := scanIDColumn(ctx, db,
		`SELECT DISTINCT season_id FROM world_csr_leaderboard_latest
		 WHERE playlist_id = ? AND season_id <> ''`, playlist)
	if err != nil {
		return nil, fmt.Errorf("loadPrevSeasonRanks: seasons: %w", err)
	}
	cur := worldSeasonRank(season)
	prev, prevRank := "", -1
	for _, s := range seasons {
		if r := worldSeasonRank(s); r < cur && r > prevRank {
			prev, prevRank = s, r
		}
	}
	if prev == "" {
		return map[string]int{}, nil
	}
	rows, err := db.QueryContext(ctx,
		`SELECT gamertag, rank FROM world_csr_leaderboard_latest WHERE playlist_id = ? AND season_id = ?`,
		playlist, prev)
	if err != nil {
		return nil, fmt.Errorf("loadPrevSeasonRanks: query: %w", err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var gt string
		var rank int
		if err := rows.Scan(&gt, &rank); err != nil {
			return nil, fmt.Errorf("loadPrevSeasonRanks: scan: %w", err)
		}
		out[strings.ToLower(gt)] = rank
	}
	return out, rows.Err()
}
