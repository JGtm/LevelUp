package migration

// steps_player_append_only_personal_score_awards.go — éradication ART de
// personal_score_awards (player DB) — Phase 2 campagne #23046 (2026-06-21).
//
// **Pourquoi** : InsertPersonalScoreAwards (sync/writes.go) faisait
// `DELETE FROM personal_score_awards WHERE match_id=? AND xuid=?` puis INSERT
// batch en TX (REPLACE-semantics). Le DELETE retire N lignes des 4 index ART
// (idx_psa_match/xuid/category/match_xuid) = vecteur DuckDB #23046 sur le
// chemin de sync LIVE (engine_process_match + convergePSA). Jumeau direct du
// DELETE LUSR match_skill_rank corrigé en Phase 1.
//
// **Stratégie append-only GÉNÉRATION** : la sémantique métier est « remplacer
// l'ENSEMBLE des awards d'un (match,xuid) par la nouvelle extraction ». On la
// préserve sans DELETE : chaque appel d'écriture alloue UN generation_id
// (séquence psa_generation_seq, partagé par toutes les rows du batch) et
// INSÈRE pur. La vue personal_score_awards_latest ne garde, par (match_id,xuid),
// QUE les lignes de la génération MAX (DENSE_RANK — pas ROW_NUMBER : on veut
// TOUS les awards de la dernière extraction, jamais dédup par award_name).
//
// **Clear (extraction vide)** : backfill_personal_scores.go appelle
// InsertPersonalScoreAwards(..., nil) pour « vérifié, zéro award ». En
// append-only un INSERT vide ne supprimerait rien → la génération précédente
// resterait visible. On INSÈRE donc une row TOMBSTONE (is_tombstone=TRUE) avec
// le nouveau generation_id ; la vue sélectionne cette génération puis filtre les
// tombstones → (match,xuid) ne retourne plus rien = cleared.
//
// **Migration TRANSACTIONNELLE** (calquée player_append_only_match_enrichment) :
// PSA n'est re-dérivable que par re-fetch API coûteux → swap CTAS sous BeginTx +
// garde anti-perte rebuilt==before + recoverOrphan. Idempotence :
// columnExists('generation_id') → refresh vue + no-op.

import (
	"database/sql"
)

func init() {
	Register(Migration{
		Name:        "player_append_only_personal_score_awards_v1",
		TargetDB:    TargetPlayer,
		Description: "Rebuild personal_score_awards en append-only (generation_id + written_at + tombstone + vue latest) — élimine DELETE+INSERT sur index ART (#23046)",
		ApplySchema: applyAppendOnlyPersonalScoreAwards,
	})
}

// psaLatestViewSQL — vue de lecture « dernière génération par (match_id,xuid) »,
// tombstones exclus. DENSE_RANK conserve TOUTES les rows de la génération MAX.
const psaLatestViewSQL = `CREATE OR REPLACE VIEW personal_score_awards_latest AS
SELECT * EXCLUDE (rk) FROM (
	SELECT *, DENSE_RANK() OVER (
		PARTITION BY match_id, xuid ORDER BY generation_id DESC
	) AS rk
	FROM personal_score_awards
)
WHERE rk = 1 AND NOT is_tombstone`

// EnsurePersonalScoreAwardsAppendOnly convertit personal_score_awards en
// append-only (generation_id + written_at + is_tombstone) et (re)crée la vue
// _latest. Idempotent. Exposé pour OpenPlayerDB + les fixtures de test.
func EnsurePersonalScoreAwardsAppendOnly(db *sql.DB) error {
	return applyAppendOnlyPersonalScoreAwards(db)
}

// applyAppendOnlyPersonalScoreAwards délègue au helper commun (cf. append_only_rebuild.go).
// Mécanisme generation_id (remplace l'ENSEMBLE des awards d'un (match,xuid) par
// génération, via DENSE_RANK) + tombstone pour le cas « extraction vide ». id
// conditionnel car d'anciens schémas PSA portent déjà un id ; marqueur d'idempotence
// = generation_id (id seul ne discrimine pas l'ancien schéma).
func applyAppendOnlyPersonalScoreAwards(db *sql.DB) error {
	return applyAppendOnlyRebuild(db, appendOnlyRebuild{
		Table:         "personal_score_awards",
		IDSeq:         "personal_score_awards_id_seq",
		IDConditional: true,
		ExtraSeqs:     []string{"psa_generation_seq"},
		SyntheticCols: "0::BIGINT AS generation_id, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP) AS written_at, FALSE AS is_tombstone",
		MarkerColumn:  "generation_id",
		// idx_psa_xuid N'EST PLUS recréé après le swap (décision 2026-08-05, miroir
		// d'idx_career_xuid) : xuid est quasi constant dans une player DB (une DB = un
		// joueur) → sélectivité nulle, coût d'écriture pur. Les DB qui le portent encore
		// le perdent via le step drop_psa_xuid_art_index_v1 (steps_player_schema_authority.go).
		// idx_psa_match_xuid non plus (décision 2026-08-05, même arbitrage) : ce PostSwap
		// était sa SEULE autorité (absent de PlayerPersonalScoreAwardsDDL → divergence
		// latente entre DB fraîche et DB convertie), et il est un pur préfixe
		// d'idx_psa_gen(match_id, xuid, generation_id) → redondant. Convergence des DB
		// existantes : step drop_psa_match_xuid_art_index_v1.
		PostSwap: []string{
			`ALTER TABLE personal_score_awards ALTER COLUMN written_at SET DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)`,
			`ALTER TABLE personal_score_awards ALTER COLUMN is_tombstone SET DEFAULT FALSE`,
			`CREATE INDEX IF NOT EXISTS idx_psa_match     ON personal_score_awards(match_id)`,
			`CREATE INDEX IF NOT EXISTS idx_psa_category  ON personal_score_awards(award_category)`,
			`CREATE INDEX IF NOT EXISTS idx_psa_gen       ON personal_score_awards(match_id, xuid, generation_id)`,
		},
		ViewSQL: psaLatestViewSQL,
	})
}
