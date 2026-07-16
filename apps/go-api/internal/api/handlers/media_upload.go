package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
)

func (h *MediaHandler) PostUploadMedia(w http.ResponseWriter, r *http.Request) {
	if h.demoMode {
		// Vitrine publique : l'upload reste visible mais inerte (cf. UploadButton
		// figé côté front). Refus serveur en garde-fou — aucune écriture disque.
		writeError(r.Context(), w, http.StatusForbidden, "demo_mode_unsupported",
			"L'import de médias n'est pas disponible en mode démo.")
		return
	}
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

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(r.Context(), w, http.StatusRequestEntityTooLarge, "upload_too_large",
			"fichiers trop volumineux (max 500 Mo)")
		return
	}

	files, err := parseUploadedFiles(r)
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
