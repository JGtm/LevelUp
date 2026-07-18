package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	titlePkg "levelup/go-api/internal/domain/title"
	mediapkg "levelup/go-api/internal/media"
)

func (h *MediaHandler) ServeMediaFile(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	rpath := chi.URLParam(r, "*")

	// Nettoyer le chemin URL (gère les "..")
	cleanURL := path.Clean("/" + rpath)
	if strings.Contains(cleanURL, "..") {
		httpError(r.Context(), w, "invalid path", http.StatusBadRequest)
		return
	}
	cleanURL = strings.TrimPrefix(cleanURL, "/")
	if cleanURL == "" || cleanURL == "." {
		httpError(r.Context(), w, "not found", http.StatusNotFound)
		return
	}

	capturesBase := ""
	if h.settingsStore != nil {
		if cfg, err := h.settingsStore.Load(); err == nil {
			capturesBase = cfg.MediaCapturesBaseDir
		}
	}

	// Bases candidates pour résoudre rpath. La première (capturesBase brut)
	// gère le format canonique {owner_slug}/{rel}. Les suivantes sont des
	// fallbacks legacy.
	var bases []string
	if capturesBase != "" {
		bases = append(bases, filepath.Clean(capturesBase))      // canonique : {base}/{owner}/{rel}
		bases = append(bases, filepath.Join(capturesBase, slug)) // legacy : rpath sans préfixe owner
	}
	if h.repoRoot != "" {
		pr := titlePkg.NewPathResolver(h.repoRoot)
		// Base média canonique {root}/data/media (rpath = {owner}/{rel}). Rattrape un
		// media_captures_base_dir absent/invalide — ex: chemin Windows déployé sur le
		// VPS Linux, qui rendait TOUS les médias en 404 (incident prod 2026-06-13).
		bases = append(bases, pr.MediaDataDir())
		internalCapturesDir := pr.PlayerCapturesDir(titlePkg.DefaultSlug, slug)
		bases = append(bases, filepath.Dir(internalCapturesDir)) // legacy interne data/
	}
	if len(bases) == 0 {
		httpError(r.Context(), w, "file serving not configured", http.StatusServiceUnavailable)
		return
	}

	resolved := resolveMediaFilePath(bases, cleanURL)
	if resolved == "" {
		httpError(r.Context(), w, "not found", http.StatusNotFound)
		return
	}

	ext := strings.ToLower(filepath.Ext(resolved))

	if mediapkg.RequiresRemux(ext) {
		serveRemuxedWebM(w, r, resolved)
		return
	}

	setMediaContentType(w, ext)
	http.ServeFile(w, r, resolved)
}

// resolveMediaFilePath cherche le fichier sur disque parmi les bases candidates.
// Pour chaque base :
//  1. tente le chemin exact (cleanURL stocké en DB) ;
//  2. si absent, tente une résolution par stem : pour chaque extension vidéo
//     reconnue, teste {dir}/{stem}{ext}. Couvre le cas où le fichier a été
//     converti localement (ex: .mp4 → .mkv) sans que la DB ait été resync.
//
// Retourne le chemin absolu trouvé (passe l'anti-traversal), ou "" si rien
// ne match.
func resolveMediaFilePath(bases []string, cleanURL string) string {
	relFS := filepath.FromSlash(cleanURL)
	stem, _ := mediapkg.StemAndExt(relFS)
	relDir := filepath.Dir(relFS)

	for _, base := range bases {
		cleanBase := filepath.Clean(base)

		// (1) Tentative exacte sur le chemin stocké.
		exact := filepath.Clean(filepath.Join(base, relFS))
		if isUnderBase(exact, cleanBase) {
			if _, err := os.Stat(exact); err == nil {
				return exact
			}
		}

		// (2) Fallback resolution multi-extension par stem. Le `cleanURL`
		// stocké en DB peut pointer vers une extension stale (ex: .mp4 stocké
		// alors que le fichier réel est devenu .mkv après conversion locale).
		if stem == "" {
			continue
		}
		for _, candExt := range mediapkg.VideoExtensions {
			candRel := filepath.Join(relDir, stem+candExt)
			cand := filepath.Clean(filepath.Join(base, candRel))
			if !isUnderBase(cand, cleanBase) {
				continue
			}
			if _, err := os.Stat(cand); err == nil {
				return cand
			}
		}
	}
	return ""
}

// isUnderBase implémente l'anti-traversal : refuse les paths qui sortent de
// base (par /.. ou via symlink résolu).
func isUnderBase(absPath, cleanBase string) bool {
	return absPath == cleanBase || strings.HasPrefix(absPath, cleanBase+string(filepath.Separator))
}

// setMediaContentType positionne Content-Type d'après l'extension d'un fichier
// servi directement (pas via remux). Ne modifie rien si le header est déjà set.
func setMediaContentType(w http.ResponseWriter, ext string) {
	if w.Header().Get("Content-Type") != "" {
		return
	}
	switch ext {
	case ".mp4":
		w.Header().Set("Content-Type", "video/mp4")
	case ".webm":
		w.Header().Set("Content-Type", "video/webm")
	case ".mov":
		w.Header().Set("Content-Type", "video/quicktime")
	case ".webp":
		w.Header().Set("Content-Type", "image/webp")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".m3u8":
		// Playlist HLS (master + sous-playlists). hls.js et Safari natif s'appuient
		// sur ce type pour démarrer la lecture adaptative.
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	case ".m4s":
		w.Header().Set("Content-Type", "video/mp4") // segment fMP4 (CMAF)
	case ".ts":
		w.Header().Set("Content-Type", "video/mp2t") // segment MPEG-TS (legacy HLS)
	}
}

// serveRemuxedWebM streame le fichier source remuxé en WebM via ffmpeg. Le
// pré-flight (probe + validation codecs) est fait AVANT tout header : un source
// aux codecs incompatibles WebM répond 415, un probe cassé répond 502 — plus de
// « 200 corps vide » quand l'échec est connu d'avance. Une fois le flux démarré
// (headers 200 envoyés), un échec ffmpeg ne peut plus qu'être loggé.
//
// 502 (et non 500) pour l'échec de probe : le handler agit en passerelle devant
// le sous-processus ffprobe/ffmpeg (dépendance de transcodage) ; un probe qui
// ne rend aucun résultat exploitable est une défaillance de cette dépendance, à
// distinguer d'un bug de handler (500) dans les logs/alertes. Le Range request
// n'est pas supporté (seek limité aux clips web-natifs).
func serveRemuxedWebM(w http.ResponseWriter, r *http.Request, absPath string) {
	plan, err := mediapkg.PlanRemuxWebM(r.Context(), absPath)
	if err != nil {
		if errors.Is(err, mediapkg.ErrRemuxIncompatibleCodec) {
			slog.ErrorContext(r.Context(), "media remux: codecs incompatibles WebM",
				"path", absPath, "err", err)
			httpError(r.Context(), w, "unsupported media codec", http.StatusUnsupportedMediaType)
			return
		}
		slog.ErrorContext(r.Context(), "media remux: pré-flight ffprobe échoué",
			"path", absPath, "err", err)
		httpError(r.Context(), w, "remux preflight failed", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "video/webm")
	if err := mediapkg.StreamRemuxWebMPlan(r.Context(), absPath, plan, w); err != nil {
		slog.ErrorContext(r.Context(), "media remux: échec en cours de flux (status déjà envoyé)",
			"path", absPath, "err", err)
	}
}
