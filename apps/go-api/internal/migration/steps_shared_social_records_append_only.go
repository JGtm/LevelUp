package migration

// steps_shared_social_records_append_only.go — migration vers le pattern
// append-only pour player_records (Phase 2 du refactor shared_social
// Collect→Persist).
//
// Contexte : `player_records` utilise un INSERT ON CONFLICT DO UPDATE
// (UPSERT) sur PK composite (xuid, metric, period). Sous concurrence, ce
// pattern pressionne l'index ART DuckDB (cf. ADR 0019) ET produit des
// entrées WAL non-rejouables (cf. bug DuckDB #7659) ce qui contribue au
// cycle de bug observé sur shared_social.
//
// Solution architecturale : append-only.
//   - Nouvelle table `player_records_history` (BIGSERIAL PK id + written_at)
//   - INSERT pur, jamais UPDATE
//   - Vue `player_records_latest` retourne la dernière valeur par
//     (xuid, metric, period) via DISTINCT ON ORDER BY written_at DESC
//
// Le SharedSocialPersister.persistPlayerRecords (Phase 1) cible déjà cette
// table _history. Le fallback persistPlayerRecordsLegacy utilise l'ancien
// UPSERT — la migration garde l'ancienne table `player_records` pour
// rétrocompat des lectures non-migrées (sera dépréciée en Phase 4).
//
// Backfill : `INSERT INTO _history SELECT * FROM player_records` one-shot.
//
// Idempotente : si `player_records_history` existe déjà, no-op.

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:        "create_player_records_history_append_only",
		TargetDB:    TargetSharedSocial,
		Description: "Pattern append-only pour player_records : nouvelle table _history (INSERT pur) + vue _latest. Backfill data existante depuis l'ancienne table player_records.",
		ApplySchema: func(db *sql.DB) error {
			// Idempotence : si _history existe déjà, no-op.
			hasHistory, err := tableExists(db, "player_records_history")
			if err != nil {
				return fmt.Errorf("create_player_records_history: check table: %w", err)
			}
			if hasHistory {
				return nil
			}

			// La table source `player_records` peut ne pas exister (DB neuve).
			// On crée _history + view dans tous les cas ; le backfill est
			// conditionnel sur l'existence de la source.
			hasSource, err := tableExists(db, "player_records")
			if err != nil {
				return fmt.Errorf("create_player_records_history: check source: %w", err)
			}

			// 1. CREATE SEQUENCE + TABLE.
			//    PK = id BIGINT (auto-incrémenté) → écritures append-only sans
			//    pression ART (pas de PK VARCHAR composite). Index secondaire
			//    optimise la vue _latest (DISTINCT ON xuid, metric, period).
			if _, err := db.ExecContext(bootCtx(), `
				CREATE SEQUENCE IF NOT EXISTS player_records_history_id_seq START 1;
				CREATE TABLE player_records_history (
					id                BIGINT PRIMARY KEY DEFAULT nextval('player_records_history_id_seq'),
					xuid              VARCHAR NOT NULL,
					metric            VARCHAR NOT NULL,
					period            VARCHAR NOT NULL DEFAULT 'all_time',
					value             DOUBLE NOT NULL,
					achieved_at       TIMESTAMP,
					achieved_match_id VARCHAR,
					written_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
				);
				CREATE INDEX idx_prh_lookup
					ON player_records_history(xuid, metric, period, written_at DESC);
			`); err != nil {
				return fmt.Errorf("create_player_records_history: schema: %w", err)
			}

			// 2. CREATE VIEW player_records_latest.
			//    DISTINCT ON (PostgreSQL-style, supporté par DuckDB) retourne
			//    la dernière row par (xuid, metric, period) selon written_at.
			//    Lectures repos consomment cette vue → transparent vs ancienne
			//    table player_records.
			if _, err := db.ExecContext(bootCtx(), `
				CREATE OR REPLACE VIEW player_records_latest AS
				SELECT DISTINCT ON (xuid, metric, period)
					id,
					xuid,
					metric,
					period,
					value,
					achieved_at,
					achieved_match_id,
					written_at
				FROM player_records_history
				ORDER BY xuid, metric, period, written_at DESC;
			`); err != nil {
				return fmt.Errorf("create_player_records_history: view: %w", err)
			}

			// 3. Backfill depuis l'ancienne table (one-shot, idempotent puisque
			//    la migration n'est appliquée qu'une fois grâce à
			//    schema_migrations).
			//
			//    Défensif vs ordre d'init Go (les migrations register dans un
			//    ordre non-déterministe) : on adapte le SELECT selon les
			//    colonnes présentes dans player_records au moment du backfill.
			//    - Si `period` existe (migration extend_player_records_with_window
			//      déjà appliquée) → inclus dans le backfill.
			//    - Sinon → period par défaut 'all_time'.
			if hasSource {
				hasPeriod, err := columnExists(db, "player_records", "period")
				if err != nil {
					return fmt.Errorf("create_player_records_history: check period column: %w", err)
				}
				var backfillSQL string
				if hasPeriod {
					backfillSQL = `
						INSERT INTO player_records_history
							(xuid, metric, period, value, achieved_at, achieved_match_id, written_at)
						SELECT
							xuid, metric, period, value, achieved_at, achieved_match_id,
							COALESCE(updated_at, CURRENT_TIMESTAMP)
						FROM player_records;
					`
				} else {
					backfillSQL = `
						INSERT INTO player_records_history
							(xuid, metric, period, value, achieved_at, achieved_match_id, written_at)
						SELECT
							xuid, metric, 'all_time', value, achieved_at, achieved_match_id,
							COALESCE(updated_at, CURRENT_TIMESTAMP)
						FROM player_records;
					`
				}
				if _, err := db.ExecContext(bootCtx(), backfillSQL); err != nil {
					return fmt.Errorf("create_player_records_history: backfill: %w", err)
				}
			}

			return nil
		},
	})
}
