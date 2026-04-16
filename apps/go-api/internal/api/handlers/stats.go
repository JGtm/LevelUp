// Package handlers — StatsHandler : endpoint des séries temporelles et stats analytiques.
package handlers

import (
	"encoding/json"
	"net/http"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/service"

	"github.com/go-chi/chi/v5"
)

// StatsHandler gère les requêtes de stats/séries analytiques.
type StatsHandler struct {
	cfg *config.AppConfig
}

// NewStatsHandler crée un StatsHandler.
func NewStatsHandler(cfg *config.AppConfig) *StatsHandler {
	return &StatsHandler{cfg: cfg}
}

// GetPage traite POST /api/v1/players/{player_slug}/pages/stats/query.
//
// Corps JSON : { "tab": "win_loss|accuracy|objective|form|lusr|all", "mode": "period|sessions" }
func (h *StatsHandler) GetPage(w http.ResponseWriter, r *http.Request) {
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "player_not_found", "joueur introuvable")
		return
	}

	var req domain.StatsQueryRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "corps JSON invalide")
		return
	}
	if req.Tab == "" {
		req.Tab = "win_loss"
	}

	repo := duckdb.NewStatsRepo(pdb)
	svc := service.NewStatsService(repo)

	resp, svcErr := svc.GetPage(r.Context(), req)
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, "stats_error", "erreur chargement stats")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *StatsHandler) resolvePlayer(r *http.Request) (*duckdb.PlayerDB, error) {
	slug := chi.URLParam(r, "player_slug")
	return config.ResolvePlayer(r.Context(), h.cfg, slug)
}
