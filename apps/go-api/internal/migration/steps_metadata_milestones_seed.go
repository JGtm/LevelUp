package migration

// steps_metadata_milestones_seed.go — seed du catalogue milestones depuis TOML.
//
// Contexte : `milestone_catalog` est créée par la migration
// `create_milestone_catalog_metadata` (TargetMetadata) mais reste VIDE. Le
// helper `milestones.SyncCatalog` existe mais n'est jamais appelé
// applicativement (cf. AUDIT_ASCENSION_PIPELINE_DISCONNECTED_2026-05-21 §4
// cause A). Résultat : endpoint /milestones renvoie items=[] pour tous les
// joueurs, page Ascension affiche "Aucun milestone configuré".
//
// Pattern aligné sur `steps_metadata_prestige_seed.go` (RegisterPrestigeSeed
// Migration). Idempotent via `INSERT ... ON CONFLICT DO UPDATE`. Multi-titres :
// itère sur tous les `config/titles/{slug}/milestones/catalog.toml` présents.

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"levelup/go-api/internal/progression/milestones"
)

const milestonesSeedMigrationName = "seed_milestone_catalog_v1"

// RegisterMilestonesSeedMigration enregistre la migration de seed du catalogue
// milestones.
//
// configTitlesRoot : chemin vers `config/titles/` (parent des dossiers par
// titre). La migration itère sur chaque sous-dossier et applique le seed si
// `<slug>/milestones/catalog.toml` existe.
//
// Idempotent : sans effet si la migration est déjà enregistrée (call depuis
// cmd/server/main.go peut être répété).
func RegisterMilestonesSeedMigration(configTitlesRoot string) {
	for _, m := range registry {
		if m.Name == milestonesSeedMigrationName {
			return
		}
	}
	Register(Migration{
		Name:        milestonesSeedMigrationName,
		TargetDB:    TargetMetadata,
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
			// Titre sans catalogue milestones (ex: titres futurs sans cette
			// feature) → skip silencieux.
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
	now := time.Now().UTC()
	for _, e := range catalogEntries {
		_, err := db.ExecContext(bootCtx(), `
			INSERT INTO milestone_catalog (
				id, title_slug, metric, threshold, title_en, title_fr,
				icon, condition, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (id) DO UPDATE SET
				title_slug = excluded.title_slug,
				metric     = excluded.metric,
				threshold  = excluded.threshold,
				title_en   = excluded.title_en,
				title_fr   = excluded.title_fr,
				icon       = excluded.icon,
				condition  = excluded.condition,
				updated_at = excluded.updated_at`,
			e.ID, e.TitleSlug, e.Metric, e.Threshold,
			e.TitleEN, e.TitleFR,
			nullableStringForSeed(e.Icon),
			nullableStringForSeed(e.Condition),
			now,
		)
		if err != nil {
			return fmt.Errorf("upsert %s: %w", e.ID, err)
		}
	}
	return nil
}

// nullableStringForSeed retourne nil si s est vide, sinon &s. Permet de stocker
// NULL en DuckDB plutôt que ” pour Icon/Condition (cohérent avec
// MilestoneCatalogRepo.Upsert).
func nullableStringForSeed(s string) any {
	if s == "" {
		return nil
	}
	return s
}
