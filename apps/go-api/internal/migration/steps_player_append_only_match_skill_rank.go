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
	"fmt"
	"log/slog"
	"strings"
)

func init() {
	Register(Migration{
		Name:        "player_append_only_match_skill_rank_v1",
		TargetDB:    TargetPlayer,
		Description: "Rebuild match_skill_rank en append-only (id PK + written_at + vue latest) — élimine bug ART par construction",
		ApplySchema: applyAppendOnlyMatchSkillRank,
	})
}

func applyAppendOnlyMatchSkillRank(db *sql.DB) error {
	ctx := bootCtx()

	hasTable, err := tableExists(db, "match_skill_rank")
	if err != nil {
		return fmt.Errorf("append-only msr: check table: %w", err)
	}
	if !hasTable {
		// Table jamais initialisée (player DB neuve avant steps_player.go).
		// Pas grave — la création initiale viendra ensuite, et la prochaine
		// run de cette migration boot-time la transformera. Pour rester
		// simple, on no-op ici et on laisse la création initiale produire
		// l'ancien schéma — la migration s'applique à la rotation suivante.
		// Acceptable car create-then-migrate est l'invariant des autres
		// migrations player_*.
		return nil
	}

	// Idempotence : si la colonne `id` existe déjà, on a déjà migré.
	hasIDCol, err := columnExists(db, "match_skill_rank", "id")
	if err != nil {
		return fmt.Errorf("append-only msr: check id column: %w", err)
	}
	if hasIDCol {
		return nil
	}

	cols, err := loadTableColumns(ctx, db, "match_skill_rank")
	if err != nil {
		return fmt.Errorf("append-only msr: enumerate columns: %w", err)
	}
	if len(cols) == 0 {
		return nil
	}
	colList := strings.Join(cols, ", ")

	var before int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_skill_rank`).Scan(&before); err != nil {
		return fmt.Errorf("append-only msr: count before: %w", err)
	}

	// La séquence et les indexes ne nécessitent pas de drop ; ils sont
	// idempotents via IF NOT EXISTS.
	stmts := []string{
		`CREATE SEQUENCE IF NOT EXISTS msr_seq START 1`,
		`DROP TABLE IF EXISTS match_skill_rank__appendonly`,
		// CTAS : reconstruire avec id (généré) + written_at (now() au moment
		// de la migration → toutes les rows existantes auront le même
		// written_at, ce qui est cohérent : elles représentent l'état au
		// moment de la bascule append-only).
		fmt.Sprintf(`
			CREATE TABLE match_skill_rank__appendonly AS
			SELECT
				nextval('msr_seq') AS id,
				%s,
				CURRENT_TIMESTAMP AS written_at
			FROM match_skill_rank
		`, colList),
		`DROP TABLE match_skill_rank`,
		`ALTER TABLE match_skill_rank__appendonly RENAME TO match_skill_rank`,
		`ALTER TABLE match_skill_rank ADD PRIMARY KEY (id)`,
		`ALTER TABLE match_skill_rank ALTER COLUMN id SET DEFAULT nextval('msr_seq')`,
		`ALTER TABLE match_skill_rank ALTER COLUMN written_at SET DEFAULT now()`,
		`CREATE INDEX IF NOT EXISTS idx_msr_match_lookup ON match_skill_rank(match_id, rating_type, written_at)`,
		`CREATE INDEX IF NOT EXISTS idx_msr_rating_type ON match_skill_rank(rating_type)`,
		`CREATE INDEX IF NOT EXISTS idx_msr_playlist ON match_skill_rank(playlist_group)`,
		`CREATE OR REPLACE VIEW match_skill_rank_latest AS
			SELECT * FROM match_skill_rank
			QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, rating_type ORDER BY written_at DESC, id DESC) = 1`,
	}
	for _, sqlStmt := range stmts {
		if _, err := db.ExecContext(ctx, sqlStmt); err != nil {
			return fmt.Errorf("append-only msr: step (%s): %w",
				firstWords(sqlStmt, 3), err)
		}
	}

	var after int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_skill_rank`).Scan(&after); err != nil {
		return fmt.Errorf("append-only msr: count after: %w", err)
	}

	slog.InfoContext(ctx, "append-only match_skill_rank: migration appliquée (ART eradication phase 2.B)",
		"rows_before", before,
		"rows_after", after,
		"columns_preserved", len(cols),
	)
	return nil
}
