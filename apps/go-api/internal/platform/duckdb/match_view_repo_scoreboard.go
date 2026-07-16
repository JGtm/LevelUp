// Package duckdb — match_view_repo_scoreboard.go : scoreboard + objective
// score de la vue Match. Découpé de match_view_repo.go (god-file split,
// refactor 2026-05-27).
package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// GetMatchScoreboard retourne les stats de tous les joueurs (Q12).
// Exécutée sur SharedReader (ADR 0016) — Q12 lit medals_earned + weapon_kills
// + match_participants + v_gamertag_lookup (shared-only).
func (r *MatchViewRepo) GetMatchScoreboard(ctx context.Context, matchID string) ([]domain.ScoreboardRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchScoreboard: shared reader: %w", err)
	}
	defer release()

	// Q12 utilise 3 fois match_id : medals CTE, weapons CTE, WHERE. Set
	// perfect-kill résolu pour le titre du joueur (HINF byte-identique).
	q := resolvePerfectKillClause(Q12MatchScoreboard, "medal_name_id", pdbTitleSlug(r.pdb))
	rows, err := sharedDB.QueryContext(ctx, q, matchID, matchID, matchID)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchScoreboard: %w", err)
	}
	defer rows.Close()

	var results []domain.ScoreboardRaw
	for rows.Next() {
		var s domain.ScoreboardRaw
		// top_weapon_id est UBIGINT côté DuckDB → scanner en *uint64 pour
		// éviter l'overflow int64 sur les hash de filmshell (bit63=1).
		var topWeaponU *uint64
		if err := rows.Scan(
			&s.XUID,
			&s.Gamertag,
			&s.IsBot,
			&s.TeamID,
			&s.RankInTeam,
			&s.OutcomeCode,
			&s.PersonalScore,
			&s.Kills,
			&s.Deaths,
			&s.Assists,
			&s.KDA,
			&s.Accuracy,
			&s.TimePlayed,
			&s.TeamMMR,
			&s.EnemyMMR,
			&s.ShotsFired,
			&s.ShotsHit,
			&s.DamageDealt,
			&s.DamageTaken,
			&s.AvgLifeSeconds,
			&s.HeadshotKills,
			&s.MaxKillingSpree,
			&s.GrenadeKills,
			&s.MeleeKills,
			&s.PowerWeaponKills,
			&s.PerfectKills,
			&topWeaponU,
			&s.KillsExpected,
			&s.DeathsExpected,
			&s.KillsStdDev,
			&s.DeathsStdDev,
		); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchScoreboard scan: %w", err)
		}
		if topWeaponU != nil {
			v := int64(*topWeaponU) //nolint:gosec
			s.TopWeaponID = &v
		}
		// Sanitize : NaN/Inf venant de la DB (ex. 0/0 dans l'API Halo) cause un
		// échec silencieux de json.Encode → corps HTTP vide. On remplace par nil.
		s.PersonalScore = sanitizeF64(s.PersonalScore)
		s.KDA = sanitizeF64(s.KDA)
		s.Accuracy = sanitizeF64(s.Accuracy)
		s.TimePlayed = sanitizeF64(s.TimePlayed)
		s.TeamMMR = sanitizeF64(s.TeamMMR)
		s.EnemyMMR = sanitizeF64(s.EnemyMMR)
		s.DamageDealt = sanitizeF64(s.DamageDealt)
		s.DamageTaken = sanitizeF64(s.DamageTaken)
		s.AvgLifeSeconds = sanitizeF64(s.AvgLifeSeconds)
		s.KillsExpected = sanitizeF64(s.KillsExpected)
		s.DeathsExpected = sanitizeF64(s.DeathsExpected)
		s.KillsStdDev = sanitizeF64(s.KillsStdDev)
		s.DeathsStdDev = sanitizeF64(s.DeathsStdDev)
		results = append(results, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Résolution des labels d'armes top-weapon pour chaque joueur du scoreboard
	var topWeaponIDs []int64
	for _, s := range results {
		if s.TopWeaponID != nil {
			topWeaponIDs = append(topWeaponIDs, *s.TopWeaponID)
		}
	}
	if len(topWeaponIDs) > 0 {
		labels := r.lookupWeaponLabels(ctx, topWeaponIDs)
		for i := range results {
			if results[i].TopWeaponID != nil {
				results[i].TopWeaponLabel = labels[*results[i].TopWeaponID]
			}
		}
	}

	return results, nil
}

// GetMatchObjectiveScore retourne la somme award_score des awards de
// catégorie 'objective' pour (xuid, matchID). Mirror du squad
// LoadObjectiveScores (cf. squad_v2_adapter.go) appliqué à un seul match.
// Dégradation silencieuse à 0 si la table est absente ou ligne vide.
func (r *MatchViewRepo) GetMatchObjectiveScore(ctx context.Context, xuid, matchID string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var tableCount int
	tblRows, err := r.pdb.ReadDB().QueryRowRecovered(ctx, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_name = 'personal_score_awards'
	`)
	if err != nil {
		return 0, nil
	}
	scanErr := tblRows.Scan(&tableCount)
	_ = tblRows.Close()
	if scanErr != nil || tableCount == 0 {
		return 0, nil
	}

	var total int
	totalRows, err := r.pdb.ReadDB().QueryRowRecovered(ctx, `
		SELECT COALESCE(SUM(award_score), 0)::INTEGER
		FROM personal_score_awards_latest
		WHERE award_category = 'objective'
		  AND xuid = ?
		  AND match_id = ?
	`, xuid, matchID)
	if err != nil {
		return 0, nil
	}
	defer totalRows.Close()
	if err := totalRows.Scan(&total); err != nil {
		return 0, nil
	}
	return total, nil
}
