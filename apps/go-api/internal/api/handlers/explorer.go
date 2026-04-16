// Package handlers — ExplorerHandler : POST .../pages/explorer/player-query
//
//	POST .../pages/explorer/matches-query
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
// Body JSON : { "target_gamertag": "...", "limit": 50 }
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
	if req.TargetGamertag == "" {
		writeError(w, http.StatusBadRequest, "missing_gamertag", "target_gamertag est requis")
		return
	}

	repo := duckdb.NewExplorerRepo(pdb, pdb.XUID)
	svc := service.NewExplorerService(repo, pdb.XUID)

	resp, err := svc.GetCommonMatches(r.Context(), req.TargetGamertag, req.Limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "explorer_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// QueryMatches retourne les matchs filtrés du joueur (explorer context).
// POST /api/v1/players/{player_slug}/pages/explorer/matches-query
// Body JSON : { "filters": {...}, "pagination": {...}, "sort_field": "...", "sort_dir": "..." }
func (h *ExplorerHandler) QueryMatches(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	pdb, err := config.ResolvePlayer(r.Context(), h.cfg, slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.ExplorerMatchesQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	// Délégation au service match-history avec les mêmes filtres/tri/pagination.
	mhReq := domain.MatchHistoryQueryRequest{
		Filters:    req.Filters,
		Pagination: req.Pagination,
		SortField:  req.SortField,
		SortDir:    req.SortDir,
	}

	repo := duckdb.NewMatchHistoryRepo(pdb)
	svc := service.NewMatchHistoryService(repo, pdb.Gamertag)

	mhResp, err := svc.GetPage(r.Context(), mhReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "explorer_matches_error", err.Error())
		return
	}

	// Projection vers ExplorerMatchesQueryResponse (sous-ensemble de match history).
	rows := make([]domain.ExplorerMatchesRow, 0, len(mhResp.Table.Items))
	for _, item := range mhResp.Table.Items {
		rows = append(rows, domain.ExplorerMatchesRow{
			MatchID:      item.MatchID,
			StartTime:    item.StartTime,
			MapUI:        item.MapUI,
			ModeUI:       item.ModeUI,
			OutcomeCode:  item.OutcomeCode,
			OutcomeLabel: item.OutcomeLabel,
			Kills:        extractKillsFromLabel(item.ScoreLabel),
			Deaths:       0, // non disponible dans MatchHistoryRow
			KDA:          nil,
			MatchURL:     item.MatchURL,
		})
	}

	resp := domain.ExplorerMatchesQueryResponse{
		Matches:    rows,
		Pagination: mhResp.Table.Pagination,
		Total:      mhResp.Summary.TotalMatchesScoped,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *ExplorerHandler) resolvePlayer(r *http.Request) (*duckdb.PlayerDB, error) {
	slug := chi.URLParam(r, "player_slug")
	return config.ResolvePlayer(r.Context(), h.cfg, slug)
}

// extractKillsFromLabel est un placeholder — MatchHistoryRow ne stocke pas les kills séparément.
// Retourne 0 jusqu'à enrichissement du service (Sprint 33).
func extractKillsFromLabel(_ string) int {
	return 0
}
