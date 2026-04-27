// Package duckdb — player_matches_cache.go : decorateur cache pour
// PlayerMatchesRepo.
//
// Strategie : TTL court (5 min par defaut) + eviction FIFO a la capacite +
// coalescence des appels concurrents (singleflight). Pas de LRU stricte
// (recency-based) : la duree de vie courte rend l'eviction FIFO suffisante
// et evite la dependance a une lib externe.
//
// Cle de cache : SHA-256 hex sur la representation JSON canonique des
// filtres (slices tries pour stabilite).
package duckdb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// playerMatchesLoader est l'interface minimale consommee par
// CachedPlayerMatchesRepo. *PlayerMatchesRepo l'implemente naturellement.
// Permet le test du cache sans PlayerDB reel (mock loader).
type playerMatchesLoader interface {
	Load(ctx context.Context, filters port.PlayerMatchFilters) ([]canonical.PlayerMatchRow, error)
}

// CachedPlayerMatchesRepo decore un loader avec un cache TTL + eviction FIFO +
// coalescence singleflight. Threadsafe.
type CachedPlayerMatchesRepo struct {
	inner   playerMatchesLoader
	cache   *ttlCache
	sf      singleflight.Group
	metrics cacheMetrics
}

// cacheMetrics expose les compteurs hit/miss via Snapshot. Mis a jour atomiquement.
type cacheMetrics struct {
	hits   int64
	misses int64
}

