package migration

// steps_metadata_rebuild_catalog_fetch_queue_no_art.go — fix RC-E catalog drain
// (2026-06-19).
//
// Root cause prouvée live (prod, logs/catalog) : le cron catalog_refresh
// (scheduler/catalog_refresh_cron.go, 1er tick à CHAQUE boot) draine
// catalog_fetch_queue ; pour chaque entrée traitée, CatalogFetcherService fait
// `DELETE FROM catalog_fetch_queue WHERE ...` (deleteFromQueue) et, sur erreur,
// `UPDATE ... SET attempts = attempts + 1` (markError). Or cette table portait :
//   - une PRIMARY KEY (title_slug, asset_type, asset_id) → index ART touché par
//     le DELETE de chaque ligne,
//   - un index secondaire idx_catalog_fetch_queue_drain incluant la colonne
//     `attempts` → index ART touché par l'UPDATE de markError.
// Le DELETE/UPDATE per-row sur une table ART-indexée déclenche le bug DuckDB
// 1.5.x #23046 ("Failed to delete all rows from index. Only deleted 0 out of N
// rows") → metadata.duckdb passe FATAL "database has been invalidated" pour tout
// le reste de la vie du process → résolution des noms d'assets (maps/playlists/
// modes FR) cassée jusqu'au prochain restart (qui re-casse au boot suivant).
//
// Le commentaire de cmd/server/main.go ("drain ART-safe via upsertRowNoConflict")
// était INSUFFISANT : l'UPSERT vers les tables catalogue est bien SELECT-then-write,
// mais la queue elle-même subissait DELETE + UPDATE sur index ART.
//
// Fix : retirer TOUTE surface ART de catalog_fetch_queue (drop index secondaire +
// rebuild sans PRIMARY KEY). La queue est minuscule (dizaines d'entrées, drain
// hebdomadaire) → un scan complet est instantané, l'index/PK n'apportaient aucun
// gain. La dédup d'enqueue passe désormais par un SELECT-then-INSERT (NOT EXISTS)
// côté code (catalog_queue.go + catalog_enqueue.go), plus par INSERT OR IGNORE.
//
// DuckDB ne sait pas DROP une PRIMARY KEY via ALTER → rebuild CTAS-swap (même
// pattern que rebuild_match_participants_defeat_art_corruption / _career_*). Le
// DROP TABLE de l'ancienne (avec son index ART) est sûr : il déalloue la structure
// entière, ce n'est pas un delete d'index per-row.
//
// Idempotent : sur une DB déjà sans PK, le rebuild recrée une table identique
// (DISTINCT préserve les lignes). Tracké par schema_migrations (1 seule exécution).

import "database/sql"

func init() {
	Register(Migration{
		Name:        "rebuild_catalog_fetch_queue_drop_art_indexes",
		TargetDB:    TargetMetadata,
		Description: "Retire PK + index secondaire de catalog_fetch_queue (DELETE/UPDATE per-row sur index ART corrompt metadata.duckdb — RC-E catalog drain)",
		ApplySchema: func(db *sql.DB) error {
			// Garde d'existence : catalog_fetch_queue est créée par le créateur de
			// schéma metadata, désormais TITLE-OWNED (relocalisé voie B dans
			// games/halo_infinite/migrations). Ce rebuild reste dans le registre
			// GLOBAL (corrige les DB existantes portant l'ancienne PK/index ART). En
			// prod, le provider title-owned est câblé → la table existe quand ce step
			// tourne. Dans le binaire de test de internal/migration, le provider est
			// nil (cycle d'import) → la table n'est pas créée → on skippe proprement
			// (rien à rebuild). Sans cette garde, le INSERT...SELECT / DROP échoue.
			has, err := tableExists(db, "catalog_fetch_queue")
			if err != nil {
				return err
			}
			if !has {
				return nil
			}
			return execScript(db, `
				-- 1. Index secondaire (colonne attempts mutée par markError → surface ART).
				DROP INDEX IF EXISTS idx_catalog_fetch_queue_drain;

				-- 2. Rebuild sans PRIMARY KEY (DELETE per-row de deleteFromQueue → surface ART).
				DROP TABLE IF EXISTS catalog_fetch_queue_rebuild;
				CREATE TABLE catalog_fetch_queue_rebuild (
					title_slug   VARCHAR NOT NULL,
					asset_type   VARCHAR NOT NULL,
					asset_id     VARCHAR NOT NULL,
					version_id   VARCHAR,
					enqueued_at  TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
					attempts     INTEGER DEFAULT 0,
					last_error   VARCHAR
				);
				INSERT INTO catalog_fetch_queue_rebuild
					SELECT DISTINCT title_slug, asset_type, asset_id, version_id,
					                enqueued_at, attempts, last_error
					FROM catalog_fetch_queue;
				DROP TABLE catalog_fetch_queue;
				ALTER TABLE catalog_fetch_queue_rebuild RENAME TO catalog_fetch_queue;
			`)
		},
	})
}
