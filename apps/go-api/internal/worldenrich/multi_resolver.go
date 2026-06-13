package worldenrich

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
)

// BuildMultiResolver construit un résolveur PeopleHub PAR compte token résolu.
// La limite de débit PeopleHub porte sur l'endpoint /users/me/... (donc PAR compte) :
// répartir les résolutions xuid sur N comptes donne ~N× le quota. Mirroir de
// BuildMultiHaloSource côté fetch de matchs. Les comptes sans token résolu sont
// sautés (warn) ; erreur seulement si AUCUN compte n'est résolu.
func BuildMultiResolver(cfg *config.AppConfig, tokenGamertags []string) ([]*auth.PeopleHubResolver, []string, error) {
	provider := auth.NewMSALProvider()
	store := auth.NewMultiUserTokenStore(title.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	var resolvers []*auth.PeopleHubResolver
	var ok []string
	for _, gt := range tokenGamertags {
		xuid, err := xuidForGamertag(cfg, gt)
		if err != nil {
			slog.WarnContext(context.Background(), "world-enrich: resolver xuid skippé (xuid db_profiles introuvable)",
				"gamertag", gt, "err", err)
			continue
		}
		legacy := loadLegacyInputs(cfg, gt)
		hp := auth.NewCachedHeaderProvider(0, func(ctx context.Context) (string, error) {
			at, e := resolveAccessToken(ctx, provider, store, xuid, legacy)
			if e != nil {
				return "", e
			}
			rta, e := auth.AcquireXSTSForRTA(ctx, at)
			if e != nil {
				return "", fmt.Errorf("AcquireXSTSForRTA: %w", e)
			}
			return fmt.Sprintf("XBL3.0 x=%s;%s", rta.UserHash, rta.Token), nil
		})
		resolvers = append(resolvers, auth.NewPeopleHubResolver(nil, hp.Header))
		ok = append(ok, gt)
	}
	if len(resolvers) == 0 {
		return nil, nil, fmt.Errorf("aucun resolver xuid construit (aucun token résolu)")
	}
	return resolvers, ok, nil
}

// CachingResolver enveloppe N résolveurs PeopleHub (round-robin) + un cache mémoire
// (graine depuis les associations déjà connues) + une persistance optionnelle des
// NOUVELLES associations. Double objectif :
//   - ne JAMAIS re-résoudre un gamertag déjà connu (cache + graine persistée → on ne
//     recommence pas à chaque run, cf. demande user) ;
//   - répartir les résolutions restantes sur tous les comptes (anti rate-limit).
//
// Implémente service.WorldXUIDResolver (signature ResolveXUID identique).
type CachingResolver struct {
	mu        sync.Mutex
	cache     map[string]string // lower(gamertag) -> xuid
	resolvers []*auth.PeopleHubResolver
	next      int
	persist   func(gamertag, xuid string) // nil = pas de persistance
	hits      int
	misses    int
}

// NewCachingResolver — seed : associations déjà connues (n'importe quelle casse de
// gamertag). persist : appelé pour chaque NOUVELLE résolution (nil = aucune).
func NewCachingResolver(resolvers []*auth.PeopleHubResolver, seed map[string]string, persist func(gamertag, xuid string)) *CachingResolver {
	c := &CachingResolver{
		cache:     make(map[string]string, len(seed)),
		resolvers: resolvers,
		persist:   persist,
	}
	for gt, x := range seed {
		if x = strings.TrimSpace(x); x != "" {
			c.cache[strings.ToLower(strings.TrimSpace(gt))] = x
		}
	}
	return c
}

// ResolveXUID : cache d'abord ; sinon round-robin sur les résolveurs (token suivant
// en cas de 429), puis met en cache + persiste la nouvelle association.
func (c *CachingResolver) ResolveXUID(ctx context.Context, gamertag string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(gamertag))
	c.mu.Lock()
	if x, ok := c.cache[key]; ok {
		c.hits++
		c.mu.Unlock()
		return x, nil
	}
	c.misses++
	c.mu.Unlock()

	var lastErr error
	for i := 0; i < len(c.resolvers); i++ {
		c.mu.Lock()
		r := c.resolvers[c.next%len(c.resolvers)]
		c.next++
		c.mu.Unlock()
		x, err := r.ResolveXUID(ctx, gamertag)
		if err == nil {
			c.mu.Lock()
			c.cache[key] = x
			c.mu.Unlock()
			if c.persist != nil {
				c.persist(gamertag, x)
			}
			return x, nil
		}
		lastErr = err
		// 429 → essayer le compte suivant (round-robin). Toute autre erreur (gamertag
		// absent des résultats, etc.) → inutile de roter, on retourne tout de suite.
		if !strings.Contains(err.Error(), "429") {
			return "", err
		}
	}
	return "", lastErr
}

// Stats retourne (hits cache, miss résolus via API). Pour le logging du backfill.
func (c *CachingResolver) Stats() (hits, misses int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hits, c.misses
}
