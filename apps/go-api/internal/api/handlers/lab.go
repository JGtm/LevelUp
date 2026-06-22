// Package handlers — lab.go : endpoints du Lab interne / Instance Lab.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le routeur chi
// (les routes Lab sont TOP-LEVEL sous /api/v1, sans path param) et enregistre les
// trois GET via huma.Get. Logique métier inchangée (LabService), seul le wrapping
// HTTP change. Le titre courant est toujours lu depuis le contexte par le service
// (ctxkeys.TitleSlug), pas via un path param.
package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/service"
)

// LabHandler gère les endpoints globaux du Lab interne.
type LabHandler struct {
	svc *service.LabService
}

// NewLabHandler crée un LabHandler.
func NewLabHandler(svc *service.LabService) *LabHandler {
	return &LabHandler{svc: svc}
}

// Mount enregistre les trois routes Lab via Huma sur le routeur chi.
// Chemins RELATIFS au point de montage (sous /api/v1 dans server.go).
func (h *LabHandler) Mount(r chi.Router) {
	api := humacore.NewAPI(r)
	huma.Get(api, "/lab/resources", h.handleGetResources)
	huma.Get(api, "/lab/contracts", h.handleGetContracts)
	huma.Get(api, "/lab/diagnostics", h.handleGetDiagnostics)
	huma.Get(api, "/lab/waypoint", h.handleGetWaypoint)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// labResourcesInput : query params de l'explorateur de ressources (tous tolérés
// vides — le service applique les normalisations). limit / medal_id sont parsés
// maison pour préserver le contrat d'erreur (400 invalid_limit / invalid_medal_id).
type labResourcesInput struct {
	SnapshotKey string `query:"snapshot_key"`
	AssetID     string `query:"asset_id"`
	AssetSearch string `query:"asset_search"`
	MedalSearch string `query:"medal_search"`
	Limit       string `query:"limit"`
	MedalID     string `query:"medal_id"`
}

type labResourcesOutput struct {
	Body *domain.LabResourcesResponse
}
type labContractsOutput struct {
	Body *domain.LabContractsResponse
}
type labDiagnosticsOutput struct {
	Body *domain.LabDiagnosticsResponse
}

// labWaypointInput : query params de l'exploration live Discovery UGC (Lab).
type labWaypointInput struct {
	Segment   string `query:"segment"`
	AssetID   string `query:"asset_id"`
	VersionID string `query:"version_id"`
	Lang      string `query:"lang"`
}

type labWaypointOutput struct {
	Body *domain.LabWaypointResponse
}

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleGetResources retourne les données de l'explorateur de ressources.
func (h *LabHandler) handleGetResources(ctx context.Context, in *labResourcesInput) (*labResourcesOutput, error) {
	query, err := parseLabResourcesQuery(in)
	if err != nil {
		return nil, err
	}
	data, err := h.svc.GetResources(ctx, query)
	if err != nil {
		return nil, labError(err)
	}
	return &labResourcesOutput{Body: data}, nil
}

// handleGetContracts retourne le diff OpenAPI calculé côté Go.
func (h *LabHandler) handleGetContracts(ctx context.Context, _ *struct{}) (*labContractsOutput, error) {
	data, err := h.svc.GetContracts(ctx)
	if err != nil {
		return nil, labError(err)
	}
	return &labContractsOutput{Body: data}, nil
}

// handleGetDiagnostics retourne les diagnostics d'instance (parity + guards).
func (h *LabHandler) handleGetDiagnostics(ctx context.Context, _ *struct{}) (*labDiagnosticsOutput, error) {
	data, err := h.svc.GetDiagnostics(ctx)
	if err != nil {
		return nil, labError(err)
	}
	return &labDiagnosticsOutput{Body: data}, nil
}

// handleGetWaypoint exécute une exploration live de l'API Discovery UGC (Lab).
// GET /lab/waypoint?segment=map&asset_id=...&version_id=...&lang=fr-FR
func (h *LabHandler) handleGetWaypoint(ctx context.Context, in *labWaypointInput) (*labWaypointOutput, error) {
	query := domain.LabWaypointQuery{
		Segment:   in.Segment,
		AssetID:   in.AssetID,
		VersionID: in.VersionID,
		Lang:      in.Lang,
	}
	data, err := h.svc.ExploreWaypoint(ctx, query)
	if err != nil {
		return nil, labError(err)
	}
	return &labWaypointOutput{Body: data}, nil
}

// parseLabResourcesQuery extrait les query params en domain.LabResourcesQuery.
// limit / medal_id non numériques → 400 (invalid_limit / invalid_medal_id),
// contrat identique à l'ancien parseLabResourcesQuery.
func parseLabResourcesQuery(in *labResourcesInput) (domain.LabResourcesQuery, error) {
	query := domain.LabResourcesQuery{
		SnapshotKey: in.SnapshotKey,
		AssetID:     in.AssetID,
		AssetSearch: in.AssetSearch,
		MedalSearch: in.MedalSearch,
	}
	if in.Limit != "" {
		limit, err := strconv.Atoi(in.Limit)
		if err != nil {
			return domain.LabResourcesQuery{}, humacore.NewError(http.StatusBadRequest, "invalid_limit", "Le paramètre limit doit être un entier.")
		}
		query.Limit = limit
	}
	if in.MedalID != "" {
		medalID, err := strconv.ParseInt(in.MedalID, 10, 64)
		if err != nil {
			return domain.LabResourcesQuery{}, humacore.NewError(http.StatusBadRequest, "invalid_medal_id", "Le paramètre medal_id doit être un entier.")
		}
		query.MedalID = medalID
	}
	return query, nil
}

// labError mappe les erreurs du service Lab vers les erreurs Huma au contrat
// d'origine : ErrLabForbidden → 403 instance_management_disabled, autres → 500.
func labError(err error) error {
	if errors.Is(err, service.ErrLabForbidden) {
		return humacore.NewError(http.StatusForbidden, "instance_management_disabled", "Le Lab interne n'est pas autorisé sur cette instance.")
	}
	if errors.Is(err, service.ErrLabWaypointInvalid) {
		return humacore.NewError(http.StatusBadRequest, "invalid_waypoint_query", "Paramètres requis : segment (map|playlist|pair|game_variant), asset_id, version_id.")
	}
	if errors.Is(err, service.ErrLabWaypointUnavailable) {
		return humacore.NewError(http.StatusServiceUnavailable, "waypoint_explorer_unavailable", "L'explorateur d'API n'est pas disponible (aucune source de token Spartan).")
	}
	return humacore.NewError(http.StatusInternalServerError, "lab_error", err.Error())
}
