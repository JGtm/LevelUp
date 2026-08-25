// Package auth — oauth_refresh.go : échange d'un OAuth v2 refresh_token contre un access_token.
//
// Le refresh_token Microsoft de chaque joueur vit dans le MultiUserTokenStore
// (data/auth/watcher_tokens/{xuid}.json) — source unique depuis ADR 0023.
//
// Ce module implémente le leg OAuth v2 :
//
//	refresh_token → POST /oauth2/v2.0/token → access_token
//	                                        ↓
//	                              ExchangeAccessToken → Halo tokens
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/platform/netguard"
)

// msalTokenURL est une var (pas une const) pour permettre l'override httptest.
var msalTokenURL = "https://login.microsoftonline.com/consumers/oauth2/v2.0/token"

// msaTokenURL — endpoint OAuth 2.0 MSA legacy (login.live.com), celui du client
// Xbox natif du flow SISU. var (pas const) pour l'override httptest.
var msaTokenURL = "https://login.live.com/oauth20_token.srf"

// xboxScopes — scopes Xbox Live joints par espace (form bodies), dérivés du
// descripteur (MT-02, source unique avec XboxScopes []string → zéro dérive).
var xboxScopes = title.DefaultHaloAuthDescriptor().OAuthScopesParam()

// oauthTokenResponse est la réponse JSON du endpoint /oauth2/v2.0/token.
type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"` // token tourné — ignoré pour l'instant
	ExpiresIn    int    `json:"expires_in"`
	ErrorCode    string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// ExchangeRefreshToken échange un OAuth v2 refresh_token contre un access_token Microsoft.
// Client/secret résolus par ResolveAzureOAuthClient (source unique).
// Retourne ("", nil) si le refresh_token est vide.
//
// Note : Microsoft rotate le refresh_token à chaque usage. Le token tourné est
// dans la réponse mais n'est PAS propagé par cette fonction — utiliser
// ExchangeRefreshTokenWithRotation si on veut le récupérer pour le persister.
func ExchangeRefreshToken(ctx context.Context, refreshToken string) (string, error) {
	accessToken, _, _, err := ExchangeRefreshTokenWithRotation(ctx, refreshToken)
	return accessToken, err
}

