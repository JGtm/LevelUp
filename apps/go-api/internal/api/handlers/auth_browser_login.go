// Package handlers — auth_browser_login.go : connexion Microsoft par le navigateur par défaut, sans
// code à recopier (profil desktop, capacité LoginViaBrowser, ADR 0045 D-22).
//
//	GET /api/v1/auth/xbox/login → { attempt_id, authorize_url, ... } (Huma) : la page de la coquille
//	                              ouvre l'adresse dans le navigateur (jamais dans sa fenêtre)
//	GET /auth/xbox/callback     → retour de Microsoft dans le navigateur, page « connexion réussie »
//
// FLUX. Code d'autorisation + PKCE S256 du client public partagé (ADR 0045 D-19), adresse de retour
// loopback http://localhost:<port lié>/auth/xbox/callback (Azure accepte tout port sur localhost).
// La tentative vit dans le registre du code d'appareil, liée à la session de la coquille ; le retour
// de Microsoft, qui arrive sans cookie, la retrouve par son `state`. Il échange le code, déroule la
// chaîne XBL, XSTS et Spartan, puis met la tentative au statut authorized : le sondage de la page
// (GET /auth/device-flow/{id}) la remet à la session et passe par la même stratégie post-flux que le
// code d'appareil (AuthCompletion : profil suivi, jetons chiffrés, cache purgé).
//
// SECRETS. Ni le `code` ni le `state` ni le verifier ne sont journalisés ; le client ne reçoit d'une
// erreur que son code fixe.
package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/i18n"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/session"
)

// BrowserLoginCallbackPath : le chemin racine de l'adresse de retour déclarée côté Azure
// (http://localhost/auth/xbox/callback). Écrit une fois, dans le garde de la coquille : l'adresse, le
// garde et le montage du routeur le partagent.
const BrowserLoginCallbackPath = middleware.DesktopBrowserCallbackPath

// Codes d'échec d'une tentative de navigateur : posés sur la tentative (le sondage de la page les lit)
// et montrés tels quels dans la page de retour.
const (
	browserErrDenied       = "oauth_denied"
	browserErrState        = "state_invalid"
	browserErrStateExpired = "state_expired"
	browserErrMissing      = "missing_params"
	browserErrCode         = "code_exchange_failed"
	browserErrHalo         = "halo_exchange_failed"
	browserErrIdentity     = "identity_missing"
)

// BrowserLoginBackend : ce que la connexion demande à Microsoft, injecté pour qu'un test joue un
// Azure simulé.
type BrowserLoginBackend interface {
	AuthorizeURL(redirectURI, state, challenge string) string
	ExchangeCode(ctx context.Context, code, redirectURI, verifier string) (*auth_platform.AuthCodeResult, error)
	AcquireRTA(ctx context.Context, accessToken string) (*auth_platform.XSTSResult, error)
}

// microsoftBackend : le vrai Microsoft, par la façade d'exploitation de platform/auth.
type microsoftBackend struct{}

func (microsoftBackend) AuthorizeURL(redirectURI, state, challenge string) string {
	return auth_platform.BuildAuthorizeURL(redirectURI, state, challenge)
}

func (microsoftBackend) ExchangeCode(ctx context.Context, code, redirectURI, verifier string) (*auth_platform.AuthCodeResult, error) {
	return auth_platform.ExchangeAuthorizationCode(ctx, code, redirectURI, verifier)
}

func (microsoftBackend) AcquireRTA(ctx context.Context, accessToken string) (*auth_platform.XSTSResult, error) {
	return auth_platform.AcquireXSTSForRTA(ctx, accessToken)
}

// BrowserLoginHandler porte les deux routes de la connexion par le navigateur.
type BrowserLoginHandler struct {
	sessionStore *session.Store
	attempts     *auth_platform.AttemptStore
	provider     auth_platform.TokenProvider
	backend      BrowserLoginBackend
	port         int
}

// NewBrowserLoginHandler crée le handler du serveur lié au port local port (celui de l'adresse de
// retour).
func NewBrowserLoginHandler(
	sessionStore *session.Store, attempts *auth_platform.AttemptStore, provider auth_platform.TokenProvider, port int,
) *BrowserLoginHandler {
	return &BrowserLoginHandler{
		sessionStore: sessionStore, attempts: attempts, provider: provider, backend: microsoftBackend{}, port: port,
	}
}

// WithBackend remplace l'échange Microsoft (tests).
func (h *BrowserLoginHandler) WithBackend(b BrowserLoginBackend) *BrowserLoginHandler {
	h.backend = b
	return h
}

// redirectURI : l'adresse de retour. « localhost » et non l'adresse numérique : c'est ce que déclare
// Azure, dont l'exemption loopback accepte tout port.
func (h *BrowserLoginHandler) redirectURI() string {
	return fmt.Sprintf("http://localhost:%d%s", h.port, BrowserLoginCallbackPath)
}

