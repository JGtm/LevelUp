// Package handlers — compare.go : handler HTTP pour la comparaison joueur vs joueur.
//
// Endpoint :
//
//	POST /api/v1/players/{player_slug}/pages/compare → CompareResponse
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre le POST via huma.Post. Logique métier
// inchangée (CompareService + enrichedCtx), seul le wrapping HTTP change.
//
// Le corps est lu via RawBody (pas de Body typé) pour reproduire EXACTEMENT le
// contrat de décodage d'origine : un JSON invalide renvoie 400 {invalid_body}
// (parse maison) et non le 422 de validation Huma qu'un Body typé produirait. Le
// corps reste REQUIS (RawBody seul suffit, pas de MarkRequestBodyOptional).
//
// Sprint 54 C.
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

// CompareAuthFactory retourne un CompareService + contexte enrichi avec les HaloTokens.
// Même pattern que HomeAuthFactory.
type CompareAuthFactory func(ctx context.Context, slug string) (svc port.CompareService, enrichedCtx context.Context, xuid, gamertag string, err error)

// CompareHandler gère l'endpoint de comparaison joueur vs joueur.
type CompareHandler struct {
	newSvc CompareAuthFactory
}

// NewCompareHandler crée un CompareHandler.
func NewCompareHandler(newSvc CompareAuthFactory) *CompareHandler {
	return &CompareHandler{newSvc: newSvc}
}

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *CompareHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/pages/compare", h.PostComparePage, humacore.Op("postComparePage", "Comparaison joueur vs joueur (12 KPIs)", "compare"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// compareInput : {player_slug} parent + corps brut décodé maison.
// RawBody (pas Body typé) → préserve le contrat 400 {invalid_body} sur JSON
// invalide (un Body typé renverrait le 422 de validation Huma). Corps REQUIS :
// RawBody seul suffit (Huma exige le body par défaut).
type compareInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}

type comparePageOutput struct{ Body domain.CompareResponse }

// ─── Endpoint ────────────────────────────────────────────────────────────────

// PostComparePage compare le joueur local (slug) avec un joueur cible (body).
// POST /api/v1/players/{player_slug}/pages/compare
func (h *CompareHandler) PostComparePage(ctx context.Context, in *compareInput) (*comparePageOutput, error) {
	svc, enrichedCtx, _, _, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	var req domain.CompareRequest
	if err := json.NewDecoder(bytes.NewReader(in.RawBody)).Decode(&req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	if err := req.Validate(); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "validation_error", err.Error())
	}

	resp, err := svc.GetPage(enrichedCtx, req)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "compare_error", err.Error())
	}

	return &comparePageOutput{Body: resp}, nil
}
