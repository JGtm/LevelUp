// Package service — media_service.go : orchestration de la galerie médias.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/jobs"
	"levelup/go-api/internal/port"
)

const defaultMediaPageSize = 24
const maxMediaPageSize = 100

// atomicMediaLiker est l'interface optionnelle satisfaite par platform/duckdb.MediaRepo
// (méthode SetMediaLikeAtomic). Permet au service de basculer en mode atomique
// (transaction) quand le repo concret la supporte, sans modifier l'interface
// publique port.MediaRepository — cf. commit 6 du refactor leased-writer-enforcement.
type atomicMediaLiker interface {
	SetMediaLikeAtomic(ctx context.Context, exec port.DBExecutor,
		filePath, likerSlug, likerGamertag string, liked bool) (bool, error)
}

// MediaService orchestre les données de la galerie médias.
type MediaService struct {
	repo          port.MediaRepository
	timezone      string                                // IANA assaini à la construction (ex: "Europe/Paris")
	acquireWriter func() (*dblease.LeasedWriter, error) // optionnel
	jobStore      *jobs.Store                           // optionnel : si nil, pas de transcoding HLS
	feedBump      func()                                // optionnel : notifie la galerie (BumpMediaFeedVersion)
	fileRemover   port.MediaFileRemover                 // optionnel : si nil, DeleteMedia refuse d'agir
}

// MediaOption configure un MediaService à la construction.
type MediaOption func(*MediaService)

// WithMediaWriterAcquirer configure l'acquisition d'un *LeasedWriter sur
// shared_social.duckdb pour rendre SetMediaLike atomique : le service ouvre
// une transaction via writer.BeginTx et appelle repo.SetMediaLikeAtomic.
//
// Si nil, ou si le repo concret n'expose pas SetMediaLikeAtomic, le service
// retombe sur le chemin legacy non-atomique (SetMediaLike + ToggleSharedLike
// séparés). Garantit la non-régression pour les tests existants.
func WithMediaWriterAcquirer(f func() (*dblease.LeasedWriter, error)) MediaOption {
	return func(s *MediaService) { s.acquireWriter = f }
}

// WithMediaTranscoding active le transcoding HLS asynchrone à l'upload. jobStore
// porte le cycle de vie des jobs (persistant, repris au boot) ; feedBump notifie
// la galerie quand un clip devient lisible. Si non configuré, les vidéos
// multipistes/MKV restent servies via le remux WebM live (dégradation gracieuse).
func WithMediaTranscoding(jobStore *jobs.Store, feedBump func()) MediaOption {
	return func(s *MediaService) {
		s.jobStore = jobStore
		s.feedBump = feedBump
	}
}

