// Package handlers — auth_xbox_oauth.go : Authorization Code Flow SSO Xbox (PR 4).
//
// Endpoints :
//   GET /auth/xbox/login    → redirect 302 vers Microsoft /authorize avec state CSRF
//   GET /auth/xbox/callback → reçoit code + state, exchange → tokens → login
//
// Pré-requis Azure : la plateforme "Web" doit être ajoutée à l'app dans le
// portail Azure avec le redirect URI configuré (typiquement le path callback
// + le host public). Sans cette config, Microsoft retourne AADSTS50011 dès la
// redirect /authorize.
package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/session"
)

// XboxOAuthHandler gère le Authorization Code Flow SSO Xbox.
type XboxOAuthHandler struct {
	sessionStore *session.Store
	provider     auth_platform.TokenProvider
	demoMode     bool
	redirectURI  string                      // URI publique de callback (config)
	linkStrategy auth_platform.LinkStrategy  // post-flow : login user
	postLoginURL string                      // où rediriger après succès (typiquement "/")
}

// NewXboxOAuthHandler crée un XboxOAuthHandler.
// redirectURI doit être strictement identique à celui configuré côté Azure.
func NewXboxOAuthHandler(
	sessionStore *session.Store,
	provider auth_platform.TokenProvider,
	demoMode bool,
	redirectURI string,
) *XboxOAuthHandler {
	return &XboxOAuthHandler{
		sessionStore: sessionStore,
		provider:     provider,
		demoMode:     demoMode,
		redirectURI:  redirectURI,
		postLoginURL: "/",
	}
}

// WithLinkStrategy injecte la LinkStrategy (typiquement XboxSSOLinkStrategy).
func (h *XboxOAuthHandler) WithLinkStrategy(s auth_platform.LinkStrategy) *XboxOAuthHandler {
	h.linkStrategy = s
	return h
}

// WithPostLoginURL définit l'URL de redirect après login réussi (défaut "/").
func (h *XboxOAuthHandler) WithPostLoginURL(url string) *XboxOAuthHandler {
	if url != "" {
		h.postLoginURL = url
	}
	return h
}

// LoginRedirect démarre le Authorization Code Flow.
// GET /auth/xbox/login
func (h *XboxOAuthHandler) LoginRedirect(w http.ResponseWriter, r *http.Request) {
	if h.demoMode {
		writeError(w, http.StatusUnprocessableEntity, "demo_mode", "authentification indisponible en mode démo")
		return
	}
	if h.redirectURI == "" {
		writeError(w, http.StatusInternalServerError, "redirect_uri_not_configured",
			"LEVELUP_OAUTH_REDIRECT_URI non configuré")
		return
	}

	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeError(w, http.StatusInternalServerError, "no_session", "session non initialisée")
		return
	}

	// Génère un state aléatoire 32 bytes (256 bits) et stocke en session.
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "state_gen_failed", "génération state échouée")
		return
	}
	state := hex.EncodeToString(stateBytes)
	sess.OAuthState = state
	if err := h.sessionStore.Save(sess); err != nil {
		slog.ErrorContext(r.Context(), "auth_xbox_oauth: save session échec", "err", err)
		writeError(w, http.StatusInternalServerError, "session_save_failed", "sauvegarde session échouée")
		return
	}

	url := auth_platform.BuildAuthorizeURL(h.redirectURI, state)
	slog.InfoContext(r.Context(), "auth_xbox_oauth: redirect vers Microsoft /authorize",
		"redirect_uri", h.redirectURI)
	http.Redirect(w, r, url, http.StatusFound)
}

