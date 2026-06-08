package pool

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"levelup/go-api/internal/observability/logging"
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

	// sources = map gamertag → CredentialSource (sauvegardées pour Refresh()).
	// La valeur RefreshToken peut être mise à jour quand Microsoft rotate le RT
	// (cf. resolveOAuthWithRotation).
	sources map[string]CredentialSource

	// onRotated est invoqué à chaque rotation OAuth réussie pour permettre au
	// caller de persister le nouveau RT (sinon le prochain refresh échouera).
	// Nullable : si nil, la rotation est seulement mise à jour en mémoire.
	onRotated TokenRotationCallback

	// onReauth signale le changement d'état « reauth requise » (RT mort / refresh OK).
	// Nullable : si nil, aucune notification.
	onReauth ReauthCallback

	// sfGroup dédupplique les Resolve concurrents pour un même gamertag. Cf.
	// `golang.org/x/sync/singleflight`. Sans cette protection, N goroutines
	// arrivant en même temps sur un cache miss appellent toutes Microsoft avec
	// le même RT — seule la 1ère réussit, les N-1 autres reçoivent invalid_grant
	// (Microsoft rotate le RT à chaque usage). Avec singleflight, 1 seul appel
	// OAuth est fait, les autres goroutines reçoivent le même résultat.
	//
	// Scénarios concernés en prod :
	//   - Boot serveur : cache vide + plusieurs requêtes HTTP simultanées
	//   - Air hot-reload : pareil, à chaque restart
	//   - Post-token-capture : cache invalidé puis trafic
	sfGroup singleflight.Group
}

// cachedToken encapsule un token échangé + sa date d'expiration estimée.
type cachedToken struct {
	resolved  *ResolvedTokens
	expiresAt time.Time
}

// NewResolver crée un Resolver avec cache TTL.
// cacheTTL : durée avant expiration (défaut ~3h30 pour Spartan ~4h).
// onRotated : callback optionnel invoqué quand Microsoft rotate un RT.
func NewResolver(provider auth.TokenProvider, cacheTTL time.Duration, onRotated TokenRotationCallback) Resolver {
	return NewResolverWithReauth(provider, cacheTTL, onRotated, nil)
}

// NewResolverWithReauth crée un Resolver avec, en plus, le callback de changement
// d'état « ré-authentification requise » (RT mort / refresh OK). PR-B.
func NewResolverWithReauth(provider auth.TokenProvider, cacheTTL time.Duration, onRotated TokenRotationCallback, onReauth ReauthCallback) Resolver {
	if cacheTTL == 0 {
		cacheTTL = 3*time.Hour + 30*time.Minute
	}
	return &resolverImpl{
		provider:  provider,
		cache:     make(map[string]*cachedToken),
		cacheTTL:  cacheTTL,
		sources:   make(map[string]CredentialSource),
		onRotated: onRotated,
		onReauth:  onReauth,
	}
}

// Resolve implémente Resolver.Resolve() — échange CredentialSource → ResolvedTokens frais.
//
// Cache TTL : appels répétés pour le même gamertag rendent le cached résultat
// jusqu'à expiration.
//
// Concurrent dedup via singleflight (champ sfGroup) : N appels Resolve
// simultanés pour le MÊME gamertag sur un cache miss → 1 seul appel OAuth
// vers Microsoft, les N-1 autres goroutines reçoivent le même *ResolvedTokens.
// Élimine le risque d'invalid_grant burst au boot serveur ou post-rotation.
func (r *resolverImpl) Resolve(ctx context.Context, src CredentialSource) (*ResolvedTokens, error) {
	// Fast path : cache hit avant tout — pas besoin de singleflight.
	if cached, ok := r.lookupCachedToken(ctx, src.Gamertag); ok {
		return cached, nil
	}

	// Singleflight : dédupplique les Resolve concurrents par gamertag.
	// La fonction passée à Do n'est exécutée QU'UNE FOIS pour les goroutines
	// concurrentes ; les followers reçoivent le même result/error que le leader.
	result, err, shared := r.sfGroup.Do(src.Gamertag, func() (any, error) {
		// Re-check cache : un Resolve précédent (leader d'un singleflight antérieur)
		// peut avoir peuplé le cache pendant qu'on attendait la place dans sf.Do.
		if cached, ok := r.lookupCachedToken(ctx, src.Gamertag); ok {
			return cached, nil
		}
		return r.resolveExpensive(ctx, src)
	})

	if shared {
		slog.DebugContext(ctx, "pool/resolver: Resolve dédupliqué via singleflight",
			"gamertag", src.Gamertag)
	}

	if err != nil {
		return nil, err
	}
	return result.(*ResolvedTokens), nil
}

