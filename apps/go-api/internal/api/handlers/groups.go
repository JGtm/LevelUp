// Package handlers — groups.go : gestion des groupes/familles (accès mutuel).
//
// Endpoints user-facing (sous RequireAuth) :
//
//	GET    /groups                     → liste les groupes du user courant
//	POST   /groups                     → crée un groupe (user = propriétaire)
//	PATCH  /groups/{id}                → renomme (propriétaire only)
//	DELETE /groups/{id}                → supprime (propriétaire only)
//	POST   /groups/{id}/invites        → génère une invitation "rejoindre le groupe" (membre only)
//	DELETE /groups/{id}/members/me     → quitte le groupe (self ; propriétaire → 409)
//	DELETE /groups/{id}/members/{xuid} → retire un membre (propriétaire only)
//
// L'autorisation est par xuid : un user sans identité Halo liée (pas de xuid) ne
// peut pas gérer de groupe (401). Title-agnostic (groupes indexés par xuid).
//
// MIGRÉ vers Huma (V72-01 / H5) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (groupe middleware-only RequireAuth, préfixe de document /api/v1) et enregistre
// les 7 routes via huma.*. Logique métier inchangée (GroupStore + InviteStore) ;
// seul le wrapping HTTP change. Le corps est lu via RawBody (décodage maison) pour
// préserver EXACTEMENT le contrat 400 {invalid_body} d'origine (un Body typé
// renverrait le 422 de validation Huma). Contrat JSON runtime identique aux routes
// chi d'origine (mêmes codes, mêmes corps).
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

// Mount enregistre les 7 routes via Huma sur le sous-routeur chi (groupe
// middleware-only RequireAuth). Le body POST /groups/{id}/invites est OPTIONNEL
// (MarkRequestBodyOptional) — corps absent → invitation par défaut.
func (h *GroupsHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/groups", h.handleListMyGroups,
		humacore.Op("listMyGroups", "Liste les groupes du user courant (auth + identité Halo requise)", "groups"))
	huma.Post(api, "/groups", h.handleCreateGroup,
		humacore.Op("createGroup", "Crée un groupe (user courant = propriétaire)", "groups"),
		humacore.DefaultStatus(http.StatusCreated))
	huma.Patch(api, "/groups/{id}", h.handleRenameGroup,
		humacore.Op("renameGroup", "Renomme un groupe (propriétaire only)", "groups"))
	huma.Delete(api, "/groups/{id}", h.handleDeleteGroup,
		humacore.Op("deleteGroup", "Supprime un groupe (propriétaire only)", "groups"))
	huma.Post(api, "/groups/{id}/invites", h.handleGenerateInvite,
		humacore.Op("createGroupInvite", "Génère une invitation 'rejoindre le groupe' (membre only)", "groups"),
		humacore.DefaultStatus(http.StatusCreated))
	humacore.MarkRequestBodyOptional(api, http.MethodPost, "/groups/{id}/invites")
	huma.Delete(api, "/groups/{id}/members/me", h.handleLeaveGroup,
		humacore.Op("leaveGroup", "Quitter un groupe (self ; propriétaire interdit → 409)", "groups"))
	huma.Delete(api, "/groups/{id}/members/{xuid}", h.handleRemoveMember,
		humacore.Op("removeGroupMember", "Retire un membre (propriétaire only)", "groups"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// groupIDInput : path param {id}.
type groupIDInput struct {
	ID string `path:"id"`
}

// groupBodyInput : corps brut décodé maison (400 {invalid_body} sur JSON malformé).
type groupBodyInput struct {
	RawBody []byte
}

// groupIDBodyInput : {id} + corps brut décodé maison.
type groupIDBodyInput struct {
	ID      string `path:"id"`
	RawBody []byte
}

// groupMemberInput : {id} + {xuid}.
type groupMemberInput struct {
	ID   string `path:"id"`
	XUID string `path:"xuid"`
}

type groupsListOutput struct {
	Body []domain.Group
}

// groupCreatedOutput / inviteCreatedOutput : 201 posé par humacore.DefaultStatus
// au montage (source unique du statut — runtime ET document).
type groupCreatedOutput struct {
	Body *domain.Group
}
type groupOutput struct {
	Body *domain.Group
}
type inviteCreatedOutput struct {
	Body *domain.InviteCode
}

// groupsNoContent : réponse 204 sans corps.
type groupsNoContent struct {
	Status int
}

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleListMyGroups retourne les groupes dont le user courant est membre.
// GET /groups
func (h *GroupsHandler) handleListMyGroups(ctx context.Context, _ *struct{}) (*groupsListOutput, error) {
	user, err := h.currentUser(ctx)
	if err != nil {
		return nil, err
	}
	groups, err := h.groups.ListForXUID(user.XUID)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "groups_load_error", "Impossible de charger les groupes.")
	}
	if groups == nil {
		groups = []domain.Group{}
	}
	return &groupsListOutput{Body: groups}, nil
}

