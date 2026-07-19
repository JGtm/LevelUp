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
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

// metadataStepNames retourne les noms des steps metadata h5, dans l'ordre
// d'exécution voulu. = CanonicalOrder du set (le set ne possède QUE metadata via
// OwnsTarget ; les autres targets passent par le fallback HINF complet).
func metadataStepNames() []string {
	return []string{
		"h5_add_xbox_achievement_definitions",
		"h5_create_milestone_catalog",
		"h5_seed_milestone_catalog",
		"h5_add_asset_translations",
		"h5_add_medal_definitions",
		"h5_add_weapon_labels",
		"h5_add_maps_catalog",
		"h5_add_map_images_registry",
		"h5_add_csr_designations",
		"h5_weapon_labels_add_icon",
		"h5_add_commendation_definitions",
		"h5_commendation_definitions_add_tier_targets",
		"h5_add_weapon_registry",
		"h5_add_team_colors",
	}
}

// MetadataSteps retourne le schéma référentiel metadata PROPRE à Halo 5 (CREATE
// seul, idempotent, zéro seed HINF). Formes alignées sur HINF pour réutiliser les
// helpers de lecture et le drain de catalogue à l'identique.
func MetadataSteps() []migration.Migration {
	return []migration.Migration{
		{
			Name:        "h5_add_xbox_achievement_definitions",
			TargetDB:    migration.TargetMetadata,
			Description: "Halo 5 — xbox_achievement_definitions (référentiel succès Xbox h5, bilingue EN/FR, title_id=219630713 dès la création ; vide → cmd/levelup sync-achievements --title halo_5). Même forme que le référentiel HINF (brique metadata commune).",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS xbox_achievement_definitions (
						achievement_id   VARCHAR PRIMARY KEY,
						name_en          VARCHAR NOT NULL DEFAULT '',
						name_fr          VARCHAR NOT NULL DEFAULT '',
						description_en   VARCHAR,
						description_fr   VARCHAR,
						locked_desc_en   VARCHAR,
						locked_desc_fr   VARCHAR,
						gamerscore       INTEGER NOT NULL DEFAULT 0,
						image_url        VARCHAR,
						is_secret        BOOLEAN NOT NULL DEFAULT FALSE,
						rarity_category  VARCHAR,
						rarity_percent   FLOAT,
						title_id         VARCHAR DEFAULT '',
						xbox_title_id    VARCHAR DEFAULT '',
						service_config_id VARCHAR DEFAULT '',
						fetched_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					);
				`)
			},
		},
		// Milestones (Progression V2) — schéma + seed PROPRES à h5 (le set isolé ne
		// retombe pas sur le create_milestone_catalog inline HINF ni le seed global).
		milestoneCatalogSchemaStep(),
		milestoneCatalogSeedStep(),
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
						personal_score   INTEGER DEFAULT 0,
						-- Icône h5 = sprite (feuille + offset), pas une PNG par médaille
						-- (API Metadata officielle Halo 5 : spriteLocation).
						sprite_sheet_url VARCHAR,
						sprite_left      INTEGER,
						sprite_top       INTEGER,
						sprite_width     INTEGER,
						sprite_height    INTEGER
					);
					-- Robustesse : si la table préexistait (provisioning antérieur sans
					-- les colonnes sprite), les ajoute (idempotent).
					ALTER TABLE medal_definitions ADD COLUMN IF NOT EXISTS sprite_sheet_url VARCHAR;
					ALTER TABLE medal_definitions ADD COLUMN IF NOT EXISTS sprite_left INTEGER;
					ALTER TABLE medal_definitions ADD COLUMN IF NOT EXISTS sprite_top INTEGER;
					ALTER TABLE medal_definitions ADD COLUMN IF NOT EXISTS sprite_width INTEGER;
					ALTER TABLE medal_definitions ADD COLUMN IF NOT EXISTS sprite_height INTEGER;
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
				`)
			},
		},
		{
			Name:        "h5_add_csr_designations",
			TargetDB:    migration.TargetMetadata,
			Description: "Halo 5 — csr_designations (palier CSR + sous-palier → image d'insigne ; API Metadata officielle, vide → fetcher h5-metadata-fetch)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS csr_designations (
						designation_name VARCHAR NOT NULL,
						tier_id          INTEGER NOT NULL,
						icon_url         VARCHAR,
						banner_url       VARCHAR,
						PRIMARY KEY (designation_name, tier_id)
					);
				`)
			},
		},
		{
			Name:        "h5_weapon_labels_add_icon",
			TargetDB:    migration.TargetMetadata,
			Description: "Halo 5 — weapon_labels : colonnes icon_url + weapon_type (API Metadata officielle largeIconImageUrl/type). Step séparé pour s'appliquer aux DB déjà provisionnées.",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE weapon_labels ADD COLUMN IF NOT EXISTS icon_url VARCHAR;
					ALTER TABLE weapon_labels ADD COLUMN IF NOT EXISTS weapon_type VARCHAR;
				`)
			},
		},
		{
			Name:        "h5_add_commendation_definitions",
			TargetDB:    migration.TargetMetadata,
			Description: "Halo 5 — commendation_definitions (référentiel commendations NATIVES par UUID : nom/description/icône CDN + type + catégorie ; API Metadata officielle /commendations, vide → fetcher h5-metadata-fetch). Clé = UUID de la commendation = ProgressiveCommendationDeltas[].Id du carnage.",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS commendation_definitions (
						commendation_id   VARCHAR PRIMARY KEY,
						name_en           VARCHAR NOT NULL,
						name_fr           VARCHAR NOT NULL,
						description_en    VARCHAR DEFAULT '',
						description_fr    VARCHAR DEFAULT '',
						commendation_type VARCHAR,
						category          VARCHAR,
						icon_url          VARCHAR,
						-- tier_targets : CSV croissant des seuils de paliers (levels[].threshold
						-- de l'API Metadata). Format IDENTIQUE à citation_mappings.tier_targets
						-- d'Infinite → réutilise analysis.ParseTierTargets/ComputeTierProgression
						-- pour le progrès + masterisé des commendations natives Halo 5.
						tier_targets      VARCHAR
					);
				`)
			},
		},
		{
			Name:        "h5_commendation_definitions_add_tier_targets",
			TargetDB:    migration.TargetMetadata,
			Description: "Halo 5 — commendation_definitions : colonne tier_targets (CSV croissant des seuils levels[].threshold). Step séparé pour s'appliquer aux DB déjà provisionnées (ALTER idempotent). Alimente le progrès/tier/masterisé des commendations natives sur les tuiles de match.",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE commendation_definitions ADD COLUMN IF NOT EXISTS tier_targets VARCHAR;
				`)
			},
		},
		// Registre d'armes CANONIQUE (weapons/weapon_ids/weapon_families). CONTRAIRE-
		// MENT aux référentiels title-specific ci-dessus (career_ranks, playlists,
		// weapon_labels… isolés car les données HINF sont FAUSSES pour H5), le registre
		// est CROSS-TITRE par conception : PK (title_slug, weapon_key), lectures
		// title-scopées (resolveWeaponMeta filtre wi.title_slug). Son seed inclut déjà
		// les 35 armes Halo 5 + leurs stock_ids (weaponRegistryH5Stock). Sans lui, la
		// metadata H5 n'a pas de rôles → resolveWeaponMeta retombe sur weapon_labels
		// seul → aucun rôle/classe d'arme dans la FragDistribution (sunburst réduit aux
		// classes API + résidu, breakdown par arme non recoloré) sur H5.
		// On réutilise l'apply idempotent partagé (INSERT OR IGNORE) plutôt que de
		// dupliquer le seed (source unique : halo_infinite/migrations, cross-titre).
		{
			Name:        "h5_add_weapon_registry",
			TargetDB:    migration.TargetMetadata,
			Description: "Halo 5 — registre d'armes canonique (weapons/weapon_ids/weapon_families, référentiel CROSS-TITRE seedé pour tous les titres). Débloque les rôles de combat H5 (donut « Frags par type d'arme ») via resolveWeaponMeta.",
			ApplySchema: func(db *sql.DB) error {
				return halomigrations.ApplyWeaponRegistry(db)
			},
		},
		{
			Name:        "h5_add_team_colors",
			TargetDB:    migration.TargetMetadata,
			Description: "Halo 5 — team_colors (TeamId → couleur #RRGGBB + nom localisé EN/FR + icône ; API Metadata officielle /team-colors, vide → fetcher h5-metadata-fetch). Alimente les libellés d'équipe « Rouge/Bleu » de la Match View H5 (fallback sur l'existant si vide).",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS team_colors (
						team_id  INTEGER PRIMARY KEY,
						name_en  VARCHAR NOT NULL DEFAULT '',
						name_fr  VARCHAR NOT NULL DEFAULT '',
						color    VARCHAR,
						icon_url VARCHAR
					);
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
