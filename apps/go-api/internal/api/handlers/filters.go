// Package handlers — FiltersHandler : POST .../filters/resolve | .../filters/match-ids.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre les 2 POST via huma.Post. Logique métier
// inchangée (FiltersService), seul le wrapping HTTP change.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// FiltersHandler gère POST .../filters/resolve et .../filters/match-ids.
type FiltersHandler struct {
	newSvc ServiceFactory[port.FiltersService]
}

// NewFiltersHandler crée un FiltersHandler.
func NewFiltersHandler(newSvc ServiceFactory[port.FiltersService]) *FiltersHandler {
	return &FiltersHandler{newSvc: newSvc}
}

// Mount enregistre les 2 routes via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *FiltersHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/filters/resolve", h.Resolve, humacore.Op("resolveFilters", "Résoudre le contexte de filtres", "filters"))
	huma.Post(api, "/filters/match-ids", h.MatchIDs, humacore.Op("filtersMatchIDs", "Résoudre la liste des match_ids correspondant à un contexte de filtres", "filters"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// filtersInput : {player_slug} parent + corps BRUT décodé à la main.
//
// RawBody (et non un Body typé Huma) : le front envoie period.start_date/end_date
// = null. Huma traite *time.Time comme optionnel mais PAS nullable et rejette le
// null en 422 (validation_error). On décode donc manuellement via json.Unmarshal
// (permissif : null → *time.Time nil), comme avant la migration Huma.
type filtersInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}

// decodeFiltersBody décode le corps brut en FilterContextInput (json.Unmarshal
// accepte les null que le schéma Huma rejetterait) puis applique la validation métier.
func decodeFiltersBody(raw []byte) (domain.FilterContextInput, error) {
	var body domain.FilterContextInput
	if err := json.Unmarshal(raw, &body); err != nil {
		return body, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	if err := body.Validate(); err != nil {
		return body, humacore.NewError(http.StatusBadRequest, "invalid_filters", err.Error())
	}
	return body, nil
}

type filtersResolveOutput struct{ Body domain.FilterContextResolved }
type filtersMatchIDsOutput struct{ Body domain.FilterMatchIDsResponse }

// ─── Endpoints ───────────────────────────────────────────────────────────────

// Resolve applique le filtre et retourne les options disponibles.
func (h *FiltersHandler) Resolve(ctx context.Context, in *filtersInput) (*filtersResolveOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}

	body, err := decodeFiltersBody(in.RawBody)
	if err != nil {
		return nil, err
	}

	result, err := svc.Resolve(ctx, body)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "filters_error", err.Error())
	}
	return &filtersResolveOutput{Body: result}, nil
}

// MatchIDs retourne la liste ordonnée (start_time DESC) des match_id de la
// sélection. Alimente le bouton "Voir les matchs" : le front ouvre le 1er
// match et parcourt la liste via prev/next. Même pipeline de filtrage que
// Resolve → respecte match_context (solo/squad), sessions, période et cascade.
func (h *FiltersHandler) MatchIDs(ctx context.Context, in *filtersInput) (*filtersMatchIDsOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}

	body, err := decodeFiltersBody(in.RawBody)
	if err != nil {
		return nil, err
	}

	ids, err := svc.ResolveMatchIDs(ctx, body)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "filters_error", err.Error())
	}
	// Slice jamais nil : un slice Go nil sérialise en JSON `null`, ce qui casse
	// `.length`/`.map` côté front. Cf. testutil.RequireNoNilSlicesWithoutOmitempty.
	if ids == nil {
		ids = []string{}
	}
	return &filtersMatchIDsOutput{Body: domain.FilterMatchIDsResponse{MatchIDs: ids}}, nil
}

// resolve résout le slug courant en FiltersService ou renvoie une erreur Huma 404
// (contrat préservé : {code:player_not_found}).
func (h *FiltersHandler) resolve(ctx context.Context, slug string) (port.FiltersService, error) {
	svc, err := h.newSvc(ctx, slug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	return svc, nil
}