// NewCachedPlayerMatchesRepo construit le wrapper. `capacity` = nombre max
// d'entrees ; `ttl` = duree de vie d'une entree.
//
// Defaults raisonnables : 200, 5 * time.Minute.
func NewCachedPlayerMatchesRepo(inner playerMatchesLoader, capacity int, ttl time.Duration) *CachedPlayerMatchesRepo {
	if capacity <= 0 {
		capacity = 200
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &CachedPlayerMatchesRepo{
		inner: inner,
		cache: newTTLCache(capacity, ttl),
	}
}

// Load delegue a inner.Load avec cache + coalescence.
func (c *CachedPlayerMatchesRepo) Load(
	ctx context.Context,
	filters port.PlayerMatchFilters,
) ([]canonical.PlayerMatchRow, error) {
	key := filtersCacheKey(filters)
	if v, ok := c.cache.Get(key); ok {
		atomic.AddInt64(&c.metrics.hits, 1)
		return v, nil
	}
	atomic.AddInt64(&c.metrics.misses, 1)

	v, err, _ := c.sf.Do(key, func() (any, error) {
		rows, err := c.inner.Load(ctx, filters)
		if err != nil {
			return nil, err
		}
		c.cache.Set(key, rows)
		return rows, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]canonical.PlayerMatchRow), nil
}

// Invalidate vide entierement le cache (appel apres sync joueur, par
// exemple). InvalidateKey pourrait etre ajoute si un usage plus fin emerge.
func (c *CachedPlayerMatchesRepo) Invalidate() {
	c.cache.InvalidateAll()
}

// MetricsSnapshot retourne hits/misses observes (lecture atomique).
func (c *CachedPlayerMatchesRepo) MetricsSnapshot() (hits, misses int64) {
	return atomic.LoadInt64(&c.metrics.hits), atomic.LoadInt64(&c.metrics.misses)
}

// filtersCacheKey produit une cle stable pour des filtres equivalents,
// independante de l'ordre des slices. SHA-256 hex sur JSON canonique.
func filtersCacheKey(f port.PlayerMatchFilters) string {
	canonicalFilters := struct {
		Period               *string  `json:"period,omitempty"`
		OutcomeIn            []string `json:"outcome_in,omitempty"`
		HadBotTeammate       *bool    `json:"had_bot_teammate,omitempty"`
		IsFirefight          *bool    `json:"is_firefight,omitempty"`
		IsRanked             *bool    `json:"is_ranked,omitempty"`
		MinTimePlayedSeconds *int     `json:"min_time_played_seconds,omitempty"`
		ExcludeFriendsXUIDs  []string `json:"exclude_friends_xuids,omitempty"`
		BTBExcluded          bool     `json:"btb_excluded,omitempty"`
		PlaylistKind         *string  `json:"playlist_kind,omitempty"`
		MapIDs               []string `json:"map_ids,omitempty"`
		Limit                int      `json:"limit,omitempty"`
		OrderBy              string   `json:"order_by,omitempty"`
	}{
		HadBotTeammate:       f.HadBotTeammate,
		IsFirefight:          f.IsFirefight,
		IsRanked:             f.IsRanked,
		MinTimePlayedSeconds: f.MinTimePlayedSeconds,
		BTBExcluded:          f.BTBExcluded,
		PlaylistKind:         f.PlaylistKind,
		Limit:                f.Limit,
		OrderBy:              f.OrderBy,
	}
	if f.Period != nil {
		s := string(*f.Period)
		canonicalFilters.Period = &s
	}
	for _, o := range f.OutcomeIn {
		canonicalFilters.OutcomeIn = append(canonicalFilters.OutcomeIn, string(o))
	}
	canonicalFilters.ExcludeFriendsXUIDs = append([]string{}, f.ExcludeFriendsXUIDs...)
	canonicalFilters.MapIDs = append([]string{}, f.MapIDs...)
	sort.Strings(canonicalFilters.OutcomeIn)
	sort.Strings(canonicalFilters.ExcludeFriendsXUIDs)
	sort.Strings(canonicalFilters.MapIDs)

	buf, _ := json.Marshal(canonicalFilters)
	sum := sha256.Sum256(buf)
	return hex.EncodeToString(sum[:])
}

// ttlCache est un cache map-based avec TTL et eviction FIFO a la capacite.
// Threadsafe (mu sync.Mutex). Pas de LRU recency-based : la duree de vie
// courte (5 min) rend la perte d'efficacite negligeable.
type ttlCache struct {
	mu       sync.Mutex
	entries  map[string]ttlCacheEntry
	keyOrder []string // FIFO order pour eviction
	capacity int
	ttl      time.Duration
}

type ttlCacheEntry struct {
	value     []canonical.PlayerMatchRow
	expiresAt time.Time
}

func newTTLCache(capacity int, ttl time.Duration) *ttlCache {
	return &ttlCache{
		entries:  make(map[string]ttlCacheEntry, capacity),
		keyOrder: make([]string, 0, capacity),
		capacity: capacity,
		ttl:      ttl,
	}
}

// Get retourne la valeur si presente ET non expiree.
func (c *ttlCache) Get(key string) ([]canonical.PlayerMatchRow, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(e.expiresAt) {
		c.removeLocked(key)
		return nil, false
	}
	return e.value, true
}

// Set insere ou remplace une entree. Evince si capacite depassee (FIFO).
func (c *ttlCache) Set(key string, value []canonical.PlayerMatchRow) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.entries[key]; !exists {
		if len(c.entries) >= c.capacity {
			c.evictOldestLocked()
		}
		c.keyOrder = append(c.keyOrder, key)
	}
	c.entries[key] = ttlCacheEntry{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Invalidate retire une cle precise.
func (c *ttlCache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removeLocked(key)
}

// InvalidateAll vide le cache.
func (c *ttlCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]ttlCacheEntry, c.capacity)
	c.keyOrder = c.keyOrder[:0]
}

// Len retourne le nombre d'entrees actuellement stockees (utile en test).
func (c *ttlCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

func (c *ttlCache) removeLocked(key string) {
	if _, ok := c.entries[key]; !ok {
		return
	}
	delete(c.entries, key)
	for i, k := range c.keyOrder {
		if k == key {
			c.keyOrder = append(c.keyOrder[:i], c.keyOrder[i+1:]...)
			return
		}
	}
}

func (c *ttlCache) evictOldestLocked() {
	if len(c.keyOrder) == 0 {
		return
	}
	oldest := c.keyOrder[0]
	c.keyOrder = c.keyOrder[1:]
	delete(c.entries, oldest)
}
