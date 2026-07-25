package handlers

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/settings"
)

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

// MediaFeedVersionHandler expose GET /media/feed-version via Huma (Phase 3b).
//
// Sans état (la version vit dans le compteur atomique de package mediaFeedVersion).
// Mount crée humacore.NewAPI(r) et reproduit EXACTEMENT le corps JSON et le statut
// de l'ancienne GetMediaFeedVersion : 200 + {"version": <int64>} sérialisé via
// JSONFormat (byte-identique à writeJSON).
type MediaFeedVersionHandler struct{}

// NewMediaFeedVersionHandler crée un MediaFeedVersionHandler.
func NewMediaFeedVersionHandler() *MediaFeedVersionHandler {
	return &MediaFeedVersionHandler{}
}

// Mount enregistre GET /media/feed-version via Huma sur le routeur chi fourni.
func (h *MediaFeedVersionHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/media/feed-version", h.handleFeedVersion)
}

// mediaFeedVersionOutput : corps {"version": <int64>} — même forme que
// map[string]int64{"version": v} de l'ancienne GetMediaFeedVersion.
type mediaFeedVersionOutput struct {
	Body map[string]int64
}

// handleFeedVersion lit le compteur atomique et renvoie {"version": <int64>}.
// Pas de path/query/body : input *struct{}.
func (h *MediaFeedVersionHandler) handleFeedVersion(_ context.Context, _ *struct{}) (*mediaFeedVersionOutput, error) {
	v := atomic.LoadInt64(&mediaFeedVersion)
	return &mediaFeedVersionOutput{Body: map[string]int64{"version": v}}, nil
}

// filePathToURL transforme un chemin stocké en DB en URL servable via l'API.
// Wrapper de méthode autour de mediaStoredPathToURL (cf. ci-dessous) liant repoRoot.
func (h *MediaHandler) filePathToURL(slug, storedPath, capturesBase string) string {
	return mediaStoredPathToURL(slug, storedPath, capturesBase, h.repoRoot)
}

// mediaStoredPathToURL transforme un chemin média stocké en URL servable via
// l'API. Fonction de package réutilisée par MediaHandler (galerie) et
// MatchViewHandler (onglet médias) — source unique de la transformation.
//
// Path stocké post-migration : déjà relatif au format {owner_slug}/{rel}.
// On le réécrit directement en URL sans aucune résolution disk.
//
// Path stocké legacy : absolu (ex: "C:\old\layout\JGtm\thumbs\xxx.gif"). On
// tente 3 préfixes (capturesBase+slug, capturesBase brut, repoRoot interne)
// pour relativiser et fallback à absPath si rien ne matche. Cette branche
// disparaîtra une fois le script de migration DB exécuté.
func mediaStoredPathToURL(slug, storedPath, capturesBase, repoRoot string) string {
	if storedPath == "" {
		return storedPath
	}

	// Path déjà relatif (post-migration) : on l'utilise tel quel.
	if !filepath.IsAbs(storedPath) {
		return "/api/v1/players/" + slug + "/media/files/" + filepath.ToSlash(storedPath)
	}

	// --- Mode legacy : path absolu stocké, on tente de relativiser. ---
	clean := filepath.Clean(storedPath)

	if capturesBase != "" {
		playerDir := filepath.Join(capturesBase, slug)
		if rel, ok := relIfWithin(playerDir, clean); ok {
			return "/api/v1/players/" + slug + "/media/files/" + filepath.ToSlash(filepath.Join(slug, rel))
		}
		// Single-player : capturesBase pointe directement sur le dossier
		// captures sans sous-folder slug (cas typique production).
		if rel, ok := relIfWithin(capturesBase, clean); ok {
			return "/api/v1/players/" + slug + "/media/files/" + filepath.ToSlash(rel)
		}
	}

	if repoRoot != "" {
		pr := titlePkg.NewPathResolver(repoRoot)
		internalCapturesDir := pr.PlayerCapturesDir(titlePkg.DefaultSlug, slug)
		internalPlayerDir := filepath.Dir(internalCapturesDir)
		if rel, ok := relIfWithin(internalPlayerDir, clean); ok {
			return "/api/v1/players/" + slug + "/media/files/" + filepath.ToSlash(rel)
		}
	}

	// Filet portable inter-OS (Windows local ↔ Debian VPS) : un path absolu
	// legacy hors des bases connues conserve toujours le segment reconnaissable
	// "/{slug}/…" (ex. C:\…\data\media\JGtm\clip.mp4 ou /srv/…/data/media/JGtm/
	// clip.mp4). On en extrait {slug}/{rel} — même heuristique que
	// cmd/migrate-media-paths.convertPath — pour servir le média au format
	// canonique au lieu d'un warning + média cassé, indépendamment du préfixe
	// absolu. Couvre aussi les lignes non encore converties côté DB (ex. VPS
	// avant passage de la migration).
	if rel, ok := relFromSlugMarker(clean, slug); ok {
		return "/api/v1/players/" + slug + "/media/files/" + rel
	}

	slog.Warn("mediaStoredPathToURL: aucun mapping trouvé (path legacy absolu hors layout)",
		"slug", slug, "abs_path", storedPath, "captures_base", capturesBase)
	return storedPath
}

