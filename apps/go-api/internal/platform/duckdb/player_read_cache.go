// Package duckdb — player_read_cache.go : cache process-wide des lectures d'un
// joueur (lignes de filtres, historique canonique), clé (xuid, titre, base, variante).
//
// Plan perf 2026-09-23, lot L5b (décisions D5b.3 et D5b.4). Une page joueur relisait
// tout l'historique à chaque appel : `/filters/resolve` deux ou trois fois par page
// (0,2 à 0,6 s chacun, sans cache serveur), `LoadPlayerMatches` une ou plusieurs fois
// par page (environ 100 ms de SQL).
//
// # Invalidation
//
// L'unité d'invalidation est le couple (xuid, titre) : InvalidatePlayerReadCaches
// retire toutes ses entrées, toutes variantes et toutes bases confondues. Appelants :
// la fin du pipeline post-sync du joueur (`sync.runPostSyncPipeline`, chemins V1 et
// V2), le recalcul « avec amis » hors sync (`sync.RecomputeIsWithFriends`) et
// l'exclusion d'un match (`MatchExclusionRepo.SetExclusion`). Le TTL court
// (playerReadCacheTTL) est le filet de ce qu'aucun point n'invalide : écriture d'un
// autre process (CLI, backfill), synchronisation Halo 5 (runner hors `internal/sync`),
// import OpenSpartan.
//
// Une génération par (xuid, titre) protège d'une course : un chargement commencé
// AVANT une invalidation ne remplit pas le cache à son retour (son résultat peut
// précéder l'écriture), et une requête arrivée après l'invalidation ne se greffe pas
// sur lui (la clé de coalescence porte la génération).
//
// # Lignes partagées
//
// La valeur cachée n'est JAMAIS rendue telle quelle : chaque appelant reçoit une
// copie (fonction clone du cache), qu'il peut trier ou modifier sans toucher ni au
// cache ni aux autres requêtes.
package duckdb

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/timing"
)

const (
	// playerReadCacheTTL : durée de vie d'une entrée — filet de sécurité, cf. en-tête.
	playerReadCacheTTL = 60 * time.Second
	// playerReadCacheCapacity : nombre maximal d'entrées par cache (éviction FIFO
	// au-delà). Une entrée par (joueur, base, variante) : une poignée de joueurs
	// suivis et de jeux de filtres tient très en dessous.
	playerReadCacheCapacity = 256
)

// filterRowsReadCache : lignes de LoadMatchesForFilters (D5b.3). Une seule variante :
// la lecture ne dépend d'aucun paramètre — les contextes solo, escouade et aperçu
// sont calculés en Go depuis les mêmes lignes (FiltersService.Resolve).
var filterRowsReadCache = newPlayerReadCache("filter_rows_cache", cloneFilterRows)

// InvalidatePlayerReadCaches retire des caches de lectures toutes les entrées du
// joueur `xuid` pour le titre `titleSlug` — titre vide : tous ses titres. Sans
// effet pour un xuid vide.
func InvalidatePlayerReadCaches(ctx context.Context, xuid, titleSlug string) {
	if xuid == "" {
		return
	}
	removed := playerMatchesReadCache.invalidate(xuid, titleSlug) + filterRowsReadCache.invalidate(xuid, titleSlug)
	slog.DebugContext(ctx, "cache des lectures joueur invalidé",
		"xuid", xuid, "titleSlug", titleSlug, "entries", removed)
}

// playerIdentity est l'unité d'invalidation : (xuid, titre).
type playerIdentity struct {
	xuid      string
	titleSlug string
}

// playerCacheScope est l'identité plus le chemin de la base lue : deux bases qui
// porteraient le même joueur (fixtures de tests, démo) ne partagent rien.
type playerCacheScope struct {
	playerIdentity
	dbPath string
}

// scopeOf rend la portée de cache d'un joueur résolu.
func scopeOf(pdb *PlayerDB) playerCacheScope {
	scope := playerCacheScope{playerIdentity: playerIdentity{xuid: pdb.XUID, titleSlug: pdb.TitleSlug}}
	if pdb.Player != nil {
		scope.dbPath = pdb.Player.Path()
	}
	return scope
}

func (s playerCacheScope) key(variant string) string {
	return s.xuid + "\x00" + s.titleSlug + "\x00" + s.dbPath + "\x00" + variant
}

type readCacheEntry[V any] struct {
	id        playerIdentity
	value     V
	expiresAt time.Time
}

// playerReadCache met en cache une lecture par joueur : TTL, éviction FIFO à la
// capacité, coalescence des chargements concurrents (singleflight), invalidation
// par (xuid, titre) et copie à la lecture. Sûr en concurrence.
type playerReadCache[V any] struct {
	name     string // préfixe des marqueurs timing `<name>_hit` / `<name>_miss`
	ttl      time.Duration
	capacity int
	clone    func(V) V
	now      func() time.Time

	mu      sync.Mutex
	entries map[string]readCacheEntry[V]
	order   []string // ordre d'insertion (éviction FIFO)
	gens    map[playerIdentity]uint64

	flights singleflight.Group
}

