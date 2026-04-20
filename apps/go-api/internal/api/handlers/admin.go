// Package handlers — admin.go : endpoints d'administration (utilisateurs + invitations).
//
// GET    /admin/users          → liste des utilisateurs
// DELETE /admin/users/{username} → suppression d'un utilisateur
// PATCH  /admin/users/{username}/role → changer le rôle
// PATCH  /admin/users/{username}/password → reset mot de passe
// GET    /admin/invites        → liste des invitations
// POST   /admin/invites        → générer une invitation
// DELETE /admin/invites/{code}  → révoquer une invitation
package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/userstore"
)

// AdminHandler gère les endpoints d'administration.
type AdminHandler struct {
	users   *userstore.Store
	invites *userstore.InviteStore
}

// NewAdminHandler crée un AdminHandler.
func NewAdminHandler(users *userstore.Store, invites *userstore.InviteStore) *AdminHandler {
	return &AdminHandler{users: users, invites: invites}
}

// ListUsers retourne la liste des utilisateurs.
// GET /admin/users
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.users.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list_error", "erreur de récupération")
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// DeleteUser supprime un utilisateur.
// DELETE /admin/users/{username}
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if err := h.users.Delete(username); err != nil {
		if errors.Is(err, userstore.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "utilisateur introuvable")
			return
		}
		slog.Error("admin: erreur delete user", "target", username, "err", err)
		writeError(w, http.StatusInternalServerError, "delete_error", "erreur de suppression")
		return
	}
	slog.Info("admin: utilisateur supprimé", "target", username, "by", adminUsername(r))
	w.WriteHeader(http.StatusNoContent)
}

// ChangeRole modifie le rôle d'un utilisateur.
// PATCH /admin/users/{username}/role
func (h *AdminHandler) ChangeRole(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	var body struct {
		Role domain.UserRole `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "corps de requête invalide")
		return
	}
	if body.Role != domain.RoleAdmin && body.Role != domain.RoleUser {
		writeError(w, http.StatusBadRequest, "invalid_role", "rôle invalide (admin ou user)")
		return
	}
	if err := h.users.SetRole(username, body.Role); err != nil {
		if errors.Is(err, userstore.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "utilisateur introuvable")
			return
		}
		slog.Error("admin: erreur change role", "target", username, "role", body.Role, "err", err)
		writeError(w, http.StatusInternalServerError, "role_error", "erreur de modification")
		return
	}
	slog.Info("admin: rôle modifié", "target", username, "role", body.Role, "by", adminUsername(r))
	w.WriteHeader(http.StatusNoContent)
}

// ResetPassword réinitialise le mot de passe d'un utilisateur.
// PATCH /admin/users/{username}/password
func (h *AdminHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	var body struct {
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "corps de requête invalide")
		return
	}
	if err := h.users.ResetPassword(username, body.NewPassword); err != nil {
		if errors.Is(err, userstore.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "utilisateur introuvable")
			return
		}
		if errors.Is(err, userstore.ErrPasswordTooShort) {
			writeError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		slog.Error("admin: erreur reset password", "target", username, "err", err)
		writeError(w, http.StatusInternalServerError, "reset_error", "erreur de réinitialisation")
		return
	}
	slog.Info("admin: mot de passe réinitialisé", "target", username, "by", adminUsername(r))
	w.WriteHeader(http.StatusNoContent)
}

// ListInvites retourne la liste des invitations.
// GET /admin/invites
func (h *AdminHandler) ListInvites(w http.ResponseWriter, r *http.Request) {
	invites, err := h.invites.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list_error", "erreur de récupération")
		return
	}
	writeJSON(w, http.StatusOK, invites)
}

// GenerateInvite crée un nouveau code d'invitation.
// POST /admin/invites
func (h *AdminHandler) GenerateInvite(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ExpiresInDays int `json:"expires_in_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		// Body optionnel — défaut 7 jours.
		body.ExpiresInDays = 7
	}
	if body.ExpiresInDays <= 0 {
		body.ExpiresInDays = 7
	}

	// Récupérer le username de l'admin depuis la session.
	sess := getSessionFromRequest(r)
	createdBy := "admin"
	if sess != nil && sess.Username != nil {
		createdBy = *sess.Username
	}

	invite, err := h.invites.Generate(createdBy, body.ExpiresInDays)
	if err != nil {
		slog.Error("admin: erreur generate invite", "by", createdBy, "err", err)
		writeError(w, http.StatusInternalServerError, "generate_error", "erreur de génération")
		return
	}
	slog.Info("admin: invitation générée", "code", invite.Code, "by", createdBy, "expires_in_days", body.ExpiresInDays)
	writeJSON(w, http.StatusCreated, invite)
}

// RevokeInvite révoque un code d'invitation.
// DELETE /admin/invites/{code}
func (h *AdminHandler) RevokeInvite(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if err := h.invites.Revoke(code); err != nil {
		if errors.Is(err, userstore.ErrInviteNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "invitation introuvable")
			return
		}
		slog.Error("admin: erreur revoke invite", "code", code, "err", err)
		writeError(w, http.StatusInternalServerError, "revoke_error", "erreur de révocation")
		return
	}
	slog.Info("admin: invitation révoquée", "code", code, "by", adminUsername(r))
	w.WriteHeader(http.StatusNoContent)
}

// getSessionFromRequest est un helper pour accéder à la session depuis le contexte.
func getSessionFromRequest(r *http.Request) *domain.SessionData {
	return middleware.GetSession(r.Context())
}

// adminUsername extrait le username admin depuis la session de la requête.
func adminUsername(r *http.Request) string {
	sess := getSessionFromRequest(r)
	if sess != nil && sess.Username != nil {
		return *sess.Username
	}
	return "unknown"
}
