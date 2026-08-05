package migration

// append_only_rebuild.go — helper unique de conversion d'une table en append-only
// (campagne d'éradication du bug DuckDB ART #23046).
//
// **Contexte** : sur DuckDB file-backed, l'enforcement d'une PK/UNIQUE via index
// ART corrompt le heap sous churn (DELETE per-row / UPDATE de colonne indexée /
// ON CONFLICT DO UPDATE) → "Failed to delete all rows from index" → DB FATAL.
// La parade est de rendre les tables d'ÉTAT append-only : PK technique `id`
// (séquence) + colonne d'horloge, écritures = INSERT pur, lecture via une vue
// `<table>_latest` qui ne garde que la dernière version par clé fonctionnelle.
//
// **Ce helper factorise le SWAP de conversion** (legacy → append-only), commun à
// 7 tables : match_skill_rank, player_csr_snapshots, match_csrs, pve_match_stats,
// lusr_component_history (mécanisme written_at) + personal_score_awards,
// match_citations (mécanisme generation_id). Il garantit, pour TOUTES :
//   - swap CTAS TRANSACTIONNEL (BeginTx + rollback intégral sur erreur) ;
//   - garde anti-perte rebuilt==before AVANT le DROP de l'ancienne table ;
//   - recoverOrphan en tête (réparation d'un crash mid-swap antérieur) ;
//   - idempotence (colonne marqueur présente → refresh vue + no-op).
//
// Avant ce helper, 5 de ces conversions étaient une suite de db.ExecContext NON
// transactionnelle qui DROPpait l'ancienne table AVANT toute vérification de
// cardinalité et sans recoverOrphan (perte de données possible sur crash/erreur
// mid-swap). Le helper supprime cette asymétrie de sûreté.
//
// **Exception documentée** : player_match_enrichment n'utilise PAS ce helper —
// sa vue _latest est un merge-on-read par colonne propriétaire (`stage`), pas une
// simple dernière-version ; son swap reste bespoke (steps_player_append_only_match_enrichment.go).

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

// synthWrittenAt — colonne synthétique d'horloge commune aux conversions « simples »
// (mécanisme written_at). Figée au moment de la migration pour les rows existantes.
const synthWrittenAt = "CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP) AS written_at"

// appendOnlyRebuild décrit la conversion append-only d'une table par swap CTAS.
// Le SQL spécifique à chaque table (colonnes synthétiques, defaults, index, vue)
// est passé verbatim ; la mécanique (TX, garde, recoverOrphan, PK) est commune.
type appendOnlyRebuild struct {
	// Table cible (nom logique, identique avant/après le swap).
	Table string
	// IDSeq : séquence backing la PK technique `id`. Requis.
	IDSeq string
	// IDConditional : si true, `id` n'est ajouté que s'il est ABSENT du schéma
	// legacy (cas personal_score_awards, dont d'anciens schémas portent déjà un id).
	// Sinon `id` est toujours synthétisé (l'idempotence garantit qu'on n'arrive ici
	// que sur un legacy sans `id`).
	IDConditional bool
	// ExtraSeqs : séquences supplémentaires créées dans la TX avant le CTAS
	// (ex: la séquence de génération pour le mécanisme generation_id).
	ExtraSeqs []string
	// SyntheticCols : expressions de colonnes synthétiques ajoutées par le CTAS
	// APRÈS les colonnes d'origine. Ex : "CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP) AS written_at" ou
	// "0::BIGINT AS generation_id, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP) AS written_at, FALSE AS is_tombstone".
	// Vide si seul `id` est synthétisé (ex: lusr_component_history préserve computed_at).
	SyntheticCols string
	// MarkerColumn : colonne dont la PRÉSENCE signale « déjà migrée » (idempotence).
	// Défaut "id". Vaut "generation_id" pour les tables générationnelles dont le
	// legacy pouvait déjà porter un `id` (id seul ne discrimine pas l'ancien schéma).
	MarkerColumn string
	// PostSwap : ALTER/INDEX appliqués APRÈS le RENAME (defaults métier + index),
	// verbatim et table-spécifiques. La PK `id` et son DEFAULT sont posés par le
	// helper (communs à toutes) — ne pas les répéter ici.
	PostSwap []string
	// ViewSQL : `CREATE OR REPLACE VIEW <table>_latest ...`. Recréée même sur le
	// chemin idempotent (refresh sûr, CREATE OR REPLACE).
	ViewSQL string
}

