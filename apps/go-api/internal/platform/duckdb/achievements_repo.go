// Package duckdb — AchievementsRepo : progression d'achievements Xbox du joueur.
//
// Lit la table player_achievements depuis stats.duckdb. Les définitions
// bilingues (nom, description, gamerscore, image) vivent dans metadata.duckdb
// et sont exposées par MetadataRepo.GetAchievementDefinitions.
package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// AchievementsRepo implémente port.AchievementsRepository.
type AchievementsRepo struct {
	pdb *PlayerDB
}

// NewAchievementsRepo crée un AchievementsRepo depuis un PlayerDB.
func NewAchievementsRepo(pdb *PlayerDB) *AchievementsRepo {
	return &AchievementsRepo{pdb: pdb}
}

// GetPlayerAchievements retourne toutes les lignes player_achievements du joueur,
// triées par achievement_id (tri stable pour les tests). Slice vide si aucune
// donnée (joueur jamais syncé pour les achievements).
func (r *AchievementsRepo) GetPlayerAchievements(ctx context.Context) ([]domain.PlayerAchievementRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().QueryRecovered(ctx, `
		SELECT achievement_id, unlocked, unlocked_at, current_progress, target_progress
		FROM player_achievements
		ORDER BY achievement_id`)
	if err != nil {
		return nil, fmt.Errorf("GetPlayerAchievements query: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	out := make([]domain.PlayerAchievementRow, 0, 128)
	for rows.Next() {
		var row domain.PlayerAchievementRow
		if err := rows.Scan(
			&row.AchievementID,
			&row.Unlocked,
			&row.UnlockedAt,
			&row.CurrentProgress,
			&row.TargetProgress,
		); err != nil {
			return nil, fmt.Errorf("GetPlayerAchievements scan: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetPlayerAchievements rows: %w", err)
	}
	return out, nil
}