// resolveExpensive exécute le pipeline OAuth refresh + Exchange (~100-500ms).
// Appelé UNE SEULE FOIS par batch de Resolve concurrents grâce à sfGroup.Do.
func (r *resolverImpl) resolveExpensive(ctx context.Context, src CredentialSource) (*ResolvedTokens, error) {
	ctx, evID := logging.WithEvent(ctx, "auth.resolve:"+src.Gamertag)
	slog.InfoContext(ctx, "pool/resolver: échange token démarré",
		"gamertag", src.Gamertag, "source", src.Source, "event", evID)

	accessToken, err := r.acquireAccessToken(ctx, &src)
	if err != nil || accessToken == "" {
		// Credentials présents mais aucun token obtenu → refresh_token mort →
		// signaler la ré-authentification requise (best-effort, cf. signalReauth).
		r.signalReauth(ctx, src, true)
		if err != nil {
			return nil, err
		}
		nerr := fmt.Errorf("pool/resolver: aucun accessToken obtenu pour %s (pas de MSAL cache et pas de refresh_token)", src.Gamertag)
		slog.ErrorContext(ctx, "pool/resolver: impossible d'obtenir accessToken",
			"gamertag", src.Gamertag, "err", nerr)
		return nil, nerr
	}

	resolved, xerr := r.exchangeAndCache(ctx, src, accessToken)
	if xerr == nil {
		// Refresh + échange OK → l'éventuel flag reauth_required est obsolète.
		r.signalReauth(ctx, src, false)
	}
	return resolved, xerr
}

// signalReauth invoque onReauth (si présent). Pour required=true (RT mort), ne
// déclenche QUE si des credentials existaient (sinon c'est un compte jamais
// authentifié, pas une « mort »). Pour required=false (succès), déclenche
// toujours afin d'effacer un éventuel flag.
func (r *resolverImpl) signalReauth(ctx context.Context, src CredentialSource, required bool) {
	if r.onReauth == nil {
		return
	}
	if required && src.MSALCache == "" && src.RefreshToken == "" {
		return
	}
	if required {
		slog.WarnContext(ctx, "pool/resolver: refresh_token mort — reauth_required",
			"gamertag", src.Gamertag, "xuid", src.XUID)
	}
	r.onReauth(ctx, src.Gamertag, src.XUID, required)
}

// lookupCachedToken retourne le token cached si encore valide, sinon (nil, false).
func (r *resolverImpl) lookupCachedToken(ctx context.Context, gamertag string) (*ResolvedTokens, bool) {
	r.mu.RLock()
	cached, ok := r.cache[gamertag]
	r.mu.RUnlock()

	if ok && time.Now().Before(cached.expiresAt) {
		slog.DebugContext(ctx, "pool/resolver: token cached utilisé",
			"gamertag", gamertag,
			"expires_in_s", time.Until(cached.expiresAt).Seconds())
		return cached.resolved, true
	}
	return nil, false
}

