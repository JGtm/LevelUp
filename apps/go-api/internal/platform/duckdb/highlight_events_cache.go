// Package duckdb — highlight_events_cache.go : decorateur cache pour
// HighlightEventsRepo.
//
// Meme strategie que player_matches_cache.go : TTL court (5 min par defaut)
// + eviction FIFO + coalescence singleflight. La duplication est volontaire
// pour eviter de generaliser le ttlCache existant ; si un 3e cache emerge,
// extraire un ttlCacheGeneric[V any] sera la prochaine etape.
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

// highlightEventsLoader est l'interface minimale consommee par
// CachedHighlightEventsRepo. *HighlightEventsRepo l'implemente naturellement.
// Permet le test du cache via mock loader (sans DB).
type highlightEventsLoader interface {
	Load(ctx context.Context, filters port.HighlightEventFilters) ([]canonical.HighlightEvent, error)
}

// CachedHighlightEventsRepo decore un loader avec un cache TTL + eviction FIFO
// + coalescence singleflight. Threadsafe.
type CachedHighlightEventsRepo struct {
	inner   highlightEventsLoader
	cache   *ttlCacheHE
	sf      singleflight.Group
	metrics cacheMetrics // reutilise le type defini dans player_matches_cache.go
}

// NewCachedHighlightEventsRepo construit le wrapper. capacity = nb max
// d'entrees ; ttl = duree de vie d'une entree.
//
// Defaults raisonnables : 200, 5 * time.Minute.
func NewCachedHighlightEventsRepo(inner highlightEventsLoader, capacity int, ttl time.Duration) *CachedHighlightEventsRepo {
	if capacity <= 0 {
		capacity = 200
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &CachedHighlightEventsRepo{
		inner: inner,
		cache: newTTLCacheHE(capacity, ttl),
	}
}

// Load delegue a inner.Load avec cache + coalescence singleflight.
func (c *CachedHighlightEventsRepo) Load(
	ctx context.Context,
	filters port.HighlightEventFilters,
) ([]canonical.HighlightEvent, error) {
	key := highlightEventsCacheKey(filters)
	if v, ok := c.cache.Get(key); ok {
		atomic.AddInt64(&c.metrics.hits, 1)
		return v, nil
	}
	atomic.AddInt64(&c.metrics.misses, 1)

	v, err, _ := c.sf.Do(key, func() (any, error) {
		events, err := c.inner.Load(ctx, filters)
		if err != nil {
			return nil, err
		}
		c.cache.Set(key, events)
		return events, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]canonical.HighlightEvent), nil
}

// Invalidate vide entierement le cache (appel apres sync match, par exemple).
func (c *CachedHighlightEventsRepo) Invalidate() {
	c.cache.InvalidateAll()
}

// MetricsSnapshot retourne hits/misses observes (lecture atomique).
func (c *CachedHighlightEventsRepo) MetricsSnapshot() (hits, misses int64) {
	return atomic.LoadInt64(&c.metrics.hits), atomic.LoadInt64(&c.metrics.misses)
}

// highlightEventsCacheKey produit une cle stable pour des filtres equivalents,
// independante de l'ordre des slices. SHA-256 hex sur JSON canonique.
func highlightEventsCacheKey(f port.HighlightEventFilters) string {
	canonicalFilters := struct {
		MatchIDs   []string `json:"match_ids,omitempty"`
		PlayerXUID *string  `json:"player_xuid,omitempty"`
		EventTypes []string `json:"event_types,omitempty"`
		Since      *string  `json:"since,omitempty"`
		Limit      int      `json:"limit,omitempty"`
		OrderBy    string   `json:"order_by,omitempty"`
	}{
		PlayerXUID: f.PlayerXUID,
		Limit:      f.Limit,
		OrderBy:    f.OrderBy,
	}
	if f.Since != nil {
		s := f.Since.UTC().Format(time.RFC3339Nano)
		canonicalFilters.Since = &s
	}
	canonicalFilters.MatchIDs = append([]string{}, f.MatchIDs...)
	for _, t := range f.EventTypes {
		canonicalFilters.EventTypes = append(canonicalFilters.EventTypes, string(t))
	}
	sort.Strings(canonicalFilters.MatchIDs)
	sort.Strings(canonicalFilters.EventTypes)

	buf, _ := json.Marshal(canonicalFilters)
	sum := sha256.Sum256(buf)
	return hex.EncodeToString(sum[:])
}

// ttlCacheHE est le pendant de ttlCache pour HighlightEvent. Code identique a
// la duplication pres ; voir player_matches_cache.go pour la rationale.
type ttlCacheHE struct {
	mu       sync.Mutex
	entries  map[string]ttlCacheEntryHE
	keyOrder []string
	capacity int
	ttl      time.Duration
}

type ttlCacheEntryHE struct {
	value     []canonical.HighlightEvent
	expiresAt time.Time
}

func newTTLCacheHE(capacity int, ttl time.Duration) *ttlCacheHE {
	return &ttlCacheHE{
		entries:  make(map[string]ttlCacheEntryHE, capacity),
		keyOrder: make([]string, 0, capacity),
		capacity: capacity,
		ttl:      ttl,
	}
}

func (c *ttlCacheHE) Get(key string) ([]canonical.HighlightEvent, bool) {
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

func (c *ttlCacheHE) Set(key string, value []canonical.HighlightEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.entries[key]; !exists {
		if len(c.entries) >= c.capacity {
			c.evictOldestLocked()
		}
		c.keyOrder = append(c.keyOrder, key)
	}
	c.entries[key] = ttlCacheEntryHE{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *ttlCacheHE) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]ttlCacheEntryHE, c.capacity)
	c.keyOrder = c.keyOrder[:0]
}

func (c *ttlCacheHE) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

func (c *ttlCacheHE) removeLocked(key string) {
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

func (c *ttlCacheHE) evictOldestLocked() {
	if len(c.keyOrder) == 0 {
		return
	}
	oldest := c.keyOrder[0]
	c.keyOrder = c.keyOrder[1:]
	delete(c.entries, oldest)
}
