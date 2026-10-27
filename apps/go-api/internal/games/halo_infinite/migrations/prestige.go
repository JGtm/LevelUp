package migrations

// prestige.go — seed_prestige_catalog_v1 (named-func + TOML loaders), déplacé
// depuis internal/migration/steps_metadata_prestige_seed.go (Phase 1.5 b10, voie B).
//
// Particularité : ce step est enregistré DYNAMIQUEMENT au boot avec le chemin
// config (RegisterPrestigeSeedMigration(tomlDir)) car son backfill lit des TOML —
// le slice statique Steps() ne porte pas de repoRoot. Boot appelle
// halomigrations.RegisterPrestigeSeedMigration ; pour un 2e titre, boot appellera
// l'équivalent du titre. Le schéma (create_prestige_metadata_schema) + les ALTER
// challenge_template restent dans Steps() (inline, voir steps.go).

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pelletier/go-toml/v2"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/migration"
)

const prestigeSeedMigrationName = "seed_prestige_catalog_v1"

// RegisterPrestigeSeedMigration enregistre la migration de seed des catalogues
// Prestige (challenge_template + preset_arc) depuis les TOML. Idempotent.
// Doit être appelé avant RunForDB(_, TargetMetadata). tomlDir = config/titles/{slug}/.
func RegisterPrestigeSeedMigration(tomlDir string) {
	if migration.IsRegistered(prestigeSeedMigrationName) {
		return
	}
	migration.Register(migration.Migration{
		Name:        prestigeSeedMigrationName,
		TargetDB:    migration.TargetMetadata,
		Description: "Seed challenge_template et preset_arc depuis TOML config",
		ApplySchema: func(db *sql.DB) error { return nil },
		ApplyBackfill: func(db *sql.DB) error {
			return seedPrestigeFromTOML(db, tomlDir)
		},
	})
}

// seedPrestigeFromTOML seede templates et preset arcs depuis les fichiers TOML.
// INSERT … ON CONFLICT DO UPDATE — jamais de DELETE/TRUNCATE (bug ART DuckDB).
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
	ID              string    `toml:"id"`
	Metric          string    `toml:"metric"`
	WindowType      string    `toml:"window_type"`
	WindowValue     string    `toml:"window_value"`
	Cadence         string    `toml:"cadence"`
	EvalType        string    `toml:"eval_type"`
	ModeFilter      string    `toml:"mode_filter"`
	Label           i18n.Text `toml:"label"`
	Description     i18n.Text `toml:"description"`
	NormalTarget    float64   `toml:"normal_target"`
	HeroicTarget    float64   `toml:"heroic_target"`
	LegendaryTarget float64   `toml:"legendary_target"`
	MythicTarget    float64   `toml:"mythic_target"`
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
	ID          string        `toml:"id"`
	Title       i18n.Text     `toml:"title"`
	Description i18n.Text     `toml:"description"`
	Steps       []seedArcStep `toml:"steps"`
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
		_, err := db.ExecContext(migration.BootCtx(), `
			INSERT INTO challenge_template (
				id, title_slug, metric, window_type, window_value, cadence, eval_type,
				mode_filter,
				normal_target, heroic_target, legendary_target, mythic_target,
				schema_version, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?)
			ON CONFLICT (id) DO UPDATE SET
				title_slug       = excluded.title_slug,
				metric           = excluded.metric,
				window_type      = excluded.window_type,
				window_value     = excluded.window_value,
				cadence          = excluded.cadence,
				eval_type        = excluded.eval_type,
				mode_filter      = excluded.mode_filter,
				normal_target    = excluded.normal_target,
				heroic_target    = excluded.heroic_target,
				legendary_target = excluded.legendary_target,
				mythic_target    = excluded.mythic_target,
				schema_version   = excluded.schema_version,
				updated_at       = excluded.updated_at`,
			t.ID, doc.Meta.TitleSlug, t.Metric, t.WindowType, t.WindowValue,
			t.Cadence, t.EvalType, modeFilter,
			t.NormalTarget, t.HeroicTarget, t.LegendaryTarget, t.MythicTarget,
			now,
		)
		if err != nil {
			return fmt.Errorf("seed templates: upsert %s: %w", t.ID, err)
		}
		if err := migration.UpsertAssetTexts(migration.BootCtx(), db, migration.AssetTypeChallengeTemplate, t.ID, t.Label, t.Description); err != nil {
			return fmt.Errorf("seed templates: textes de %s: %w", t.ID, err)
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
		_, err := db.ExecContext(migration.BootCtx(), `
			INSERT INTO preset_arc (id, title_slug, schema_version, updated_at)
			VALUES (?, ?, 1, ?)
			ON CONFLICT (id) DO UPDATE SET
				title_slug     = excluded.title_slug,
				schema_version = excluded.schema_version,
				updated_at     = excluded.updated_at`,
			a.ID, doc.Meta.TitleSlug, now,
		)
		if err != nil {
			return fmt.Errorf("seed presets: upsert arc %s: %w", a.ID, err)
		}
		if err := migration.UpsertAssetTexts(migration.BootCtx(), db, migration.AssetTypePresetArc, a.ID, a.Title, a.Description); err != nil {
			return fmt.Errorf("seed presets: textes de l'arc %s: %w", a.ID, err)
		}
		for _, s := range a.Steps {
			_, err := db.ExecContext(migration.BootCtx(), `
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
