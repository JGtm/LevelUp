// Package handlers — user_auth.go : endpoints auth locale (login/register/logout).
//
// POST /auth/login     → authentification par username/password
// POST /auth/register  → inscription (premier user = admin)
// POST /auth/logout     → déconnexion
// POST /auth/password   → définition opt-in d'un mot de passe (compte SSO)
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) au même point de
// montage que les routes chi d'origine (chemins /auth/* absolus) et enregistre
// les 4 POST via huma.Post. Logique métier inchangée (userstore + session), seul
// le wrapping HTTP change.
//
// Les corps sont lus via RawBody (pas de Body typé) + json.Unmarshal maison pour
// reproduire EXACTEMENT le contrat de décodage d'origine : un JSON invalide renvoie
// 400 {invalid_body} (et non le 422 de validation Huma qu'un Body typé produirait).
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/loginlimit"
	"levelup/go-api/internal/platform/session"
	"levelup/go-api/internal/platform/userstore"
)

// UserAuthHandler gère les endpoints d'authentification locale.
type UserAuthHandler struct {
	users        *userstore.Store
	invites      *userstore.InviteStore
	sessionStore *session.Store
	regMode      string // "invite" | "open" | "closed"
	authMode     string // "none" | "password" | "xbox" — D3 cohabitation
	// instanceLocked résout le verrou « instance fermée » (env OU app_settings).
	// nil → jamais verrouillé (rétrocompat tests). Cf. WithInstanceLock.
	instanceLocked func() bool
	// loginAttempts borne les échecs de mot de passe par identifiant et par IP
	// cliente, partagé par POST /auth/login et POST /auth/password.
	loginAttempts *loginlimit.Limiter
}

// NewUserAuthHandler crée un UserAuthHandler.
func NewUserAuthHandler(
	users *userstore.Store,
	invites *userstore.InviteStore,
	sessionStore *session.Store,
	regMode string,
) *UserAuthHandler {
	return &UserAuthHandler{
		users:         users,
		invites:       invites,
		sessionStore:  sessionStore,
		regMode:       regMode,
		loginAttempts: loginlimit.New(),
	}
}

// WithAuthMode injecte le mode d'authentification (D3 cohabitation password ↔ xbox).
// En mode "xbox" : Login password réservé aux admins ; Register password réservé
// au bootstrap du premier admin (users.json vide).
func (h *UserAuthHandler) WithAuthMode(authMode string) *UserAuthHandler {
	h.authMode = authMode
	return h
}

// WithInstanceLock injecte le résolveur du verrou « instance fermée ». Quand il
// retourne true, l'inscription de nouveaux comptes est refusée (sauf bootstrap
// du tout premier admin, users.json vide).
func (h *UserAuthHandler) WithInstanceLock(fn func() bool) *UserAuthHandler {
	h.instanceLocked = fn
	return h
}

// isInstanceLocked retourne true si le verrou est actif (résolveur non nil).
func (h *UserAuthHandler) isInstanceLocked() bool {
	return h.instanceLocked != nil && h.instanceLocked()
}

