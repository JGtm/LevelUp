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
	"log/slog"
	"net/http"
	"net/url"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/halo"
	"levelup/go-api/internal/platform/session"
)

// InviteValidator valide un code d'invitation (avec ou sans groupe).
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
	invites      InviteValidator              // optionnel : flow d'invitation
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

// WithInviteStore injecte le validateur d'invitations (flow d'invitation).
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
		// Lecture HTTP qui ÉCRIT (session, puis jetons) : la garde démo ne raisonne que sur
		// le verbe, d'où ce refus explicite, au même contrat (middleware/demo_read_only.go).
		slog.InfoContext(r.Context(), "demo: connexion Xbox refusée (démo en lecture seule)", "path", r.URL.Path)
		middleware.WriteDemoForbidden(w)
		return
	}
	// Navigation plein écran : tout refus renvoie le navigateur à l'écran de connexion avec un
	// code (refuseCallback), jamais un corps JSON affiché brut.
	if h.redirectURI == "" {
		slog.ErrorContext(r.Context(), "auth_xbox_oauth: LEVELUP_OAUTH_REDIRECT_URI non configuré")
		h.refuseCallback(w, r, "redirect_uri_not_configured")
		return
	}

	sess := middleware.GetSession(r.Context())
	if sess == nil {
		slog.ErrorContext(r.Context(), "auth_xbox_oauth: connexion sans session")
		h.refuseCallback(w, r, "no_session")
		return
	}

	// Génère un state aléatoire 32 bytes (256 bits) et stocke en session.
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		slog.ErrorContext(r.Context(), "auth_xbox_oauth: génération du state échouée", "err", err)
		h.refuseCallback(w, r, "state_gen_failed")
		return
	}
	state := hex.EncodeToString(stateBytes)
	sess.OAuthState = state

	// PKCE (RFC 7636) : verifier secret en session, challenge S256 dans l'URL.
	verifier, challenge, err := auth_platform.GeneratePKCE()
	if err != nil {
		slog.ErrorContext(r.Context(), "auth_xbox_oauth: génération PKCE échouée", "err", err)
		h.refuseCallback(w, r, "pkce_gen_failed")
		return
	}
	sess.OAuthCodeVerifier = verifier

	// Flow d'invitation : capter le code ?invite= et le porter en session jusqu'au
	// callback, même invalide : il ne donne rien par lui-même (la stratégie le réserve au
	// callback, Consume, et un code invalide n'est pas réservé), mais il permet au refus
	// du verrou de dire invite_invalid plutôt qu'instance fermée. Un code invalide ne
	// casse pas la redirection plein écran vers Microsoft (login normal).
	if code := r.URL.Query().Get("invite"); code != "" {
		if h.invites != nil {
			if verr := h.invites.Validate(code); verr != nil {
				slog.WarnContext(r.Context(), "auth_xbox_oauth: invitation ?invite= invalide, sans effet au callback", "err", verr)
			}
		}
		sess.PendingInviteCode = code
	}

	if err := h.sessionStore.Save(sess); err != nil {
		slog.ErrorContext(r.Context(), "auth_xbox_oauth: save session échec", "err", err)
		h.refuseCallback(w, r, "session_save_failed")
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
		// Lecture HTTP qui ÉCRIT (session, puis jetons) : la garde démo ne raisonne que sur
		// le verbe, d'où ce refus explicite, au même contrat (middleware/demo_read_only.go).
		slog.InfoContext(r.Context(), "demo: connexion Xbox refusée (démo en lecture seule)", "path", r.URL.Path)
		middleware.WriteDemoForbidden(w)
		return
	}

	sess := middleware.GetSession(r.Context())
	if sess == nil {
		slog.ErrorContext(r.Context(), "auth_xbox_oauth: callback sans session")
		h.refuseCallback(w, r, "no_session")
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
	if !h.completeOAuthCallback(w, r, sess, tokenResult, exchangeResult, xstsRTA) {
		return
	}

	// Redirect vers postLoginURL (front prend la suite).
	http.Redirect(w, r, h.postLoginURL, http.StatusFound)
}

