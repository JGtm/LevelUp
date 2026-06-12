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

const (
	msalTokenURL = "https://login.microsoftonline.com/consumers/oauth2/v2.0/token"
	xboxScopes   = "Xboxlive.signin Xboxlive.offline_access"
)

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
		if err != nil {
			slog.WarnContext(ctx, "oauth_refresh: échange échoué",
				"duration_ms", time.Since(start).Milliseconds(), "err", err)
		} else if accessToken == "" {
			slog.WarnContext(ctx, "oauth_refresh: token révoqué (réponse vide)",
				"duration_ms", time.Since(start).Milliseconds())
		} else {
			slog.InfoContext(ctx, "oauth_refresh: échange OK",
				"duration_ms", time.Since(start).Milliseconds(),
				"rotated", rotatedRefreshToken != "" && rotatedRefreshToken != refreshToken)
		}
	}()

	clientID := os.Getenv("SPNKR_AZURE_CLIENT_ID")
	if clientID == "" {
		clientID = LevelUpClientID
	}

	// Parc d'auth MIXTE : les RT émis par le flux web CONFIDENTIEL exigent le
	// client_secret (AADSTS70002 sans lui) ; ceux émis par un flux client PUBLIC
	// (token-capture/device code, halo-tools) le REJETTENT (AADSTS90023). On ne stocke
	// pas le client_id par token → 1re tentative AVEC le secret (si défini, app non-
	// LevelUp), et si Azure le refuse (AADSTS90023), on retente UNE fois SANS. Stateless,
	// ~1 round-trip de plus tous les ~3h30. (Même comportement que main/auth.)
	secret := ""
	if s := os.Getenv("SPNKR_AZURE_CLIENT_SECRET"); s != "" && clientID != LevelUpClientID {
		secret = s
	}

	accessToken, rotatedRefreshToken, err = postTokenExchange(ctx, clientID, refreshToken, secret)
	if err != nil && secret != "" && strings.Contains(err.Error(), "AADSTS90023") {
		slog.InfoContext(ctx, "oauth_refresh: client_secret refusé (AADSTS90023) — retry en flux client public")
		accessToken, rotatedRefreshToken, err = postTokenExchange(ctx, clientID, refreshToken, "")
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
		// DebugContext (pas Warn) : la 1re tentative qui échoue en AADSTS90023 est
		// attendue pour les comptes publics et suivie d'un retry — pas une vraie erreur.
		slog.DebugContext(ctx, "oauth_refresh: erreur serveur",
			"error", tok.ErrorCode, "description", tok.ErrorDesc, "public_flow", secret == "")
		return "", "", fmt.Errorf("oauth_refresh: %s — %s", tok.ErrorCode, tok.ErrorDesc)
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
