// Package duckdb — media_repo_missing.go : lectures et fusion des médias introuvables (statut
// MediaStatusMissing).
//
// La fusion n'écrit que des INSERT dans les tables append-only (likes, rattachements) et l'UPDATE de
// la colonne non indexée `status` : jamais de DELETE sur media_files ni d'UPDATE d'une colonne
// indexée (justification en tête de internal/domain/media_delete.go).
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
)

// mediaMissingPredicate sélectionne les seules fiches introuvables.
const mediaMissingPredicate = "status = '" + domain.MediaStatusMissing + "'"

// ListMissingMedia rend les médias introuvables du joueur du dépôt, du plus récemment capturé au
// plus ancien. Le match est celui de la vue des rattachements (le manuel prime sur l'automatique).
func (r *MediaRepo) ListMissingMedia(ctx context.Context) ([]domain.MediaMissingItem, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	items := []domain.MediaMissingItem{}
	if r.pdb.SharedSocial == nil {
		return items, nil
	}
	rows, err := r.socialDB().Query(ctx, `
		SELECT mf.file_path, COALESCE(mf.file_name, ''), COALESCE(mf.kind, ''), mf.thumbnail_path,
		       mf.capture_end_utc,
		       (SELECT mma.match_id FROM media_match_associations_latest mma
		         WHERE mma.media_file_id = mf.id
		         ORDER BY mma.is_manual DESC, mma.written_at DESC LIMIT 1)
		FROM media_files mf
		WHERE mf.player_slug = ? AND mf.`+mediaMissingPredicate+`
		ORDER BY mf.capture_end_utc DESC NULLS LAST, mf.id DESC
	`, r.pdb.Gamertag)
	if err != nil {
		return nil, fmt.Errorf("ListMissingMedia: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var it domain.MediaMissingItem
		var thumb, match sql.NullString
		var captured sql.NullTime
		if err := rows.Scan(&it.FilePath, &it.Basename, &it.Kind, &thumb, &captured, &match); err != nil {
			return nil, fmt.Errorf("ListMissingMedia scan: %w", err)
		}
		it.ThumbnailPath = optionalMediaString(thumb)
		it.CaptureEndUTC = optionalMediaTime(captured)
		it.MatchID = optionalMediaString(match)
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListMissingMedia rows: %w", err)
	}
	return items, nil
}

// LoadMissingTarget charge la fiche introuvable désignée par son chemin stocké.
func (r *MediaRepo) LoadMissingTarget(ctx context.Context, filePath string) (domain.MediaFileRef, error) {
	return r.loadMediaRef(ctx, filePath, mediaMissingPredicate)
}

// LoadMergeTarget charge la fiche présente (ni supprimée ni introuvable) désignée par son chemin
// stocké.
func (r *MediaRepo) LoadMergeTarget(ctx context.Context, filePath string) (domain.MediaFileRef, error) {
	return r.loadMediaRef(ctx, filePath, MediaVisiblePredicate(""))
}

// loadMediaRef charge la fiche de chemin filePath filtrée par predicate (fragment SQL non qualifié
// sur media_files). Rend ErrNotFound quand aucune fiche ne correspond.
func (r *MediaRepo) loadMediaRef(ctx context.Context, filePath, predicate string) (domain.MediaFileRef, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if r.pdb.SharedSocial == nil {
		return domain.MediaFileRef{}, domain.ErrNotFound("media", filePath)
	}
	var ref domain.MediaFileRef
	var owner, thumb, kind sql.NullString
	var captured sql.NullTime
	err := r.socialDB().QueryRow(ctx, `
		SELECT id, player_slug, file_path, thumbnail_path, kind,
		       COALESCE(capture_start_utc, capture_end_utc)
		FROM media_files
		WHERE file_path = ? AND `+predicate+`
		ORDER BY id LIMIT 1
	`, filePath).Scan(&ref.ID, &owner, &ref.FilePath, &thumb, &kind, &captured)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.MediaFileRef{}, domain.ErrNotFound("media", filePath)
	}
	if err != nil {
		return domain.MediaFileRef{}, fmt.Errorf("loadMediaRef: %w", err)
	}
	ref.OwnerSlug, ref.ThumbnailPath, ref.Kind = owner.String, thumb.String, kind.String
	ref.CaptureUTC = optionalMediaTime(captured)
	ref.HLSPath = r.loadMediaHLSPath(ctx, ref.ID)
	ref.SourcePath, err = r.loadMediaSourcePath(ctx, ref.ID)
	if err != nil {
		return domain.MediaFileRef{}, err
	}
	return ref, nil
}

