// Package handlers — admin.go : endpoints d'administration (utilisateurs + invitations).
//
// GET    /admin/users          → liste des utilisateurs
// DELETE /admin/users/{username} → suppression d'un utilisateur
// PATCH  /admin/users/{username}/role → changer le rôle
// PATCH  /admin/users/{username}/password → reset mot de passe
// GET    /admin/invites        → liste des invitations
// POST   /admin/invites        → générer une invitation
// DELETE /admin/invites/{code}  → révoquer une invitation
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// /admin (middleware RequireAuth/RequireAdmin hérités) et enregistre les 7 routes
// via huma.*. Logique métier inchangée (userstore.Store + InviteStore), seul le
// wrapping HTTP change. Les chemins relatifs sont identiques aux routes chi
// d'origine (montées sous /admin par server.go).
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

// Mount enregistre les 7 routes via Huma sur le sous-routeur chi (préfixe /admin
// + middleware RequireAuth/RequireAdmin hérités). Le body POST /invites est
// OPTIONNEL (MarkRequestBodyOptional) — corps absent → défaut 7 jours.
func (h *AdminHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/users", h.handleListUsers)
	huma.Delete(api, "/users/{username}", h.handleDeleteUser)
	huma.Patch(api, "/users/{username}/role", h.handleChangeRole)
	huma.Patch(api, "/users/{username}/password", h.handleResetPassword)
	huma.Get(api, "/invites", h.handleListInvites)
	huma.Post(api, "/invites", h.handleGenerateInvite)
	humacore.MarkRequestBodyOptional(api, http.MethodPost, "/invites")
	huma.Delete(api, "/invites/{code}", h.handleRevokeInvite)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// adminUsernameInput : path param {username}.
type adminUsernameInput struct {
	Username string `path:"username"`
}

// adminUsernameBodyInput : {username} + body brut (décodage maison → 400
// invalid_body si JSON malformé, contrat préservé des PATCH role/password).
type adminUsernameBodyInput struct {
	Username string `path:"username"`
	RawBody  []byte
}

// adminCodeInput : path param {code}.
type adminCodeInput struct {
	Code string `path:"code"`
}

// adminGenerateInviteInput : body OPTIONNEL (RawBody + MarkRequestBodyOptional)
// — décodage best-effort, défaut 7 jours si absent ou malformé.
type adminGenerateInviteInput struct {
	RawBody []byte
}

type adminListUsersOutput struct {
	Body []domain.AdminUserSummary
}
type adminListInvitesOutput struct {
	Body []domain.AdminInviteSummary
}
type adminGenerateInviteOutput struct {
	Status int
	Body   *domain.InviteCode
}

// adminNoContent : réponse 204 sans corps.
type adminNoContent struct {
	Status int
}

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleListUsers retourne la liste des utilisateurs.
// GET /admin/users
func (h *AdminHandler) handleListUsers(ctx context.Context, _ *struct{}) (*adminListUsersOutput, error) {
	users, err := h.users.List()
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "list_error", "erreur de récupération")
	}
	return &adminListUsersOutput{Body: users}, nil
}

// handleDeleteUser supprime un utilisateur.
// DELETE /admin/users/{username}
func (h *AdminHandler) handleDeleteUser(ctx context.Context, in *adminUsernameInput) (*adminNoContent, error) {
	if err := h.users.Delete(in.Username); err != nil {
		if errors.Is(err, userstore.ErrUserNotFound) {
			return nil, humacore.NewError(http.StatusNotFound, "not_found", "utilisateur introuvable")
		}
		slog.ErrorContext(ctx, "admin: erreur delete user", "target", in.Username, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "delete_error", "erreur de suppression")
	}
	slog.Info("admin: utilisateur supprimé", "target", in.Username, "by", adminUsername(ctx))
	return &adminNoContent{Status: http.StatusNoContent}, nil
}

// handleChangeRole modifie le rôle d'un utilisateur.
// PATCH /admin/users/{username}/role
func (h *AdminHandler) handleChangeRole(ctx context.Context, in *adminUsernameBodyInput) (*adminNoContent, error) {
	var body struct {
		Role domain.UserRole `json:"role"`
	}
	if err := json.Unmarshal(in.RawBody, &body); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "corps de requête invalide")
	}
	if body.Role != domain.RoleAdmin && body.Role != domain.RoleUser {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_role", "rôle invalide (admin ou user)")
	}
	if err := h.users.SetRole(in.Username, body.Role); err != nil {
		if errors.Is(err, userstore.ErrUserNotFound) {
			return nil, humacore.NewError(http.StatusNotFound, "not_found", "utilisateur introuvable")
		}
		slog.ErrorContext(ctx, "admin: erreur change role", "target", in.Username, "role", body.Role, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "role_error", "erreur de modification")
	}
	slog.Info("admin: rôle modifié", "target", in.Username, "role", body.Role, "by", adminUsername(ctx))
	return &adminNoContent{Status: http.StatusNoContent}, nil
}