func (s appendOnlyRebuild) marker() string {
	if s.MarkerColumn != "" {
		return s.MarkerColumn
	}
	return "id"
}

// recoverOrphanAppendOnly répare l'état laissé par un crash AU MILIEU d'un swap
// append-only : table principale absente + `<table>__appendonly` orpheline. On
// renomme l'orpheline en principale. Idempotent (no-op si la principale existe ou
// si l'orpheline est absente). Générique — remplace les recoverOrphan* par-table.
func recoverOrphanAppendOnly(ctx context.Context, db *sql.DB, table string) error {
	hasMain, err := tableExists(db, table)
	if err != nil {
		return fmt.Errorf("append-only %s: check main: %w", table, err)
	}
	if hasMain {
		return nil
	}
	orphan := table + "__appendonly"
	hasOrphan, err := tableExists(db, orphan)
	if err != nil {
		return fmt.Errorf("append-only %s: check %s: %w", table, orphan, err)
	}
	if !hasOrphan {
		return nil
	}
	slog.WarnContext(ctx, "append-only: orphelin __appendonly (crash mid-swap) — récupération",
		"table", table, "action", "RENAME "+orphan+" -> "+table)
	if _, err := db.ExecContext(ctx, fmt.Sprintf(
		`ALTER TABLE %s__appendonly RENAME TO %s`, table, table)); err != nil {
		return fmt.Errorf("append-only %s: recover orphan: %w", table, err)
	}
	return nil
}

// applyAppendOnlyRebuild : point d'entrée idempotent d'une conversion append-only.
//  1. recoverOrphan (répare un crash mid-swap antérieur)
//  2. table absente → no-op (invariant create-then-migrate des player DB neuves)
//  3. déjà migrée (colonne marqueur présente) → refresh la vue + no-op
//  4. sinon : swap CTAS transactionnel + création de la vue
func applyAppendOnlyRebuild(db *sql.DB, spec appendOnlyRebuild) error {
	ctx := bootCtx()

	if err := recoverOrphanAppendOnly(ctx, db, spec.Table); err != nil {
		return err
	}

	hasTable, err := tableExists(db, spec.Table)
	if err != nil {
		return fmt.Errorf("append-only %s: check table: %w", spec.Table, err)
	}
	if !hasTable {
		return nil
	}

	migrated, err := columnExists(db, spec.Table, spec.marker())
	if err != nil {
		return fmt.Errorf("append-only %s: check marker %s: %w", spec.Table, spec.marker(), err)
	}
	if migrated {
		if err := refreshAppendOnlyView(ctx, db, spec); err != nil {
			return err
		}
		return nil
	}

	if err := rebuildAppendOnlyTx(ctx, db, spec); err != nil {
		return err
	}
	return refreshAppendOnlyView(ctx, db, spec)
}

func refreshAppendOnlyView(ctx context.Context, db *sql.DB, spec appendOnlyRebuild) error {
	if spec.ViewSQL == "" {
		return nil
	}
	if _, err := db.ExecContext(ctx, spec.ViewSQL); err != nil {
		return fmt.Errorf("append-only %s: view: %w", spec.Table, err)
	}
	return nil
}

