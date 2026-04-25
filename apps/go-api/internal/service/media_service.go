// Package service — media_service.go : orchestration de la galerie médias.
package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/port"
)

const defaultMediaPageSize = 24
const maxMediaPageSize = 100

// MediaService orchestre les données de la galerie médias.
type MediaService struct {
	repo     port.MediaRepository
	timezone string // IANA assaini à la construction (ex: "Europe/Paris")
}

// NewMediaService crée un MediaService avec le repository et la timezone injectés.
// La timezone est validée par SanitizeMediaTimezone (caractères IANA autorisés uniquement).
func NewMediaService(repo port.MediaRepository, timezone string) *MediaService {
	return &MediaService{
		repo:     repo,
		timezone: ops.SanitizeMediaTimezone(timezone),
	}
}

// GetMediaPage construit la réponse paginée de la galerie médias.
func (s *MediaService) GetMediaPage(
	ctx context.Context,
	req domain.MediaPageRequest,
) (*domain.MediaPageResponse, error) {
	page := req.ResolvePage()
	limit := req.ResolvePageSize(defaultMediaPageSize, maxMediaPageSize)
	offset := (page - 1) * limit

	// Construction des filtres depuis la requête
	kindFilter := req.KindFilter
	if kindFilter == "" {
		kindFilter = req.Kind // compat. legacy
	}
	filters := domain.MediaFilters{
		KindFilter:    kindFilter,
		SectionFilter: req.SectionFilter,
		AuthorSlugs:   req.AuthorSlugs,
		MapFilter:     req.MapFilter,
		ModeFilter:    req.ModeFilter,
		LikedOnly:     req.LikedOnly,
		Sort:          req.Sort,
		GroupBy:       req.GroupBy,
	}

	files, err := s.repo.LoadMediaFiles(ctx, filters, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.repo.CountMediaFiles(ctx, filters)
	if err != nil {
		total = len(files)
	}

	hasNext := offset+len(files) < total
	availableFilters := s.resolveAvailableFilters(ctx, filters, files)

	items := buildMediaItems(files)

	// Enrichissement des likers depuis shared DB (best-effort)
	paths := make([]string, len(items))
	for i, it := range items {
		paths[i] = it.FilePath
	}
	if likersMap, err := s.repo.GetMediaLikers(ctx, paths); err == nil && len(likersMap) > 0 {
		for i := range items {
			if info, ok := likersMap[items[i].FilePath]; ok {
				items[i].Likers = info.Names
				items[i].TotalLikers = info.Total
				items[i].LikeCount = info.Total
			}
		}
	}

	return &domain.MediaPageResponse{
		Items: domain.MediaItemsPage{
			Items: items,
			Pagination: domain.PaginationMeta{
				Total:    total,
				Page:     page,
				PageSize: limit,
				HasNext:  hasNext,
				HasPrev:  page > 1,
			},
		},
		TotalMine:        total,
		TotalTeammates:   0,
		TotalUnassigned:  0,
		AvailableFilters: availableFilters,
	}, nil
}

// SetMediaLike persiste l'état liked d'un média.
func (s *MediaService) SetMediaLike(
	ctx context.Context,
	req domain.MediaLikeRequest,
) (*domain.MediaLikeResponse, error) {
	if req.FilePath == "" {
		return nil, domain.ErrBadRequest("file_path est requis")
	}

	ok, err := s.repo.SetMediaLike(ctx, req.FilePath, req.Liked)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.ErrNotFound("media", req.FilePath)
	}

	// Écriture dans media_likes partagé si on a un likerSlug
	if req.LikerSlug != "" {
		likerGamertag := req.LikerGamertag
		if likerGamertag == "" {
			likerGamertag = req.LikerSlug
		}
		if err := s.repo.ToggleSharedLike(ctx, req.FilePath, req.LikerSlug, likerGamertag, req.Liked); err != nil {
			// Non-bloquant : le like local est déjà persisté
			slog.Warn("ToggleSharedLike: erreur non-bloquante", "err", err)
		}
	}

	// Récupération des likers pour la réponse
	resp := &domain.MediaLikeResponse{
		FilePath:  req.FilePath,
		Liked:     req.Liked,
		LikeCount: boolToLikeCount(req.Liked),
	}
	if likersMap, err := s.repo.GetMediaLikers(ctx, []string{req.FilePath}); err == nil {
		if info, ok := likersMap[req.FilePath]; ok {
			resp.Likers = info.Names
			resp.TotalLikers = info.Total
			resp.LikeCount = info.Total
		}
	}
	return resp, nil
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
		dest := safeDestPath(req.CapturesDir, f.OriginalName)
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
		tol = 2
	}

	// Construction de la map basename → unix ts client (pour le filename parser)
	captureTimes := make(map[string]*int64, len(req.Files))
	for _, f := range req.Files {
		if f.CaptureTimeUnix != nil {
			captureTimeUnix := f.CaptureTimeUnix
			captureTimes[f.OriginalName] = captureTimeUnix
		}
	}

	slog.InfoContext(ctx, "upload: démarrage indexation",
		"captures_dir", req.CapturesDir, "buffer_min", tol, "timezone", s.timezone)

	idxResult, err := ops.IndexMedia(ops.MediaIndexOptions{
		PlayerDBPath:        req.DBPath,
		SharedSocialDBPath:  req.SharedSocialDBPath,
		SharedMatchesDBPath: req.SharedMatchesDBPath,
		CapturesDir:         req.CapturesDir,
		BufferMin:           tol,
		Timezone:            s.timezone,
		CaptureTimes:        captureTimes,
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

	// Lier les miniatures générées aux enregistrements DB (thumbnail_path NULL → chemin absolu).
	targetPath := req.DBPath
	if req.SharedSocialDBPath != "" {
		targetPath = req.SharedSocialDBPath
	}
	if targetPath != "" {
		if db, err := sql.Open("duckdb", targetPath); err == nil {
			if backfilled, err := ops.BackfillThumbnailPaths(db, req.CapturesDir, thumbsDir); err != nil {
				slog.WarnContext(ctx, "upload: backfill thumbnail_path échoué", "err", err)
			} else {
				result.Thumbnails += backfilled
			}
			db.Close() //nolint:errcheck
		}
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

// ReassociateMedia recrée toutes les associations médias↔matchs depuis zéro.
// Avant de supprimer, crée un snapshot horodaté (backup) dans la même DB.
func (s *MediaService) ReassociateMedia(ctx context.Context, req domain.ReassociateRequest) (*domain.ReassociateResult, error) {
	result := &domain.ReassociateResult{}

	// Sélectionner la DB cible (shared_social si dispo, sinon stats.duckdb).
	targetPath := req.DBPath
	if req.SharedSocialDBPath != "" {
		targetPath = req.SharedSocialDBPath
	}
	if targetPath == "" {
		return nil, fmt.Errorf("ReassociateMedia: aucun chemin DB fourni")
	}

	bufferMin := req.BufferMin
	if bufferMin <= 0 {
		bufferMin = 2
	}

	// Ouvrir la DB en lecture-écriture.
	db, err := sql.Open("duckdb", targetPath)
	if err != nil {
		return nil, fmt.Errorf("ouverture DB: %w", err)
	}
	defer db.Close() //nolint:errcheck

	// Appliquer la timezone pour les opérations sur timestamps.
	if tz := ops.SanitizeMediaTimezone(s.timezone); tz != "" {
		if _, err := db.Exec("SET TimeZone = '" + tz + "'"); err != nil {
			slog.WarnContext(ctx, "ReassociateMedia: SET TimeZone échoué",
				"timezone", s.timezone, "err", err)
		}
	}

	// 1 — Backup : snapshot horodaté avant suppression.
	backupTable := fmt.Sprintf("media_match_associations_bak_%s",
		time.Now().UTC().Format("20060102T150405Z"))
	createBackupSQL := fmt.Sprintf(
		`CREATE TABLE %s AS SELECT * FROM media_match_associations`, backupTable)
	if _, err := db.Exec(createBackupSQL); err != nil {
		return nil, fmt.Errorf("backup table: %w", err)
	}
	result.BackupTable = backupTable
	slog.InfoContext(ctx, "ReassociateMedia: backup créé", "table", backupTable)

	// 2 — Compter puis supprimer les anciennes associations.
	var oldCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM media_match_associations").Scan(&oldCount); err != nil {
		slog.WarnContext(ctx, "ReassociateMedia: COUNT avant suppression échoué", "err", err)
	}
	if _, err := db.Exec("DELETE FROM media_match_associations"); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("DELETE: %v", err))
		return result, fmt.Errorf("supprimer associations: %w", err)
	}
	result.DeletedAssoc = oldCount
	slog.InfoContext(ctx, "ReassociateMedia: associations supprimées", "count", oldCount)

	// 3 — Re-créer les associations.
	newAssoc, err := ops.AssociateMediaWithMatches(db, req.SharedMatchesDBPath, bufferMin, s.timezone)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("association: %v", err))
		slog.ErrorContext(ctx, "ReassociateMedia: association échouée", "err", err)
	}
	result.NewAssoc = newAssoc

	// 4 — Backfill thumbnail_path pour les GIFs existants dans thumbs/.
	if req.CapturesDir != "" {
		thumbsDir := filepath.Join(req.CapturesDir, "thumbs")
		if n, backfillErr := ops.BackfillThumbnailPaths(db, req.CapturesDir, thumbsDir); backfillErr != nil {
			slog.WarnContext(ctx, "ReassociateMedia: backfill thumbnail_path échoué", "err", backfillErr)
		} else {
			slog.InfoContext(ctx, "ReassociateMedia: thumbnail_path backfillé", "updated", n)
		}
	}

	slog.InfoContext(ctx, "ReassociateMedia: terminé",
		"backup_table", backupTable,
		"deleted", result.DeletedAssoc,
		"new_assoc", newAssoc,
		"errors", len(result.Errors))

	return result, nil
}

