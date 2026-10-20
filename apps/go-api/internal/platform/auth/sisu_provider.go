// Package auth — sisu_provider.go : TokenProvider du login Xbox natif (sans app Azure).
//
// SISUProvider implémente TokenProvider via :
//  1. RFC 8628 Device Code Flow sur login.live.com avec le client Xbox natif et le
//     scope MSA MBI_SSL (AcquireToken) ;
//  2. la chaîne XBL classique access_token → User Token (RpsTicket « t= ») → XSTS
//     du titre → Spartan → Clearance (ExchangeFlow).
//
// Le nom vient de l'échange par sisu.xboxlive.com/authorize, retiré : ce service
// répond 400 corps vide au ticket du device-flow, alors que la chaîne classique
// accepte ce même ticket — c'est elle qui rafraîchit les comptes de cette famille
// (TokenFamilyXboxNative). Aucune paire PoP ni device token n'est requis.
package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"levelup/go-api/internal/domain/title"
)

const (
	// SISUDefaultAppID est le client_id Xbox natif utilisé par l'app Xbox sur Windows.
	SISUDefaultAppID = "000000004c20a908"
	// SISUDefaultTitleID est le title_id de Halo Infinite.
	SISUDefaultTitleID = "144209987"
)

// sisuDeviceFlow implémente DeviceFlow pour le login Xbox natif : l'utilisateur
// saisit GetUserCode() sur GetVerificationURL(), AcquireToken attend sa validation,
// ExchangeFlow convertit le ticket obtenu en jetons Halo. Sans état partagé : deux
// flows concurrents n'interfèrent jamais.
type sisuDeviceFlow struct {
	verificationURL string // page Microsoft de saisie du code
	userCode        string // code RFC 8628 à saisir
	deviceCode      string // code opaque pour le polling
	interval        int    // intervalle de polling en secondes
	appID           string
	expiresIn       int

	// refreshToken est le refresh_token Microsoft rendu par le polling
	// (client Xbox natif, scope MSA). Écrit par AcquireToken, lu ensuite par
	// OAuthRefreshToken dans la même goroutine de polling — pas de concurrence.
	refreshToken string

	// httpClient porte les appels de la chaîne d'échange ; nil = client par défaut
	// (xblExchangeTimeout). Injecté par les tests.
	httpClient *http.Client
}

// ExchangeFlow convertit le ticket MSA du device-flow en jetons Halo par la chaîne
// XBL classique (descripteur d'auth Halo par défaut). La provenance du ticket est
// connue — client Xbox natif — et posée en ctx : le User Token est demandé d'emblée
// avec le préfixe RpsTicket « t= », sans 401 de tâtonnement. L'erreur est propagée
// (le caller la journalise et la porte sur la tentative). Honore auth.FlowExchanger.
func (f *sisuDeviceFlow) ExchangeFlow(ctx context.Context, accessToken string) (*ExchangeResult, error) {
	client := f.httpClient
	if client == nil {
		client = &http.Client{Timeout: xblExchangeTimeout}
	}
	ctx = WithTokenClientFamily(ctx, TokenFamilyXboxNative)
	res, err := exchangeAccessTokenWithClient(ctx, client, accessToken, title.DefaultHaloAuthDescriptor())
	if err != nil {
		return nil, fmt.Errorf("sisu_provider: échange XBL du ticket device-flow: %w", err)
	}
	slog.InfoContext(ctx, "sisu_provider: ExchangeFlow OK", "gamertag", res.Gamertag, "xuid", res.XUID)
	return res, nil
}

// Vérification compile-time : sisuDeviceFlow honore FlowExchanger.
var _ FlowExchanger = (*sisuDeviceFlow)(nil)

func (f *sisuDeviceFlow) GetMessage() string {
	return "Ouvrez l'URL ou entrez le code pour vous authentifier."
}
func (f *sisuDeviceFlow) GetUserCode() string        { return f.userCode }
func (f *sisuDeviceFlow) GetVerificationURL() string { return f.verificationURL }
func (f *sisuDeviceFlow) GetExpiresIn() int          { return f.expiresIn }
func (f *sisuDeviceFlow) GetFlowType() string        { return "sisu" }

// AcquireToken attend la validation Xbox Device Code et retourne l'access_token Microsoft.
// Conserve aussi le refresh_token pour la persistance ADR 0023 (cf. OAuthRefreshToken).
func (f *sisuDeviceFlow) AcquireToken(ctx context.Context) (string, error) {
	accessToken, refreshToken, err := PollXboxDeviceCode(ctx, f.appID, f.deviceCode, f.interval)
	if err != nil {
		return "", err
	}
	f.refreshToken = refreshToken
	return accessToken, nil
}

// OAuthRefreshToken expose le refresh_token Microsoft obtenu par le polling.
// Consommé par le handler device-flow pour la persistance dans watcher_tokens
// (ADR 0023) ; le pool le rafraîchit ensuite via le fallback MSA natif de
// ExchangeRefreshTokenWithRotation. Vide tant qu'AcquireToken n'a pas abouti.
func (f *sisuDeviceFlow) OAuthRefreshToken() string { return f.refreshToken }