// acquireAccessToken applique le pipeline TrySilentRefresh → TryOAuthRefresh.
// Mute src.RefreshToken si Microsoft rotate le RT (propagation via onRotated).
func (r *resolverImpl) acquireAccessToken(ctx context.Context, src *CredentialSource) (string, error) {
	// Étape 1 : TrySilentRefresh (si MSAL cache présent).
	if src.MSALCache != "" {
		token, err := r.provider.TrySilentRefresh(ctx, src.MSALCache)
		if err != nil {
			slog.WarnContext(ctx, "pool/resolver: TrySilentRefresh erreur, fallback OAuth",
				"gamertag", src.Gamertag, "err", err)
		} else if token != "" {
			slog.DebugContext(ctx, "pool/resolver: TrySilentRefresh OK",
				"gamertag", src.Gamertag)
			return token, nil
		} else {
			slog.InfoContext(ctx, "pool/resolver: TrySilentRefresh impossible (cache expiré?), fallback OAuth",
				"gamertag", src.Gamertag)
		}
	} else {
		slog.DebugContext(ctx, "pool/resolver: pas de MSAL cache, tentative OAuth directe",
			"gamertag", src.Gamertag)
	}

	// Étape 2 : TryOAuthRefreshWithRotation (si refresh_token présent).
	if src.RefreshToken == "" {
		return "", nil
	}
	return r.tryOAuthRefreshAndPropagateRotation(ctx, src)
}

// tryOAuthRefreshAndPropagateRotation appelle TryOAuthRefreshWithRotation et
// propage le RT rotaté (mutation in-place de src + callback onRotated).
func (r *resolverImpl) tryOAuthRefreshAndPropagateRotation(ctx context.Context, src *CredentialSource) (string, error) {
	token, rotatedRT, err := r.provider.TryOAuthRefreshWithRotation(ctx, src.RefreshToken)
	if err != nil {
		slog.WarnContext(ctx, "pool/resolver: TryOAuthRefresh erreur",
			"gamertag", src.Gamertag, "err", err)
		return "", fmt.Errorf("pool/resolver: TryOAuthRefresh échoué pour %s: %w", src.Gamertag, err)
	}
	if token == "" {
		slog.WarnContext(ctx, "pool/resolver: TryOAuthRefresh retourné vide",
			"gamertag", src.Gamertag)
		return "", fmt.Errorf("pool/resolver: pas de token OAuth pour %s", src.Gamertag)
	}
	slog.InfoContext(ctx, "pool/resolver: OAuth v2 fallback OK",
		"gamertag", src.Gamertag,
		"rotated_rt_received", rotatedRT != "",
	)
	if rotatedRT != "" && rotatedRT != src.RefreshToken {
		src.RefreshToken = rotatedRT
		if r.onRotated != nil {
			if cbErr := r.onRotated(ctx, src.Gamertag, rotatedRT); cbErr != nil {
				slog.WarnContext(ctx, "pool/resolver: persistance RT rotaté échouée",
					"gamertag", src.Gamertag, "err", cbErr)
			}
		}
	}
	return token, nil
}

// exchangeAndCache appelle Exchange et insère le résultat dans le cache + sources.
func (r *resolverImpl) exchangeAndCache(ctx context.Context, src CredentialSource, accessToken string) (*ResolvedTokens, error) {
	start := time.Now()
	result, err := r.provider.Exchange(ctx, accessToken)
	if err != nil {
		slog.ErrorContext(ctx, "pool/resolver: Exchange échoué",
			"gamertag", src.Gamertag, "err", err)
		return nil, fmt.Errorf("pool/resolver: Exchange échoué pour %s: %w", src.Gamertag, err)
	}
	slog.InfoContext(ctx, "pool/resolver: Exchange OK",
		"gamertag", src.Gamertag, "duration_ms", time.Since(start).Milliseconds())

	resolved := &ResolvedTokens{
		Gamertag:  src.Gamertag,
		XUID:      src.XUID,
		Tokens:    result.Tokens,
		ExpiresAt: time.Now().Add(r.cacheTTL),
		Source:    src.Source,
	}

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

	// Sprint B1 commit 17 : event_id pour tracer un refresh forcé (souvent
	// déclenché par 401/403 — utile pour corréler la cause amont).
	ctx, evID := logging.WithEvent(ctx, "auth.refresh:"+gamertag)
	slog.InfoContext(ctx, "pool/resolver: force refresh du token",
		"gamertag", gamertag, "source", src.Source, "event", evID)

	// Invalider le cache.
	r.mu.Lock()
	delete(r.cache, gamertag)
	r.mu.Unlock()

	// Ré-échanger.
	return r.Resolve(ctx, src)
}
