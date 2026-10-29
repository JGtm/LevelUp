// oauth_refresh.go : échange d'un OAuth v2 refresh_token contre un access_token Microsoft.
//
//	refresh_token -> POST /oauth2/v2.0/token (app Azure) -> access_token
//	                 (repli : POST login.live.com, client Xbox natif)
//	                                        |
//	                              ExchangeAccessToken -> jetons Halo
//
// La brique ne garde ni ne persiste aucun jeton : le refresh_token tourné est RENDU, l'appelant le
// persiste (magasin de `platform/auth`, ADR 0023). Elle ne compte rien : l'issue typée
// [RefreshOutcome] dit quelles reprises ont eu lieu, l'appelant tient les compteurs.

package xblauth

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

	"levelup/go-api/internal/haloapi"
)

// OAuth : ce que les échanges OAuth v2 (rafraîchissement, code d'autorisation) reçoivent.
type OAuth struct {
	// Doer émet les requêtes (obligatoire) ; il porte la garde de sortie et le délai.
	Doer haloapi.Doer
	// Azure : le client de l'application Azure.
	Azure ClientAzure
	// URLAzure : endpoint /token de l'application Azure.
	URLAzure string
	// URLNatif : endpoint /token du client Xbox natif (en production [URLTokenMSA]), pour le
	// repli du rafraîchissement.
	URLNatif string
}

// RefreshOutcome : l'issue d'un rafraîchissement, que l'appelant persiste et compte.
type RefreshOutcome struct {
	// AccessToken : "" si Microsoft n'en a pas rendu (jeton révoqué sans code d'erreur).
	AccessToken string
	// RotatedRefreshToken : le refresh_token tourné par Microsoft ("" si aucune rotation).
	RotatedRefreshToken string
	// Famille : le client qui a RÉELLEMENT répondu ([FamilleAzure], [FamilleXboxNative]), ""
	// sur échec.
	Famille string
	// RetentePublique : la 1re tentative avec secret a été refusée (AADSTS90023) et l'échange a
	// été retenté en flux client public.
	RetentePublique bool
	// RetenteNatif : Azure a refusé le refresh_token (invalid_grant) et le repli sur le client
	// Xbox natif a été tenté.
	RetenteNatif bool
}

// oauthTokenResponse est la réponse JSON des endpoints /token.
type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	ErrorCode    string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// RefreshWithRotation échange un refresh_token et rend aussi le refresh_token tourné par
// Microsoft (qui le fait tourner à chaque appel) : l'appelant DOIT le persister, sinon le
// prochain appel échouera avec un jeton révoqué.
//
// Retourne (RefreshOutcome{}, nil) si refreshToken est vide (aucun appel).
// Retourne (issue, err) si l'appel HTTP ou Microsoft retourne une erreur ; l'issue dit quelles
// reprises ont été tentées.
// Retourne (issue avec AccessToken, nil) en cas de succès ; RotatedRefreshToken peut être vide
// (Microsoft ne renvoie pas de rotation : rare, toléré).
func (o OAuth) RefreshWithRotation(ctx context.Context, refreshToken string) (out RefreshOutcome, err error) {
	if refreshToken == "" {
		return RefreshOutcome{}, nil
	}

	start := time.Now()
	defer func() {
		// Échecs en Debug : l'erreur est propagée et loguée UNE fois au niveau du caller
		// (pool/resolver, avec sa classe) — cf. plan anti-bruit 2026-06-11. Les compteurs
		// expvar levelup.auth.* de l'appelant gardent la trace.
		if err != nil {
			slog.DebugContext(ctx, "oauth_refresh: échange échoué",
				"duration_ms", time.Since(start).Milliseconds(), "err", err)
		} else if out.AccessToken == "" {
			slog.DebugContext(ctx, "oauth_refresh: token révoqué (réponse vide)",
				"duration_ms", time.Since(start).Milliseconds())
		} else {
			slog.InfoContext(ctx, "oauth_refresh: échange OK",
				"duration_ms", time.Since(start).Milliseconds(),
				"rotated", out.RotatedRefreshToken != "" && out.RotatedRefreshToken != refreshToken)
		}
	}()

	// 1re tentative avec le secret confidentiel s'il est fourni : les RT émis par le flux web
	// confidentiel en ont besoin. Si Azure répond AADSTS90023 (RT émis par un flux client
	// PUBLIC — ex. device code — le secret est alors interdit), retenter UNE fois sans secret.
	// Stateless : 1 round-trip de plus par refresh concerné, aucun état à mémoriser.
	secret := o.Azure.Secret
	out.AccessToken, out.RotatedRefreshToken, err = o.postTokenExchange(ctx, refreshToken, secret)
	if err != nil && secret != "" {
		var oerr *OAuthExchangeError
		if errors.As(err, &oerr) && oerr.IsSecretRejected() {
			out.RetentePublique = true
			slog.InfoContext(ctx, "oauth_refresh: client_secret refusé (AADSTS90023) — retry en flux client public")
			out.AccessToken, out.RotatedRefreshToken, err = o.postTokenExchange(ctx, refreshToken, "")
		}
	}
	if err == nil && out.AccessToken != "" {
		out.Famille = FamilleAzure // l'app Azure a rafraîchi -> préfixe « d= ».
	}

	// Repli MSA natif : un RT émis par le client Xbox natif (device-flow, scope MBI_SSL) est
	// inconnu de l'app Azure — l'endpoint v2 le refuse en invalid_grant (« RT étranger »,
	// AADSTS70000). On retente alors UNE fois sur login.live.com avec le client Xbox et le scope
	// MSA d'origine. Si le repli échoue aussi, on propage l'erreur Azure INITIALE (classification
	// intacte pour le pool/resolver).
	if err != nil {
		var tente bool
		if at, rt, ferr := o.tryMSANativeFallback(ctx, refreshToken, err, &tente); ferr == nil {
			out.AccessToken, out.RotatedRefreshToken, out.Famille, err = at, rt, FamilleXboxNative, nil
		}
		out.RetenteNatif = tente
	}
	return out, err
}