// completeOAuthCallback termine le callback une fois les échanges avec Microsoft
// faits : identité complétée, connexion, persistance du refresh token. Rend faux,
// navigateur renvoyé à l'écran de connexion (refuseCallback), si la connexion est refusée.
func (h *XboxOAuthHandler) completeOAuthCallback(
	w http.ResponseWriter, r *http.Request, sess *domain.SessionData,
	tokenResult *auth_platform.AuthCodeResult, exchangeResult *auth_platform.ExchangeResult, xstsRTA *auth_platform.XSTSResult,
) bool {
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

	if !h.finishOAuthLogin(w, r, sess, buildXboxOAuthAttempt(tokenResult, exchangeResult, xstsRTA)) {
		return false
	}

	// Persistance ADR 0023 du RT frais, APRÈS la décision de connexion : une identité
	// refusée (instance verrouillée) ne laisse aucun fichier de jetons. Et après la
	// résolution d'identité : le XUID vient souvent de xstsRTA (l'exchange Halo ne
	// renvoie que `uhs`). Le store reçoit le RT frais → pas de split-brain au refresh
	// suivant.
	h.persistRefreshTokenToStore(r, exchangeResult.XUID, exchangeResult.Gamertag, tokenResult.RefreshToken)
	slog.InfoContext(r.Context(), "auth_xbox_oauth: login réussi via Authorization Code Flow",
		"gamertag", exchangeResult.Gamertag, "xuid", exchangeResult.XUID)
	return true
}

// finishOAuthLogin applique la stratégie post-flux (completeLogin : rien n'est écrit
// dans la session si elle refuse), puis renouvelle l'identifiant de session. Rend
// faux si la stratégie refuse, navigateur renvoyé à l'écran de connexion avec le code
// de loginFailureCode (instance_locked, invite_invalid, link_failed).
func (h *XboxOAuthHandler) finishOAuthLogin(
	w http.ResponseWriter, r *http.Request, sess *domain.SessionData, attempt *auth_platform.Attempt,
) bool {
	if err := completeLogin(r.Context(), h.linkStrategy, attempt, sess); err != nil {
		code := loginFailureCode(err)
		slog.WarnContext(r.Context(), "auth_xbox_oauth: connexion refusée par la stratégie post-flux",
			"xuid", attempt.XUID, "error_code", code, "err", err)
		h.refuseCallback(w, r, code)
		return false
	}
	// Nouvel identifiant de session : un cookie planté avant la connexion ne vaut rien.
	if _, err := h.sessionStore.Rotate(sess); err != nil {
		slog.ErrorContext(r.Context(), "auth_xbox_oauth: rotation de la session post-login échouée", "err", err)
	}
	return true
}

// validateCallbackParams vérifie error/code/state + CSRF ; un refus renvoie le
// navigateur à l'écran de connexion (refuseCallback). Retourne (code, true) si tout
// est OK, sinon ("", false).
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
		h.refuseCallback(w, r, "oauth_denied")
		return "", false
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		slog.WarnContext(r.Context(), "auth_xbox_oauth: callback sans code ou sans state",
			"code_set", code != "", "state_set", state != "")
		h.refuseCallback(w, r, "missing_params")
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
		h.refuseCallback(w, r, "state_mismatch")
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
		h.refuseCallback(w, r, "code_exchange_failed")
		return nil, nil, false
	}
	exchangeResult, err := h.provider.Exchange(r.Context(), tokenResult.AccessToken)
	if err != nil {
		slog.ErrorContext(r.Context(), "auth_xbox_oauth: provider.Exchange échec", "err", err)
		h.refuseCallback(w, r, "halo_exchange_failed")
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

// oauthLoginPath : l'écran de connexion du front, où le callback renvoie le navigateur
// quand il refuse ; oauthErrorParam porte le code du refus (le front le traduit en
// message FR / EN).
const (
	oauthLoginPath  = "/login"
	oauthErrorParam = "oauth_error"
)

// refuseCallback termine un refus de la connexion Xbox, au départ (LoginRedirect) comme au
// retour (Callback) : ce sont des navigations plein écran (pas des appels du front), un corps
// JSON s'afficherait brut. Le navigateur est renvoyé à l'écran de connexion avec le code du
// refus ; le détail reste dans les journaux.
func (h *XboxOAuthHandler) refuseCallback(w http.ResponseWriter, r *http.Request, code string) {
	slog.WarnContext(r.Context(), "auth_xbox_oauth: connexion Xbox refusée, retour à l'écran de connexion",
		"error_code", code)
	target := oauthLoginPath + "?" + url.Values{oauthErrorParam: {code}}.Encode()
	http.Redirect(w, r, target, http.StatusFound)
}
