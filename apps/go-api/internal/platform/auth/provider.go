// Package auth — provider.go : interface TokenProvider et implémentation MSAL.
//
// TokenProvider abstrait l'acquisition de tokens Halo Infinite.
// L'appelant ne connaît pas le mécanisme utilisé (MSAL Device Code Flow,
// silent refresh depuis DuckDB, ou futur legacy OAuth2 refresh token).
package auth

import (
	"context"
	"log/slog"
)

// TokenProvider abstrait l'acquisition de tokens Halo pour les appels API.
type TokenProvider interface {
	// InitDeviceFlow démarre un Device Code Flow interactif.
	// Le *DeviceCodeFlow retourné expose UserCode, VerificationURL, ExpiresIn
	// et la méthode AcquireToken(ctx) — bloquante.
	InitDeviceFlow(ctx context.Context) (*DeviceCodeFlow, error)

	// TrySilentRefresh tente un refresh non-interactif depuis un cache persisté.
	// cacheJSON : cache sérialisé (JSON) stocké dans sync_meta DuckDB ; "" si absent.
	// Retourne ("", nil) si aucun refresh n'est possible.
	TrySilentRefresh(ctx context.Context, cacheJSON string) (string, error)

	// TryOAuthRefresh tente d'obtenir un access_token via OAuth v2 refresh_token.
	// refreshToken : valeur brute du refresh token (env var ou sync_meta DuckDB).
	// Retourne ("", nil) si le token est vide ou si le refresh échoue proprement.
	TryOAuthRefresh(ctx context.Context, refreshToken string) (string, error)

	// Exchange convertit un access_token Microsoft en tokens Halo Infinite.
	Exchange(ctx context.Context, accessToken string) (*ExchangeResult, error)
}

// MSALProvider implémente TokenProvider via MSAL Device Code Flow.
type MSALProvider struct{}

// NewMSALProvider crée un MSALProvider.
func NewMSALProvider() *MSALProvider { return &MSALProvider{} }

// InitDeviceFlow démarre un Device Code Flow Microsoft.
// Crée un cache en mémoire vide pour stocker le résultat MSAL après complétion.
func (p *MSALProvider) InitDeviceFlow(ctx context.Context) (*DeviceCodeFlow, error) {
	slog.DebugContext(ctx, "provider: démarrage Device Code Flow")
	flow, err := InitDeviceFlow(ctx, &InMemoryCacheAccessor{})
	if err != nil {
		slog.ErrorContext(ctx, "provider: échec InitDeviceFlow", "err", err)
		return nil, err
	}
	slog.DebugContext(ctx, "provider: Device Code Flow prêt", "user_code", flow.UserCode)
	return flow, nil
}

// TrySilentRefresh tente un refresh silencieux depuis le cache MSAL (JSON DuckDB).
func (p *MSALProvider) TrySilentRefresh(ctx context.Context, cacheJSON string) (string, error) {
	if cacheJSON == "" {
		slog.DebugContext(ctx, "provider: TrySilentRefresh ignoré (cache vide)")
		return "", nil
	}
	slog.DebugContext(ctx, "provider: tentative silent refresh")
	accessor := NewInMemoryCacheAccessorFromJSON(cacheJSON)
	token, err := AcquireTokenSilent(ctx, accessor)
	if err != nil {
		slog.WarnContext(ctx, "provider: TrySilentRefresh erreur", "err", err)
		return "", err
	}
	if token == "" {
		slog.DebugContext(ctx, "provider: silent refresh impossible (cache invalide ou expiré)")
	} else {
		slog.DebugContext(ctx, "provider: silent refresh OK")
	}
	return token, nil
}

// Exchange échange un access_token Microsoft contre des tokens Halo Infinite.
func (p *MSALProvider) Exchange(ctx context.Context, accessToken string) (*ExchangeResult, error) {
	slog.DebugContext(ctx, "provider: échange access_token → tokens Halo")
	result, err := ExchangeAccessToken(ctx, accessToken)
	if err != nil {
		slog.ErrorContext(ctx, "provider: échec Exchange", "err", err)
		return nil, err
	}
	if result.Gamertag != "" || result.XUID != "" {
		slog.InfoContext(ctx, "provider: Exchange OK", "gamertag", result.Gamertag, "xuid", result.XUID)
	} else {
		// Le XSTS Halo (audience prod.xsts.halowaypoint.com) ne retourne pas gtg/xid.
		slog.DebugContext(ctx, "provider: Exchange OK")
	}
	return result, nil
}

// TryOAuthRefresh tente d'obtenir un access_token via OAuth v2 refresh_token.
func (p *MSALProvider) TryOAuthRefresh(ctx context.Context, refreshToken string) (string, error) {
	if refreshToken == "" {
		slog.DebugContext(ctx, "provider: TryOAuthRefresh ignoré (refresh_token vide)")
		return "", nil
	}
	slog.DebugContext(ctx, "provider: tentative OAuth v2 refresh")
	token, err := ExchangeRefreshToken(ctx, refreshToken)
	if err != nil {
		slog.WarnContext(ctx, "provider: TryOAuthRefresh erreur", "err", err)
		return "", err
	}
	if token == "" {
		slog.DebugContext(ctx, "provider: OAuth v2 refresh impossible (token révoqué ou expiré)")
	} else {
		slog.DebugContext(ctx, "provider: OAuth v2 refresh OK")
	}
	return token, nil
}
