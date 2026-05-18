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
	"strings"
)

// notificationDefaultCategories liste les catégories du MVP avec leur état
// initial. Toutes activées par défaut. Le seed est lazy : `IsCategoryEnabled`
// retourne true si pas d'entrée pour la catégorie, donc pas besoin d'insérer
// au boot. La table reste vide tant qu'aucune préférence n'a été modifiée.
var notificationDefaultCategories = []struct {
	Category string
	Enabled  bool
	Delivery string
}{
	{"app_release", true, "both"},
	{"match_synced", true, "both"},
	{"media_added", true, "both"},
	{"objective_assigned", true, "both"},
	{"objective_completed", true, "both"},
	{"challenge_added", true, "inapp"},
	{"challenge_completed", true, "both"},
	// season_pass_level : déprécié 2026-05-16 — gardé pour rétro-compat seulement,
	// off par défaut puisque plus jamais émis.
	{"season_pass_level", false, "off"},
	{"sync_error", true, "both"},
	{"personal_record", true, "both"},
	{"threshold_crossed", true, "both"},
	// §6 Squad/Sessions overhaul : flow ami.
	{"friend_added", true, "inapp"},          // notif sobre, pas de toast
	{"friend_sync_completed", true, "inapp"}, // récap silencieux post-recompute
	// 2026-05-08 : audit santé DB périodique → warnings admin.
	{"data_health_warning", true, "inapp"},
	// 2026-05-16 : extension notifications — rang Halo, skill, BP, citations.
	{"career_rank", true, "both"},           // rare, marquant → toast + inapp
	{"skill_tier", true, "both"},            // CSR/LUSR unifié, peu fréquent
	{"battlepass_completed", true, "both"},  // milestone
	{"citation_tier", true, "inapp"},        // potentiellement fréquent → silent
	{"citation_mastery", true, "both"},      // rare → toast
	// 2026-05-18 : couche 3 du plan PROGRESSION_TRACKING (Ascension V2) — coach proactif.
	{"record_near_miss", true, "inapp"},     // potentiellement fréquent → silent (pas de toast)
	{"milestone_unlocked", true, "both"},    // marquant → toast + inapp
	{"milestone_near_miss", true, "inapp"},  // silent
	{"lusr_tier_approach", true, "both"},    // peu fréquent + actionable → toast + inapp
	{"streak_milestone", true, "both"},      // palier de streak (7/14/30j) marquant
	{"comeback_welcome", true, "both"},      // bienveillant, rare → toast + inapp
}

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
				CREATE INDEX IF NOT EXISTS idx_pn_xuid_unread       ON player_notifications(xuid, read_at);
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

// seedNotificationPreferencesForXUID insère les catégories par défaut pour un
// xuid donné, sans écraser une entrée existante. Pas appelé par les migrations
// (les prefs sont default-on via IsCategoryEnabled), mais exposé pour des seeds
// programmatique si besoin futur.
//
//nolint:unused // utilitaire pour seed manuel ; conservé pour API future.
func seedNotificationPreferencesForXUID(db *sql.DB, xuid string) error {
	for _, c := range notificationDefaultCategories {
		_, err := db.Exec(
			`INSERT INTO notification_preferences (xuid, category, enabled, delivery)
			 SELECT ?, ?, ?, ?
			 WHERE NOT EXISTS (
				 SELECT 1 FROM notification_preferences
				 WHERE xuid = ? AND category = ?
			 )`,
			xuid, c.Category, c.Enabled, c.Delivery, xuid, c.Category,
		)
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			return err
		}
	}
	return nil
}
