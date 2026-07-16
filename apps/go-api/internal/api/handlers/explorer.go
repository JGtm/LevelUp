// Package handlers — ExplorerHandler : POST .../pages/explorer/player-query
//
//	POST .../pages/explorer/matches-query
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre les 2 POST via huma.Post. Logique métier
// inchangée (ExplorerService + MatchHistoryService), seul le wrapping HTTP change.
//
// Les corps sont lus via RawBody (pas de Body typé) pour reproduire EXACTEMENT le
// contrat de décodage d'origine : un JSON invalide renvoie 400 {invalid_body}
// (parse maison) et non le 422 de validation Huma qu'un Body typé produirait.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
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

// Mount enregistre les 2 routes via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *ExplorerHandler) Mount(r chi.Router) {
	api := humacore.NewAPI(r)
	huma.Post(api, "/pages/explorer/player-query", h.handleQueryPlayer)
	huma.Post(api, "/pages/explorer/matches-query", h.handleQueryMatches)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// explorerQueryInput : {player_slug} parent + corps brut décodé maison.
// RawBody (pas Body typé) → préserve le contrat 400 {invalid_body} sur JSON
// invalide (un Body typé renverrait le 422 de validation Huma).
type explorerQueryInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}

type explorerPlayerQueryOutput struct {
	Body domain.ExplorerPlayerQueryResponse
}
type explorerMatchesQueryOutput struct {
	Body domain.ExplorerMatchesQueryResponse
}

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleQueryPlayer retourne les matchs en commun avec un autre joueur.
// POST /api/v1/players/{player_slug}/pages/explorer/player-query
// Body JSON : { "target_gamertag": "...", "limit": 50 }
func (h *ExplorerHandler) handleQueryPlayer(ctx context.Context, in *explorerQueryInput) (*explorerPlayerQueryOutput, error) {
	svc, enrichedCtx, _, _, err := h.newExplorerSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	var req domain.ExplorerPlayerQueryRequest
	if err := json.NewDecoder(bytes.NewReader(in.RawBody)).Decode(&req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	if req.TargetGamertag == "" && req.TargetXUID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_target", "target_gamertag ou target_xuid est requis")
	}

	resp, err := svc.GetCommonMatches(enrichedCtx, req.TargetGamertag, req.TargetXUID, req.Page)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "explorer_error", err.Error())
	}
	return &explorerPlayerQueryOutput{Body: resp}, nil
}

// handleQueryMatches retourne les matchs filtrés du joueur (explorer context).
// POST /api/v1/players/{player_slug}/pages/explorer/matches-query
// Body JSON : { "filters": {...}, "pagination": {...}, "sort_field": "...", "sort_dir": "..." }
func (h *ExplorerHandler) handleQueryMatches(ctx context.Context, in *explorerQueryInput) (*explorerMatchesQueryOutput, error) {
	mhSvc, _, _, err := h.newMatchHistSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	var req domain.ExplorerMatchesQueryRequest
	if err := json.NewDecoder(bytes.NewReader(in.RawBody)).Decode(&req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
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
		// Opt-in du bandeau de briefing étendu (mode Matchs). L'Explorer l'envoie
		// à true ; la page Historique (autre handler) ne pose pas le flag.
		IncludeExplorerBriefing: req.IncludeBriefing,
	}

	mhResp, err := mhSvc.GetPage(ctx, mhReq)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "explorer_matches_error", err.Error())
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
		// Bandeau de briefing (mapping pur) : nil si non demandé ou scope vide.
		Briefing: mhResp.Briefing,
	}
	return &explorerMatchesQueryOutput{Body: resp}, nil
}
