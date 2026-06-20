package migration

// steps_metadata_catalog.go — Phase A du plan PLAN_PLAYLISTS_CATALOG.md.
//
// Crée les 8 tables du catalogue global Playlists / Pairs / Maps dans metadata.duckdb.
// Toutes les tables sont title-aware via title_slug en PK composite.
// Cohérent avec la convention VARCHAR pour les asset IDs (cf. match_registry, waypoint_assets_raw).

import "database/sql"

func init() {
	Register(Migration{
		Name:        "add_catalog_playlists",
		TargetDB:    TargetMetadata,
		Description: "Catalogue global Playlists/Pairs/Maps (8 tables title-aware) — Phase A plan catalogue",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				-- 1. Référentiel des playlists, par titre
				CREATE TABLE IF NOT EXISTS playlists_catalog (
					title_slug          VARCHAR NOT NULL,
					playlist_asset_id   VARCHAR NOT NULL,
					current_version_id  VARCHAR,
					name_canonical      VARCHAR,
					experience          VARCHAR,
					is_ranked           BOOLEAN DEFAULT FALSE,
					is_active           BOOLEAN DEFAULT TRUE,
					first_seen_at       TIMESTAMP,
					last_seen_at        TIMESTAMP,
					last_fetched_at     TIMESTAMP,
					PRIMARY KEY (title_slug, playlist_asset_id)
				);

				-- 2. Référentiel des maps. NB (fix RC-E 2026-06-01) : aucun index
				-- secondaire sur playlists_catalog ci-dessus. seedPlaylistsCatalog UPDATE
				-- ses colonnes experience/is_active a chaque cycle, et un UPDATE sur une
				-- colonne ART-indexee corrompt l index DuckDB puis FATAL-invalide
				-- metadata.duckdb (cascade shared RO). A 34 lignes un scan est instantane.
				-- Cf. steps_metadata_drop_playlists_catalog_indexes.go (drop sur DB existantes).
				CREATE TABLE IF NOT EXISTS maps_catalog (
					title_slug          VARCHAR NOT NULL,
					map_asset_id        VARCHAR NOT NULL,
					current_version_id  VARCHAR,
					name_canonical      VARCHAR,
					image_url           VARCHAR,
					last_fetched_at     TIMESTAMP,
					PRIMARY KEY (title_slug, map_asset_id)
				);

				-- 3. Référentiel des game variants (un variant peut être dans N pairs)
				CREATE TABLE IF NOT EXISTS game_variants_catalog (
					title_slug              VARCHAR NOT NULL,
					game_variant_asset_id   VARCHAR NOT NULL,
					current_version_id      VARCHAR,
					name_canonical          VARCHAR,
					mode_canonical          VARCHAR,
					game_variant_category   INTEGER,
					last_fetched_at         TIMESTAMP,
					PRIMARY KEY (title_slug, game_variant_asset_id)
				);
				-- AUCUN index secondaire : upsertGameVariant UPDATE mode_canonical →
				-- un index ART sur une colonne mutée corrompt metadata.duckdb (FATAL
				-- invalidated). PK-only. Cf. drop_metadata_art_surface_indexes_v1.

				-- 4. Pair = jonction map + game_variant + nom composite, par titre
				CREATE TABLE IF NOT EXISTS map_mode_pair_definitions (
					title_slug             VARCHAR NOT NULL,
					pair_asset_id          VARCHAR NOT NULL,
					current_version_id     VARCHAR,
					name_canonical         VARCHAR,
					map_asset_id           VARCHAR,
					game_variant_asset_id  VARCHAR,
					mode_category          VARCHAR,
					last_fetched_at        TIMESTAMP,
					PRIMARY KEY (title_slug, pair_asset_id)
				);
				-- AUCUN index secondaire : upsertPair UPDATE map_asset_id /
				-- game_variant_asset_id / mode_category (ex-colonnes indexées) →
				-- surface ART corruptrice. PK-only. Cf. drop_metadata_art_surface_indexes_v1.

				-- 5. Relation N-N playlist <-> pair, avec poids de tirage
				CREATE TABLE IF NOT EXISTS playlist_pair_links (
					title_slug         VARCHAR NOT NULL,
					playlist_asset_id  VARCHAR NOT NULL,
					pair_asset_id      VARCHAR NOT NULL,
					weight             DOUBLE,
					PRIMARY KEY (title_slug, playlist_asset_id, pair_asset_id)
				);
				CREATE INDEX IF NOT EXISTS idx_playlist_pair_links_pair ON playlist_pair_links(title_slug, pair_asset_id);

				-- 6. File d'attente du fetcher (pattern Kinds, drain mensuel)
				CREATE TABLE IF NOT EXISTS catalog_fetch_queue (
					title_slug    VARCHAR NOT NULL,
					asset_type    VARCHAR NOT NULL,
					asset_id      VARCHAR NOT NULL,
					version_id    VARCHAR,
					enqueued_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					attempts      INTEGER DEFAULT 0,
					last_error    VARCHAR,
					PRIMARY KEY (title_slug, asset_type, asset_id)
				);
				-- PAS d'index : catalog_fetch_queue est DELETE/UPDATE per-row par le drain
				-- → toute surface ART la corrompt. Rebuild sans PK/index par
				-- rebuild_catalog_fetch_queue_drop_art_indexes. PK-less + dédup SELECT-then-INSERT.

				-- 7. Labels normalisés multi-langues (sortie NormalizeModeLabel par langue)
				CREATE TABLE IF NOT EXISTS pair_mode_label_translations (
					title_slug      VARCHAR NOT NULL,
					pair_asset_id   VARCHAR NOT NULL,
					lang            VARCHAR NOT NULL,
					label           VARCHAR,
					PRIMARY KEY (title_slug, pair_asset_id, lang)
				);
				CREATE INDEX IF NOT EXISTS idx_pair_mode_label_lang ON pair_mode_label_translations(title_slug, lang);

				-- 8. Auto-détection des préfixes inconnus (alerting sur nouvelles catégories candidates)
				CREATE TABLE IF NOT EXISTS unknown_prefix_candidates (
					title_slug    VARCHAR NOT NULL,
					prefix        VARCHAR NOT NULL,
					n_matches     INTEGER DEFAULT 1,
					first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					last_seen_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					pair_examples VARCHAR[],
					PRIMARY KEY (title_slug, prefix)
				);
				CREATE INDEX IF NOT EXISTS idx_unknown_prefix_n ON unknown_prefix_candidates(title_slug, n_matches DESC);
			`)
		},
	})
}
