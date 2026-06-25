// Package main — snapshot_reader_wiring.go : câblage du reader snapshot-préféré global
// (Phase 4 du PLAN_DURABILITE_SNAPSHOT_IMMUABLE).
//
// Enveloppe le SharedReader live de chaque titre dans un sync.SnapshotPreferredSharedReader
// (lecture découplée du B-swap, fallback live). Mémoïsé PAR TITRE : un seul reader
// (donc un seul cache de queriers :memory: versionnés) partagé entre toutes les requêtes
// d'un titre. Posé sur cfg.SnapshotReaderWrapper, appliqué par config.sharedReaderForTitle.
package main

import (
	"sync"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
)

// snapshotReaderCache mémoïse un SnapshotPreferredSharedReader par titre.
type snapshotReaderCache struct {
	repoRoot string
	mu       sync.Mutex
	byTitle  map[string]*syncpkg.SnapshotPreferredSharedReader
}

func newSnapshotReaderCache(repoRoot string) *snapshotReaderCache {
	return &snapshotReaderCache{repoRoot: repoRoot, byTitle: map[string]*syncpkg.SnapshotPreferredSharedReader{}}
}

// wrap retourne le reader snapshot-préféré du titre (créé à la 1re demande), enveloppant
// `live`. Signature alignée sur config.AppConfig.SnapshotReaderWrapper.
func (c *snapshotReaderCache) wrap(titleSlug string, live duckdb.SharedReader) duckdb.SharedReader {
	c.mu.Lock()
	defer c.mu.Unlock()
	if r, ok := c.byTitle[titleSlug]; ok {
		return r
	}
	r := syncpkg.NewSnapshotPreferredSharedReader(titlePkg.NewPathResolver(c.repoRoot), titleSlug, live)
	c.byTitle[titleSlug] = r
	return r
}

// closeAll ferme les queriers :memory: cachés de tous les titres (shutdown).
func (c *snapshotReaderCache) closeAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for slug, r := range c.byTitle {
		r.Close()
		delete(c.byTitle, slug)
	}
}
