// Package migration — steps_shared_create_squad_offset.go : LUSR v2 Sprint 1.C.
//
// Table player_squad_offset : offset de synergie d'escouade par paire de
// coéquipiers et par groupe de modes. Estimé hors-ligne par
// cmd/lusr_v2_squad_estimate, consommé au runtime par le shadow runner (gated
// LEVELUP_LUSR_V2_SQUAD_OFFSET=1) pour corriger la sur-estimation des joueurs
// qui jouent souvent ensemble.
//
// Append-only par construction (PK technique id + written_at, lecture via vue
// _latest) — cohérent avec player_skill_state_v2 / lusr_hyperparams_v2 et le
// pattern anti-ART du projet.

package migration

import "database/sql"

func init() {
	Register(Migration{
		Name:        "shared_create_player_squad_offset",
		TargetDB:    TargetShared,
		Description: "LUSR v2 Sprint 1.C — player_squad_offset (append-only) + vue _latest : offset synergie par paire de coéquipiers",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE SEQUENCE IF NOT EXISTS player_squad_offset_seq START 1;
				CREATE TABLE IF NOT EXISTS player_squad_offset (
					id              BIGINT DEFAULT nextval('player_squad_offset_seq') PRIMARY KEY,
					xuid            VARCHAR NOT NULL,
					partner_xuid    VARCHAR NOT NULL,
					playlist_group  VARCHAR NOT NULL,
					offset_value    DOUBLE  NOT NULL,
					match_count     INTEGER NOT NULL DEFAULT 0,
					source          VARCHAR NOT NULL,
					written_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
				);
				CREATE INDEX IF NOT EXISTS idx_pso_lookup
					ON player_squad_offset(xuid, playlist_group, partner_xuid, written_at DESC);

				CREATE OR REPLACE VIEW player_squad_offset_latest AS
				SELECT o.*
				FROM player_squad_offset o
				JOIN (
					SELECT xuid, partner_xuid, playlist_group, MAX(written_at) AS max_written_at
					FROM player_squad_offset
					GROUP BY xuid, partner_xuid, playlist_group
				) m
					ON o.xuid = m.xuid
					AND o.partner_xuid = m.partner_xuid
					AND o.playlist_group = m.playlist_group
					AND o.written_at = m.max_written_at;
			`)
		},
	})
}
