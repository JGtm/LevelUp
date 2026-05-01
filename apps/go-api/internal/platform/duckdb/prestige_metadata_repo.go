// Package duckdb — repositories Prestige référentiels (metadata.duckdb).
//
// Implémente prestige.TemplateRepo et prestige.PresetArcRepo.

package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/prestige"
)

// ─────────── TemplateRepo ───────────

// PrestigeTemplateRepo implémente prestige.TemplateRepo.
type PrestigeTemplateRepo struct{ db *DB }

func NewPrestigeTemplateRepo(db *DB) *PrestigeTemplateRepo { return &PrestigeTemplateRepo{db: db} }

var _ prestige.TemplateRepo = (*PrestigeTemplateRepo)(nil)

func (r *PrestigeTemplateRepo) ListByTitle(ctx context.Context, titleSlug string) ([]prestige.Template, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := r.db.Query(ctx, templateSelectColumns+" WHERE title_slug = ? ORDER BY cadence, id", titleSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []prestige.Template
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *PrestigeTemplateRepo) GetByID(ctx context.Context, id string) (prestige.Template, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	row := r.db.QueryRow(ctx, templateSelectColumns+" WHERE id = ?", id)
	t, err := scanTemplate(row)
	if errors.Is(err, sql.ErrNoRows) {
		return prestige.Template{}, fmt.Errorf("%w: template %s", prestige.ErrChallengeNotFound, id)
	}
	return t, err
}

func (r *PrestigeTemplateRepo) Suggest(ctx context.Context, titleSlug string, excludeIDs []string, count int) ([]prestige.Template, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if count <= 0 {
		count = 3
	}

	q := templateSelectColumns + " WHERE title_slug = ?"
	args := []any{titleSlug}
	if len(excludeIDs) > 0 {
		placeholders := strings.Repeat("?,", len(excludeIDs))
		placeholders = placeholders[:len(placeholders)-1]
		q += fmt.Sprintf(" AND id NOT IN (%s)", placeholders)
		for _, id := range excludeIDs {
			args = append(args, id)
		}
	}
	q += " ORDER BY id LIMIT ?"
	args = append(args, count)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []prestige.Template
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *PrestigeTemplateRepo) Replace(ctx context.Context, titleSlug string, templates []prestige.Template) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Workaround bug ART index DuckDB : DELETE (même par PK) échoue avec
	// "Failed to delete all rows from index" dans ce contexte pool partagé.
	// TRUNCATE évite entièrement les opérations d'index — vidage atomique bulk.
	// Acceptable car Replace porte TOUJOURS la liste complète pour le title_slug.
	if _, err := r.db.Exec(ctx, `TRUNCATE TABLE challenge_template`); err != nil {
		return fmt.Errorf("truncate: %w", err)
	}
	for _, t := range templates {
		if _, err := r.db.Exec(ctx, `
			INSERT INTO challenge_template (
				id, title_slug, metric, window_type, window_value, cadence, eval_type,
				mode_filter, label_en, label_fr, description_en, description_fr,
				normal_target, heroic_target, legendary_target, mythic_target,
				schema_version, updated_at
			) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		`,
			t.ID, t.TitleSlug, t.Metric, string(t.WindowType), t.WindowValue,
			string(t.Cadence), string(t.EvalType), t.ModeFilter,
			t.LabelEN, t.LabelFR, t.DescriptionEN, t.DescriptionFR,
			t.NormalTarget, t.HeroicTarget, t.LegendaryTarget, t.MythicTarget,
			t.SchemaVersion, t.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert %s: %w", t.ID, err)
		}
	}
	return nil
}

const templateSelectColumns = `
	SELECT id, title_slug, metric, window_type, COALESCE(window_value, ''),
	       cadence, eval_type, COALESCE(mode_filter, 'universal'),
	       label_en, label_fr, COALESCE(description_en, ''), COALESCE(description_fr, ''),
	       normal_target, heroic_target, legendary_target, mythic_target,
	       schema_version, updated_at
	FROM challenge_template`

func scanTemplate(row rowScanner) (prestige.Template, error) {
	var t prestige.Template
	var windowType, cadence, evalType string
	err := row.Scan(
		&t.ID, &t.TitleSlug, &t.Metric, &windowType, &t.WindowValue,
		&cadence, &evalType, &t.ModeFilter,
		&t.LabelEN, &t.LabelFR, &t.DescriptionEN, &t.DescriptionFR,
		&t.NormalTarget, &t.HeroicTarget, &t.LegendaryTarget, &t.MythicTarget,
		&t.SchemaVersion, &t.UpdatedAt,
	)
	if err != nil {
		return t, err
	}
	t.WindowType = prestige.WindowType(windowType)
	t.Cadence = prestige.Cadence(cadence)
	t.EvalType = prestige.EvalType(evalType)
	return t, nil
}