func newPlayerReadCache[V any](name string, clone func(V) V) *playerReadCache[V] {
	return &playerReadCache[V]{
		name:     name,
		ttl:      playerReadCacheTTL,
		capacity: playerReadCacheCapacity,
		clone:    clone,
		now:      time.Now,
		entries:  make(map[string]readCacheEntry[V]),
		gens:     make(map[playerIdentity]uint64),
	}
}

// load rend une copie de la valeur cachée pour (scope, variant), ou la charge via
// loadFn. Pose un marqueur timing de durée nulle `<name>_hit` ou `<name>_miss` : la
// durée du chargement reste portée par la section de l'appelant (les sections
// timing sont des feuilles). Une erreur de chargement n'est pas cachée.
func (c *playerReadCache[V]) load(
	ctx context.Context,
	scope playerCacheScope,
	variant string,
	loadFn func(context.Context) (V, error),
) (V, error) {
	key := scope.key(variant)
	if v, ok := c.get(key); ok {
		timing.FromContext(ctx).Section(c.name + "_hit")()
		return c.clone(v), nil
	}
	timing.FromContext(ctx).Section(c.name + "_miss")()

	gen := c.generation(scope.playerIdentity)
	flightKey := key + "\x00" + strconv.FormatUint(gen, 10)
	v, err, shared := c.flights.Do(flightKey, func() (any, error) {
		return c.fetch(ctx, key, scope.playerIdentity, gen, loadFn)
	})
	if err != nil && shared && ctx.Err() == nil && isContextEnd(err) {
		// Le chargement partagé a été annulé avec la requête qui le portait ; celle-ci,
		// toujours vivante, recharge pour son propre compte.
		v, err = c.fetch(ctx, key, scope.playerIdentity, gen, loadFn)
	}
	if err != nil {
		var zero V
		return zero, err
	}
	return c.clone(v.(V)), nil
}

// fetch charge la valeur et la met en cache si aucune invalidation n'est survenue
// depuis la génération `gen` lue avant le chargement.
func (c *playerReadCache[V]) fetch(
	ctx context.Context,
	key string,
	id playerIdentity,
	gen uint64,
	loadFn func(context.Context) (V, error),
) (any, error) {
	v, err := loadFn(ctx)
	if err != nil {
		return nil, err
	}
	c.store(key, id, gen, v)
	return v, nil
}

// isContextEnd dit si err vient de l'annulation ou de l'échéance d'un contexte.
func isContextEnd(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// get rend la valeur cachée si elle existe et n'a pas expiré.
func (c *playerReadCache[V]) get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if ok && c.now().Before(e.expiresAt) {
		return e.value, true
	}
	if ok {
		c.removeLocked(key)
	}
	var zero V
	return zero, false
}

// generation rend la génération courante de l'identité (en l'enregistrant : une
// invalidation ultérieure doit la faire monter).
func (c *playerReadCache[V]) generation(id playerIdentity) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	g, ok := c.gens[id]
	if !ok {
		c.gens[id] = 0
	}
	return g
}

// store insère la valeur, sauf si l'identité a été invalidée depuis `gen`.
func (c *playerReadCache[V]) store(key string, id playerIdentity, gen uint64, v V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.gens[id] != gen {
		return
	}
	if _, exists := c.entries[key]; !exists {
		if len(c.entries) >= c.capacity && len(c.order) > 0 {
			oldest := c.order[0]
			c.order = c.order[1:]
			delete(c.entries, oldest)
		}
		c.order = append(c.order, key)
	}
	c.entries[key] = readCacheEntry[V]{id: id, value: v, expiresAt: c.now().Add(c.ttl)}
}

// invalidate retire les entrées de (xuid, titre) — titre vide : tous les titres —
// et fait monter leur génération. Rend le nombre d'entrées retirées.
func (c *playerReadCache[V]) invalidate(xuid, titleSlug string) int {
	matches := func(id playerIdentity) bool {
		return id.xuid == xuid && (titleSlug == "" || id.titleSlug == titleSlug)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for id := range c.gens {
		if matches(id) {
			c.gens[id]++
		}
	}
	removed := 0
	kept := c.order[:0]
	for _, key := range c.order {
		if matches(c.entries[key].id) {
			delete(c.entries, key)
			removed++
			continue
		}
		kept = append(kept, key)
	}
	c.order = kept
	return removed
}

// removeLocked retire une entrée (c.mu tenu).
func (c *playerReadCache[V]) removeLocked(key string) {
	delete(c.entries, key)
	if i := slices.Index(c.order, key); i >= 0 {
		c.order = slices.Delete(c.order, i, i+1)
	}
}

// cloneFilterRows copie les lignes de filtres (copie des valeurs) : un appelant peut
// trier la tranche rendue (FilteredMatchIDs trie ses lignes filtrées, qui peuvent
// être la tranche d'entrée elle-même) sans toucher au cache. Les pointeurs des lignes
// (noms, dates) restent partagés : leur seul consommateur, FiltersService
// (`service/filters_service.go`, `service/filters_options.go`), n'écrit jamais à
// travers eux (vérifié par grep le 2026-09-23).
func cloneFilterRows(rows []domain.FilterMatchRow) []domain.FilterMatchRow {
	return slices.Clone(rows)
}
