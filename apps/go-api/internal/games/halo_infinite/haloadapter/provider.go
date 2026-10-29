package haloadapter

// provider.go — le fournisseur Halo Infinite des pages de LevelUp (passe de combat, défis,
// service record, confidentialité, médailles, saisons, actifs Discovery) : une instance par
// processus ([DefaultHaloProvider]), limiteur de débit sortant partagé par ses clones, jetons et
// xuid lus dans le contexte (`ctxkeys.HaloTokens`, `ctxkeys.HaloXUID`) sauf pour les CLI batch
// ([Provider.WithTokens]). Le transport est celui de la brique, sous la politique
// `infinite.RetryProvider` (Discovery : `infinite.RetryDiscovery`).

import (
	"context"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/haloapi/infinite"
)

// Hints d'erreur retournés dans domain.*Response.ErrorHint pour signaler pourquoi un appel live
// a échoué côté fournisseur.
const (
	errHintAuthRequired = "auth_required"
	errHintFetchError   = "fetch_error"
)

const (
	// providerMaxRetries : essais de la politique du fournisseur.
	providerMaxRetries = 3
	// providerHTTPTimeout : délai de chaque requête du fournisseur (porté par son client HTTP).
	providerHTTPTimeout = 15 * time.Second
	// providerDefaultRatePerMinute : débit sortant par défaut du fournisseur.
	providerDefaultRatePerMinute = 60
)

// rateLimiter implémente un seau à jetons pour limiter les requêtes sortantes.
type rateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // jetons par seconde
	lastRefill time.Time
}

func newRateLimiter(maxPerMinute int) *rateLimiter {
	return &rateLimiter{
		tokens:     float64(maxPerMinute),
		maxTokens:  float64(maxPerMinute),
		refillRate: float64(maxPerMinute) / 60.0,
		lastRefill: time.Now(),
	}
}

// tryConsume tente de consommer un jeton, sans bloquer ; rend l'attente avant le suivant.
func (r *rateLimiter) tryConsume() (bool, time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	r.tokens += now.Sub(r.lastRefill).Seconds() * r.refillRate
	if r.tokens > r.maxTokens {
		r.tokens = r.maxTokens
	}
	r.lastRefill = now
	if r.tokens >= 1.0 {
		r.tokens--
		return true, 0
	}
	return false, time.Duration((1.0-r.tokens)/r.refillRate*1000) * time.Millisecond
}