func buildMediaItems(rows []domain.MediaFileRow) []domain.MediaItem {
	items := make([]domain.MediaItem, 0, len(rows))
	for _, r := range rows {
		basename := r.FileName
		if basename == "" {
			basename = filepath.Base(r.FilePath)
		}
		items = append(items, domain.MediaItem{
			Basename:       basename,
			FilePath:       r.FilePath,
			Kind:           r.Kind,
			ThumbnailPath:  r.ThumbnailPath,
			CaptureEndUTC:  r.CaptureEndUTC,
			MatchID:        r.MatchID,
			MatchStartTime: r.MatchStartTime,
			MapName:        r.MapName,
			ModeName:       r.ModeName,
			Section:        "mine",
			Liked:          r.Liked,
			LikeCount:      boolToLikeCount(r.Liked),
		})
	}
	return items
}

func (s *MediaService) resolveAvailableFilters(
	ctx context.Context,
	filters domain.MediaFilters,
	files []domain.MediaFileRow,
) domain.MediaFilterOptions {
	fallback := buildMediaFilterOptionsFromRows(files)

	options, err := s.repo.LoadMediaFilterOptions(ctx, filters)
	if err != nil {
		slog.WarnContext(ctx, "media: options de filtres indisponibles", "err", err)
		return fallback
	}

	if len(options.Maps) == 0 {
		options.Maps = fallback.Maps
	}
	if len(options.Modes) == 0 {
		options.Modes = fallback.Modes
	}
	return options
}

