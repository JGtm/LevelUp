// Package handlers — groups.go : gestion des groupes/familles (accès mutuel).
//
// Endpoints user-facing (sous RequireAuth) :
//
//	GET    /groups               → liste les groupes du user courant
//	POST   /groups               → crée un groupe (user = propriétaire)
//	PATCH  /groups/{id}          → renomme (propriétaire only)
//	DELETE /groups/{id}          → supprime (propriétaire only)
//	POST   /groups/{id}/invites  → génère une invitation "rejoindre le groupe" (membre only)
//
// L'autorisation est par xuid : un user sans identité Halo liée (pas de xuid) ne
// peut pas gérer de groupe (401). Title-agnostic (groupes indexés par xuid).
package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/groupstore"
	"levelup/go-api/internal/platform/userstore"
)

// GroupsHandler sert les endpoints de gestion des groupes.
type GroupsHandler struct {
	groups  *groupstore.GroupStore
	invites *userstore.InviteStore
	users   authz.UserLookup
}

// NewGroupsHandler crée un GroupsHandler.
func NewGroupsHandler(groups *groupstore.GroupStore, invites *userstore.InviteStore, users authz.UserLookup) *GroupsHandler {
	return &GroupsHandler{groups: groups, invites: invites, users: users}
}

// currentUser résout l'utilisateur courant lié à une identité Halo (xuid requis).
// Écrit une réponse 401 et retourne nil si la session n'identifie aucun user lié.
func (h *GroupsHandler) currentUser(w http.ResponseWriter, r *http.Request) *domain.User {
	user := authz.CurrentUser(getSessionFromRequest(r), h.users)
	if user == nil || user.XUID == "" {
		writeError(r.Context(), w, http.StatusUnauthorized, "identity_required",
			"Une identité Halo liée est requise pour gérer des groupes.")
		return nil
	}
	return user
}

// ListMyGroups retourne les groupes dont le user courant est membre.
// GET /groups
func (h *GroupsHandler) ListMyGroups(w http.ResponseWriter, r *http.Request) {
	user := h.currentUser(w, r)
	if user == nil {
		return
	}
	groups, err := h.groups.ListForXUID(user.XUID)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "groups_load_error", "Impossible de charger les groupes.")
		return
	}
	if groups == nil {
		groups = []domain.Group{}
	}
	writeJSON(w, http.StatusOK, groups)
}

// CreateGroup crée un groupe avec le user courant comme propriétaire.
// POST /groups
func (h *GroupsHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	user := h.currentUser(w, r)
	if user == nil {
		return
	}
	var req domain.CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", "Corps de requête invalide.")
		return
	}
	g, err := h.groups.Create(req.Name, user.XUID, user.Gamertag)
	if err != nil {
		if errors.Is(err, groupstore.ErrInvalidName) {
			writeError(r.Context(), w, http.StatusBadRequest, "invalid_name", "Nom de groupe invalide (1-60 caractères).")
			return
		}
		writeError(r.Context(), w, http.StatusInternalServerError, "group_create_error", "Impossible de créer le groupe.")
		return
	}
	slog.InfoContext(r.Context(), "groups: groupe créé", "id", g.ID, "owner", user.Gamertag)
	writeJSON(w, http.StatusCreated, g)
}

// RenameGroup renomme un groupe (propriétaire only).
// PATCH /groups/{id}
func (h *GroupsHandler) RenameGroup(w http.ResponseWriter, r *http.Request) {
	user := h.currentUser(w, r)
	if user == nil {
		return
	}
	g := h.requireOwnedGroup(w, r, user)
	if g == nil {
		return
	}
	var req domain.RenameGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", "Corps de requête invalide.")
		return
	}
	if err := h.groups.Rename(g.ID, req.Name); err != nil {
		if errors.Is(err, groupstore.ErrInvalidName) {
			writeError(r.Context(), w, http.StatusBadRequest, "invalid_name", "Nom de groupe invalide (1-60 caractères).")
			return
		}
		writeError(r.Context(), w, http.StatusInternalServerError, "group_rename_error", "Impossible de renommer le groupe.")
		return
	}
	updated, _ := h.groups.Get(g.ID)
	writeJSON(w, http.StatusOK, updated)
}

// DeleteGroup supprime un groupe (propriétaire only).
// DELETE /groups/{id}
func (h *GroupsHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	user := h.currentUser(w, r)
	if user == nil {
		return
	}
	g := h.requireOwnedGroup(w, r, user)
	if g == nil {
		return
	}
	if err := h.groups.Delete(g.ID); err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "group_delete_error", "Impossible de supprimer le groupe.")
		return
	}
	slog.InfoContext(r.Context(), "groups: groupe supprimé", "id", g.ID, "by", user.Gamertag)
	w.WriteHeader(http.StatusNoContent)
}

// GenerateInvite crée une invitation "rejoindre le groupe" (membre only).
// POST /groups/{id}/invites
func (h *GroupsHandler) GenerateInvite(w http.ResponseWriter, r *http.Request) {
	user := h.currentUser(w, r)
	if user == nil {
		return
	}
	g := h.requireMemberGroup(w, r, user)
	if g == nil {
		return
	}
	var body struct {
		ExpiresInDays int `json:"expires_in_days"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body) // corps optionnel

	invite, err := h.invites.Generate(user.Gamertag, body.ExpiresInDays, g.ID)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "invite_generate_error", "Impossible de générer l'invitation.")
		return
	}
	slog.InfoContext(r.Context(), "groups: invitation générée", "code", invite.Code, "group_id", g.ID, "by", user.Gamertag)
	writeJSON(w, http.StatusCreated, invite)
}

// requireOwnedGroup charge le groupe {id} et exige que le user en soit propriétaire.
// Écrit 404/403 et retourne nil sinon.
func (h *GroupsHandler) requireOwnedGroup(w http.ResponseWriter, r *http.Request, user *domain.User) *domain.Group {
	g := h.loadGroup(w, r)
	if g == nil {
		return nil
	}
	if !g.IsOwner(user.XUID) {
		writeError(r.Context(), w, http.StatusForbidden, "group_forbidden", "Seul le propriétaire peut effectuer cette action.")
		return nil
	}
	return g
}

// requireMemberGroup charge le groupe {id} et exige que le user en soit membre.
func (h *GroupsHandler) requireMemberGroup(w http.ResponseWriter, r *http.Request, user *domain.User) *domain.Group {
	g := h.loadGroup(w, r)
	if g == nil {
		return nil
	}
	if !g.HasMember(user.XUID) {
		writeError(r.Context(), w, http.StatusForbidden, "group_forbidden", "Vous n'êtes pas membre de ce groupe.")
		return nil
	}
	return g
}

// loadGroup charge le groupe {id} de l'URL. Écrit 404 et retourne nil si absent.
func (h *GroupsHandler) loadGroup(w http.ResponseWriter, r *http.Request) *domain.Group {
	id := chi.URLParam(r, "id")
	g, err := h.groups.Get(id)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "group_not_found", "Groupe introuvable.")
		return nil
	}
	return g
}
