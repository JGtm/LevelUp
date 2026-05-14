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

	slog.DebugContext(ctx, "oauth_refresh: échange refresh_token → access_token")

	clientID := os.Getenv("SPNKR_AZURE_CLIENT_ID")
	if clientID == "" {
		clientID = LevelUpClientID
	}

	body := url.Values{
		"client_id":     {clientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"scope":         {xboxScopes},
	}

	// Si l'app Azure est confidentielle (SPNKR_AZURE_CLIENT_SECRET défini),
	// inclure le client_secret dans la requête.
	if secret := os.Getenv("SPNKR_AZURE_CLIENT_SECRET"); secret != "" && clientID != LevelUpClientID {
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
		slog.WarnContext(ctx, "oauth_refresh: erreur serveur", "error", tok.ErrorCode, "description", tok.ErrorDesc)
		return "", "", fmt.Errorf("oauth_refresh: %s — %s", tok.ErrorCode, tok.ErrorDesc)
	}

	if tok.AccessToken == "" {
		slog.WarnContext(ctx, "oauth_refresh: access_token vide dans la réponse")
		return "", "", nil
	}

	slog.DebugContext(ctx, "oauth_refresh: access_token obtenu",
		"expires_in", tok.ExpiresIn,
		"rotated_refresh_token", tok.RefreshToken != "",
	)
	return tok.AccessToken, tok.RefreshToken, nil
}
