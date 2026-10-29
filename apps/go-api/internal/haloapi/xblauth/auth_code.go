// auth_code.go : échange d'un Authorization Code OAuth v2 contre un access_token Microsoft
// (flow redirect SSO).
//
// Architecture du flow Authorization Code (RFC 6749 §4.1) :
//  1. Frontend redirect 302 vers l'adresse /authorize ([BuildAuthorizeURL]) avec
//     response_type=code + state CSRF stocké en session.
//  2. User s'authentifie chez Microsoft + autorise l'app LevelUp.
//  3. Microsoft redirect 302 vers redirect_uri?code=...&state=...
//  4. Backend (handler Callback) vérifie state vs session, puis échange code
//     -> access_token + refresh_token ([OAuth.ExchangeAuthorizationCode]).
//
// Différent du Device Code Flow (xbox_device_code.go, RFC 8628) qui demande à l'utilisateur de
// copier un code court vers une page Microsoft.

package xblauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

// GeneratePKCE génère un code_verifier et son code_challenge S256 (RFC 7636).
// Le flux Authorization Code les consomme via le handler : verifier en session,
// challenge dans BuildAuthorizeURL, verifier renvoyé à ExchangeAuthorizationCode.
// Retourne (codeVerifier, codeChallenge, error).
func GeneratePKCE() (string, string, error) {
	b := make([]byte, pkceVerifierBytes)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("auth_code: génération PKCE: %w", err)
	}
	verifier := base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])
	return verifier, challenge, nil
}

// pkceVerifierBytes est l'entropie du code_verifier PKCE (32 octets -> 43 caractères
// base64url, minimum de la RFC 7636 §4.1).
const pkceVerifierBytes = 32

// AuthCodeResult regroupe ce que retourne ExchangeAuthorizationCode.
type AuthCodeResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

// ExchangeAuthorizationCode échange un Authorization Code OAuth v2 contre un access_token +
// refresh_token Microsoft, auprès de `o.URLAzure` avec le client `o.Azure`.
//
// Le redirectURI doit être strictement identique à celui passé dans la redirect vers /authorize
// (Microsoft vérifie ce match). codeVerifier : PKCE (RFC 7636) — vide = pas de PKCE.
func (o OAuth) ExchangeAuthorizationCode(ctx context.Context, code, redirectURI, codeVerifier string) (*AuthCodeResult, error) {
	if code == "" {
		return nil, fmt.Errorf("auth_code: code vide")
	}
	if redirectURI == "" {
		return nil, fmt.Errorf("auth_code: redirect_uri vide")
	}

	slog.DebugContext(ctx, "auth_code: échange code → tokens", "pkce", codeVerifier != "")

	body := url.Values{
		oauthFieldClientID:  {o.Azure.ID},
		oauthFieldGrantType: {"authorization_code"},
		oauthFieldCode:      {code},
		"redirect_uri":      {redirectURI},
		oauthFieldScope:     {o.Azure.ScopesParam()},
	}

	// PKCE : renvoyer le code_verifier qui matche le code_challenge de l'authorize.
	if codeVerifier != "" {
		body.Set("code_verifier", codeVerifier)
	}

	// App confidentielle : inclure client_secret s'il est fourni.
	if secret := o.Azure.Secret; secret != "" {
		body.Set("client_secret", secret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.URLAzure,
		strings.NewReader(body.Encode()))
	if err != nil {
		return nil, fmt.Errorf("auth_code: construction requête: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.Doer.Do(req)
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

// BuildAuthorizeURL construit l'URL de redirect vers l'adresse /authorize du client `az` pour
// initier le Authorization Code Flow. Le state doit être généré aléatoirement (32+ bytes) et
// persisté en session pour vérification au callback. codeChallenge : PKCE (RFC 7636, S256) —
// vide = pas de PKCE.
func BuildAuthorizeURL(az ClientAzure, redirectURI, state, codeChallenge string) string {
	params := url.Values{
		oauthFieldClientID: {az.ID},
		"response_type":    {"code"},
		"redirect_uri":     {redirectURI},
		oauthFieldScope:    {az.ScopesParam()},
		"state":            {state},
		"response_mode":    {"query"},
	}
	if codeChallenge != "" {
		params.Set("code_challenge", codeChallenge)
		params.Set("code_challenge_method", "S256")
	}
	return fmt.Sprintf("%s?%s", az.URLAutorisation(), params.Encode())
}
