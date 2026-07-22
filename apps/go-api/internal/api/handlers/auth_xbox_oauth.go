// Package handlers — auth_xbox_oauth.go : Authorization Code Flow SSO Xbox (PR 4).
//
// Endpoints :
//
//	GET /auth/xbox/login    → redirect 302 vers Microsoft /authorize avec state CSRF
//	GET /auth/xbox/callback → reçoit code + state, exchange → tokens → login
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
	"levelup/go-api/internal/platform/halo"
	"levelup/go-api/internal/platform/session"
)

// InviteValidator valide un code d'invitation (flow "rejoindre un groupe").
// Satisfait par *userstore.InviteStore.
type InviteValidator interface {
	Validate(code string) error
}

// XboxOAuthHandler gère le Authorization Code Flow SSO Xbox.
type XboxOAuthHandler struct {
	sessionStore *session.Store
	provider     auth_platform.TokenProvider
	demoMode     bool
	redirectURI  string                       // URI publique de callback (config)
	linkStrategy auth_platform.LinkStrategy   // post-flow : login user
	postLoginURL string                       // où rediriger après succès (typiquement "/")
	authStore    auth_platform.UserTokenStore // ADR 0023 : persister le RT post-SSO
	invites      InviteValidator              // optionnel : flow "rejoindre un groupe"
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

// WithAuthStore injecte le MultiUserTokenStore (ADR 0023). Quand présent, le
// callback OAuth persiste le refresh_token Microsoft frais reçu (post-flow)
// dans le store — indispensable pour que le serveur puisse rafraîchir les
// tokens du joueur après la première authentification SSO sans re-prompter.
// Nil → mode legacy (pas de persistance, le RT est perdu après la requête).
func (h *XboxOAuthHandler) WithAuthStore(s auth_platform.UserTokenStore) *XboxOAuthHandler {
	h.authStore = s
	return h
}

// WithPostLoginURL définit l'URL de redirect après login réussi (défaut "/").
func (h *XboxOAuthHandler) WithPostLoginURL(url string) *XboxOAuthHandler {
	if url != "" {
		h.postLoginURL = url
	}
	return h
}

// WithInviteStore injecte le validateur d'invitations (flow "rejoindre un groupe").
// Quand présent, LoginRedirect valide le code ?invite= et le stocke en session pour
// que la LinkStrategy l'applique après login (bypass instance lock + ajout au groupe).
func (h *XboxOAuthHandler) WithInviteStore(inv InviteValidator) *XboxOAuthHandler {
	h.invites = inv
	return h
}

// LoginRedirect démarre le Authorization Code Flow.
// GET /auth/xbox/login
func (h *XboxOAuthHandler) LoginRedirect(w http.ResponseWriter, r *http.Request) {
	if h.demoMode {
		writeError(r.Context(), w, http.StatusUnprocessableEntity, "demo_mode", "authentification indisponible en mode démo")
		return
	}
	if h.redirectURI == "" {
		writeError(r.Context(), w, http.StatusInternalServerError, "redirect_uri_not_configured",
			"LEVELUP_OAUTH_REDIRECT_URI non configuré")
		return
	}

	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "no_session", "session non initialisée")
		return
	}

	// Génère un state aléatoire 32 bytes (256 bits) et stocke en session.
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "state_gen_failed", "génération state échouée")
		return
	}
	state := hex.EncodeToString(stateBytes)
	sess.OAuthState = state

	// PKCE (RFC 7636) : verifier secret en session, challenge S256 dans l'URL.
	verifier, challenge, err := auth_platform.GeneratePKCE()
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "pkce_gen_failed", "génération PKCE échouée")
		return
	}
	sess.OAuthCodeVerifier = verifier

	// Flow "rejoindre un groupe" : capter le code ?invite= et le porter en session
	// jusqu'au callback. Un code invalide est ignoré (login normal) plutôt que de
	// casser la redirection plein écran vers Microsoft.
	if code := r.URL.Query().Get("invite"); code != "" {
		if h.invites != nil {
			if verr := h.invites.Validate(code); verr != nil {
				slog.WarnContext(r.Context(), "auth_xbox_oauth: invitation ?invite= invalide, ignorée", "err", verr)
			} else {
				sess.PendingInviteCode = code
			}
		} else {
			sess.PendingInviteCode = code
		}
	}

	if err := h.sessionStore.Save(sess); err != nil {
		slog.ErrorContext(r.Context(), "auth_xbox_oauth: save session échec", "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "session_save_failed", "sauvegarde session échouée")
		return
	}

	url := auth_platform.BuildAuthorizeURL(h.redirectURI, state, challenge)
	slog.InfoContext(r.Context(), "auth_xbox_oauth: redirect vers Microsoft /authorize",
		"redirect_uri", h.redirectURI)
	http.Redirect(w, r, url, http.StatusFound)
}

