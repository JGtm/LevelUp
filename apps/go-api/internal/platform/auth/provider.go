// Package auth — provider.go : interface TokenProvider, DeviceFlow, et implémentations.
//
// TokenProvider abstrait l'acquisition de tokens Halo Infinite.
// Depuis le retrait de MSAL (2026-07-15, SISU validé bout-en-bout), le seul
// provider réel est SISUProvider (SISU/PoP natif Xbox, zéro app Azure) ; le
// stub reste pour les tests.
package auth

import (
	"context"
)

// DeviceFlow abstrait un flow interactif d'authentification.
// Chaque provider retourne sa propre implémentation privée.
type DeviceFlow interface {
	// Accesseurs pour l'UI / sérialisation HTTP.
	GetMessage() string
	GetUserCode() string
	GetVerificationURL() string // page Microsoft de saisie du user_code
	GetExpiresIn() int
	GetFlowType() string // "sisu" (stubs de test : valeur libre)

	// AcquireToken bloque jusqu'à l'authentification et retourne l'access_token Microsoft.
	// Bloquant — à appeler dans une goroutine.
	AcquireToken(ctx context.Context) (string, error)
}

// FlowExchanger est un DeviceFlow qui complète LUI-MÊME l'échange en tokens Halo,
// en portant son propre contexte éphémère (flow SISU : kp/deviceToken). Il
// n'existe alors AUCUN slot partagé sur le provider — deux onboardings
// concurrents ou un refresh stateless du pool ne peuvent plus se
// consommer/écraser mutuellement. Un flow qui n'implémente pas FlowExchanger
// (stub) retombe sur TokenProvider.Exchange (échange stateless standard).
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

// NewStubDeviceFlow crée un DeviceFlow de test sans dépendance réseau.
func NewStubDeviceFlow(userCode, verificationURL, message string, expiresIn int, flowType string) DeviceFlow {
	if flowType == "" {
		flowType = "stub"
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
	// VOIE MORTE depuis le retrait de MSAL (2026-07-15) : SISUProvider répond
	// toujours ("", nil) et les callers (resolver/pool/cli_refresh) tombent sur
	// la voie refresh_token. Retrait de la méthode + des call sites + des champs
	// MSALCacheJSON planifié avec le lot D2 de purge legacy (armable ≥ 2026-07-20,
	// critère : telemetrie legacy_source_used muette). Ne pas réimplémenter.
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
