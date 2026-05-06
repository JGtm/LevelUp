package pool

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"levelup/go-api/internal/platform/auth"
)

// resolverImpl implémente Resolver avec cache TTL.
// Cache = map[gamertag] → (ResolvedTokens, expiresAt).
// Accès thread-safe via RWMutex.
type resolverImpl struct {
	provider auth.TokenProvider

	mu    sync.RWMutex
	cache map[string]*cachedToken // gamertag → token + expiration

	// TTL avant rafraîchissement (défaut ~3h30, Spartan ~4h)
	cacheTTL time.Duration

	// sources = map gamertag → CredentialSource (sauvegardées pour Refresh())
	sources map[string]CredentialSource
}

// cachedToken encapsule un token échangé + sa date d'expiration estimée.
type cachedToken struct {
	resolved  *ResolvedTokens
	expiresAt time.Time
}

// NewResolver crée un Resolver avec cache TTL.
// cacheTTL : durée avant expiration (défaut ~3h30 pour Spartan ~4h).
func NewResolver(provider auth.TokenProvider, cacheTTL time.Duration) Resolver {
	if cacheTTL == 0 {
		cacheTTL = 3*time.Hour + 30*time.Minute
	}
	return &resolverImpl{
		provider: provider,
		cache:    make(map[string]*cachedToken),
		cacheTTL: cacheTTL,
		sources:  make(map[string]CredentialSource),
	}
}

// Resolve implémente Resolver.Resolve() — échange CredentialSource → ResolvedTokens frais.
// Cache TTL : appels répétés pour le même gamertag rendent le cached résultat jusqu'à expiration.
func (r *resolverImpl) Resolve(ctx context.Context, src CredentialSource) (*ResolvedTokens, error) {
	// Vérifier le cache d'abord (fast path).
	r.mu.RLock()
	cached, ok := r.cache[src.Gamertag]
	r.mu.RUnlock()

	if ok && time.Now().Before(cached.expiresAt) {
		slog.DebugContext(ctx, "pool/resolver: token cached utilisé",
			"gamertag", src.Gamertag,
			"expires_in_s", time.Until(cached.expiresAt).Seconds())
		return cached.resolved, nil
	}

	// Cache miss ou expiré → échanger.
	slog.DebugContext(ctx, "pool/resolver: échange token",
		"gamertag", src.Gamertag, "source", src.Source)

	// Pipeline : TrySilentRefresh → TryOAuthRefresh → Exchange.
	var accessToken string
	var exchangeErr error

	// Étape 1 : TrySilentRefresh (si MSAL cache présent).
	if src.MSALCache != "" {
		token, err := r.provider.TrySilentRefresh(ctx, src.MSALCache)
		if err != nil {
			slog.WarnContext(ctx, "pool/resolver: TrySilentRefresh erreur, fallback OAuth",
				"gamertag", src.Gamertag, "err", err)
		} else if token != "" {
			accessToken = token
			slog.DebugContext(ctx, "pool/resolver: TrySilentRefresh OK",
				"gamertag", src.Gamertag)
		} else {
			slog.InfoContext(ctx, "pool/resolver: TrySilentRefresh impossible (cache expiré?), fallback OAuth",
				"gamertag", src.Gamertag)
		}
	} else {
		slog.DebugContext(ctx, "pool/resolver: pas de MSAL cache, tentative OAuth directe",
			"gamertag", src.Gamertag)
	}

	// Étape 2 : TryOAuthRefresh (si pas d'access_token et refresh_token présent).
	if accessToken == "" && src.RefreshToken != "" {
		token, err := r.provider.TryOAuthRefresh(ctx, src.RefreshToken)
		if err != nil {
			slog.WarnContext(ctx, "pool/resolver: TryOAuthRefresh erreur",
				"gamertag", src.Gamertag, "err", err)
			// Non-fatal : on retourne une erreur mais le joueur ne cause pas de crash.
			return nil, fmt.Errorf("pool/resolver: TryOAuthRefresh échoué pour %s: %w", src.Gamertag, err)
		}
		if token != "" {
			accessToken = token
			slog.InfoContext(ctx, "pool/resolver: OAuth v2 fallback OK",
				"gamertag", src.Gamertag)
		} else {
			slog.WarnContext(ctx, "pool/resolver: TryOAuthRefresh retourné vide",
				"gamertag", src.Gamertag)
			return nil, fmt.Errorf("pool/resolver: pas de token OAuth pour %s", src.Gamertag)
		}
	}

	// Étape 3 : Exchange.
	if accessToken == "" {
		err := fmt.Errorf("pool/resolver: aucun accessToken obtenu pour %s (pas de MSAL cache et pas de refresh_token)", src.Gamertag)
		slog.ErrorContext(ctx, "pool/resolver: impossible d'obtenir accessToken",
			"gamertag", src.Gamertag, "err", err)
		return nil, err
	}

	start := time.Now()
	result, exchangeErr := r.provider.Exchange(ctx, accessToken)
	if exchangeErr != nil {
		slog.ErrorContext(ctx, "pool/resolver: Exchange échoué",
			"gamertag", src.Gamertag, "err", exchangeErr)
		return nil, fmt.Errorf("pool/resolver: Exchange échoué pour %s: %w", src.Gamertag, exchangeErr)
	}

	slog.InfoContext(ctx, "pool/resolver: Exchange OK",
		"gamertag", src.Gamertag, "duration_ms", time.Since(start).Milliseconds())

	// Emballer le résultat dans ResolvedTokens.
	resolved := &ResolvedTokens{
		Gamertag:  src.Gamertag,
		XUID:      src.XUID,
		Tokens:    result.Tokens,
		ExpiresAt: time.Now().Add(r.cacheTTL),
		Source:    src.Source,
	}

	// Mettre en cache + sauvegarder la source pour Refresh().
	r.mu.Lock()
	r.cache[src.Gamertag] = &cachedToken{
		resolved:  resolved,
		expiresAt: resolved.ExpiresAt,
	}
	r.sources[src.Gamertag] = src
	r.mu.Unlock()

	slog.DebugContext(ctx, "pool/resolver: token mis en cache",
		"gamertag", src.Gamertag, "ttl_s", r.cacheTTL.Seconds())

	return resolved, nil
}

// Refresh force un re-échange du token pour un gamertag donné (par ex après un 401/403).
// Retourne ErrNoCredentialSource si le gamertag n'a jamais été resolveé.
func (r *resolverImpl) Refresh(ctx context.Context, gamertag string) (*ResolvedTokens, error) {
	r.mu.RLock()
	src, ok := r.sources[gamertag]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("pool/resolver: aucune source de credentials pour %s (jamais resolveé)", gamertag)
	}

	slog.InfoContext(ctx, "pool/resolver: force refresh du token",
		"gamertag", gamertag, "source", src.Source)

	// Invalider le cache.
	r.mu.Lock()
	delete(r.cache, gamertag)
	r.mu.Unlock()

	// Ré-échanger.
	return r.Resolve(ctx, src)
}
