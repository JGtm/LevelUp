package ops

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
)

// conversionUpdate décrit la reprise d'une fiche par le fichier d'un autre format.
type conversionUpdate struct {
	id         int64
	oldPath    string // file_path de la fiche avant la reprise
	storedPath string
	baseName   string
	ext        string
	hash       string
	kind       string
	win        captureTimes
	start, end *time.Time
	source     *string
	srcSize    sql.NullInt64
	srcMtime   sql.NullInt64
}

// updateConvertedRow reprend la fiche pour le nouveau fichier et lui transfère ses
// likes, dans une même transaction : les likes sont indexés par file_path, qui change.
// La durée suit le nouveau fichier (COALESCE : NULL garde l'ancienne). La fenêtre de
// capture n'est réécrite que si son heure vient du nom. Une fiche supprimée dont le
// fichier est re-déposé redevient visible (le CASE : écrire NULL sans condition
// dégraderait une fiche 'active').
func updateConvertedRow(ctx context.Context, db *sql.DB, u conversionUpdate) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("reprise de la fiche %d: ouverture de la transaction: %w", u.id, err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE media_files
		SET file_path = ?, file_name = ?, file_ext = ?, file_hash = ?, kind = ?,
			duration_seconds = COALESCE(?, duration_seconds),
			capture_start_utc = COALESCE(?, capture_start_utc),
			capture_end_utc = COALESCE(?, capture_end_utc),
			capture_time_source = COALESCE(?, capture_time_source),
			source_path = ?, source_size = ?, source_mtime_ns = ?,
			status = CASE WHEN COALESCE(status, '') = ? THEN NULL ELSE status END
		WHERE id = ?
	`, u.storedPath, u.baseName, u.ext, u.hash, u.kind, u.win.DurationSec, u.start, u.end, u.source,
		u.storedPath, u.srcSize, u.srcMtime, domain.MediaStatusDeleted, u.id); err != nil {
		return rollbackWith(ctx, tx, fmt.Errorf("reprise de la fiche %d: %w", u.id, err))
	}
	if err := platform_duckdb.TransferMediaLikes(ctx, tx, u.oldPath, u.storedPath); err != nil {
		return rollbackWith(ctx, tx, fmt.Errorf("reprise de la fiche %d: %w", u.id, err))
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("reprise de la fiche %d: validation de la transaction: %w", u.id, err)
	}
	return nil
}

// revival décrit le fichier re-déposé d'une fiche supprimée.
type revival struct {
	id         int64
	storedPath string
	srcSize    sql.NullInt64
	srcMtime   sql.NullInt64
}

// reviveDeletedMedia annule la suppression d'une fiche dont le fichier vient d'être
// re-déposé. Si la fiche est convertie et que sa version convertie n'existe plus sur
// le disque (retirée avec la suppression), elle repasse en lecture directe dans le
// même UPDATE : file_path = source_path (sinon le fichier déposé), hls_path et
// transcode_status à NULL, que le balayage de conversion reprend ; ses likes suivent
// le changement de file_path.
func reviveDeletedMedia(ctx context.Context, db *sql.DB, store MediaPathStore, r revival) error {
	var fp, hp, sp sql.NullString
	if err := db.QueryRowContext(ctx,
		`SELECT file_path, hls_path, source_path FROM media_files WHERE id = ?`, r.id).Scan(&fp, &hp, &sp); err != nil {
		return fmt.Errorf("résurrection de la fiche %d: lecture: %w", r.id, err)
	}
	if hp.String == "" || fileExists(store.ToAbs(hp.String)) {
		_, err := db.ExecContext(ctx,
			`UPDATE media_files SET status = NULL, source_path = ?, source_size = ?, source_mtime_ns = ?
			 WHERE id = ?`, r.storedPath, r.srcSize, r.srcMtime, r.id)
		return err
	}

	direct := r.storedPath
	if sp.String != "" {
		direct = sp.String
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("résurrection de la fiche %d: ouverture de la transaction: %w", r.id, err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE media_files
		SET status = NULL, source_path = ?, source_size = ?, source_mtime_ns = ?,
			file_path = ?, hls_path = NULL, transcode_status = NULL
		WHERE id = ?`, r.storedPath, r.srcSize, r.srcMtime, direct, r.id); err != nil {
		return rollbackWith(ctx, tx, fmt.Errorf("résurrection de la fiche %d: %w", r.id, err))
	}
	if err := platform_duckdb.TransferMediaLikes(ctx, tx, fp.String, direct); err != nil {
		return rollbackWith(ctx, tx, fmt.Errorf("résurrection de la fiche %d: %w", r.id, err))
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("résurrection de la fiche %d: validation de la transaction: %w", r.id, err)
	}
	return nil
}

// rollbackWith annule la transaction et rend cause ; un échec de l'annulation est journalisé.
func rollbackWith(ctx context.Context, tx *sql.Tx, cause error) error {
	if rbErr := tx.Rollback(); rbErr != nil {
		slog.ErrorContext(ctx, "insertMediaFile: annulation de la transaction échouée", "err", rbErr, "cause", cause)
	}
	return cause
}