// tryMSANativeFallback retente un refresh sur l'endpoint du client Xbox natif quand Azure a
// refusé le RT en invalid_grant (classe revoked). Rend (accessToken, rotatedRT, nil) si le repli
// réussit ; sinon ("", "", azureErr) — l'erreur Azure INITIALE est propagée. `tente` dit si le
// repli a été tenté.
func (o OAuth) tryMSANativeFallback(ctx context.Context, refreshToken string, azureErr error, tente *bool) (string, string, error) {
	var oerr *OAuthExchangeError
	if !errors.As(azureErr, &oerr) || oerr.Class() != AuthErrorRevoked {
		return "", "", azureErr
	}
	*tente = true
	slog.InfoContext(ctx, "oauth_refresh: invalid_grant côté Azure — retry refresh MSA natif (client Xbox/SISU)")
	at, rt, msaErr := o.postMSATokenExchange(ctx, refreshToken)
	if msaErr == nil && at != "" {
		return at, rt, nil
	}
	slog.DebugContext(ctx, "oauth_refresh: retry MSA natif échoué", "err", msaErr)
	return "", "", azureErr
}

// postTokenExchange exécute un POST /oauth2/v2.0/token (endpoint Azure v2) et décode la réponse.
// secret vide = flux client public (pas de champ client_secret).
func (o OAuth) postTokenExchange(ctx context.Context, refreshToken, secret string) (string, string, error) {
	body := url.Values{
		oauthFieldClientID:     {o.Azure.ID},
		oauthFieldGrantType:    {oauthFieldRefreshToken},
		oauthFieldRefreshToken: {refreshToken},
		oauthFieldScope:        {o.Azure.ScopesParam()},
	}
	if secret != "" {
		body.Set("client_secret", secret)
	}
	return o.postOAuthTokenForm(ctx, o.URLAzure, body)
}

// postMSATokenExchange rafraîchit un refresh_token MSA natif (émis par le device-flow du client
// Xbox natif, scope MBI_SSL). Flux client public — jamais de secret. Même contrat de retour que
// postTokenExchange.
func (o OAuth) postMSATokenExchange(ctx context.Context, refreshToken string) (string, string, error) {
	body := url.Values{
		oauthFieldClientID:     {IDAppXboxNatif},
		oauthFieldGrantType:    {oauthFieldRefreshToken},
		oauthFieldRefreshToken: {refreshToken},
		oauthFieldScope:        {ScopeMSA},
	}
	return o.postOAuthTokenForm(ctx, o.URLNatif, body)
}

// postOAuthTokenForm exécute un POST form-encoded sur un endpoint token OAuth 2.0 et décode la
// réponse (partagé Azure v2 / MSA — même contrat JSON). Une erreur [haloapi.ErrDefinitif] du
// Doer (mode démo) est rendue telle quelle : rien n'a été émis.
func (o OAuth) postOAuthTokenForm(ctx context.Context, tokenURL string, body url.Values) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL,
		strings.NewReader(body.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("oauth_refresh: construction requête: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.Doer.Do(req)
	if err != nil {
		if errors.Is(err, haloapi.ErrDefinitif) {
			return "", "", err
		}
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
