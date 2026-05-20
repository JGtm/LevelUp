// Package handlers — handler GET /api/v1/help/release-notes.
//
// P8.10 (revue 2026-04-29 gap #4) : la logique git + parsing markdown a été
// extraite dans `internal/service/release_notes_service.go`. Ce handler ne
// fait que :
//  1. Décoder la query (lang).
//  2. Cache mémoire + disque (24h TTL).
//  3. Appeler service.Build(lang) en cache miss.
//  4. Sérialiser en JSON.
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// helpDiskCache est la structure sérialisée sur disque.
type helpDiskCache struct {
	Content  string    `json:"content"`
	CachedAt time.Time `json:"cached_at"`
}

// ReleaseNotesBuilder est l'interface implémentée par
// service.ReleaseNotesService.Build (port-style pour testabilité).
type ReleaseNotesBuilder interface {
	Build(lang string) (string, error)
}

// HelpHandler sert les notes de version. Cache double couche : mémoire (TTL
// 24h) + disque (data/cache/help_release_notes_{lang}.json).
type HelpHandler struct {
	builder  ReleaseNotesBuilder
	cacheDir string
	mu       sync.RWMutex
	memory   map[string]helpCacheEntry
	ttl      time.Duration
}

type helpCacheEntry struct {
	content  string
	loadedAt time.Time
}

// NewHelpHandler crée le handler avec un builder injecté et un cacheDir
// (typiquement `<repoRoot>/data/cache`).
func NewHelpHandler(builder ReleaseNotesBuilder, cacheDir string) *HelpHandler {
	return &HelpHandler{
		builder:  builder,
		cacheDir: cacheDir,
		memory:   make(map[string]helpCacheEntry),
		ttl:      24 * time.Hour,
	}
}

// GetReleaseNotes retourne le markdown des notes de version.
// Query param : lang=fr|en (défaut : fr).
func (h *HelpHandler) GetReleaseNotes(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")
	if lang != "en" {
		lang = "fr"
	}

	content, err := h.loadCached(lang)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "RELEASE_NOTES_ERROR", "Impossible de charger les notes de version")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"content": content})
}

func (h *HelpHandler) loadCached(lang string) (string, error) {
	// 1. Cache mémoire (lecture rapide).
	h.mu.RLock()
	if e, ok := h.memory[lang]; ok && time.Since(e.loadedAt) < h.ttl {
		h.mu.RUnlock()
		return e.content, nil
	}
	h.mu.RUnlock()

	h.mu.Lock()
	defer h.mu.Unlock()

	// Double-check après acquisition du write lock.
	if e, ok := h.memory[lang]; ok && time.Since(e.loadedAt) < h.ttl {
		return e.content, nil
	}

	// 2. Cache disque — évite de ré-invoquer git à chaque redémarrage.
	if content, ok := h.loadDiskCache(lang); ok {
		h.memory[lang] = helpCacheEntry{content: content, loadedAt: time.Now()}
		return content, nil
	}

	// 3. Reconstruction via le service + écriture cache.
	content, err := h.builder.Build(lang)
	if err != nil {
		return "", err
	}
	h.memory[lang] = helpCacheEntry{content: content, loadedAt: time.Now()}
	h.writeDiskCache(lang, content)
	return content, nil
}

func (h *HelpHandler) diskCachePath(lang string) string {
	return filepath.Join(h.cacheDir, fmt.Sprintf("help_release_notes_%s.json", lang))
}

func (h *HelpHandler) loadDiskCache(lang string) (string, bool) {
	path := h.diskCachePath(lang)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	var dc helpDiskCache
	if err := json.Unmarshal(data, &dc); err != nil {
		return "", false
	}
	if time.Since(dc.CachedAt) >= h.ttl {
		return "", false
	}
	return dc.Content, true
}

func (h *HelpHandler) writeDiskCache(lang, content string) {
	if err := os.MkdirAll(h.cacheDir, 0o755); err != nil {
		return
	}
	dc := helpDiskCache{Content: content, CachedAt: time.Now()}
	raw, err := json.Marshal(dc)
	if err != nil {
		return
	}
	path := h.diskCachePath(lang)
	tmp, err := os.CreateTemp(h.cacheDir, ".help_cache_tmp_")
	if err != nil {
		return
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return
	}
	_ = os.Rename(tmpName, path)
}
