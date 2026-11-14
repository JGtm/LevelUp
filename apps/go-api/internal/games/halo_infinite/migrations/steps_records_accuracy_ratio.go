package migrations

// steps_records_accuracy_ratio.go — records_accuracy_to_ratio_v1 : les records de
// précision posés avant la correction de l'échelle sont convertis du pourcentage vers le ratio.
//
// Avant la correction, la précision (match_participants.accuracy, en pourcentage 0 à 100) arrivait
// aux records telle quelle, alors que leurs bornes sont en ratio 0 à 1 (ADR 0006). Les records de
// précision déjà rangés sont donc en pourcentage : laissés tels quels, ils empêcheraient tout
// nouveau record (une précision de 0,73 ne bat jamais 73,3). La conversion est INSERT-only, comme
// toute écriture de cette table append-only (ADR 0026) : chaque dernière ligne d'une précision au-dessus
// de 1 reçoit une ligne neuve en ratio, que la vue player_records_latest sert ensuite. Une valeur au-dessus
// de 100 n'est pas un pourcentage : elle n'est pas touchée.

import (
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/migration"
)

func stepAccuracyRecordsToRatio() migration.Migration {
	return migration.Migration{
		Name:        "records_accuracy_to_ratio_v1",
		TargetDB:    migration.TargetSharedSocial,
		Description: "Convertit en ratio (0..1) les records de précision rangés en pourcentage : une ligne neuve par dernier record concerné (INSERT seul).",
		ApplySchema: func(db *sql.DB) error {
			has, err := migration.TableExists(db, "player_records_history")
			if err != nil {
				return fmt.Errorf("accuracy_ratio: check table: %w", err)
			}
			if !has {
				return nil
			}
			res, err := db.ExecContext(migration.BootCtx(), `
				INSERT INTO player_records_history
					(xuid, metric, period, value, achieved_at, achieved_match_id, previous_value, previous_achieved_at)
				SELECT xuid, metric, period, value / 100.0, achieved_at, achieved_match_id,
					CASE WHEN previous_value > 1 THEN previous_value / 100.0 ELSE previous_value END,
					previous_achieved_at
				FROM player_records_latest
				WHERE metric = 'accuracy' AND value > 1 AND value <= 100`)
			if err != nil {
				return fmt.Errorf("accuracy_ratio: insert: %w", err)
			}
			if n, _ := res.RowsAffected(); n > 0 {
				slog.InfoContext(migration.BootCtx(), "migration records_accuracy_to_ratio_v1 : records de précision convertis en ratio",
					"lignes", n)
			}
			return nil
		},
	}
}
