// Package handlers — sync_now.go : « Synchroniser maintenant ».
//
// POST /sync/now lance, en arrière-plan, le cycle de synchronisation de l'ordonnanceur : le même que le
// cycle périodique, rattrapage de l'historique compris, pour tous les joueurs suivis. Il répond 202 aussitôt ;
// un cycle déjà en cours est refusé en 409, un cycle terminé depuis moins de deux minutes en 429 (Retry-After).
// Le handler décode et traduit : le lancement vit derrière port.SyncNowService.
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

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
	codeSyncCycleCooldown    = "sync_cycle_cooldown"
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
// cycle tourne déjà ; 429 sync_cycle_cooldown si un cycle s'est terminé il y a moins de deux minutes ;
// 503 scheduler_unavailable si aucun cycle ne peut être lancé.
func (h *SyncNowHandler) StartSyncNow(ctx context.Context, _ *struct{}) (*syncNowOutput, error) {
	if h.svc == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, codeSchedulerUnavailable, "scheduler unavailable")
	}
	err := h.svc.StartSyncNow()
	var cooldown *port.SyncCooldownError
	switch {
	case err == nil:
		slog.InfoContext(ctx, "sync_now: cycle demandé")
		return &syncNowOutput{Body: &domain.SyncNowResponse{Status: "started"}}, nil
	case errors.As(err, &cooldown):
		return nil, syncCooldownError(ctx, cooldown.RetryAfter)
	case errors.Is(err, port.ErrSyncCycleBusy):
		return nil, humacore.NewError(http.StatusConflict, codeSyncCycleBusy,
			i18n.Msg(i18n.FromContext(ctx), "sync.now.busy", nil))
	default:
		return nil, humacore.NewErrorFrom(ctx, http.StatusServiceUnavailable, codeSchedulerUnavailable, err)
	}
}

// syncCooldownError est le 429 d'un cycle demandé trop tôt : code stable, message du catalogue (délai en
// secondes) et `Retry-After`. L'écran relit GET /sync/cycle pour l'heure de disponibilité.
func syncCooldownError(ctx context.Context, retryAfter time.Duration) error {
	secs := int(math.Ceil(retryAfter.Seconds()))
	if secs < 1 {
		secs = 1
	}
	return huma.ErrorWithHeaders(
		humacore.NewError(http.StatusTooManyRequests, codeSyncCycleCooldown,
			i18n.Msg(i18n.FromContext(ctx), "sync.now.cooldown", i18n.Params{"seconds": strconv.Itoa(secs)})),
		http.Header{headerRetryAfter: []string{strconv.Itoa(secs)}})
}

// utcPtr rend l'instant en UTC, ou nil quand il est inconnu (zéro).
func utcPtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	u := t.UTC()
	return &u
}

// GetSyncCycle dit si un cycle tourne. GET /api/v1/sync/cycle -> 200 SyncCycleState ; 503
// scheduler_unavailable sans ordonnanceur.
func (h *SyncNowHandler) GetSyncCycle(_ context.Context, _ *struct{}) (*syncCycleOutput, error) {
	if h.svc == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, codeSchedulerUnavailable, "scheduler unavailable")
	}
	last, availableAt := h.svc.ManualWindow()
	state := &domain.SyncCycleState{Running: h.svc.CycleRunning(), LastCycleAt: utcPtr(last), ManualAvailableAt: utcPtr(availableAt)}
	return &syncCycleOutput{Body: state}, nil
}
