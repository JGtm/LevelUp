// Package service — home_matches_cache.go : cache TTL in-process pour les données lourdes
// de la page Home (matches + sessions).
//
// Stratégie : un singleton process-level (HomeMatchesCache) partagé entre requêtes
// pour le même joueur, avec expiration TTL et invalidation explicite.
// Pas de singleflight : l'app est mono-joueur et le thundering herd est peu probable.
package service

import (
	"log/slog"
	"sync"
	"time"

	"levelup/go-api/internal/domain"
)

// homeMatchesTTL est la durée de vie des données en cache.
// 45s : assez court pour ne jamais afficher de données vieilles après un sync,
// assez long pour absorber les rechargements rapides de page.
const homeMatchesTTL = 45 * time.Second

// homeMatchesCacheEntry est une entrée du cache pour un joueur.
type homeMatchesCacheEntry struct {
	matches   []domain.HomeMatchRow
	sessions  []domain.HomeSessionRow
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

// Get retourne les données cachées pour un xuid, et un booléen hit/miss.
func (c *HomeMatchesCache) Get(xuid string) ([]domain.HomeMatchRow, []domain.HomeSessionRow, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[xuid]
	if !ok || !e.isValid() {
		return nil, nil, false
	}
	return e.matches, e.sessions, true
}

// Set stocke les données pour un xuid avec le TTL standard.
func (c *HomeMatchesCache) Set(xuid string, matches []domain.HomeMatchRow, sessions []domain.HomeSessionRow) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[xuid] = &homeMatchesCacheEntry{
		matches:   matches,
		sessions:  sessions,
		expiresAt: time.Now().Add(homeMatchesTTL),
	}
	slog.Debug("home_cache: entrée mise en cache", "xuid", xuid, "matches", len(matches), "sessions", len(sessions))
}

// Invalidate supprime l'entrée cache d'un joueur (appelé après un sync).
func (c *HomeMatchesCache) Invalidate(xuid string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, xuid)
	slog.Debug("home_cache: entrée invalidée", "xuid", xuid)
}
