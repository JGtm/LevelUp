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
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
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

func applyAppendOnlyMatchCitations(db *sql.DB) error {
	ctx := bootCtx()

	if err := recoverOrphanMatchCitations(ctx, db); err != nil {
		return err
	}

	hasTable, err := tableExists(db, "match_citations")
	if err != nil {
		return fmt.Errorf("append-only mc: check table: %w", err)
	}
	if !hasTable {
		return nil
	}

	hasGen, err := columnExists(db, "match_citations", "generation_id")
	if err != nil {
		return fmt.Errorf("append-only mc: check generation_id column: %w", err)
	}
	if hasGen {
		if _, err := db.ExecContext(ctx, mcLatestViewSQL); err != nil {
			return fmt.Errorf("append-only mc: refresh view: %w", err)
		}
		return nil
	}

	if err := swapMatchCitationsAppendOnlyTx(ctx, db); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, mcLatestViewSQL); err != nil {
		return fmt.Errorf("append-only mc: create view: %w", err)
	}
	return nil
}

func recoverOrphanMatchCitations(ctx context.Context, db *sql.DB) error {
	hasMain, err := tableExists(db, "match_citations")
	if err != nil {
		return fmt.Errorf("append-only mc: check main: %w", err)
	}
	if hasMain {
		return nil
	}
	hasRebuild, err := tableExists(db, "match_citations__appendonly")
	if err != nil {
		return fmt.Errorf("append-only mc: check __appendonly: %w", err)
	}
	if !hasRebuild {
		return nil
	}
	slog.WarnContext(ctx, "append-only mc: __appendonly orphelin (crash mid-swap) — récupération",
		"action", "RENAME match_citations__appendonly -> match_citations")
	if _, err := db.ExecContext(ctx,
		`ALTER TABLE match_citations__appendonly RENAME TO match_citations`); err != nil {
		return fmt.Errorf("append-only mc: recover orphan: %w", err)
	}
	return nil
}

// swapMatchCitationsAppendOnlyTx : swap CTAS transactionnel. Drop la PK composite
// (match_id, citation_name_norm), pose un id technique + generation_id (legacy=0)
// + written_at. Garde anti-perte rebuilt==before. Rollback intégral sur erreur.
func swapMatchCitationsAppendOnlyTx(ctx context.Context, db *sql.DB) error {
	cols, err := loadTableColumns(ctx, db, "match_citations")
	if err != nil {
		return fmt.Errorf("append-only mc: enumerate columns: %w", err)
	}
	if len(cols) == 0 {
		return nil
	}
	colList := strings.Join(cols, ", ")

	var before int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM match_citations`).Scan(&before); err != nil {
		return fmt.Errorf("append-only mc: count before: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("append-only mc: begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	for _, stmt := range []string{
		`CREATE SEQUENCE IF NOT EXISTS match_citations_id_seq START 1`,
		`CREATE SEQUENCE IF NOT EXISTS match_citations_generation_seq START 1`,
		`DROP TABLE IF EXISTS match_citations__appendonly`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("append-only mc: prep (%s): %w", firstWords(stmt, 3), err)
		}
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE match_citations__appendonly AS
		SELECT nextval('match_citations_id_seq') AS id,
		       %s,
		       0::BIGINT AS generation_id,
		       CURRENT_TIMESTAMP AS written_at
		FROM match_citations`, colList)); err != nil {
		return fmt.Errorf("append-only mc: create __appendonly: %w", err)
	}
	var rebuilt int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM match_citations__appendonly`).Scan(&rebuilt); err != nil {
		return fmt.Errorf("append-only mc: count __appendonly: %w", err)
	}
	if rebuilt != before {
		return fmt.Errorf("append-only mc: swap abandonné, rebuilt=%d != before=%d (rollback, zéro perte)", rebuilt, before)
	}
	for _, stmt := range []string{
		`DROP TABLE match_citations`,
		`ALTER TABLE match_citations__appendonly RENAME TO match_citations`,
		`ALTER TABLE match_citations ADD PRIMARY KEY (id)`,
		`ALTER TABLE match_citations ALTER COLUMN id SET DEFAULT nextval('match_citations_id_seq')`,
		`ALTER TABLE match_citations ALTER COLUMN written_at SET DEFAULT now()`,
		`CREATE INDEX IF NOT EXISTS idx_mc_match_gen ON match_citations(match_id, generation_id)`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("append-only mc: swap step (%s): %w", firstWords(stmt, 3), err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("append-only mc: commit swap: %w", err)
	}
	committed = true

	slog.InfoContext(ctx, "append-only match_citations: migration appliquée (ART éradiqué)",
		"rows", before, "columns_preserved", len(cols))
	return nil
}
