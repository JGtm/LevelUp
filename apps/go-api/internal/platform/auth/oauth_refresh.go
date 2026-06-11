// Package auth — oauth_refresh.go : échange d'un OAuth v2 refresh_token contre un access_token.
//
// Certains joueurs (ex : JGtm) n'ont pas de cache MSAL dans sync_meta.
// Leur refresh_token Microsoft est stocké soit dans .env.local
// (SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>), soit dans sync_meta sous la clé
// "oauth_refresh_token".
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
	"os"
	"strings"
	"time"

	"levelup/go-api/internal/observability/logging"
)

// msalTokenURL est une var (pas une const) pour permettre l'override httptest.
var msalTokenURL = "https://login.microsoftonline.com/consumers/oauth2/v2.0/token"

const xboxScopes = "Xboxlive.signin Xboxlive.offline_access"

// oauthTokenResponse est la réponse JSON du endpoint /oauth2/v2.0/token.
type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"` // token tourné — ignoré pour l'instant
	ExpiresIn    int    `json:"expires_in"`
	ErrorCode    string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// ExchangeRefreshToken échange un OAuth v2 refresh_token contre un access_token Microsoft.
// Utilise SPNKR_AZURE_CLIENT_ID si défini (tokens legacy), sinon LevelUpClientID.
// Retourne ("", nil) si le refresh_token est vide.
//
// Note : Microsoft rotate le refresh_token à chaque usage. Le token tourné est
// dans la réponse mais n'est PAS propagé par cette fonction — utiliser
// ExchangeRefreshTokenWithRotation si on veut le récupérer pour le persister.
func ExchangeRefreshToken(ctx context.Context, refreshToken string) (string, error) {
	accessToken, _, err := ExchangeRefreshTokenWithRotation(ctx, refreshToken)
	return accessToken, err
}

// ExchangeRefreshTokenWithRotation échange un refresh_token et retourne aussi
// le nouveau refresh_token retourné par Microsoft (qui rotate à chaque appel
// pour la sécurité). Le caller doit persister `rotatedRefreshToken` quelque
// part (sync_meta.oauth_refresh_token typiquement) sinon le prochain appel
// échouera avec un token révoqué.
//
// Retourne ("", "", nil) si refreshToken est vide.
// Retourne ("", "", err) si l'appel HTTP ou Microsoft retourne une erreur.
// Retourne (accessToken, "", nil) si Microsoft ne renvoie pas de rotation
// (rare — supporté par tolérance, mais l'appel suivant utilisera l'ancien RT).
func ExchangeRefreshTokenWithRotation(ctx context.Context, refreshToken string) (accessToken, rotatedRefreshToken string, err error) {
	if refreshToken == "" {
		return "", "", nil
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

	clientID := os.Getenv("SPNKR_AZURE_CLIENT_ID")
	if clientID == "" {
		clientID = LevelUpClientID
	}

	// App confidentielle supposée si SPNKR_AZURE_CLIENT_SECRET est défini :
	// 1re tentative avec le secret (les RT émis par le flux web confidentiel
	// en ont besoin). Si Azure répond AADSTS90023 (RT émis par un flux client
	// PUBLIC — ex. token-capture/device code — le secret est alors interdit),
	// retenter UNE fois sans secret. Stateless : 1 round-trip de plus par
	// refresh concerné (~toutes les 3h30), aucun état à mémoriser.
	secret := ""
	if s := os.Getenv("SPNKR_AZURE_CLIENT_SECRET"); s != "" && clientID != LevelUpClientID {
		secret = s
	}

	accessToken, rotatedRefreshToken, err = postTokenExchange(ctx, clientID, refreshToken, secret)
	if err != nil && secret != "" {
		var oerr *OAuthExchangeError
		if errors.As(err, &oerr) && oerr.IsSecretRejected() {
			oauthRefreshRetryPublicTotal.Add(1)
			slog.InfoContext(ctx, "oauth_refresh: client_secret refusé (AADSTS90023) — retry en flux client public")
			accessToken, rotatedRefreshToken, err = postTokenExchange(ctx, clientID, refreshToken, "")
		}
	}
	return accessToken, rotatedRefreshToken, err
}

// postTokenExchange exécute un POST /oauth2/v2.0/token et décode la réponse.
// secret vide = flux client public (pas de champ client_secret).
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, msalTokenURL,
		strings.NewReader(body.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("oauth_refresh: construction requête: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
			"error", tok.ErrorCode, "description", tok.ErrorDesc, "public_flow", secret == "")
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
