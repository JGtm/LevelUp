// Package handlers — media.go : handler HTTP pour la galerie médias.
//
// Endpoints :
//
//	POST /api/v1/players/{player_slug}/pages/media            → MediaPageResponse
//	PATCH /api/v1/players/{player_slug}/media/likes            → MediaLikeResponse
//	POST /api/v1/players/{player_slug}/media/upload            → UploadResult (multipart)
//	POST /api/v1/players/{player_slug}/media/reassociate       → ReassociateResult
//	GET  /api/v1/players/{player_slug}/media/files/*           → fichier servi depuis captures
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
)

// maxUploadSize limite la taille totale d'un upload à 500 Mo.
const maxUploadSize = 500 << 20

// maxUploadFiles limite le nombre de fichiers par requête.
const maxUploadFiles = 50

// allowedMediaExts liste les extensions acceptées en upload.
var allowedMediaExts = map[string]bool{
	".mp4": true, ".mov": true, ".avi": true, ".mkv": true, ".webm": true,
	".png": true, ".jpg": true, ".jpeg": true, ".bmp": true, ".gif": true,
}

// MediaUploadContextFactory retourne service + gamertag + titleSlug + dbPath + sharedSocialDBPath + sharedMatchesDBPath.
// Utilisée par PostUploadMedia pour résoudre le répertoire captures.
type MediaUploadContextFactory func(ctx context.Context, slug string) (
	port.MediaService, string, string, string, string, string, error,
)

// MediaPlayerContextFactory résout slug → (titleSlug, gamertag) sans construire de service.
// Utilisée par les endpoints méta-données (ex : liste des auteurs).
type MediaPlayerContextFactory func(ctx context.Context, slug string) (titleSlug, gamertag string, err error)

// MediaProfilesProvider liste les profils connus pour un titre (db_profiles.json).
type MediaProfilesProvider func(ctx context.Context, titleSlug string) ([]domain.PlayerSummary, error)

// MediaHandler gère les endpoints de la galerie médias.
type MediaHandler struct {
	newSvc            ServiceFactory[port.MediaService]
	newUpload         MediaUploadContextFactory
	newPlayerCtx      MediaPlayerContextFactory
	loadProfiles      MediaProfilesProvider
	repoRoot          string
	settingsStore     *settings.Store
	notifierFor       NotificationsEmitterFactory // optionnel : émission media_added
	recipientResolver MediaRecipientResolver      // optionnel : fan-out aux autres joueurs
	demoMode          bool                        // true = upload figé (vitrine publique)
}

// NewMediaHandler crée un MediaHandler.
// newUpload peut être nil : dans ce cas l'endpoint upload retourne 501.
func NewMediaHandler(
	newSvc ServiceFactory[port.MediaService],
	newUpload MediaUploadContextFactory,
	repoRoot string,
) *MediaHandler {
	return &MediaHandler{newSvc: newSvc, newUpload: newUpload, repoRoot: repoRoot}
}

// WithSettingsStore injecte le settings store pour lire media_captures_base_dir.
func (h *MediaHandler) WithSettingsStore(store *settings.Store) *MediaHandler {
	h.settingsStore = store
	return h
}

// WithDemoMode fige l'upload de médias en mode démo (vitrine publique partagée).
// Sans appel : false (upload autorisé).
func (h *MediaHandler) WithDemoMode(demo bool) *MediaHandler {
	h.demoMode = demo
	return h
}

// WithAuthorsContext câble la résolution slug → (titleSlug, gamertag) et la liste
// des profils du titre. Optionnel : sert uniquement à enrichir le gamertag d'affichage
// dans GetMediaAuthors ; sans ces callbacks, l'endpoint retombe sur le player_slug
// comme libellé (la liste d'auteurs elle-même vient de la DB via le service).
func (h *MediaHandler) WithAuthorsContext(playerCtx MediaPlayerContextFactory, loadProfiles MediaProfilesProvider) *MediaHandler {
	h.newPlayerCtx = playerCtx
	h.loadProfiles = loadProfiles
	return h
}

