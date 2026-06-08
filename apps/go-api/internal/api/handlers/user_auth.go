// Package handlers — user_auth.go : endpoints auth locale (login/register/logout).
//
// POST /auth/login     → authentification par username/password
// POST /auth/register  → inscription (premier user = admin)
// POST /auth/logout    → déconnexion
package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

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

// Login authentifie un utilisateur par username/password.
// POST /auth/login
func (h *UserAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", "corps de requête invalide")
		return
	}

	user, err := h.users.Authenticate(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, userstore.ErrInvalidCredentials) {
			slog.Warn("auth: login échoué", "username", req.Username, "ip", r.RemoteAddr)
			writeError(r.Context(), w, http.StatusUnauthorized, "invalid_credentials", "identifiants incorrects")
			return
		}
		slog.Error("auth: erreur authenticate", "username", req.Username, "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "auth_error", "erreur d'authentification")
		return
	}

	// D3 cohabitation : en mode "xbox", le login password est réservé aux admins
	// (fallback si SSO down). Les users normaux passent obligatoirement par le SSO Xbox.
	if h.authMode == authModeXbox && user.Role != domain.RoleAdmin {
		slog.Warn("auth: login password non-admin bloqué en mode xbox",
			"username", user.Username, "role", user.Role)
		writeError(r.Context(), w, http.StatusForbidden, "password_login_admin_only",
			"mode SSO Xbox actif : login password réservé aux admins")
		return
	}

	if err := h.users.UpdateLastLogin(user.Username); err != nil {
		slog.Warn("auth: échec UpdateLastLogin", "username", user.Username, "err", err)
	}

	// Stocker les infos dans la session.
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "no_session", "session non initialisée")
		return
	}
	sess.Username = &user.Username
	role := string(user.Role)
	sess.Role = &role
	h.autoSelectPlayer(sess, user)
	if err := h.sessionStore.Save(sess); err != nil {
		slog.Error("auth: échec save session login", "username", user.Username, "err", err)
	}

	slog.Info("auth: login réussi", "username", user.Username, "role", user.Role)

	writeJSON(w, http.StatusOK, domain.LoginResponse{
		Username: user.Username,
		Role:     user.Role,
		Gamertag: user.Gamertag,
	})
}

// Register inscrit un nouvel utilisateur.
// POST /auth/register
func (h *UserAuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", "corps de requête invalide")
		return
	}

	// Vérifier si c'est le premier utilisateur (auto-admin).
	empty, err := h.users.IsEmpty()
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "store_error", "erreur interne")
		return
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
			writeError(r.Context(), w, http.StatusForbidden, "instance_locked",
				"Cette instance est fermée aux nouvelles inscriptions.")
			return
		}
		// D3 cohabitation : en mode "xbox", register password réservé au bootstrap admin
		// initial (users.json vide). Hors bootstrap, les comptes sont créés via le flow SSO.
		if h.authMode == "xbox" {
			slog.Warn("auth: register password bloqué en mode xbox", "username", req.Username)
			writeError(r.Context(), w, http.StatusForbidden, "register_xbox_mode",
				"mode SSO Xbox actif : les nouveaux comptes sont créés via la connexion Xbox")
			return
		}
		// Vérifier le mode d'inscription.
		if h.regMode == "closed" {
			writeError(r.Context(), w, http.StatusForbidden, "registration_closed", "les inscriptions sont fermées")
			return
		}
		if h.regMode == "invite" {
			if req.InviteCode == "" {
				writeError(r.Context(), w, http.StatusBadRequest, "invite_required", "code d'invitation requis")
				return
			}
			if err := h.invites.Validate(req.InviteCode); err != nil {
				writeError(r.Context(), w, http.StatusForbidden, "invalid_invite", "code d'invitation invalide ou expiré")
				return
			}
		}
	}

	user, err := h.users.Create(req.Username, req.Password, role)
	if err != nil {
		if errors.Is(err, userstore.ErrUserAlreadyExists) {
			slog.Warn("auth: register username déjà pris", "username", req.Username)
			writeError(r.Context(), w, http.StatusConflict, "user_exists", "nom d'utilisateur déjà pris")
			return
		}
		if errors.Is(err, userstore.ErrInvalidUsername) || errors.Is(err, userstore.ErrPasswordTooShort) {
			writeError(r.Context(), w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		writeError(r.Context(), w, http.StatusInternalServerError, "create_error", "erreur de création")
		return
	}

	// Consommer le code d'invitation si utilisé.
	if req.InviteCode != "" && !empty {
		if err := h.invites.Consume(req.InviteCode, user.Username); err != nil {
			slog.Error("auth: échec consume invite", "code", req.InviteCode, "username", user.Username, "err", err)
		}
	}

	// Auto-login après inscription.
	sess := middleware.GetSession(r.Context())
	if sess != nil {
		sess.Username = &user.Username
		roleStr := string(user.Role)
		sess.Role = &roleStr
		if err := h.sessionStore.Save(sess); err != nil {
			slog.Error("auth: échec save session register", "username", user.Username, "err", err)
		}
	}

	slog.Info("auth: inscription réussie", "username", user.Username, "role", user.Role)

	writeJSON(w, http.StatusCreated, domain.RegisterResponse{
		Username: user.Username,
		Role:     user.Role,
	})
}

// Logout déconnecte l'utilisateur.
// POST /auth/logout
func (h *UserAuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess != nil {
		username := "<unknown>"
		if sess.Username != nil {
			username = *sess.Username
		}
		sess.Username = nil
		sess.Role = nil
		if err := h.sessionStore.Save(sess); err != nil {
			slog.Error("auth: échec save session logout", "err", err)
		}
		slog.Info("auth: logout", "username", username)
	}
	w.WriteHeader(http.StatusNoContent)
}

// autoSelectPlayer sélectionne automatiquement le joueur lié au gamertag de l'utilisateur.
func (h *UserAuthHandler) autoSelectPlayer(sess *domain.SessionData, user *domain.User) {
	if user.Gamertag == "" || sess == nil {
		return
	}
	slug := user.Gamertag
	sess.CurrentPlayerSlug = &slug
}
