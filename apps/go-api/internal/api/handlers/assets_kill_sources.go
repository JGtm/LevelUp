// Package handlers — assets_kill_sources.go : GET /assets/{title_id}/kill-sources, le référentiel
// des pictogrammes d'arme d'un titre (clé d'arme -> nom, image, teinte).
//
// Même montage que les routes de l'Asset Drawer (Huma, {title_id} seul path param). Aucune logique
// ici : le service résout, le handler encode. Un titre sans capacité reçoit une liste vide (200),
// jamais une erreur.
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

// KillSourceIconsHandler sert le référentiel des pictogrammes d'arme.
type KillSourceIconsHandler struct {
	svc port.KillSourceIconService
}

// NewKillSourceIconsHandler crée le handler.
func NewKillSourceIconsHandler(svc port.KillSourceIconService) *KillSourceIconsHandler {
	return &KillSourceIconsHandler{svc: svc}
}

// Mount enregistre la route sur le routeur fourni (préfixe hérité du point de montage).
func (h *KillSourceIconsHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/assets/{title_id}/kill-sources", h.List, humacore.Op(
		"listKillSourceIcons",
		"Liste les pictogrammes d'arme du titre par weapon_key (noms dans la langue de la requete)",
		"assets"))
}

type killSourceIconsInput struct {
	TitleID string `path:"title_id"`
}

type killSourceIconsOutput struct {
	Body []domain.KillSourceIconEntry
}

// List répond GET /assets/{title_id}/kill-sources.
func (h *KillSourceIconsHandler) List(ctx context.Context, in *killSourceIconsInput) (*killSourceIconsOutput, error) {
	items, err := h.svc.ListKillSourceIcons(ctx, in.TitleID)
	if err != nil {
		return nil, humacore.NewErrorFrom(ctx, http.StatusInternalServerError, "internal_error", err)
	}
	if items == nil {
		items = []domain.KillSourceIconEntry{}
	}
	return &killSourceIconsOutput{Body: items}, nil
}