// WithNotificationsEmitterFactory branche la factory d'émetteurs de notifications
// pour publier media_added après un upload réussi.
func (h *MediaHandler) WithNotificationsEmitterFactory(f NotificationsEmitterFactory) *MediaHandler {
	h.notifierFor = f
	return h
}

// MediaRecipientResolver retourne la liste des player_slug à notifier après un
// upload, à partir du sharedSocialDBPath et du sharedMatchesDBPath. Doit exclure
// l'uploader (passé via uploaderSlug).
type MediaRecipientResolver func(
	ctx context.Context,
	uploaderSlug, sharedSocialDBPath, sharedMatchesDBPath string,
	since time.Time,
) ([]string, error)

// WithMediaRecipientResolver branche le resolver de destinataires (fan-out).
// Si nil, l'émission media_added reste limitée à l'uploader.
func (h *MediaHandler) WithMediaRecipientResolver(r MediaRecipientResolver) *MediaHandler {
	h.recipientResolver = r
	return h
}

// emitMediaAdded émet media_added pour l'uploader puis fan-out vers les autres
// joueurs associés aux matchs concernés.
func (h *MediaHandler) emitMediaAdded(
	ctx context.Context,
	uploaderSlug, gamertag string,
	newIndexed int,
	since time.Time,
	sharedSocialDBPath, sharedMatchesDBPath string,
) {
	if h.notifierFor == nil || newIndexed <= 0 {
		return
	}
	// Fan-out aux autres joueurs (max ~5 destinataires).
	recipients := []string{}
	if h.recipientResolver != nil {
		var err error
		recipients, err = h.recipientResolver(ctx, uploaderSlug, sharedSocialDBPath, sharedMatchesDBPath, since)
		if err != nil {
			slog.WarnContext(ctx, "notifications: media_added recipients", "err", err)
		}
	}
	for _, slug := range recipients {
		if slug == uploaderSlug {
			continue // safety : exclure l'uploader (le resolver est censé le faire aussi)
		}
		em, err := h.notifierFor(ctx, slug)
		if err != nil || em == nil {
			continue
		}
		_ = em.Emit(ctx, notifications.EmitInput{
			Category:    notifications.CategoryMediaAdded,
			Severity:    notifications.SeverityInfo,
			TitleKey:    "notif.media_added.title",
			BodyKey:     "notif.media_added.body",
			Params:      map[string]any{"actor_name": gamertag, jsonKeyCount: newIndexed},
			TargetRoute: fmt.Sprintf("/players/%s/media", slug),
			Actor:       &notifications.Actor{Name: gamertag},
			Source:      "media_handler",
		})
	}
}

// GetMediaLibrary retourne la page paginée de la galerie médias.
// POST /api/v1/players/{player_slug}/pages/media
// Body (optionnel) : { "page": 1, "page_size": 24, "kind": "clip" }
func (h *MediaHandler) GetMediaLibrary(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	req := domain.MediaPageRequest{Page: 1}
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
			return
		}
	}
	if req.Page < 1 {
		req.Page = 1
	}

	resp, err := svc.GetMediaPage(r.Context(), req)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "media_page_error", err.Error())
		return
	}

	// Transformer les chemins absolus en URLs servables
	h.transformMediaURLs(slug, resp.Items.Items)

	writeJSON(w, http.StatusOK, resp)
}