// relFromSlugMarker extrait le sous-chemin relatif {slug}/{rest} d'un path absolu
// legacy en cherchant le marqueur "/{slug}/" (ou un préfixe "{slug}/" déjà en
// tête). Portable inter-OS : normalise via filepath.ToSlash puis ignore
// entièrement le préfixe absolu (lettre de lecteur Windows OU racine POSIX).
// Miroir de l'heuristique cmd/migrate-media-paths.convertPath, pour que le read
// serve les paths absolus résiduels au format canonique {owner_slug}/{rel}.
func relFromSlugMarker(cleanPath, slug string) (string, bool) {
	if slug == "" {
		return "", false
	}
	normalized := filepath.ToSlash(cleanPath)
	if idx := strings.Index(normalized, "/"+slug+"/"); idx >= 0 {
		return normalized[idx+1:], true // depuis "{slug}/…"
	}
	if strings.HasPrefix(normalized, slug+"/") {
		return normalized, true
	}
	return "", false
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

// mediaServableURLToStoredPath est l'inverse EXACT de la branche relative
// (post-migration) de mediaStoredPathToURL : il retire le préfixe d'URL servable
// `/api/v1/players/{viewerSlug}/media/files/` pour recomposer le chemin RELATIF
// (forward-slash) stocké dans la colonne media_files.file_path.
//
// Pour une vidéo, la galerie sert `file_path` sous forme de playlist HLS
// (`.../hls/<stem>/master.m3u8`) qui EST le file_path stocké — donc le simple strip
// du préfixe recompose la clé de lookup. Envoyée telle quelle à /media/match-candidates
// ou /media/associate, l'URL ne matchait NI mf.file_path (préfixe URL en trop) NI
// mf.file_name (basename = "master.m3u8" pour une vidéo HLS) → ErrNoRows → 500
// « loading error ». On garde des forward-slashes (format DB), sans filepath.FromSlash.
//
// Idempotent : une entrée qui n'est pas une URL servable (chemin déjà stocké,
// appelant legacy) est renvoyée inchangée.
func mediaServableURLToStoredPath(viewerSlug, input string) string {
	prefix := "/api/v1/players/" + viewerSlug + "/media/files/"
	if rel := strings.TrimPrefix(input, prefix); rel != input {
		return rel
	}
	return input
}

// urlToFilePath fait l'inverse de filePathToURL : convertit une URL servable
// `/api/v1/players/{slug}/media/files/{relPath}` en chemin absolu de stockage.
// Si l'entrée n'est pas une URL transformée (déjà un chemin absolu, par exemple),
// retourne tel quel.
func (h *MediaHandler) urlToFilePath(slug, input string) string {
	stored := mediaServableURLToStoredPath(slug, input)
	if stored == input {
		return input // pas une URL servable → passthrough
	}
	relPath := filepath.FromSlash(stored)

	// Tentative 1 : capturesBase configuré.
	capturesBase := ""
	if h.settingsStore != nil {
		if cfg, err := h.settingsStore.Load(); err == nil && cfg.MediaCapturesBaseDir != "" {
			capturesBase = filepath.Clean(cfg.MediaCapturesBaseDir)
		}
	}
	if capturesBase != "" {
		// relPath contient déjà le slug du propriétaire en préfixe (ex: "JGtm/clip.mp4").
		// On ne rajoute PAS slug (viewer) pour éviter le double-slug "capturesBase/viewer/owner/clip.mp4".
		candidate := filepath.Join(capturesBase, relPath)
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

	// Fallback : retourner relPath (pas l'URL). Pour les paths post-migration
	// stockés en relatif (ex: "JGtm/clip.mp4"), relPath == stored path → UPDATE match.
	return relPath
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
	capturesBase := mediaCapturesBase(h.settingsStore)
	for i := range items {
		items[i].FilePath = h.filePathToURL(slug, items[i].FilePath, capturesBase)
		if items[i].ThumbnailPath != nil {
			u := h.filePathToURL(slug, *items[i].ThumbnailPath, capturesBase)
			items[i].ThumbnailPath = &u
		}
	}
}

// mediaCapturesBase lit media_captures_base_dir depuis le settings store.
// Retourne "" si le store est nil ou la valeur absente — la transformation
// d'URL retombe alors sur le mode relatif (post-migration), suffisant en prod.
// Partagée par MediaHandler (galerie) et MatchViewHandler (onglet médias).
func mediaCapturesBase(store *settings.Store) string {
	if store == nil {
		return ""
	}
	if cfg, err := store.Load(); err == nil && cfg.MediaCapturesBaseDir != "" {
		return filepath.Clean(cfg.MediaCapturesBaseDir)
	}
	return ""
}

// ServeMediaFile sert un fichier depuis le répertoire captures.
// GET /api/v1/players/{player_slug}/media/files/*
//
// Convention post-migration : rpath = {owner_slug}/{rel_in_owner_dir}. La
// résolution est déterministe : filepath.Join(capturesBase, rpath). Le
// {player_slug} de l'URL ne sert qu'à scoper l'autorisation (le owner est
// implicite via le préfixe rpath).
//
// Compat legacy : fallbacks vers capturesBase+slug (rpath sans préfixe owner)
// et repoRoot interne pour les URLs encore présentes en cache navigateur ou
// dans des paths absolus non migrés.
