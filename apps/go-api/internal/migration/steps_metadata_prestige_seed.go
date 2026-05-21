package migration

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pelletier/go-toml/v2"
)

const prestigeSeedMigrationName = "seed_prestige_catalog_v1"

// RegisterPrestigeSeedMigration enregistre la migration de seed des catalogues
// Prestige (challenge_template + preset_arc) depuis les fichiers TOML.
//
// tomlDir : chemin vers config/titles/{slug}/ contenant :
//   - challenges/templates.toml
//   - arcs/presets.toml
//
// Idempotent : sans effet si la migration est déjà enregistrée.
// Doit être appelé avant RunForDB(_, TargetMetadata).
// Si le TOML est absent au moment du backfill, l'erreur est loguée (WARN)
// et backfill_done reste false — retentative au prochain boot.
func RegisterPrestigeSeedMigration(tomlDir string) {
	for _, m := range registry {
		if m.Name == prestigeSeedMigrationName {
			return
		}
	}
	Register(Migration{
		Name:        prestigeSeedMigrationName,
		TargetDB:    TargetMetadata,
		Description: "Seed challenge_template et preset_arc depuis TOML config",
		ApplySchema: func(db *sql.DB) error { return nil },
		ApplyBackfill: func(db *sql.DB) error {
			return seedPrestigeFromTOML(db, tomlDir)
		},
	})
}

// seedPrestigeFromTOML seede templates et preset arcs depuis les fichiers TOML.
// Utilise INSERT … ON CONFLICT DO UPDATE — jamais de DELETE ni TRUNCATE
// pour éviter le bug ART index de DuckDB.
func seedPrestigeFromTOML(db *sql.DB, tomlDir string) error {
	if err := seedTemplates(db, filepath.Join(tomlDir, "challenges", "templates.toml")); err != nil {
		return err
	}
	return seedPresets(db, filepath.Join(tomlDir, "arcs", "presets.toml"))
}

// ─────────── structs TOML (indépendantes du package prestige) ───────────

type seedCatalogMeta struct {
	TitleSlug string `toml:"title_slug"`
}

type seedTemplateEntry struct {
	ID              string  `toml:"id"`
	Metric          string  `toml:"metric"`
	WindowType      string  `toml:"window_type"`
	WindowValue     string  `toml:"window_value"`
	Cadence         string  `toml:"cadence"`
	EvalType        string  `toml:"eval_type"`
	ModeFilter      string  `toml:"mode_filter"`
	LabelEN         string  `toml:"label_en"`
	LabelFR         string  `toml:"label_fr"`
	DescriptionEN   string  `toml:"description_en"`
	DescriptionFR   string  `toml:"description_fr"`
	NormalTarget    float64 `toml:"normal_target"`
	HeroicTarget    float64 `toml:"heroic_target"`
	LegendaryTarget float64 `toml:"legendary_target"`
	MythicTarget    float64 `toml:"mythic_target"`
}

type seedTemplatesTOML struct {
	Meta      seedCatalogMeta     `toml:"meta"`
	Templates []seedTemplateEntry `toml:"templates"`
}

type seedArcStep struct {
	Position   int    `toml:"position"`
	TemplateID string `toml:"template_id"`
	TargetTier string `toml:"target_tier"`
}

type seedArcEntry struct {
	ID            string        `toml:"id"`
	TitleEN       string        `toml:"title_en"`
	TitleFR       string        `toml:"title_fr"`
	DescriptionEN string        `toml:"description_en"`
	DescriptionFR string        `toml:"description_fr"`
	Steps         []seedArcStep `toml:"steps"`
}

type seedPresetsTOML struct {
	Meta seedCatalogMeta `toml:"meta"`
	Arcs []seedArcEntry  `toml:"arcs"`
}

// ─────────── fonctions de seed ───────────

