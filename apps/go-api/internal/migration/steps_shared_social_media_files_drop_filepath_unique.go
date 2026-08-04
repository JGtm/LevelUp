package migration

// steps_shared_social_media_files_drop_filepath_unique.go — éradication ART de la
// contrainte UNIQUE(file_path) sur media_files (shared_social.duckdb) — 2026-06-20.
//
// **Pourquoi** : media_files.file_path est UNIQUE (index ART), MUTÉE par 3 UPDATE :
//   - ops/media.go (insertMediaFile, conversion de format)  : SET file_path, kind WHERE id
//   - ops/media_hls.go (finalizeMediaHLS)                   : SET file_path, hls_path WHERE file_path
//   - ops/media_reconcile.go (reconcileOrphanFiles)         : SET file_path WHERE id
// Muter une colonne couverte par un index ART = vecteur du bug DuckDB amont #23046
// ("Failed to delete all rows from index" → DB FATAL invalidated). shared_social est le
// handle RW PARTAGÉ → 1 FATAL = TOUTE l'app (blast MAX).
//
// **Stratégie** : DuckDB ne sait pas retirer une contrainte UNIQUE via ALTER (pas de
// DROP CONSTRAINT — même limite que pour les PK, cf. rebuild_catalog_fetch_queue). Le
// SEUL moyen = rebuild par swap CTAS. On calque le pattern PROUVÉ EN PROD
// swapMatchParticipantsTx (steps_shared_rebuild_match_participants.go) : swap dans UNE
// transaction (atomique, zéro perte sur crash), garde anti-perte row-count, et
// recoverOrphan en tête. Fixes spécifiques media_files (vérif adversariale 2026-06-20) :
//   - restaurer le DEFAULT nextval('media_files_id_seq') sur id DANS la tx (les writers
//     INSERT omettent id → sans ce default, INSERT écrit NULL dans la PK → indexation cassée) ;
//   - resync de la séquence via CREATE SEQUENCE IF NOT EXISTS START max(id)+1 (si la seq a
//     disparu, éviter une collision id=1) ;
//   - restaurer les autres DEFAULT perdus par le CTAS (created_at, updated_at,
//     discord_notified, indexed_at, file_size ; plus `liked`, colonne retirée du
//     schéma le 2026-08-04 — cf. drop_media_files_liked_columns_v1) ;
//   - DROP idx_mf_kind (kind est muté = surface ART) + recréer idx_mf_player_stem (sur
//     colonnes NON mutées = ART-safe), reconciliation idempotente hors tx.
// La dédup file_path (ex-rôle de UNIQUE) passe en applicatif (SELECT-then-INSERT dans
// insertMediaFile + persistMediaFiles).
//
// **Idempotence** : guard sur la présence de UNIQUE(file_path) (duckdb_constraints,
// vérifié empiriquement sur la DB live). La reconciliation d'index tourne TOUJOURS
// (idempotente). recoverOrphan répare un crash mid-swap antérieur.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

func init() {
	Register(Migration{
		Name:        "media_files_drop_filepath_unique_v1",
		TargetDB:    TargetSharedSocial,
		Description: "Rebuild media_files sans contrainte UNIQUE sur file_path via swap CTAS transactionnel — élimine la surface ART (UPDATE file_path conversion/HLS/reconcile, #23046, blast MAX)",
		ApplySchema: applyMediaFilesDropFilePathUnique,
	})
}