// NewMediaService crée un MediaService avec le repository et la timezone injectés.
// La timezone est validée par SanitizeMediaTimezone (caractères IANA autorisés uniquement).
func NewMediaService(repo port.MediaRepository, timezone string, opts ...MediaOption) *MediaService {
	s := &MediaService{
		repo:     repo,
		timezone: ops.SanitizeMediaTimezone(timezone),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
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
		KindFilter:     kindFilter,
		SectionFilter:  req.SectionFilter,
		AuthorSlugs:    req.AuthorSlugs,
		PlaylistFilter: req.PlaylistFilter,
		MapFilter:      req.MapFilter,
		ModeFilter:     req.ModeFilter,
		LikedOnly:      req.LikedOnly,
		UnassignedOnly: req.UnassignedOnly,
		Sort:           req.Sort,
		GroupBy:        req.GroupBy,
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

	items := buildMediaItems(files, s.repo.CurrentPlayerSlug())

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

	totalUnassigned := 0
	if filters.UnassignedOnly {
		totalUnassigned = total
	} else {
		unassignedFilters := domain.MediaFilters{
			SectionFilter:  filters.SectionFilter,
			AuthorSlugs:    filters.AuthorSlugs,
			UnassignedOnly: true,
		}
		if n, err := s.repo.CountMediaFiles(ctx, unassignedFilters); err == nil {
			totalUnassigned = n
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
		TotalUnassigned:  totalUnassigned,
		AvailableFilters: availableFilters,
	}, nil
}

// ListMediaAuthors retourne les auteurs sélectionnables dans le filtre Auteurs,
// depuis shared_social.media_files (même source que la galerie). Marque is_self
// pour le joueur courant. Le gamertag d'affichage est laissé vide ici : le handler
// l'enrichit depuis db_profiles.json.
func (s *MediaService) ListMediaAuthors(ctx context.Context) ([]domain.MediaAuthor, error) {
	authors, err := s.repo.ListMediaAuthors(ctx)
	if err != nil {
		return nil, err
	}
	currentSlug := s.repo.CurrentPlayerSlug()
	for i := range authors {
		authors[i].IsSelf = strings.EqualFold(authors[i].PlayerSlug, currentSlug)
	}
	return authors, nil
}

// GetMatchCandidates retourne les matchs candidats pour réassocier manuellement
// un média. Si la fenêtre est <=0, défaut à 15 min. Maximum 180 min (3h).
func (s *MediaService) GetMatchCandidates(ctx context.Context, filePath string, windowMinutes int) (*domain.MediaMatchCandidatesResponse, error) {
	if filePath == "" {
		return nil, domain.ErrBadRequest("file_path est requis")
	}
	if windowMinutes <= 0 {
		windowMinutes = 15
	}
	if windowMinutes > 180 {
		windowMinutes = 180
	}
	resp, err := s.repo.LoadMatchCandidatesForMedia(ctx, filePath, windowMinutes)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// AssociateMediaToMatch force l'association d'un média à un match précis,
// remplaçant l'association existante (si présente).
func (s *MediaService) AssociateMediaToMatch(ctx context.Context, req domain.MediaAssociateRequest) (*domain.MediaAssociateResponse, error) {
	if req.FilePath == "" {
		return nil, domain.ErrBadRequest("file_path est requis")
	}
	if req.MatchID == "" {
		return nil, domain.ErrBadRequest("match_id est requis")
	}
	mapName, modeName, err := s.repo.SetMediaMatchAssociation(ctx, req.FilePath, req.MatchID)
	if err != nil {
		return nil, err
	}
	return &domain.MediaAssociateResponse{
		FilePath: req.FilePath,
		MatchID:  req.MatchID,
		MapName:  mapName,
		ModeName: modeName,
	}, nil
}

// SetMediaLike persiste le like DU LIKER (req.LikerSlug) sur un média.
//
// Depuis le 2026-08-04 le like est PAR VIEWER : l'unique support de stockage est
// l'event append-only media_likes_history (lu via media_likes_latest). La colonne
// globale media_files.liked, qui allumait le cœur de tous les joueurs dès qu'un
// seul likait, a été droppée du schéma le 2026-08-04.
//
// Deux chemins :
//   - **Atomique** (commit 6 db-concurrency) : si un WriterAcquirer est
//     configuré ET le repo concret expose SetMediaLikeAtomic, le service ouvre
//     une transaction via writer.BeginTx ; le contrôle d'existence du média et
//     l'INSERT de l'event y vivent ensemble (rollback si l'un échoue).
//   - **Legacy** : sinon (tests existants, mock repo), exécute MediaExists puis
//     ToggleSharedLike séparément. Erreur ToggleSharedLike non-bloquante.
func (s *MediaService) SetMediaLike(
	ctx context.Context,
	req domain.MediaLikeRequest,
) (*domain.MediaLikeResponse, error) {
	if req.FilePath == "" {
		return nil, domain.ErrBadRequest("file_path est requis")
	}

	if atomic, ok := s.repo.(atomicMediaLiker); ok && s.acquireWriter != nil {
		return s.setMediaLikeAtomic(ctx, atomic, req)
	}
	return s.setMediaLikeLegacy(ctx, req)
}

// setMediaLikeAtomic exécute les deux writes (media_files + media_likes) dans
// une transaction unique sous un *LeasedWriter acquis. Rollback si l'un
// des deux SQL échoue.
func (s *MediaService) setMediaLikeAtomic(
	ctx context.Context,
	atomic atomicMediaLiker,
	req domain.MediaLikeRequest,
) (*domain.MediaLikeResponse, error) {
	w, err := s.acquireWriter()
	if err != nil {
		return nil, err
	}
	defer w.Release()

	tx, err := w.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("media_like begin tx: %w", err)
	}
	likerGamertag := req.LikerGamertag
	if likerGamertag == "" && req.LikerSlug != "" {
		likerGamertag = req.LikerSlug
	}
	updated, err := atomic.SetMediaLikeAtomic(ctx, tx, req.FilePath, req.LikerSlug, likerGamertag, req.Liked)
	if err != nil {
		_ = tx.Rollback()
		slog.WarnContext(ctx, "media_like atomic rollback", "err", err, "file_path", req.FilePath)
		return nil, err
	}
	if !updated {
		_ = tx.Rollback()
		return nil, domain.ErrNotFound("media", req.FilePath)
	}
	// GARANTIE DE DURABILITÉ DU LIKE (lecture-après-écriture) — v7.3 lot 2, item 1.5.
	//
	// Le like N'EST PAS confirmé au client tant que la transaction n'est pas
	// commitée ET le WAL shared_social flushé sur disque : CommitWithCheckpoint
	// (ADR 0021 Phase 3.2 bis / ADR 0022) enchaîne Commit puis CHECKPOINT
	// immédiat. Toute réponse 200 vaut donc « écrit et durable ».
	//
	// Pourquoi c'est indispensable ICI et pas un détail de perf : un tx.Commit()
	// nu laisse le like dans le WAL jusqu'au prochain tick du scheduler
	// périodique (5 min, cmd/server/main.go) ou du CHECKPOINT de shutdown. Tout
	// redémarrage dans cette fenêtre — et un push sur main déploie en prod — ou
	// toute quarantaine de WAL orphelin (ADR 0021) fait DISPARAÎTRE le like
	// après coup : l'UI l'affiche (réponse 200), puis il s'évapore au rechargement
	// suivant. C'est la classe de régression « le like ne tient pas », observée
	// plusieurs fois. Le commentaire de cmd/server/main.go désignait déjà les
	// likes comme cas d'usage de CommitWithCheckpoint ; ce call-site ne l'utilisait
	// pas.
	//
	// Ne pas revenir à tx.Commit() ici : verrouillé par
	// TestMediaService_SetMediaLike_Atomic_SurvivesWALLoss.
	if err := w.CommitWithCheckpoint(ctx, tx); err != nil {
		return nil, fmt.Errorf("media_like commit: %w", err)
	}
	return s.buildLikeResponse(ctx, req), nil
}

// setMediaLikeLegacy conserve le comportement pré-commit-6 (non-atomique).
// Les tests existants qui passent un mock sans SetMediaLikeAtomic continuent
// à exercer ce chemin.
func (s *MediaService) setMediaLikeLegacy(
	ctx context.Context,
	req domain.MediaLikeRequest,
) (*domain.MediaLikeResponse, error) {
	ok, err := s.repo.MediaExists(ctx, req.FilePath)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.ErrNotFound("media", req.FilePath)
	}

	// Écriture de l'event de like si on a un likerSlug. Sans liker, le like n'a
	// plus aucun support de stockage depuis le passage au par-viewer.
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

	return s.buildLikeResponse(ctx, req), nil
}

// buildLikeResponse construit la réponse en récupérant les likers (lecture
// best-effort, pas d'erreur si la requête échoue).
//
// `Liked` reflète l'état DU LIKER qui vient d'agir (donc du viewer de la
// session) ; `LikeCount`/`TotalLikers`/`Likers` restent GLOBAUX (tous likers
// confondus). C'est la même asymétrie que dans la galerie : un cœur personnel,
// un compteur social.
func (s *MediaService) buildLikeResponse(ctx context.Context, req domain.MediaLikeRequest) *domain.MediaLikeResponse {
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
	return resp
}
