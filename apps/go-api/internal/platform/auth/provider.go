// Package auth — provider.go : interface TokenProvider, DeviceFlow, et implémentations.
//
// TokenProvider abstrait l'acquisition de tokens Halo Infinite.
// L'appelant ne connaît pas le mécanisme utilisé (MSAL Device Code Flow,
// silent refresh depuis DuckDB, ou SISU/PoP natif Xbox).
package auth

import (
	"context"
	"log/slog"
)

// DeviceFlow abstrait un flow interactif d'authentification (MSAL ou SISU).
// Chaque provider retourne sa propre implémentation privée.
type DeviceFlow interface {
	// Accesseurs pour l'UI / sérialisation HTTP.
	GetMessage() string
	GetUserCode() string        // vide si SISU
	GetVerificationURL() string // microsoft.com/devicelogin (MSAL) ou URL OAuth directe (SISU)
	GetExpiresIn() int
	GetFlowType() string // "msal" | "sisu"

	// AcquireToken bloque jusqu'à l'authentification et retourne l'access_token Microsoft.
	// Bloquant — à appeler dans une goroutine.
	AcquireToken(ctx context.Context) (string, error)
}

// FlowExchanger est un DeviceFlow qui complète LUI-MÊME l'échange en tokens Halo,
// en portant son propre contexte éphémère (flow SISU : kp/deviceToken/sessionID/
// codeVerifier). Il n'existe alors AUCUN slot partagé sur le provider — deux
// onboardings concurrents ou un refresh stateless du pool ne peuvent plus se
// consommer/écraser mutuellement. Un flow qui n'implémente pas FlowExchanger
// (MSAL, stub) retombe sur TokenProvider.Exchange (échange stateless standard).
type FlowExchanger interface {
	ExchangeFlow(ctx context.Context, accessToken string) (*ExchangeResult, error)
}

// stubDeviceFlow implémente DeviceFlow pour les tests unitaires.
type stubDeviceFlow struct {
	message         string
	userCode        string
	verificationURL string
	expiresIn       int
	flowType        string
}

func (f *stubDeviceFlow) GetMessage() string         { return f.message }
func (f *stubDeviceFlow) GetUserCode() string        { return f.userCode }
func (f *stubDeviceFlow) GetVerificationURL() string { return f.verificationURL }
func (f *stubDeviceFlow) GetExpiresIn() int          { return f.expiresIn }
func (f *stubDeviceFlow) GetFlowType() string        { return f.flowType }
func (f *stubDeviceFlow) AcquireToken(_ context.Context) (string, error) {
	// Stub : retourne immédiatement (utilisé uniquement dans les tests où AcquireToken n'est pas exercé).
	return "", nil
}

// NewStubDeviceFlow crée un DeviceFlow de test sans dépendance MSAL/réseau.
// flowType : "msal" | "sisu".
func NewStubDeviceFlow(userCode, verificationURL, message string, expiresIn int, flowType string) DeviceFlow {
	if flowType == "" {
		flowType = "msal"
	}
	return &stubDeviceFlow{
		message:         message,
		userCode:        userCode,
		verificationURL: verificationURL,
		expiresIn:       expiresIn,
		flowType:        flowType,
	}
}

