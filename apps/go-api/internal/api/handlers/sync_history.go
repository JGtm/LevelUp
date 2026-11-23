// Package handlers — sync_history.go : état de l'historique d'un joueur.
//
// GET /players/{player_slug}/sync/history rend, par titre suivi, l'état de l'historique (complet ou
// rattrapage en cours), le nombre de matchs rangés et le plus ancien, la dernière passe de synchronisation
// et le calendrier de la synchronisation automatique. Lecture seule : le handler décode et traduit, la
// composition vit derrière port.HistorySyncService. Les libellés arrivent résolus dans la langue de la
// requête.
package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

const codeHistorySyncFailed = "history_sync_failed"

// SyncHistoryHandler expose GET /players/{player_slug}/sync/history.
type SyncHistoryHandler struct {
	svc port.HistorySyncService
}

// NewSyncHistoryHandler crée le handler. svc peut être nil (démo : aucun ordonnanceur câblé) : la route
// est montée pour que le contrat publié la décrive, et répond 503.
func NewSyncHistoryHandler(svc port.HistorySyncService) *SyncHistoryHandler {
	return &SyncHistoryHandler{svc: svc}
}

// syncHistoryInput : path param {player_slug}.
type syncHistoryInput struct {
	PlayerSlug string `path:"player_slug"`
}

// syncHistoryOutput : corps = état de l'historique du joueur.
type syncHistoryOutput struct {
	Body *domain.HistorySyncStatus
}

// Mount enregistre GET /sync/history via Huma sur le sous-routeur chi (préfixe /players/{player_slug} et
// gardes de propriété hérités).
func (h *SyncHistoryHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/sync/history", h.GetSyncHistory,
		humacore.Op("getPlayerSyncHistory", "Historique d'un joueur, par titre suivi", "sync"))
}

// GetSyncHistory rend l'état de l'historique du joueur.
// GET /api/v1/players/{player_slug}/sync/history -> 200 HistorySyncStatus.
func (h *SyncHistoryHandler) GetSyncHistory(ctx context.Context, in *syncHistoryInput) (*syncHistoryOutput, error) {
	if h.svc == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, codeSchedulerUnavailable, "scheduler unavailable")
	}
	if in.PlayerSlug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_player_slug", "player_slug manquant.")
	}
	status, err := h.svc.HistorySyncStatus(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewErrorFrom(ctx, http.StatusInternalServerError, codeHistorySyncFailed, err)
	}
	return &syncHistoryOutput{Body: status}, nil
}
