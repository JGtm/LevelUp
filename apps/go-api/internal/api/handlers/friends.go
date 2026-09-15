// Package handlers — friends.go : la liste d'amis D'UN PROFIL JOUEUR.
//
// Endpoints, montés sous le groupe player-scoped `/players/{player_slug}` (donc
// derrière `RequirePlayerOwnership`, ADR 0029) :
//
//	GET /players/{slug}/friends → PlayerFriends (+ can_edit)
//	PUT /players/{slug}/friends → PlayerFriends (remplacement COMPLET)
//
// Deux portes distinctes, et elles ne disent pas la même chose :
//   - le middleware d'ownership répond « ce profil t'est-il accessible ? » (403
//     player_forbidden sinon) — un co-membre de groupe passe ;
//   - `can_edit` ci-dessous répond « peux-tu l'écrire ? » (D4 : propriétaire
//     direct ou admin). Un co-membre LIT, il n'écrit pas → 403 friends_forbidden.
//
// `can_edit` est porté par la RÉPONSE du GET : c'est la seule source du mode
// lecture seule côté front, qui n'a jamais à interpréter un 403 pour décider de
// son affichage.
package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/friendstore"
)

// FriendsRecomputer relance le recompute `is_with_friends` d'un joueur après
// l'écriture de sa liste. Implémenté par *service.FriendsOrchestratorService.
// Retourne le nombre de matchs promus (0 → aucune notification à émettre).
type FriendsRecomputer interface {
	RecomputeForPlayer(ctx context.Context, xuid string) (int64, error)
}

// PlayerXUIDResolver mappe un slug joueur vers le xuid de son profil, sans
// ouvrir de DuckDB (même résolution que le middleware d'ownership).
type PlayerXUIDResolver func(ctx context.Context, slug string) (string, bool)

// FriendsHandler sert la liste d'amis d'un profil joueur.
type FriendsHandler struct {
	friends      *friendstore.FriendStore
	users        authz.UserLookup
	resolveXUID  PlayerXUIDResolver
	resolveGT    func(ctx context.Context, slug string) (string, bool)
	demoMode     bool
	authMode     string
	recomputer   FriendsRecomputer           // nil → pas de recompute (dégradation silencieuse assumée : le CLI reste)
	notifierFor  NotificationsEmitterFactory // nil → pas de notif friend_added
	settingsPath string                      // pour la config Discord des notifications
}

// NewFriendsHandler crée un FriendsHandler. `resolveXUID` et `resolveGT`
// résolvent le profil depuis db_profiles.json (xuid pour la clé de stockage,
// gamertag pour s'exclure de sa propre liste).
func NewFriendsHandler(
	friends *friendstore.FriendStore,
	users authz.UserLookup,
	resolveXUID PlayerXUIDResolver,
	resolveGT func(ctx context.Context, slug string) (string, bool),
	demoMode bool,
	authMode string,
) *FriendsHandler {
	return &FriendsHandler{
		friends:     friends,
		users:       users,
		resolveXUID: resolveXUID,
		resolveGT:   resolveGT,
		demoMode:    demoMode,
		authMode:    authMode,
	}
}

// WithRecomputer branche l'orchestrateur is_with_friends déclenché par le PUT.
func (h *FriendsHandler) WithRecomputer(r FriendsRecomputer) *FriendsHandler {
	h.recomputer = r
	return h
}

// WithNotifications branche la factory d'émetteurs (notif friend_added) et le
// chemin d'app_settings.json (webhook Discord).
func (h *FriendsHandler) WithNotifications(f NotificationsEmitterFactory, appSettingsPath string) *FriendsHandler {
	h.notifierFor = f
	h.settingsPath = appSettingsPath
	return h
}

