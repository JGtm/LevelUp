// Package duckdb — CompareRepo : stats normalisées depuis shared.match_participants.
//
// Sprint 54 C : CompareRepository.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
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

	// shared-only via SharedReader (root-level naming). PMT-5 : win_rate title-aware
	// (fallback "mp.outcome = 2" byte-identique Halo).
	winExpr := outcomeSQLEqSlug(titleSlug, "mp.outcome", canonical.OutcomeWin, "mp.outcome = 2")
	// KDA agrégé = moyenne des KDA par match, SANS recalcul de la forme par match
	// (règle : le KDA par match est figé à l'ingestion). Le KDA per-match est NET
	// ((k + a/3) − d, peut être négatif) pour TOUS les titres (Infinite : valeur
	// CoreStats "KDA" d'API ; Halo 5 : FDA NET native). Donc AVG(mp.kda) ==
	// ((Σk + Σa/3) − Σd)/N : c'est l'agrégat NET carrière correct, identique pour
	// tous les titres. AUCUNE division par les morts (le quotient serait un BUG).
	kdaExpr := "AVG(mp.kda)"
	q := `
		SELECT
			mp.xuid,
			COALESCE(vg.gamertag, xa.gamertag, '') AS gamertag,
			COUNT(*)                               AS matches,
			AVG(CASE WHEN ` + winExpr + ` THEN 1.0 ELSE 0.0 END) AS win_rate,
			` + kdaExpr + `            AS kda,
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
		FROM match_participants mp
		LEFT JOIN v_gamertag_lookup vg ON vg.xuid = mp.xuid
		LEFT JOIN xuid_aliases xa ON xa.xuid = mp.xuid
		LEFT JOIN (
			SELECT match_id, xuid, SUM(count) AS perfect_count
			FROM medals_earned
			WHERE ` + perfectKillMedalInClause("medal_name_id", titleSlug) + `
			GROUP BY match_id, xuid
		) me ON me.match_id = mp.match_id AND me.xuid = mp.xuid
		WHERE mp.xuid = ?` + excludeCampaignByMatchID(titleSlug, "mp.match_id") + `
		GROUP BY mp.xuid, COALESCE(vg.gamertag, xa.gamertag, '')`

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CompareRepo.GetLocalStats: shared reader: %w", err)
	}
	defer release()

	row := db.QueryRowContext(ctx, q, xuid)

	var s domain.NormalizedPlayerStats
	var kda, kdr sql.NullFloat64
	err = row.Scan(
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
				(SELECT MAX(pme.performance_score) FROM player_match_enrichment_latest pme),
				0.0) AS perf_ath,
			COALESCE(
				(SELECT MAX(msr.rating_value) FROM match_skill_rank_latest msr
				 WHERE msr.rating_type = 'LUSR'),
				0.0) AS lusr_ath`

	var perfATH, lusrATH float64
	var careerRank int64
	rows, err := r.pdb.Player.QueryRowRecovered(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("CompareRepo.GetPlayerATH: %w", err)
	}
	defer rows.Close()
	if err := rows.Scan(&careerRank, &perfATH, &lusrATH); err != nil {
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
				(SELECT MAX(pme.performance_score) FROM player_match_enrichment_latest pme),
				0.0) AS perf_ath,
			COALESCE(
				(SELECT MAX(msr.rating_value) FROM match_skill_rank_latest msr
				 WHERE msr.rating_type = 'LUSR'),
				0.0) AS lusr_ath`

	var perfATH, lusrATH float64
	var careerRank int64
	rows, err := pdb.Player.QueryRowRecovered(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("CompareRepo.GetPlayerATHFor %s: %w", gamertag, err)
	}
	defer rows.Close()
	if err := rows.Scan(&careerRank, &perfATH, &lusrATH); err != nil {
		return nil, fmt.Errorf("CompareRepo.GetPlayerATHFor %s: %w", gamertag, err)
	}
	return &domain.PlayerATH{
		CareerRank: int(careerRank),
		PerfATH:    perfATH,
		LusrATH:    lusrATH,
	}, nil
}

