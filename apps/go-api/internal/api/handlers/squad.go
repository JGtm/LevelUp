// Package handlers — squad.go : handler HTTP pour la page Escouade.
//
// Endpoint :
//
//	GET /api/v1/players/{player_slug}/pages/squad             → SquadPageResponse
//	GET /api/v1/players/{player_slug}/pages/squad?teammate=xuid
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre la route via huma.Get. Logique métier
// inchangée (SquadService), seul le wrapping HTTP change.
//
// Note : la page Synthèse (POST /pages/synthesis) est servie depuis Sprint 55 D1
// par SynthesisHandler (synthesis.go) — pas par ce handler.
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

// SquadHandler gère l'endpoint de la page Escouade.
type SquadHandler struct {
	newSvc ContextFactory[port.SquadService]
}

// NewSquadHandler crée un SquadHandler.
func NewSquadHandler(newSvc ContextFactory[port.SquadService]) *SquadHandler {
	return &SquadHandler{newSvc: newSvc}
}

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *SquadHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/pages/squad", h.handleGetSquadPage)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// squadPageInput : {player_slug} + ?teammate= optionnel (toléré vide).
type squadPageInput struct {
	PlayerSlug string `path:"player_slug"`
	Teammate   string `query:"teammate"`
}

type squadPageOutput struct{ Body *domain.SquadPageResponse }

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleGetSquadPage retourne la page Escouade.
// GET /api/v1/players/{player_slug}/pages/squad[?teammate=xuid]
func (h *SquadHandler) handleGetSquadPage(ctx context.Context, in *squadPageInput) (*squadPageOutput, error) {
	svc, xuid, gamertag, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}

	page, err := svc.GetSquadPage(ctx, xuid, gamertag, in.Teammate)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "squad_page_error", err.Error())
	}

	return &squadPageOutput{Body: page}, nil
}

// resolve résout le slug courant en SquadService (+ xuid/gamertag) ou renvoie une
// erreur Huma 404 (contrat préservé : {code:player_not_found}).
func (h *SquadHandler) resolve(ctx context.Context, slug string) (port.SquadService, string, string, error) {
	svc, xuid, gamertag, err := h.newSvc(ctx, slug)
	if err != nil {
		return nil, "", "", humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	return svc, xuid, gamertag, nil
}
