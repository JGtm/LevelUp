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
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
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

// WithAuthorsContext câble la résolution slug → (titleSlug, gamertag) et la liste
// des profils du titre. Sans ces deux callbacks, GetMediaAuthors retourne 501.
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
			Params:      map[string]any{"actor_name": gamertag, "count": newIndexed},
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
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.MediaLikeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	// Le frontend reçoit les file_path déjà transformés en URL HTTP
	// (cf. transformMediaURLs / filePathToURL). Quand il rejoue ce file_path
	// dans une mutation (like, réassociation), on doit reverser la transformation
	// vers le chemin absolu de stockage tel que présent en DB.
	req.FilePath = h.urlToFilePath(slug, req.FilePath)

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
			writeError(w, http.StatusServiceUnavailable, "db_busy",
				"database is currently busy, please retry")
			return
		}
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
func (h *MediaHandler) PostUploadMedia(w http.ResponseWriter, r *http.Request) {
	if h.newUpload == nil {
		writeError(w, http.StatusNotImplemented, "upload_not_configured", "upload factory non configurée")
		return
	}

	slug := chi.URLParam(r, "player_slug")
	svc, gamertag, titleSlug, dbPath, sharedSocialDBPath, sharedMatchesDBPath, err := h.newUpload(r.Context(), slug)
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

	capturesDir := h.resolveCapturesDir(titleSlug, gamertag)
	slog.InfoContext(r.Context(), "upload: requête reçue",
		"player", gamertag, "files", len(files), "captures_dir", capturesDir)

	req := domain.UploadRequest{
		Files:               files,
		CapturesDir:         capturesDir,
		DBPath:              dbPath,
		SharedSocialDBPath:  sharedSocialDBPath,
		SharedMatchesDBPath: sharedMatchesDBPath,
		Tolerance:           2,
	}

	uploadStart := time.Now().UTC()
	result, err := svc.UploadMedia(r.Context(), req)
	if err != nil {
		slog.ErrorContext(r.Context(), "upload: erreur service", "err", err)
		writeError(w, http.StatusInternalServerError, "upload_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
	BumpMediaFeedVersion()
	h.emitMediaAdded(r.Context(), slug, gamertag, result.NewIndexed,
		uploadStart, sharedSocialDBPath, sharedMatchesDBPath)
}

// parseUploadedFiles extrait et valide les fichiers du formulaire multipart.
// Le champ attendu est "files". Un champ JSON optionnel "capture_times" peut
// fournir un tableau d'entiers Unix (secondes) aligné sur l'ordre des fichiers.
// Limite : maxUploadFiles fichiers.
func parseUploadedFiles(r *http.Request) ([]domain.UploadedFile, error) {
	headers := r.MultipartForm.File["files"]
	if len(headers) == 0 {
		return nil, nil
	}
	if len(headers) > maxUploadFiles {
		return nil, fmt.Errorf("trop de fichiers : %d (max %d)", len(headers), maxUploadFiles)
	}

	// Lire les capture_times client (tableau JSON d'entiers Unix, même ordre que files).
	var clientTimes []int64
	if raw := r.FormValue("capture_times"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &clientTimes); err != nil {
			slog.Warn("parseUploadedFiles: capture_times JSON invalide, ignoré", "err", err)
			clientTimes = nil
		}
	}

	out := make([]domain.UploadedFile, 0, len(headers))
	for i, fh := range headers {
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
		uf := domain.UploadedFile{
			OriginalName: fh.Filename,
			Data:         data,
		}
		if i < len(clientTimes) && clientTimes[i] > 0 {
			ts := clientTimes[i]
			uf.CaptureTimeUnix = &ts
		}
		out = append(out, uf)
	}
	return out, nil
}

// GetMediaMatchCandidates retourne les matchs candidats pour réassocier un média.
// GET /api/v1/players/{player_slug}/media/match-candidates?file_path=...&window_minutes=15
func (h *MediaHandler) GetMediaMatchCandidates(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	filePath := r.URL.Query().Get("file_path")
	if filePath == "" {
		writeError(w, http.StatusBadRequest, "missing_file_path", "file_path query param requis")
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
		writeError(w, http.StatusInternalServerError, "candidates_error", err.Error())
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
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	var req domain.MediaAssociateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	resp, err := svc.AssociateMediaToMatch(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "associate_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
	BumpMediaFeedVersion()
}

// GetMediaAuthors retourne la liste des profils db_profiles.json ayant au moins
// un fichier média dans leur dossier captures (croisement avec le filesystem).
// GET /api/v1/players/{player_slug}/media/authors
func (h *MediaHandler) GetMediaAuthors(w http.ResponseWriter, r *http.Request) {
	if h.newPlayerCtx == nil || h.loadProfiles == nil {
		writeError(w, http.StatusNotImplemented, "authors_not_configured", "authors context non configuré")
		return
	}

	slug := chi.URLParam(r, "player_slug")
	titleSlug, currentGamertag, err := h.newPlayerCtx(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	profiles, err := h.loadProfiles(r.Context(), titleSlug)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "profiles_load_error", err.Error())
		return
	}

	authors := make([]domain.MediaAuthor, 0, len(profiles))
	for _, p := range profiles {
		dir := h.resolveCapturesDir(titleSlug, p.Gamertag)
		count := countMediaInDir(dir)
		if count == 0 {
			continue
		}
		authors = append(authors, domain.MediaAuthor{
			PlayerSlug: p.PlayerSlug,
			Gamertag:   p.Gamertag,
			IsSelf:     strings.EqualFold(p.Gamertag, currentGamertag),
			MediaCount: count,
		})
	}

	writeJSON(w, http.StatusOK, domain.MediaAuthorsResponse{Authors: authors})
}

// countMediaInDir compte les fichiers média (extensions allowedMediaExts) dans un
// dossier — best-effort, retourne 0 sur erreur ou dossier inexistant. Ne descend
// pas en récursif (le sous-dossier `thumbs/` est ignoré naturellement comme dossier).
func countMediaInDir(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if allowedMediaExts[strings.ToLower(filepath.Ext(e.Name()))] {
			n++
		}
	}
	return n
}

// resolveCapturesDir construit le chemin captures pour un joueur en s'appuyant
// sur l'helper canonique PathResolver.ResolveCapturesDir.
// Si media_captures_base_dir est défini dans les settings, utilise {baseDir}/{gamertag}.
// Sinon, fallback sur le chemin interne data/titles/.../players/{gamertag}/captures/.
func (h *MediaHandler) resolveCapturesDir(titleSlug, gamertag string) string {
	baseDir := ""
	if h.settingsStore != nil {
		if cfg, err := h.settingsStore.Load(); err == nil {
			baseDir = cfg.MediaCapturesBaseDir
		}
	}
	return resolveCapturesDirWith(h.repoRoot, titleSlug, gamertag, baseDir)
}

// resolveCapturesDir construit le chemin captures title-aware via PathResolver
// sans accès aux settings (toujours fallback interne). Conservée pour les
// call-sites qui ne disposent pas d'un settingsStore (tests, code legacy).
// repoRoot peut être vide : dans ce cas le chemin retourné est relatif (à éviter en production).
func resolveCapturesDir(repoRoot, titleSlug, gamertag string) string {
	return resolveCapturesDirWith(repoRoot, titleSlug, gamertag, "")
}

// resolveCapturesDirWith est l'adapter mince autour de PathResolver.ResolveCapturesDir
// qui normalise le titleSlug vide vers DefaultSlug — toutes les autres logiques
// (fallback interne, jonction baseDir+gamertag) vivent dans le PathResolver.
func resolveCapturesDirWith(repoRoot, titleSlug, gamertag, baseDir string) string {
	if titleSlug == "" {
		titleSlug = titlePkg.DefaultSlug
	}
	return titlePkg.NewPathResolver(repoRoot).ResolveCapturesDir(titleSlug, gamertag, baseDir)
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

// filePathToURL transforme un chemin absolu en URL servable via l'API.
// Bug #8 : 3 tentatives — capturesBase+slug (multi-player), capturesBase brut
// (single-player, dossier captures sans sous-folder slug), repoRoot interne.
func (h *MediaHandler) filePathToURL(slug, absPath, capturesBase string) string {
	clean := filepath.Clean(absPath)

	if capturesBase != "" {
		playerDir := filepath.Join(capturesBase, slug)
		if rel, ok := relIfWithin(playerDir, clean); ok {
			return "/api/v1/players/" + slug + "/media/files/" + filepath.ToSlash(rel)
		}
		// Single-player : capturesBase pointe directement sur le dossier
		// captures sans sous-folder slug (cas typique production).
		if rel, ok := relIfWithin(capturesBase, clean); ok {
			return "/api/v1/players/" + slug + "/media/files/" + filepath.ToSlash(rel)
		}
	}

	if h.repoRoot != "" {
		pr := titlePkg.NewPathResolver(h.repoRoot)
		internalCapturesDir := pr.PlayerCapturesDir(titlePkg.DefaultSlug, slug)
		internalPlayerDir := filepath.Dir(internalCapturesDir)
		if rel, ok := relIfWithin(internalPlayerDir, clean); ok {
			return "/api/v1/players/" + slug + "/media/files/" + filepath.ToSlash(rel)
		}
	}

	slog.Warn("filePathToURL: aucun mapping trouvé",
		"slug", slug, "abs_path", absPath, "captures_base", capturesBase)
	return absPath
}

// relIfWithin retourne (rel, true) si target est sous base.
func relIfWithin(base, target string) (string, bool) {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "", false
	}
	if strings.HasPrefix(rel, "..") || rel == "." || rel == "" {
		return "", false
	}
	return rel, true
}

// urlToFilePath fait l'inverse de filePathToURL : convertit une URL servable
// `/api/v1/players/{slug}/media/files/{relPath}` en chemin absolu de stockage.
// Si l'entrée n'est pas une URL transformée (déjà un chemin absolu, par exemple),
// retourne tel quel.
func (h *MediaHandler) urlToFilePath(slug, input string) string {
	prefix := "/api/v1/players/" + slug + "/media/files/"
	if !strings.HasPrefix(input, prefix) {
		return input
	}
	relPath := strings.TrimPrefix(input, prefix)
	relPath = filepath.FromSlash(relPath)

	// Tentative 1 : capturesBase configuré.
	capturesBase := ""
	if h.settingsStore != nil {
		if cfg, err := h.settingsStore.Load(); err == nil && cfg.MediaCapturesBaseDir != "" {
			capturesBase = filepath.Clean(cfg.MediaCapturesBaseDir)
		}
	}
	if capturesBase != "" {
		candidate := filepath.Join(capturesBase, slug, relPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// Tentative 2 : chemin interne repo data/.
	if h.repoRoot != "" {
		pr := titlePkg.NewPathResolver(h.repoRoot)
		internalDir := filepath.Dir(pr.PlayerCapturesDir(titlePkg.DefaultSlug, slug))
		candidate := filepath.Join(internalDir, relPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// Si on ne trouve pas le fichier, renvoyer l'input pour laisser le repo
	// retourner un 404 explicite.
	return input
}

// resolveLikerGamertag résout le gamertag affichable depuis le slug du joueur
// connecté (pour le badge "♥ Alice et Bob"). Fallback sur le slug si la
// résolution échoue.
func (h *MediaHandler) resolveLikerGamertag(ctx context.Context, slug string) string {
	if h.newPlayerCtx == nil {
		return slug
	}
	_, gamertag, err := h.newPlayerCtx(ctx, slug)
	if err != nil || gamertag == "" {
		return slug
	}
	return gamertag
}

// transformMediaURLs transforme les chemins absolus des items en URLs servables.
func (h *MediaHandler) transformMediaURLs(slug string, items []domain.MediaItem) {
	capturesBase := ""
	if h.settingsStore != nil {
		if cfg, err := h.settingsStore.Load(); err == nil && cfg.MediaCapturesBaseDir != "" {
			capturesBase = filepath.Clean(cfg.MediaCapturesBaseDir)
		}
	}
	for i := range items {
		items[i].FilePath = h.filePathToURL(slug, items[i].FilePath, capturesBase)
		if items[i].ThumbnailPath != nil {
			u := h.filePathToURL(slug, *items[i].ThumbnailPath, capturesBase)
			items[i].ThumbnailPath = &u
		}
	}
}

// ServeMediaFile sert un fichier depuis le répertoire captures du joueur.
// Essaie d'abord capturesBase (settings), puis le chemin interne data/ du repo.
// GET /api/v1/players/{player_slug}/media/files/*
func (h *MediaHandler) ServeMediaFile(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	rpath := chi.URLParam(r, "*")

	// Nettoyer le chemin URL (gère les "..")
	cleanURL := path.Clean("/" + rpath)
	if strings.Contains(cleanURL, "..") {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	cleanURL = strings.TrimPrefix(cleanURL, "/")
	if cleanURL == "" || cleanURL == "." {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	capturesBase := ""
	if h.settingsStore != nil {
		if cfg, err := h.settingsStore.Load(); err == nil {
			capturesBase = cfg.MediaCapturesBaseDir
		}
	}

	// Bug #8 : symétrique de filePathToURL — capturesBase+slug ET capturesBase brut.
	var playerDirs []string
	if capturesBase != "" {
		playerDirs = append(playerDirs, filepath.Join(capturesBase, slug))
		playerDirs = append(playerDirs, filepath.Clean(capturesBase))
	}
	if h.repoRoot != "" {
		pr := titlePkg.NewPathResolver(h.repoRoot)
		internalCapturesDir := pr.PlayerCapturesDir(titlePkg.DefaultSlug, slug)
		playerDirs = append(playerDirs, filepath.Dir(internalCapturesDir))
	}
	if len(playerDirs) == 0 {
		http.Error(w, "file serving not configured", http.StatusServiceUnavailable)
		return
	}

	for _, playerDir := range playerDirs {
		absPath := filepath.Join(playerDir, filepath.FromSlash(cleanURL))

		// Anti-traversal : vérifier que le chemin résolu est dans playerDir
		cleanPlayerDir := filepath.Clean(playerDir)
		cleanAbsPath := filepath.Clean(absPath)
		if cleanAbsPath != cleanPlayerDir &&
			!strings.HasPrefix(cleanAbsPath, cleanPlayerDir+string(filepath.Separator)) {
			continue
		}

		if _, err := os.Stat(cleanAbsPath); err == nil {
			if w.Header().Get("Content-Type") == "" {
				switch strings.ToLower(filepath.Ext(cleanAbsPath)) {
				case ".mp4":
					w.Header().Set("Content-Type", "video/mp4")
				case ".webm":
					w.Header().Set("Content-Type", "video/webm")
				case ".mov":
					w.Header().Set("Content-Type", "video/quicktime")
				case ".avi":
					w.Header().Set("Content-Type", "video/x-msvideo")
				case ".mkv":
					w.Header().Set("Content-Type", "video/x-matroska")
				case ".webp":
					w.Header().Set("Content-Type", "image/webp")
				}
			}
			http.ServeFile(w, r, cleanAbsPath)
			return
		}
	}

	http.Error(w, "not found", http.StatusNotFound)
}
