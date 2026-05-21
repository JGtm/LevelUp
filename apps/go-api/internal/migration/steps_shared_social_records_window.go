package migration

// steps_shared_social_records_window.go — extension de `player_records`
// (shared_social.duckdb) pour supporter les fenêtres temporelles 30d/90d/all_time
// du plan Progression Tracking V2.
//
// Schéma initial (steps_player_notifications.go) :
//   PRIMARY KEY (xuid, metric)
//
// Schéma cible :
//   PRIMARY KEY (xuid, metric, period)
//   + previous_value DOUBLE
//   + previous_achieved_at TIMESTAMP
//
// Note : la colonne s'appelle `period` (pas `window`) car window est un mot
// réservé DuckDB. Le domaine reste « fenêtre temporelle 30d/90d/all_time ».
//
// DuckDB ne supporte pas ALTER TABLE pour modifier la PK en place. La migration
// procède donc en 4 temps : create new, copy (period='all_time' pour les
// lignes existantes), drop old, rename new -> player_records.
//
// Idempotente via check `columnExists("player_records", "period")` — si le
// nouveau schéma est déjà en place, no-op.
//
// Réf : .ai/PLAN_PROGRESSION_TRACKING_ASCENSION.md §2bis + §7.2

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:        "extend_player_records_with_window",
		TargetDB:    TargetSharedSocial,
		Description: "Étend player_records avec period (30d/90d/all_time) + previous_value/previous_achieved_at + PK migrée vers (xuid, metric, period).",
		ApplySchema: func(db *sql.DB) error {
			// Idempotence : si la colonne `period` existe déjà, la migration
			// a été appliquée précédemment (potentiellement via un schema
			// recreate). Pas de double-application.
			hasPeriod, err := columnExists(db, "player_records", "period")
			if err != nil {
				return fmt.Errorf("extend_player_records: check period column: %w", err)
			}
			if hasPeriod {
				return nil
			}

			// Si la table source n'existe pas (cas DB neuve où la migration
			// originale `create_notifications_in_shared_social` n'a pas
			// encore tourné), no-op : cette migration ne crée pas la table
			// from scratch, elle l'étend uniquement.
			hasTable, err := tableExists(db, "player_records")
			if err != nil {
				return fmt.Errorf("extend_player_records: check table: %w", err)
			}
			if !hasTable {
				return nil
			}

			// Cleanup défensif : si une migration partielle précédente a laissé
			// la table temporaire, la supprimer avant de recommencer.
			if _, err := db.ExecContext(bootCtx(), `DROP TABLE IF EXISTS player_records_v2`); err != nil {
				return fmt.Errorf("extend_player_records: drop stale v2: %w", err)
			}

			return execScript(db, `
				CREATE TABLE player_records_v2 (
					xuid                 VARCHAR NOT NULL,
					metric               VARCHAR NOT NULL,
					period               VARCHAR NOT NULL DEFAULT 'all_time',
					value                DOUBLE NOT NULL,
					achieved_at          TIMESTAMP,
					achieved_match_id    VARCHAR,
					previous_value       DOUBLE,
					previous_achieved_at TIMESTAMP,
					updated_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					PRIMARY KEY (xuid, metric, period)
				);

				INSERT INTO player_records_v2 (xuid, metric, period, value, achieved_at, achieved_match_id, updated_at)
				SELECT xuid, metric, 'all_time', value, achieved_at, achieved_match_id, updated_at
				FROM player_records;

				DROP TABLE player_records;
				ALTER TABLE player_records_v2 RENAME TO player_records;
			`)
		},
	})
}
