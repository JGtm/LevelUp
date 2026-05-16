// Package duckdb — CompareRepo : stats normalisées depuis shared.match_participants.
//
// Sprint 54 C : CompareRepository.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"levelup/go-api/internal/domain"
)

// CompareRepo implémente port.CompareRepository.
type CompareRepo struct {
	pdb *PlayerDB
}

// NewCompareRepo crée un CompareRepo.
func NewCompareRepo(pdb *PlayerDB) *CompareRepo {
	return &CompareRepo{pdb: pdb}
}

// GetLocalStats calcule les stats normalisées depuis shared.match_participants.
// Tous les champs utilisent le shared DB — peut être appelé pour n'importe quel XUID croisé.
func (r *CompareRepo) GetLocalStats(ctx context.Context, xuid, titleSlug string) (*domain.NormalizedPlayerStats, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	const q = `
		SELECT
			mp.xuid,
			COALESCE(vg.gamertag, xa.gamertag, '') AS gamertag,
			COUNT(*)                               AS matches,
			AVG(CASE WHEN mp.outcome = 2 THEN 1.0 ELSE 0.0 END) AS win_rate,
			AVG(COALESCE(mp.kills, 0) + 0.33 * COALESCE(mp.assists, 0)) /
				NULLIF(AVG(COALESCE(mp.deaths, 0)), 0)            AS kda,
			AVG(COALESCE(mp.kills, 0)) /
				NULLIF(AVG(COALESCE(mp.deaths, 0)), 0)            AS kdr,
			AVG(COALESCE(mp.kills, 0))                            AS kills_per_game,
			AVG(COALESCE(mp.deaths, 0))                           AS deaths_per_game,
			AVG(COALESCE(mp.assists, 0))                          AS assists_per_game,
			AVG(COALESCE(mp.accuracy, 0.0)) / 100.0              AS accuracy,
			AVG(COALESCE(mp.damage_dealt, 0.0))                  AS damage_per_game,
			AVG(COALESCE(mp.damage_taken, 0.0))                  AS damage_taken_per_game,
			MAX(COALESCE(mp.max_killing_spree, 0))               AS max_killing_spree,
			AVG(COALESCE(mp.avg_life_seconds, 0.0))              AS avg_life_secs,
			AVG(COALESCE(mp.headshot_kills, 0))                  AS headshot_kills_per_game,
			AVG(COALESCE(me.perfect_count, 0.0))                 AS perfect_kills_per_game
		FROM shared.match_participants mp
		LEFT JOIN shared.v_gamertag_lookup vg ON vg.xuid = mp.xuid
		LEFT JOIN shared.xuid_aliases xa ON xa.xuid = mp.xuid
		LEFT JOIN (
			SELECT match_id, xuid, SUM(count) AS perfect_count
			FROM shared.medals_earned
			WHERE medal_name_id = 1512363953
			GROUP BY match_id, xuid
		) me ON me.match_id = mp.match_id AND me.xuid = mp.xuid
		WHERE mp.xuid = ?
		GROUP BY mp.xuid, COALESCE(vg.gamertag, xa.gamertag, '')`

	row := r.pdb.Player.QueryRow(ctx, q, xuid)

	var s domain.NormalizedPlayerStats
	var kda, kdr sql.NullFloat64
	err := row.Scan(
		&s.XUID, &s.Gamertag, &s.Matches,
		&s.WinRate,
		&kda, &kdr,
		&s.KillsPerGame, &s.DeathsPerGame, &s.AssistsPerGame,
		&s.Accuracy, &s.DamagePerGame,
		&s.DamageTakenPerGame,
		&s.MaxKillingSpree,
		&s.AvgLifeSecs,
		&s.HeadshotKillsPerGame,
		&s.PerfectKillsPerGame,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("CompareRepo.GetLocalStats: joueur %s non trouvé", xuid)
	}
	if err != nil {
		return nil, fmt.Errorf("CompareRepo.GetLocalStats: %w", err)
	}
	if kda.Valid {
		s.KDA = kda.Float64
	}
	if kdr.Valid {
		s.KDR = kdr.Float64
	}
	s.TitleSlug = titleSlug
	return &s, nil
}