// browserLoginStartResponse : la réponse du démarrage.
type browserLoginStartResponse struct {
	AttemptID       string `json:"attempt_id" doc:"Attempt identifier, polled with GET /auth/device-flow/{attempt_id}."`
	AuthorizeURL    string `json:"authorize_url" doc:"Microsoft authorization URL to open in the default browser (never in the application window)."`
	ExpiresIn       int    `json:"expires_in" doc:"Attempt lifetime in seconds."`
	PollIntervalSec int    `json:"poll_interval_sec" doc:"Suggested polling interval in seconds."`
}

type browserLoginStartOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         browserLoginStartResponse
}

// MountStart enregistre GET /auth/xbox/login (Huma) sous le préfixe de montage de l'appelant (/api/v1).
func (h *BrowserLoginHandler) MountStart(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/auth/xbox/login", h.handleStart,
		humacore.Op("getAuthXboxLogin", "Start the Microsoft sign-in in the default browser", "setup"))
}

// handleStart ouvre une tentative liée à la session de la coquille et rend l'adresse Microsoft à ouvrir.
// GET /api/v1/auth/xbox/login
func (h *BrowserLoginHandler) handleStart(ctx context.Context, _ *struct{}) (*browserLoginStartOutput, error) {
	sess := middleware.GetSession(ctx)
	if sess == nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "no_session", "session not initialised")
	}
	state, err := randomHex(32)
	if err != nil {
		return nil, humacore.NewErrorFrom(ctx, http.StatusInternalServerError, "state_gen_failed", err)
	}
	verifier, challenge, err := auth_platform.GeneratePKCE()
	if err != nil {
		return nil, humacore.NewErrorFrom(ctx, http.StatusInternalServerError, "pkce_gen_failed", err)
	}
	attempt, err := h.attempts.StartBrowser(sess.SessionID, chimiddleware.GetClientIP(ctx), state, verifier,
		string(i18n.FromContext(ctx)))
	if err != nil {
		slog.WarnContext(ctx, "auth_browser_login: connexion refusée, trop de tentatives en attente", "err", err)
		return nil, huma.ErrorWithHeaders(
			humacore.NewError(http.StatusTooManyRequests, "too_many_device_flows",
				"too many sign-in attempts in progress, try again shortly"),
			http.Header{headerRetryAfter: []string{deviceFlowRetryAfterSec}})
	}
	// Lie la session à la tentative : une session anonyme n'est persistée que si elle porte
	// quelque chose, et le sondage retrouve la tentative par la session.
	sess.DeviceFlowAttemptID = &attempt.AttemptID
	sess.PendingDeviceFlowAttempt = attempt.AttemptID
	if err := h.sessionStore.Save(sess); err != nil {
		h.attempts.Fail(attempt.AttemptID, "session_save_failed")
		return nil, humacore.NewErrorFrom(ctx, http.StatusInternalServerError, "session_save_failed", err)
	}
	slog.InfoContext(ctx, "auth_browser_login: tentative ouverte", "attempt_id", attempt.AttemptID,
		"redirect_uri", h.redirectURI())
	return &browserLoginStartOutput{
		CacheControl: "no-store",
		Body: browserLoginStartResponse{
			AttemptID:       attempt.AttemptID,
			AuthorizeURL:    h.backend.AuthorizeURL(h.redirectURI(), state, challenge),
			ExpiresIn:       int(auth_platform.BrowserStateTTL.Seconds()),
			PollIntervalSec: deviceFlowPollIntervalSec,
		},
	}, nil
}

// Callback termine la connexion dans le navigateur : consomme le state, échange le code, déroule la
// chaîne d'échange et rend la page de retour. Route ouverte sans cookie de la coquille (le navigateur
// externe n'en a pas) : le state à usage unique et le Host loopback strict (garde de la coquille) la
// protègent.
// GET /auth/xbox/callback?code=…&state=…
func (h *BrowserLoginHandler) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	attempt, err := h.attempts.ClaimBrowserState(q.Get("state"))
	if err != nil {
		code := browserErrState
		if errors.Is(err, auth_platform.ErrBrowserStateExpired) {
			code = browserErrStateExpired
		}
		slog.WarnContext(ctx, "auth_browser_login: retour refusé", "error_code", code)
		renderBrowserLoginPage(w, i18n.Default, code)
		return
	}
	loc := i18n.Parse(attempt.Locale)
	if code := h.complete(ctx, attempt, q.Get("code"), q.Get("error")); code != "" {
		h.attempts.Fail(attempt.AttemptID, code)
		renderBrowserLoginPage(w, loc, code)
		return
	}
	renderBrowserLoginPage(w, loc, "")
}