// Wait bloque jusqu'à ce qu'un jeton soit disponible ou que ctx soit annulé.
func (r *rateLimiter) Wait(ctx context.Context) error {
	for {
		ok, wait := r.tryConsume()
		if ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

// TrackDefinitionPersister est notifié après chaque résolution réussie d'une définition de Reward
// Track (GameCMS), pour la persister dans battlepass_track_definitions. Implémenté par
// platform/duckdb.PersistSink.
type TrackDefinitionPersister interface {
	UpsertTrackDefinition(ctx context.Context, trackPath string, raw []byte) error
}

// ItemDefinitionPersister est notifié après chaque résolution réussie d'un objet de passe de
// combat, pour le persister dans battlepass_item_definitions + _translations. Implémenté par
// platform/duckdb.PersistSink.
type ItemDefinitionPersister interface {
	UpsertItemDefinition(ctx context.Context, itemPath string, raw []byte) error
}

// Provider est le fournisseur Halo Infinite (343 Industries). Sûr en concurrence : une seule
// instance par processus ; ses clones `With*` partagent son limiteur (sauf WithRateLimit) et son
// cache de confidentialité.
type Provider struct {
	// client émet les requêtes (derrière la garde netguard de la surface) et porte le délai.
	client     *http.Client
	limiter    *rateLimiter
	maxRetries int
	// Surcharges d'hôte pour les tests : vides en production (le résolveur de titre sert).
	battlePassBaseURL string
	challengesBaseURL string
	gameCMSBaseURL    string
	// assetResolver est le résolveur unifié des définitions et images.
	assetResolver assets.Resolver
	// trackDefPersister persiste les définitions de pistes dans battlepass_track_definitions.
	trackDefPersister TrackDefinitionPersister
	// itemDefPersister persiste les définitions d'objets dans battlepass_item_definitions.
	itemDefPersister ItemDefinitionPersister
	// titleSlug est le titre courant (ex. "halo_infinite").
	titleSlug string
	// privacyCache : cache de la confidentialité par xuid, partagé par les clones (pointeur).
	privacyCache *privacyTTLCache
	// staticTokens : jetons injectés par WithTokens pour les CLI batch (hors contexte HTTP).
	staticTokens *domain.HaloTokens
}

// DefaultHaloProvider est l'instance globale du fournisseur (60 req/min, 3 essais).
var DefaultHaloProvider = NewHaloProvider()

// challengesFetchSFGroup déduplique les lectures live concurrentes des défis. La clé inclut l'hôte
// des défis pour ne partager que des réponses construites dans le même contexte.
var challengesFetchSFGroup singleflight.Group

// NewHaloProvider crée un fournisseur avec les paramètres par défaut.
func NewHaloProvider() *Provider {
	return &Provider{
		client:       &http.Client{Timeout: providerHTTPTimeout},
		limiter:      newRateLimiter(providerDefaultRatePerMinute),
		maxRetries:   providerMaxRetries,
		privacyCache: &privacyTTLCache{entries: make(map[string]privacyCacheEntry)},
	}
}

// WithRateLimit rend un clone sur un limiteur NEUF de maxPerMinute (opérations batch sur des API
// publiques).
func (p *Provider) WithRateLimit(maxPerMinute int) *Provider {
	if p == nil {
		return nil
	}
	clone := *p
	clone.limiter = newRateLimiter(maxPerMinute)
	return &clone
}

// WithTokens rend un clone porteur de jetons statiques (CLI batch hors contexte HTTP) : ce sont
// eux que lisent les requêtes Discovery UGC, qui exigent l'authentification.
func (p *Provider) WithTokens(tokens *domain.HaloTokens) *Provider {
	if p == nil {
		return nil
	}
	clone := *p
	clone.staticTokens = tokens
	return &clone
}

// WithAssetResolver rend un clone câblé sur le résolveur unifié des actifs.
func (p *Provider) WithAssetResolver(resolver assets.Resolver) *Provider {
	if p == nil {
		return nil
	}
	clone := *p
	clone.assetResolver = resolver
	return &clone
}

// WithTrackDefPersister rend un clone qui persiste chaque définition de piste résolue.
func (p *Provider) WithTrackDefPersister(persister TrackDefinitionPersister) *Provider {
	if p == nil {
		return nil
	}
	clone := *p
	clone.trackDefPersister = persister
	return &clone
}

// WithItemDefPersister rend un clone qui persiste chaque définition d'objet résolue.
func (p *Provider) WithItemDefPersister(persister ItemDefinitionPersister) *Provider {
	if p == nil {
		return nil
	}
	clone := *p
	clone.itemDefPersister = persister
	return &clone
}

// WithTitleSlug rend un clone pour le titre `slug` (vide : title.DefaultSlug).
func (p *Provider) WithTitleSlug(slug string) *Provider {
	if p == nil {
		return nil
	}
	clone := *p
	if slug == "" {
		clone.titleSlug = title.DefaultSlug
	} else {
		clone.titleSlug = slug
	}
	return &clone
}

// titleID retourne le titre effectif (repli title.DefaultSlug).
func (p *Provider) titleID() string {
	if p.titleSlug == "" {
		return title.DefaultSlug
	}
	return p.titleSlug
}

// adresses rend les adresses de la brique pour le titre du contexte : surcharge d'instance
// d'abord, puis le résolveur partagé, puis le défaut.
func (p *Provider) adresses(ctx context.Context, cles ...infinite.Cle) infinite.Endpoints {
	s := surcharges{
		infinite.CleEconomy:    p.battlePassBaseURL,
		infinite.CleChallenges: p.challengesBaseURL,
		infinite.CleGameCMS:    p.gameCMSBaseURL,
	}
	e := EndpointsDepuisResolveur(ctx, games.DefaultEndpointResolver(), ctxkeys.TitleSlug(ctx), s.restantes(cles)...)
	return s.appliquer(e)
}

// hostFor rend l'hôte de la clé pour le titre du contexte.
func (p *Provider) hostFor(ctx context.Context, cle infinite.Cle) string {
	return p.adresses(ctx, cle).Hote(cle)
}

// api rend le client de la brique pour un appel du fournisseur : sortie gardée sous `surface`,
// politique `politique` (essais du fournisseur), limiteur partagé, adresses des clés `cles`.
func (p *Provider) api(ctx context.Context, surface string, politique infinite.Retry, cles ...infinite.Cle) *infinite.Client {
	politique.Essais = p.maxRetries
	c := infinite.New(netguardDoer{surface: surface, next: p.client}, politique).WithEndpoints(p.adresses(ctx, cles...))
	if p.limiter != nil {
		c = c.WithLimiter(p.limiter)
	}
	return c
}

// authDe rend les en-têtes de jetons de `tokens` (nil : aucun en-tête).
func authDe(tokens *domain.HaloTokens) infinite.Auth {
	if tokens == nil {
		return infinite.Auth{}
	}
	return infinite.Auth{Spartan: tokens.SpartanToken, Clearance: tokens.ClearanceToken}
}
