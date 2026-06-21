// Package migration — steps_shared_append_only_match_csrs.go :
// Phase 2.F du plan d'éradication ART (cf. .ai/PLAN_LUSR_ART_HOME_CRASH.md).
//
// Transforme `shared.match_csrs` en table append-only et crée la vue
// `match_csrs_latest` pour la lecture "version courante".
//
// **Pourquoi** : l'ancienne PK composite `(match_id, xuid)` forçait les
// `INSERT ... ON CONFLICT DO UPDATE` (DELETE+INSERT côté moteur DuckDB)
// — pattern reproduit comme à risque ART par csr_art_repro_test.go
// (19/20 workers crashent sous concurrence).
//
// Nouveau schéma :
//   - PK technique `id BIGINT` (auto via séquence `mcsrs_seq`)
//   - Colonne `written_at TIMESTAMP DEFAULT now()`
//   - Index `(match_id, xuid, written_at)` pour les lookups
//   - Vue `match_csrs_latest` : 1 row par (match_id, xuid), la plus récente
//
// Toutes les écritures futures sont des INSERT purs. Le bug ART devient
// impossible par construction sur cette table.
//
// **Idempotence** : check `columnExists(id)` → no-op si déjà appliquée.

package migration

import (
	"database/sql"
)

func init() {
	Register(Migration{
		Name:        "shared_append_only_match_csrs_v1",
		TargetDB:    TargetShared,
		Description: "Rebuild shared.match_csrs en append-only (id PK + written_at + vue latest) — élimine bug ART par construction",
		ApplySchema: applyAppendOnlyMatchCSRs,
	})
}

// applyAppendOnlyMatchCSRs délègue au helper commun (cf. append_only_rebuild.go).
// Mécanisme written_at (dernière version par match_id+xuid).
func applyAppendOnlyMatchCSRs(db *sql.DB) error {
	return applyAppendOnlyRebuild(db, appendOnlyRebuild{
		Table:         "match_csrs",
		IDSeq:         "mcsrs_seq",
		SyntheticCols: "CURRENT_TIMESTAMP AS written_at",
		PostSwap: []string{
			`ALTER TABLE match_csrs ALTER COLUMN written_at SET DEFAULT now()`,
			`CREATE INDEX IF NOT EXISTS idx_match_csrs_lookup ON match_csrs(match_id, xuid, written_at)`,
			`CREATE INDEX IF NOT EXISTS idx_match_csrs_xuid    ON match_csrs(xuid)`,
			`CREATE INDEX IF NOT EXISTS idx_match_csrs_season  ON match_csrs(season_id)`,
			`CREATE INDEX IF NOT EXISTS idx_match_csrs_match   ON match_csrs(match_id)`,
		},
		ViewSQL: `CREATE OR REPLACE VIEW match_csrs_latest AS
			SELECT * FROM match_csrs
			QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, xuid ORDER BY written_at DESC, id DESC) = 1`,
	})
}
