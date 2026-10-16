// Package handlers — pool_sharing.go : décision du joueur sur la mise en commun de son jeton.
//
// PUT /me/pool-sharing → accord ou retrait, pour le SEUL compte de la session.
//
// Le jeton d'un joueur n'entre dans le round-robin qui sert la synchro des autres que s'il
// l'a décidé (auth.UserTokens.PoolSharing). L'état courant voyage dans /bootstrap
// (pool_sharing) : ce handler n'a que l'écriture.
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
)

// PoolSharingStore est la part du MultiUserTokenStore dont le handler a besoin.
type PoolSharingStore interface {
	SetPoolSharing(xuid string, granted bool, basis string) error
}

// PoolSharingApplier applique la décision au pool en mémoire (AutoSyncScheduler).
type PoolSharingApplier interface {
	SetPoolShared(gamertag string, shared bool) bool
}

// PoolSharingHandler enregistre la décision d'un joueur sur son propre jeton.
type PoolSharingHandler struct {
	store   PoolSharingStore
	applier PoolSharingApplier
	users   authz.UserLookup
}

// NewPoolSharingHandler crée le handler. applier peut être nil (aucun pool en mémoire) :
// la décision est alors seulement persistée.
func NewPoolSharingHandler(store PoolSharingStore, applier PoolSharingApplier, users authz.UserLookup) *PoolSharingHandler {
	return &PoolSharingHandler{store: store, applier: applier, users: users}
}

// Mount enregistre la route (au point de montage de l'appelant, derrière RequireAuth).
func (h *PoolSharingHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Put(api, "/me/pool-sharing", h.handleSet,
		humacore.Op("putMyPoolSharing", "Grants or withdraws token pooling for the connected user", "auth"))
}

type setPoolSharingInput struct {
	Body struct {
		Granted bool `json:"granted" doc:"true = make my token available to the shared pool; false = keep it for myself."`
	}
}

type setPoolSharingOutput struct {
	Body struct {
		PoolSharing string `json:"pool_sharing" enum:"granted,declined" doc:"Recorded decision."`
	}
}

func (h *PoolSharingHandler) handleSet(ctx context.Context, in *setPoolSharingInput) (*setPoolSharingOutput, error) {
	sess := middleware.GetSession(ctx)
	user := authz.CurrentUser(sess, h.users)
	if user == nil || user.XUID == "" {
		return nil, humacore.NewError(http.StatusForbidden, "no_halo_identity", "no Xbox account linked to this session")
	}

	if err := h.store.SetPoolSharing(user.XUID, in.Body.Granted, auth.PoolSharingBasisUserChoice); err != nil {
		if errors.Is(err, auth.ErrUserTokensNotFound) {
			return nil, humacore.NewError(http.StatusConflict, "no_token", "no token stored for this account")
		}
		slog.ErrorContext(ctx, "pool_sharing: saving the decision failed",
			"xuid", user.XUID, "granted", in.Body.Granted, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "pool_sharing_error", "decision not saved")
	}
	if h.applier != nil && user.Gamertag != "" {
		h.applier.SetPoolShared(user.Gamertag, in.Body.Granted)
	}
	slog.InfoContext(ctx, "pool_sharing: decision saved", "xuid", user.XUID, "granted", in.Body.Granted)

	out := &setPoolSharingOutput{}
	out.Body.PoolSharing = domain.PoolSharingDeclined
	if in.Body.Granted {
		out.Body.PoolSharing = domain.PoolSharingGranted
	}
	return out, nil
}
