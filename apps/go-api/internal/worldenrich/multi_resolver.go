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

// XUIDResolver — résout un gamertag en xuid numérique. Satisfait par
// *auth.PeopleHubResolver, *auth.XboxProfileResolver et chainResolver. Interface
// locale (duck-typing) pour chaîner des résolveurs hétérogènes sans coupler
// CachingResolver à une implémentation unique.
type XUIDResolver interface {
	ResolveXUID(ctx context.Context, gamertag string) (string, error)
}

// chainResolver essaie une liste ORDONNÉE de résolveurs et retourne le 1er succès.
// Usage (fix #10) : PeopleHub d'ABORD (graphe social, rapide quand le joueur y est)
// puis l'endpoint profil Xbox en FALLBACK (universel — résout TOUT gamertag public,
// y compris les adversaires de matchmaking hors graphe social). Sur échec NON-429
// d'un résolveur (ex. "gamertag absent des résultats peoplehub"), on continue vers
// le suivant ; sur 429, on continue aussi (le profil a une limite distincte). Si
// AUCUN ne résout, on propage la dernière erreur — en préservant un marqueur "429"
// si tous ont throttle (pour que le round-robin du CachingResolver rote de compte).
type chainResolver struct {
	resolvers []XUIDResolver
}

func (c chainResolver) ResolveXUID(ctx context.Context, gamertag string) (string, error) {
	var lastErr error
	allThrottled := true
	for _, r := range c.resolvers {
		x, err := r.ResolveXUID(ctx, gamertag)
		if err == nil {
			return x, nil
		}
		if !strings.Contains(err.Error(), "429") {
			allThrottled = false
		}
		lastErr = err
	}
	if lastErr == nil {
		return "", fmt.Errorf("chain resolver: aucun résolveur configuré")
	}
	if allThrottled {
		// Tous throttlés → garder "429" en surface pour que le round-robin rote.
		return "", fmt.Errorf("chain resolver throttled (429): %w", lastErr)
	}
	return "", lastErr
}

// BuildMultiResolver construit un résolveur xuid PAR compte token résolu. Chaque
// résolveur est une CHAÎNE PeopleHub→Profil Xbox partageant le MÊME header XSTS
// (audience http://xboxlive.com) : PeopleHub pour le graphe social, l'endpoint
// profil en fallback UNIVERSEL (fix #10 — adversaires de matchmaking hors graphe).
// La limite de débit porte PAR COMPTE : répartir les résolutions sur N comptes donne
// ~N× le quota. Miroir de BuildMultiHaloSource côté fetch de matchs. Les comptes sans
// token résolu sont sautés (warn) ; erreur seulement si AUCUN compte n'est résolu.
func BuildMultiResolver(cfg *config.AppConfig, tokenGamertags []string) ([]XUIDResolver, []string, error) {
	provider := auth.NewSISUProvider()
	store := auth.NewMultiUserTokenStore(title.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	var resolvers []XUIDResolver
	var ok []string
	for _, gt := range tokenGamertags {
		xuid, err := xuidForGamertag(cfg, gt)
		if err != nil {
			slog.WarnContext(context.Background(), "world-enrich: resolver xuid skippé (xuid db_profiles introuvable)",
				"gamertag", gt, "err", err)
			continue
		}

		hp := auth.NewCachedHeaderProvider(0, func(ctx context.Context) (string, error) {
			at, e := resolveAccessToken(ctx, provider, store, xuid)
			if e != nil {
				return "", e
			}
			rta, e := auth.AcquireXSTSForRTA(ctx, at)
			if e != nil {
				return "", fmt.Errorf("AcquireXSTSForRTA: %w", e)
			}
			return fmt.Sprintf("XBL3.0 x=%s;%s", rta.UserHash, rta.Token), nil
		})
		// Même headerFn (XSTS xboxlive.com) pour les DEUX endpoints — le projet sait
		// déjà produire ce token (cf. AcquireXSTSForRTA), aucune nouvelle chaîne requise.
		resolvers = append(resolvers, chainResolver{resolvers: []XUIDResolver{
			auth.NewPeopleHubResolver(nil, hp.Header),
			auth.NewXboxProfileResolver(nil, hp.Header),
		}})
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
	resolvers []XUIDResolver
	next      int
	persist   func(gamertag, xuid string) // nil = pas de persistance
	hits      int
	misses    int
}

// NewCachingResolver — seed : associations déjà connues (n'importe quelle casse de
// gamertag). persist : appelé pour chaque NOUVELLE résolution (nil = aucune).
func NewCachingResolver(resolvers []XUIDResolver, seed map[string]string, persist func(gamertag, xuid string)) *CachingResolver {
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