func applyMediaFilesDropFilePathUnique(db *sql.DB) error {
	ctx := bootCtx()

	// 1. Récupération d'un __rebuild orphelin (crash mid-swap antérieur).
	if err := recoverOrphanMediaFiles(ctx, db); err != nil {
		return err
	}

	hasTable, err := tableExists(db, "media_files")
	if err != nil {
		return fmt.Errorf("media_files_drop_unique: check table: %w", err)
	}
	if !hasTable {
		return nil
	}

	// 2. Guard : rebuild si UNIQUE(file_path) existe encore (cas normal) OU si la PK
	//    manque (état dégradé post orphan-recovery : un __rebuild post-CTAS récupéré
	//    n'a ni UNIQUE ni PK → le rebuild restaure PK + DEFAULT id). Sinon no-op.
	hasUnique, err := mediaFilesHasFilePathUnique(ctx, db)
	if err != nil {
		return fmt.Errorf("media_files_drop_unique: check UNIQUE: %w", err)
	}
	hasPK, err := hasPrimaryKey(db, "media_files")
	if err != nil {
		return fmt.Errorf("media_files_drop_unique: check PK: %w", err)
	}
	if hasUnique || !hasPK {
		if err := swapMediaFilesDropUniqueTx(ctx, db); err != nil {
			return err
		}
	}

	// 3. Reconciliation d'index — TOUJOURS exécutée (idempotente), couvre l'état
	//    partiel d'un crash post-commit avant la recréation d'index, et l'état des
	//    DB fraîches. kind est muté → idx_mf_kind interdit ; idx_mf_player_stem
	//    (player_slug, file_stem) est sur colonnes non mutées → ART-safe.
	if _, err := db.ExecContext(ctx, `DROP INDEX IF EXISTS idx_mf_kind`); err != nil {
		return fmt.Errorf("media_files_drop_unique: drop idx_mf_kind: %w", err)
	}
	if _, err := db.ExecContext(ctx,
		`CREATE INDEX IF NOT EXISTS idx_mf_player_stem ON media_files(player_slug, file_stem)`); err != nil {
		return fmt.Errorf("media_files_drop_unique: recreate idx_mf_player_stem: %w", err)
	}
	return nil
}

