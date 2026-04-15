// Package handlers — ExplorerHandler : POST .../pages/explorer/player-query.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/service"
)

// ExplorerHandler gère les endpoints de l'Explorer.
type ExplorerHandler struct {
	cfg *config.AppConfig
}

// NewExplorerHandler crée un ExplorerHandler.
func NewExplorerHandler(cfg *config.AppConfig) *ExplorerHandler {
	return &ExplorerHandler{cfg: cfg}
}

// QueryPlayer retourne les matchs en commun avec un autre joueur.
// POST /api/v1/players/{player_slug}/pages/explorer/player-query
// Body JSON : { "other_gamertag": "...", "limit": 50 }
func (h *ExplorerHandler) QueryPlayer(w http.ResponseWriter, r *http.Request) {
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.ExplorerPlayerQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if req.OtherGamertag == "" {
		writeError(w, http.StatusBadRequest, "missing_gamertag", "other_gamertag est requis")
		return
	}

	repo := duckdb.NewExplorerRepo(pdb, pdb.XUID)
	svc := service.NewExplorerService(repo, pdb.XUID)

	resp, err := svc.GetCommonMatches(r.Context(), req.OtherGamertag, req.Limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "explorer_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *ExplorerHandler) resolvePlayer(r *http.Request) (*duckdb.PlayerDB, error) {
	slug := chi.URLParam(r, "player_slug")
	return config.ResolvePlayer(r.Context(), h.cfg, slug)
}
