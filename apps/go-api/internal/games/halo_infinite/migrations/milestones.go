package migrations

// milestones.go — seed_milestone_catalog_v1 (named-func + TOML loaders), déplacé
// depuis internal/migration/steps_metadata_milestones_seed.go (Phase 1.5 b11, voie B).
//
// Même pattern que prestige.go : seed TOML enregistré DYNAMIQUEMENT au boot via
// RegisterMilestonesSeedMigration(configTitlesRoot) car il lit
// config/titles/{slug}/milestones/catalog.toml (le slice statique Steps() ne porte
// pas de repoRoot). Le schéma create_milestone_catalog_metadata reste dans Steps()
// (inline, voir steps.go). Boot appelle halomigrations.RegisterMilestonesSeedMigration.

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/progression/milestones"
)

// v2 (A9) : bump depuis v1 pour RE-SEEDER les DBs existantes avec les colonnes
// condition_fr / condition_en (descriptions lisibles localisées des jalons).
const milestonesSeedMigrationName = "seed_milestone_catalog_v2"

// RegisterMilestonesSeedMigration enregistre la migration de seed du catalogue
// milestones. configTitlesRoot = chemin vers config/titles/ (parent des dossiers
// par titre) ; la migration itère sur chaque sous-dossier et applique le seed si
// <slug>/milestones/catalog.toml existe. Idempotent.
func RegisterMilestonesSeedMigration(configTitlesRoot string) {
	if migration.IsRegistered(milestonesSeedMigrationName) {
		return
	}
	migration.Register(migration.Migration{
		Name:        milestonesSeedMigrationName,
		TargetDB:    migration.TargetMetadata,
		Description: "Seed milestone_catalog depuis config/titles/*/milestones/catalog.toml",
		ApplySchema: func(_ *sql.DB) error { return nil },
		ApplyBackfill: func(db *sql.DB) error {
			return seedMilestonesAllTitles(db, configTitlesRoot)
		},
	})
}

// seedMilestonesAllTitles itère sur config/titles/{slug}/milestones/catalog.toml
// et applique le seed pour chaque titre trouvé.
func seedMilestonesAllTitles(db *sql.DB, configTitlesRoot string) error {
	entries, err := os.ReadDir(configTitlesRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Pas de config/titles/ → no-op (cas tests :memory:).
			return nil
		}
		return fmt.Errorf("seed milestones: read %s: %w", configTitlesRoot, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(configTitlesRoot, e.Name(), "milestones", "catalog.toml")
		if _, statErr := os.Stat(path); statErr != nil {
			// Titre sans catalogue milestones → skip silencieux.
			continue
		}
		if err := seedMilestonesFromTOML(db, path); err != nil {
			return fmt.Errorf("seed milestones %s: %w", e.Name(), err)
		}
	}
	return nil
}

// seedMilestonesFromTOML applique le seed pour un titre. Utilise
// milestones.LoadCatalogFromFile pour parser le TOML (validation incluse) puis
// fait des UPSERTs SQL en série.
func seedMilestonesFromTOML(db *sql.DB, path string) error {
	catalogEntries, err := milestones.LoadCatalogFromFile(path)
	if err != nil {
		return fmt.Errorf("load %s: %w", path, err)
	}
	// A9 : garantit les colonnes condition_fr/condition_en avant l'INSERT (le seed
	// tourne aussi hors migration en test, sans passer par create_milestone_catalog).
	// AddColumnIfMissing est idempotent.
	if err := migration.AddColumnIfMissing(db, "milestone_catalog", "condition_fr", "VARCHAR"); err != nil {
		return fmt.Errorf("add condition_fr: %w", err)
	}
	if err := migration.AddColumnIfMissing(db, "milestone_catalog", "condition_en", "VARCHAR"); err != nil {
		return fmt.Errorf("add condition_en: %w", err)
	}
	now := time.Now().UTC()
	for _, e := range catalogEntries {
		_, err := db.ExecContext(migration.BootCtx(), `
			INSERT INTO milestone_catalog (
				id, title_slug, metric, threshold, title_en, title_fr,
				icon, condition, condition_fr, condition_en, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (id) DO UPDATE SET
				title_slug   = excluded.title_slug,
				metric       = excluded.metric,
				threshold    = excluded.threshold,
				title_en     = excluded.title_en,
				title_fr     = excluded.title_fr,
				icon         = excluded.icon,
				condition    = excluded.condition,
				condition_fr = excluded.condition_fr,
				condition_en = excluded.condition_en,
				updated_at   = excluded.updated_at`,
			e.ID, e.TitleSlug, e.Metric, e.Threshold,
			e.TitleEN, e.TitleFR,
			nullableStringForSeed(e.Icon),
			nullableStringForSeed(e.Condition),
			nullableStringForSeed(e.ConditionFR),
			nullableStringForSeed(e.ConditionEN),
			now,
		)
		if err != nil {
			return fmt.Errorf("upsert %s: %w", e.ID, err)
		}
	}
	return nil
}

// nullableStringForSeed retourne nil si s est vide, sinon s. Stocke NULL en DuckDB
// plutôt que ” pour Icon/Condition (cohérent avec MilestoneCatalogRepo.Upsert).
func nullableStringForSeed(s string) any {
	if s == "" {
		return nil
	}
	return s
}
