// Package migrations — jeu de migrations PROPRE à Halo 5 (ROOT FIX assets, PMT-9).
//
// PROBLÈME RÉSOLU. Sans set enregistré, RunForTitleDB(halo_5, …) retombe sur le
// chemin legacy = le registre global d'Halo Infinite pour TOUS les targets. Pour
// `shared`/`player`/`sharedsocial`/`pve` c'est VOULU (uniformité inter-titres :
// match_registry, medals_earned, player_match_enrichment… identiques). Mais pour
// `metadata` cela injecte les RÉFÉRENTIELS d'Infinite dans la metadata.duckdb
// d'Halo 5 : career_rank_translations (échelle 272 rangs HINF ≠ SR 1-152 H5),
// playlists_catalog seedé HINF, citation_mappings HINF, weapon_labels HINF (IDs
// d'armes différents), prestige/battlepass/xbox_achievements HINF. h5 étant
// `status="active"`, cette pollution se produit AU BOOT (provisionAdditionalTitle).
//
// SOLUTION. On enregistre un TitleMigrationSet pour halo_5 qui :
//   - target `metadata` → SES PROPRES tables référentielles (schéma seul, AUCUN
//     seed HINF) → metadata.duckdb h5 vierge, prête pour les fetchers CMS Halo 5 ;
//   - tout autre target → délègue à halo_infinite/migrations.StepsFor(target) →
//     hérite du schéma uniforme (le set est all-or-nothing par target : déléguer
//     est la façon canonique de conserver l'héritage là où on le veut).
//
// Les tables sont créées VIDES : les noms/images réels (médailles, maps, armes)
// sont peuplés par le sous-projet « fetchers CMS Halo 5 » (halo5api.svc /
// content-hacs / ugc). Créer le schéma maintenant ISOLE la donnée h5 de HINF et
// donne aux fetchers une cible propre — ce n'est pas du schéma mort mais la
// fondation référentielle (même forme que HINF → les helpers de lecture et le
// pattern catalog-drain marchent à l'identique).
package migrations

import (
	"database/sql"

	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/migration"
)

// metadataStepNames retourne les noms des steps metadata h5, dans l'ordre
// d'exécution voulu. = CanonicalOrder du set (le set ne possède QUE metadata via
// OwnsTarget ; les autres targets passent par le fallback HINF complet).
func metadataStepNames() []string {
	return []string{
		"h5_add_asset_translations",
		"h5_add_medal_definitions",
		"h5_add_weapon_labels",
		"h5_add_maps_catalog",
		"h5_add_map_images_registry",
	}
}

