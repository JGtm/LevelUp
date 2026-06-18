// Package handlers — lab.go : endpoints du Lab interne / Instance Lab.
package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

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

// GetResources retourne les données de l'explorateur de ressources.
func (h *LabHandler) GetResources(w http.ResponseWriter, r *http.Request) {
	query, ok := parseLabResourcesQuery(w, r)
	if !ok {
		return
	}
	data, err := h.svc.GetResources(r.Context(), query)
	if err != nil {
		writeLabError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// GetContracts retourne le diff OpenAPI calculé côté Go.
func (h *LabHandler) GetContracts(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.GetContracts(r.Context())
	if err != nil {
		writeLabError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// GetDiagnostics retourne les diagnostics d'instance (parity + guards).
func (h *LabHandler) GetDiagnostics(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.GetDiagnostics(r.Context())
	if err != nil {
		writeLabError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// GetWaypoint exécute une exploration live de l'API Discovery UGC (Atelier).
// GET /lab/waypoint?segment=map&asset_id=...&version_id=...&lang=fr-FR
func (h *LabHandler) GetWaypoint(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := domain.LabWaypointQuery{
		Segment:   q.Get("segment"),
		AssetID:   q.Get("asset_id"),
		VersionID: q.Get("version_id"),
		Lang:      q.Get("lang"),
	}
	data, err := h.svc.ExploreWaypoint(r.Context(), query)
	if err != nil {
		writeLabError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func parseLabResourcesQuery(w http.ResponseWriter, r *http.Request) (domain.LabResourcesQuery, bool) {
	q := r.URL.Query()
	query := domain.LabResourcesQuery{
		SnapshotKey: q.Get("snapshot_key"),
		AssetID:     q.Get("asset_id"),
		AssetSearch: q.Get("asset_search"),
		MedalSearch: q.Get("medal_search"),
	}
	if raw := q.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil {
			writeError(r.Context(), w, http.StatusBadRequest, "invalid_limit", "Le paramètre limit doit être un entier.")
			return domain.LabResourcesQuery{}, false
		}
		query.Limit = limit
	}
	if raw := q.Get("medal_id"); raw != "" {
		medalID, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			writeError(r.Context(), w, http.StatusBadRequest, "invalid_medal_id", "Le paramètre medal_id doit être un entier.")
			return domain.LabResourcesQuery{}, false
		}
		query.MedalID = medalID
	}
	return query, true
}

func writeLabError(ctx context.Context, w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrLabForbidden) {
		writeError(ctx, w, http.StatusForbidden, "instance_management_disabled", "Le Lab interne n'est pas autorisé sur cette instance.")
		return
	}
	if errors.Is(err, service.ErrLabWaypointInvalid) {
		writeError(ctx, w, http.StatusBadRequest, "invalid_waypoint_query", "Paramètres requis : segment (map|playlist|pair|game_variant), asset_id, version_id.")
		return
	}
	if errors.Is(err, service.ErrLabWaypointUnavailable) {
		writeError(ctx, w, http.StatusServiceUnavailable, "waypoint_explorer_unavailable", "L'explorateur d'API n'est pas disponible (aucune source de token Spartan).")
		return
	}
	writeError(ctx, w, http.StatusInternalServerError, "lab_error", err.Error())
}
