// Package duckdb — milestones_catalog_repo.go : référentiel des milestones
// (table `milestone_catalog` dans metadata.duckdb).
//
// Chargé du TOML au boot via milestones.SyncCatalog. Lecture cross-titres.
//
// Le titre et la condition lisible de chaque jalon vivent dans asset_translations (type
// `milestone`, une ligne par langue : titre dans `name`, condition dans `description`), pas dans
// des colonnes appariées de milestone_catalog (ADR 0040 D-5). La ligne de milestone_catalog ne
// porte que les champs non localisés, dont la formule technique `condition`.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/migration"
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
SELECT id, title_slug, metric, threshold, icon, condition
FROM milestone_catalog`

// Upsert insère ou remplace une entrée du catalogue : la ligne de milestone_catalog et ses
// traductions, dans une même transaction.
//
// ART-safe : SELECT-then-UPDATE-or-INSERT (pas d'ON CONFLICT). L'UPDATE mute title_slug et
// metric, indexés autrefois (idx_ms_cat_title/idx_ms_cat_metric) → ces deux index sont droppés par
// drop_metadata_art_surface_indexes_v2 (table minuscule, scan instantané) pour éliminer la
// surface ART.
func (r *MilestoneCatalogRepo) Upsert(ctx context.Context, e milestones.CatalogEntry) error {
	if e.ID == "" {
		return fmt.Errorf("MilestoneCatalogRepo.Upsert: empty ID")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err := r.db.WithReopenOnInvalidated(func() error {
		tx, err := r.db.loadSQL().BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback() //nolint:errcheck // sans effet après Commit
		if err := migration.UpsertMilestoneCatalogEntry(ctx, tx, e); err != nil {
			return err
		}
		return tx.Commit()
	})
	if err != nil {
		return fmt.Errorf("MilestoneCatalogRepo.Upsert: %w", err)
	}
	return nil
}

// ListByTitle retourne toutes les entrées d'un titre, triées par (metric, threshold). Le titre et
// la condition lisible portent toutes les langues pour lesquelles une traduction existe ; le choix
// de la langue de la requête se fait en aval (`Text.Pick`).
func (r *MilestoneCatalogRepo) ListByTitle(ctx context.Context, titleSlug string) ([]milestones.CatalogEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := r.listBaseRows(ctx, titleSlug)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return out, nil
	}
	ids := make([]string, len(out))
	for i, e := range out {
		ids[i] = e.ID
	}
	texts, err := LoadAssetTexts(ctx, r.db, AssetTypeMilestone, ids)
	if err != nil {
		return nil, fmt.Errorf("MilestoneCatalogRepo.ListByTitle: %w", err)
	}
	for i := range out {
		t := texts[out[i].ID]
		out[i].Title = textOrEmpty(t.Name)
		out[i].ConditionText = textOrEmpty(t.Description)
	}
	return out, nil
}

// listBaseRows lit les champs non localisés des jalons d'un titre.
func (r *MilestoneCatalogRepo) listBaseRows(ctx context.Context, titleSlug string) ([]milestones.CatalogEntry, error) {
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
			e         milestones.CatalogEntry
			icon      sql.NullString
			condition sql.NullString
		)
		if err := rows.Scan(&e.ID, &e.TitleSlug, &e.Metric, &e.Threshold, &icon, &condition); err != nil {
			return nil, fmt.Errorf("MilestoneCatalogRepo.ListByTitle scan: %w", err)
		}
		e.Icon = icon.String
		e.Condition = condition.String
		out = append(out, e)
	}
	return out, rows.Err()
}

// textOrEmpty rend un Text non nul (le contrat sert un objet de langues, éventuellement vide).
func textOrEmpty(t i18n.Text) i18n.Text {
	if t == nil {
		return i18n.Text{}
	}
	return t
}
