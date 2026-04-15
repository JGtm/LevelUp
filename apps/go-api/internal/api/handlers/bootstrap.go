// Package handlers — handlers GET /api/v1/bootstrap et GET /api/v1/players.
package handlers

import (
	"net/http"

	"levelup/go-api/internal/service"
)

// BootstrapHandler gère GET /api/v1/bootstrap.
type BootstrapHandler struct {
	svc *service.BootstrapService
}

// NewBootstrapHandler crée un BootstrapHandler.
func NewBootstrapHandler(svc *service.BootstrapService) *BootstrapHandler {
	return &BootstrapHandler{svc: svc}
}

// ServeHTTP retourne le BootstrapResponse au shell React.
func (h *BootstrapHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Build(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "bootstrap_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// PlayersHandler gère GET /api/v1/players.
type PlayersHandler struct {
	svc *service.BootstrapService
}

// NewPlayersHandler crée un PlayersHandler.
func NewPlayersHandler(svc *service.BootstrapService) *PlayersHandler {
	return &PlayersHandler{svc: svc}
}

// ServeHTTP retourne la liste des joueurs disponibles.
func (h *PlayersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.BuildPlayersList(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "players_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
