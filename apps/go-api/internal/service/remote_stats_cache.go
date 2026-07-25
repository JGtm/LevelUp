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

	"levelup/go-api/internal/ctxkeys"
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
	record    *domain.RemoteServiceRecord
	fetchedAt time.Time
}

// seasonStatsEntry : cache d'un compte de matchs par (gamertag, saison, filtre
// ranked). Une saison passée ne bouge jamais ; la saison courante bouge entre
// deux matchs → même TTL court que les stats carrière (acceptable).
type seasonStatsEntry struct {
	matches   int
	fetchedAt time.Time
}

// CachedStatsProvider décore un port.ServiceRecordProvider d'un cache TTL
// process-level + singleflight. Thread-safe. Instancié une fois (singleton) et
// partagé entre toutes les requêtes (Explorer, Compare). Implémente
// port.PlayerStatsProvider (FetchRemoteStats, pour Compare) ET
// port.ServiceRecordProvider (FetchServiceRecord, pour Explorer : stats +
// médailles + temps de jeu).
//
// Pas de borne sur le nombre d'entrées : nombre modéré de cibles consultées
// simultanément (cf. même hypothèse que CareerLiveCache). Ajouter une LRU si
// le besoin se présente.
type CachedStatsProvider struct {
	inner       port.ServiceRecordProvider
	seasonInner port.SeasonStatsProvider // non-nil si inner implémente aussi le fetch par saison
	ttl         time.Duration
	now         func() time.Time

	mu            sync.RWMutex
	entries       map[string]remoteStatsEntry
	seasonEntries map[string]seasonStatsEntry

	sf singleflight.Group
}

// NewCachedStatsProvider enveloppe inner. ttl<=0 → DefaultRemoteStatsTTL.
// now nil → time.Now (injectable pour les tests TTL déterministes).
func NewCachedStatsProvider(inner port.ServiceRecordProvider, ttl time.Duration, now func() time.Time) *CachedStatsProvider {
	if ttl <= 0 {
		ttl = DefaultRemoteStatsTTL
	}
	if now == nil {
		now = time.Now
	}
	c := &CachedStatsProvider{
		inner:         inner,
		ttl:           ttl,
		now:           now,
		entries:       make(map[string]remoteStatsEntry),
		seasonEntries: make(map[string]seasonStatsEntry),
	}
	// inner implémente-t-il aussi le fetch par saison ? (HaloProvider oui ;
	// un mock minimal peut ne pas l'implémenter → FetchSeasonServiceRecord
	// retournera alors (0, nil), dégradation gracieuse.)
	if ss, ok := inner.(port.SeasonStatsProvider); ok {
		c.seasonInner = ss
	}
	return c
}

// FetchServiceRecord sert depuis le cache si l'entrée est fraîche, sinon délègue
// à inner (dédupliqué par singleflight) et mémorise le résultat.
func (c *CachedStatsProvider) FetchServiceRecord(ctx context.Context, gamertag, titleSlug string) (*domain.RemoteServiceRecord, error) {
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
		rec, fErr := c.inner.FetchServiceRecord(ctx, gamertag, titleSlug)
		if fErr != nil {
			remoteStatsFetchError.Add(1)
			return nil, fErr
		}
		c.put(key, rec)
		return rec, nil
	})
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return v.(*domain.RemoteServiceRecord), nil
}

// FetchRemoteStats projette le service record caché sur les stats normalisées
// (sans médailles). Conservé pour Compare (port.PlayerStatsProvider).
func (c *CachedStatsProvider) FetchRemoteStats(ctx context.Context, gamertag, titleSlug string) (*domain.NormalizedPlayerStats, error) {
	rec, err := c.FetchServiceRecord(ctx, gamertag, titleSlug)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, nil
	}
	s := rec.Stats
	return &s, nil
}

func (c *CachedStatsProvider) get(key string) *domain.RemoteServiceRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok || c.now().Sub(entry.fetchedAt) > c.ttl {
		return nil
	}
	return entry.record
}

func (c *CachedStatsProvider) put(key string, rec *domain.RemoteServiceRecord) {
	if rec == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = remoteStatsEntry{record: rec, fetchedAt: c.now()}
}

// FetchSeasonServiceRecord sert depuis le cache si frais, sinon délègue à inner
// (dédupliqué par singleflight) et mémorise. Clé = (gamertag | seasonID | filtre
// ranked). Implémente port.SeasonStatsProvider. (0, nil) si inner ne supporte
// pas le fetch par saison (dégradation gracieuse).
func (c *CachedStatsProvider) FetchSeasonServiceRecord(ctx context.Context, gamertag, seasonID string, isRanked *bool) (int, error) {
	if c.seasonInner == nil {
		return 0, nil
	}
	rk := "all"
	if isRanked != nil {
		if *isRanked {
			rk = scopeRanked
		} else {
			rk = "social"
		}
	}
	// Titre inclus dans la clé (défense en profondeur V72-29, 2026-07-25) : ce cache est
	// un singleton partagé entre tous les titres (registry.go), et le seasonID est un
	// chemin CMS brut ("Seasons/SeasonN.json") — rien ne garantit qu'un autre titre ne
	// réutilisera pas la même chaîne. On aligne donc la clé sur celle de FetchServiceRecord
	// (qui porte déjà titleSlug) pour éliminer tout croisement cross-titre sous un même GT.
	key := "season|" + ctxkeys.TitleSlug(ctx) + "|" + strings.ToLower(strings.TrimSpace(gamertag)) + "|" + seasonID + "|" + rk

	if v, ok := c.getSeason(key); ok {
		remoteStatsCacheHit.Add(1)
		return v, nil
	}
	v, err, _ := c.sf.Do(key, func() (any, error) {
		if cached, ok := c.getSeason(key); ok {
			remoteStatsCacheHit.Add(1)
			return cached, nil
		}
		remoteStatsCacheMiss.Add(1)
		n, fErr := c.seasonInner.FetchSeasonServiceRecord(ctx, gamertag, seasonID, isRanked)
		if fErr != nil {
			remoteStatsFetchError.Add(1)
			return 0, fErr
		}
		c.putSeason(key, n)
		return n, nil
	})
	if err != nil {
		return 0, err
	}
	return v.(int), nil
}

func (c *CachedStatsProvider) getSeason(key string) (int, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.seasonEntries[key]
	if !ok || c.now().Sub(e.fetchedAt) > c.ttl {
		return 0, false
	}
	return e.matches, true
}

func (c *CachedStatsProvider) putSeason(key string, n int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seasonEntries[key] = seasonStatsEntry{matches: n, fetchedAt: c.now()}
}
