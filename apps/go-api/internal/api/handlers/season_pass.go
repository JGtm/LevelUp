// Package handlers — season_pass.go : handler HTTP pour la page Season Pass.
//
// Endpoint :
//
//	GET /api/v1/players/{player_slug}/pages/palmares/season-pass → SeasonPassPageResponse
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre le GET via huma.Get. Logique métier
// inchangée (SeasonPassService + contexte enrichi), seul le wrapping HTTP change.
package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// SeasonPassAuthFactory est une factory qui retourne un SeasonPassService + contexte enrichi.
type SeasonPassAuthFactory func(ctx context.Context, slug string) (svc port.SeasonPassService, enrichedCtx context.Context, err error)

// SeasonPassHandler gère l'endpoint /pages/palmares/season-pass.
type SeasonPassHandler struct {
	newSvc SeasonPassAuthFactory
}

// NewSeasonPassHandler crée un SeasonPassHandler.
func NewSeasonPassHandler(newSvc SeasonPassAuthFactory) *SeasonPassHandler {
	return &SeasonPassHandler{newSvc: newSvc}
}

// Mount enregistre la route via Huma sur le sous-routeur chi /players/{player_slug}
// (middleware ownership/title hérités). Chemin relatif complet pour reproduire le
// path absolu /players/{player_slug}/pages/palmares/season-pass de server.go.
func (h *SeasonPassHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/pages/palmares/season-pass", h.GetSeasonPass)
}

// seasonPassInput : path param parent {player_slug}.
type seasonPassInput struct {
	PlayerSlug string `path:"player_slug"`
}

type seasonPassOutput struct{ Body domain.SeasonPassPageResponse }

// GetSeasonPass retourne la page Season Pass complète.
// GET /api/v1/players/{player_slug}/pages/palmares/season-pass
func (h *SeasonPassHandler) GetSeasonPass(ctx context.Context, in *seasonPassInput) (*seasonPassOutput, error) {
	svc, enrichedCtx, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", "joueur introuvable")
	}

	page, err := svc.GetSeasonPassPage(enrichedCtx)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "season_pass_error", "erreur chargement page Season Pass")
	}

	return &seasonPassOutput{Body: page}, nil
}
