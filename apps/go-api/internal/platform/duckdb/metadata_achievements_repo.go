// Package duckdb — metadata_achievements_repo.go : référentiel multi-langue d'achievements.
//
// Étend MetadataRepo (même pattern que medal_cache_repo.go, map_cache_repo.go).
// Lit xbox_achievement_definitions dans metadata.duckdb. Cette table est peuplée
// par sync.SyncAchievements (sync engine ou CLI levelup sync-achievements).
//
// Les noms et descriptions vivent dans asset_translations, une ligne par langue : type
// `achievement` (nom dans `name`, description dans `description`) et `achievement_locked` (texte
// de succès verrouillé dans `name`), `asset_id` = achievement_id (ADR 0040 D-5).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
)

// GetAchievementDefinitions retourne toutes les définitions d'achievements du titre `titleSlug`
// (filtre title_id), triées par achievement_id (tri stable), avec leurs textes dans chaque langue
// pour laquelle une traduction existe (le choix de la langue se fait en aval, `Text.Pick`).
// titleSlug == "" → titre par défaut (halo_infinite), préservant le comportement
// historique. Slice vide si la table est encore vide (backfill jamais lancé).
// Implémente port.MetadataAchievementsRepository (PMT-6 : lecture title-aware).
func (r *MetadataRepo) GetAchievementDefinitions(ctx context.Context, titleSlug string) ([]domain.AchievementDefinitionRow, error) {
	if titleSlug == "" {
		titleSlug = titlePkg.DefaultSlug
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	out, err := r.achievementBaseRows(ctx, titleSlug)
	if err != nil {
		return nil, err
	}
	if len(out) > 0 {
		if err := r.attachAchievementTexts(ctx, out); err != nil {
			return nil, err
		}
	}
	slog.DebugContext(ctx, "GetAchievementDefinitions", "title", titleSlug, "count", len(out))
	return out, nil
}

// achievementBaseRows lit les champs non localisés des définitions d'un titre.
func (r *MetadataRepo) achievementBaseRows(ctx context.Context, titleSlug string) ([]domain.AchievementDefinitionRow, error) {
	rows, err := r.meta.Query(ctx, `
		SELECT achievement_id, gamerscore, image_url, is_secret,
		       rarity_category, rarity_percent, xbox_title_id, service_config_id
		FROM xbox_achievement_definitions
		WHERE title_id = ?
		ORDER BY achievement_id`, titleSlug)
	if err != nil {
		return nil, fmt.Errorf("GetAchievementDefinitions query: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	out := make([]domain.AchievementDefinitionRow, 0, 256)
	for rows.Next() {
		var (
			row             domain.AchievementDefinitionRow
			imageURL        sql.NullString
			rarityCat       sql.NullString
			rarityPct       sql.NullFloat64
			xboxTitleID     sql.NullString
			serviceConfigID sql.NullString
		)
		if err := rows.Scan(
			&row.AchievementID,
			&row.Gamerscore,
			&imageURL,
			&row.IsSecret,
			&rarityCat,
			&rarityPct,
			&xboxTitleID,
			&serviceConfigID,
		); err != nil {
			return nil, fmt.Errorf("GetAchievementDefinitions scan: %w", err)
		}
		row.ImageURL = imageURL.String
		row.RarityCategory = rarityCat.String
		if rarityPct.Valid {
			row.RarityPercent = rarityPct.Float64
		}
		row.XboxTitleID = xboxTitleID.String
		row.ServiceConfigID = serviceConfigID.String
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetAchievementDefinitions rows: %w", err)
	}
	return out, nil
}

// attachAchievementTexts renseigne Name, Description (type `achievement`) et LockedDesc (type
// `achievement_locked`) de chaque définition depuis asset_translations.
func (r *MetadataRepo) attachAchievementTexts(ctx context.Context, defs []domain.AchievementDefinitionRow) error {
	ids := make([]string, len(defs))
	for i, d := range defs {
		ids[i] = d.AchievementID
	}
	texts, err := LoadAssetTexts(ctx, r.meta, AssetTypeAchievement, ids)
	if err != nil {
		return fmt.Errorf("GetAchievementDefinitions texts: %w", err)
	}
	locked, err := LoadAssetTexts(ctx, r.meta, AssetTypeAchievementLocked, ids)
	if err != nil {
		return fmt.Errorf("GetAchievementDefinitions locked texts: %w", err)
	}
	for i := range defs {
		t := texts[defs[i].AchievementID]
		defs[i].Name = textOrEmpty(t.Name)
		defs[i].Description = textOrEmpty(t.Description)
		defs[i].LockedDesc = textOrEmpty(locked[defs[i].AchievementID].Name)
	}
	return nil
}