// handleCreateGroup crée un groupe avec le user courant comme propriétaire.
// POST /groups
func (h *GroupsHandler) handleCreateGroup(ctx context.Context, in *groupBodyInput) (*groupCreatedOutput, error) {
	user, err := h.currentUser(ctx)
	if err != nil {
		return nil, err
	}
	var req domain.CreateGroupRequest
	if err := json.Unmarshal(in.RawBody, &req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "Corps de requête invalide.")
	}
	g, err := h.groups.Create(req.Name, user.XUID, user.Gamertag)
	if err != nil {
		if errors.Is(err, groupstore.ErrInvalidName) {
			return nil, humacore.NewError(http.StatusBadRequest, "invalid_name", "Nom de groupe invalide (1-60 caractères).")
		}
		return nil, humacore.NewError(http.StatusInternalServerError, "group_create_error", "Impossible de créer le groupe.")
	}
	slog.InfoContext(ctx, "groups: groupe créé", "id", g.ID, "owner", user.Gamertag)
	return &groupCreatedOutput{Body: g}, nil
}

// handleRenameGroup renomme un groupe (propriétaire only).
// PATCH /groups/{id}
func (h *GroupsHandler) handleRenameGroup(ctx context.Context, in *groupIDBodyInput) (*groupOutput, error) {
	user, err := h.currentUser(ctx)
	if err != nil {
		return nil, err
	}
	g, err := h.requireOwnedGroup(in.ID, user)
	if err != nil {
		return nil, err
	}
	var req domain.RenameGroupRequest
	if err := json.Unmarshal(in.RawBody, &req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "Corps de requête invalide.")
	}
	if err := h.groups.Rename(g.ID, req.Name); err != nil {
		if errors.Is(err, groupstore.ErrInvalidName) {
			return nil, humacore.NewError(http.StatusBadRequest, "invalid_name", "Nom de groupe invalide (1-60 caractères).")
		}
		return nil, humacore.NewError(http.StatusInternalServerError, "group_rename_error", "Impossible de renommer le groupe.")
	}
	updated, _ := h.groups.Get(g.ID)
	return &groupOutput{Body: updated}, nil
}

// handleDeleteGroup supprime un groupe (propriétaire only).
// DELETE /groups/{id}
func (h *GroupsHandler) handleDeleteGroup(ctx context.Context, in *groupIDInput) (*groupsNoContent, error) {
	user, err := h.currentUser(ctx)
	if err != nil {
		return nil, err
	}
	g, err := h.requireOwnedGroup(in.ID, user)
	if err != nil {
		return nil, err
	}
	if err := h.groups.Delete(g.ID); err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "group_delete_error", "Impossible de supprimer le groupe.")
	}
	slog.InfoContext(ctx, "groups: groupe supprimé", "id", g.ID, "by", user.Gamertag)
	return &groupsNoContent{Status: http.StatusNoContent}, nil
}

// handleGenerateInvite crée une invitation "rejoindre le groupe" (membre only).
// POST /groups/{id}/invites — corps optionnel {expires_in_days?}.
func (h *GroupsHandler) handleGenerateInvite(ctx context.Context, in *groupIDBodyInput) (*inviteCreatedOutput, error) {
	user, err := h.currentUser(ctx)
	if err != nil {
		return nil, err
	}
	g, err := h.requireMemberGroup(in.ID, user)
	if err != nil {
		return nil, err
	}
	var body struct {
		ExpiresInDays int `json:"expires_in_days"`
	}
	_ = json.Unmarshal(in.RawBody, &body) // corps optionnel (absent → défaut du store)

	invite, err := h.invites.Generate(user.Gamertag, body.ExpiresInDays, g.ID)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "invite_generate_error", "Impossible de générer l'invitation.")
	}
	slog.InfoContext(ctx, "groups: invitation générée", "code", invite.Code, "group_id", g.ID, "by", user.Gamertag)
	return &inviteCreatedOutput{Body: invite}, nil
}

