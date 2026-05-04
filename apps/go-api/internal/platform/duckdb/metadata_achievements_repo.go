// Package duckdb — metadata_achievements_repo.go : référentiel bilingue d'achievements.
//
// Étend MetadataRepo (même pattern que medal_cache_repo.go, map_cache_repo.go).
// Lit xbox_achievement_definitions dans metadata.duckdb. Cette table est peuplée
// par sync.SyncAchievements (sync engine ou CLI levelup sync-achievements).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// GetAchievementDefinitions retourne toutes les définitions bilingues d'achievements
// triées par achievement_id (tri stable). Slice vide si la table est encore vide
// (backfill jamais lancé). Implémente port.MetadataAchievementsRepository.
func (r *MetadataRepo) GetAchievementDefinitions(ctx context.Context) ([]domain.AchievementDefinitionRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.meta.Query(ctx, `
		SELECT achievement_id, name_en, name_fr, description_en, description_fr,
		       locked_desc_en, locked_desc_fr, gamerscore, image_url, is_secret,
		       rarity_category, rarity_percent
		FROM xbox_achievement_definitions
		ORDER BY achievement_id`)
	if err != nil {
		return nil, fmt.Errorf("GetAchievementDefinitions query: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	out := make([]domain.AchievementDefinitionRow, 0, 256)
	for rows.Next() {
		var (
			row            domain.AchievementDefinitionRow
			descEN, descFR sql.NullString
			lockEN, lockFR sql.NullString
			imageURL       sql.NullString
			rarityCat      sql.NullString
			rarityPct      sql.NullFloat64
		)
		if err := rows.Scan(
			&row.AchievementID,
			&row.NameEN, &row.NameFR,
			&descEN, &descFR,
			&lockEN, &lockFR,
			&row.Gamerscore,
			&imageURL,
			&row.IsSecret,
			&rarityCat,
			&rarityPct,
		); err != nil {
			return nil, fmt.Errorf("GetAchievementDefinitions scan: %w", err)
		}
		row.DescriptionEN = descEN.String
		row.DescriptionFR = descFR.String
		row.LockedDescEN = lockEN.String
		row.LockedDescFR = lockFR.String
		row.ImageURL = imageURL.String
		row.RarityCategory = rarityCat.String
		if rarityPct.Valid {
			row.RarityPercent = rarityPct.Float64
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetAchievementDefinitions rows: %w", err)
	}
	return out, nil
}
