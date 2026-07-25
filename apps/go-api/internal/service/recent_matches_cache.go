// Package service — recent_matches_cache.go : décorateur de cache TTL +
// singleflight autour d'un port.RecentMatchesProvider (20 derniers matchs live).
//
// Pourquoi : le fetch live des derniers matchs d'une cible (liste + 1 appel stats
// PAR match) est coûteux (N+1 round-trips Halo). On le mémorise en mémoire,
// process-level, comme remote_stats_cache.go / CareerLiveCache — AUCUNE écriture
// DB. L'historique des derniers matchs bouge moins vite qu'un service record
// agrégé → TTL plus long (20 min) que DefaultRemoteStatsTTL.
//
// Clé = (titleSlug | xuid | limit). Le titre (lu du contexte) fait partie de la clé
// (défense en profondeur V72-29) : le même xuid peut être consulté sous plusieurs
// titres et leurs derniers matchs ne doivent jamais se croiser. Les matchs d'une cible
// sont identiques quel que soit le requérant → cache partagé entre utilisateurs sans
// fuite (données publiques).
package service

import (
	"context"
	"expvar"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// DefaultRecentMatchesTTL : durée de validité d'une entrée de derniers matchs en
// cache. Plus long que DefaultRemoteStatsTTL (5 min) car l'historique récent ne
// change qu'à la complétion d'un nouveau match.
const DefaultRecentMatchesTTL = 20 * time.Minute

const recentMatchesLogModule = "recent_matches_cache"

var (
	recentMatchesCacheHit   = expvar.NewInt("recent_matches.cache_hit")
	recentMatchesCacheMiss  = expvar.NewInt("recent_matches.cache_miss")
	recentMatchesFetchError = expvar.NewInt("recent_matches.fetch_error")
)

type recentMatchesEntry struct {
	rows      []domain.ExplorerTargetRecentMatch
	fetchedAt time.Time
}

// CachedRecentMatchesProvider décore un port.RecentMatchesProvider d'un cache TTL
// process-level + singleflight. Thread-safe. Instancié une fois (singleton) et
// partagé entre toutes les requêtes (Explorer, Compare).
type CachedRecentMatchesProvider struct {
	inner port.RecentMatchesProvider
	ttl   time.Duration
	now   func() time.Time

	mu      sync.RWMutex
	entries map[string]recentMatchesEntry

	sf singleflight.Group
}

// NewCachedRecentMatchesProvider enveloppe inner. ttl<=0 → DefaultRecentMatchesTTL.
// now nil → time.Now (injectable pour les tests TTL déterministes).
func NewCachedRecentMatchesProvider(inner port.RecentMatchesProvider, ttl time.Duration, now func() time.Time) *CachedRecentMatchesProvider {
	if ttl <= 0 {
		ttl = DefaultRecentMatchesTTL
	}
	if now == nil {
		now = time.Now
	}
	return &CachedRecentMatchesProvider{
		inner:   inner,
		ttl:     ttl,
		now:     now,
		entries: make(map[string]recentMatchesEntry),
	}
}

// FetchRecentMatches sert depuis le cache si frais, sinon délègue à inner
// (dédupliqué par singleflight) et mémorise les résultats NON vides.
func (c *CachedRecentMatchesProvider) FetchRecentMatches(
	ctx context.Context, xuid string, limit int,
) ([]domain.ExplorerTargetRecentMatch, error) {
	if xuid == "" || limit <= 0 {
		return nil, nil
	}
	// Titre inclus dans la clé (V72-29) : jamais de fuite cross-titre sous un même xuid.
	key := ctxkeys.TitleSlug(ctx) + "|" + xuid + "|" + strconv.Itoa(limit)

	if rows, ok := c.get(key); ok {
		recentMatchesCacheHit.Add(1)
		slog.DebugContext(ctx, recentMatchesLogModule+": cache hit", "xuid", xuid, "limit", limit)
		return rows, nil
	}

	v, err, _ := c.sf.Do(key, func() (any, error) {
		if rows, ok := c.get(key); ok {
			recentMatchesCacheHit.Add(1)
			return rows, nil
		}
		recentMatchesCacheMiss.Add(1)
		slog.DebugContext(ctx, recentMatchesLogModule+": cache miss → fetch live", "xuid", xuid, "limit", limit)
		rows, fErr := c.inner.FetchRecentMatches(ctx, xuid, limit)
		if fErr != nil {
			recentMatchesFetchError.Add(1)
			return nil, fErr
		}
		// On ne met en cache que les résultats non vides : un (nil) (pas d'auth /
		// aucun match) ne doit pas figer 20 min — retenter au prochain appel.
		if len(rows) > 0 {
			c.put(key, rows)
		}
		return rows, nil
	})
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return v.([]domain.ExplorerTargetRecentMatch), nil
}

func (c *CachedRecentMatchesProvider) get(key string) ([]domain.ExplorerTargetRecentMatch, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[key]
	if !ok || c.now().Sub(e.fetchedAt) > c.ttl {
		return nil, false
	}
	return e.rows, true
}

func (c *CachedRecentMatchesProvider) put(key string, rows []domain.ExplorerTargetRecentMatch) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = recentMatchesEntry{rows: rows, fetchedAt: c.now()}
}