func seedTemplates(db *sql.DB, path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("seed templates: read %s: %w", path, err)
	}
	var doc seedTemplatesTOML
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("seed templates: parse: %w", err)
	}
	if doc.Meta.TitleSlug == "" {
		return fmt.Errorf("seed templates: meta.title_slug manquant dans %s", path)
	}
	now := time.Now().UTC()
	for _, t := range doc.Templates {
		modeFilter := t.ModeFilter
		if modeFilter == "" {
			modeFilter = "universal"
		}
		_, err := db.ExecContext(bootCtx(), `
			INSERT INTO challenge_template (
				id, title_slug, metric, window_type, window_value, cadence, eval_type,
				mode_filter, label_en, label_fr, description_en, description_fr,
				normal_target, heroic_target, legendary_target, mythic_target,
				schema_version, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?)
			ON CONFLICT (id) DO UPDATE SET
				title_slug       = excluded.title_slug,
				metric           = excluded.metric,
				window_type      = excluded.window_type,
				window_value     = excluded.window_value,
				cadence          = excluded.cadence,
				eval_type        = excluded.eval_type,
				mode_filter      = excluded.mode_filter,
				label_en         = excluded.label_en,
				label_fr         = excluded.label_fr,
				description_en   = excluded.description_en,
				description_fr   = excluded.description_fr,
				normal_target    = excluded.normal_target,
				heroic_target    = excluded.heroic_target,
				legendary_target = excluded.legendary_target,
				mythic_target    = excluded.mythic_target,
				schema_version   = excluded.schema_version,
				updated_at       = excluded.updated_at`,
			t.ID, doc.Meta.TitleSlug, t.Metric, t.WindowType, t.WindowValue,
			t.Cadence, t.EvalType, modeFilter,
			t.LabelEN, t.LabelFR, t.DescriptionEN, t.DescriptionFR,
			t.NormalTarget, t.HeroicTarget, t.LegendaryTarget, t.MythicTarget,
			now,
		)
		if err != nil {
			return fmt.Errorf("seed templates: upsert %s: %w", t.ID, err)
		}
	}
	return nil
}

func seedPresets(db *sql.DB, path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("seed presets: read %s: %w", path, err)
	}
	var doc seedPresetsTOML
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("seed presets: parse: %w", err)
	}
	if doc.Meta.TitleSlug == "" {
		return fmt.Errorf("seed presets: meta.title_slug manquant dans %s", path)
	}
	now := time.Now().UTC()
	for _, a := range doc.Arcs {
		_, err := db.ExecContext(bootCtx(), `
			INSERT INTO preset_arc (id, title_slug, title_en, title_fr,
			                       description_en, description_fr, schema_version, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, 1, ?)
			ON CONFLICT (id) DO UPDATE SET
				title_slug     = excluded.title_slug,
				title_en       = excluded.title_en,
				title_fr       = excluded.title_fr,
				description_en = excluded.description_en,
				description_fr = excluded.description_fr,
				schema_version = excluded.schema_version,
				updated_at     = excluded.updated_at`,
			a.ID, doc.Meta.TitleSlug, a.TitleEN, a.TitleFR,
			a.DescriptionEN, a.DescriptionFR, now,
		)
		if err != nil {
			return fmt.Errorf("seed presets: upsert arc %s: %w", a.ID, err)
		}
		for _, s := range a.Steps {
			_, err := db.ExecContext(bootCtx(), `
				INSERT INTO preset_arc_step (preset_arc_id, position, template_id, target_tier)
				VALUES (?, ?, ?, ?)
				ON CONFLICT (preset_arc_id, position) DO UPDATE SET
					template_id = excluded.template_id,
					target_tier = excluded.target_tier`,
				a.ID, s.Position, s.TemplateID, s.TargetTier,
			)
			if err != nil {
				return fmt.Errorf("seed presets: upsert step arc=%s pos=%d: %w", a.ID, s.Position, err)
			}
		}
	}
	return nil
}
