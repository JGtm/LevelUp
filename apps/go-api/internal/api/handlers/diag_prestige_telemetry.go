// Package handlers — DiagPrestigeTelemetryHandler :
// GET /api/v1/_diag/prestige/telemetry/{player_slug}.
//
// ADR 0020 (coach→pont Prestige). Endpoint admin/dev qui agrège la table
// append-only prestige_telemetry d'un joueur par origine du défi (coach / user /
// pilot_mode / unknown) : taux d'acceptation et de complétion par origine. Permet
// de mesurer si les défis proposés par le coach sont acceptés/complétés davantage
// que ceux d'autres origines, sans schéma dédié.
//
// Usage typique :
//
//	curl -s http://127.0.0.1:8000/api/v1/_diag/prestige/telemetry/JGtm | jq
package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
)

// PrestigeTelemetryDiagProvider est l'interface attendue par le handler.
// Implémentée par duckdb.PrestigeTelemetryDiagRepo.
type PrestigeTelemetryDiagProvider interface {
	GetPrestigeTelemetryDiag(ctx context.Context, slug string) (*domain.PrestigeTelemetryDiag, error)
}

// PrestigeTelemetryDiagFactory : résout slug → provider (player-scoped).
type PrestigeTelemetryDiagFactory func(ctx context.Context, slug string) (PrestigeTelemetryDiagProvider, error)

// DiagPrestigeTelemetryHandler expose l'endpoint d'agrégation télémétrie.
type DiagPrestigeTelemetryHandler struct {
	newProvider PrestigeTelemetryDiagFactory
}

// NewDiagPrestigeTelemetryHandler crée un handler avec une factory injectée.
func NewDiagPrestigeTelemetryHandler(newProvider PrestigeTelemetryDiagFactory) *DiagPrestigeTelemetryHandler {
	return &DiagPrestigeTelemetryHandler{newProvider: newProvider}
}

// Mount enregistre la route via Huma au point de montage chi (préfixe /api/v1 +
// middleware hérités). Lit {player_slug} dans son propre path.
func (h *DiagPrestigeTelemetryHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/_diag/prestige/telemetry/{player_slug}", h.handleGetDiag)
}

// diagPrestigeTelemetryInput : path param {player_slug}.
type diagPrestigeTelemetryInput struct {
	PlayerSlug string `path:"player_slug"`
}

type diagPrestigeTelemetryOutput struct {
	Body *domain.PrestigeTelemetryDiag
}

// handleGetDiag : GET /api/v1/_diag/prestige/telemetry/{player_slug}.
func (h *DiagPrestigeTelemetryHandler) handleGetDiag(ctx context.Context, in *diagPrestigeTelemetryInput) (*diagPrestigeTelemetryOutput, error) {
	if in.PlayerSlug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_slug", "player_slug requis")
	}
	provider, err := h.newProvider(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	diag, err := provider.GetPrestigeTelemetryDiag(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "diag_error", err.Error())
	}
	return &diagPrestigeTelemetryOutput{Body: diag}, nil
}