// handleLeaveGroup retire le user courant d'un groupe (self). Le propriétaire ne
// peut pas quitter (il doit supprimer le groupe) → 409.
// DELETE /groups/{id}/members/me
func (h *GroupsHandler) handleLeaveGroup(ctx context.Context, in *groupIDInput) (*groupsNoContent, error) {
	user, err := h.currentUser(ctx)
	if err != nil {
		return nil, err
	}
	g, err := h.loadGroup(in.ID)
	if err != nil {
		return nil, err
	}
	if !g.HasMember(user.XUID) {
		// Déjà non-membre : idempotent.
		return &groupsNoContent{Status: http.StatusNoContent}, nil
	}
	if g.IsOwner(user.XUID) {
		return nil, humacore.NewError(http.StatusConflict, "owner_cannot_leave",
			"Le propriétaire ne peut pas quitter le groupe ; supprimez-le à la place.")
	}
	if err := h.groups.RemoveMember(g.ID, user.XUID); err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "group_leave_error", "Impossible de quitter le groupe.")
	}
	slog.InfoContext(ctx, "groups: membre a quitté", "id", g.ID, "xuid", user.XUID)
	return &groupsNoContent{Status: http.StatusNoContent}, nil
}

// handleRemoveMember retire un membre d'un groupe (propriétaire only). Le
// propriétaire ne peut pas se retirer lui-même (ErrCannotRemoveOwner → 400).
// DELETE /groups/{id}/members/{xuid}
func (h *GroupsHandler) handleRemoveMember(ctx context.Context, in *groupMemberInput) (*groupsNoContent, error) {
	user, err := h.currentUser(ctx)
	if err != nil {
		return nil, err
	}
	g, err := h.requireOwnedGroup(in.ID, user)
	if err != nil {
		return nil, err
	}
	if err := h.groups.RemoveMember(g.ID, in.XUID); err != nil {
		if errors.Is(err, groupstore.ErrCannotRemoveOwner) {
			return nil, humacore.NewError(http.StatusBadRequest, "cannot_remove_owner",
				"Le propriétaire ne peut pas être retiré ; supprimez le groupe.")
		}
		return nil, humacore.NewError(http.StatusInternalServerError, "group_remove_error", "Impossible de retirer le membre.")
	}
	slog.InfoContext(ctx, "groups: membre retiré", "id", g.ID, "xuid", in.XUID, "by", user.Gamertag)
	return &groupsNoContent{Status: http.StatusNoContent}, nil
}

// ─── Helpers d'autorisation ──────────────────────────────────────────────────

// currentUser résout l'utilisateur courant lié à une identité Halo (xuid requis).
// Retourne une erreur 401 si la session n'identifie aucun user lié.
func (h *GroupsHandler) currentUser(ctx context.Context) (*domain.User, error) {
	user := authz.CurrentUser(getSessionFromContext(ctx), h.users)
	if user == nil || user.XUID == "" {
		return nil, humacore.NewError(http.StatusUnauthorized, "identity_required",
			"Une identité Halo liée est requise pour gérer des groupes.")
	}
	return user, nil
}

// loadGroup charge le groupe {id}. Retourne une erreur 404 si absent.
func (h *GroupsHandler) loadGroup(id string) (*domain.Group, error) {
	g, err := h.groups.Get(id)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "group_not_found", "Groupe introuvable.")
	}
	return g, nil
}

// requireOwnedGroup charge le groupe {id} et exige que le user en soit propriétaire
// (404 si absent, 403 sinon).
func (h *GroupsHandler) requireOwnedGroup(id string, user *domain.User) (*domain.Group, error) {
	g, err := h.loadGroup(id)
	if err != nil {
		return nil, err
	}
	if !g.IsOwner(user.XUID) {
		return nil, humacore.NewError(http.StatusForbidden, "group_forbidden", "Seul le propriétaire peut effectuer cette action.")
	}
	return g, nil
}

// requireMemberGroup charge le groupe {id} et exige que le user en soit membre.
func (h *GroupsHandler) requireMemberGroup(id string, user *domain.User) (*domain.Group, error) {
	g, err := h.loadGroup(id)
	if err != nil {
		return nil, err
	}
	if !g.HasMember(user.XUID) {
		return nil, humacore.NewError(http.StatusForbidden, "group_forbidden", "Vous n'êtes pas membre de ce groupe.")
	}
	return g, nil
}