// handleResetPassword réinitialise le mot de passe d'un utilisateur.
// PATCH /admin/users/{username}/password
func (h *AdminHandler) handleResetPassword(ctx context.Context, in *adminUsernameBodyInput) (*adminNoContent, error) {
	var body struct {
		NewPassword string `json:"new_password"`
	}
	if err := json.Unmarshal(in.RawBody, &body); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "corps de requête invalide")
	}
	if err := h.users.ResetPassword(in.Username, body.NewPassword); err != nil {
		if errors.Is(err, userstore.ErrUserNotFound) {
			return nil, humacore.NewError(http.StatusNotFound, "not_found", "utilisateur introuvable")
		}
		if errors.Is(err, userstore.ErrPasswordTooShort) {
			return nil, humacore.NewError(http.StatusBadRequest, "validation_error", err.Error())
		}
		slog.ErrorContext(ctx, "admin: erreur reset password", "target", in.Username, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "reset_error", "erreur de réinitialisation")
	}
	slog.Info("admin: mot de passe réinitialisé", "target", in.Username, "by", adminUsername(ctx))
	return &adminNoContent{Status: http.StatusNoContent}, nil
}

// handleListInvites retourne la liste des invitations.
// GET /admin/invites
func (h *AdminHandler) handleListInvites(ctx context.Context, _ *struct{}) (*adminListInvitesOutput, error) {
	invites, err := h.invites.List()
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "list_error", "erreur de récupération")
	}
	return &adminListInvitesOutput{Body: invites}, nil
}

// handleGenerateInvite crée un nouveau code d'invitation.
// POST /admin/invites — body optionnel {expires_in_days?}.
func (h *AdminHandler) handleGenerateInvite(ctx context.Context, in *adminGenerateInviteInput) (*adminGenerateInviteOutput, error) {
	var body struct {
		ExpiresInDays int `json:"expires_in_days"`
	}
	if err := json.Unmarshal(in.RawBody, &body); err != nil {
		// Body optionnel — défaut 7 jours.
		body.ExpiresInDays = 7
	}
	if body.ExpiresInDays <= 0 {
		body.ExpiresInDays = 7
	}

	// Récupérer le username de l'admin depuis la session.
	sess := getSessionFromContext(ctx)
	createdBy := "admin"
	if sess != nil && sess.Username != nil {
		createdBy = *sess.Username
	}

	// Invitation admin legacy (inscription mot de passe) : pas de groupe associé.
	invite, err := h.invites.Generate(createdBy, body.ExpiresInDays, "")
	if err != nil {
		slog.ErrorContext(ctx, "admin: erreur generate invite", "by", createdBy, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "generate_error", "erreur de génération")
	}
	slog.Info("admin: invitation générée", "code", invite.Code, "by", createdBy, "expires_in_days", body.ExpiresInDays)
	return &adminGenerateInviteOutput{Status: http.StatusCreated, Body: invite}, nil
}

// handleRevokeInvite révoque un code d'invitation.
// DELETE /admin/invites/{code}
func (h *AdminHandler) handleRevokeInvite(ctx context.Context, in *adminCodeInput) (*adminNoContent, error) {
	if err := h.invites.Revoke(in.Code); err != nil {
		if errors.Is(err, userstore.ErrInviteNotFound) {
			return nil, humacore.NewError(http.StatusNotFound, "not_found", "invitation introuvable")
		}
		slog.ErrorContext(ctx, "admin: erreur revoke invite", "code", in.Code, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "revoke_error", "erreur de révocation")
	}
	slog.Info("admin: invitation révoquée", "code", in.Code, "by", adminUsername(ctx))
	return &adminNoContent{Status: http.StatusNoContent}, nil
}

// getSessionFromContext est un helper pour accéder à la session depuis le contexte.
func getSessionFromContext(ctx context.Context) *domain.SessionData {
	return middleware.GetSession(ctx)
}

// adminUsername extrait le username admin depuis la session du contexte.
func adminUsername(ctx context.Context) string {
	sess := getSessionFromContext(ctx)
	if sess != nil && sess.Username != nil {
		return *sess.Username
	}
	return "unknown"
}