// PatchMediaLike persiste le like/unlike d'un média.
// PATCH /api/v1/players/{player_slug}/media/likes
func (h *MediaHandler) PatchMediaLike(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.MediaLikeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	// Le frontend reçoit les file_path déjà transformés en URL HTTP
	// (cf. transformMediaURLs / filePathToURL). Quand il rejoue ce file_path
	// dans une mutation (like, réassociation), on doit reverser la transformation
	// vers le chemin absolu de stockage tel que présent en DB.
	rawPath := req.FilePath
	req.FilePath = h.urlToFilePath(slug, req.FilePath)
	slog.DebugContext(r.Context(), "media_like: path resolved",
		"slug", slug, "raw", rawPath, "resolved", req.FilePath)

	// Auto-injecter le liker depuis la session si absent du body.
	// Sans liker_slug, le service ne peuple pas media_likes (table partagée
	// entre joueurs) → les badges "♥ Alice et Bob" ne s'affichent pas.
	if req.LikerSlug == "" {
		if sess := middleware.GetSession(r.Context()); sess != nil && sess.CurrentPlayerSlug != nil {
			req.LikerSlug = *sess.CurrentPlayerSlug
			if req.LikerGamertag == "" {
				req.LikerGamertag = h.resolveLikerGamertag(r.Context(), *sess.CurrentPlayerSlug)
			}
		}
	}

	resp, err := svc.SetMediaLike(r.Context(), req)
	if err != nil {
		if errors.Is(err, dblease.ErrDBLocked) {
			w.Header().Set("Retry-After", "5")
			writeError(r.Context(), w, http.StatusServiceUnavailable, "db_busy",
				"database is currently busy, please retry")
			return
		}
		var apiErr *domain.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.Code {
			case "bad_request":
				writeError(r.Context(), w, http.StatusBadRequest, apiErr.Code, apiErr.Message)
				return
			case "not_found":
				writeError(r.Context(), w, http.StatusNotFound, apiErr.Code, apiErr.Message)
				return
			}
		}
		writeError(r.Context(), w, http.StatusInternalServerError, "media_like_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
	BumpMediaFeedVersion()

	// Notification au owner si quelqu'un d'autre a liké son média.
	if req.Liked && req.LikerSlug != "" {
		h.emitMediaLiked(r.Context(), req.FilePath, req.LikerSlug, req.LikerGamertag)
	}
}

// emitMediaLiked notifie le owner d'un média quand quelqu'un d'autre le like.
// Le owner_slug est déduit du file_path (.../Captures/<slug>/file.ext).
// Silent fail si notifierFor absent, liker == owner, ou owner indéterminable.
func (h *MediaHandler) emitMediaLiked(ctx context.Context, filePath, likerSlug, likerGamertag string) {
	if h.notifierFor == nil {
		return
	}
	ownerSlug := ownerSlugFromFilePath(filePath)
	if ownerSlug == "" || ownerSlug == likerSlug {
		return
	}
	em, err := h.notifierFor(ctx, ownerSlug)
	if err != nil || em == nil {
		return
	}
	displayName := likerGamertag
	if displayName == "" {
		displayName = likerSlug
	}
	_ = em.Emit(ctx, notifications.EmitInput{
		Category:    notifications.CategoryMediaLiked,
		Severity:    notifications.SeverityInfo,
		TitleKey:    "notif.media_liked.title",
		BodyKey:     "notif.media_liked.body",
		Params:      map[string]any{"actor_name": displayName},
		TargetRoute: fmt.Sprintf("/players/%s/media", ownerSlug),
		Actor:       &notifications.Actor{Name: displayName},
		Source:      "media_handler",
	})
}

// ownerSlugFromFilePath extrait le slug du joueur propriétaire depuis le chemin.
// Conventions reconnues :
//   - .../Captures/<slug>/file.ext         (capturesBase utilisateur, ex. Windows Captures)
//   - .../players/<slug>/captures/file.ext (chemin interne data/)
//
// Retourne "" si la convention n'est pas détectée.
func ownerSlugFromFilePath(filePath string) string {
	clean := filepath.Clean(filePath)
	parts := strings.Split(clean, string(filepath.Separator))
	for i := 0; i < len(parts)-1; i++ {
		if strings.EqualFold(parts[i], "Captures") && i+1 < len(parts)-1 {
			return parts[i+1]
		}
		if parts[i] == "captures" && i > 0 && i+1 < len(parts) {
			// .../<slug>/captures/file.ext → slug est avant
			return parts[i-1]
		}
	}
	return ""
}

// PostUploadMedia reçoit des fichiers via multipart/form-data, les sauvegarde
// dans le répertoire captures du joueur, puis déclenche l'indexation immédiate.
// POST /api/v1/players/{player_slug}/media/upload
func (h *MediaHandler) GetMediaMatchCandidates(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	filePath := r.URL.Query().Get("file_path")
	if filePath == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_file_path", "file_path query param requis")
		return
	}
	window := 15
	if w := r.URL.Query().Get("window_minutes"); w != "" {
		if n, err := strconv.Atoi(w); err == nil && n > 0 {
			window = n
		}
	}
	resp, err := svc.GetMatchCandidates(r.Context(), filePath, window)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "candidates_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// PostMediaAssociate force l'association d'un média à un match précis.
// POST /api/v1/players/{player_slug}/media/associate { file_path, match_id }
func (h *MediaHandler) PostMediaAssociate(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	var req domain.MediaAssociateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	resp, err := svc.AssociateMediaToMatch(r.Context(), req)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "associate_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
	BumpMediaFeedVersion()
}

// GetMediaAuthors retourne la liste des auteurs sélectionnables dans le filtre
// Auteurs : les player_slug distincts présents dans shared_social.media_files
// (même source que la galerie), enrichis du gamertag d'affichage via db_profiles.json.
//
// Historique : cet endpoint scannait le filesystem (countMediaInDir) par gamertag,
// ce qui renvoyait une liste vide dès que les captures d'un auteur n'étaient pas
// présentes sur le disque local (multi-user) ou rangées en sous-dossiers — d'où le
// bug "Aucun auteur disponible" alors que la galerie affichait bien des médias. La
// source est désormais la DB, strictement cohérente avec ce que la galerie peut afficher.
// GET /api/v1/players/{player_slug}/media/authors
func (h *MediaHandler) GetMediaAuthors(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")

	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	authors, err := svc.ListMediaAuthors(r.Context())
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "authors_query_error", err.Error())
		return
	}

	// Enrichissement du gamertag d'affichage (best-effort) depuis db_profiles.json.
	// Sans contexte profils câblé (WithAuthorsContext), on retombe sur le player_slug.
	gamertagBySlug := h.authorGamertags(r.Context(), slug)
	for i := range authors {
		if gt := gamertagBySlug[strings.ToLower(authors[i].PlayerSlug)]; gt != "" {
			authors[i].Gamertag = gt
		} else {
			authors[i].Gamertag = authors[i].PlayerSlug
		}
	}

	writeJSON(w, http.StatusOK, domain.MediaAuthorsResponse{Authors: authors})
}

// authorGamertags retourne un index lower(slug|gamertag) → gamertag construit depuis
// db_profiles.json, pour afficher un libellé propre dans le filtre Auteurs. Best-effort :
// retourne une map vide si le contexte profils n'est pas câblé (WithAuthorsContext).
func (h *MediaHandler) authorGamertags(ctx context.Context, slug string) map[string]string {
	out := map[string]string{}
	if h.newPlayerCtx == nil || h.loadProfiles == nil {
		return out
	}
	titleSlug, _, err := h.newPlayerCtx(ctx, slug)
	if err != nil {
		return out
	}
	profiles, err := h.loadProfiles(ctx, titleSlug)
	if err != nil {
		return out
	}
	for _, p := range profiles {
		if p.Gamertag == "" {
			continue
		}
		out[strings.ToLower(p.PlayerSlug)] = p.Gamertag
		out[strings.ToLower(p.Gamertag)] = p.Gamertag
	}
	return out
}
