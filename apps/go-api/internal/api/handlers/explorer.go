// Package handlers — ExplorerHandler : POST .../pages/explorer/player-query
//
//	POST .../pages/explorer/matches-query
package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// ExplorerAuthFactory retourne un ExplorerService + contexte enrichi avec les
// HaloTokens du propriétaire de la page. Même pattern que CompareAuthFactory :
// le ctx enrichi doit être propagé au service pour que les fetchs live de
// l'encart "Profil joueur cible" disposent des tokens (auth_available).
type ExplorerAuthFactory func(ctx context.Context, slug string) (svc port.ExplorerService, enrichedCtx context.Context, xuid, gamertag string, err error)

// ExplorerHandler gère les endpoints de l'Explorer.
type ExplorerHandler struct {
	newExplorerSvc  ExplorerAuthFactory
	newMatchHistSvc ContextFactory[port.MatchHistoryService]
}

// NewExplorerHandler crée un ExplorerHandler.
func NewExplorerHandler(
	newExplorerSvc ExplorerAuthFactory,
	newMatchHistSvc ContextFactory[port.MatchHistoryService],
) *ExplorerHandler {
	return &ExplorerHandler{
		newExplorerSvc:  newExplorerSvc,
		newMatchHistSvc: newMatchHistSvc,
	}
}

// QueryPlayer retourne les matchs en commun avec un autre joueur.
// POST /api/v1/players/{player_slug}/pages/explorer/player-query
// Body JSON : { "target_gamertag": "...", "limit": 50 }
func (h *ExplorerHandler) QueryPlayer(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, enrichedCtx, _, _, err := h.newExplorerSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.ExplorerPlayerQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if req.TargetGamertag == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_gamertag", "target_gamertag est requis")
		return
	}

	resp, err := svc.GetCommonMatches(enrichedCtx, req.TargetGamertag, req.Page)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "explorer_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// QueryMatches retourne les matchs filtrés du joueur (explorer context).
// POST /api/v1/players/{player_slug}/pages/explorer/matches-query
// Body JSON : { "filters": {...}, "pagination": {...}, "sort_field": "...", "sort_dir": "..." }
func (h *ExplorerHandler) QueryMatches(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	mhSvc, _, _, err := h.newMatchHistSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.ExplorerMatchesQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	// Délégation au service match-history avec les mêmes filtres/tri/pagination.
	mhReq := domain.MatchHistoryQueryRequest{
		Filters:           req.Filters,
		Pagination:        req.Pagination,
		SortField:         req.SortField,
		SortDir:           req.SortDir,
		IncludeExportHint: req.IncludeExportHint,
		PerfTiers:         req.PerfTiers,
		SkillTiers:        req.SkillTiers,
		RankedContext:     req.RankedContext,
		OutcomeFilter:     req.OutcomeFilter,
		MatchStartDate:    req.MatchStartDate,
		MatchEndDate:      req.MatchEndDate,
		ExperienceTypes:   req.ExperienceTypes,
		Playlists:         req.Playlists,
		MapNames:          req.MapNames,
		ModeNames:         req.ModeNames,
		SquadScope:        req.SquadScope,
		MatchIDSearch:     req.MatchIDSearch,
		MatchIDs:          req.MatchIDs,
	}

	mhResp, err := mhSvc.GetPage(r.Context(), mhReq)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "explorer_matches_error", err.Error())
		return
	}

	// Génération du token d'export si demandé (même mécanisme que MatchHistoryHandler.Query).
	if req.IncludeExportHint && mhResp.ExportHint != nil {
		if token, terr := encodeExportToken(mhReq); terr == nil {
			mhResp.ExportHint.Token = &token
		}
	}

	// Projection vers ExplorerMatchesQueryResponse (sous-ensemble de match history).
	rows := make([]domain.ExplorerMatchesRow, 0, len(mhResp.Table.Items))
	for _, item := range mhResp.Table.Items {
		rows = append(rows, BuildExplorerRowFromMatchHistory(item))
	}

	resp := domain.ExplorerMatchesQueryResponse{
		Summary: domain.ExplorerMatchesSummary{
			TotalMatches:             mhResp.Summary.TotalMatchesScoped,
			AvailableExperienceTypes: mhResp.Summary.AvailableExperienceTypes,
			AvailablePlaylists:       mhResp.Summary.AvailablePlaylists,
			AvailableMaps:            mhResp.Summary.AvailableMaps,
			AvailableModes:           mhResp.Summary.AvailableModes,
			AvailableOutcomes:        mhResp.Summary.AvailableOutcomes,
			AvailablePerfTiers:       mhResp.Summary.AvailablePerfTiers,
			AvailableSkillTiers:      mhResp.Summary.AvailableSkillTiers,
			AvailableRankedContexts:  mhResp.Summary.AvailableRankedContexts,
			AvailableSquadScopes:     mhResp.Summary.AvailableSquadScopes,
		},
		ExportHint: mhResp.ExportHint,
		Table: domain.ExplorerMatchesTable{
			Items:      rows,
			Pagination: mhResp.Table.Pagination,
		},
	}
	writeJSON(w, http.StatusOK, resp)
}
