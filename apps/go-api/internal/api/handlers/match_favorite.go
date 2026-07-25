// Package handlers — match_favorite.go : handler HTTP pour les favoris de matchs.
//
// Endpoint :
//
//	PATCH /api/v1/players/{player_slug}/matches/{match_id}/favorite → MatchFavoriteResponse
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre le PATCH via huma.Patch. Logique métier
// inchangée (SocialService), seul le wrapping HTTP change.
//
// Le corps est lu via RawBody (pas de Body typé) pour reproduire EXACTEMENT le
// contrat de décodage d'origine : un JSON invalide renvoie 400 {invalid_body}
// (parse maison) et non le 422 de validation Huma qu'un Body typé produirait.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
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

// MatchFavoriteHandler gère les bascules de favoris de matchs.
type MatchFavoriteHandler struct {
	newSvc ServiceFactory[port.SocialService]
}

// NewMatchFavoriteHandler crée un MatchFavoriteHandler.
func NewMatchFavoriteHandler(newSvc ServiceFactory[port.SocialService]) *MatchFavoriteHandler {
	return &MatchFavoriteHandler{newSvc: newSvc}
}

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *MatchFavoriteHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Patch(api, "/matches/{match_id}/favorite", h.handlePatchFavorite, humacore.Op("patchMatchFavorite", "Toggle le statut favori d'un match", "match-view"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// matchFavoriteInput : {player_slug} parent + {match_id} + corps brut décodé
// maison. RawBody (pas Body typé) → préserve le contrat 400 {invalid_body} sur
// JSON invalide (un Body typé renverrait le 422 de validation Huma).
type matchFavoriteInput struct {
	PlayerSlug string `path:"player_slug"`
	MatchID    string `path:"match_id"`
	RawBody    []byte
}

type matchFavoriteOutput struct{ Body domain.MatchFavoriteResponse }

// ─── Endpoint ────────────────────────────────────────────────────────────────

// handlePatchFavorite bascule l'état favori d'un match pour un joueur.
// PATCH /api/v1/players/{player_slug}/matches/{match_id}/favorite
func (h *MatchFavoriteHandler) handlePatchFavorite(ctx context.Context, in *matchFavoriteInput) (*matchFavoriteOutput, error) {
	matchID := in.MatchID
	slug := in.PlayerSlug

	if matchID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_match_id", "match_id requis")
	}

	svc, err := h.newSvc(ctx, slug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	var req domain.MatchFavoriteRequest
	if err := json.NewDecoder(bytes.NewReader(in.RawBody)).Decode(&req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "corps JSON invalide")
	}
	req.PlayerSlug = slug
	req.MatchID = matchID

	if err := svc.ToggleMatchFavorite(ctx, req); err != nil {
		if errors.Is(err, dblease.ErrDBLocked) {
			return nil, huma.ErrorWithHeaders(
				humacore.NewError(http.StatusServiceUnavailable, "db_busy",
					"database is currently busy, please retry"),
				http.Header{"Retry-After": []string{"5"}},
			)
		}
		slog.ErrorContext(ctx, "match_favorite: erreur bascule",
			"err", err, "match_id", matchID, "player", slug)
		return nil, humacore.NewError(http.StatusInternalServerError, "favorite_error", err.Error())
	}

	slog.DebugContext(ctx, "match_favorite: bascule ok",
		"match_id", matchID, "player", slug, "favorited", req.Favorited)

	return &matchFavoriteOutput{Body: domain.MatchFavoriteResponse(req)}, nil
}
