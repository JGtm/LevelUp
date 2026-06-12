// Package auth — auth_code.go : échange d'un Authorization Code OAuth v2 contre
// un access_token Microsoft (PR 4 — flow redirect SSO).
//
// Architecture du flow Authorization Code (RFC 6749 §4.1) :
//  1. Frontend redirect 302 vers https://login.microsoftonline.com/.../authorize
//     avec response_type=code + state CSRF stocké en session.
//  2. User s'authentifie chez Microsoft + autorise l'app LevelUp.
//  3. Microsoft redirect 302 vers redirect_uri?code=...&state=...
//  4. Backend (handler Callback) vérifie state vs session, puis exchange code
//     → access_token + refresh_token via ce module.
//
// Différent du Device Code Flow (msal_client.go, RFC 8628) qui demande à l'user
// de copier un code 9 caractères vers login.live.com/devicelogin.
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

// AuthCodeResult regroupe ce que retourne ExchangeAuthorizationCode.
type AuthCodeResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

// ExchangeAuthorizationCode échange un Authorization Code OAuth v2 contre
// un access_token + refresh_token Microsoft.
//
// Utilise SPNKR_AZURE_CLIENT_ID si défini (tokens legacy), sinon LevelUpClientID.
// Le redirectURI doit être strictement identique à celui passé dans la redirect
// vers /authorize (Microsoft vérifie ce match).
func ExchangeAuthorizationCode(ctx context.Context, code, redirectURI string) (*AuthCodeResult, error) {
	if code == "" {
		return nil, fmt.Errorf("auth_code: code vide")
	}
	if redirectURI == "" {
		return nil, fmt.Errorf("auth_code: redirect_uri vide")
	}

	slog.DebugContext(ctx, "auth_code: échange code → tokens")

	clientID := os.Getenv("SPNKR_AZURE_CLIENT_ID")
	if clientID == "" {
		clientID = LevelUpClientID
	}

	body := url.Values{
		oauthFieldClientID: {clientID},
		"grant_type":       {"authorization_code"},
		oauthFieldCode:     {code},
		"redirect_uri":     {redirectURI},
		oauthFieldScope:    {xboxScopes},
	}

	// App confidentielle : inclure client_secret si défini ET client_id non-public.
	// Les clients PUBLICS connus (LevelUp, halo-tools) ne doivent jamais recevoir de
	// secret (AADSTS90023). Cf. IsPublicAzureClient (source unique avec oauth_refresh.go).
	if secret := os.Getenv("SPNKR_AZURE_CLIENT_SECRET"); secret != "" && !IsPublicAzureClient(clientID) {
		body.Set("client_secret", secret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, msalTokenURL,
		strings.NewReader(body.Encode()))
	if err != nil {
		return nil, fmt.Errorf("auth_code: construction requête: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("auth_code: appel token endpoint: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("auth_code: lecture réponse: %w", err)
	}

	var tokenResp oauthTokenResponse
	if err := json.Unmarshal(data, &tokenResp); err != nil {
		return nil, fmt.Errorf("auth_code: décodage réponse: %w", err)
	}

	if tokenResp.ErrorCode != "" {
		return nil, fmt.Errorf("auth_code: Microsoft retourne erreur %s: %s",
			tokenResp.ErrorCode, tokenResp.ErrorDesc)
	}
	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("auth_code: access_token vide dans la réponse")
	}

	expiresIn := tokenResp.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 3600 // fallback conservateur ~1h
	}

	slog.DebugContext(ctx, "auth_code: échange réussi",
		"has_refresh_token", tokenResp.RefreshToken != "",
		"expires_in", expiresIn,
	)
	return &AuthCodeResult{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

// BuildAuthorizeURL construit l'URL de redirect vers Microsoft /authorize pour
// initier le Authorization Code Flow. Le state doit être généré aléatoirement
// (32+ bytes) et persisté en session pour vérification au callback.
func BuildAuthorizeURL(redirectURI, state string) string {
	clientID := os.Getenv("SPNKR_AZURE_CLIENT_ID")
	if clientID == "" {
		clientID = LevelUpClientID
	}
	params := url.Values{
		oauthFieldClientID: {clientID},
		"response_type":    {"code"},
		"redirect_uri":     {redirectURI},
		oauthFieldScope:    {xboxScopes},
		"state":            {state},
		"response_mode":    {"query"},
	}
	return fmt.Sprintf("%s/oauth2/v2.0/authorize?%s", MSALAuthority, params.Encode())
}
