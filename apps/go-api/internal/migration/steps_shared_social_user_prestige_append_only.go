package migration

// steps_shared_social_user_prestige_append_only.go — bascule user_prestige en
// APPEND-ONLY.
//
// AVANT : table d'ÉTAT mutée sur shared_social.duckdb (handle RW PARTAGÉ concurrent
// → 1 FATAL = TOUTE l'app down) :
//   - EmitEvent          : INSERT … ON CONFLICT (user_id, title_slug) DO UPDATE
//                          total_pp = total_pp + EXCLUDED.total_pp (accumulation)
//   - UpsertUserPrestige : INSERT … ON CONFLICT DO UPDATE overwrite total/level
// L'ON CONFLICT DO UPDATE réécrit la ligne via l'index ART de la PK → surface du bug
// DuckDB "Failed to delete all rows from index".
//
// APRÈS : event-log user_prestige_history (PK technique seq BIGINT séquence). Chaque
// écriture = un INSERT pur d'un snapshot complet du total :
//   - EmitEvent          : INSERT…SELECT carry-forward (total courant via _latest + delta).
//   - UpsertUserPrestige : INSERT d'un snapshot (total_pp/current_level fournis).
// État courant = vue user_prestige_latest (latest wins par (user_id, title_slug)). Le
// journal immuable prestige_events reste la source des gains ; user_prestige n'est qu'un
// total dérivé maintenu. Aucun DELETE sur cette table → pas de tombstone.
//
// Pattern calqué sur shared_social_notif_prefs_append_only_v1 (version, latest wins).
// Idempotent : no-op si _history existe.

import (
	"database/sql"
	"fmt"
	"log/slog"
)

func init() {
	Register(Migration{
		Name:        "shared_social_user_prestige_append_only_v1",
		TargetDB:    TargetSharedSocial,
		Description: "user_prestige append-only : table _history + vue _latest + backfill — élimine ON CONFLICT DO UPDATE (surface ART shared_social)",
		ApplySchema: applyUserPrestigeAppendOnly,
	})
}

func applyUserPrestigeAppendOnly(db *sql.DB) error {
	ctx := bootCtx()

	hasHistory, err := tableExists(db, "user_prestige_history")
	if err != nil {
		return fmt.Errorf("user_prestige append-only: check history: %w", err)
	}
	if hasHistory {
		return nil // déjà migré
	}

	stmts := []string{
		`CREATE SEQUENCE IF NOT EXISTS user_prestige_history_seq START 1`,
		`CREATE TABLE user_prestige_history (
			seq           BIGINT PRIMARY KEY DEFAULT nextval('user_prestige_history_seq'),
			user_id       VARCHAR NOT NULL,
			title_slug    VARCHAR NOT NULL,
			total_pp      INTEGER NOT NULL DEFAULT 0,
			current_level INTEGER NOT NULL DEFAULT 0,
			updated_at    TIMESTAMP,
			written_at    TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_uph_lookup ON user_prestige_history(user_id, title_slug, written_at DESC)`,
		`CREATE OR REPLACE VIEW user_prestige_latest AS
			SELECT user_id, title_slug, total_pp, current_level, updated_at, written_at
			FROM user_prestige_history
			QUALIFY ROW_NUMBER() OVER (
				PARTITION BY user_id, title_slug
				ORDER BY written_at DESC, seq DESC
			) = 1`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("user_prestige append-only: step (%s): %w", firstWords(s, 3), err)
		}
	}

	if hasLegacy, _ := tableExists(db, "user_prestige"); hasLegacy {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO user_prestige_history (user_id, title_slug, total_pp, current_level, updated_at, written_at)
			SELECT user_id, title_slug, total_pp, current_level, updated_at, COALESCE(updated_at, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))
			FROM user_prestige`); err != nil {
			return fmt.Errorf("user_prestige append-only: backfill: %w", err)
		}
	}

	var n int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_prestige_history`).Scan(&n)
	slog.InfoContext(ctx, "user_prestige append-only: migration appliquée", "rows_backfilled", n)
	return nil
}
