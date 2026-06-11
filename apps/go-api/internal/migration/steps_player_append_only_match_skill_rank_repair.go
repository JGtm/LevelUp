// Package migration — steps_player_append_only_match_skill_rank_repair.go.
//
// Nommé pour s'initialiser JUSTE APRÈS steps_player_append_only_match_skill_rank.go
// (l'ordre d'enregistrement des migrations suit l'ordre alphabétique des fichiers ;
// canonicalOrder doit refléter cet ordre — cf. order_test.go).
//
// **Pourquoi** : la migration append-only (steps_player_append_only_match_skill_rank)
// est idempotente sur la colonne `id`. Si, sur une base donnée, le CTAS a bien créé
// la colonne `written_at` mais que le `ALTER ... SET DEFAULT now()` (l.106) n'a pas
// abouti (interruption, ordre), la base reste à `written_at DEFAULT NULL` — et comme
// `id` existe, la migration d'origine ne re-tente JAMAIS le ALTER.
//
// Symptôme observé (2026-06-11) : base de Chocoboflor avec `written_at DEFAULT NULL`
// → 3685 rows à written_at NULL → `loadPreviousLUSRRating` (qui ordonnait par
// written_at DESC, NULLS LAST) renvoyait une vieille row figée → delta LUSR +75 faux.
// Idem la vue `match_skill_rank_latest` (ORDER BY written_at DESC) pouvait remonter
// une version périmée.
//
// Cette migration (nom inédit → tourne une fois sur CHAQUE base) répare :
//   1. (ré)applique `ALTER COLUMN written_at SET DEFAULT now()` (idempotent),
//   2. backfille les `written_at` NULL existants depuis `created_at` (fallback now()).
//
// Le delta LUSR lui-même est désormais ordonné par `start_time` (cf.
// skill_v2_canonical.go) — robuste indépendamment de written_at ; cette migration
// fiabilise la vue latest et sert de garde-fou.

package migration

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:        "msr_written_at_default_now_repair_v1",
		TargetDB:    TargetPlayer,
		Description: "Répare written_at de match_skill_rank (DEFAULT now() + backfill des NULL) — migration append-only partielle sur certaines bases",
		ApplySchema: repairMatchSkillRankWrittenAt,
	})
}

func repairMatchSkillRankWrittenAt(db *sql.DB) error {
	ctx := bootCtx()

	hasTable, err := tableExists(db, "match_skill_rank")
	if err != nil {
		return fmt.Errorf("repair written_at: check table: %w", err)
	}
	if !hasTable {
		return nil
	}
	// La colonne written_at n'existe que post-migration append-only. Si absente,
	// la bascule append-only n'a pas (encore) eu lieu → rien à réparer ici.
	hasCol, err := columnExists(db, "match_skill_rank", "written_at")
	if err != nil {
		return fmt.Errorf("repair written_at: check column: %w", err)
	}
	if !hasCol {
		return nil
	}

	// SEULEMENT du DDL. On NE fait PAS d'UPDATE des written_at NULL existants :
	// match_skill_rank est append-only et possède un index sur (match_id,
	// rating_type, written_at) — un UPDATE de written_at déclenche le bug ART
	// DuckDB "Failed to delete all rows from index" (incident 2026-06-11 sur la
	// base de Chocoboflor). Les rows NULL héritées sont gérées au READ : la vue
	// match_skill_rank_latest et loadPreviousLUSRRating ordonnent désormais par
	// start_time (chronologique), pas written_at — les rows re-backfillées
	// (start_time peuplé) dominent sans toucher written_at. Cet ALTER ne fait que
	// garantir un written_at peuplé pour les FUTURES insertions.
	if _, err := db.ExecContext(ctx, `ALTER TABLE match_skill_rank ALTER COLUMN written_at SET DEFAULT now()`); err != nil {
		return fmt.Errorf("repair written_at: ALTER SET DEFAULT: %w", err)
	}
	return nil
}
