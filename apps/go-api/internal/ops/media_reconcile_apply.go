package ops

import (
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
)

// Les écritures de la passe ne touchent ni file_stem, ni player_slug, ni
// created_at (colonnes indexées : un UPDATE y expose au bug ART DuckDB #23645).
// Chacune porte le même garde : une ligne passée supprimée entre la lecture et
// l'écriture n'est pas modifiée, et n'affecte donc aucune ligne.
const (
	sqlReconcileNotDeleted = "COALESCE(status, '') <> '" + domain.MediaStatusDeleted + "'"

	sqlRelocate = `UPDATE media_files SET source_path = ?, source_size = ?, source_mtime_ns = ?,
		file_name = ?, file_ext = ?, file_hash = ?,
		status = CASE WHEN status = '` + domain.MediaStatusMissing + `' THEN NULL ELSE status END`

	sqlAttachSource = `UPDATE media_files SET source_path = ?, source_size = ?, source_mtime_ns = ?
		WHERE id = ? AND ` + sqlReconcileNotDeleted

	sqlRefreshSource = `UPDATE media_files SET source_size = ?, source_mtime_ns = ?, file_hash = ?
		WHERE id = ? AND ` + sqlReconcileNotDeleted

	sqlClearSource = `UPDATE media_files SET source_path = NULL, source_size = NULL, source_mtime_ns = NULL
		WHERE id = ? AND ` + sqlReconcileNotDeleted

	sqlRevert = `UPDATE media_files SET file_path = source_path, hls_path = NULL, transcode_status = NULL
		WHERE id = ? AND ` + sqlReconcileNotDeleted

	sqlRestore = `UPDATE media_files SET status = NULL
		WHERE id = ? AND status = '` + domain.MediaStatusMissing + `'`

	sqlMarkMissing = `UPDATE media_files SET status = '` + domain.MediaStatusMissing + `'
		WHERE id = ? AND COALESCE(status, '') NOT IN ('` +
		domain.MediaStatusDeleted + `', '` + domain.MediaStatusMissing + `')`
)

// apply (étape 8) écrit toutes les écritures prévues dans une seule transaction.
// Une passe sans écriture n'en ouvre aucune. Les compteurs ne sont reportés dans le
// résultat qu'après le commit.
func (r *reconciler) apply() error {
	if len(r.ops) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(r.ctx, nil)
	if err != nil {
		return fmt.Errorf("reconcile: ouverture de la transaction: %w", err)
	}
	var done ReconcileResult
	for _, op := range r.ops {
		if err := r.applyOp(tx, op, &done); err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				slog.ErrorContext(r.ctx, "reconcile: annulation de la transaction échouée", "err", rbErr)
			}
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("reconcile: validation de la transaction: %w", err)
	}
	r.res.Relinked, r.res.Converted, r.res.Backfilled = done.Relinked, done.Converted, done.Backfilled
	r.res.MarkedMissing, r.res.Restored = done.MarkedMissing, done.Restored
	r.res.Reverted, r.res.SourcesCleared = done.Reverted, done.SourcesCleared
	r.res.writes = len(r.ops)
	slog.InfoContext(r.ctx, "reconcile: médias alignés sur le disque",
		"player", r.in.PlayerSlug, "relinked", done.Relinked, "converted", done.Converted,
		"backfilled", done.Backfilled, "marked_missing", done.MarkedMissing,
		"restored", done.Restored, "reverted", done.Reverted, "sources_cleared", done.SourcesCleared)
	return nil
}

// applyOp exécute une écriture. Une écriture qui n'affecte aucune ligne ne compte
// pas et ne déclenche aucun transfert de likes.
func (r *reconciler) applyOp(tx *sql.Tx, op reconcileOp, done *ReconcileResult) error {
	res, err := r.execOp(tx, op)
	if err != nil {
		return fmt.Errorf("reconcile: écriture de la ligne %d: %w", op.id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("reconcile: lignes affectées (ligne %d): %w", op.id, err)
	}
	if n == 0 {
		return nil
	}
	switch op.kind {
	case opRelocate:
		if op.converted {
			done.Converted++
		} else {
			done.Relinked++
		}
		if !op.hasHLS {
			return r.transferLikes(tx, op.oldFile, op.path)
		}
	case opAttachSource:
		done.Backfilled++
	case opClearSource:
		done.SourcesCleared++
	case opRevert:
		done.Reverted++
		return r.transferLikes(tx, op.oldFile, op.path)
	case opRestore:
		done.Restored++
	case opMarkMissing:
		done.MarkedMissing++
	}
	return nil
}

func (r *reconciler) transferLikes(tx *sql.Tx, oldPath, newPath string) error {
	if err := platform_duckdb.TransferMediaLikes(r.ctx, tx, oldPath, newPath); err != nil {
		return fmt.Errorf("reconcile: %w", err)
	}
	return nil
}

// execOp envoie l'UPDATE du genre d'écriture.
func (r *reconciler) execOp(tx *sql.Tx, op reconcileOp) (sql.Result, error) {
	switch op.kind {
	case opRelocate:
		q := sqlRelocate
		args := []any{op.path, op.size, op.mtime, op.name, op.ext, op.hash}
		if !op.hasHLS {
			q += `, file_path = ?`
			args = append(args, op.path)
		}
		q += ` WHERE id = ? AND ` + sqlReconcileNotDeleted
		return tx.ExecContext(r.ctx, q, append(args, op.id)...)
	case opAttachSource:
		return tx.ExecContext(r.ctx, sqlAttachSource, op.path, op.size, op.mtime, op.id)
	case opRefreshSource:
		return tx.ExecContext(r.ctx, sqlRefreshSource, op.size, op.mtime, op.hash, op.id)
	case opClearSource:
		return tx.ExecContext(r.ctx, sqlClearSource, op.id)
	case opRevert:
		return tx.ExecContext(r.ctx, sqlRevert, op.id)
	case opRestore:
		return tx.ExecContext(r.ctx, sqlRestore, op.id)
	case opMarkMissing:
		return tx.ExecContext(r.ctx, sqlMarkMissing, op.id)
	}
	return nil, fmt.Errorf("genre d'écriture inconnu: %d", op.kind)
}
