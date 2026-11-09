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

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/progression/milestones"
)

// v2 (A9) : bump depuis v1 pour RE-SEEDER les DBs existantes avec les descriptions lisibles
// localisées des jalons (aujourd'hui rangées dans asset_translations, type `milestone`).
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
// milestones.LoadCatalogFromFile pour parser le TOML (validation incluse) puis écrit chaque jalon
// (ligne de milestone_catalog + titre et condition lisible par langue dans asset_translations)
// par migration.UpsertMilestoneCatalogEntry, dans une seule transaction.
//
// Ce seed tourne APRÈS metadata_definition_texts_to_translations_v1 (il n'est pas dans
// canonicalOrder : les seeds TOML dynamiques passent en dernier), donc sur une table sans colonnes
// `title_en` / `title_fr` / `condition_*` : les textes ne s'écrivent que par asset_translations.
func seedMilestonesFromTOML(db *sql.DB, path string) error {
	catalogEntries, err := milestones.LoadCatalogFromFile(path)
	if err != nil {
		return fmt.Errorf("load %s: %w", path, err)
	}
	ctx := migration.BootCtx()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // sans effet après Commit
	for _, e := range catalogEntries {
		if err := migration.UpsertMilestoneCatalogEntry(ctx, tx, e); err != nil {
			return fmt.Errorf("upsert %s: %w", e.ID, err)
		}
	}
	return tx.Commit()
}
