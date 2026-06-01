package migration

// steps_metadata_drop_playlists_catalog_indexes.go — fix RC-E (2026-06-01).
//
// Root cause prouvée live : seedPlaylistsCatalog (sync/career.go) fait un UPDATE
// sur playlists_catalog modifiant les colonnes `experience` et `is_active`, qui
// sont précisément les colonnes des 2 index ART secondaires
// (idx_playlists_catalog_active, idx_playlists_catalog_experience). Un UPDATE qui
// touche une colonne ART-indexée déclenche le bug DuckDB
// "Failed to delete all rows from index. Only deleted 0 out of N rows" → la
// metadata.duckdb passe en FATAL "database has been invalidated" à chaque cycle de
// sync → cascade : le SharedDBProvider sert shared_matches_v2 en read-only au
// post-sync → toute la complétion combat (events/skill/registry-names) échoue RO.
//
// Le commentaire historique de career.go ("ART-safe UPDATE-then-INSERT, pas
// d'ON CONFLICT") est INSUFFISANT pour cette table : éviter ON CONFLICT n'aide
// pas quand l'UPDATE lui-même modifie une colonne indexée par ART.
//
// Fix : supprimer les 2 index secondaires. À 34 lignes (1 titre), un scan complet
// de playlists_catalog est instantané ; ces index n'apportent aucun gain mesurable
// et sont la SEULE surface de corruption ART de cette table (la PK
// (title_slug, playlist_asset_id) n'est pas touchée par les UPDATE de seed, donc
// son index reste sain). Idempotent (DROP INDEX IF EXISTS).
//
// NB : la migration d'origine (add_catalog_playlists) crée encore ces index avec
// CREATE INDEX IF NOT EXISTS. Sur une DB neuve ils seront recréés puis droppés par
// CETTE migration (ordre garanti : registration plus tardive). Sur les DB
// existantes, ce DROP les retire définitivement.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "drop_playlists_catalog_secondary_indexes",
		TargetDB:    TargetMetadata,
		Description: "Supprime idx_playlists_catalog_active/_experience (source RC-E : UPDATE sur colonne ART-indexée corrompt metadata.duckdb)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				DROP INDEX IF EXISTS idx_playlists_catalog_active;
				DROP INDEX IF EXISTS idx_playlists_catalog_experience;
			`)
		},
	})
}