// Callback reçoit la redirect Microsoft après authentification.
// GET /auth/xbox/callback?code=...&state=...
func (h *XboxOAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if h.demoMode {
		writeError(r.Context(), w, http.StatusUnprocessableEntity, "demo_mode", "authentification indisponible en mode démo")
		return
	}

	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "no_session", "session non initialisée")
		return
	}

	// PKCE : capturer le code_verifier AVANT que validateCallbackParams ne le consomme.
	codeVerifier := sess.OAuthCodeVerifier

	code, ok := h.validateCallbackParams(w, r, sess)
	if !ok {
		return
	}

	tokenResult, exchangeResult, ok := h.exchangeCallbackTokens(w, r, code, codeVerifier)
	if !ok {
		return
	}

	xstsRTA := h.tryAcquireXSTSForRTA(r, tokenResult.AccessToken)

	// L'audience XSTS Halo ne renvoie que `uhs` dans DisplayClaims (pas xid/gtg) :
	// suffisant pour les tokens Halo, mais pas pour identifier le joueur au login SSO.
	// Le XSTS Xbox Live (xstsRTA, relying party http://xboxlive.com) porte lui xid +
	// gamertag : on complète l'identité depuis cette source déjà acquise.
	if xstsRTA != nil {
		if exchangeResult.XUID == "" {
			exchangeResult.XUID = xstsRTA.XUID
		}
		if exchangeResult.Gamertag == "" {
			exchangeResult.Gamertag = xstsRTA.Gamertag
		}
	}

	// Persistance ADR 0023 du RT frais APRÈS résolution d'identité : le XUID vient
	// souvent de xstsRTA (l'exchange Halo ne renvoie que `uhs`). Garantit que le
	// store reçoit le RT e1cb35ab frais → pas de split-brain au refresh suivant.
	h.persistRefreshTokenToStore(r, exchangeResult.XUID, exchangeResult.Gamertag, tokenResult.RefreshToken)

	attempt := buildXboxOAuthAttempt(tokenResult, exchangeResult, xstsRTA)

	if !h.linkAttemptToSession(w, r, attempt, sess) {
		return
	}

	h.persistSessionAfterOAuth(r, sess, exchangeResult)

	slog.InfoContext(r.Context(), "auth_xbox_oauth: login réussi via Authorization Code Flow",
		"gamertag", exchangeResult.Gamertag, "xuid", exchangeResult.XUID)

	// Redirect vers postLoginURL (front prend la suite).
	http.Redirect(w, r, h.postLoginURL, http.StatusFound)
}

