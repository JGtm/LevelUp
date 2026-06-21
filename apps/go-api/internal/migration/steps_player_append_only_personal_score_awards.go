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
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
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

func applyAppendOnlyPersonalScoreAwards(db *sql.DB) error {
	ctx := bootCtx()

	if err := recoverOrphanPSA(ctx, db); err != nil {
		return err
	}

	hasTable, err := tableExists(db, "personal_score_awards")
	if err != nil {
		return fmt.Errorf("append-only psa: check table: %w", err)
	}
	if !hasTable {
		return nil
	}

	hasGen, err := columnExists(db, "personal_score_awards", "generation_id")
	if err != nil {
		return fmt.Errorf("append-only psa: check generation_id column: %w", err)
	}
	if hasGen {
		// Déjà append-only : (ré)assurer la vue (idempotent).
		if _, err := db.ExecContext(ctx, psaLatestViewSQL); err != nil {
			return fmt.Errorf("append-only psa: refresh view: %w", err)
		}
		return nil
	}

	if err := swapPSAAppendOnlyTx(ctx, db); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, psaLatestViewSQL); err != nil {
		return fmt.Errorf("append-only psa: create view: %w", err)
	}
	return nil
}

// recoverOrphanPSA répare un crash mid-swap (table absente + __appendonly présent).
func recoverOrphanPSA(ctx context.Context, db *sql.DB) error {
	hasMain, err := tableExists(db, "personal_score_awards")
	if err != nil {
		return fmt.Errorf("append-only psa: check main: %w", err)
	}
	if hasMain {
		return nil
	}
	hasRebuild, err := tableExists(db, "personal_score_awards__appendonly")
	if err != nil {
		return fmt.Errorf("append-only psa: check __appendonly: %w", err)
	}
	if !hasRebuild {
		return nil
	}
	slog.WarnContext(ctx, "append-only psa: __appendonly orphelin (crash mid-swap) — récupération",
		"action", "RENAME personal_score_awards__appendonly -> personal_score_awards")
	if _, err := db.ExecContext(ctx,
		`ALTER TABLE personal_score_awards__appendonly RENAME TO personal_score_awards`); err != nil {
		return fmt.Errorf("append-only psa: recover orphan: %w", err)
	}
	return nil
}

// swapPSAAppendOnlyTx : swap CTAS transactionnel. Les rows existantes deviennent
// la génération 0 (toute nouvelle écriture alloue generation_id >= 1 et la
// supersède). Garde anti-perte rebuilt==before. Rollback intégral sur erreur.
func swapPSAAppendOnlyTx(ctx context.Context, db *sql.DB) error {
	cols, err := loadTableColumns(ctx, db, "personal_score_awards")
	if err != nil {
		return fmt.Errorf("append-only psa: enumerate columns: %w", err)
	}
	if len(cols) == 0 {
		return nil
	}
	colList := strings.Join(cols, ", ")

	// Le legacy RÉEL peut ne PAS avoir de colonne id (PSA créé par un ancien schéma /
	// pipeline Python sans PK technique). Si absente on l'ajoute via la séquence ;
	// si présente on la préserve telle quelle.
	hasID := false
	for _, c := range cols {
		if c == "id" {
			hasID = true
			break
		}
	}
	selectList := colList
	if !hasID {
		selectList = "nextval('personal_score_awards_id_seq') AS id, " + colList
	}

	var before int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM personal_score_awards`).Scan(&before); err != nil {
		return fmt.Errorf("append-only psa: count before: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("append-only psa: begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `CREATE SEQUENCE IF NOT EXISTS psa_generation_seq START 1`); err != nil {
		return fmt.Errorf("append-only psa: create generation sequence: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `CREATE SEQUENCE IF NOT EXISTS personal_score_awards_id_seq START 1`); err != nil {
		return fmt.Errorf("append-only psa: create id sequence: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS personal_score_awards__appendonly`); err != nil {
		return fmt.Errorf("append-only psa: drop stale __appendonly: %w", err)
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE personal_score_awards__appendonly AS
		SELECT %s,
		       0::BIGINT AS generation_id,
		       CURRENT_TIMESTAMP AS written_at,
		       FALSE AS is_tombstone
		FROM personal_score_awards`, selectList)); err != nil {
		return fmt.Errorf("append-only psa: create __appendonly: %w", err)
	}
	var rebuilt int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM personal_score_awards__appendonly`).Scan(&rebuilt); err != nil {
		return fmt.Errorf("append-only psa: count __appendonly: %w", err)
	}
	if rebuilt != before {
		return fmt.Errorf("append-only psa: swap abandonné, rebuilt=%d != before=%d (rollback, zéro perte)", rebuilt, before)
	}
	for _, stmt := range []string{
		`DROP TABLE personal_score_awards`,
		`ALTER TABLE personal_score_awards__appendonly RENAME TO personal_score_awards`,
		`ALTER TABLE personal_score_awards ADD PRIMARY KEY (id)`,
		`ALTER TABLE personal_score_awards ALTER COLUMN id SET DEFAULT nextval('personal_score_awards_id_seq')`,
		`ALTER TABLE personal_score_awards ALTER COLUMN written_at SET DEFAULT now()`,
		`ALTER TABLE personal_score_awards ALTER COLUMN is_tombstone SET DEFAULT FALSE`,
		`CREATE INDEX IF NOT EXISTS idx_psa_match     ON personal_score_awards(match_id)`,
		`CREATE INDEX IF NOT EXISTS idx_psa_xuid      ON personal_score_awards(xuid)`,
		`CREATE INDEX IF NOT EXISTS idx_psa_category  ON personal_score_awards(award_category)`,
		`CREATE INDEX IF NOT EXISTS idx_psa_match_xuid ON personal_score_awards(match_id, xuid)`,
		`CREATE INDEX IF NOT EXISTS idx_psa_gen       ON personal_score_awards(match_id, xuid, generation_id)`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("append-only psa: swap step (%s): %w", firstWords(stmt, 3), err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("append-only psa: commit swap: %w", err)
	}
	committed = true

	slog.InfoContext(ctx, "append-only personal_score_awards: migration appliquée (ART éradiqué)",
		"rows", before, "columns_preserved", len(cols))
	return nil
}
