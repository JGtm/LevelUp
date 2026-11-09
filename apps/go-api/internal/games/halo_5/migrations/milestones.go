package migrations

// milestones.go — schéma + seed du milestone_catalog PROPRE à Halo 5.
//
// Le set metadata h5 possède ENTIÈREMENT le target metadata (OwnsTarget) → il ne
// retombe PAS sur les steps HINF. Le schéma create_milestone_catalog (inline dans
// HINF steps.go) et le seed global seed_milestone_catalog_v1 (registre global,
// jamais joué pour un set isolé) DOIVENT donc être répliqués ici, sinon la metadata
// h5 n'a pas de table milestone_catalog → la couche Progression V2 (milestones) ne
// peut rien servir pour Halo 5 (milestone_catalog_count=0, aucun milestone gagné).
//
// Le schéma créé ici est celui d'HINF d'avant le passage des textes aux tables par langue : il
// porte encore les colonnes appariées title_en / title_fr / condition_fr / condition_en, que
// metadata_definition_texts_to_translations_v1 copie dans asset_translations (type `milestone`)
// puis retire. Le lecteur unique (duckdb.MilestoneCatalogRepo) ne lit plus ces colonnes.
//
// Le seed lit config/titles/halo_5/milestones/catalog.toml (créé en C5) et vient APRÈS
// metadata_definition_texts_to_translations_v1 dans l'ordre du set : il écrit titre et condition
// lisible par langue dans asset_translations (table créée par h5_add_asset_translations), sur une
// table de jalons sans colonnes appariées. Comme
// MetadataSteps() est statique (pas de repoRoot), le chemin racine est injecté via
// SetMilestonesSeedRoot au boot (cmd/server) et par les CLI qui provisionnent la
// metadata h5. Root vide → le schéma est créé mais le seed no-op (catalogue vide,
// dégradation gracieuse — pas d'échec de migration).

import (
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/progression/milestones"
)

// milestonesSeedRoot = chemin config/titles/ (parent des dossiers par titre).
// Protégé par mutex : posé au boot avant tout RunForTitleDB(halo_5), lu à
// l'exécution de la migration de seed. Vide par défaut → seed no-op.
var (
	milestonesSeedMu   sync.RWMutex
	milestonesSeedRoot string
)

// SetMilestonesSeedRoot injecte la racine config/titles/ utilisée par le seed du
// milestone_catalog h5. À appeler AVANT halo5migrations.Register()+RunForTitleDB
// (boot serveur ; CLI provisionnant la metadata h5). Idempotent.
func SetMilestonesSeedRoot(configTitlesRoot string) {
	milestonesSeedMu.Lock()
	milestonesSeedRoot = configTitlesRoot
	milestonesSeedMu.Unlock()
}

func milestonesSeedRootValue() string {
	milestonesSeedMu.RLock()
	defer milestonesSeedMu.RUnlock()
	return milestonesSeedRoot
}

// milestoneCatalogSchemaStep crée la table milestone_catalog (référentiel des
// milestones). Schéma historique IDENTIQUE à HINF (create_milestone_catalog_metadata), colonnes
// appariées comprises : metadata_definition_texts_to_translations_v1, dernier step du set, les
// copie dans asset_translations puis les retire. Pas d'index secondaire (title_slug/metric
// mutés par Upsert → un index ART FATAL-invaliderait metadata.duckdb).
func milestoneCatalogSchemaStep() migration.Migration {
	return migration.Migration{
		Name:        "h5_create_milestone_catalog",
		TargetDB:    migration.TargetMetadata,
		Description: "Halo 5 — table milestone_catalog (référentiel milestones, forme HINF ; textes migrés ensuite vers asset_translations ; seedée depuis config/titles/halo_5/milestones/catalog.toml).",
		ApplySchema: func(db *sql.DB) error {
			return migration.ExecScript(db, `
				CREATE TABLE IF NOT EXISTS milestone_catalog (
					id           VARCHAR PRIMARY KEY,
					title_slug   VARCHAR NOT NULL,
					metric       VARCHAR NOT NULL,
					threshold    DOUBLE NOT NULL,
					title_en     VARCHAR NOT NULL,
					title_fr     VARCHAR NOT NULL,
					icon         VARCHAR,
					condition    VARCHAR,
					condition_fr VARCHAR,
					condition_en VARCHAR,
					updated_at   TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
				);
			`)
		},
	}
}