// Mount enregistre les 4 routes via Huma au même point de montage que les routes
// chi d'origine (chemins /auth/* absolus). En mode xbox/lockdown la logique métier
// (déléguée aux handlers) reste inchangée.
func (h *UserAuthHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/auth/login", h.handleLogin, humacore.Op("postAuthLogin", "Authentification email + password", "auth"))
	huma.Post(api, "/auth/register", h.handleRegister, humacore.Op("postAuthRegister", "Création de compte (mode invite-only ou public selon registration_mode)", "auth"),
		humacore.DefaultStatus(http.StatusCreated))
	huma.Post(api, "/auth/logout", h.handleLogout, humacore.Op("postAuthLogout", "Invalide la session courante", "auth"))
	huma.Post(api, "/auth/password", h.handleSetPassword, humacore.Op("postAuthPassword", "Définit/change le mot de passe de l'utilisateur connecté (opt-in, PR-C)", "auth"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// authBodyInput : corps brut décodé maison → 400 {invalid_body} sur JSON malformé
// (un Body typé renverrait le 422 de validation Huma).
type authBodyInput struct {
	RawBody []byte
}

type loginOutput struct{ Body domain.LoginResponse }

// registerOutput : 201 Created (statut posé par humacore.DefaultStatus au
// montage — source unique, runtime ET document).
type registerOutput struct {
	Body domain.RegisterResponse
}

// authNoContent : réponse 204 sans corps (Status override la valeur par défaut).
type authNoContent struct {
	Status int
}

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleLogin authentifie un utilisateur par username/password.
// POST /auth/login
func (h *UserAuthHandler) handleLogin(ctx context.Context, in *authBodyInput) (*loginOutput, error) {
	var req domain.LoginRequest
	if err := json.Unmarshal(in.RawBody, &req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "corps de requête invalide")
	}

	// Tentative refusée sans vérifier le mot de passe tant que l'identifiant ou l'IP
	// est bloqué (ni coût bcrypt, ni échec de plus).
	ip := chimiddleware.GetClientIP(ctx)
	if err := h.refuseIfLoginBlocked(ctx, req.Username, ip); err != nil {
		return nil, err
	}

	user, err := h.users.Authenticate(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, userstore.ErrInvalidCredentials) {
			h.loginAttempts.RecordFailure(req.Username, ip)
			slog.WarnContext(ctx, "auth: login échoué", "username", loginlimit.Truncate(req.Username), "ip", ip)
			return nil, humacore.NewError(http.StatusUnauthorized, "invalid_credentials", "identifiants incorrects")
		}
		slog.ErrorContext(ctx, "auth: erreur authenticate", "username", loginlimit.Truncate(req.Username), "ip", ip, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "auth_error", "erreur d'authentification")
	}
	h.loginAttempts.RecordSuccess(req.Username, ip)

	// PR-C : en mode xbox, le login password est autorisé pour tout utilisateur
	// AYANT défini un mot de passe (opt-in en fin d'onboarding via POST /auth/password).
	// Atteindre ce point implique qu'Authenticate a réussi → l'utilisateur a bien un
	// mot de passe ; les comptes SSO sans mot de passe échouent déjà à Authenticate.
	// (Avant PR-C : login password réservé aux admins en mode xbox.)

	if err := h.users.UpdateLastLogin(user.Username); err != nil {
		slog.Warn("auth: échec UpdateLastLogin", "username", user.Username, "err", err)
	}

	// Stocker les infos dans la session.
	sess := middleware.GetSession(ctx)
	if sess == nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "no_session", "session non initialisée")
	}
	sess.Username = &user.Username
	role := string(user.Role)
	sess.Role = &role
	h.autoSelectPlayer(sess, user)
	// Nouvel identifiant de session : un cookie planté avant la connexion ne vaut rien.
	if _, err := h.sessionStore.Rotate(sess); err != nil {
		slog.ErrorContext(ctx, "auth: rotation de la session à la connexion échouée", "username", user.Username, "err", err)
	}

	slog.InfoContext(ctx, "auth: login réussi", "username", user.Username, "role", user.Role, "ip", ip)

	return &loginOutput{Body: domain.LoginResponse{
		Username: user.Username,
		Role:     user.Role,
		Gamertag: user.Gamertag,
	}}, nil
}

// handleRegister inscrit un nouvel utilisateur.
// POST /auth/register
func (h *UserAuthHandler) handleRegister(ctx context.Context, in *authBodyInput) (*registerOutput, error) {
	var req domain.RegisterRequest
	if err := json.Unmarshal(in.RawBody, &req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "corps de requête invalide")
	}

	user, invited, err := h.createAccount(ctx, req)
	if err != nil {
		var herr huma.StatusError
		if errors.As(err, &herr) {
			return nil, err
		}
		if invited {
			h.releaseInvite(ctx, req)
		}
		if errors.Is(err, userstore.ErrUserAlreadyExists) {
			slog.Warn("auth: register username déjà pris", "username", loginlimit.Truncate(req.Username))
			return nil, humacore.NewError(http.StatusConflict, "user_exists", "nom d'utilisateur déjà pris")
		}
		if errors.Is(err, userstore.ErrInvalidUsername) || errors.Is(err, userstore.ErrPasswordTooShort) || errors.Is(err, userstore.ErrPasswordTooLong) {
			return nil, humacore.NewError(http.StatusBadRequest, "validation_error", err.Error())
		}
		return nil, humacore.NewError(http.StatusInternalServerError, "create_error", "erreur de création")
	}

	// Auto-login après inscription.
	sess := middleware.GetSession(ctx)
	if sess != nil {
		sess.Username = &user.Username
		roleStr := string(user.Role)
		sess.Role = &roleStr
		if _, err := h.sessionStore.Rotate(sess); err != nil {
			slog.ErrorContext(ctx, "auth: rotation de la session à l'inscription échouée", "username", user.Username, "err", err)
		}
	}

	slog.Info("auth: inscription réussie", "username", user.Username, "role", user.Role)

	return &registerOutput{Body: domain.RegisterResponse{
		Username: user.Username,
		Role:     user.Role,
	}}, nil
}