// ExchangeRefreshTokenWithRotation échange un refresh_token et retourne aussi
// le nouveau refresh_token retourné par Microsoft (qui rotate à chaque appel
// pour la sécurité). Le caller doit persister `rotatedRefreshToken` quelque
// part (MultiUserTokenStore, ADR 0023) sinon le prochain appel
// échouera avec un token révoqué.
//
// Retourne aussi `clientFamily` (AU4/F12) : la famille de client OAuth qui a
// RÉELLEMENT répondu — TokenFamilyAzure si l'app Azure a rafraîchi, TokenFamilyXboxNative
// si le fallback MSA natif (client Xbox/SISU) a réussi, "" si aucun (échec). Le caller
// la persiste sur UserTokens pour un préfixe RpsTicket déterministe au prochain échange.
//
// Retourne ("", "", "", nil) si refreshToken est vide.
// Retourne ("", "", "", err) si l'appel HTTP ou Microsoft retourne une erreur.
// Retourne (accessToken, "", family, nil) si Microsoft ne renvoie pas de rotation
// (rare — supporté par tolérance, mais l'appel suivant utilisera l'ancien RT).
func ExchangeRefreshTokenWithRotation(ctx context.Context, refreshToken string) (accessToken, rotatedRefreshToken, clientFamily string, err error) {
	if refreshToken == "" {
		return "", "", "", nil
	}

	// Sprint B1 commit 18 : event_id pour tracer l'échange OAuth v2 refresh
	// → access_token Microsoft. Critique pour diag des refresh tokens révoqués.
	// Le ContextHandler injecte automatiquement event_id sur tous les sous-logs ;
	// on évite le log "démarré" redondant (la duration + l'event_id sont déjà
	// portés par le log de clôture "OK"/"échoué").
	ctx, _ = logging.WithEvent(ctx, "auth.oauth_exchange")
	start := time.Now()
	defer func() {
		// Échecs en Debug : l'erreur est propagée et loguée UNE fois au niveau
		// du caller (pool/resolver, avec sa classe) — cf. plan anti-bruit
		// 2026-06-11. Les compteurs expvar levelup.auth.* gardent la trace.
		if err != nil {
			slog.DebugContext(ctx, "oauth_refresh: échange échoué",
				"duration_ms", time.Since(start).Milliseconds(), "err", err)
		} else if accessToken == "" {
			slog.DebugContext(ctx, "oauth_refresh: token révoqué (réponse vide)",
				"duration_ms", time.Since(start).Milliseconds())
		} else {
			slog.InfoContext(ctx, "oauth_refresh: échange OK",
				"duration_ms", time.Since(start).Milliseconds(),
				"rotated", rotatedRefreshToken != "" && rotatedRefreshToken != refreshToken)
		}
	}()

	defer func() { recordOAuthRefreshOutcome(err) }()

	client := ResolveAzureOAuthClient()
	clientID := client.ClientID

	// 1re tentative avec le secret confidentiel si la garde public/confidentiel
	// l'autorise (cf. AzureOAuthClient.SecretToSend) : les RT émis par le flux web
	// confidentiel en ont besoin. Si Azure répond AADSTS90023 (RT émis par un flux
	// client PUBLIC — ex. token-capture/device code — le secret est alors interdit),
	// retenter UNE fois sans secret. Stateless : 1 round-trip de plus par refresh
	// concerné (~toutes les 3h30), aucun état à mémoriser.
	secret := client.SecretToSend()

	accessToken, rotatedRefreshToken, err = postTokenExchange(ctx, clientID, refreshToken, secret)
	if err != nil && secret != "" {
		var oerr *OAuthExchangeError
		if errors.As(err, &oerr) && oerr.IsSecretRejected() {
			oauthRefreshRetryPublicTotal.Add(1)
			slog.InfoContext(ctx, "oauth_refresh: client_secret refusé (AADSTS90023) — retry en flux client public")
			accessToken, rotatedRefreshToken, err = postTokenExchange(ctx, clientID, refreshToken, "")
		}
	}
	if err == nil && accessToken != "" {
		clientFamily = TokenFamilyAzure // l'app Azure a rafraîchi (AU4/F12) → préfixe "d=".
	}

	// Fallback MSA natif (flow SISU) : un RT émis par le client Xbox natif
	// (device-flow SISU, scope MBI_SSL) est inconnu de l'app Azure — le endpoint
	// v2 le refuse en invalid_grant (« RT étranger », type AADSTS70000). On
	// retente alors UNE fois sur login.live.com avec le client Xbox et le scope
	// MSA d'origine. Stateless, symétrique du retry AADSTS90023. Si le fallback
	// échoue aussi, on propage l'erreur Azure INITIALE (classification intacte
	// pour le pool/resolver) et on logge l'échec MSA avant dégradation.
	if err != nil {
		if at, rt, fam, ferr := tryMSANativeFallback(ctx, refreshToken, err); ferr == nil {
			accessToken, rotatedRefreshToken, clientFamily, err = at, rt, fam, nil
		}
	}
	return accessToken, rotatedRefreshToken, clientFamily, err
}

