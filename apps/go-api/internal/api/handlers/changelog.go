// Package handlers — handler GET /api/v1/changelog.
package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ChangelogHandler sert le contenu de docs/CHANGELOG.md avec un cache en mémoire.
type ChangelogHandler struct {
	filePath string
	mu       sync.RWMutex
	cached   string
	loadedAt time.Time
	cacheTTL time.Duration
}

// NewChangelogHandler crée un handler pour le changelog.
// rootDir est la racine du projet (contenant docs/).
func NewChangelogHandler(rootDir string) *ChangelogHandler {
	return &ChangelogHandler{
		filePath: filepath.Join(rootDir, "docs", "CHANGELOG.md"),
		cacheTTL: 5 * time.Minute,
	}
}

// GetChangelog retourne le contenu markdown du changelog.
func (h *ChangelogHandler) GetChangelog(w http.ResponseWriter, r *http.Request) {
	content, err := h.loadCached()
	if err != nil {
		writeError(w, http.StatusNotFound, "CHANGELOG_NOT_FOUND", "Changelog introuvable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"content": content})
}

func (h *ChangelogHandler) loadCached() (string, error) {
	h.mu.RLock()
	if h.cached != "" && time.Since(h.loadedAt) < h.cacheTTL {
		defer h.mu.RUnlock()
		return h.cached, nil
	}
	h.mu.RUnlock()

	h.mu.Lock()
	defer h.mu.Unlock()

	// Double-check après acquisition du write lock.
	if h.cached != "" && time.Since(h.loadedAt) < h.cacheTTL {
		return h.cached, nil
	}

	data, err := os.ReadFile(h.filePath)
	if err != nil {
		return "", err
	}
	h.cached = string(data)
	h.loadedAt = time.Now()
	return h.cached, nil
}
