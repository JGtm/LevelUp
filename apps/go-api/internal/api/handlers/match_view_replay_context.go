// Package handlers — GET .../matches/{match_id}/replay/context : le contexte de match de la
// page de rejeu (en-tête, lignes du tableau de score, événements du fil, médias).
package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
)

// matchReplayContextOutput : CacheControl porte `no-store` sur un contexte construit avec une
// lecture en échec ou pendant l'analyse du film (réponse transitoire, comme la vue complète).
type matchReplayContextOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         domain.MatchReplayContext
}

// MountReplayContext enregistre GET /matches/{match_id}/replay/context. Monté par l'appelant
// sous les gardes du rejeu (capability de titre, garde locale, participation), à côté des
// autres routes /replay* : un titre sans rejeu rend 503, jamais ce contexte.
func (h *MatchViewHandler) MountReplayContext(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/matches/{match_id}/replay/context", h.handleGetReplayContext,
		humacore.Op("getMatchReplayContext", "Contexte de match de la page de rejeu", "match-view"))
}

// handleGetReplayContext — GET /api/v1/players/{player_slug}/matches/{match_id}/replay/context.
// Erreurs et transformation des médias identiques à handleGetMatchView.
func (h *MatchViewHandler) handleGetReplayContext(ctx context.Context, in *matchViewInput) (*matchReplayContextOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	if in.MatchID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_match_id", "match_id est requis")
	}
	resp, err := svc.GetReplayContext(ctx, in.MatchID)
	if err != nil {
		return nil, matchViewServiceError(ctx, err, "replay_context_error")
	}
	resp.Media.MediaItems = filterMatchMediaByAudience(resp.Media.MediaItems,
		resolveMediaAudience(ctx, h.mediaAudience, in.PlayerSlug))
	h.transformMatchMediaURLs(in.PlayerSlug, resp.Media.MediaItems)
	out := &matchReplayContextOutput{Body: resp}
	if resp.SourceUnavailable || resp.Header.AnalysisPending {
		out.CacheControl = noStoreCacheControl
	}
	return out, nil
}

// matchViewServiceError traduit une erreur du service de la vue match en refus HTTP : 404
// `match_not_participant` (ADR 0029), 404 `match_not_found` (absence typée par le service,
// domain.ErrNotFound — aucun reniflage du texte de l'erreur, une panne dont le message
// contiendrait « no rows » ne se déguise pas en 404), 499 sur requête abandonnée, sinon 500
// sous `code` (la cause va au journal, jamais au client).
func matchViewServiceError(ctx context.Context, err error, code string) error {
	var apiErr *domain.APIError
	if errors.As(err, &apiErr) && apiErr.Code == "match_not_participant" {
		return humacore.NewError(http.StatusNotFound, "match_not_participant", apiErr.Message)
	}
	if errors.As(err, &apiErr) && apiErr.Code == "not_found" {
		return humacore.NewError(http.StatusNotFound, "match_not_found", apiErr.Message)
	}
	if isClientClosed(ctx) {
		return mapServiceError(ctx, err, code)
	}
	return humacore.NewErrorFrom(ctx, http.StatusInternalServerError, code, err)
}