// MetadataSteps retourne le schéma référentiel metadata PROPRE à Halo 5 (CREATE
// seul, idempotent, zéro seed HINF). Formes alignées sur HINF pour réutiliser les
// helpers de lecture et le drain de catalogue à l'identique.
func MetadataSteps() []migration.Migration {
	return []migration.Migration{
		{
			Name:        "h5_add_asset_translations",
			TargetDB:    migration.TargetMetadata,
			Description: "Halo 5 — asset_translations + medal_translations (pivot multi-langue, vide → fetchers CMS h5)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS asset_translations (
						asset_id    VARCHAR NOT NULL,
						asset_type  VARCHAR NOT NULL,
						lang        VARCHAR NOT NULL,
						name        VARCHAR NOT NULL,
						description VARCHAR,
						fetched_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (asset_id, asset_type, lang)
					);
					CREATE INDEX IF NOT EXISTS idx_asset_tr_id_type ON asset_translations(asset_id, asset_type);
					CREATE TABLE IF NOT EXISTS medal_translations (
						medal_name_id BIGINT NOT NULL,
						lang          VARCHAR NOT NULL,
						name          VARCHAR NOT NULL,
						description   VARCHAR,
						PRIMARY KEY (medal_name_id, lang)
					);
				`)
			},
		},
		{
			Name:        "h5_add_medal_definitions",
			TargetDB:    migration.TargetMetadata,
			Description: "Halo 5 — medal_definitions (référentiel médailles h5 par medal_name_id, vide → fetcher content-hacs h5)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS medal_definitions (
						medal_name_id    BIGINT PRIMARY KEY,
						name_fr          VARCHAR NOT NULL,
						name_en          VARCHAR NOT NULL,
						description_fr   VARCHAR DEFAULT '',
						description_en   VARCHAR DEFAULT '',
						is_custom        BOOLEAN DEFAULT FALSE,
						difficulty_index TINYINT DEFAULT 0,
						type_index       TINYINT DEFAULT 0,
						difficulty       VARCHAR,
						medal_type       VARCHAR,
						personal_score   INTEGER DEFAULT 0
					);
				`)
			},
		},
		{
			Name:        "h5_add_weapon_labels",
			TargetDB:    migration.TargetMetadata,
			Description: "Halo 5 — weapon_labels PROPRE (IDs d'armes h5 ≠ HINF, vide → fetcher halo5api.svc/weapons)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS weapon_labels (
						weapon_id UBIGINT PRIMARY KEY,
						name_en   VARCHAR NOT NULL,
						name_fr   VARCHAR NOT NULL
					);
				`)
			},
		},
		{
			Name:        "h5_add_maps_catalog",
			TargetDB:    migration.TargetMetadata,
			Description: "Halo 5 — maps_catalog (nom + image_url des maps h5, title_slug-keyé, vide → fetcher ugc.svc)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS maps_catalog (
						title_slug          VARCHAR NOT NULL,
						map_asset_id        VARCHAR NOT NULL,
						current_version_id  VARCHAR,
						name_canonical      VARCHAR,
						image_url           VARCHAR,
						last_fetched_at     TIMESTAMP,
						PRIMARY KEY (title_slug, map_asset_id)
					);
				`)
			},
		},
		{
			Name:        "h5_add_map_images_registry",
			TargetDB:    migration.TargetMetadata,
			Description: "Halo 5 — map_images_registry (cache-aside images de maps avec local_path, vide → fetcher d'images)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS map_images_registry (
						title_id     VARCHAR NOT NULL,
						map_id       VARCHAR NOT NULL,
						image_url    VARCHAR NOT NULL DEFAULT '',
						local_path   VARCHAR NOT NULL DEFAULT '',
						fetched_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
						content_hash VARCHAR NOT NULL DEFAULT '',
						PRIMARY KEY (title_id, map_id)
					);
					CREATE INDEX IF NOT EXISTS idx_map_images_registry_fetched ON map_images_registry(fetched_at);
				`)
			},
		},
	}
}

// Set construit le TitleMigrationSet d'Halo 5 : isolation PARTIELLE via OwnsTarget.
// Le set ne possède QUE le target `metadata` (schéma référentiel h5 propre) ; tout
// autre target (shared/player/sharedsocial/pve) retombe sur le fallback HINF
// COMPLET (registre global + titleStepsProvider) → uniformité inter-titres
// préservée. CanonicalOrder ne couvre donc que les steps metadata h5 (seuls
// exécutés pour le target possédé).
func Set() migration.TitleMigrationSet {
	return migration.TitleMigrationSet{
		Slug:           halo5.TitleSlug,
		CanonicalOrder: metadataStepNames(),
		OwnsTarget: func(target migration.TargetDB) bool {
			return target == migration.TargetMetadata
		},
		Steps: func(target migration.TargetDB) []migration.Migration {
			if target == migration.TargetMetadata {
				return MetadataSteps()
			}
			return nil // jamais atteint (OwnsTarget filtre) — défensif.
		},
	}
}

// Register enregistre le set Halo 5 auprès du runner de migrations. À appeler au
// boot (et dans les CLI provisionnant une DB h5) AVANT tout RunForTitleDB(halo_5).
// Idempotent (RegisterMigrationSet : dernier gagne).
func Register() {
	migration.RegisterMigrationSet(Set())
}
