// Package service â€” home_matches_cache.go : cache TTL in-process pour les donnÃ©es lourdes
// de la page Home (matches + sessions).
//
// StratÃ©gie : un singleton process-level (HomeMatchesCache) partagÃ© entre requÃªtes
// pour le mÃªme joueur, avec expiration TTL et invalidation explicite.
// Pas de singleflight : l'app est mono-joueur et le thundering herd est peu probable.
package service

import (
	"log/slog"
	"sync"
	"time"

	"levelup/go-api/internal/legacymatch"
)

// homeMatchesTTL est la durÃ©e de vie des donnÃ©es en cache.
// 45s : assez court pour ne jamais afficher de donnÃ©es vieilles aprÃ¨s un sync,
// assez long pour absorber les rechargements rapides de page.
const homeMatchesTTL = 45 * time.Second

// homeMatchesCacheEntry est une entrÃ©e du cache pour un joueur.
type homeMatchesCacheEntry struct {
	matches   []legacymatch.HomeMatchRow
	sessions  []legacymatch.HomeSessionRow
	expiresAt time.Time
}

func (e *homeMatchesCacheEntry) isValid() bool {
	return e != nil && time.Now().Before(e.expiresAt)
}

// HomeMatchesCache est un cache TTL process-level, partagÃ© entre requÃªtes.
// CrÃ©Ã© une fois dans ServiceRegistry et injectÃ© dans chaque HomeService.
type HomeMatchesCache struct {
	mu      sync.Mutex
	entries map[string]*homeMatchesCacheEntry
}

// NewHomeMatchesCache crÃ©e un cache vide.
func NewHomeMatchesCache() *HomeMatchesCache {
	return &HomeMatchesCache{entries: make(map[string]*homeMatchesCacheEntry)}
}

// Get retourne les donnÃ©es cachÃ©es pour un xuid, et un boolÃ©en hit/miss.
func (c *HomeMatchesCache) Get(xuid string) ([]legacymatch.HomeMatchRow, []legacymatch.HomeSessionRow, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[xuid]
	if !ok || !e.isValid() {
		return nil, nil, false
	}
	return e.matches, e.sessions, true
}

// Set stocke les donnÃ©es pour un xuid avec le TTL standard.
func (c *HomeMatchesCache) Set(xuid string, matches []legacymatch.HomeMatchRow, sessions []legacymatch.HomeSessionRow) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[xuid] = &homeMatchesCacheEntry{
		matches:   matches,
		sessions:  sessions,
		expiresAt: time.Now().Add(homeMatchesTTL),
	}
	slog.Debug("home_cache: entrÃ©e mise en cache", "xuid", xuid, "matches", len(matches), "sessions", len(sessions))
}

// Invalidate supprime l'entrÃ©e cache d'un joueur (appelÃ© aprÃ¨s un sync).
func (c *HomeMatchesCache) Invalidate(xuid string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, xuid)
	slog.Debug("home_cache: entrÃ©e invalidÃ©e", "xuid", xuid)
}
