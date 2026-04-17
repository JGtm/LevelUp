// Package service — media_service.go : orchestration de la galerie médias.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/ops"
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

// UploadMedia sauvegarde les fichiers uploadés sur disque puis les indexe
// (scan + association matchs + génération miniatures).
func (s *MediaService) UploadMedia(ctx context.Context, req domain.UploadRequest) (*domain.UploadResult, error) {
	result := &domain.UploadResult{}

	if len(req.Files) == 0 {
		return result, nil
	}

	if err := os.MkdirAll(req.CapturesDir, 0o755); err != nil {
		return nil, fmt.Errorf("créer captures dir %q: %w", req.CapturesDir, err)
	}

	for _, f := range req.Files {
		dest := filepath.Join(req.CapturesDir, filepath.Base(f.OriginalName))
		if err := os.WriteFile(dest, f.Data, 0o644); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", f.OriginalName, err))
			slog.WarnContext(ctx, "upload: écriture fichier échouée",
				"file", f.OriginalName, "err", err)
			continue
		}
		result.Saved++
		slog.DebugContext(ctx, "upload: fichier sauvegardé",
			"file", f.OriginalName, "dest", dest)
	}

	if result.Saved == 0 {
		slog.WarnContext(ctx, "upload: aucun fichier sauvegardé", "attempted", len(req.Files))
		return result, nil
	}

	tol := req.Tolerance
	if tol <= 0 {
		tol = 5
	}

	slog.InfoContext(ctx, "upload: démarrage indexation",
		"captures_dir", req.CapturesDir, "tolerance_min", tol)

	idxResult, err := ops.IndexMedia(ops.MediaIndexOptions{
		PlayerDBPath: req.DBPath,
		CapturesDir:  req.CapturesDir,
		ToleranceMin: tol,
	})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("indexation: %v", err))
		slog.ErrorContext(ctx, "upload: indexation échouée", "err", err)
	}
	result.NewIndexed = idxResult.NewFiles
	result.Associated = idxResult.Associated
	result.Errors = append(result.Errors, idxResult.Errors...)

	// Miniatures — non bloquant, erreurs tracées mais pas remontées
	thumbsDir := filepath.Join(req.CapturesDir, "thumbs")
	n, thumbErrs := ops.GenerateThumbnails(req.CapturesDir, thumbsDir)
	result.Thumbnails = n
	if len(thumbErrs) > 0 {
		slog.WarnContext(ctx, "upload: erreurs miniatures", "count", len(thumbErrs))
		result.Errors = append(result.Errors, thumbErrs...)
	}

	slog.InfoContext(ctx, "upload: terminé",
		"saved", result.Saved,
		"new_indexed", result.NewIndexed,
		"associated", result.Associated,
		"thumbnails", result.Thumbnails,
		"errors", len(result.Errors),
	)
	return result, nil
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
