// Package handlers — TeammatesHandler : POST /pages/teammates et
// GET /pages/teammates/sessions.
//
// Sprint 33 — contrat API Lot 4. Lot perf L4b (2026-09-23) : la route GET rend les seules
// sessions de la composition (composition_sessions, latest_composition_session) sans calculer
// la page, pour que l'Escouade s'ancre sur la bonne session AVANT la requête lourde.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre le POST via huma.Post. Logique métier
// inchangée (TeammatesService), seul le wrapping HTTP change.
//
// Le corps est lu via RawBody (pas de Body typé) pour reproduire EXACTEMENT le
// contrat de décodage d'origine : un JSON invalide renvoie 400 {invalid_json}
// (parse maison) et non le 422 de validation Huma qu'un Body typé produirait.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// TeammatesHandler gère POST /pages/teammates et GET /pages/teammates/sessions.
type TeammatesHandler struct {
	newSvc ContextFactory[port.TeammatesService]
}

// NewTeammatesHandler crée un TeammatesHandler.
func NewTeammatesHandler(newSvc ContextFactory[port.TeammatesService]) *TeammatesHandler {
	return &TeammatesHandler{newSvc: newSvc}
}

// Mount enregistre les deux routes via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités) : la route légère des
// sessions vit sous les MÊMES garde-fous que la page.
func (h *TeammatesHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/pages/teammates", h.handleGetPage, humacore.Op("postTeammatesPage", "Analyse coéquipiers (filtres en body)", "teammates"))
	huma.Get(api, "/pages/teammates/sessions", h.handleGetSessions, humacore.Op("getTeammatesSessions", "Sessions de la composition, sans la page", "teammates"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// teammatesQueryInput : {player_slug} parent + corps brut décodé maison.
// RawBody (pas Body typé) → préserve le contrat 400 {invalid_json} sur JSON
// invalide (un Body typé renverrait le 422 de validation Huma).
type teammatesQueryInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}

type teammatesPageOutput struct{ Body domain.TeammatesPageResponse }

// teammatesSessionsInput : {player_slug} parent, la composition et l'option composition
// exacte. `exact` absent vaut false, comme `filter_exact_composition` absent du corps de
// POST /pages/teammates.
type teammatesSessionsInput struct {
	PlayerSlug string   `path:"player_slug"`
	Teammates  []string `query:"teammates" doc:"Coéquipiers de la composition (gamertags séparés par des virgules). Absent ou vide : sessions escouade du joueur principal."`
	Exact      bool     `query:"exact" doc:"Option composition exacte (filter_exact_composition de POST /pages/teammates). Défaut false."`
}

// compositionSessionsResponse : les deux champs homonymes de TeammatesPageResponse, mêmes
// valeurs pour la même composition et la même option. Toujours présents (liste vide, chaîne
// vide), là où la page les omet quand ils sont vides.
type compositionSessionsResponse struct {
	CompositionSessions      []domain.CompositionSessionEntry `json:"composition_sessions"`
	LatestCompositionSession string                           `json:"latest_composition_session"`
}

type teammatesSessionsOutput struct{ Body compositionSessionsResponse }

// ─── Endpoint ────────────────────────────────────────────────────────────────

// handleGetPage traite POST /api/v1/players/{player_slug}/pages/teammates.
func (h *TeammatesHandler) handleGetPage(ctx context.Context, in *teammatesQueryInput) (*teammatesPageOutput, error) {
	svc, xuid, _, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	var req domain.TeammatesQueryRequest
	if err := json.NewDecoder(bytes.NewReader(in.RawBody)).Decode(&req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_json", "corps JSON invalide")
	}

	resp, svcErr := svc.GetPage(ctx, xuid, req)
	if svcErr != nil {
		return nil, mapServiceError(ctx, svcErr, "teammates_error")
	}

	return &teammatesPageOutput{Body: resp}, nil
}

// handleGetSessions traite GET /api/v1/players/{player_slug}/pages/teammates/sessions.
//
// Erreurs : capability absente du titre -> 503 `capability_not_supported`
// (MapCapabilityError, sonde `match.history`) ; le reste par mapServiceError, comme la page
// (499 client parti, 503 base occupée, 500 sinon).
func (h *TeammatesHandler) handleGetSessions(ctx context.Context, in *teammatesSessionsInput) (*teammatesSessionsOutput, error) {
	svc, xuid, _, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	sessions, latest, svcErr := svc.CompositionSessions(ctx, xuid, compositionDemandee(in.Teammates), in.Exact)
	if svcErr != nil {
		if mapped, ok := MapCapabilityError(ctx, svcErr, "match.history"); ok {
			return nil, mapped
		}
		return nil, mapServiceError(ctx, svcErr, "teammates_sessions_error")
	}
	if sessions == nil {
		sessions = []domain.CompositionSessionEntry{}
	}
	return &teammatesSessionsOutput{Body: compositionSessionsResponse{
		CompositionSessions:      sessions,
		LatestCompositionSession: latest,
	}}, nil
}

// compositionDemandee nettoie la composition lue dans la requête : gamertags rognés, vides
// écartés (`?teammates=` ou une virgule de trop ne désignent personne).
func compositionDemandee(raw []string) []string {
	out := make([]string, 0, len(raw))
	for _, gt := range raw {
		if gt = strings.TrimSpace(gt); gt != "" {
			out = append(out, gt)
		}
	}
	return out
}
