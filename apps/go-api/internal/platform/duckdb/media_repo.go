// Package duckdb — media_repo.go : accès DB pour la galerie médias.
package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// MediaRepo implémente port.MediaRepository.
type MediaRepo struct {
	pdb *PlayerDB
}

// NewMediaRepo crée un MediaRepo pour un joueur.
func NewMediaRepo(pdb *PlayerDB) *MediaRepo {
	return &MediaRepo{pdb: pdb}
}

// LoadMediaFiles charge une page de fichiers médias avec filtres dynamiques (Q37).
func (r *MediaRepo) LoadMediaFiles(ctx context.Context, filters domain.MediaFilters, limit, offset int) ([]domain.MediaFileRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	q, args := BuildQ37MediaQuery(filters, limit, offset)
	rows, err := r.pdb.Player.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("LoadMediaFiles: %w", err)
	}
	defer rows.Close()

	var result []domain.MediaFileRow
	for rows.Next() {
		var row domain.MediaFileRow
		if err := rows.Scan(
			&row.FilePath,
			&row.FileName,
			&row.Kind,
			&row.ThumbnailPath,
			&row.CaptureEndUTC,
			&row.MatchID,
			&row.MatchStartTime,
			&row.Liked,
			&row.MapName,
			&row.ModeName,
		); err != nil {
			return nil, fmt.Errorf("LoadMediaFiles scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// CountMediaFiles retourne le nombre total de fichiers médias actifs selon les filtres.
func (r *MediaRepo) CountMediaFiles(ctx context.Context, filters domain.MediaFilters) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	q, args := BuildQ37MediaCountQuery(filters)
	var count int
	err := r.pdb.Player.QueryRow(ctx, q, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("CountMediaFiles: %w", err)
	}
	return count, nil
}

// SetMediaLike persiste l'état liked d'un média dans media_files (cache local).
func (r *MediaRepo) SetMediaLike(ctx context.Context, filePath string, liked bool) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := r.pdb.Player.Exec(ctx, `
		UPDATE media_files
		SET liked = ?,
			liked_at = CASE WHEN ? THEN CURRENT_TIMESTAMP ELSE NULL END
		WHERE file_path = ?
	`, liked, liked, filePath)
	if err != nil {
		return false, fmt.Errorf("SetMediaLike: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("SetMediaLike rows affected: %w", err)
	}
	return rowsAffected > 0, nil
}

// ToggleSharedLike écrit ou supprime un like dans media_likes (shared DB).
// Retourne l'état liked résultant.
func (r *MediaRepo) ToggleSharedLike(ctx context.Context, mediaPath, likerSlug, likerGamertag string, liked bool) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if liked {
		_, err := r.pdb.Shared.Exec(ctx, `
			INSERT INTO media_likes (media_path, liker_slug, liker_gamertag, liked_at)
			VALUES (?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT (media_path, liker_slug) DO UPDATE SET
				liker_gamertag = EXCLUDED.liker_gamertag,
				liked_at = CURRENT_TIMESTAMP
		`, mediaPath, likerSlug, likerGamertag)
		return err
	}
	_, err := r.pdb.Shared.Exec(ctx, `
		DELETE FROM media_likes WHERE media_path = ? AND liker_slug = ?
	`, mediaPath, likerSlug)
	return err
}

// GetMediaLikers retourne pour chaque media_path ses likers (max 3 noms + total).
func (r *MediaRepo) GetMediaLikers(ctx context.Context, mediaPaths []string) (map[string]domain.MediaLikersInfo, error) {
	if len(mediaPaths) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Construire le placeholder IN (?, ?, ...)
	placeholders := make([]string, len(mediaPaths))
	args := make([]any, len(mediaPaths))
	for i, p := range mediaPaths {
		placeholders[i] = "?"
		args[i] = p
	}

	q := `SELECT media_path, liker_gamertag, ROW_NUMBER() OVER (PARTITION BY media_path ORDER BY liked_at) AS rn,
		COUNT(*) OVER (PARTITION BY media_path) AS total
	FROM media_likes
	WHERE media_path IN (` + joinStrings(placeholders) + `)
	ORDER BY media_path, liked_at`

	rows, err := r.pdb.Shared.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("GetMediaLikers: %w", err)
	}
	defer rows.Close()

	result := make(map[string]domain.MediaLikersInfo)
	for rows.Next() {
		var path, gamertag string
		var rn, total int
		if err := rows.Scan(&path, &gamertag, &rn, &total); err != nil {
			return nil, err
		}
		info := result[path]
		info.Total = total
		if rn <= 3 {
			info.Names = append(info.Names, gamertag)
		}
		result[path] = info
	}
	return result, rows.Err()
}

func joinStrings(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}