// ─────────── PresetArcRepo ───────────

// PrestigePresetArcRepo implémente prestige.PresetArcRepo.
type PrestigePresetArcRepo struct{ db *DB }

func NewPrestigePresetArcRepo(db *DB) *PrestigePresetArcRepo { return &PrestigePresetArcRepo{db: db} }

var _ prestige.PresetArcRepo = (*PrestigePresetArcRepo)(nil)

func (r *PrestigePresetArcRepo) ListByTitle(ctx context.Context, titleSlug string) ([]prestige.PresetArc, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := r.db.Query(ctx, `
		SELECT id, title_slug, title_en, title_fr,
		       COALESCE(description_en, ''), COALESCE(description_fr, ''),
		       schema_version, updated_at
		FROM preset_arc WHERE title_slug = ? ORDER BY id
	`, titleSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []prestige.PresetArc
	for rows.Next() {
		var p prestige.PresetArc
		if err := rows.Scan(&p.ID, &p.TitleSlug, &p.TitleEN, &p.TitleFR,
			&p.DescriptionEN, &p.DescriptionFR, &p.SchemaVersion, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PrestigePresetArcRepo) GetByID(ctx context.Context, id string) (prestige.PresetArc, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var p prestige.PresetArc
	err := r.db.QueryRow(ctx, `
		SELECT id, title_slug, title_en, title_fr,
		       COALESCE(description_en, ''), COALESCE(description_fr, ''),
		       schema_version, updated_at
		FROM preset_arc WHERE id = ?
	`, id).Scan(&p.ID, &p.TitleSlug, &p.TitleEN, &p.TitleFR,
		&p.DescriptionEN, &p.DescriptionFR, &p.SchemaVersion, &p.UpdatedAt)
	return p, err
}

func (r *PrestigePresetArcRepo) GetSteps(ctx context.Context, presetArcID string) ([]prestige.PresetArcStep, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.db.Query(ctx, `
		SELECT preset_arc_id, position, template_id, target_tier
		FROM preset_arc_step WHERE preset_arc_id = ? ORDER BY position
	`, presetArcID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []prestige.PresetArcStep
	for rows.Next() {
		var s prestige.PresetArcStep
		var tier string
		if err := rows.Scan(&s.PresetArcID, &s.Position, &s.TemplateID, &tier); err != nil {
			return nil, err
		}
		s.TargetTier = prestige.Tier(tier)
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PrestigePresetArcRepo) Replace(ctx context.Context, titleSlug string, arcs []prestige.PresetArc, steps []prestige.PresetArcStep) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Workaround bug ART index DuckDB : DELETE (même par PK) échoue avec
	// "Failed to delete all rows from index" dans ce contexte pool partagé.
	// TRUNCATE évite entièrement les opérations d'index — vidage atomique bulk.
	if _, err := r.db.Exec(ctx, `TRUNCATE TABLE preset_arc_step`); err != nil {
		return fmt.Errorf("truncate steps: %w", err)
	}
	if _, err := r.db.Exec(ctx, `TRUNCATE TABLE preset_arc`); err != nil {
		return fmt.Errorf("truncate arcs: %w", err)
	}

	for _, a := range arcs {
		if _, err := r.db.Exec(ctx, `
			INSERT INTO preset_arc (id, title_slug, title_en, title_fr,
			                       description_en, description_fr, schema_version, updated_at)
			VALUES (?,?,?,?,?,?,?,?)
		`, a.ID, a.TitleSlug, a.TitleEN, a.TitleFR, a.DescriptionEN, a.DescriptionFR,
			a.SchemaVersion, a.UpdatedAt); err != nil {
			return fmt.Errorf("insert arc %s: %w", a.ID, err)
		}
	}
	for _, s := range steps {
		if _, err := r.db.Exec(ctx, `
			INSERT INTO preset_arc_step (preset_arc_id, position, template_id, target_tier)
			VALUES (?, ?, ?, ?)
		`, s.PresetArcID, s.Position, s.TemplateID, string(s.TargetTier)); err != nil {
			return fmt.Errorf("insert step: %w", err)
		}
	}
	return nil
}
