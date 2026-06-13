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
	"strings"
	"time"
)

// Note PKCE : la génération du couple (code_verifier, code_challenge S256) est
// fournie par GeneratePKCE (sisu_client.go, partagée dans ce package). Le flux
// Authorization Code la consomme via le handler (verifier en session, challenge
// dans BuildAuthorizeURL, verifier renvoyé à ExchangeAuthorizationCode).

// AuthCodeResult regroupe ce que retourne ExchangeAuthorizationCode.
type AuthCodeResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

// ExchangeAuthorizationCode échange un Authorization Code OAuth v2 contre
// un access_token + refresh_token Microsoft.
//
// Client/secret résolus par ResolveAzureOAuthClient (source unique).
// Le redirectURI doit être strictement identique à celui passé dans la redirect
// vers /authorize (Microsoft vérifie ce match).
// codeVerifier : PKCE (RFC 7636) — vide = pas de PKCE (rétrocompat / device flow).
func ExchangeAuthorizationCode(ctx context.Context, code, redirectURI, codeVerifier string) (*AuthCodeResult, error) {
	if code == "" {
		return nil, fmt.Errorf("auth_code: code vide")
	}
	if redirectURI == "" {
		return nil, fmt.Errorf("auth_code: redirect_uri vide")
	}

	slog.DebugContext(ctx, "auth_code: échange code → tokens", "pkce", codeVerifier != "")

	azClient := ResolveAzureOAuthClient()

	body := url.Values{
		oauthFieldClientID: {azClient.ClientID},
		"grant_type":       {"authorization_code"},
		oauthFieldCode:     {code},
		"redirect_uri":     {redirectURI},
		oauthFieldScope:    {xboxScopes},
	}

	// PKCE : renvoyer le code_verifier qui matche le code_challenge de l'authorize.
	if codeVerifier != "" {
		body.Set("code_verifier", codeVerifier)
	}

	// App confidentielle : inclure client_secret si la garde public/confidentiel
	// l'autorise (cf. AzureOAuthClient.SecretToSend).
	if secret := azClient.SecretToSend(); secret != "" {
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
// codeChallenge : PKCE (RFC 7636, S256) — vide = pas de PKCE (rétrocompat).
func BuildAuthorizeURL(redirectURI, state, codeChallenge string) string {
	clientID := ResolveAzureOAuthClient().ClientID
	params := url.Values{
		oauthFieldClientID: {clientID},
		"response_type":    {"code"},
		"redirect_uri":     {redirectURI},
		oauthFieldScope:    {xboxScopes},
		"state":            {state},
		"response_mode":    {"query"},
	}
	if codeChallenge != "" {
		params.Set("code_challenge", codeChallenge)
		params.Set("code_challenge_method", "S256")
	}
	return fmt.Sprintf("%s/oauth2/v2.0/authorize?%s", MSALAuthority, params.Encode())
}
