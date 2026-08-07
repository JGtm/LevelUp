package migration

// steps_shared_social_squad_challenge_participant_append_only.go — bascule
// squad_challenge_participant en APPEND-ONLY.
//
// AVANT : table d'ÉTAT mutée en place sur shared_social.duckdb (handle RW PARTAGÉ
// process-wide → 1 FATAL = TOUTE l'app down) :
//   - AddParticipant            : INSERT … ON CONFLICT (squad_challenge_id, user_id) DO NOTHING
//   - UpdateParticipantProgress : UPDATE current_value/completed_at WHERE (clé PK)
// L'ON CONFLICT et l'UPDATE pressionnent l'index ART de la PK (squad_challenge_id,
// user_id) → surface du bug DuckDB "Failed to delete all rows from index".
//
// APRÈS : event-log immuable squad_challenge_participant_history (PK technique seq
// BIGINT séquence, jamais retiré). Chaque progression = un INSERT pur carry-forward
// des champs immuables (chosen_tier, data_tier, is_private, joined_at) depuis
// _latest. Le join (AddParticipant) = INSERT idempotent (WHERE NOT EXISTS dans
// _latest) → un re-join NE RÉINITIALISE PAS la progression (point relevé en revue).
// État courant via vue squad_challenge_participant_latest (latest wins par
// (squad_challenge_id, user_id)). Aucun DELETE sur cette table → pas de tombstone.
//
// Table prouvablement vide en prod (aucun endpoint de création d'escouade livré,
// cf. rekey_squad_member_xuid) ; backfill défensif quand même. Pattern calqué sur
// shared_social_notif_prefs_append_only_v1. Idempotent : no-op si _history existe.

import (
	"database/sql"
	"fmt"
	"log/slog"
)

func init() {
	Register(Migration{
		Name:        "shared_social_squad_challenge_participant_append_only_v1",
		TargetDB:    TargetSharedSocial,
		Description: "squad_challenge_participant append-only : table _history + vue _latest + backfill — élimine ON CONFLICT + UPDATE (surface ART shared_social)",
		ApplySchema: applySquadChallengeParticipantAppendOnly,
	})
}

func applySquadChallengeParticipantAppendOnly(db *sql.DB) error {
	ctx := bootCtx()

	hasHistory, err := tableExists(db, "squad_challenge_participant_history")
	if err != nil {
		return fmt.Errorf("squad_challenge_participant append-only: check history: %w", err)
	}
	if hasHistory {
		return nil // déjà migré
	}

	stmts := []string{
		`CREATE SEQUENCE IF NOT EXISTS squad_challenge_participant_history_seq START 1`,
		`CREATE TABLE squad_challenge_participant_history (
			seq                BIGINT PRIMARY KEY DEFAULT nextval('squad_challenge_participant_history_seq'),
			squad_challenge_id VARCHAR NOT NULL,
			user_id            VARCHAR NOT NULL,
			chosen_tier        VARCHAR,
			data_tier          VARCHAR NOT NULL DEFAULT 'full',
			current_value      DOUBLE NOT NULL DEFAULT 0,
			completed_at       TIMESTAMP,
			is_private         BOOLEAN DEFAULT FALSE,
			joined_at          TIMESTAMP,
			written_at         TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		)`,
		// Index secondaires NON NULL-bearing (clés NOT NULL), alimentés
		// uniquement par INSERT → jamais de retrait/relocation d'entrée ART.
		`CREATE INDEX IF NOT EXISTS idx_scph_lookup ON squad_challenge_participant_history(squad_challenge_id, user_id, written_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_scph_user ON squad_challenge_participant_history(user_id)`,
		`CREATE OR REPLACE VIEW squad_challenge_participant_latest AS
			SELECT squad_challenge_id, user_id, chosen_tier, data_tier,
			       current_value, completed_at, is_private, joined_at, written_at
			FROM squad_challenge_participant_history
			QUALIFY ROW_NUMBER() OVER (
				PARTITION BY squad_challenge_id, user_id
				ORDER BY written_at DESC, seq DESC
			) = 1`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("squad_challenge_participant append-only: step (%s): %w", firstWords(s, 3), err)
		}
	}

	// Backfill : chaque ligne legacy = l'état courant d'une participation → un event.
	if hasLegacy, _ := tableExists(db, "squad_challenge_participant"); hasLegacy {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO squad_challenge_participant_history (
				squad_challenge_id, user_id, chosen_tier, data_tier,
				current_value, completed_at, is_private, joined_at, written_at
			)
			SELECT squad_challenge_id, user_id, chosen_tier, data_tier,
			       current_value, completed_at, is_private, joined_at,
			       COALESCE(joined_at, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))
			FROM squad_challenge_participant`); err != nil {
			return fmt.Errorf("squad_challenge_participant append-only: backfill: %w", err)
		}
	}

	var n int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM squad_challenge_participant_history`).Scan(&n)
	slog.InfoContext(ctx, "squad_challenge_participant append-only: migration appliquée", "rows_backfilled", n)
	return nil
}
