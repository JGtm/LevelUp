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
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
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
}

// NewUserAuthHandler crée un UserAuthHandler.
func NewUserAuthHandler(
	users *userstore.Store,
	invites *userstore.InviteStore,
	sessionStore *session.Store,
	regMode string,
) *UserAuthHandler {
	return &UserAuthHandler{
		users:        users,
		invites:      invites,
		sessionStore: sessionStore,
		regMode:      regMode,
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
	huma.Post(api, "/auth/login", h.handleLogin)
	huma.Post(api, "/auth/register", h.handleRegister)
	huma.Post(api, "/auth/logout", h.handleLogout)
	huma.Post(api, "/auth/password", h.handleSetPassword)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// authBodyInput : corps brut décodé maison → 400 {invalid_body} sur JSON malformé
// (un Body typé renverrait le 422 de validation Huma).
type authBodyInput struct {
	RawBody []byte
}

type loginOutput struct{ Body domain.LoginResponse }

type registerOutput struct {
	Status int
	Body   domain.RegisterResponse
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

	user, err := h.users.Authenticate(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, userstore.ErrInvalidCredentials) {
			slog.Warn("auth: login échoué", "username", req.Username)
			return nil, humacore.NewError(http.StatusUnauthorized, "invalid_credentials", "identifiants incorrects")
		}
		slog.ErrorContext(ctx, "auth: erreur authenticate", "username", req.Username, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "auth_error", "erreur d'authentification")
	}

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
	if err := h.sessionStore.Save(sess); err != nil {
		slog.ErrorContext(ctx, "auth: échec save session login", "username", user.Username, "err", err)
	}

	slog.Info("auth: login réussi", "username", user.Username, "role", user.Role)

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

	// Vérifier si c'est le premier utilisateur (auto-admin).
	empty, err := h.users.IsEmpty()
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "store_error", "erreur interne")
	}

	role := domain.RoleUser
	if empty {
		// Premier utilisateur = admin automatique, pas besoin de code d'invitation.
		role = domain.RoleAdmin
		slog.Info("auth: premier utilisateur, rôle admin automatique", "username", req.Username)
	} else {
		// Instance fermée (lockdown) : aucune nouvelle inscription hors bootstrap
		// du premier admin (déjà exempté par la branche `empty` ci-dessus).
		if h.isInstanceLocked() {
			slog.Warn("auth: register refusé — instance verrouillée", "username", req.Username)
			return nil, humacore.NewError(http.StatusForbidden, "instance_locked",
				"Cette instance est fermée aux nouvelles inscriptions.")
		}
		// D3 cohabitation : en mode "xbox", register password réservé au bootstrap admin
		// initial (users.json vide). Hors bootstrap, les comptes sont créés via le flow SSO.
		if h.authMode == "xbox" {
			slog.Warn("auth: register password bloqué en mode xbox", "username", req.Username)
			return nil, humacore.NewError(http.StatusForbidden, "register_xbox_mode",
				"mode SSO Xbox actif : les nouveaux comptes sont créés via la connexion Xbox")
		}
		// Vérifier le mode d'inscription.
		if h.regMode == "closed" {
			return nil, humacore.NewError(http.StatusForbidden, "registration_closed", "les inscriptions sont fermées")
		}
		if h.regMode == "invite" {
			if req.InviteCode == "" {
				return nil, humacore.NewError(http.StatusBadRequest, "invite_required", "code d'invitation requis")
			}
			if err := h.invites.Validate(req.InviteCode); err != nil {
				return nil, humacore.NewError(http.StatusForbidden, "invalid_invite", "code d'invitation invalide ou expiré")
			}
		}
	}

	user, err := h.users.Create(req.Username, req.Password, role)
	if err != nil {
		if errors.Is(err, userstore.ErrUserAlreadyExists) {
			slog.Warn("auth: register username déjà pris", "username", req.Username)
			return nil, humacore.NewError(http.StatusConflict, "user_exists", "nom d'utilisateur déjà pris")
		}
		if errors.Is(err, userstore.ErrInvalidUsername) || errors.Is(err, userstore.ErrPasswordTooShort) || errors.Is(err, userstore.ErrPasswordTooLong) {
			return nil, humacore.NewError(http.StatusBadRequest, "validation_error", err.Error())
		}
		return nil, humacore.NewError(http.StatusInternalServerError, "create_error", "erreur de création")
	}

	// Consommer le code d'invitation si utilisé.
	if req.InviteCode != "" && !empty {
		if err := h.invites.Consume(req.InviteCode, user.Username); err != nil {
			slog.ErrorContext(ctx, "auth: échec consume invite", "code", req.InviteCode, "username", user.Username, "err", err)
		}
	}

	// Auto-login après inscription.
	sess := middleware.GetSession(ctx)
	if sess != nil {
		sess.Username = &user.Username
		roleStr := string(user.Role)
		sess.Role = &roleStr
		if err := h.sessionStore.Save(sess); err != nil {
			slog.ErrorContext(ctx, "auth: échec save session register", "username", user.Username, "err", err)
		}
	}

	slog.Info("auth: inscription réussie", "username", user.Username, "role", user.Role)

	return &registerOutput{Status: http.StatusCreated, Body: domain.RegisterResponse{
		Username: user.Username,
		Role:     user.Role,
	}}, nil
}

// handleLogout déconnecte l'utilisateur.
// POST /auth/logout
func (h *UserAuthHandler) handleLogout(ctx context.Context, _ *struct{}) (*authNoContent, error) {
	sess := middleware.GetSession(ctx)
	if sess != nil {
		username := "<unknown>"
		if sess.Username != nil {
			username = *sess.Username
		}
		sess.Username = nil
		sess.Role = nil
		if err := h.sessionStore.Save(sess); err != nil {
			slog.ErrorContext(ctx, "auth: échec save session logout", "err", err)
		}
		slog.Info("auth: logout", "username", username)
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

	var req domain.SetPasswordRequest
	if err := json.Unmarshal(in.RawBody, &req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "corps de requête invalide")
	}

	if err := h.users.ResetPassword(*sess.Username, req.Password); err != nil {
		switch {
		case errors.Is(err, userstore.ErrPasswordTooShort), errors.Is(err, userstore.ErrPasswordTooLong):
			return nil, humacore.NewError(http.StatusBadRequest, "validation_error", err.Error())
		case errors.Is(err, userstore.ErrUserNotFound):
			return nil, humacore.NewError(http.StatusNotFound, "user_not_found", "utilisateur introuvable")
		default:
			slog.ErrorContext(ctx, "auth: échec set password", "username", *sess.Username, "err", err)
			return nil, humacore.NewError(http.StatusInternalServerError, "set_password_error", "erreur lors de la définition du mot de passe")
		}
	}

	slog.Info("auth: mot de passe défini (opt-in)", "username", *sess.Username)
	return &authNoContent{Status: http.StatusNoContent}, nil
}

// autoSelectPlayer sélectionne automatiquement le joueur lié au gamertag de l'utilisateur.
func (h *UserAuthHandler) autoSelectPlayer(sess *domain.SessionData, user *domain.User) {
	if user.Gamertag == "" || sess == nil {
		return
	}
	slug := user.Gamertag
	sess.CurrentPlayerSlug = &slug
}
