// Package sync — local_film_cache.go : lecture du cache disque film hérité du
// projet Python (`<legacy>/data/cache/film_manifests/*.json` +
// `<legacy>/data/cache/film_chunks/<8char>/chunk_<index>.bin`).
//
// Le cache Python conserve :
//   - 1 manifest JSON par match (`<8char>.json` = 8 premiers chars du UUID)
//   - les chunks REPLICATION_DATA (chunk_type=2) qui contiennent des kills
//
// Le HighlightEvents chunk (chunk_type=3) n'est pas systématiquement caché ;
// quand il est absent, on tombe sur le blob URL CDN (qui survit en général
// à l'expiration de l'endpoint manifest API).
//
// Activation : env `LEVELUP_LEGACY_FILM_CACHE_DIR` (ex.
// `C:\...\LevelUp\data\cache`). Si vide ou inexistant, le cache est désactivé
// et le client retombe sur les appels API standards.
package haloclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LocalFilmCache lit les manifestes et chunks du cache disque hérité du
// projet Python. Read-only : on ne réécrit jamais dans ce cache (le projet
// Go a son propre flux de cache si besoin futur).
type LocalFilmCache struct {
	rootDir string // ex: C:\...\LevelUp\data\cache
}

// NewLocalFilmCache crée un cache si le répertoire racine existe et contient
// les sous-dossiers attendus. Retourne nil si le répertoire est introuvable
// (cache désactivé).
func NewLocalFilmCache(rootDir string) *LocalFilmCache {
	rootDir = strings.TrimSpace(rootDir)
	if rootDir == "" {
		return nil
	}
	manifests := filepath.Join(rootDir, "film_manifests")
	if info, err := os.Stat(manifests); err != nil || !info.IsDir() {
		return nil
	}
	return &LocalFilmCache{rootDir: rootDir}
}

// CachedManifest est un manifest film tel que stocké par Python.
// Forme JSON : { "blob_prefix": "...", "chunks": [{"index", "chunk_type",
// "start_ms", "duration_ms", "file_relative_path"}] }
type CachedManifest struct {
	BlobPrefix string        `json:"blob_prefix"`
	Chunks     []CachedChunk `json:"chunks"`
}

// CachedChunk est l'entrée d'un chunk dans le manifest caché.
type CachedChunk struct {
	Index            int    `json:"index"`
	ChunkType        int    `json:"chunk_type"`
	StartMS          int    `json:"start_ms"`
	DurationMS       int    `json:"duration_ms"`
	FileRelativePath string `json:"file_relative_path"`
}

// shortID retourne les 8 premiers chars hexa du match_id (avant le premier
// tiret), forme attendue par le cache Python.
func shortID(matchID string) string {
	if i := strings.IndexByte(matchID, '-'); i > 0 {
		return matchID[:i]
	}
	if len(matchID) > 8 {
		return matchID[:8]
	}
	return matchID
}

// LoadManifest renvoie le manifest caché pour matchID, ou (nil, nil) si
// absent. Erreurs de parsing JSON ou IO retournées explicitement.
func (c *LocalFilmCache) LoadManifest(matchID string) (*CachedManifest, error) {
	if c == nil {
		return nil, nil
	}
	path := filepath.Join(c.rootDir, "film_manifests", shortID(matchID)+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("LoadManifest(%s): %w", matchID, err)
	}
	var m CachedManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("LoadManifest decode(%s): %w", matchID, err)
	}
	return &m, nil
}

// LoadChunk renvoie les bytes d'un chunk caché à l'index donné, ou (nil, nil)
// si absent. L'index correspond au champ Index du manifest (pas le ChunkType).
func (c *LocalFilmCache) LoadChunk(matchID string, index int) ([]byte, error) {
	if c == nil {
		return nil, nil
	}
	path := filepath.Join(c.rootDir, "film_chunks", shortID(matchID), fmt.Sprintf("chunk_%02d.bin", index))
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("LoadChunk(%s/%d): %w", matchID, index, err)
	}
	return data, nil
}
