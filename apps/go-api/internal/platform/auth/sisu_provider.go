// Package auth — sisu_provider.go : TokenProvider basé sur SISU/PoP (sans Azure app).
//
// SISUProvider implémente TokenProvider via :
//  1. RFC 8628 Device Code Flow sur login.live.com (AcquireToken)
//  2. Session SISU pour la récupération du XSTS Xbox Live (Exchange)
//
// Contrairement à MSALProvider, SISUProvider ne dépend pas de MSAL ni d'un app Azure enregistré.
// Il utilise le client_id Xbox officiel (same as the Xbox app).
package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"levelup/go-api/internal/domain"
)

const (
	// SISUDefaultAppID est le client_id Xbox natif utilisé par l'app Xbox sur Windows.
	SISUDefaultAppID = "000000004c20a908"
	// SISUDefaultTitleID est le title_id de Halo Infinite.
	SISUDefaultTitleID = "144209987"
)

// sisuFlowContext contient l'état éphémère créé par InitDeviceFlow et consommé par Exchange.
// Protégé par le mutex de SISUProvider.
type sisuFlowContext struct {
	kp           *PoPKeyPair
	deviceToken  string
	sessionID    string
	codeVerifier string
}

// sisuDeviceFlow implémente DeviceFlow pour le flow SISU.
// L'utilisateur visite GetVerificationURL() (MsaOauthRedirect de SISU) dans son navigateur,
// ou entre GetUserCode() sur login.live.com/devicelogin (identique au Device Code Flow Xbox).
type sisuDeviceFlow struct {
	verificationURL string // MsaOauthRedirect de SISU
	userCode        string // code RFC 8628 de l'Xbox Device Code Flow
	deviceCode      string // code opaque pour le polling
	interval        int    // intervalle de polling en secondes
	appID           string
	expiresIn       int
}

func (f *sisuDeviceFlow) GetMessage() string {
	return "Ouvrez l'URL ou entrez le code pour vous authentifier."
}
func (f *sisuDeviceFlow) GetUserCode() string        { return f.userCode }
func (f *sisuDeviceFlow) GetVerificationURL() string { return f.verificationURL }
func (f *sisuDeviceFlow) GetExpiresIn() int          { return f.expiresIn }
func (f *sisuDeviceFlow) GetFlowType() string        { return "sisu" }

// AcquireToken attend la validation Xbox Device Code et retourne l'access_token Microsoft.
func (f *sisuDeviceFlow) AcquireToken(ctx context.Context) (string, error) {
	accessToken, _, err := PollXboxDeviceCode(ctx, f.appID, f.deviceCode, f.interval)
	return accessToken, err
}

// Vérification compile-time : sisuDeviceFlow implémente DeviceFlow.
var _ DeviceFlow = (*sisuDeviceFlow)(nil)

