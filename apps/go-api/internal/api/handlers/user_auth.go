// Package handlers — user_auth.go : endpoints auth locale (login/register/logout).
//
// POST /auth/login     → authentification par username/password
// POST /auth/register  → inscription (premier user = admin)
// POST /auth/logout    → déconnexion
package handlers

import (
	"encoding/json"
	"errors"
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

// Login authentifie un utilisateur par username/password.
// POST /auth/login
func (h *UserAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "corps de requête invalide")
		return
	}

	user, err := h.users.Authenticate(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, userstore.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "identifiants incorrects")
			return
		}
		writeError(w, http.StatusInternalServerError, "auth_error", "erreur d'authentification")
		return
	}

	_ = h.users.UpdateLastLogin(user.Username)

	// Stocker les infos dans la session.
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeError(w, http.StatusInternalServerError, "no_session", "session non initialisée")
		return
	}
	sess.Username = &user.Username
	role := string(user.Role)
	sess.Role = &role
	h.autoSelectPlayer(sess, user)
	_ = h.sessionStore.Save(sess)

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
		writeError(w, http.StatusBadRequest, "invalid_body", "corps de requête invalide")
		return
	}

	// Vérifier si c'est le premier utilisateur (auto-admin).
	empty, err := h.users.IsEmpty()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", "erreur interne")
		return
	}

	role := domain.RoleUser
	if empty {
		// Premier utilisateur = admin automatique, pas besoin de code d'invitation.
		role = domain.RoleAdmin
	} else {
		// Vérifier le mode d'inscription.
		if h.regMode == "closed" {
			writeError(w, http.StatusForbidden, "registration_closed", "les inscriptions sont fermées")
			return
		}
		if h.regMode == "invite" {
			if req.InviteCode == "" {
				writeError(w, http.StatusBadRequest, "invite_required", "code d'invitation requis")
				return
			}
			if err := h.invites.Validate(req.InviteCode); err != nil {
				writeError(w, http.StatusForbidden, "invalid_invite", "code d'invitation invalide ou expiré")
				return
			}
		}
	}

	user, err := h.users.Create(req.Username, req.Password, role)
	if err != nil {
		if errors.Is(err, userstore.ErrUserAlreadyExists) {
			writeError(w, http.StatusConflict, "user_exists", "nom d'utilisateur déjà pris")
			return
		}
		if errors.Is(err, userstore.ErrInvalidUsername) || errors.Is(err, userstore.ErrPasswordTooShort) {
			writeError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "create_error", "erreur de création")
		return
	}

	// Consommer le code d'invitation si utilisé.
	if req.InviteCode != "" && !empty {
		_ = h.invites.Consume(req.InviteCode, user.Username)
	}

	// Auto-login après inscription.
	sess := middleware.GetSession(r.Context())
	if sess != nil {
		sess.Username = &user.Username
		roleStr := string(user.Role)
		sess.Role = &roleStr
		_ = h.sessionStore.Save(sess)
	}

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
		sess.Username = nil
		sess.Role = nil
		_ = h.sessionStore.Save(sess)
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