// GetPlayerATH retourne les métriques all-time depuis pdb.Player (stats.duckdb).
// N'appeler que pour le joueur A (connexion du joueur principal).
func (r *CompareRepo) GetPlayerATH(ctx context.Context) (*domain.PlayerATH, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Toutes les tables sont dans stats.duckdb (pdb.Player, sans préfixe de schéma).
	const q = `
		SELECT
			COALESCE(
				(SELECT cp.rank FROM career_progression cp
				 ORDER BY cp.recorded_at DESC LIMIT 1),
				0) AS career_rank,
			COALESCE(
				(SELECT MAX(pme.performance_score) FROM player_match_enrichment pme),
				0.0) AS perf_ath,
			COALESCE(
				(SELECT MAX(msr.rating_value) FROM match_skill_rank msr
				 WHERE msr.rating_type = 'LUSR'),
				0.0) AS lusr_ath`

	var perfATH, lusrATH float64
	var careerRank int64
	err := r.pdb.Player.QueryRow(ctx, q).Scan(&careerRank, &perfATH, &lusrATH)
	if err != nil {
		return nil, fmt.Errorf("CompareRepo.GetPlayerATH: %w", err)
	}
	return &domain.PlayerATH{
		CareerRank: int(careerRank),
		PerfATH:    perfATH,
		LusrATH:    lusrATH,
	}, nil
}

