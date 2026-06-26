// Package handlers — relations.go : handler HTTP du hub Communauté > Relations.
//
// Endpoint :
//
//	GET /api/v1/players/{player_slug}/pages/palmares/relations → RelationsPageResponse
//
// Page transverse (non capability-gated) : Mount crée humacore.NewAPI(r) sur le
// sous-routeur /players/{player_slug} (middleware ownership/title hérités).
// Aucune logique métier / SQL ici : decode → service → encode.
package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// RelationsFactory résout slug → RelationsService.
type RelationsFactory func(ctx context.Context, slug string) (svc port.RelationsService, err error)

// RelationsHandler gère l'endpoint /pages/palmares/relations.
type RelationsHandler struct {
	newSvc RelationsFactory
}

// NewRelationsHandler crée un RelationsHandler.
func NewRelationsHandler(newSvc RelationsFactory) *RelationsHandler {
	return &RelationsHandler{newSvc: newSvc}
}

// Mount enregistre la route via Huma sur le sous-routeur chi /players/{player_slug}.
func (h *RelationsHandler) Mount(r chi.Router) {
	api := humacore.NewAPI(r)
	huma.Get(api, "/pages/palmares/relations", h.GetRelations)
}

// relationsInput : path param parent {player_slug}.
type relationsInput struct {
	PlayerSlug string `path:"player_slug"`
}

type relationsOutput struct{ Body domain.RelationsPageResponse }

// GetRelations retourne le hub Relations complet.
// GET /api/v1/players/{player_slug}/pages/palmares/relations
func (h *RelationsHandler) GetRelations(ctx context.Context, in *relationsInput) (*relationsOutput, error) {
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		slog.WarnContext(ctx, "relations.player_not_found", "player", in.PlayerSlug, "err", err)
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", "joueur introuvable")
	}

	page, err := svc.GetRelationsPage(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "relations.page_error", "player", in.PlayerSlug, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "relations_error", "erreur chargement hub Relations")
	}

	return &relationsOutput{Body: page}, nil
}
