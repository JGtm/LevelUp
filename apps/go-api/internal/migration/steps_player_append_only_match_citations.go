package migration

// steps_player_append_only_match_citations.go — éradication ART de
// match_citations (player DB) — Phase 2 campagne #23046 (2026-06-21).
//
// **Pourquoi** : BackfillMatchCitations faisait `DELETE FROM match_citations
// WHERE match_id=?` (deleteCitationForMatch) avant réécriture, sur une table à
// PK composite (match_id, citation_name_norm). Le DELETE per-match retire N
// lignes de l'index PK ART = vecteur DuckDB #23046 sur le chemin post-sync +
// heal convergent. Le recompute des citations EST soustractif (capLeafDelta peut
// faire décroître/disparaître une citation) → le DELETE est LOAD-BEARING (pas
// supprimable). La forme correcte est donc append-only générationnel.
//
// **Stratégie append-only GÉNÉRATION** : PK composite remplacée par PK technique
// id (séquence) ; chaque réécriture d'un match alloue UN generation_id (séquence
// match_citations_generation_seq, partagé par toutes les rows du match) en INSERT
// pur. La vue match_citations_latest ne garde, par match_id, QUE les lignes de la
// génération MAX (DENSE_RANK). Le cas « 0 citation » est déjà géré par la row
// sentinelle '_processed' que writeCitations insère (pas de tombstone séparé) :
// elle fait partie de la génération et les readers de valeurs la filtrent déjà.
//
// **Migration TRANSACTIONNELLE** (calquée PSA/PME). Idempotence :
// columnExists('generation_id').

import (
	"database/sql"
)

func init() {
	Register(Migration{
		Name:        "player_append_only_match_citations_v1",
		TargetDB:    TargetPlayer,
		Description: "Rebuild match_citations en append-only (id PK + generation_id + vue latest par match) — élimine DELETE+PK composite ART (#23046)",
		ApplySchema: applyAppendOnlyMatchCitations,
	})
}

// mcLatestViewSQL — vue de lecture « dernière génération par match_id »
// (DENSE_RANK conserve TOUTES les rows de la génération MAX, sentinel '_processed' inclus).
const mcLatestViewSQL = `CREATE OR REPLACE VIEW match_citations_latest AS
SELECT * EXCLUDE (rk) FROM (
	SELECT *, DENSE_RANK() OVER (PARTITION BY match_id ORDER BY generation_id DESC) AS rk
	FROM match_citations
)
WHERE rk = 1`

// EnsureMatchCitationsAppendOnly convertit match_citations en append-only
// (generation_id + written_at) et (re)crée la vue _latest. Idempotent. Exposé
// pour OpenPlayerDB + les fixtures de test.
func EnsureMatchCitationsAppendOnly(db *sql.DB) error {
	return applyAppendOnlyMatchCitations(db)
}

// applyAppendOnlyMatchCitations délègue au helper commun (cf. append_only_rebuild.go).
// Mécanisme generation_id (remplace l'ENSEMBLE des citations d'un match par
// génération, via DENSE_RANK — la sentinelle '_processed' fait partie de la
// génération). Drop la PK composite (match_id, citation_name_norm) au profit d'un id
// technique ; marqueur d'idempotence = generation_id.
func applyAppendOnlyMatchCitations(db *sql.DB) error {
	return applyAppendOnlyRebuild(db, appendOnlyRebuild{
		Table:         "match_citations",
		IDSeq:         "match_citations_id_seq",
		ExtraSeqs:     []string{"match_citations_generation_seq"},
		SyntheticCols: "0::BIGINT AS generation_id, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP) AS written_at",
		MarkerColumn:  "generation_id",
		PostSwap: []string{
			`ALTER TABLE match_citations ALTER COLUMN written_at SET DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)`,
			`CREATE INDEX IF NOT EXISTS idx_mc_match_gen ON match_citations(match_id, generation_id)`,
		},
		ViewSQL: mcLatestViewSQL,
	})
}