// ListMergeCandidates rend les fiches présentes du propriétaire de ref, de même nature, autres que
// ref, triées par écart absolu de date de capture croissant (les fiches sans date en dernier), limit
// au plus. L'écart est calculé par la base à partir de la fiche ref.ID : aucun instant ne traverse
// la liaison de paramètres.
func (r *MediaRepo) ListMergeCandidates(
	ctx context.Context,
	ref domain.MediaFileRef,
	limit int,
) ([]domain.MediaMergeCandidate, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	out := []domain.MediaMergeCandidate{}
	if r.pdb.SharedSocial == nil {
		return out, nil
	}
	rows, err := r.socialDB().Query(ctx, `
		WITH ref AS (
			SELECT COALESCE(capture_start_utc, capture_end_utc) AS t
			FROM media_files WHERE id = ? AND `+MediaNotDeletedPredicate("")+`
		)
		SELECT c.file_path, COALESCE(c.file_name, ''), COALESCE(c.kind, ''), c.thumbnail_path,
		       c.capture_end_utc,
		       CAST(round(abs(epoch(COALESCE(c.capture_start_utc, c.capture_end_utc)) - epoch(ref.t))) AS INTEGER) AS delta
		FROM media_files c CROSS JOIN ref
		WHERE c.player_slug = ? AND c.id <> ? AND c.kind = ? AND `+MediaVisiblePredicate("c")+`
		ORDER BY delta IS NULL, delta, c.id
		LIMIT ?
	`, ref.ID, ref.OwnerSlug, ref.ID, ref.Kind, limit)
	if err != nil {
		return nil, fmt.Errorf("ListMergeCandidates: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var c domain.MediaMergeCandidate
		var thumb sql.NullString
		var captured sql.NullTime
		var delta sql.NullInt64
		if err := rows.Scan(&c.FilePath, &c.Basename, &c.Kind, &thumb, &captured, &delta); err != nil {
			return nil, fmt.Errorf("ListMergeCandidates scan: %w", err)
		}
		c.ThumbnailPath = optionalMediaString(thumb)
		c.CaptureEndUTC = optionalMediaTime(captured)
		if delta.Valid {
			d := int(delta.Int64)
			c.DeltaSeconds = &d
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListMergeCandidates rows: %w", err)
	}
	return out, nil
}

// ApplyMediaMerge fond la fiche introuvable missing dans la fiche présente target, dans la
// transaction exec :
//   - les likes actifs de missing passent sur le chemin de target (TransferMediaLikes, INSERT purs) ;
//   - le rattachement courant de missing, manuel ou automatique, devient le rattachement manuel de
//     target, sauf si target en a déjà un (INSERT dans media_match_associations_history, même forme
//     que SetMediaMatchAssociation) ;
//   - missing est masquée (status = 'deleted', seulement si elle est encore introuvable).
//
// Aucune ligne affectée par le masquage (la fiche a changé d'état entre le chargement et l'écriture)
// est une erreur : l'appelant annule la transaction.
func (r *MediaRepo) ApplyMediaMerge(ctx context.Context, exec dblease.DBExecutor, missing, target domain.MediaFileRef) error {
	if err := TransferMediaLikes(ctx, exec, missing.FilePath, target.FilePath); err != nil {
		return err
	}
	if err := carryMatchAssociation(ctx, exec, missing.ID, target.ID); err != nil {
		return err
	}
	res, err := exec.ExecContext(ctx, `
		UPDATE media_files
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND `+mediaMissingPredicate, domain.MediaStatusDeleted, missing.ID)
	if err != nil {
		return fmt.Errorf("ApplyMediaMerge: masquage de la fiche introuvable: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("ApplyMediaMerge rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound("media", missing.FilePath)
	}
	return nil
}

// carryMatchAssociation donne à la fiche toID, en rattachement manuel, le match courant de la fiche
// fromID : celui de la vue des rattachements, où le manuel prime sur l'automatique. Une fiche
// introuvable est une fiche que le disque ne confirme plus : le match que l'utilisateur ou la passe
// d'association lui avait trouvé est le seul acquis à ne pas perdre. Rien n'est écrit quand fromID n'a
// pas de rattachement, ni quand toID a déjà un rattachement manuel : le choix de l'utilisateur sur la
// cible prime.
func carryMatchAssociation(ctx context.Context, exec dblease.DBExecutor, fromID, toID int64) error {
	var targetManual int
	if err := exec.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM media_match_associations_latest
		WHERE media_file_id = ? AND is_manual = TRUE`, toID).Scan(&targetManual); err != nil {
		return fmt.Errorf("ApplyMediaMerge: lecture du rattachement de la cible: %w", err)
	}
	if targetManual > 0 {
		return nil
	}
	var matchID string
	err := exec.QueryRowContext(ctx, `
		SELECT match_id FROM media_match_associations_latest
		WHERE media_file_id = ?
		ORDER BY is_manual DESC, written_at DESC LIMIT 1`, fromID).Scan(&matchID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("ApplyMediaMerge: lecture du rattachement de la fiche introuvable: %w", err)
	}
	if _, err := exec.ExecContext(ctx, `
		INSERT INTO media_match_associations_history
			(media_file_id, match_id, delta_seconds, is_manual, is_active, associated_at, written_at)
		VALUES (?, ?, 0, TRUE, TRUE, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP), CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))`,
		toID, matchID); err != nil {
		return fmt.Errorf("ApplyMediaMerge: report du rattachement: %w", err)
	}
	return nil
}

// nullableString rend nil pour une valeur SQL NULL ou vide.
func optionalMediaString(s sql.NullString) *string {
	if !s.Valid || s.String == "" {
		return nil
	}
	v := s.String
	return &v
}

// nullableTime rend nil pour une valeur SQL NULL.
func optionalMediaTime(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}
