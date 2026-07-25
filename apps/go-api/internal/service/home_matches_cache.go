// Package service — home_matches_cache.go : cache TTL in-process pour les données
// lourdes de la page Home (matches + sessions).
//
// Stratégie : un singleton process-level (HomeMatchesCache) partagé entre requêtes,
// avec expiration TTL et invalidation explicite. Pas de singleflight : le thundering
// herd est peu probable sur cette surface.
//
// Clé d'entrée = (xuid, titleSlug) via cacheKey (séparateur NUL, cf. career_live_cache).
// Le titre fait partie de la clé (défense en profondeur V72-29) : un même xuid peut
// consulter plusieurs titres et leurs matches/sessions ne doivent jamais se croiser.
package service

import (
	"log/slog"
	"sync"
	"time"

	"levelup/go-api/internal/legacymatch"
)

// homeMatchesTTL est la durée de vie des données en cache.
// 45s : assez court pour ne jamais afficher de données vieilles après un sync,
// assez long pour absorber les rechargements rapides de page.
const homeMatchesTTL = 45 * time.Second

// homeMatchesCacheEntry est une entrée du cache pour un couple (joueur, titre).
type homeMatchesCacheEntry struct {
	matches   []legacymatch.HomeMatchRow
	sessions  []legacymatch.HomeSessionRow
	expiresAt time.Time
}

func (e *homeMatchesCacheEntry) isValid() bool {
	return e != nil && time.Now().Before(e.expiresAt)
}

// HomeMatchesCache est un cache TTL process-level, partagé entre requêtes.
// Créé une fois dans ServiceRegistry et injecté dans chaque HomeService.
type HomeMatchesCache struct {
	mu      sync.Mutex
	entries map[string]*homeMatchesCacheEntry
}

// NewHomeMatchesCache crée un cache vide.
func NewHomeMatchesCache() *HomeMatchesCache {
	return &HomeMatchesCache{entries: make(map[string]*homeMatchesCacheEntry)}
}

// Get retourne les données cachées pour un couple (xuid, titleSlug), et un booléen hit/miss.
func (c *HomeMatchesCache) Get(xuid, titleSlug string) ([]legacymatch.HomeMatchRow, []legacymatch.HomeSessionRow, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[cacheKey(xuid, titleSlug)]
	if !ok || !e.isValid() {
		return nil, nil, false
	}
	return e.matches, e.sessions, true
}

// Set stocke les données pour un couple (xuid, titleSlug) avec le TTL standard.
func (c *HomeMatchesCache) Set(xuid, titleSlug string, matches []legacymatch.HomeMatchRow, sessions []legacymatch.HomeSessionRow) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[cacheKey(xuid, titleSlug)] = &homeMatchesCacheEntry{
		matches:   matches,
		sessions:  sessions,
		expiresAt: time.Now().Add(homeMatchesTTL),
	}
	slog.Debug("home_cache: entrée mise en cache", "xuid", xuid, "titleSlug", titleSlug, "matches", len(matches), "sessions", len(sessions))
}

// Invalidate supprime l'entrée cache d'un couple (xuid, titleSlug) (appelé après un sync).
func (c *HomeMatchesCache) Invalidate(xuid, titleSlug string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, cacheKey(xuid, titleSlug))
	slog.Debug("home_cache: entrée invalidée", "xuid", xuid, "titleSlug", titleSlug)
}
