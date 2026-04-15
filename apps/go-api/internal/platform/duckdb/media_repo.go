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

// LoadMediaFiles charge une page de fichiers médias (Q37).
func (r *MediaRepo) LoadMediaFiles(ctx context.Context, limit, offset int) ([]domain.MediaFileRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.Player.Query(ctx, Q37MediaFiles, limit, offset)
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
		); err != nil {
			return nil, fmt.Errorf("LoadMediaFiles scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// CountMediaFiles retourne le nombre total de fichiers médias actifs (Q37Count).
func (r *MediaRepo) CountMediaFiles(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var count int
	err := r.pdb.Player.QueryRow(ctx, Q37MediaCount).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("CountMediaFiles: %w", err)
	}
	return count, nil
}
