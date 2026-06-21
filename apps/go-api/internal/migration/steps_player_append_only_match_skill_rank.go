// Package migration — steps_player_append_only_match_skill_rank.go :
// Phase 2.B du plan d'éradication ART (cf. .ai/PLAN_LUSR_ART_HOME_CRASH.md).
//
// Transforme `match_skill_rank` en table append-only et expose une vue
// `match_skill_rank_latest` pour la lecture "version courante".
//
// **Pourquoi** : l'ancien schéma avait `match_id` comme PRIMARY KEY simple,
// ce qui forçait tous les `INSERT ... ON CONFLICT DO UPDATE` (et donc
// implicitement des DELETE+INSERT côté moteur DuckDB) sur l'index ART —
// déclencheur empirique du crash "Failed to delete all rows from index"
// observé en prod 2026-05-24 20:41:04 sur Chocoboflor.
//
// **Stratégie append-only** : la table reçoit désormais N versions par
// (match_id, rating_type). La PK technique `id BIGINT` (auto-incrémentée
// via séquence `msr_seq`) garantit l'unicité physique sans aucune
// contrainte fonctionnelle. La vue `match_skill_rank_latest` filtre la
// dernière version par (match_id, rating_type) via window function.
//
// Toutes les écritures futures doivent être de simples INSERT (cf.
// AppendOnlyLUSRPersister, Phase 2.A). Aucun DELETE/UPDATE/UPSERT.
// Le bug ART devient impossible par construction.
//
// **Idempotence** : check `columnExists(id)` → no-op si déjà appliquée.

package migration

import (
	"database/sql"
)

func init() {
	Register(Migration{
		Name:        "player_append_only_match_skill_rank_v1",
		TargetDB:    TargetPlayer,
		Description: "Rebuild match_skill_rank en append-only (id PK + written_at + vue latest) — élimine bug ART par construction",
		ApplySchema: applyAppendOnlyMatchSkillRank,
	})
}

// applyAppendOnlyMatchSkillRank délègue au helper commun rebuildAppendOnlyTx
// (swap CTAS transactionnel + garde rebuilt==before + recoverOrphan). Mécanisme
// written_at (dernière version par match_id+rating_type). Cf. append_only_rebuild.go.
func applyAppendOnlyMatchSkillRank(db *sql.DB) error {
	return applyAppendOnlyRebuild(db, appendOnlyRebuild{
		Table:         "match_skill_rank",
		IDSeq:         "msr_seq",
		SyntheticCols: "CURRENT_TIMESTAMP AS written_at",
		PostSwap: []string{
			`ALTER TABLE match_skill_rank ALTER COLUMN written_at SET DEFAULT now()`,
			`CREATE INDEX IF NOT EXISTS idx_msr_match_lookup ON match_skill_rank(match_id, rating_type, written_at)`,
			`CREATE INDEX IF NOT EXISTS idx_msr_rating_type ON match_skill_rank(rating_type)`,
			`CREATE INDEX IF NOT EXISTS idx_msr_playlist ON match_skill_rank(playlist_group)`,
		},
		ViewSQL: `CREATE OR REPLACE VIEW match_skill_rank_latest AS
			SELECT * FROM match_skill_rank
			QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, rating_type ORDER BY written_at DESC, id DESC) = 1`,
	})
}