// GetPlayerATHFor retourne les métriques all-time pour n'importe quel joueur local
// en le cherchant dans le pool global par la clé "{titleSlug}:{gamertag}".
// Retourne nil, nil si le joueur n'est pas dans le pool (best-effort).
func (r *CompareRepo) GetPlayerATHFor(ctx context.Context, gamertag, titleSlug string) (*domain.PlayerATH, error) {
	key := titleSlug + ":" + gamertag
	pdb, ok := LookupFromPool(key)
	if !ok {
		slog.DebugContext(ctx, "CompareRepo.GetPlayerATHFor: joueur non dans le pool", "gamertag", gamertag)
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	const q = `
		SELECT
			COALESCE(
				(SELECT cp.rank FROM career_progression cp
				 ORDER BY cp.recorded_at DESC LIMIT 1),
				0) AS career_rank,
			COALESCE(
				(SELECT MAX(pme.performance_score) FROM player_match_enrichment pme),
				0.0) AS perf_ath,
			COALESCE(
				(SELECT MAX(msr.rating_value) FROM match_skill_rank msr
				 WHERE msr.rating_type = 'LUSR'),
				0.0) AS lusr_ath`

	var perfATH, lusrATH float64
	var careerRank int64
	err := pdb.Player.QueryRow(ctx, q).Scan(&careerRank, &perfATH, &lusrATH)
	if err != nil {
		return nil, fmt.Errorf("CompareRepo.GetPlayerATHFor %s: %w", gamertag, err)
	}
	return &domain.PlayerATH{
		CareerRank: int(careerRank),
		PerfATH:    perfATH,
		LusrATH:    lusrATH,
	}, nil
}

// GetFavoriteWeapon retourne l'arme avec le plus de kills depuis shared.v_weapon_kills.
// Lecture depuis pdb.Player (shared attaché) + labels depuis pdb.Metadata.
// Retourne nil si aucune donnée disponible — best-effort, jamais d'erreur fatale.
func (r *CompareRepo) GetFavoriteWeapon(ctx context.Context, xuid string) (*domain.WeaponHighlight, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	const q = `
		SELECT wk.effective_weapon_id AS weapon_id, COUNT(*) AS kills
		FROM shared.v_weapon_kills wk
		WHERE wk.xuid = ?
		  AND wk.effective_weapon_id NOT IN (0, 1, 2)
		GROUP BY wk.effective_weapon_id
		ORDER BY kills DESC
		LIMIT 1`

	var wid UBigint
	var kills int
	err := r.pdb.Player.QueryRow(ctx, q, xuid).Scan(&wid, &kills)
	if err != nil {
		// Inclut ErrNoRows et erreur de table manquante — best-effort.
		if err != sql.ErrNoRows {
			slog.DebugContext(ctx, "CompareRepo.GetFavoriteWeapon: aucune donnée arme (best-effort)", "xuid", xuid, "err", err)
		}
		return nil, nil //nolint:nilerr
	}

	idStr := strconv.FormatUint(uint64(wid), 10) //nolint:gosec
	wh := &domain.WeaponHighlight{
		WeaponID: wid.Int64(),
		Kills:    kills,
		LabelFR:  idStr,
		LabelEN:  idStr,
	}

	if r.pdb.Metadata != nil {
		wh.LabelFR, wh.LabelEN = r.lookupWeaponLabelCompare(ctx, wid.Int64())
	}
	return wh, nil
}

// lookupWeaponLabelCompare résout les labels EN/FR depuis metadata.weapon_labels.
// Injecte l'ID comme literal décimal pour éviter les problèmes de cast UBIGINT.
func (r *CompareRepo) lookupWeaponLabelCompare(ctx context.Context, weaponID int64) (labelFR, labelEN string) {
	idStr := strconv.FormatUint(uint64(weaponID), 10) //nolint:gosec
	labelFR = idStr
	labelEN = idStr

	q := fmt.Sprintf( //nolint:gosec
		`SELECT name_fr, name_en FROM weapon_labels WHERE weapon_id = %s LIMIT 1`, idStr)
	var nameFR, nameEN sql.NullString
	if err := r.pdb.Metadata.QueryRow(ctx, q).Scan(&nameFR, &nameEN); err != nil {
		return
	}
	if nameFR.Valid && nameFR.String != "" {
		labelFR = nameFR.String
	} else if nameEN.Valid && nameEN.String != "" {
		labelFR = nameEN.String
	}
	if nameEN.Valid && nameEN.String != "" {
		labelEN = nameEN.String
	}
	return
}

// GetEncounterStats retourne les stats de rencontres historiques entre xuidA et xuidB.
// Deux requêtes sur shared : match_participants (ally/enemy split + winrate) +
// killer_victim_pairs (kills croisés). Best-effort — retourne nil si aucun match commun.
func (r *CompareRepo) GetEncounterStats(ctx context.Context, xuidA, xuidB string) (*domain.CompareEncounterStats, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	const qMatches = `
		SELECT
			COUNT(*) AS total,
			COUNT(CASE WHEN a.team_id IS NOT NULL AND b.team_id IS NOT NULL AND a.team_id = b.team_id THEN 1 END) AS ally_count,
			COUNT(CASE WHEN a.team_id IS NOT NULL AND b.team_id IS NOT NULL AND a.team_id != b.team_id THEN 1 END) AS enemy_count,
			SUM(CASE WHEN a.team_id IS NOT NULL AND b.team_id IS NOT NULL AND a.team_id = b.team_id AND a.outcome = 2 THEN 1.0 ELSE 0.0 END) /
				NULLIF(COUNT(CASE WHEN a.team_id IS NOT NULL AND b.team_id IS NOT NULL AND a.team_id = b.team_id THEN 1 END), 0) AS winrate_as_ally,
			SUM(CASE WHEN a.team_id IS NOT NULL AND b.team_id IS NOT NULL AND a.team_id != b.team_id AND a.outcome = 2 THEN 1.0 ELSE 0.0 END) /
				NULLIF(COUNT(CASE WHEN a.team_id IS NOT NULL AND b.team_id IS NOT NULL AND a.team_id != b.team_id THEN 1 END), 0) AS winrate_vs_enemy
		FROM shared.match_participants a
		JOIN shared.match_participants b ON b.match_id = a.match_id AND b.xuid = ?
		WHERE a.xuid = ?`

	var total, allyCount, enemyCount int
	var winrateAsAlly, winrateVsEnemy sql.NullFloat64
	if err := r.pdb.Player.QueryRow(ctx, qMatches, xuidB, xuidA).Scan(&total, &allyCount, &enemyCount, &winrateAsAlly, &winrateVsEnemy); err != nil {
		return nil, fmt.Errorf("CompareRepo.GetEncounterStats matches: %w", err)
	}
	if total == 0 {
		return nil, nil
	}

	const qKV = `
		SELECT
			COALESCE((SELECT SUM(kill_count) FROM shared.killer_victim_pairs WHERE killer_xuid = ? AND victim_xuid = ?), 0),
			COALESCE((SELECT SUM(kill_count) FROM shared.killer_victim_pairs WHERE killer_xuid = ? AND victim_xuid = ?), 0)`

	var killsDealt, deathsSuffered int
	if err := r.pdb.Player.QueryRow(ctx, qKV, xuidA, xuidB, xuidB, xuidA).Scan(&killsDealt, &deathsSuffered); err != nil {
		slog.DebugContext(ctx, "CompareRepo.GetEncounterStats: killer_victim non disponible (best-effort)", "err", err)
	}

	enc := &domain.CompareEncounterStats{
		TotalEncounters: total,
		AllyCount:       allyCount,
		EnemyCount:      enemyCount,
		KillsDealt:      killsDealt,
		DeathsSuffered:  deathsSuffered,
	}
	if winrateAsAlly.Valid {
		enc.WinrateAsAlly = &winrateAsAlly.Float64
	}
	if winrateVsEnemy.Valid {
		enc.WinrateVsEnemy = &winrateVsEnemy.Float64
	}
	return enc, nil
}

// GetCrossMatchSample agrège les métriques locale-only du joueur xuidB calculées
// uniquement sur les matchs où xuidA et xuidB ont joué ensemble.
//
// Réutilise le pattern d'auto-jointure de GetEncounterStats et les agrégats de
// GetLocalStats : la jointure restreint l'échantillon, mais les formules
// (MAX max_killing_spree, AVG avg_life_seconds, AVG headshot_kills, AVG perfect_count)
// sont strictement identiques à celles utilisées pour un joueur local.
//
// Best-effort : retourne (nil, nil) si aucun match croisé n'existe.
func (r *CompareRepo) GetCrossMatchSample(ctx context.Context, xuidA, xuidB string) (*domain.CrossMatchSample, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	const q = `
		SELECT
			COUNT(*)                                        AS matches_count,
			COALESCE(MAX(b.max_killing_spree), 0)           AS max_killing_spree,
			COALESCE(AVG(b.avg_life_seconds), 0.0)          AS avg_life_secs,
			COALESCE(AVG(b.headshot_kills), 0.0)            AS headshot_kills_per_game,
			COALESCE(AVG(COALESCE(me.perfect_count, 0)), 0.0) AS perfect_kills_per_game
		FROM shared.match_participants a
		JOIN shared.match_participants b ON b.match_id = a.match_id AND b.xuid = ?
		LEFT JOIN (
			SELECT match_id, xuid, SUM(count) AS perfect_count
			FROM shared.medals_earned
			WHERE medal_name_id = 1512363953
			GROUP BY match_id, xuid
		) me ON me.match_id = b.match_id AND me.xuid = b.xuid
		WHERE a.xuid = ?`

	var sample domain.CrossMatchSample
	err := r.pdb.Player.QueryRow(ctx, q, xuidB, xuidA).Scan(
		&sample.MatchesCount,
		&sample.MaxKillingSpree,
		&sample.AvgLifeSecs,
		&sample.HeadshotKillsPerGame,
		&sample.PerfectKillsPerGame,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("CompareRepo.GetCrossMatchSample: %w", err)
	}
	if sample.MatchesCount == 0 {
		return nil, nil
	}
	return &sample, nil
}

// ResolveXUID retourne le XUID correspondant à un gamertag dans le registre partagé.
func (r *CompareRepo) ResolveXUID(ctx context.Context, gamertag string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const q = `
		SELECT xuid FROM shared.xuid_aliases
		WHERE lower(gamertag) = lower(?)
		LIMIT 1`

	var xuid string
	err := r.pdb.Player.QueryRow(ctx, q, gamertag).Scan(&xuid)
	if err == sql.ErrNoRows {
		return "", nil // non trouvé localement — pas une erreur fatale
	}
	if err != nil {
		return "", fmt.Errorf("CompareRepo.ResolveXUID: %w", err)
	}
	return xuid, nil
}