// tryMSANativeFallback retente un refresh sur login.live.com (client Xbox natif/SISU)
// quand Azure a refusé le RT en invalid_grant (« RT étranger » AADSTS70000). Retourne
// (accessToken, rotatedRT, TokenFamilyXboxNative, nil) si le fallback réussit ; sinon
// ("", "", "", azureErr) — l'erreur Azure INITIALE est propagée (classification revoked
// intacte pour le pool/resolver). Extrait d'ExchangeRefreshTokenWithRotation (gocyclo).
func tryMSANativeFallback(ctx context.Context, refreshToken string, azureErr error) (string, string, string, error) {
	var oerr *OAuthExchangeError
	if !errors.As(azureErr, &oerr) || oerr.Class() != AuthErrorRevoked {
		return "", "", "", azureErr
	}
	oauthRefreshRetryMSATotal.Add(1)
	slog.InfoContext(ctx, "oauth_refresh: invalid_grant côté Azure — retry refresh MSA natif (client Xbox/SISU)")
	at, rt, msaErr := postMSATokenExchange(ctx, refreshToken)
	if msaErr == nil && at != "" {
		return at, rt, TokenFamilyXboxNative, nil
	}
	slog.DebugContext(ctx, "oauth_refresh: retry MSA natif échoué", "err", msaErr)
	return "", "", "", azureErr
}

// postTokenExchange exécute un POST /oauth2/v2.0/token (endpoint Azure v2) et
// décode la réponse. secret vide = flux client public (pas de champ client_secret).
func postTokenExchange(ctx context.Context, clientID, refreshToken, secret string) (string, string, error) {
	body := url.Values{
		oauthFieldClientID:     {clientID},
		"grant_type":           {oauthFieldRefreshToken},
		oauthFieldRefreshToken: {refreshToken},
		oauthFieldScope:        {xboxScopes},
	}
	if secret != "" {
		body.Set("client_secret", secret)
	}
	return postOAuthTokenForm(ctx, msalTokenURL, body)
}

// postMSATokenExchange rafraîchit un refresh_token MSA natif (émis par le
// device-flow SISU : client Xbox, scope MBI_SSL) sur login.live.com. Flux
// client public — jamais de secret. Même contrat de retour que postTokenExchange.
func postMSATokenExchange(ctx context.Context, refreshToken string) (string, string, error) {
	body := url.Values{
		oauthFieldClientID:     {SISUDefaultAppID},
		"grant_type":           {oauthFieldRefreshToken},
		oauthFieldRefreshToken: {refreshToken},
		oauthFieldScope:        {sisuMSAScope},
	}
	return postOAuthTokenForm(ctx, msaTokenURL, body)
}

// postOAuthTokenForm exécute un POST form-encoded sur un endpoint token OAuth 2.0
// et décode la réponse (partagé Azure v2 / MSA legacy — même contrat JSON).
func postOAuthTokenForm(ctx context.Context, tokenURL string, body url.Values) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL,
		strings.NewReader(body.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("oauth_refresh: construction requête: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Mode démo : aucune sortie tierce. Coupe le rafraîchissement OAuth Microsoft
	// à la racine — sans quoi une démo lancée sur un poste porteur de vrais
	// refresh tokens obtient des tokens VALIDES et interroge l'API Halo pour les
	// xuid factices de la fixture (cf. internal/platform/netguard).
	if gErr := netguard.Check(req.Context(), "microsoft_oauth.refresh"); gErr != nil {
		return "", "", gErr
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("oauth_refresh: appel token endpoint: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("oauth_refresh: lecture réponse: %w", err)
	}

	var tok oauthTokenResponse
	if err := json.Unmarshal(raw, &tok); err != nil {
		return "", "", fmt.Errorf("oauth_refresh: décodage JSON: %w", err)
	}

	if tok.ErrorCode != "" {
		slog.DebugContext(ctx, "oauth_refresh: erreur serveur",
			"error", tok.ErrorCode, "description", tok.ErrorDesc,
			"public_flow", body.Get("client_secret") == "")
		return "", "", &OAuthExchangeError{ErrorCode: tok.ErrorCode, Description: tok.ErrorDesc}
	}

	if tok.AccessToken == "" {
		slog.DebugContext(ctx, "oauth_refresh: access_token vide dans la réponse")
		return "", "", nil
	}

	slog.DebugContext(ctx, "oauth_refresh: access_token obtenu",
		"expires_in", tok.ExpiresIn,
		"rotated_refresh_token", tok.RefreshToken != "",
	)
	return tok.AccessToken, tok.RefreshToken, nil
}
