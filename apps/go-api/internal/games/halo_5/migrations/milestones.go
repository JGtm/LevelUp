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
// Le seed lit config/titles/halo_5/milestones/catalog.toml (créé en C5). Comme
// MetadataSteps() est statique (pas de repoRoot), le chemin racine est injecté via
// SetMilestonesSeedRoot au boot (cmd/server) et par les CLI qui provisionnent la
// metadata h5. Root vide → le schéma est créé mais le seed no-op (catalogue vide,
// dégradation gracieuse — pas d'échec de migration).

import (
	"database/sql"
	"path/filepath"
	"sync"
	"time"

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
// milestones). Schéma IDENTIQUE à HINF (create_milestone_catalog_metadata) : pas
// d'index secondaire (title_slug/metric mutés par Upsert → un index ART
// FATAL-invaliderait metadata.duckdb).
func milestoneCatalogSchemaStep() migration.Migration {
	return migration.Migration{
		Name:        "h5_create_milestone_catalog",
		TargetDB:    migration.TargetMetadata,
		Description: "Halo 5 — table milestone_catalog (référentiel milestones, même forme que HINF ; seedée depuis config/titles/halo_5/milestones/catalog.toml).",
		ApplySchema: func(db *sql.DB) error {
			return migration.ExecScript(db, `
				CREATE TABLE IF NOT EXISTS milestone_catalog (
					id          VARCHAR PRIMARY KEY,
					title_slug  VARCHAR NOT NULL,
					metric      VARCHAR NOT NULL,
					threshold   DOUBLE NOT NULL,
					title_en    VARCHAR NOT NULL,
					title_fr    VARCHAR NOT NULL,
					icon        VARCHAR,
					condition   VARCHAR,
					updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);
			`)
		},
	}
}

// milestoneCatalogSeedStep seede le catalogue h5 depuis le TOML. ApplyBackfill
// (idempotent, UPSERT sur la PK id) : no-op si la racine n'est pas injectée ou si
// le TOML est absent. Découplé du schéma (ordre canonique : schéma avant seed).
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
				return nil
			}
			now := time.Now().UTC()
			for _, e := range catalogEntries {
				if _, err := db.ExecContext(migration.BootCtx(), `
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
					nullableSeedString(e.Icon),
					nullableSeedString(e.Condition),
					now,
				); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

// nullableSeedString stocke NULL plutôt que ” pour icon/condition (cohérent HINF).
func nullableSeedString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