// createAccount crée le compte d'une inscription. Instance vide : le premier compte
// devient administrateur, décision prise sous le verrou du magasin
// (CreateFirstAdmin) ; une inscription qui perd la course ou arrive sur une instance
// non vide passe par les gardes ordinaires (admitRegistration). invited dit si un code
// d'invitation a été réservé. Une erreur HTTP (garde) est rendue telle quelle ; les
// erreurs du magasin sont traduites par l'appelant.
func (h *UserAuthHandler) createAccount(ctx context.Context, req domain.RegisterRequest) (*domain.User, bool, error) {
	empty, err := h.users.IsEmpty()
	if err != nil {
		return nil, false, humacore.NewError(http.StatusInternalServerError, "store_error", "erreur interne")
	}
	if empty {
		user, err := h.users.CreateFirstAdmin(req.Username, req.Password)
		if !errors.Is(err, userstore.ErrNotFirstAccount) {
			if err == nil {
				slog.InfoContext(ctx, "auth: premier utilisateur, rôle admin automatique", "username", loginlimit.Truncate(req.Username))
			}
			return user, false, err
		}
	}
	invited, err := h.admitRegistration(ctx, req)
	if err != nil {
		return nil, false, err
	}
	user, err := h.users.Create(req.Username, req.Password, domain.RoleUser)
	return user, invited, err
}

// admitRegistration applique les gardes d'une inscription hors premier compte :
// instance fermée, mode xbox, inscriptions fermées, puis, en mode invitation, la
// RÉSERVATION du code (Consume, atomique : un code ne crée qu'un compte). invited dit
// si un code a été réservé (à rendre si la création échoue).
func (h *UserAuthHandler) admitRegistration(ctx context.Context, req domain.RegisterRequest) (invited bool, err error) {
	// Instance fermée (lockdown) : aucune nouvelle inscription hors bootstrap
	// du premier admin.
	if h.isInstanceLocked() {
		slog.Warn("auth: register refusé — instance verrouillée", "username", loginlimit.Truncate(req.Username))
		return false, humacore.NewError(http.StatusForbidden, "instance_locked",
			"Cette instance est fermée aux nouvelles inscriptions.")
	}
	// D3 cohabitation : en mode "xbox", register password réservé au bootstrap admin
	// initial (users.json vide). Hors bootstrap, les comptes sont créés via le flow SSO.
	if h.authMode == "xbox" {
		slog.Warn("auth: register password bloqué en mode xbox", "username", loginlimit.Truncate(req.Username))
		return false, humacore.NewError(http.StatusForbidden, "register_xbox_mode",
			"mode SSO Xbox actif : les nouveaux comptes sont créés via la connexion Xbox")
	}
	if h.regMode == "closed" {
		return false, humacore.NewError(http.StatusForbidden, "registration_closed", "les inscriptions sont fermées")
	}
	if h.regMode != "invite" {
		return false, nil
	}
	if req.InviteCode == "" {
		return false, humacore.NewError(http.StatusBadRequest, "invite_required", "code d'invitation requis")
	}
	if _, err := h.invites.Consume(req.InviteCode, req.Username); err != nil {
		slog.WarnContext(ctx, "auth: register refusé, invitation inutilisable", "username", loginlimit.Truncate(req.Username), "err", err)
		return false, humacore.NewError(http.StatusForbidden, "invalid_invite", "code d'invitation invalide ou expiré")
	}
	return true, nil
}

// releaseInvite rend le code réservé par une inscription dont la création a échoué.
func (h *UserAuthHandler) releaseInvite(ctx context.Context, req domain.RegisterRequest) {
	if err := h.invites.Release(req.InviteCode, req.Username); err != nil {
		slog.ErrorContext(ctx, "auth: invitation réservée non rendue après une inscription échouée",
			"username", loginlimit.Truncate(req.Username), "err", err)
	}
}

