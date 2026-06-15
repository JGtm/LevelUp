// Package handlers — FiltersHandler : POST /api/v1/players/{player_slug}/filters/resolve.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// FiltersHandler gère POST .../filters/resolve.
type FiltersHandler struct {
	newSvc ServiceFactory[port.FiltersService]
}

// NewFiltersHandler crée un FiltersHandler.
func NewFiltersHandler(newSvc ServiceFactory[port.FiltersService]) *FiltersHandler {
	return &FiltersHandler{newSvc: newSvc}
}

// Resolve applique le filtre et retourne les options disponibles.
func (h *FiltersHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var input domain.FilterContextInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	if err := input.Validate(); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_filters", err.Error())
		return
	}

	result, err := svc.Resolve(r.Context(), input)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "filters_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// MatchIDs retourne la liste ordonnée (start_time DESC) des match_id de la
// sélection. Alimente le bouton "Voir les matchs" : le front ouvre le 1er
// match et parcourt la liste via prev/next. Même pipeline de filtrage que
// Resolve → respecte match_context (solo/squad), sessions, période et cascade.
func (h *FiltersHandler) MatchIDs(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var input domain.FilterContextInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	if err := input.Validate(); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_filters", err.Error())
		return
	}

	ids, err := svc.ResolveMatchIDs(r.Context(), input)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "filters_error", err.Error())
		return
	}
	// Slice jamais nil : un slice Go nil sérialise en JSON `null`, ce qui casse
	// `.length`/`.map` côté front. Cf. testutil.RequireNoNilSlicesWithoutOmitempty.
	if ids == nil {
		ids = []string{}
	}
	writeJSON(w, http.StatusOK, domain.FilterMatchIDsResponse{MatchIDs: ids})
}
