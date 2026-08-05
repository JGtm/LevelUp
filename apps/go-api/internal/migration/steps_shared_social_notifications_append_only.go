package migration

// steps_shared_social_notifications_append_only.go — bascule player_notifications
// en APPEND-ONLY (doctrine : zéro DELETE, zéro UPDATE indexé, tout horodaté).
//
// AVANT : table d'ÉTAT mutée en place sur shared_social.duckdb (handle RW PARTAGÉ
// process-wide → 1 FATAL = TOUTE l'app down) :
//   - DELETE per-row : Delete (notif unique) + CapAndSweep (purge rétention)
//     → retire des entrées de la PK ART (xuid, id) → surface du bug DuckDB
//       "Failed to delete all rows from index. Only deleted 0 out of N rows".
//   - UPDATE read_at : MarkRead / MarkUnread / MarkAllRead.
//
// APRÈS : event-log immuable player_notifications_history. Chaque opération = un
// INSERT pur :
//   - create  : nouvel event (read_at NULL, is_deleted FALSE).
//   - mark-read / mark-unread : INSERT…SELECT carry-forward du payload depuis
//     player_notifications_latest, avec read_at positionné/NULL.
//   - delete / cap-sweep : INSERT d'un event tombstone (is_deleted TRUE).
// PK technique seq BIGINT (séquence, monotone, jamais retiré) → aucune pression
// ART. État courant = dernier event par (xuid, id), lu via la vue
// player_notifications_latest (filtrée is_deleted=FALSE). Plus aucun DELETE/UPDATE
// → pansement ExecRecovered/WithReopenOnInvalidated inutile sur ce chemin.
//
// Croissance : l'historique n'est jamais purgé (doctrine append-only). Volume
// borné en pratique (notifs capées à l'affichage via les tombstones cap-sweep).
// Une compaction physique éventuelle resterait un job sérialisé séparé (cf.
// match_skill_rank), hors hot path.
//
// Pattern calqué sur shared_social_notif_prefs_append_only_v1 (table sœur _history
// + vue _latest + backfill) + tombstone is_deleted comme media_match_associations.
// Idempotent : no-op si _history existe.

import (
	"database/sql"
	"fmt"
	"log/slog"
)

func init() {
	Register(Migration{
		Name:        "shared_social_notifications_append_only_v1",
		TargetDB:    TargetSharedSocial,
		Description: "player_notifications append-only : table _history (event read_at + tombstone is_deleted) + vue _latest + backfill — élimine DELETE/UPDATE (surface ART shared_social)",
		ApplySchema: applyNotificationsAppendOnly,
	})
}

func applyNotificationsAppendOnly(db *sql.DB) error {
	ctx := bootCtx()

	hasHistory, err := tableExists(db, "player_notifications_history")
	if err != nil {
		return fmt.Errorf("notifications append-only: check history: %w", err)
	}
	if hasHistory {
		return nil // déjà migré
	}

	stmts := []string{
		`CREATE SEQUENCE IF NOT EXISTS player_notifications_history_seq START 1`,
		`CREATE TABLE player_notifications_history (
			seq           BIGINT PRIMARY KEY DEFAULT nextval('player_notifications_history_seq'),
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
			created_at    TIMESTAMP NOT NULL,
			read_at       TIMESTAMP,
			is_deleted    BOOLEAN NOT NULL DEFAULT FALSE,
			written_at    TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		)`,
		// Index secondaire NON NULL-bearing (xuid, id NOT NULL), alimenté
		// uniquement par INSERT → jamais de retrait/relocation d'entrée ART.
		`CREATE INDEX IF NOT EXISTS idx_pnh_lookup ON player_notifications_history(xuid, id, written_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_pnh_xuid_created ON player_notifications_history(xuid, created_at DESC)`,
		// Vue _latest : dernier event par (xuid, id), tombstones exclus. Forme
		// sous-requête (rn=1 AND is_deleted=FALSE) — le filtre d'état doit
		// s'appliquer APRÈS le ranking (un WHERE is_deleted=FALSE avant le window
		// exposerait à tort l'avant-dernier event d'une notif supprimée).
		`CREATE OR REPLACE VIEW player_notifications_latest AS
			SELECT xuid, id, category, severity, title_key, body_key, params,
			       target_route, target_search, actor_xuid, actor_name, source,
			       created_at, read_at, written_at
			FROM (
				SELECT *,
				       ROW_NUMBER() OVER (
				           PARTITION BY xuid, id ORDER BY written_at DESC, seq DESC
				       ) AS rn
				FROM player_notifications_history
			) ranked
			WHERE rn = 1 AND is_deleted = FALSE`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("notifications append-only: step (%s): %w", firstWords(s, 3), err)
		}
	}

	// Backfill : chaque ligne legacy = l'état courant d'une notif → un event
	// (is_deleted=FALSE, read_at préservé). written_at = created_at pour garder
	// un ordre stable.
	if hasLegacy, _ := tableExists(db, "player_notifications"); hasLegacy {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO player_notifications_history (
				xuid, id, category, severity, title_key, body_key, params,
				target_route, target_search, actor_xuid, actor_name, source,
				created_at, read_at, is_deleted, written_at
			)
			SELECT xuid, id, category, severity, title_key, body_key, params,
			       target_route, target_search, actor_xuid, actor_name, source,
			       created_at, read_at, FALSE, COALESCE(created_at, CURRENT_TIMESTAMP)
			FROM player_notifications`); err != nil {
			return fmt.Errorf("notifications append-only: backfill: %w", err)
		}
	}

	var n int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM player_notifications_history`).Scan(&n)
	slog.InfoContext(ctx, "player_notifications append-only: migration appliquée", "rows_backfilled", n)
	return nil
}