// validateCallbackParams vérifie error/code/state + CSRF, écrit la réponse d'erreur
// si nécessaire. Retourne (code, true) si tout est OK, sinon ("", false).
func (h *XboxOAuthHandler) validateCallbackParams(w http.ResponseWriter, r *http.Request, sess *domain.SessionData) (string, bool) {
	// Erreur Microsoft (user a refusé ou autre).
	if errCode := r.URL.Query().Get("error"); errCode != "" {
		errDesc := r.URL.Query().Get("error_description")
		slog.WarnContext(r.Context(), "auth_xbox_oauth: erreur Microsoft", "error", errCode, "description", errDesc)
		sess.OAuthState = ""
		sess.OAuthCodeVerifier = ""
		if err := h.sessionStore.Save(sess); err != nil {
			slog.ErrorContext(r.Context(), "auth_xbox_oauth: persistance session (reset état OAuth) échouée", "err", err)
		}
		writeError(r.Context(), w, http.StatusBadRequest, "oauth_denied", errDesc)
		return "", false
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_params", "code ou state absent")
		return "", false
	}

	// Vérification CSRF : state doit matcher celui stocké en session.
	expected := sess.OAuthState
	sess.OAuthState = ""        // consommer (one-shot)
	sess.OAuthCodeVerifier = "" // PKCE one-shot (déjà capturé par le Callback)
	if err := h.sessionStore.Save(sess); err != nil {
		slog.ErrorContext(r.Context(), "auth_xbox_oauth: persistance session (consommation state/PKCE) échouée", "err", err)
	}
	if expected == "" || state != expected {
		slog.WarnContext(r.Context(), "auth_xbox_oauth: state mismatch — possible CSRF",
			"expected_set", expected != "", "received_set", state != "")
		writeError(r.Context(), w, http.StatusForbidden, "state_mismatch", "state CSRF invalide")
		return "", false
	}
	return code, true
}

