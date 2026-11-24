// Package handlers — admin_replay_schema.go : endpoint monitoring « rejeux à l'ancien schéma »
// (GET /admin/monitoring/replay-schema). Runner injecté ; le handler ne décide rien.
package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
)

// ReplaySchemaRunner rend le décompte des rejeux par schéma (implémenté par ServiceRegistry.ReplaySchemaReport).
type ReplaySchemaRunner func(ctx context.Context) (domain.AdminReplaySchema, error)

// AdminReplaySchemaHandler sert l'endpoint du décompte.
type AdminReplaySchemaHandler struct {
	report ReplaySchemaRunner
}

// NewAdminReplaySchemaHandler construit le handler.
func NewAdminReplaySchemaHandler(report ReplaySchemaRunner) *AdminReplaySchemaHandler {
	return &AdminReplaySchemaHandler{report: report}
}

// Mount enregistre la route Huma sur le sous-routeur /admin.
func (h *AdminReplaySchemaHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/monitoring/replay-schema", h.handleGet, humacore.Op(
		"getAdminMonitoringReplaySchema",
		"Monitoring dashboard: stored replays counted by schema version (current, on an older schema, unreadable), per title (admin auth required)",
		"admin"))
}

type adminReplaySchemaOutput struct {
	Body domain.AdminReplaySchema
}

// handleGet — GET /admin/monitoring/replay-schema.
func (h *AdminReplaySchemaHandler) handleGet(ctx context.Context, _ *struct{}) (*adminReplaySchemaOutput, error) {
	resp, err := h.report(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "admin_monitoring: replay schema census failed", "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "replay_schema_error",
			"Unable to count the stored replays by schema version.")
	}
	return &adminReplaySchemaOutput{Body: resp}, nil
}
