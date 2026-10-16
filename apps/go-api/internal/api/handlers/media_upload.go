package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	mediapkg "levelup/go-api/internal/media"
	"levelup/go-api/internal/platform/mediaaudio"
)

// uploadBusyRetryAfterSec : délai conseillé (Retry-After) à un envoi refusé faute de place.
const uploadBusyRetryAfterSec = "30"

func (h *MediaHandler) PostUploadMedia(w http.ResponseWriter, r *http.Request) {
	if h.newUpload == nil {
		writeError(r.Context(), w, http.StatusNotImplemented, "upload_not_configured", "upload factory non configurée")
		return
	}

	slug := chi.URLParam(r, "player_slug")
	svc, gamertag, titleSlug, dbPath, sharedSocialDBPath, sharedMatchesDBPath, err := h.newUpload(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	// Au plus maxConcurrentUploads envois en vol pour tout le process : un envoi de plus
	// attend sa place, puis reçoit un 503 upload_busy clair s'il l'attend trop longtemps.
	release, ok := h.acquireUploadSlot(r.Context(), w)
	if !ok {
		return
	}
	defer release()

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(r.Context(), w, http.StatusRequestEntityTooLarge, "upload_too_large",
			"fichiers trop volumineux (max 500 Mo)")
		return
	}

	files, err := parseUploadedFiles(r)
	if errors.Is(err, errUploadContentMismatch) {
		slog.WarnContext(r.Context(), "upload: contenu non concordant avec l'extension, envoi refusé",
			"player", gamertag, "err", err)
		writeError(r.Context(), w, http.StatusUnsupportedMediaType, "upload_content_mismatch",
			"file content does not match its extension")
		return
	}
	if err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "upload_parse_error", err.Error())
		return
	}
	if len(files) == 0 {
		writeError(r.Context(), w, http.StatusBadRequest, "no_valid_files",
			"aucun fichier valide (extensions acceptées : mp4 mov avi mkv webm png jpg jpeg bmp gif)")
		return
	}

	capturesDir := h.resolveCapturesDir(titleSlug, gamertag)
	capturesBase := ""
	var deleteSourceVal *bool
	if h.settingsStore != nil {
		if cfg, err := h.settingsStore.Load(); err == nil {
			capturesBase = cfg.MediaCapturesBaseDir
			deleteSourceVal = cfg.MediaDeleteSourceAfterTranscode
		}
	}
	// Politique de rétention du source résolue LIVE (env > store > isProd).
	deleteSource := config.ResolveMediaDeleteSource(
		os.Getenv(config.EnvMediaDeleteSource), deleteSourceVal, h.isProduction)
	slog.InfoContext(r.Context(), "upload: requête reçue",
		"player", gamertag, "files", len(files), "captures_dir", capturesDir)

	// Réglage audio du joueur (voix/jeu/autres) chargé AVANT le transcodage : en mode
	// manuel, les rôles court-circuitent l'analyse NNLS ; en auto, nil (comportement
	// historique). Un sidecar illisible est loggé et traité comme auto (jamais bloquant).
	manualAudioRoles := mediaaudio.ManualRolesForPlayer(
		r.Context(), titlePkg.NewPathResolver(h.repoRoot), titleSlug, gamertag)

	req := domain.UploadRequest{
		Files:               files,
		CapturesDir:         capturesDir,
		CapturesBase:        capturesBase,
		Gamertag:            gamertag,
		DBPath:              dbPath,
		SharedSocialDBPath:  sharedSocialDBPath,
		SharedMatchesDBPath: sharedMatchesDBPath,
		Tolerance:           2,
		TitleSlug:           titleSlug,
		DeleteSource:        deleteSource,
		ManualAudioRoles:    manualAudioRoles,
	}

	uploadStart := time.Now().UTC()
	result, err := svc.UploadMedia(r.Context(), req)
	if err != nil {
		slog.ErrorContext(r.Context(), "upload: erreur service", "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "upload_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
	BumpMediaFeedVersion()
	h.emitMediaAdded(r.Context(), slug, gamertag, result.NewIndexed,
		uploadStart, sharedSocialDBPath, sharedMatchesDBPath)
}

// acquireUploadSlot prend une place d'envoi, en attendant au plus uploadSlotWait. Sans
// place, répond 503 upload_busy (Retry-After) et rend ok=false.
func (h *MediaHandler) acquireUploadSlot(ctx context.Context, w http.ResponseWriter) (release func(), ok bool) {
	pool, wait := h.uploadSlots, h.uploadSlotWait
	if pool == nil {
		pool = globalUploadSlots
	}
	if wait <= 0 {
		wait = defaultUploadSlotWait
	}
	waitCtx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	release, err := pool.Acquire(waitCtx)
	if err != nil {
		slog.WarnContext(ctx, "upload: aucune place d'envoi libre, requête refusée",
			"wait", wait.String(), "in_flight", pool.InUse(), "err", err)
		w.Header().Set(headerRetryAfter, uploadBusyRetryAfterSec)
		writeError(ctx, w, http.StatusServiceUnavailable, "upload_busy",
			"server busy with other uploads, retry shortly")
		return nil, false
	}
	return release, true
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
		if err := checkUploadedContent(fh, ext); err != nil {
			return nil, err
		}
		// Le contenu n'est PAS lu ici : le service le copie en flux vers le disque.
		uf := domain.UploadedFile{
			OriginalName: fh.Filename,
			Open:         func() (io.ReadCloser, error) { return fh.Open() },
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

// errUploadContentMismatch : le contenu d'un fichier envoyé ne concorde pas avec son
// extension (PostUploadMedia répond 415 upload_content_mismatch, rien n'est écrit).
var errUploadContentMismatch = errors.New("contenu non concordant avec l'extension")

// checkUploadedContent lit la tête du fichier (au plus media.SniffLen octets) et vérifie
// qu'elle concorde avec la famille de son extension (media.ContentMatchesExtension).
func checkUploadedContent(fh *multipart.FileHeader, ext string) error {
	f, err := fh.Open()
	if err != nil {
		return fmt.Errorf("ouverture de %q : %w", fh.Filename, err)
	}
	defer f.Close()
	head := make([]byte, mediapkg.SniffLen)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return fmt.Errorf("lecture de %q : %w", fh.Filename, err)
	}
	if !mediapkg.ContentMatchesExtension(ext, head[:n]) {
		return fmt.Errorf("%w : %q", errUploadContentMismatch, fh.Filename)
	}
	return nil
}