// exchangeCallbackTokens fait ExchangeAuthorizationCode puis provider.Exchange.
// Persiste aussi le refresh_token Microsoft dans le MultiUserTokenStore (ADR 0023)
// pour que les refresh ultérieurs n'aient pas besoin d'un nouveau flow interactif.
func (h *XboxOAuthHandler) exchangeCallbackTokens(
	w http.ResponseWriter, r *http.Request, code, codeVerifier string,
) (*auth_platform.AuthCodeResult, *auth_platform.ExchangeResult, bool) {
	tokenResult, err := auth_platform.ExchangeAuthorizationCode(r.Context(), code, h.redirectURI, codeVerifier)
	if err != nil {
		slog.ErrorContext(r.Context(), "auth_xbox_oauth: échange code échec", "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "code_exchange_failed", err.Error())
		return nil, nil, false
	}
	exchangeResult, err := h.provider.Exchange(r.Context(), tokenResult.AccessToken)
	if err != nil {
		slog.ErrorContext(r.Context(), "auth_xbox_oauth: provider.Exchange échec", "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "halo_exchange_failed", err.Error())
		return nil, nil, false
	}

	// NB : la persistance du RT au MultiUserTokenStore est faite par le Callback
	// APRÈS la résolution d'identité (le XUID peut venir de xstsRTA, l'exchange Halo
	// ne renvoyant que `uhs`). La faire ici la sauterait (XUID encore vide).
	return tokenResult, exchangeResult, true
}

// persistRefreshTokenToStore écrit le refresh_token Microsoft frais dans le
// MultiUserTokenStore (ADR 0023), keyé par le XUID RÉSOLU. À appeler APRÈS la
// résolution d'identité : sinon le XUID est vide (l'exchange Halo ne renvoie que
// `uhs`) et le RT n'est jamais persisté → le store garde un RT périmé →
// AADSTS70000 « token issued for a different client id » aux refresh suivants
// (split-brain tracker/store, bannière de reconnexion à tort).
func (h *XboxOAuthHandler) persistRefreshTokenToStore(r *http.Request, xuid, gamertag, refreshToken string) {
	if h.authStore == nil || refreshToken == "" || xuid == "" {
		return
	}
	if werr := h.authStore.UpdateOAuthRefreshToken(xuid, refreshToken); werr != nil {
		slog.WarnContext(r.Context(), "auth_xbox_oauth: persistance RT au store échouée",
			"xuid", xuid, "err", werr)
		return
	}
	slog.InfoContext(r.Context(), "auth_xbox_oauth: RT persisté au store",
		"xuid", xuid, "gamertag", gamertag)
	// Re-login = nouvelle chaîne d'auth : purger le cache process des HaloTokens
	// (Spartan/Clearance, TTL 50min) pour que les fetchs live (career, challenges,
	// battle pass, identité Explorer) repartent IMMÉDIATEMENT sur les tokens frais,
	// sans attendre l'expiration du cache. Cf. onRotated (même invalidation).
	halo.InvalidateCachedPlayerTokens(xuid)
}

// tryAcquireXSTSForRTA best-effort, retourne nil si l'acquisition échoue (non bloquant).
func (h *XboxOAuthHandler) tryAcquireXSTSForRTA(r *http.Request, accessToken string) *auth_platform.XSTSResult {
	rtaResult, xerr := auth_platform.AcquireXSTSForRTA(r.Context(), accessToken)
	if xerr != nil {
		slog.WarnContext(r.Context(), "auth_xbox_oauth: AcquireXSTSForRTA échec (non bloquant)", "err", xerr)
		return nil
	}
	return rtaResult
}

// buildXboxOAuthAttempt assemble l'Attempt éphémère pour LinkStrategy.OnAuthSuccess.
func buildXboxOAuthAttempt(
	tokenResult *auth_platform.AuthCodeResult,
	exchangeResult *auth_platform.ExchangeResult,
	xstsRTA *auth_platform.XSTSResult,
) *auth_platform.Attempt {
	attempt := &auth_platform.Attempt{
		Status:               auth_platform.AttemptStatusAuthorized,
		SpartanToken:         exchangeResult.Tokens.SpartanToken,
		ClearanceToken:       exchangeResult.Tokens.ClearanceToken,
		SpartanExpiresAt:     exchangeResult.Tokens.SpartanExpiresAt,
		Gamertag:             exchangeResult.Gamertag,
		XUID:                 exchangeResult.XUID,
		MicrosoftAccessToken: tokenResult.AccessToken,
	}
	if xstsRTA != nil {
		attempt.XSTSRTAToken = xstsRTA.Token
		attempt.XSTSRTAUserHash = xstsRTA.UserHash
		attempt.XSTSRTAExpiresAt = xstsRTA.NotAfter
	}
	return attempt
}

// linkAttemptToSession invoque LinkStrategy.OnAuthSuccess. Retourne true si OK
// (ou erreur tolérée non-bloquante), false si l'erreur est fatale (réponse écrite).
func (h *XboxOAuthHandler) linkAttemptToSession(
	w http.ResponseWriter, r *http.Request, attempt *auth_platform.Attempt, sess *domain.SessionData,
) bool {
	if h.linkStrategy == nil {
		return true
	}
	if err := h.linkStrategy.OnAuthSuccess(r.Context(), attempt, sess); err != nil {
		if errors.Is(err, auth_platform.ErrSessionNotAuthenticated) {
			slog.WarnContext(r.Context(), "auth_xbox_oauth: LinkStrategy renvoie ErrSessionNotAuthenticated")
			return true
		}
		slog.ErrorContext(r.Context(), "auth_xbox_oauth: LinkStrategy.OnAuthSuccess échec", "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "link_failed", err.Error())
		return false
	}
	return true
}

// persistSessionAfterOAuth renseigne les Halo tokens et l'identité, puis sauvegarde la session.
func (h *XboxOAuthHandler) persistSessionAfterOAuth(
	r *http.Request, sess *domain.SessionData, exchangeResult *auth_platform.ExchangeResult,
) {
	sess.AuthReady = true
	sess.HaloTokens = &domain.HaloTokens{
		SpartanToken:     exchangeResult.Tokens.SpartanToken,
		ClearanceToken:   exchangeResult.Tokens.ClearanceToken,
		SpartanExpiresAt: exchangeResult.Tokens.SpartanExpiresAt,
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
}
