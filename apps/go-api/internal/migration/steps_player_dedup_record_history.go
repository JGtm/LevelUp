package migration

// steps_player_dedup_record_history.go — dédup one-shot de record_history
// (fix 2026-05-30).
//
// Contexte : la migration des records vers l'append-only (player_records_history
// + vue player_records_latest) a laissé, le temps d'un backfill de transition,
// player_records_latest vide alors que record_history contenait déjà les PB. Le
// détecteur a donc re-détecté ces PB comme "nouveaux" et ré-appendé une entrée
// timeline → doublons logiques (même user/metric/period/value/achieved_at).
//
// Nettoyage ART-safe : rebuild CTAS (PAS de DELETE — qui rejouerait le bug ART
// "delete from index"), en gardant 1 ligne par clé logique. CONDITIONNEL : no-op
// si aucun doublon (la grande majorité des player DB), pour ne pas reconstruire
// inutilement une table saine.
//
// Idempotente : après dédup, total == clés distinctes → no-op au re-run.

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:        "dedup_record_history_v1",
		TargetDB:    TargetPlayer,
		Description: "Dédoublonne record_history (doublons de transition append-only records). Rebuild CTAS conditionnel, no-op si aucun doublon.",
		ApplySchema: func(db *sql.DB) error {
			has, err := tableExists(db, "record_history")
			if err != nil {
				return fmt.Errorf("dedup_record_history: check table: %w", err)
			}
			if !has {
				return nil
			}

			// Compter total vs clés logiques distinctes. Si égal → aucun doublon.
			var total, distinctKeys int
			if err := db.QueryRowContext(bootCtx(), `SELECT COUNT(*) FROM record_history`).Scan(&total); err != nil {
				return fmt.Errorf("dedup_record_history: count total: %w", err)
			}
			if err := db.QueryRowContext(bootCtx(), `
				SELECT COUNT(*) FROM (
					SELECT 1 FROM record_history
					GROUP BY user_id, title_slug, metric, period, value, achieved_at
				)
			`).Scan(&distinctKeys); err != nil {
				return fmt.Errorf("dedup_record_history: count distinct: %w", err)
			}
			if total == distinctKeys {
				return nil // aucun doublon → no-op
			}

			// Rebuild CTAS : garde la 1re ligne (id min) par clé logique.
			if _, err := db.ExecContext(bootCtx(), `
				CREATE TABLE record_history__dedup AS
				SELECT * FROM record_history
				QUALIFY ROW_NUMBER() OVER (
					PARTITION BY user_id, title_slug, metric, period, value, achieved_at
					ORDER BY id
				) = 1;
				DROP TABLE record_history;
				ALTER TABLE record_history__dedup RENAME TO record_history;
				ALTER TABLE record_history ADD PRIMARY KEY (id);
				CREATE INDEX IF NOT EXISTS idx_rec_hist_user_title_metric
					ON record_history(user_id, title_slug, metric);
				CREATE INDEX IF NOT EXISTS idx_rec_hist_achieved_desc
					ON record_history(achieved_at DESC);
			`); err != nil {
				return fmt.Errorf("dedup_record_history: rebuild: %w", err)
			}
			return nil
		},
	})
}
