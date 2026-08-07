package migration

// steps_shared_social_squad_member_append_only.go — bascule squad_member en APPEND-ONLY.
//
// AVANT : AddMember = INSERT ON CONFLICT DO NOTHING ; RemoveMember = DELETE FROM
// squad_member. Le DELETE retire l'entrée de la PK ART (squad_id, xuid) + idx_sm_xuid
// + idx_sm_user → surface ART sur shared_social.duckdb (handle partagé concurrent).
//
// APRÈS : join/leave = INSERT pur d'event (is_member TRUE/FALSE) dans
// squad_member_history. État courant (membres actifs d'une squad) lu via
// squad_member_latest. Zéro DELETE / ON CONFLICT.
//
// NB : la table héritée squad_member est re-créée par rekey_squad_member_xuid
// (DROP+CREATE). Le backfill est donc gardé par columnExists(xuid) : si cette
// migration s'exécute avant le rekey (schéma sans xuid), on saute le backfill
// (la feature squad est neuve/vide — routes non exposées). squad_member_history
// est créée dans tous les cas.

import (
	"database/sql"
	"fmt"
	"log/slog"
)

func init() {
	Register(Migration{
		Name:        "shared_social_squad_member_append_only_v1",
		TargetDB:    TargetSharedSocial,
		Description: "squad_member append-only : table _history (event is_member) + vue _latest + backfill — élimine DELETE + ON CONFLICT",
		ApplySchema: applySquadMemberAppendOnly,
	})
}

func applySquadMemberAppendOnly(db *sql.DB) error {
	ctx := bootCtx()

	hasHistory, err := tableExists(db, "squad_member_history")
	if err != nil {
		return fmt.Errorf("squad_member append-only: check history: %w", err)
	}
	if hasHistory {
		return nil
	}

	stmts := []string{
		`CREATE SEQUENCE IF NOT EXISTS squad_member_history_id_seq START 1`,
		`CREATE TABLE squad_member_history (
			id         BIGINT PRIMARY KEY DEFAULT nextval('squad_member_history_id_seq'),
			squad_id   VARCHAR NOT NULL,
			xuid       VARCHAR NOT NULL,
			user_id    VARCHAR,
			is_member  BOOLEAN NOT NULL,
			joined_at  TIMESTAMP,
			written_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_smh_lookup ON squad_member_history(squad_id, xuid, written_at DESC)`,
		`CREATE OR REPLACE VIEW squad_member_latest AS
			SELECT id, squad_id, xuid, user_id, is_member, joined_at, written_at
			FROM squad_member_history
			QUALIFY ROW_NUMBER() OVER (
				PARTITION BY squad_id, xuid
				ORDER BY written_at DESC, id DESC
			) = 1`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("squad_member append-only: step (%s): %w", firstWords(s, 3), err)
		}
	}

	// Backfill seulement si squad_member existe AVEC la colonne xuid (post-rekey).
	hasLegacy, _ := tableExists(db, "squad_member")
	hasXuid, _ := columnExists(db, "squad_member", "xuid")
	if hasLegacy && hasXuid {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO squad_member_history (squad_id, xuid, user_id, is_member, joined_at, written_at)
			SELECT squad_id, xuid, user_id, TRUE, joined_at, COALESCE(joined_at, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))
			FROM squad_member`); err != nil {
			return fmt.Errorf("squad_member append-only: backfill: %w", err)
		}
	}

	var n int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM squad_member_history`).Scan(&n)
	slog.InfoContext(ctx, "squad_member append-only: migration appliquée", "rows_backfilled", n)
	return nil
}
