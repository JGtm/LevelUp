package assets

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

const (
	// negativeCacheSize borne le nombre de refus mémorisés.
	negativeCacheSize = 4096
	// negativeCacheTTL est la durée pendant laquelle un refus est resservi sans
	// réinterroger le catalogue ni la source distante.
	negativeCacheTTL = 10 * time.Minute
)

// gateResolver est la part du resolver que la porte HTTP utilise.
type gateResolver interface {
	// LookupLocal sert ref depuis le stock local seulement (jamais d'appel sortant).
	LookupLocal(ctx context.Context, ref Ref) (Resolved, bool)
	// Get sert ref, source distante comprise en cas de manque local.
	Get(ctx context.Context, ref Ref) (Resolved, error)
}

// Gate est l'entrée des assets pour les requêtes HTTP : seule une référence que le
// catalogue local reconnaît peut déclencher un appel sortant. Le stock local passe
// d'abord (un asset déjà présent est servi sans consulter le catalogue), si bien que
// les images nombreuses d'une page ne coûtent rien de plus qu'avant.
type Gate struct {
	resolver gateResolver
	catalog  Catalog
	refused  *negativeCache
}

// NewGate construit la porte devant resolver, jugée par catalog.
func NewGate(resolver gateResolver, catalog Catalog) *Gate {
	return &Gate{
		resolver: resolver,
		catalog:  catalog,
		refused:  newNegativeCache(negativeCacheSize, negativeCacheTTL),
	}
}

// Get sert ref.
//
//   - ref mal formée (Ref.Validate) : ErrInvalidRef, sans lecture ni appel sortant ;
//   - stock local : servi ;
//   - refus mémorisé : ErrNotFound, sans autre travail ;
//   - identifiant inconnu du catalogue : ErrNotFound, sans appel sortant ;
//   - kind sans catalogue local : appel sortant pour un appelant de confiance
//     (trusted : session authentifiée, ou instance sans authentification) seulement ;
//   - identifiant connu : résolution complète. Un 404 amont est mémorisé.
func (g *Gate) Get(ctx context.Context, ref Ref, trusted bool) (Resolved, error) {
	if err := ref.Validate(); err != nil {
		slog.WarnContext(ctx, "assets: référence refusée", append(ref.LogAttrs(), "err", err)...)
		return Resolved{}, err
	}
	if res, ok := g.resolver.LookupLocal(ctx, ref); ok {
		return res, nil
	}
	key := ref.String()
	if g.refused.has(key) {
		return Resolved{}, ErrNotFound
	}
	switch g.catalog.Classify(ctx, ref) {
	case VerdictUnknown:
		g.refused.add(key)
		slog.DebugContext(ctx, "assets: identifiant absent du catalogue local, aucun appel sortant", ref.LogAttrs()...)
		return Resolved{}, ErrNotFound
	case VerdictUnverifiable:
		if !trusted {
			slog.DebugContext(ctx, "assets: kind sans catalogue local, appelant anonyme servi en local seulement", ref.LogAttrs()...)
			return Resolved{}, ErrNotFound
		}
	case VerdictKnown:
	}
	res, err := g.resolver.Get(ctx, ref)
	if errors.Is(err, ErrNotFound) {
		g.refused.add(key)
	}
	return res, err
}

// negativeCache mémorise des clés refusées pour une durée et en nombre bornés.
type negativeCache struct {
	mu      sync.Mutex
	max     int
	ttl     time.Duration
	entries map[string]time.Time // clé → échéance
}

func newNegativeCache(max int, ttl time.Duration) *negativeCache {
	return &negativeCache{max: max, ttl: ttl, entries: make(map[string]time.Time, max)}
}

// has dit si key est un refus encore valable.
func (c *negativeCache) has(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	exp, ok := c.entries[key]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(c.entries, key)
		return false
	}
	return true
}

// add mémorise key. À saturation, les entrées échues sont retirées ; s'il n'en reste
// pas moins que le plafond, le cache est vidé : sa taille ne dépasse jamais max.
func (c *negativeCache) add(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, present := c.entries[key]; !present && len(c.entries) >= c.max {
		now := time.Now()
		for k, exp := range c.entries {
			if now.After(exp) {
				delete(c.entries, k)
			}
		}
		if len(c.entries) >= c.max {
			clear(c.entries)
		}
	}
	c.entries[key] = time.Now().Add(c.ttl)
}

// len rend le nombre d'entrées (échues comprises).
func (c *negativeCache) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}