// Callback reçoit la redirect Microsoft après authentification.
// GET /auth/xbox/callback?code=...&state=...
func (h *XboxOAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if h.demoMode {
		writeError(w, http.StatusUnprocessableEntity, "demo_mode", "authentification indisponible en mode démo")
		return
	}

	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeError(w, http.StatusInternalServerError, "no_session", "session non initialisée")
		return
	}

	// Erreur Microsoft (user a refusé ou autre).
	if errCode := r.URL.Query().Get("error"); errCode != "" {
		errDesc := r.URL.Query().Get("error_description")
		slog.WarnContext(r.Context(), "auth_xbox_oauth: erreur Microsoft", "error", errCode, "description", errDesc)
		// Effacer le state pour empêcher la rejouabilité.
		sess.OAuthState = ""
		_ = h.sessionStore.Save(sess)
		writeError(w, http.StatusBadRequest, "oauth_denied", errDesc)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		writeError(w, http.StatusBadRequest, "missing_params", "code ou state absent")
		return
	}

	// Vérification CSRF : state doit matcher celui stocké en session.
	expected := sess.OAuthState
	sess.OAuthState = "" // consommer (one-shot)
	_ = h.sessionStore.Save(sess)
	if expected == "" || state != expected {
		slog.WarnContext(r.Context(), "auth_xbox_oauth: state mismatch — possible CSRF",
			"expected_set", expected != "", "received_set", state != "")
		writeError(w, http.StatusForbidden, "state_mismatch", "state CSRF invalide")
		return
	}

	// Exchange code → tokens Microsoft.
	tokenResult, err := auth_platform.ExchangeAuthorizationCode(r.Context(), code, h.redirectURI)
	if err != nil {
		slog.ErrorContext(r.Context(), "auth_xbox_oauth: échange code échec", "err", err)
		writeError(w, http.StatusInternalServerError, "code_exchange_failed", err.Error())
		return
	}

	// Chaîne d'échange : access_token Microsoft → tokens Halo + identité.
	exchangeResult, err := h.provider.Exchange(r.Context(), tokenResult.AccessToken)
	if err != nil {
		slog.ErrorContext(r.Context(), "auth_xbox_oauth: provider.Exchange échec", "err", err)
		writeError(w, http.StatusInternalServerError, "halo_exchange_failed", err.Error())
		return
	}

	// AcquireXSTSForRTA best-effort (PR 2.5a).
	var xstsRTA *auth_platform.XSTSResult
	if rtaResult, xerr := auth_platform.AcquireXSTSForRTA(r.Context(), tokenResult.AccessToken); xerr == nil {
		xstsRTA = rtaResult
	} else {
		slog.WarnContext(r.Context(), "auth_xbox_oauth: AcquireXSTSForRTA échec (non bloquant)", "err", xerr)
	}

	// Construire un Attempt éphémère pour réutiliser LinkStrategy.OnAuthSuccess
	// (même contrat que pollDeviceFlow du Device Code Flow).
	attempt := &auth_platform.Attempt{
		Status:               auth_platform.AttemptStatusAuthorized,
		SpartanToken:         exchangeResult.Tokens.SpartanToken,
		ClearanceToken:       exchangeResult.Tokens.ClearanceToken,
		Gamertag:             exchangeResult.Gamertag,
		XUID:                 exchangeResult.XUID,
		MicrosoftAccessToken: tokenResult.AccessToken,
	}
	if xstsRTA != nil {
		attempt.XSTSRTAToken = xstsRTA.Token
		attempt.XSTSRTAUserHash = xstsRTA.UserHash
		attempt.XSTSRTAExpiresAt = xstsRTA.NotAfter
	}
	// Note : Authorization Code Flow donne directement le refresh_token brut
	// (contrairement au Device Code via MSAL qui ne l'expose pas). Pour rester
	// compatible avec le MultiUserTokenStore (qui attend MSALCacheJSON), on ne
	// peut pas réutiliser ce RT tel quel — il faudrait un store distinct ou un
	// helper qui construit un cache MSAL à partir d'un RT brut. Différé.
	// En pratique : la persistance via XboxSSOLinkStrategy fonctionne (XSTS RTA
	// + access_token), mais le refresh ultérieur via MSAL silent ne sera pas
	// possible. À l'expiration du XSTS (~55min), le user devra re-faire le
	// flow. Acceptable pour PR 4 (flux interactif).

	// Wire la session via la LinkStrategy injectée (XboxSSOLinkStrategy).
	if h.linkStrategy != nil {
		if err := h.linkStrategy.OnAuthSuccess(r.Context(), attempt, sess); err != nil {
			if errors.Is(err, auth_platform.ErrSessionNotAuthenticated) {
				// PasswordLinkStrategy renvoie ça en mode password — on est en mode xbox,
				// donc ne devrait pas arriver. Log warning et continue.
				slog.WarnContext(r.Context(), "auth_xbox_oauth: LinkStrategy renvoie ErrSessionNotAuthenticated")
			} else {
				slog.ErrorContext(r.Context(), "auth_xbox_oauth: LinkStrategy.OnAuthSuccess échec", "err", err)
				writeError(w, http.StatusInternalServerError, "link_failed", err.Error())
				return
			}
		}
	}

	// Stocker les Halo tokens dans la session (jamais exposés).
	sess.AuthReady = true
	sess.HaloTokens = &domain.HaloTokens{
		SpartanToken:   exchangeResult.Tokens.SpartanToken,
		ClearanceToken: exchangeResult.Tokens.ClearanceToken,
	}
	if exchangeResult.Gamertag != "" {
		sess.LinkedHaloIdentity = &domain.HaloIdentity{
			Gamertag: exchangeResult.Gamertag,
			XUID:     exchangeResult.XUID,
		}
	}
	if err := h.sessionStore.Save(sess); err != nil {
		slog.ErrorContext(r.Context(), "auth_xbox_oauth: save session post-login échec", "err", err)
	}

	slog.InfoContext(r.Context(), "auth_xbox_oauth: login réussi via Authorization Code Flow",
		"gamertag", exchangeResult.Gamertag, "xuid", exchangeResult.XUID)

	// Redirect vers postLoginURL (front prend la suite).
	http.Redirect(w, r, h.postLoginURL, http.StatusFound)
}
