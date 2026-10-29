// Package auth — xbl_oauth.go : façade d'exploitation des échanges OAuth v2 Microsoft
// (rafraîchissement d'un refresh_token, code d'autorisation du SSO web). Le protocole vit dans
// `internal/haloapi/xblauth` ; la façade y ajoute le client Azure résolu depuis
// l'environnement (azure_credentials.go), les adresses, le client HTTP et son délai, la garde
// `netguard` du rafraîchissement, l'identifiant d'événement du journal et les compteurs expvar
// (metrics.go) tenus à partir de l'issue typée que rend la brique.
//
// Le refresh_token de chaque joueur vit dans le MultiUserTokenStore
// (data/auth/watcher_tokens/{xuid}.json) — source unique depuis ADR 0023 ; la brique rend le
// jeton tourné, l'appelant le persiste.
package auth

import (
	"context"
	"net/http"
	"time"

	"levelup/go-api/internal/haloapi"
	"levelup/go-api/internal/haloapi/xblauth"
	"levelup/go-api/internal/observability/logging"
)

// msalTokenURL : endpoint /token de l'app Azure. var (pas const) pour l'override httptest.
var msalTokenURL = "https://login.microsoftonline.com/consumers/oauth2/v2.0/token"

// msaTokenURL : endpoint /token MSA du client Xbox natif (repli du rafraîchissement). var (pas
// const) pour l'override httptest.
var msaTokenURL = xblauth.URLTokenMSA

// oauthTimeout borne chaque appel aux endpoints /token (rafraîchissement, code d'autorisation).
const oauthTimeout = 15 * time.Second

// Classes d'échec d'un échange OAuth et erreur typée de l'endpoint /token (définies par la
// brique, re-exportées sous leurs noms historiques).
type (
	// AuthErrorClass est la classe d'un échec d'authentification OAuth.
	AuthErrorClass = xblauth.AuthErrorClass
	// OAuthExchangeError est l'erreur structurée renvoyée par l'endpoint /oauth2/v2.0/token.
	OAuthExchangeError = xblauth.OAuthExchangeError
	// AuthCodeResult regroupe ce que retourne ExchangeAuthorizationCode.
	AuthCodeResult = xblauth.AuthCodeResult
)

const (
	// AuthErrorConfig : requête refusée de façon déterministe (app/client mal configuré).
	AuthErrorConfig = xblauth.AuthErrorConfig
	// AuthErrorRevoked : refresh_token mort — ré-authentification interactive requise.
	AuthErrorRevoked = xblauth.AuthErrorRevoked
	// AuthErrorTransient : incident réseau/serveur — un retry ultérieur peut réussir.
	AuthErrorTransient = xblauth.AuthErrorTransient
)

// ClassifyAuthError extrait la classe d'une erreur quelconque de la chaîne d'authentification.
// Toute erreur non typée (réseau, JSON, timeout) est considérée transitoire.
func ClassifyAuthError(err error) AuthErrorClass { return xblauth.ClassifyAuthError(err) }

// oauthClient assemble ce que les échanges OAuth de la brique reçoivent : le client Azure résolu
// (environnement), les adresses /token et le Doer.
func oauthClient(doer haloapi.Doer) xblauth.OAuth {
	return xblauth.OAuth{
		Doer:     doer,
		Azure:    ResolveAzureOAuthClient().versXBL(),
		URLAzure: msalTokenURL,
		URLNatif: msaTokenURL,
	}
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
// Retourne ("", "", "", nil) si refreshToken est vide (aucun appel, aucun compteur).
// Retourne ("", "", "", err) si l'appel HTTP ou Microsoft retourne une erreur.
// Retourne (accessToken, "", family, nil) si Microsoft ne renvoie pas de rotation
// (rare — supporté par tolérance, mais l'appel suivant utilisera l'ancien RT).
func ExchangeRefreshTokenWithRotation(ctx context.Context, refreshToken string) (accessToken, rotatedRefreshToken, clientFamily string, err error) {
	if refreshToken == "" {
		return "", "", "", nil
	}

	// Sprint B1 commit 18 : event_id pour tracer l'échange OAuth v2 refresh
	// → access_token Microsoft. Critique pour diag des refresh tokens révoqués.
	// Le ContextHandler injecte automatiquement event_id sur tous les sous-logs,
	// y compris ceux de la brique.
	ctx, _ = logging.WithEvent(ctx, "auth.oauth_exchange")

	doer := netguardDoer{surface: "microsoft_oauth.refresh", next: &http.Client{Timeout: oauthTimeout}}
	out, err := oauthClient(doer).RefreshWithRotation(ctx, refreshToken)
	recordOAuthRefreshIssue(out, err)
	return out.AccessToken, out.RotatedRefreshToken, out.Famille, err
}

// ExchangeAuthorizationCode échange un Authorization Code OAuth v2 contre
// un access_token + refresh_token Microsoft.
//
// Client/secret résolus par ResolveAzureOAuthClient (source unique).
// Le redirectURI doit être strictement identique à celui passé dans la redirect
// vers /authorize (Microsoft vérifie ce match).
// codeVerifier : PKCE (RFC 7636) — vide = pas de PKCE (rétrocompat / device flow).
func ExchangeAuthorizationCode(ctx context.Context, code, redirectURI, codeVerifier string) (*AuthCodeResult, error) {
	// Poignée de main interactive (login SSO) : pas de garde netguard. Le flux de connexion n'est
	// pas monté en démo (voir l'en-tête de netguard_coverage_test.go).
	doer := &http.Client{Timeout: oauthTimeout}
	return oauthClient(doer).ExchangeAuthorizationCode(ctx, code, redirectURI, codeVerifier)
}

// BuildAuthorizeURL construit l'URL de redirect vers Microsoft /authorize pour
// initier le Authorization Code Flow. Le state doit être généré aléatoirement
// (32+ bytes) et persisté en session pour vérification au callback.
// codeChallenge : PKCE (RFC 7636, S256) — vide = pas de PKCE (rétrocompat).
func BuildAuthorizeURL(redirectURI, state, codeChallenge string) string {
	return xblauth.BuildAuthorizeURL(ResolveAzureOAuthClient().versXBL(), redirectURI, state, codeChallenge)
}

// GeneratePKCE génère un code_verifier et son code_challenge S256 (RFC 7636).
// Retourne (codeVerifier, codeChallenge, error).
func GeneratePKCE() (string, string, error) { return xblauth.GeneratePKCE() }