func buildMediaFilterOptionsFromRows(rows []domain.MediaFileRow) domain.MediaFilterOptions {
	return domain.MediaFilterOptions{
		Maps:  collectMediaLabelValues(rows, func(row domain.MediaFileRow) *string { return row.MapName }),
		Modes: collectMediaLabelValues(rows, func(row domain.MediaFileRow) *string { return row.ModeName }),
	}
}

func collectMediaLabelValues(
	rows []domain.MediaFileRow,
	selector func(domain.MediaFileRow) *string,
) []domain.LabelValue {
	seen := make(map[string]struct{})
	labels := make([]string, 0)
	for _, row := range rows {
		labelPtr := selector(row)
		if labelPtr == nil {
			continue
		}
		label := strings.TrimSpace(*labelPtr)
		if label == "" {
			continue
		}
		if _, exists := seen[label]; exists {
			continue
		}
		seen[label] = struct{}{}
		labels = append(labels, label)
	}

	sort.Strings(labels)
	options := make([]domain.LabelValue, 0, len(labels))
	for _, label := range labels {
		options = append(options, domain.LabelValue{Label: label, Value: label})
	}
	return options
}

func boolToLikeCount(liked bool) int {
	if liked {
		return 1
	}
	return 0
}

// safeDestPath retourne un chemin de destination sûr pour un upload.
// Si un fichier du même nom existe déjà, ajoute un suffixe timestamp
// pour éviter les collisions silencieuses lors d'uploads simultanés.
func safeDestPath(dir, originalName string) string {
	base := filepath.Base(originalName)
	dest := filepath.Join(dir, base)
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return dest
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	ts := time.Now().UTC().Format("20060102T150405Z")
	return filepath.Join(dir, stem+"_"+ts+ext)
}