// TokenProvider abstrait l'acquisition de tokens Halo pour les appels API.
type TokenProvider interface {
	// InitDeviceFlow démarre un Device Code Flow interactif.
	// Le DeviceFlow retourné expose les accesseurs UI et AcquireToken.
	InitDeviceFlow(ctx context.Context) (DeviceFlow, error)

	// TrySilentRefresh tente un refresh non-interactif depuis un cache persisté.
	// cacheJSON : cache sérialisé (JSON) stocké dans sync_meta DuckDB ; "" si absent.
	// Retourne ("", nil) si aucun refresh n'est possible.
	TrySilentRefresh(ctx context.Context, cacheJSON string) (string, error)

	// TryOAuthRefresh tente d'obtenir un access_token via OAuth v2 refresh_token.
	// refreshToken : valeur brute du refresh token (env var ou sync_meta DuckDB).
	// Retourne ("", nil) si le token est vide ou si le refresh échoue proprement.
	//
	// DEPRECATED : préférer TryOAuthRefreshWithRotation qui expose le RT
	// rotaté par Microsoft (sans rotation persistée, le RT initial devient
	// invalide après le 1er usage). Conservé pour compat sur les call sites
	// qui n'ont pas besoin de la rotation.
	TryOAuthRefresh(ctx context.Context, refreshToken string) (string, error)

	// TryOAuthRefreshWithRotation tente d'obtenir un access_token et retourne
	// AUSSI le nouveau refresh_token retourné par Microsoft. Microsoft rotate
	// systématiquement le refresh_token à chaque usage pour des raisons de
	// sécurité. Sans persister le rotatedRT, le prochain refresh échouera
	// avec invalid_grant.
	//
	// Retourne ("", "", nil) si refreshToken est vide.
	// Retourne ("", "", err) si l'appel HTTP / Microsoft retourne une erreur.
	// Retourne (accessToken, rotatedRT, nil) en cas de succès ; rotatedRT peut
	// être "" dans de rares cas où Microsoft ne renvoie pas de rotation.
	TryOAuthRefreshWithRotation(ctx context.Context, refreshToken string) (accessToken, rotatedRefreshToken string, err error)

	// Exchange convertit un access_token Microsoft en tokens Halo Infinite.
	Exchange(ctx context.Context, accessToken string) (*ExchangeResult, error)
}

// MSALProvider implémente TokenProvider via MSAL Device Code Flow.
type MSALProvider struct{}

// NewMSALProvider crée un MSALProvider.
func NewMSALProvider() *MSALProvider { return &MSALProvider{} }

// Vérification compile-time : MSALProvider implémente TokenProvider.
var _ TokenProvider = (*MSALProvider)(nil)

// InitDeviceFlow démarre un Device Code Flow Microsoft.
// Crée un cache en mémoire vide pour stocker le résultat MSAL après complétion.
func (p *MSALProvider) InitDeviceFlow(ctx context.Context) (DeviceFlow, error) {
	slog.DebugContext(ctx, "provider: démarrage Device Code Flow")
	flow, err := InitDeviceFlow(ctx, &InMemoryCacheAccessor{})
	if err != nil {
		slog.ErrorContext(ctx, "provider: échec InitDeviceFlow", "err", err)
		return nil, err
	}
	slog.DebugContext(ctx, "provider: Device Code Flow prêt", "user_code", flow.GetUserCode())
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
//
// DEPRECATED : préférer TryOAuthRefreshWithRotation qui expose le RT rotaté.
// Conservé pour compat sur les call sites qui n'ont pas besoin du rotated RT.
func (p *MSALProvider) TryOAuthRefresh(ctx context.Context, refreshToken string) (string, error) {
	token, _, err := p.TryOAuthRefreshWithRotation(ctx, refreshToken)
	return token, err
}

// TryOAuthRefreshWithRotation : voir docstring sur l'interface TokenProvider.
// MSALProvider délègue à ExchangeRefreshTokenWithRotation (login.microsoftonline.com).
func (p *MSALProvider) TryOAuthRefreshWithRotation(ctx context.Context, refreshToken string) (string, string, error) {
	if refreshToken == "" {
		slog.DebugContext(ctx, "provider: TryOAuthRefreshWithRotation ignoré (refresh_token vide)")
		return "", "", nil
	}
	slog.DebugContext(ctx, "provider: tentative OAuth v2 refresh + rotation")
	accessToken, rotatedRT, err := ExchangeRefreshTokenWithRotation(ctx, refreshToken)
	if err != nil {
		// Debug : l'erreur est propagée et loguée une seule fois par le caller
		// (pool/resolver) avec sa classe — cf. plan anti-bruit 2026-06-11.
		slog.DebugContext(ctx, "provider: TryOAuthRefreshWithRotation erreur", "err", err)
		return "", "", err
	}
	if accessToken == "" {
		slog.DebugContext(ctx, "provider: OAuth v2 refresh impossible (token révoqué ou expiré)")
	} else {
		slog.DebugContext(ctx, "provider: OAuth v2 refresh OK",
			"rotated_rt_received", rotatedRT != "",
		)
	}
	return accessToken, rotatedRT, nil
}
