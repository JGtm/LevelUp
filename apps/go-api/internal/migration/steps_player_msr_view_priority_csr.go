// Package migration — steps_player_msr_view_priority_csr.go :
// Phase 2.E du plan d'éradication ART (cf. .ai/PLAN_LUSR_ART_HOME_CRASH.md).
//
// **Objectif** : recréer la vue `match_skill_rank_latest` avec une
// priorité fonctionnelle CSR > LUSR par match_id.
//
// **Pourquoi** : avant l'append-only, le schéma garantissait "1 rating par
// match" via la PK simple (match_id) + le filtre SQL
// `WHERE rating_type <> 'CSR'` dans upsertLUSRRatings. Avec append-only,
// la PK est technique (id), et plus rien n'empêche d'avoir physiquement
// une row CSR et une row LUSR pour le même match. La garde Go (filtre
// existingCSR amont) reste en place mais on ne peut plus s'y reposer
// seul : si elle saute, la vue doit favoriser CSR (irrécupérable côté API)
// sur LUSR (recalculable depuis les stats locales).
//
// **Nouveau ORDER BY** : `CASE rating_type WHEN 'CSR' THEN 0 ELSE 1 END`
// → CSR est toujours préféré quand un match a les deux. Ensuite par
// written_at DESC (dernière version), puis id DESC (tie-break).
//
// **Idempotente** : `CREATE OR REPLACE VIEW` est intrinsèquement
// idempotente. La migration peut être ré-exécutée sans effet de bord.

package migration

import (
	"database/sql"
	"fmt"
	"log/slog"
)

func init() {
	Register(Migration{
		Name:        "player_msr_view_priority_csr_v1",
		TargetDB:    TargetPlayer,
		Description: "Recrée match_skill_rank_latest avec priorité CSR > LUSR par match_id (sémantique préservée sans garde SQL)",
		ApplySchema: applyMSRViewPriorityCSR,
	})
}

func applyMSRViewPriorityCSR(db *sql.DB) error {
	ctx := bootCtx()

	// Pré-condition : la table match_skill_rank doit exister ET avoir été
	// migrée en append-only (colonne `id` présente). Sinon la migration v1
	// ne s'est pas encore appliquée → la vue créée ici pointerait vers un
	// schéma incompatible. Skip silencieusement et laisser le cycle de
	// migrations suivant la rattraper.
	hasIDCol, err := columnExists(db, "match_skill_rank", "id")
	if err != nil {
		return fmt.Errorf("msr_view_priority_csr: check id column: %w", err)
	}
	if !hasIDCol {
		return nil
	}

	// CREATE OR REPLACE remplace l'éventuelle ancienne vue créée par
	// la migration v1. Le nouveau ORDER BY met CSR avant LUSR.
	const stmt = `
		CREATE OR REPLACE VIEW match_skill_rank_latest AS
			SELECT * FROM match_skill_rank
			QUALIFY ROW_NUMBER() OVER (
				PARTITION BY match_id
				ORDER BY
					CASE rating_type WHEN 'CSR' THEN 0 ELSE 1 END,
					written_at DESC,
					id DESC
			) = 1
	`
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("msr_view_priority_csr: recreate view: %w", err)
	}

	slog.InfoContext(ctx, "match_skill_rank_latest: vue recréée avec priorité CSR > LUSR (Phase 2.E)")
	return nil
}
