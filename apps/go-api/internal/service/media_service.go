// Package service — media_service.go : orchestration de la galerie médias.
package service

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

const defaultMediaPageSize = 24

// MediaService orchestre les données de la galerie médias.
type MediaService struct {
	repo port.MediaRepository
}

// NewMediaService crée un MediaService avec le repository injecté.
func NewMediaService(repo port.MediaRepository) *MediaService {
	return &MediaService{repo: repo}
}

// GetMediaPage construit la réponse paginée de la galerie médias.
func (s *MediaService) GetMediaPage(
	ctx context.Context,
	page int,
) (*domain.MediaPageResponse, error) {
	if page < 1 {
		page = 1
	}
	limit := defaultMediaPageSize
	offset := (page - 1) * limit

	files, err := s.repo.LoadMediaFiles(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.repo.CountMediaFiles(ctx)
	if err != nil {
		total = len(files)
	}

	hasMore := offset+len(files) < total

	return &domain.MediaPageResponse{
		Items:      buildMediaItems(files),
		TotalCount: total,
		Page:       page,
		PageSize:   limit,
		HasMore:    hasMore,
	}, nil
}

func buildMediaItems(rows []domain.MediaFileRow) []domain.MediaItem {
	items := make([]domain.MediaItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, domain.MediaItem{
			FileName:       r.FileName,
			FilePath:       r.FilePath,
			Kind:           r.Kind,
			ThumbnailPath:  r.ThumbnailPath,
			CaptureEndUTC:  r.CaptureEndUTC,
			MatchID:        r.MatchID,
			MatchStartTime: r.MatchStartTime,
		})
	}
	return items
}
