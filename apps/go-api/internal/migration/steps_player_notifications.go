package migration

// steps_player_notifications.go — migrations pour le système de notifications in-app.
//
// Décision architecturale (2026-04-26) : les notifications NE SONT PAS des
// stats de match — elles vivent dans `shared_social.duckdb` (cohérent avec
// media_likes / match_favorites), avec `xuid` en colonne pour scoper par joueur.
//
// Ce fichier porte donc 2 ensembles de migrations :
//   - TargetSharedSocial : crée les 3 tables (player_notifications,
//     notification_preferences, player_records) avec PK composite incluant xuid.
//   - TargetPlayer       : DROP les 3 anciennes tables qui avaient été
//     créées par erreur dans stats.duckdb.
//
// Les anciennes migrations `create_player_records` et `create_notifications_tables`
// (target=TargetPlayer) sont conservées dans schema_migrations comme appliquées
// pour les DBs déjà passées par cette version intermédiaire ; un nouvel entry
// `drop_notifications_from_player` se charge du nettoyage.

import (
	"database/sql"
)

func init() {
	// ─── TargetSharedSocial : nouvelles tables avec xuid scoping ────────

	Register(Migration{
		Name:        "create_notifications_in_shared_social",
		TargetDB:    TargetSharedSocial,
		Description: "Tables player_notifications + notification_preferences + player_records dans shared_social.duckdb (multi-joueur, xuid PK).",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS player_notifications (
					xuid          VARCHAR NOT NULL,
					id            BIGINT NOT NULL,
					category      VARCHAR NOT NULL,
					severity      VARCHAR NOT NULL DEFAULT 'info',
					title_key     VARCHAR NOT NULL,
					body_key      VARCHAR,
					params        VARCHAR,
					target_route  VARCHAR,
					target_search VARCHAR,
					actor_xuid    VARCHAR,
					actor_name    VARCHAR,
					source        VARCHAR NOT NULL,
					created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					read_at       TIMESTAMP,
					PRIMARY KEY (xuid, id)
				);
				-- PAS d'index sur read_at : MarkNotifications(Read|Unread|AllRead) UPDATE
				-- read_at → un index ART sur cette colonne mutée corrompt shared_social
				-- (bug DuckDB "Failed to delete all rows from index"). Drop historique
				-- drop_idx_pn_xuid_unread + drop_pn_unread_art_index_v2. Garde-fou cross-DB.
				CREATE INDEX IF NOT EXISTS idx_pn_xuid_created_desc ON player_notifications(xuid, created_at DESC);
				CREATE INDEX IF NOT EXISTS idx_pn_xuid_category     ON player_notifications(xuid, category);

				CREATE TABLE IF NOT EXISTS notification_preferences (
					xuid       VARCHAR NOT NULL,
					category   VARCHAR NOT NULL,
					enabled    BOOLEAN NOT NULL DEFAULT TRUE,
					delivery   VARCHAR NOT NULL DEFAULT 'both',
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					PRIMARY KEY (xuid, category)
				);

				CREATE TABLE IF NOT EXISTS player_records (
					xuid              VARCHAR NOT NULL,
					metric            VARCHAR NOT NULL,
					value             DOUBLE NOT NULL,
					achieved_at       TIMESTAMP,
					achieved_match_id VARCHAR,
					updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					PRIMARY KEY (xuid, metric)
				);
			`)
		},
	})

	// ─── TargetPlayer : nettoyage des anciennes tables ─────────────────

	Register(Migration{
		Name:        "drop_notifications_from_player_db",
		TargetDB:    TargetPlayer,
		Description: "Supprime player_notifications, notification_preferences, player_records de stats.duckdb (déplacés dans shared_social).",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				DROP TABLE IF EXISTS player_notifications;
				DROP TABLE IF EXISTS notification_preferences;
				DROP TABLE IF EXISTS player_records;
			`)
		},
	})

	// 2026-05-15 : drop idx_pn_xuid_unread.
	//
	// L'index ART secondaire sur (xuid, read_at) déclenche un bug DuckDB
	// connu lorsque read_at est NULL : tout UPDATE/DELETE sur une notif
	// non-lue lève « Failed to delete all rows from index. Only deleted 0
	// out of 1 rows. » et invalide définitivement la connexion (la base
	// reste inutilisable jusqu'au restart du process).
	//
	// L'index n'apporte aucun gain mesurable : la PK (xuid, id) sélectionne
	// déjà toutes les notifs d'un joueur (≤ DefaultRetentionCap), et le
	// filtrage `read_at IS NULL` se fait en moins d'une ms sur ce volume.
	//
	// Les 2 autres index restent en place (idx_pn_xuid_created_desc et
	// idx_pn_xuid_category) : ils indexent des colonnes NOT NULL, donc pas
	// concernés par le bug.
	Register(Migration{
		Name:        "drop_idx_pn_xuid_unread",
		TargetDB:    TargetSharedSocial,
		Description: "Supprime idx_pn_xuid_unread (bug DuckDB ART/NULL sur UPDATE read_at).",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `DROP INDEX IF EXISTS idx_pn_xuid_unread;`)
		},
	})
}
