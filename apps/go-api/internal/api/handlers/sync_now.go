// Package handlers — sync_now.go : « Synchroniser maintenant ».
//
// POST /sync/now lance, en arrière-plan, le cycle de synchronisation de l'ordonnanceur : le même que le
// cycle périodique, rattrapage de l'historique compris, pour tous les joueurs suivis. Il répond 202 aussitôt ;
// un cycle déjà en cours est refusé en 409. Le handler décode et traduit : le lancement vit derrière
// port.SyncNowService.
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/port"
)

// Codes d'erreur de POST /sync/now (le front écrit lui-même le texte à partir du code).
const (
	codeSyncCycleBusy        = "sync_cycle_busy"
	codeSchedulerUnavailable = "scheduler_unavailable"
)

// SyncNowHandler expose POST /sync/now.
type SyncNowHandler struct {
	svc port.SyncNowService
}

// NewSyncNowHandler crée le handler. svc peut être nil (démo : aucun ordonnanceur câblé) : la route est
// montée pour que le contrat publié la décrive, et répond 503.
func NewSyncNowHandler(svc port.SyncNowService) *SyncNowHandler {
	return &SyncNowHandler{svc: svc}
}

// syncCycleOutput : l'état du cycle (200).
type syncCycleOutput struct {
	Body *domain.SyncCycleState
}

// syncNowOutput : 202 Accepted (posé une fois au montage par humacore.DefaultStatus).
type syncNowOutput struct {
	Body *domain.SyncNowResponse
}

// Mount enregistre POST /sync/now et GET /sync/cycle via Huma sur le routeur chi fourni (garde
// d'authentification héritée du groupe où ils sont montés).
func (h *SyncNowHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/sync/cycle", h.GetSyncCycle,
		humacore.Op("getSyncCycle", "Un cycle de synchronisation tourne-t-il", "sync"))
	huma.Post(api, "/sync/now", h.StartSyncNow,
		humacore.Op("postSyncNow", "Lancer tout de suite le cycle de synchronisation", "sync"),
		humacore.DefaultStatus(http.StatusAccepted))
}

// StartSyncNow lance le cycle. POST /api/v1/sync/now -> 202 SyncNowResponse ; 409 sync_cycle_busy si un
// cycle tourne déjà ; 503 scheduler_unavailable si aucun cycle ne peut être lancé.
func (h *SyncNowHandler) StartSyncNow(ctx context.Context, _ *struct{}) (*syncNowOutput, error) {
	if h.svc == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, codeSchedulerUnavailable, "scheduler unavailable")
	}
	switch err := h.svc.StartSyncNow(); {
	case err == nil:
		slog.InfoContext(ctx, "sync_now: cycle demandé")
		return &syncNowOutput{Body: &domain.SyncNowResponse{Status: "started"}}, nil
	case errors.Is(err, port.ErrSyncCycleBusy):
		return nil, humacore.NewError(http.StatusConflict, codeSyncCycleBusy,
			i18n.Msg(i18n.FromContext(ctx), "sync.now.busy", nil))
	default:
		return nil, humacore.NewErrorFrom(ctx, http.StatusServiceUnavailable, codeSchedulerUnavailable, err)
	}
}

// GetSyncCycle dit si un cycle tourne. GET /api/v1/sync/cycle -> 200 SyncCycleState ; 503
// scheduler_unavailable sans ordonnanceur.
func (h *SyncNowHandler) GetSyncCycle(_ context.Context, _ *struct{}) (*syncCycleOutput, error) {
	if h.svc == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, codeSchedulerUnavailable, "scheduler unavailable")
	}
	return &syncCycleOutput{Body: &domain.SyncCycleState{Running: h.svc.CycleRunning()}}, nil
}
