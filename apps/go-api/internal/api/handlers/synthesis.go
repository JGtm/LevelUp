// Package handlers — synthesis.go : handler HTTP pour la page Synthèse.
//
// Sprint 55 D1 : extrait de squad.go — SynthesisHandler devient autonome.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre le POST via huma.Post. Logique métier
// inchangée (SynthesisService), seul le wrapping HTTP change.
//
// Le corps est lu via RawBody (pas de Body typé) pour reproduire EXACTEMENT le
// contrat de décodage d'origine : un JSON invalide renvoie 400 {invalid_body}
// (parse maison) et non le 422 de validation Huma qu'un Body typé produirait.
// Corps absent (ContentLength == 0) → requête vide tolérée.
//
// Endpoint :
//
//	POST /api/v1/players/{player_slug}/pages/synthesis → SynthesisPageV2Response
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// SynthesisHandler gère l'endpoint de la page Synthèse.
// Sprint 55 D1 : séparé de SquadHandler pour refléter la frontière produit.
type SynthesisHandler struct {
	newSvc ContextFactory[port.SynthesisService]
}

// NewSynthesisHandler crée un SynthesisHandler.
func NewSynthesisHandler(newSvc ContextFactory[port.SynthesisService]) *SynthesisHandler {
	return &SynthesisHandler{newSvc: newSvc}
}

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *SynthesisHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/pages/synthesis", h.handleGetSynthesisPage)
	// Body OPTIONNEL (décodé seulement si présent) : RawBody requis par défaut
	// côté Huma → optionnel pour préserver le 200 sur corps absent.
	humacore.MarkRequestBodyOptional(api, http.MethodPost, "/pages/synthesis")
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// synthesisInput : {player_slug} parent + corps brut décodé maison.
// RawBody (pas Body typé) → préserve le contrat 400 {invalid_body} sur JSON
// invalide (un Body typé renverrait le 422 de validation Huma). Corps vide
// (RawBody nil) → requête zéro-valeur tolérée (reproduit ContentLength > 0).
type synthesisInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}

type synthesisPageOutput struct {
	Body *domain.SynthesisPageV2Response
}

// ─── Endpoint ────────────────────────────────────────────────────────────────

// handleGetSynthesisPage retourne la page Synthèse avec scope explicite et filtres appliqués.
// POST /api/v1/players/{player_slug}/pages/synthesis
// Body (optionnel) : { "period": "1m", "filters": {...} }
func (h *SynthesisHandler) handleGetSynthesisPage(ctx context.Context, in *synthesisInput) (*synthesisPageOutput, error) {
	svc, xuid, _, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		slog.WarnContext(ctx, "synthesis: joueur introuvable", "player_slug", in.PlayerSlug, "err", err)
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	var req domain.SynthesisRequest
	if len(in.RawBody) > 0 {
		if err := json.NewDecoder(bytes.NewReader(in.RawBody)).Decode(&req); err != nil {
			slog.WarnContext(ctx, "synthesis: corps de requête invalide", "player_slug", in.PlayerSlug, "err", err)
			return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
		}
	}

	slog.DebugContext(ctx, "synthesis: calcul page", "player_slug", in.PlayerSlug, "period", req.Period)

	page, err := svc.GetSynthesisPage(ctx, xuid, req)
	if err != nil {
		slog.ErrorContext(ctx, "synthesis: erreur service", "player_slug", in.PlayerSlug, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "synthesis_page_error", err.Error())
	}

	slog.InfoContext(ctx, "synthesis: page générée",
		"player_slug", in.PlayerSlug,
		"period", req.Period,
		"match_count", page.Scope.MatchCount,
	)
	return &synthesisPageOutput{Body: page}, nil
}