// Mount enregistre les 2 routes via Huma sur le sous-routeur player-scoped.
func (h *FriendsHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/friends", h.handleGetFriends,
		humacore.Op("getPlayerFriends", "Liste d'amis du joueur (lecture : propriétaire, co-membre de groupe ou admin)", "friends"))
	huma.Put(api, "/friends", h.handlePutFriends,
		humacore.Op("putPlayerFriends", "Remplace la liste d'amis du joueur (propriétaire direct ou admin)", "friends"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// friendsInput : path param parent {player_slug}.
type friendsInput struct {
	PlayerSlug string `path:"player_slug"`
}

// friendsBodyInput : {player_slug} + corps brut décodé maison (400 invalid_body
// sur JSON malformé, plutôt que le 422 de validation Huma).
type friendsBodyInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}

type friendsOutput struct {
	Body domain.PlayerFriends
}

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleGetFriends retourne la liste d'amis du joueur.
// GET /players/{slug}/friends
func (h *FriendsHandler) handleGetFriends(ctx context.Context, in *friendsInput) (*friendsOutput, error) {
	xuid, err := h.playerXUID(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	gamertags, err := h.friends.Get(xuid)
	if err != nil {
		slog.ErrorContext(ctx, "friends: lecture de la liste échouée", "player_slug", in.PlayerSlug, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "friends_load_error",
			"Impossible de charger la liste d'amis.")
	}
	updatedAt, err := h.friends.UpdatedAt(xuid)
	if err != nil {
		// La liste est déjà lue : l'horodatage n'est qu'un ornement, mais son
		// échec ne doit pas passer sous silence.
		slog.WarnContext(ctx, "friends: horodatage illisible", "player_slug", in.PlayerSlug, "err", err)
	}
	return &friendsOutput{Body: domain.PlayerFriends{
		XUID:      xuid,
		Gamertags: gamertags,
		UpdatedAt: updatedAt,
		CanEdit:   h.canEdit(ctx, xuid),
	}}, nil
}

// handlePutFriends remplace la liste d'amis du joueur (propriétaire direct ou
// admin — D4). PUT /players/{slug}/friends
func (h *FriendsHandler) handlePutFriends(ctx context.Context, in *friendsBodyInput) (*friendsOutput, error) {
	xuid, err := h.playerXUID(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	if !h.canEdit(ctx, xuid) {
		return nil, humacore.NewError(http.StatusForbidden, "friends_forbidden",
			"Seul le propriétaire du profil (ou un administrateur) peut modifier sa liste d'amis.")
	}

	var req domain.PutFriendsRequest
	if err := json.Unmarshal(in.RawBody, &req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "Corps de requête JSON invalide.")
	}
	ownGamertag, _ := h.resolveGT(ctx, in.PlayerSlug)
	normalized := domain.NormalizeFriendGamertags(req.Gamertags, ownGamertag)
	if reason := domain.ValidateFriendGamertags(normalized); reason != "" {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_friends", friendsRejectMessage(reason))
	}

	previous, err := h.friends.Get(xuid)
	if err != nil {
		slog.ErrorContext(ctx, "friends: lecture avant écriture échouée", "player_slug", in.PlayerSlug, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "friends_load_error",
			"Impossible de charger la liste d'amis.")
	}
	saved, err := h.friends.Set(xuid, ownGamertag, normalized)
	if err != nil {
		slog.ErrorContext(ctx, "friends: écriture de la liste échouée", "player_slug", in.PlayerSlug, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "friends_save_error",
			"Impossible d'enregistrer la liste d'amis.")
	}
	slog.InfoContext(ctx, "friends: liste mise à jour", "player_slug", in.PlayerSlug, "count", len(saved))

	if added := newFriendsAdded(previous, saved); len(added) > 0 {
		emitFriendsAdded(ctx, h.notifierFor, h.settingsPath, in.PlayerSlug, "friends_handler", added)
	}
	if friendGamertagsChanged(previous, saved) {
		h.recomputeInBackground(in.PlayerSlug, xuid)
	}

	return &friendsOutput{Body: domain.PlayerFriends{
		XUID:      xuid,
		Gamertags: saved,
		UpdatedAt: h.updatedAtBestEffort(ctx, xuid),
		CanEdit:   true,
	}}, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// playerXUID résout le slug → xuid du profil. Slug inconnu → 404
// player_not_found (le middleware d'ownership laisse passer un slug inconnu) ;
// profil sans xuid → 409 : il n'a pas d'identité à laquelle rattacher des amis.
func (h *FriendsHandler) playerXUID(ctx context.Context, slug string) (string, error) {
	xuid, found := h.resolveXUID(ctx, slug)
	if !found {
		return "", humacore.NewError(http.StatusNotFound, "player_not_found", "Joueur introuvable.")
	}
	if xuid == "" {
		return "", humacore.NewError(http.StatusConflict, "player_without_xuid",
			"Ce profil n'a pas de XUID : impossible de lui attacher une liste d'amis.")
	}
	return xuid, nil
}

// canEdit implémente D4 : propriétaire DIRECT du profil (son xuid lié) ou admin.
// Enforcement désactivé (démo / auth_mode=none) → écriture libre, cohérent avec
// authz.Enforced : sans identités, l'instance entière est à son opérateur.
func (h *FriendsHandler) canEdit(ctx context.Context, playerXUID string) bool {
	if !authz.Enforced(h.demoMode, h.authMode) {
		return true
	}
	user := authz.CurrentUser(getSessionFromContext(ctx), h.users)
	if user == nil {
		return false
	}
	if user.Role == domain.RoleAdmin {
		return true
	}
	return user.XUID != "" && user.XUID == playerXUID
}

// recomputeInBackground relance le recompute is_with_friends du joueur hors du
// cycle de la requête. Erreur journalisée, jamais avalée.
func (h *FriendsHandler) recomputeInBackground(slug, xuid string) {
	if h.recomputer == nil {
		return
	}
	go func() {
		bgCtx := context.Background()
		promoted, err := h.recomputer.RecomputeForPlayer(bgCtx, xuid)
		if err != nil {
			slog.ErrorContext(bgCtx, "friends: recompute is_with_friends échoué", "player", slug, "err", err)
			return
		}
		slog.InfoContext(bgCtx, "friends: recompute is_with_friends", "player", slug, "added", promoted)
	}()
}

// updatedAtBestEffort relit l'horodatage posé par l'écriture ; échec → vide.
func (h *FriendsHandler) updatedAtBestEffort(ctx context.Context, xuid string) string {
	ts, err := h.friends.UpdatedAt(xuid)
	if err != nil {
		slog.WarnContext(ctx, "friends: horodatage illisible après écriture", "err", err)
		return ""
	}
	return ts
}

// friendsRejectMessage traduit un motif de refus en message utilisateur.
func friendsRejectMessage(reason string) string {
	switch reason {
	case "too_many":
		return "Trop d'amis dans la liste (50 au maximum)."
	case "gamertag_too_long":
		return "Un gamertag dépasse 50 caractères."
	default:
		return "Liste d'amis invalide."
	}
}
