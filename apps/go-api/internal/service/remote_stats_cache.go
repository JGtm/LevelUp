// Package service — remote_stats_cache.go : décorateur de cache TTL +
// singleflight autour d'un port.PlayerStatsProvider (service record agrégé
// Waypoint, /hi/players/{player}/Matchmade/servicerecord).
//
// Pourquoi : l'encart "Profil joueur cible" de l'Explorer (et Compare) fetche
// les stats carrière d'un joueur via Waypoint à CHAQUE chargement. C'est un
// round-trip Halo de plusieurs centaines de ms (voire secondes) refait à
// l'identique quand on rouvre la même cible. SpartanRecord (projet parent)
// doit sa vitesse au cache-first Firebase ; on applique ici le même principe,
// process-level, sur le modèle de CareerLiveCache.
//
// La clé est (titleSlug | gamertag-lowercased). Le service-record/career-stats
// d'un joueur ne bouge qu'entre deux matchs → TTL court de 5 min, suffisant
// pour rendre instantanée la réouverture d'une même cible dans une session.
package service

import (
	"context"
	"expvar"
	"log/slog"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// DefaultRemoteStatsTTL est la durée de validité d'une entrée de stats carrière
// remote en cache. Aligné sur DefaultCareerProgressTTL (les stats agrégées ne
// bougent qu'entre deux matchs).
const DefaultRemoteStatsTTL = 5 * time.Minute

const remoteStatsLogModule = "remote_stats_cache"

// Compteurs expvar (exposés via /debug/vars) pour mesurer l'efficacité du cache
// stats carrière remote — même style que career_live_metrics.go.
var (
	remoteStatsCacheHit   = expvar.NewInt("remote_stats.cache_hit")
	remoteStatsCacheMiss  = expvar.NewInt("remote_stats.cache_miss")
	remoteStatsFetchError = expvar.NewInt("remote_stats.fetch_error")
)

type remoteStatsEntry struct {
	stats     *domain.NormalizedPlayerStats
	fetchedAt time.Time
}

// CachedStatsProvider décore un port.PlayerStatsProvider d'un cache TTL
// process-level + singleflight. Thread-safe. Instancié une fois (singleton) et
// partagé entre toutes les requêtes (Explorer, Compare).
//
// Pas de borne sur le nombre d'entrées : nombre modéré de cibles consultées
// simultanément (cf. même hypothèse que CareerLiveCache). Ajouter une LRU si
// le besoin se présente.
type CachedStatsProvider struct {
	inner port.PlayerStatsProvider
	ttl   time.Duration
	now   func() time.Time

	mu      sync.RWMutex
	entries map[string]remoteStatsEntry

	sf singleflight.Group
}

// NewCachedStatsProvider enveloppe inner. ttl<=0 → DefaultRemoteStatsTTL.
// now nil → time.Now (injectable pour les tests TTL déterministes).
func NewCachedStatsProvider(inner port.PlayerStatsProvider, ttl time.Duration, now func() time.Time) *CachedStatsProvider {
	if ttl <= 0 {
		ttl = DefaultRemoteStatsTTL
	}
	if now == nil {
		now = time.Now
	}
	return &CachedStatsProvider{
		inner:   inner,
		ttl:     ttl,
		now:     now,
		entries: make(map[string]remoteStatsEntry),
	}
}

// FetchRemoteStats sert depuis le cache si l'entrée est fraîche, sinon délègue
// à inner (dédupliqué par singleflight) et mémorise le résultat.
func (c *CachedStatsProvider) FetchRemoteStats(ctx context.Context, gamertag, titleSlug string) (*domain.NormalizedPlayerStats, error) {
	key := titleSlug + "|" + strings.ToLower(strings.TrimSpace(gamertag))

	if cached := c.get(key); cached != nil {
		remoteStatsCacheHit.Add(1)
		slog.DebugContext(ctx, remoteStatsLogModule+": cache hit",
			"gamertag", gamertag, "titleSlug", titleSlug)
		return cached, nil
	}

	v, err, _ := c.sf.Do(key, func() (any, error) {
		// Re-check sous singleflight : une requête concurrente a pu remplir
		// le cache pendant qu'on attendait notre tour.
		if cached := c.get(key); cached != nil {
			remoteStatsCacheHit.Add(1)
			return cached, nil
		}
		remoteStatsCacheMiss.Add(1)
		slog.DebugContext(ctx, remoteStatsLogModule+": cache miss → fetch live",
			"gamertag", gamertag, "titleSlug", titleSlug)
		stats, fErr := c.inner.FetchRemoteStats(ctx, gamertag, titleSlug)
		if fErr != nil {
			remoteStatsFetchError.Add(1)
			return nil, fErr
		}
		c.put(key, stats)
		return stats, nil
	})
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return v.(*domain.NormalizedPlayerStats), nil
}

func (c *CachedStatsProvider) get(key string) *domain.NormalizedPlayerStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok || c.now().Sub(entry.fetchedAt) > c.ttl {
		return nil
	}
	return entry.stats
}

func (c *CachedStatsProvider) put(key string, stats *domain.NormalizedPlayerStats) {
	if stats == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = remoteStatsEntry{stats: stats, fetchedAt: c.now()}
}