// milestoneCatalogConditionLocalesStep ajoute condition_fr / condition_en aux metadata.duckdb h5
// DÉJÀ provisionnées sans elles : `CREATE TABLE IF NOT EXISTS` n'ajoute jamais de colonne à une
// table existante, et un step déjà tracé dans schema_migrations n'est jamais rejoué. Step SÉPARÉ
// (nouveau nom = appliqué au prochain boot) et 100 % idempotent (AddColumnIfMissing), même motif
// que h5_weapon_labels_add_icon / h5_commendation_definitions_add_tier_targets. Historique : le
// step de textes (metadata_definition_texts_to_translations_v1) retire ces colonnes ensuite.
func milestoneCatalogConditionLocalesStep() migration.Migration {
	return migration.Migration{
		Name:     "h5_milestone_catalog_add_condition_locales",
		TargetDB: migration.TargetMetadata,
		Description: "Halo 5 — milestone_catalog : colonnes condition_fr/condition_en " +
			"(ALTER idempotent pour les DB déjà provisionnées sans elles ; historique, retirées " +
			"ensuite par metadata_definition_texts_to_translations_v1).",
		ApplySchema: func(db *sql.DB) error {
			if err := migration.AddColumnIfMissing(db, "milestone_catalog", "condition_fr", "VARCHAR"); err != nil {
				return err
			}
			return migration.AddColumnIfMissing(db, "milestone_catalog", "condition_en", "VARCHAR")
		},
	}
}

// milestoneCatalogSeedStep seede le catalogue h5 depuis le TOML. ApplyBackfill
// (idempotent, SELECT-puis-UPDATE-ou-INSERT sur la clé id) : no-op si la racine n'est pas
// injectée ou si le TOML est absent. Placé APRÈS le step de textes dans l'ordre du set : la table
// n'a plus de colonnes appariées et les textes vont dans asset_translations.
func milestoneCatalogSeedStep() migration.Migration {
	return migration.Migration{
		Name:        "h5_seed_milestone_catalog",
		TargetDB:    migration.TargetMetadata,
		Description: "Halo 5 — seed milestone_catalog depuis config/titles/halo_5/milestones/catalog.toml (idempotent ; no-op si racine non injectée).",
		ApplySchema: func(_ *sql.DB) error { return nil },
		ApplyBackfill: func(db *sql.DB) error {
			root := milestonesSeedRootValue()
			if root == "" {
				return nil // racine non injectée (CLI sans milestones) → no-op gracieux
			}
			path := filepath.Join(root, halo5.TitleSlug, "milestones", "catalog.toml")
			catalogEntries, err := milestones.LoadCatalogFromFile(path)
			if err != nil {
				// TOML absent/illisible → no-op gracieux (la metadata reste valide,
				// milestones simplement non peuplés). On NE casse PAS la migration.
				slog.WarnContext(migration.BootCtx(), "migration h5: catalogue de jalons illisible, seed ignoré",
					"path", path, "err", err)
				return nil
			}
			return seedMilestoneEntries(db, catalogEntries)
		},
	}
}

// seedMilestoneEntries écrit les jalons (ligne + traductions) dans une seule transaction.
func seedMilestoneEntries(db *sql.DB, entries []milestones.CatalogEntry) error {
	ctx := migration.BootCtx()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("seed milestones h5: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // sans effet après Commit
	for _, e := range entries {
		if err := migration.UpsertMilestoneCatalogEntry(ctx, tx, e); err != nil {
			return err
		}
	}
	return tx.Commit()
}