// SISUProvider implémente TokenProvider via SISU/PoP natif Xbox.
type SISUProvider struct {
	appID   string
	titleID string
	mu      sync.Mutex
	current *sisuFlowContext // nil entre deux flows
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

// sisuProviderURLs regroupe les URLs des endpoints utilisés par InitDeviceFlow.
// Permet l'injection en test via initDeviceFlowWithURLs.
type sisuProviderURLs struct {
	deviceAuth string
	xboxDevice string
	sisuAuth   string
}

var defaultSISUProviderURLs = sisuProviderURLs{
	deviceAuth: deviceAuthURL,
	xboxDevice: xboxDeviceCodeURL,
	sisuAuth:   sisuAuthenticateURL,
}

// InitDeviceFlow démarre un Device Code Flow Xbox natif + initialise la session SISU.
// Stocke le sisuFlowContext (kp, deviceToken, sessionID, codeVerifier) pour Exchange.
func (p *SISUProvider) InitDeviceFlow(ctx context.Context) (DeviceFlow, error) {
	return p.initDeviceFlowWithURLs(ctx, defaultSISUProviderURLs)
}

// initDeviceFlowWithURLs est la version testable avec URLs configurables.
func (p *SISUProvider) initDeviceFlowWithURLs(ctx context.Context, urls sisuProviderURLs) (DeviceFlow, error) {
	slog.DebugContext(ctx, "sisu_provider: démarrage InitDeviceFlow")

	kp, err := GeneratePoPKeyPair()
	if err != nil {
		return nil, fmt.Errorf("sisu_provider: GeneratePoPKeyPair: %w", err)
	}

	deviceToken, err := requestDeviceTokenWithURL(ctx, nil, kp, urls.deviceAuth)
	if err != nil {
		return nil, fmt.Errorf("sisu_provider: RequestDeviceToken: %w", err)
	}

	dcResult, err := startXboxDeviceCodeWithURL(ctx, nil, p.appID, urls.xboxDevice)
	if err != nil {
		return nil, fmt.Errorf("sisu_provider: StartXboxDeviceCode: %w", err)
	}

	codeVerifier, codeChallenge, err := GeneratePKCE()
	if err != nil {
		return nil, fmt.Errorf("sisu_provider: GeneratePKCE: %w", err)
	}

	sisuSession, err := initSISUSessionWithURL(ctx, nil, kp, deviceToken, p.appID, p.titleID, codeChallenge, "sisu-state", urls.sisuAuth)
	if err != nil {
		return nil, fmt.Errorf("sisu_provider: InitSISUSession: %w", err)
	}

	p.mu.Lock()
	p.current = &sisuFlowContext{
		kp:           kp,
		deviceToken:  deviceToken,
		sessionID:    sisuSession.SessionID,
		codeVerifier: codeVerifier,
	}
	p.mu.Unlock()

	// Vérification URL : préférer MsaOauthRedirect (SISU) si disponible, sinon VerificationURI Xbox.
	verificationURL := sisuSession.MsaOauthRedirect
	if verificationURL == "" {
		verificationURL = dcResult.VerificationURL
	}

	slog.InfoContext(ctx, "sisu_provider: Device Code Flow + session SISU initiés",
		"user_code", dcResult.UserCode,
		"session_id", sisuSession.SessionID,
	)

	return &sisuDeviceFlow{
		verificationURL: verificationURL,
		userCode:        dcResult.UserCode,
		deviceCode:      dcResult.DeviceCode,
		interval:        dcResult.Interval,
		appID:           p.appID,
		expiresIn:       dcResult.ExpiresIn,
	}, nil
}

// Exchange complète le flow SISU après la validation OAuth.
// Utilise le sisuFlowContext posé par InitDeviceFlow, puis l'efface.
// Panics si appelé sans InitDeviceFlow préalable (bug d'utilisation).
func (p *SISUProvider) Exchange(ctx context.Context, accessToken string) (*ExchangeResult, error) {
	p.mu.Lock()
	flowCtx := p.current
	p.current = nil
	p.mu.Unlock()

	if flowCtx == nil {
		panic("sisu_provider: Exchange appelé sans InitDeviceFlow préalable — bug d'utilisation")
	}

	slog.DebugContext(ctx, "sisu_provider: Exchange — CompleteSISUFlow")
	xstsResult, err := CompleteSISUFlow(ctx, nil, flowCtx.kp, flowCtx.deviceToken, accessToken,
		p.appID, flowCtx.sessionID, flowCtx.codeVerifier)
	if err != nil {
		return nil, fmt.Errorf("sisu_provider: CompleteSISUFlow: %w", err)
	}

	// XSTS Xbox Live → Spartan Token + Clearance Token
	client := &http.Client{Timeout: 20 * time.Second}
	spartanToken, err := requestSpartanToken(ctx, client, xstsResult.Token)
	if err != nil {
		return nil, fmt.Errorf("sisu_provider: requestSpartanToken: %w", err)
	}

	clearanceToken, err := requestClearanceToken(ctx, client, spartanToken)
	if err != nil {
		return nil, fmt.Errorf("sisu_provider: requestClearanceToken: %w", err)
	}

	slog.InfoContext(ctx, "sisu_provider: Exchange OK",
		"gamertag", xstsResult.Gamertag,
		"xuid", xstsResult.XUID,
	)
	return &ExchangeResult{
		Tokens: &domain.HaloTokens{
			SpartanToken:   spartanToken,
			ClearanceToken: clearanceToken,
		},
		Gamertag: xstsResult.Gamertag,
		XUID:     xstsResult.XUID,
	}, nil
}

// TrySilentRefresh délègue à TryOAuthRefresh (pas de cache MSAL pour SISU).
func (p *SISUProvider) TrySilentRefresh(ctx context.Context, _ string) (string, error) {
	slog.DebugContext(ctx, "sisu_provider: TrySilentRefresh ignoré (pas de cache MSAL)")
	return "", nil
}

// TryOAuthRefresh tente d'obtenir un access_token via OAuth v2 refresh_token.
// Identique à MSALProvider — même endpoint login.live.com.
func (p *SISUProvider) TryOAuthRefresh(ctx context.Context, refreshToken string) (string, error) {
	if refreshToken == "" {
		slog.DebugContext(ctx, "sisu_provider: TryOAuthRefresh ignoré (refresh_token vide)")
		return "", nil
	}
	slog.DebugContext(ctx, "sisu_provider: tentative OAuth v2 refresh")
	token, err := ExchangeRefreshToken(ctx, refreshToken)
	if err != nil {
		slog.WarnContext(ctx, "sisu_provider: TryOAuthRefresh erreur", "err", err)
		return "", err
	}
	return token, nil
}