// rebuildAppendOnlyTx : swap CTAS transactionnel avec garde anti-perte
// (rebuilt==before) et rollback intégral. Les rows existantes deviennent la
// version 0 ; toute écriture future alloue un id (et éventuellement un
// generation_id) supérieur et les supersède via la vue _latest.
func rebuildAppendOnlyTx(ctx context.Context, db *sql.DB, spec appendOnlyRebuild) error {
	cols, err := loadTableColumns(ctx, db, spec.Table)
	if err != nil {
		return fmt.Errorf("append-only %s: enumerate columns: %w", spec.Table, err)
	}
	if len(cols) == 0 {
		return nil
	}

	selectList := buildAppendOnlySelectList(cols, spec)

	var before int
	if err := db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM %s`, spec.Table)).Scan(&before); err != nil {
		return fmt.Errorf("append-only %s: count before: %w", spec.Table, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("append-only %s: begin tx: %w", spec.Table, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// Séquences (id + extras éventuelles) avant le CTAS qui les référence.
	for _, seq := range append([]string{spec.IDSeq}, spec.ExtraSeqs...) {
		if _, err := tx.ExecContext(ctx,
			fmt.Sprintf(`CREATE SEQUENCE IF NOT EXISTS %s START 1`, seq)); err != nil {
			return fmt.Errorf("append-only %s: create sequence %s: %w", spec.Table, seq, err)
		}
	}
	if _, err := tx.ExecContext(ctx,
		fmt.Sprintf(`DROP TABLE IF EXISTS %s__appendonly`, spec.Table)); err != nil {
		return fmt.Errorf("append-only %s: drop stale __appendonly: %w", spec.Table, err)
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(
		`CREATE TABLE %s__appendonly AS SELECT %s FROM %s`,
		spec.Table, selectList, spec.Table)); err != nil {
		return fmt.Errorf("append-only %s: create __appendonly: %w", spec.Table, err)
	}

	var rebuilt int
	if err := tx.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM %s__appendonly`, spec.Table)).Scan(&rebuilt); err != nil {
		return fmt.Errorf("append-only %s: count __appendonly: %w", spec.Table, err)
	}
	if rebuilt != before {
		return fmt.Errorf("append-only %s: swap abandonné, rebuilt=%d != before=%d (rollback, zéro perte)",
			spec.Table, rebuilt, before)
	}

	stmts := make([]string, 0, 4+len(spec.PostSwap))
	stmts = append(stmts,
		fmt.Sprintf(`DROP TABLE %s`, spec.Table),
		fmt.Sprintf(`ALTER TABLE %s__appendonly RENAME TO %s`, spec.Table, spec.Table),
		fmt.Sprintf(`ALTER TABLE %s ADD PRIMARY KEY (id)`, spec.Table),
		fmt.Sprintf(`ALTER TABLE %s ALTER COLUMN id SET DEFAULT nextval('%s')`, spec.Table, spec.IDSeq),
	)
	stmts = append(stmts, spec.PostSwap...)
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("append-only %s: swap step (%s): %w", spec.Table, firstWords(stmt, 3), err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("append-only %s: commit swap: %w", spec.Table, err)
	}
	committed = true

	slog.InfoContext(ctx, "append-only: migration appliquée (ART éradiqué par construction)",
		"table", spec.Table, "rows", before, "columns_preserved", len(cols))
	return nil
}

// buildAppendOnlySelectList construit la clause SELECT du CTAS :
// [nextval('IDSeq') AS id,] <colonnes d'origine> [, <SyntheticCols>].
// L'id est conditionnel si IDConditional ET déjà présent (préservé tel quel).
func buildAppendOnlySelectList(cols []string, spec appendOnlyRebuild) string {
	hasID := false
	for _, c := range cols {
		if c == "id" {
			hasID = true
			break
		}
	}
	selectList := strings.Join(cols, ", ")
	if !spec.IDConditional || !hasID {
		selectList = fmt.Sprintf("nextval('%s') AS id, %s", spec.IDSeq, selectList)
	}
	if spec.SyntheticCols != "" {
		selectList += ", " + spec.SyntheticCols
	}
	return selectList
}