// mediaFilesHasFilePathUnique détecte la contrainte UNIQUE sur file_path via
// duckdb_constraints() (vérifié empiriquement : la live legacy expose
// "UNIQUE(file_path)" en constraint_type='UNIQUE').
func mediaFilesHasFilePathUnique(ctx context.Context, db *sql.DB) (bool, error) {
	var n int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM duckdb_constraints()
		WHERE table_name = 'media_files' AND constraint_type = 'UNIQUE'
	`).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// recoverOrphanMediaFiles répare un crash mid-swap antérieur : media_files absente
// + media_files__rebuild présente → on promeut le __rebuild. No-op sinon.
func recoverOrphanMediaFiles(ctx context.Context, db *sql.DB) error {
	hasMain, err := tableExists(db, "media_files")
	if err != nil {
		return fmt.Errorf("media_files_drop_unique: check main table: %w", err)
	}
	if hasMain {
		return nil
	}
	hasRebuild, err := tableExists(db, "media_files__rebuild")
	if err != nil {
		return fmt.Errorf("media_files_drop_unique: check __rebuild table: %w", err)
	}
	if !hasRebuild {
		return nil
	}
	slog.WarnContext(ctx, "media_files_drop_unique: __rebuild orphelin détecté (crash mid-swap antérieur) — récupération",
		"action", "RENAME media_files__rebuild -> media_files")
	if _, err := db.ExecContext(ctx,
		`ALTER TABLE media_files__rebuild RENAME TO media_files`); err != nil {
		return fmt.Errorf("media_files_drop_unique: recover orphan __rebuild: %w", err)
	}
	return nil
}

// swapMediaFilesDropUniqueTx effectue le swap CTAS dans UNE transaction (atomique).
// Garde anti-perte : refuse de détruire l'original si le rebuild n'a pas exactement le
// même nombre de rows. Restaure DANS la tx : PK(id), DEFAULT id (nextval), resync seq,
// et les autres DEFAULTs perdus par le CTAS. Crash/échec → rollback intégral, original intact.
func swapMediaFilesDropUniqueTx(ctx context.Context, db *sql.DB) error {
	cols, err := loadTableColumns(ctx, db, "media_files")
	if err != nil {
		return fmt.Errorf("media_files_drop_unique: enumerate columns: %w", err)
	}
	if len(cols) == 0 {
		return nil
	}
	colList := strings.Join(cols, ", ")

	var before int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_files`).Scan(&before); err != nil {
		return fmt.Errorf("media_files_drop_unique: count before: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("media_files_drop_unique: begin swap tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS media_files__rebuild`); err != nil {
		return fmt.Errorf("media_files_drop_unique: drop stale __rebuild: %w", err)
	}
	// CTAS SELECT * : table-scan complet (préserve data + types, dont TIMESTAMPTZ) ;
	// PERD contraintes (UNIQUE — le but) + PK + DEFAULTs (restaurés ci-dessous).
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(
		`CREATE TABLE media_files__rebuild AS SELECT %s FROM media_files`, colList)); err != nil {
		return fmt.Errorf("media_files_drop_unique: create __rebuild: %w", err)
	}
	var rebuilt int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_files__rebuild`).Scan(&rebuilt); err != nil {
		return fmt.Errorf("media_files_drop_unique: count __rebuild: %w", err)
	}
	if rebuilt != before {
		return fmt.Errorf("media_files_drop_unique: swap abandonné, rebuilt=%d != before=%d (rollback, zéro perte)", rebuilt, before)
	}

	// Resync séquence : la live a media_files_id_seq (compteur > max(id), préservé).
	// START max(id)+1 ne sert que si la seq a disparu (évite collision id=1).
	var maxID int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) FROM media_files__rebuild`).Scan(&maxID); err != nil {
		return fmt.Errorf("media_files_drop_unique: max(id): %w", err)
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(
		`CREATE SEQUENCE IF NOT EXISTS media_files_id_seq START %d`, maxID+1)); err != nil {
		return fmt.Errorf("media_files_drop_unique: ensure sequence: %w", err)
	}

	for _, sqlStmt := range []string{
		`DROP TABLE media_files`,
		`ALTER TABLE media_files__rebuild RENAME TO media_files`,
		`ALTER TABLE media_files ADD PRIMARY KEY (id)`,
		// DEFAULT id OBLIGATOIRE : les writers INSERT omettent id (cf. insertMediaFile,
		// persistMediaFiles) → sans ce DEFAULT, INSERT écrit NULL dans la PK NOT NULL.
		`ALTER TABLE media_files ALTER COLUMN id SET DEFAULT nextval('media_files_id_seq')`,
	} {
		if _, err := tx.ExecContext(ctx, sqlStmt); err != nil {
			return fmt.Errorf("media_files_drop_unique: swap step (%s): %w", firstWords(sqlStmt, 3), err)
		}
	}

	// Restaure les autres DEFAULTs perdus par le CTAS (guardés par existence colonne :
	// created_at/updated_at/file_size viennent de migrations align tardives).
	rebuiltCols := make(map[string]struct{}, len(cols))
	for _, c := range cols {
		rebuiltCols[strings.ToLower(c)] = struct{}{}
	}
	for _, d := range []struct{ col, expr string }{
		{"created_at", "CURRENT_TIMESTAMP"},
		{"updated_at", "CURRENT_TIMESTAMP"},
		{"discord_notified", "FALSE"},
		{"indexed_at", "NOW()"},
		{"file_size", "0"},
	} {
		if _, ok := rebuiltCols[d.col]; !ok {
			continue
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(
			`ALTER TABLE media_files ALTER COLUMN %s SET DEFAULT %s`, d.col, d.expr)); err != nil {
			return fmt.Errorf("media_files_drop_unique: restore default %s: %w", d.col, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("media_files_drop_unique: commit swap: %w", err)
	}
	committed = true

	slog.InfoContext(ctx, "media_files_drop_unique: rebuild appliqué (contrainte UNIQUE sur file_path ART éradiquée)",
		"rows", before, "columns_preserved", len(cols), "max_id", maxID)
	return nil
}
