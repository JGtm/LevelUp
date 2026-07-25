// Package handlers — match_exclusion.go : gestion des matchs non pertinents.
//
// Endpoint :
//
//	PATCH /api/v1/players/{player_slug}/matches/{match_id}/exclusion
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre le PATCH via huma.Patch. Logique métier
// inchangée (MatchExclusionService), seul le wrapping HTTP change.
//
// NOTE : GET /match-exclusions a été supprimé en revue 2026-04-29 P0.2 Q6
// (orphelin côté front). À réintroduire si une vue admin de listing devient
// nécessaire.
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/port"
)

// MatchExclusionHandler gère les endpoints d'exclusion de matchs.
type MatchExclusionHandler struct {
	newSvc ServiceFactory[port.MatchExclusionService]
}

// NewMatchExclusionHandler crée un MatchExclusionHandler.
func NewMatchExclusionHandler(newSvc ServiceFactory[port.MatchExclusionService]) *MatchExclusionHandler {
	return &MatchExclusionHandler{newSvc: newSvc}
}

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *MatchExclusionHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Patch(api, "/matches/{match_id}/exclusion", h.SetExclusion)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// matchExclusionInput : {player_slug} parent + {match_id} + body {excluded}.
type matchExclusionInput struct {
	PlayerSlug string `path:"player_slug"`
	MatchID    string `path:"match_id"`
	Body       domain.SetMatchExclusionRequest
}

// matchExclusionNoContent : réponse 204 sans corps.
type matchExclusionNoContent struct {
	Status int
}

// ─── Endpoint ────────────────────────────────────────────────────────────────

// SetExclusion marque ou démarque un match comme non pertinent.
// PATCH /api/v1/players/{player_slug}/matches/{match_id}/exclusion
// Body: {"excluded": true}
// Réponse: 204 No Content
func (h *MatchExclusionHandler) SetExclusion(ctx context.Context, in *matchExclusionInput) (*matchExclusionNoContent, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}

	if err := svc.SetExclusion(ctx, in.MatchID, in.Body.Excluded); err != nil {
		switch {
		case errors.Is(err, domain.ErrMatchNotFound):
			return nil, humacore.NewError(http.StatusNotFound, "match_not_found",
				"Match introuvable dans le registre")
		case errors.Is(err, domain.ErrRankedMatchNotExcludable):
			return nil, humacore.NewError(http.StatusUnprocessableEntity, "ranked_not_excludable",
				"Les matchs classés ne peuvent pas être exclus")
		case errors.Is(err, dblease.ErrDBLocked):
			return nil, huma.ErrorWithHeaders(
				humacore.NewError(http.StatusServiceUnavailable, "db_busy",
					"database is currently busy, please retry"),
				http.Header{"Retry-After": []string{"5"}},
			)
		}
		slog.WarnContext(ctx, "match exclusion: db error",
			"match_id", in.MatchID,
			"err", err,
		)
		return nil, humacore.NewError(http.StatusInternalServerError, "exclusion_error", err.Error())
	}

	slog.InfoContext(ctx, "match exclusion updated",
		"player_slug", in.PlayerSlug,
		"match_id", in.MatchID,
		"excluded", in.Body.Excluded,
	)
	return &matchExclusionNoContent{Status: http.StatusNoContent}, nil
}

// resolve résout le slug courant en MatchExclusionService ou renvoie une erreur
// Huma 404 (contrat préservé : {code:player_not_found}).
func (h *MatchExclusionHandler) resolve(ctx context.Context, slug string) (port.MatchExclusionService, error) {
	svc, err := h.newSvc(ctx, slug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	return svc, nil
}
