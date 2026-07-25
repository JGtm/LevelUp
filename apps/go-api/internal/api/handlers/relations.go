// Package handlers — relations.go : handler HTTP du hub Communauté > Relations.
//
// Endpoint :
//
//	POST /api/v1/players/{player_slug}/pages/palmares/relations → RelationsPageResponse
//
// Phase 2 : passé en POST (comme /pages/synthesis) pour accepter un
// FilterContextInput en body (segmentation serveur : expérience/classé,
// saison/période, playlist/mode, vue solo/escouade). Corps absent → sélection
// zéro-valeur tolérée (= comportement Phase 1, tous les matchs). JSON invalide
// → 400 {invalid_body} via parse maison (RawBody, pas Body typé).
//
// Page transverse (non capability-gated) : Mount crée humacore.NewAPI(r) sur le
// sous-routeur /players/{player_slug} (middleware ownership/title hérités).
// Aucune logique métier / SQL ici : decode → service → encode.
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

// Mount enregistre les routes via Huma sur le sous-routeur chi /players/{player_slug}.
func (h *RelationsHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/pages/palmares/relations", h.GetRelations, humacore.Op(
		"postRelationsPage",
		"Hub Communauté > Relations segmenté (filtres en body : expérience, saison/période, playlist/mode, vue solo/escouade)",
		"palmares"))
	// Body OPTIONNEL (décodé seulement si présent) — préserve le 200 sur corps absent.
	humacore.MarkRequestBodyOptional(api, http.MethodPost, "/pages/palmares/relations")

	// Phase 3a : section « Moments & Rivalités » (heatmap relation × tranche
	// horaire + cartes revanche). Même FilterContextInput optionnel en body.
	huma.Post(api, "/pages/palmares/relations/moments", h.GetMoments, humacore.Op(
		"postRelationsMoments",
		"Section Moments & Rivalités (heatmap relation x tranche horaire + cartes revanche), segmentée par les memes filtres en body",
		"palmares"))
	humacore.MarkRequestBodyOptional(api, http.MethodPost, "/pages/palmares/relations/moments")
}

// relationsInput : path param parent {player_slug} + corps brut décodé maison.
// RawBody (pas Body typé) → 400 {invalid_body} sur JSON invalide (un Body typé
// renverrait le 422 Huma). Corps vide → FilterContextInput zéro-valeur (= tout).
type relationsInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}

type relationsOutput struct{ Body domain.RelationsPageResponse }

type relationsMomentsOutput struct {
	Body domain.RelationsMomentsResponse
}

// decodeFilterBody décode le corps brut optionnel en FilterContextInput. Corps
// vide → input zéro-valeur (= tout). JSON invalide / filtre invalide → erreur
// Huma 400 {invalid_body}. Mutualisé entre GetRelations et GetMoments.
func decodeFilterBody(ctx context.Context, slug string, rawBody []byte) (domain.FilterContextInput, error) {
	var input domain.FilterContextInput
	if len(rawBody) == 0 {
		return input, nil
	}
	if err := json.NewDecoder(bytes.NewReader(rawBody)).Decode(&input); err != nil {
		slog.WarnContext(ctx, "relations.invalid_body", "player", slug, "err", err)
		return input, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	if err := input.Validate(); err != nil {
		slog.WarnContext(ctx, "relations.invalid_filter", "player", slug, "err", err)
		return input, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	return input, nil
}

// GetRelations retourne le hub Relations, segmenté par le FilterContextInput du body.
// POST /api/v1/players/{player_slug}/pages/palmares/relations
// Body (optionnel) : FilterContextInput { experience/playlists/modes via cascade,
// period, sessions, match_context (solo|squad|all) }.
func (h *RelationsHandler) GetRelations(ctx context.Context, in *relationsInput) (*relationsOutput, error) {
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		slog.WarnContext(ctx, "relations.player_not_found", "player", in.PlayerSlug, "err", err)
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", "joueur introuvable")
	}

	input, err := decodeFilterBody(ctx, in.PlayerSlug, in.RawBody)
	if err != nil {
		return nil, err
	}

	page, err := svc.GetRelationsPage(ctx, input)
	if err != nil {
		slog.ErrorContext(ctx, "relations.page_error", "player", in.PlayerSlug, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "relations_error", "erreur chargement hub Relations")
	}

	return &relationsOutput{Body: page}, nil
}

// GetMoments retourne la section « Moments & Rivalités » (heatmap relation ×
// tranche horaire + cartes revanche), segmentée par le FilterContextInput.
// POST /api/v1/players/{player_slug}/pages/palmares/relations/moments
func (h *RelationsHandler) GetMoments(ctx context.Context, in *relationsInput) (*relationsMomentsOutput, error) {
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		slog.WarnContext(ctx, "relations.player_not_found", "player", in.PlayerSlug, "err", err)
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", "joueur introuvable")
	}

	input, err := decodeFilterBody(ctx, in.PlayerSlug, in.RawBody)
	if err != nil {
		return nil, err
	}

	moments, err := svc.GetRelationsMoments(ctx, input)
	if err != nil {
		slog.ErrorContext(ctx, "relations.moments_error", "player", in.PlayerSlug, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "relations_error", "erreur chargement Moments & Rivalités")
	}

	return &relationsMomentsOutput{Body: moments}, nil
}
