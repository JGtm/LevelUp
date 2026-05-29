// Package migration — steps_player_msr_view_lusr_over_v2.go :
// Fix du masquage LUSR par LUSR_V2 dans match_skill_rank_latest.
//
// **Bug** : la Stratégie C (ADR 0024) écrit DEUX rows par match dans le MÊME
// batch (donc même written_at) : rating_type='LUSR' (lu par l'UI) et
// rating_type='LUSR_V2' (audit). La vue match_skill_rank_latest (Phase 2.E)
// départageait par `written_at DESC, id DESC` — or LUSR_V2, inséré juste après
// LUSR, a un `id` plus grand → il GAGNE le ROW_NUMBER=1. La vue représentait
// donc chaque match v2 par sa row LUSR_V2, et les readers UI qui filtrent
// `rating_type='LUSR'` récupéraient une row PÉRIMÉE (rating sous-évalué, ex.
// Gold au lieu de Diamant alors que la bonne row LUSR était bien présente).
//
// **Fix** : ordre de priorité CSR (0) > LUSR (1) > LUSR_V2 (2). LUSR_V2 est
// audit-only → ne doit jamais être le représentant "latest" d'un match quand
// une row LUSR/CSR existe (elle existe toujours, écrite dans le même batch).
//
// **Sentinelle** : le check dual-row (RunDualRowSentinel) ne lit plus cette vue
// (qui collapse à 1 row/match) mais la table raw — voir skill_v2_metrics.go.
//
// **Idempotente** : CREATE OR REPLACE VIEW.

package migration

import (
	"database/sql"
	"fmt"
	"log/slog"
)

func init() {
	Register(Migration{
		Name:        "player_msr_view_lusr_over_v2_v1",
		TargetDB:    TargetPlayer,
		Description: "match_skill_rank_latest : priorité CSR > LUSR > LUSR_V2 (fin du masquage LUSR par la row audit LUSR_V2)",
		ApplySchema: applyMSRViewLUSROverV2,
	})
}

func applyMSRViewLUSROverV2(db *sql.DB) error {
	ctx := bootCtx()

	hasIDCol, err := columnExists(db, "match_skill_rank", "id")
	if err != nil {
		return fmt.Errorf("msr_view_lusr_over_v2: check id column: %w", err)
	}
	if !hasIDCol {
		return nil // append-only pas encore appliqué → cycle suivant rattrapera
	}

	const stmt = `
		CREATE OR REPLACE VIEW match_skill_rank_latest AS
			SELECT * FROM match_skill_rank
			QUALIFY ROW_NUMBER() OVER (
				PARTITION BY match_id
				ORDER BY
					CASE rating_type WHEN 'CSR' THEN 0 WHEN 'LUSR' THEN 1 ELSE 2 END,
					written_at DESC,
					id DESC
			) = 1
	`
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("msr_view_lusr_over_v2: recreate view: %w", err)
	}

	slog.InfoContext(ctx, "match_skill_rank_latest: vue recréée avec priorité CSR > LUSR > LUSR_V2")
	return nil
}