// handleLogout déconnecte l'utilisateur : la session est supprimée côté serveur
// (identité, jetons Halo, état de navigation) et son identifiant révoqué ; le
// middleware de session répond par un cookie expiré. Le même cookie, rejoué, ne
// désigne plus aucune session. Quel que soit le mode (mot de passe, SSO Xbox).
// POST /auth/logout
func (h *UserAuthHandler) handleLogout(ctx context.Context, _ *struct{}) (*authNoContent, error) {
	sess := middleware.GetSession(ctx)
	if sess != nil {
		username := "<unknown>"
		if sess.Username != nil {
			username = *sess.Username
		}
		if err := h.sessionStore.Revoke(sess.SessionID); err != nil {
			slog.ErrorContext(ctx, "auth: suppression de la session à la déconnexion échouée", "err", err)
			return nil, humacore.NewError(http.StatusInternalServerError, "logout_failed", "sign-out failed, try again")
		}
		slog.InfoContext(ctx, "auth: logout", "username", username)
	}
	return &authNoContent{Status: http.StatusNoContent}, nil
}

// handleSetPassword définit/change le mot de passe de l'utilisateur connecté (opt-in PR-C).
// POST /auth/password — permet à un compte SSO Xbox de se reconnecter ensuite par
// mot de passe (re-login instantané, sans round-trip Microsoft).
func (h *UserAuthHandler) handleSetPassword(ctx context.Context, in *authBodyInput) (*authNoContent, error) {
	sess := middleware.GetSession(ctx)
	if sess == nil || sess.Username == nil {
		return nil, humacore.NewError(http.StatusUnauthorized, "auth_required", "authentification requise")
	}
	// Un identifiant (ou une IP) bloqué par des échecs de mot de passe ne change pas
	// non plus de mot de passe pendant le blocage.
	ip := chimiddleware.GetClientIP(ctx)
	if err := h.refuseIfLoginBlocked(ctx, *sess.Username, ip); err != nil {
		return nil, err
	}

	var req domain.SetPasswordRequest
	if err := json.Unmarshal(in.RawBody, &req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "corps de requête invalide")
	}
	if err := h.checkCurrentPassword(ctx, *sess.Username, req.CurrentPassword, ip); err != nil {
		return nil, err
	}

	if err := h.users.ResetPassword(*sess.Username, req.Password); err != nil {
		switch {
		case errors.Is(err, userstore.ErrPasswordTooShort), errors.Is(err, userstore.ErrPasswordTooLong):
			return nil, humacore.NewError(http.StatusBadRequest, "validation_error", err.Error())
		case errors.Is(err, userstore.ErrUserNotFound):
			return nil, humacore.NewError(http.StatusNotFound, "user_not_found", "utilisateur introuvable")
		default:
			slog.ErrorContext(ctx, "auth: échec set password", "username", *sess.Username, "err", err)
			return nil, errSetPassword()
		}
	}

	slog.Info("auth: mot de passe défini (opt-in)", "username", *sess.Username)
	h.closeOtherSessions(ctx, sess)
	return &authNoContent{Status: http.StatusNoContent}, nil
}

// refuseIfLoginBlocked rend le refus 429 too_many_login_attempts, avec Retry-After en
// secondes, quand l'identifiant ou l'IP cliente est bloqué par le limiteur d'échecs ;
// nil sinon.
func (h *UserAuthHandler) refuseIfLoginBlocked(ctx context.Context, identifier, ip string) error {
	wait, blocked := h.loginAttempts.Blocked(identifier, ip)
	if !blocked {
		return nil
	}
	slog.WarnContext(ctx, "auth: tentative refusée, trop d'échecs de mot de passe récents",
		"username", loginlimit.Truncate(identifier), "ip", ip, "retry_after", wait.Round(time.Second).String())
	retryAfter := strconv.Itoa(int(math.Ceil(wait.Seconds())))
	return huma.ErrorWithHeaders(
		humacore.NewError(http.StatusTooManyRequests, "too_many_login_attempts",
			"too many failed sign-in attempts, try again later"),
		http.Header{headerRetryAfter: []string{retryAfter}})
}

// autoSelectPlayer sélectionne automatiquement le joueur lié au gamertag de l'utilisateur.
func (h *UserAuthHandler) autoSelectPlayer(sess *domain.SessionData, user *domain.User) {
	if user.Gamertag == "" || sess == nil {
		return
	}
	slug := user.Gamertag
	sess.CurrentPlayerSlug = &slug
}