// Vérification compile-time : sisuDeviceFlow implémente DeviceFlow.
var _ DeviceFlow = (*sisuDeviceFlow)(nil)

// SISUProvider implémente TokenProvider pour le login Xbox natif. Sans état de flow :
// chaque InitDeviceFlow retourne un sisuDeviceFlow autonome qui se complète via
// ExchangeFlow ; le pool auto-sync passe par Exchange (stateless).
type SISUProvider struct {
	appID string
	// titleID est conservé comme identité de configuration (MT-02 : injecté
	// depuis le descripteur du titre, loggé au boot). Il n'est pas envoyé sur le
	// fil : l'association au titre est portée par le client_id.
	titleID string
}

// NewSISUProvider crée un SISUProvider avec les IDs Xbox par défaut.
func NewSISUProvider() *SISUProvider {
	return &SISUProvider{
		appID:   SISUDefaultAppID,
		titleID: SISUDefaultTitleID,
	}
}

// NewSISUProviderWithIDs crée un SISUProvider avec des IDs configurables.
func NewSISUProviderWithIDs(appID, titleID string) *SISUProvider {
	return &SISUProvider{appID: appID, titleID: titleID}
}

// Vérification compile-time : SISUProvider implémente TokenProvider.
var _ TokenProvider = (*SISUProvider)(nil)

// InitDeviceFlow démarre un Device Code Flow Xbox natif sur login.live.com.
func (p *SISUProvider) InitDeviceFlow(ctx context.Context) (DeviceFlow, error) {
	return p.initDeviceFlowWithURL(ctx, xboxDeviceCodeURL)
}

// initDeviceFlowWithURL est la version testable avec l'endpoint de démarrage configurable.
func (p *SISUProvider) initDeviceFlowWithURL(ctx context.Context, deviceCodeURL string) (DeviceFlow, error) {
	slog.DebugContext(ctx, "sisu_provider: démarrage InitDeviceFlow")

	dcResult, err := startXboxDeviceCodeWithURL(ctx, nil, p.appID, deviceCodeURL)
	if err != nil {
		return nil, fmt.Errorf("sisu_provider: StartXboxDeviceCode: %w", err)
	}

	slog.InfoContext(ctx, "sisu_provider: Device Code Flow initié",
		"user_code", dcResult.UserCode,
	)

	return &sisuDeviceFlow{
		// Page Microsoft où l'utilisateur SAISIT le user_code affiché par l'UI
		// (typiquement https://www.microsoft.com/link).
		verificationURL: dcResult.VerificationURL,
		userCode:        dcResult.UserCode,
		deviceCode:      dcResult.DeviceCode,
		interval:        dcResult.Interval,
		appID:           p.appID,
		expiresIn:       dcResult.ExpiresIn,
	}, nil
}

// Exchange convertit un access_token Microsoft en tokens Halo Infinite via la chaîne
// XSTS standard (ExchangeAccessToken). TOUJOURS stateless : c'est le chemin du pool
// auto-sync / scheduler / SSO web, qui fournit un access_token déjà obtenu (OAuth
// refresh / auth code) et pose sa provenance en ctx. Le device-flow interactif passe
// par sisuDeviceFlow.ExchangeFlow, qui connaît la provenance de son ticket.
func (p *SISUProvider) Exchange(ctx context.Context, accessToken string) (*ExchangeResult, error) {
	slog.DebugContext(ctx, "sisu_provider: Exchange stateless — ExchangeAccessToken")
	return ExchangeAccessToken(ctx, accessToken)
}

// TryOAuthRefresh tente d'obtenir un access_token via OAuth v2 refresh_token.
//
// DEPRECATED : préférer TryOAuthRefreshWithRotation.
func (p *SISUProvider) TryOAuthRefresh(ctx context.Context, refreshToken string) (string, error) {
	token, _, err := p.TryOAuthRefreshWithRotation(ctx, refreshToken)
	return token, err
}

// TryOAuthRefreshWithRotation : voir interface TokenProvider.
// SISUProvider délègue au même endpoint login.microsoftonline.com.
func (p *SISUProvider) TryOAuthRefreshWithRotation(ctx context.Context, refreshToken string) (string, string, error) {
	if refreshToken == "" {
		slog.DebugContext(ctx, "sisu_provider: TryOAuthRefreshWithRotation ignoré (refresh_token vide)")
		return "", "", nil
	}
	slog.DebugContext(ctx, "sisu_provider: tentative OAuth v2 refresh + rotation")
	accessToken, rotatedRT, _, err := ExchangeRefreshTokenWithRotation(ctx, refreshToken)
	if err != nil {
		// Debug : l'erreur est propagée et loguée une seule fois par le caller
		// (pool/resolver) avec sa classe — cf. plan anti-bruit 2026-06-11.
		slog.DebugContext(ctx, "sisu_provider: TryOAuthRefreshWithRotation erreur", "err", err)
		return "", "", err
	}
	return accessToken, rotatedRT, nil
}
