package migration

// steps_shared_skill_v2.go — création des tables LUSR v2 (Phase 1b du chantier).
//
// Append-only par construction (PK technique id + colonne written_at), lecture
// via vues *_latest. Cohérent avec le pattern de match_skill_rank et match_csrs
// (cf. Phase 2 du refactor ART, 2026-05-24, CLAUDE.md "Tables append-only").
//
// Aucun impact sur le LUSR v1 — tables séparées, vues séparées, le service v2
// est gated par le feature flag LEVELUP_LUSR_V2_ENABLED. Tant que le flag est
// off (défaut), aucune donnée n'y entre.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "shared_create_skill_v2_tables",
		TargetDB:    TargetShared,
		Description: "LUSR v2 — player_skill_state_v2 (append-only) + lusr_hyperparams_v2 + vues _latest",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE SEQUENCE IF NOT EXISTS player_skill_state_v2_seq START 1;
				CREATE TABLE IF NOT EXISTS player_skill_state_v2 (
					id              BIGINT DEFAULT nextval('player_skill_state_v2_seq') PRIMARY KEY,
					xuid            VARCHAR NOT NULL,
					playlist_group  VARCHAR NOT NULL,
					mu              DOUBLE  NOT NULL,
					sigma           DOUBLE  NOT NULL,
					experience      INTEGER NOT NULL DEFAULT 0,
					last_match_id   VARCHAR,
					last_match_at   TIMESTAMP,
					written_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
				);
				CREATE INDEX IF NOT EXISTS idx_pssv2_xuid_group_written
					ON player_skill_state_v2(xuid, playlist_group, written_at DESC);

				CREATE OR REPLACE VIEW player_skill_state_v2_latest AS
				SELECT s.*
				FROM player_skill_state_v2 s
				JOIN (
					SELECT xuid, playlist_group, MAX(written_at) AS max_written_at
					FROM player_skill_state_v2
					GROUP BY xuid, playlist_group
				) m
					ON s.xuid = m.xuid
					AND s.playlist_group = m.playlist_group
					AND s.written_at = m.max_written_at;

				CREATE SEQUENCE IF NOT EXISTS lusr_hyperparams_v2_seq START 1;
				CREATE TABLE IF NOT EXISTS lusr_hyperparams_v2 (
					id              BIGINT DEFAULT nextval('lusr_hyperparams_v2_seq') PRIMARY KEY,
					playlist_group  VARCHAR NOT NULL,
					name            VARCHAR NOT NULL,
					value           DOUBLE  NOT NULL,
					source          VARCHAR NOT NULL,
					written_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
				);
				CREATE INDEX IF NOT EXISTS idx_lhv2_group_name_written
					ON lusr_hyperparams_v2(playlist_group, name, written_at DESC);

				CREATE OR REPLACE VIEW lusr_hyperparams_v2_latest AS
				SELECT h.*
				FROM lusr_hyperparams_v2 h
				JOIN (
					SELECT playlist_group, name, MAX(written_at) AS max_written_at
					FROM lusr_hyperparams_v2
					GROUP BY playlist_group, name
				) m
					ON h.playlist_group = m.playlist_group
					AND h.name = m.name
					AND h.written_at = m.max_written_at;
			`)
		},
	})
}
