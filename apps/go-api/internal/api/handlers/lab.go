// Package handlers — lab.go : diagnostic d'instance (ex-Lab).
//
// A3.5 (DC-9, 2026-07-10) : le Lab est retiré de l'app — seule reste la route
// GET /lab/diagnostics, consommée par le panneau Diagnostics de l'onglet admin
// Données. Le titre courant est lu depuis le contexte par le service
// (ctxkeys.TitleSlug), pas via un path param.
package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/service"
)

// LabHandler gère le diagnostic d'instance.
type LabHandler struct {
	svc *service.LabService
}

// NewLabHandler crée un LabHandler.
func NewLabHandler(svc *service.LabService) *LabHandler {
	return &LabHandler{svc: svc}
}

// Mount enregistre la route diagnostics via Huma sur le routeur chi.
// Chemin RELATIF au point de montage (sous /api/v1 dans server.go).
func (h *LabHandler) Mount(r chi.Router) {
	api := humacore.NewAPI(r)
	huma.Get(api, "/lab/diagnostics", h.handleGetDiagnostics)
}

type labDiagnosticsOutput struct {
	Body *domain.LabDiagnosticsResponse
}

// handleGetDiagnostics retourne les diagnostics d'instance (parity + guards).
func (h *LabHandler) handleGetDiagnostics(ctx context.Context, _ *struct{}) (*labDiagnosticsOutput, error) {
	data, err := h.svc.GetDiagnostics(ctx)
	if err != nil {
		return nil, labError(err)
	}
	return &labDiagnosticsOutput{Body: data}, nil
}

// labError mappe les erreurs du service vers les erreurs Huma au contrat
// d'origine : ErrLabForbidden → 403 instance_management_disabled, autres → 500.
func labError(err error) error {
	if errors.Is(err, service.ErrLabForbidden) {
		return humacore.NewError(http.StatusForbidden, "instance_management_disabled", "Le diagnostic d'instance n'est pas autorisé sur cette instance.")
	}
	return humacore.NewError(http.StatusInternalServerError, "lab_error", err.Error())
}
