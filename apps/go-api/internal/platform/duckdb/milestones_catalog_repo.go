// Package duckdb — milestones_catalog_repo.go : référentiel des milestones
// (table `milestone_catalog` dans metadata.duckdb).
//
// Chargé du TOML au boot via milestones.SyncCatalog. Lecture cross-titres.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/progression/milestones"
)

// MilestoneCatalogRepo persiste le référentiel des milestones (metadata.duckdb).
type MilestoneCatalogRepo struct {
	db *DB
}

// NewMilestoneCatalogRepo construit le repo.
func NewMilestoneCatalogRepo(db *DB) *MilestoneCatalogRepo {
	return &MilestoneCatalogRepo{db: db}
}

// Compile-time assertion.
var _ milestones.CatalogRepo = (*MilestoneCatalogRepo)(nil)

const milestoneCatalogSelectColumns = `
SELECT id, title_slug, metric, threshold, title_en, title_fr, icon,
       condition, condition_fr, condition_en
FROM milestone_catalog`

// Upsert insère ou remplace une entrée du catalogue.
func (r *MilestoneCatalogRepo) Upsert(ctx context.Context, e milestones.CatalogEntry) error {
	if e.ID == "" {
		return fmt.Errorf("MilestoneCatalogRepo.Upsert: empty ID")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	now := time.Now().UTC()
	// ART-safe : SELECT-then-UPDATE-or-INSERT (pas d'ON CONFLICT). L'UPDATE mute
	// title_slug et metric, indexés (idx_ms_cat_title/idx_ms_cat_metric) → ces deux
	// index sont droppés par drop_metadata_art_surface_indexes_v2 (table minuscule,
	// scan instantané) pour éliminer la surface ART.
	if err := r.db.UpsertNoConflict(ctx,
		`SELECT 1 FROM milestone_catalog WHERE id = ?`,
		[]any{e.ID},
		`UPDATE milestone_catalog SET title_slug = ?, metric = ?, threshold = ?, title_en = ?,
		 title_fr = ?, icon = ?, condition = ?, condition_fr = ?, condition_en = ?, updated_at = ? WHERE id = ?`,
		[]any{e.TitleSlug, e.Metric, e.Threshold, e.TitleEN, e.TitleFR,
			NullableStr(e.Icon), NullableStr(e.Condition), NullableStr(e.ConditionFR), NullableStr(e.ConditionEN), now, e.ID},
		`INSERT INTO milestone_catalog (id, title_slug, metric, threshold, title_en, title_fr, icon, condition, condition_fr, condition_en, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		[]any{e.ID, e.TitleSlug, e.Metric, e.Threshold, e.TitleEN, e.TitleFR,
			NullableStr(e.Icon), NullableStr(e.Condition), NullableStr(e.ConditionFR), NullableStr(e.ConditionEN), now},
	); err != nil {
		return fmt.Errorf("MilestoneCatalogRepo.Upsert: %w", err)
	}
	return nil
}

// ListByTitle retourne toutes les entrées d'un titre, triées par (metric, threshold).
func (r *MilestoneCatalogRepo) ListByTitle(ctx context.Context, titleSlug string) ([]milestones.CatalogEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := r.db.Query(ctx, milestoneCatalogSelectColumns+`
		WHERE title_slug = ?
		ORDER BY metric ASC, threshold ASC`,
		titleSlug,
	)
	if err != nil {
		return nil, fmt.Errorf("MilestoneCatalogRepo.ListByTitle: %w", err)
	}
	defer rows.Close()
	var out []milestones.CatalogEntry
	for rows.Next() {
		var (
			e           milestones.CatalogEntry
			icon        sql.NullString
			condition   sql.NullString
			conditionFR sql.NullString
			conditionEN sql.NullString
		)
		if err := rows.Scan(&e.ID, &e.TitleSlug, &e.Metric, &e.Threshold,
			&e.TitleEN, &e.TitleFR, &icon, &condition, &conditionFR, &conditionEN); err != nil {
			return nil, fmt.Errorf("MilestoneCatalogRepo.ListByTitle scan: %w", err)
		}
		if icon.Valid {
			e.Icon = icon.String
		}
		if condition.Valid {
			e.Condition = condition.String
		}
		if conditionFR.Valid {
			e.ConditionFR = conditionFR.String
		}
		if conditionEN.Valid {
			e.ConditionEN = conditionEN.String
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