// GetEncounterStats retourne les stats de rencontres historiques entre xuidA et xuidB.
// Deux requêtes sur shared : match_participants (ally/enemy split + winrate) +
// killer_victim_pairs (kills croisés). Best-effort — retourne nil si aucun match commun.
func (r *CompareRepo) GetEncounterStats(ctx context.Context, xuidA, xuidB string) (*domain.CompareEncounterStats, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 2 queries shared-only via SharedReader. PMT-5 : winrate ally/enemy title-aware
	// (fallback "a.outcome = 2" byte-identique Halo).
	winExpr := outcomeSQLEq(ctx, "a.outcome", canonical.OutcomeWin, "a.outcome = 2")
	qMatches := `
		SELECT
			COUNT(*) AS total,
			COUNT(CASE WHEN a.team_id IS NOT NULL AND b.team_id IS NOT NULL AND a.team_id = b.team_id THEN 1 END) AS ally_count,
			COUNT(CASE WHEN a.team_id IS NOT NULL AND b.team_id IS NOT NULL AND a.team_id != b.team_id THEN 1 END) AS enemy_count,
			SUM(CASE WHEN a.team_id IS NOT NULL AND b.team_id IS NOT NULL AND a.team_id = b.team_id AND ` + winExpr + ` THEN 1.0 ELSE 0.0 END) /
				NULLIF(COUNT(CASE WHEN a.team_id IS NOT NULL AND b.team_id IS NOT NULL AND a.team_id = b.team_id THEN 1 END), 0) AS winrate_as_ally,
			SUM(CASE WHEN a.team_id IS NOT NULL AND b.team_id IS NOT NULL AND a.team_id != b.team_id AND ` + winExpr + ` THEN 1.0 ELSE 0.0 END) /
				NULLIF(COUNT(CASE WHEN a.team_id IS NOT NULL AND b.team_id IS NOT NULL AND a.team_id != b.team_id THEN 1 END), 0) AS winrate_vs_enemy
		FROM match_participants a
		JOIN match_participants b ON b.match_id = a.match_id AND b.xuid = ?
		WHERE a.xuid = ?`

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CompareRepo.GetEncounterStats: shared reader: %w", err)
	}
	defer release()

	var total, allyCount, enemyCount int
	var winrateAsAlly, winrateVsEnemy sql.NullFloat64
	if err := db.QueryRowContext(ctx, qMatches, xuidB, xuidA).Scan(&total, &allyCount, &enemyCount, &winrateAsAlly, &winrateVsEnemy); err != nil {
		return nil, fmt.Errorf("CompareRepo.GetEncounterStats matches: %w", err)
	}
	if total == 0 {
		return nil, nil
	}

	const qKV = `
		SELECT
			COALESCE((SELECT SUM(kill_count) FROM killer_victim_pairs WHERE killer_xuid = ? AND victim_xuid = ?), 0),
			COALESCE((SELECT SUM(kill_count) FROM killer_victim_pairs WHERE killer_xuid = ? AND victim_xuid = ?), 0)`

	var killsDealt, deathsSuffered int
	if err := db.QueryRowContext(ctx, qKV, xuidA, xuidB, xuidB, xuidA).Scan(&killsDealt, &deathsSuffered); err != nil {
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

	// shared-only via SharedReader. Set perfect-kill title-aware via le titre du
	// joueur courant (pdb.TitleSlug ; HINF byte-identique = {1512363953}).
	q := `
		SELECT
			COUNT(*)                                        AS matches_count,
			COALESCE(MAX(b.max_killing_spree), 0)           AS max_killing_spree,
			COALESCE(AVG(b.avg_life_seconds), 0.0)          AS avg_life_secs,
			COALESCE(AVG(b.headshot_kills), 0.0)            AS headshot_kills_per_game,
			COALESCE(AVG(COALESCE(me.perfect_count, 0)), 0.0) AS perfect_kills_per_game
		FROM match_participants a
		JOIN match_participants b ON b.match_id = a.match_id AND b.xuid = ?
		LEFT JOIN (
			SELECT match_id, xuid, SUM(count) AS perfect_count
			FROM medals_earned
			WHERE ` + perfectKillMedalInClause("medal_name_id", pdbTitleSlug(r.pdb)) + `
			GROUP BY match_id, xuid
		) me ON me.match_id = b.match_id AND me.xuid = b.xuid
		WHERE a.xuid = ?`

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CompareRepo.GetCrossMatchSample: shared reader: %w", err)
	}
	defer release()

	var sample domain.CrossMatchSample
	err = db.QueryRowContext(ctx, q, xuidB, xuidA).Scan(
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

	// shared-only via SharedReader.
	const q = `
		SELECT xuid FROM xuid_aliases
		WHERE lower(gamertag) = lower(?)
		LIMIT 1`

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return "", fmt.Errorf("CompareRepo.ResolveXUID: shared reader: %w", err)
	}
	defer release()

	var xuid string
	err = db.QueryRowContext(ctx, q, gamertag).Scan(&xuid)
	if err == sql.ErrNoRows {
		return "", nil // non trouvé localement — pas une erreur fatale
	}
	if err != nil {
		return "", fmt.Errorf("CompareRepo.ResolveXUID: %w", err)
	}
	return xuid, nil
}