// complete mène la tentative de l'état pending à authorized : échange du code, chaîne d'échange,
// identité. Rend le code d'échec, ou "" quand la tentative est autorisée.
func (h *BrowserLoginHandler) complete(ctx context.Context, attempt *auth_platform.Attempt, code, msError string) string {
	if msError != "" {
		// Le code d'erreur Microsoft (access_denied...) n'est pas un secret ; sa description reste hors du journal.
		slog.WarnContext(ctx, "auth_browser_login: Microsoft a refusé la connexion", "attempt_id", attempt.AttemptID, "error", msError)
		return browserErrDenied
	}
	if code == "" {
		slog.WarnContext(ctx, "auth_browser_login: retour sans code", "attempt_id", attempt.AttemptID)
		return browserErrMissing
	}
	tokenResult, err := h.backend.ExchangeCode(ctx, code, h.redirectURI(), attempt.CodeVerifier)
	if err != nil {
		slog.ErrorContext(ctx, "auth_browser_login: échange du code échoué", "attempt_id", attempt.AttemptID, "err", err)
		return browserErrCode
	}
	exchange, err := h.provider.Exchange(ctx, tokenResult.AccessToken)
	if err != nil {
		slog.ErrorContext(ctx, "auth_browser_login: échange en jetons Halo refusé", "attempt_id", attempt.AttemptID, "err", err)
		return browserErrHalo
	}
	rta, rerr := h.backend.AcquireRTA(ctx, tokenResult.AccessToken)
	if rerr != nil {
		slog.WarnContext(ctx, "auth_browser_login: XSTS Xbox Live échoué (non bloquant)", "attempt_id", attempt.AttemptID, "err", rerr)
		rta = nil
	}
	gamertag, xuid := exchange.Gamertag, exchange.XUID
	if (gamertag == "" || xuid == "") && rta != nil {
		gamertag, xuid = rta.Gamertag, rta.XUID
	}
	if gamertag == "" || xuid == "" {
		slog.ErrorContext(ctx, "auth_browser_login: identité introuvable après l'échange", "attempt_id", attempt.AttemptID)
		return browserErrIdentity
	}
	h.attempts.Update(attempt.AttemptID, func(a *auth_platform.Attempt) {
		a.Status = auth_platform.AttemptStatusAuthorized
		a.Gamertag, a.XUID = gamertag, xuid
		a.SpartanToken = exchange.Tokens.SpartanToken
		a.ClearanceToken = exchange.Tokens.ClearanceToken
		a.SpartanExpiresAt = exchange.Tokens.SpartanExpiresAt
		a.MicrosoftAccessToken = tokenResult.AccessToken
		a.OAuthRefreshToken = tokenResult.RefreshToken
		if rta != nil {
			a.XSTSRTAToken, a.XSTSRTAUserHash, a.XSTSRTAExpiresAt = rta.Token, rta.UserHash, rta.NotAfter
		}
	})
	slog.InfoContext(ctx, "auth_browser_login: connexion autorisée", "attempt_id", attempt.AttemptID, "xuid", xuid)
	return ""
}

// randomHex rend n octets aléatoires en hexadécimal.
func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// renderBrowserLoginPage écrit la page de retour, dans la langue loc : « connexion réussie » quand
// failure est vide, sinon l'échec et son code (jamais un détail). Page autonome : aucune ressource
// externe, une feuille de style en ligne que sa propre politique autorise.
func renderBrowserLoginPage(w http.ResponseWriter, loc i18n.Locale, failure string) {
	status := http.StatusOK
	title := i18n.Msg(loc, "desktop.browser_login.ok_title", nil)
	body := i18n.Msg(loc, "desktop.browser_login.ok_body", nil)
	if failure != "" {
		status = http.StatusBadRequest
		title = i18n.Msg(loc, "desktop.browser_login.error_title", nil)
		body = i18n.Msg(loc, "desktop.browser_login.error_body", i18n.Params{"code": failure})
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.WriteHeader(status)
	fmt.Fprintf(w, browserLoginPageTemplate, html.EscapeString(string(loc)), html.EscapeString(title),
		html.EscapeString(title), html.EscapeString(body))
}

// browserLoginPageTemplate : lang, title, h1, p.
const browserLoginPageTemplate = `<!doctype html>
<html lang="%s">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title>
<style>
:root{color-scheme:light dark}
body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;font:16px/1.5 system-ui,sans-serif}
main{max-width:28rem;padding:2rem;text-align:center}
h1{font-size:1.4rem;margin:0 0 .75rem}
p{margin:0;opacity:.8}
</style>
</head>
<body>
<main>
<h1>%s</h1>
<p>%s</p>
</main>
</body>
</html>
`
