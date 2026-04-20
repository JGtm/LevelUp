// Package handlers — media.go : handler HTTP pour la galerie médias.
//
// Endpoints :
//
//	POST /api/v1/players/{player_slug}/pages/media    → MediaPageResponse
//	PATCH /api/v1/players/{player_slug}/media/likes   → MediaLikeResponse
//	POST /api/v1/players/{player_slug}/media/upload   → UploadResult (multipart)
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
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

// MediaUploadContextFactory retourne service + gamertag + titleSlug + dbPath + sharedSocialDBPath.
// Utilisée par PostUploadMedia pour résoudre le répertoire captures.
type MediaUploadContextFactory func(ctx context.Context, slug string) (
	port.MediaService, string, string, string, string, error,
)

// MediaHandler gère les endpoints de la galerie médias.
type MediaHandler struct {
	newSvc    ServiceFactory[port.MediaService]
	newUpload MediaUploadContextFactory
}

// NewMediaHandler crée un MediaHandler.
// newUpload peut être nil : dans ce cas l'endpoint upload retourne 501.
func NewMediaHandler(
	newSvc ServiceFactory[port.MediaService],
	newUpload MediaUploadContextFactory,
) *MediaHandler {
	return &MediaHandler{newSvc: newSvc, newUpload: newUpload}
}

// GetMediaLibrary retourne la page paginée de la galerie médias.
// POST /api/v1/players/{player_slug}/pages/media
// Body (optionnel) : { "page": 1, "page_size": 24, "kind": "clip" }
func (h *MediaHandler) GetMediaLibrary(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	req := domain.MediaPageRequest{Page: 1}
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
			return
		}
	}
	if req.Page < 1 {
		req.Page = 1
	}

	resp, err := svc.GetMediaPage(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "media_page_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// PatchMediaLike persiste le like/unlike d'un média.
// PATCH /api/v1/players/{player_slug}/media/likes
func (h *MediaHandler) PatchMediaLike(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.MediaLikeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	resp, err := svc.SetMediaLike(r.Context(), req)
	if err != nil {
		var apiErr *domain.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.Code {
			case "bad_request":
				writeError(w, http.StatusBadRequest, apiErr.Code, apiErr.Message)
				return
			case "not_found":
				writeError(w, http.StatusNotFound, apiErr.Code, apiErr.Message)
				return
			}
		}
		writeError(w, http.StatusInternalServerError, "media_like_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
	BumpMediaFeedVersion()
}

// PostUploadMedia reçoit des fichiers via multipart/form-data, les sauvegarde
// dans le répertoire captures du joueur, puis déclenche l'indexation immédiate.
// POST /api/v1/players/{player_slug}/media/upload
func (h *MediaHandler) PostUploadMedia(w http.ResponseWriter, r *http.Request) {
	if h.newUpload == nil {
		writeError(w, http.StatusNotImplemented, "upload_not_configured", "upload factory non configurée")
		return
	}

	slug := chi.URLParam(r, "player_slug")
	svc, gamertag, titleSlug, dbPath, sharedSocialDBPath, err := h.newUpload(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "upload_too_large",
			"fichiers trop volumineux (max 500 Mo)")
		return
	}

	files, err := parseUploadedFiles(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "upload_parse_error", err.Error())
		return
	}
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, "no_valid_files",
			"aucun fichier valide (extensions acceptées : mp4 mov avi mkv webm png jpg jpeg bmp gif)")
		return
	}

	capturesDir := resolveCapturesDir(titleSlug, gamertag)
	slog.InfoContext(r.Context(), "upload: requête reçue",
		"player", gamertag, "files", len(files), "captures_dir", capturesDir)

	req := domain.UploadRequest{
		Files:              files,
		CapturesDir:        capturesDir,
		DBPath:             dbPath,
		SharedSocialDBPath: sharedSocialDBPath,
		Tolerance:          5,
	}

	result, err := svc.UploadMedia(r.Context(), req)
	if err != nil {
		slog.ErrorContext(r.Context(), "upload: erreur service", "err", err)
		writeError(w, http.StatusInternalServerError, "upload_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
	BumpMediaFeedVersion()
}

// parseUploadedFiles extrait et valide les fichiers du formulaire multipart.
// Le champ attendu est "files". Limite : maxUploadFiles fichiers.
func parseUploadedFiles(r *http.Request) ([]domain.UploadedFile, error) {
	headers := r.MultipartForm.File["files"]
	if len(headers) == 0 {
		return nil, nil
	}
	if len(headers) > maxUploadFiles {
		return nil, fmt.Errorf("trop de fichiers : %d (max %d)", len(headers), maxUploadFiles)
	}

	var out []domain.UploadedFile
	for _, fh := range headers {
		ext := strings.ToLower(filepath.Ext(fh.Filename))
		if !allowedMediaExts[ext] {
			slog.Warn("upload: extension refusée", "file", fh.Filename, "ext", ext)
			continue
		}
		f, err := fh.Open()
		if err != nil {
			return nil, fmt.Errorf("lecture %s: %w", fh.Filename, err)
		}
		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("lecture données %s: %w", fh.Filename, err)
		}
		out = append(out, domain.UploadedFile{
			OriginalName: fh.Filename,
			Data:         data,
		})
	}
	return out, nil
}

// resolveCapturesDir construit le chemin captures title-aware.
// Utilise le même calcul que PathResolver.PlayerCapturesDir.
// Pour une résolution complète (avec repoRoot), le handler devrait recevoir
// le PathResolver — ici on utilise le chemin relatif standard.
func resolveCapturesDir(titleSlug, gamertag string) string {
	if titleSlug != "" {
		return filepath.Join("data", "titles", titleSlug, "players", gamertag, "captures")
	}
	return filepath.Join("data", "players", gamertag, "captures")
}

// ---------------------------------------------------------------------------
// Feed-version — polling léger pour rafraîchir la galerie
// ---------------------------------------------------------------------------

// mediaFeedVersion est incrémenté à chaque upload ou like pour que les
// clients puissent détecter les changements sans ouvrir un websocket.
var mediaFeedVersion int64

// BumpMediaFeedVersion incrémente le compteur de version de la galerie.
// Appelé après upload ou toggle like.
func BumpMediaFeedVersion() {
	atomic.AddInt64(&mediaFeedVersion, 1)
}

// GetMediaFeedVersion retourne la version courante du flux médias.
// GET /api/v1/media/feed-version
func GetMediaFeedVersion(w http.ResponseWriter, _ *http.Request) {
	v := atomic.LoadInt64(&mediaFeedVersion)
	writeJSON(w, http.StatusOK, map[string]int64{"version": v})
}
