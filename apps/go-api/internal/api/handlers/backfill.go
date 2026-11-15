// Package handlers — backfill.go : lancement du pipeline de backfill.
//
// POST /backfill/start → crée un job "backfill" et retourne 202 immédiatement.
// Règles :
//   - 400 si player_slug absent ou scope entièrement vide
//   - 404 si le joueur est introuvable dans db_profiles.json (pour le titre demandé)
//   - 409 si un job backfill actif existe déjà pour ce player_slug
//
// Mount crée humacore.NewAPI(r) sur le routeur chi (point de montage /backfill/start, middleware
// RequireAuth/RequireAdmin hérités) et enregistre le POST via huma.Post. Le corps est lu via RawBody
// (pas de Body typé) : un JSON invalide renvoie 400 {invalid_body} et non le 422 de validation Huma ;
// un corps absent est refusé par Huma avant le handler (400 bad_request). Le handler ne garde que le décodage, la validation du slug et le 202 ; le
// joueur, le moteur et l'orchestration multi-phase vivent derrière port.BackfillService.
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

// BackfillHandler expose le lancement du pipeline de backfill.
type BackfillHandler struct {
	svc port.BackfillService
}

// NewBackfillHandler crée un BackfillHandler.
func NewBackfillHandler(svc port.BackfillService) *BackfillHandler {
	return &BackfillHandler{svc: svc}
}

// Mount enregistre la route via Huma sur le routeur chi (point de montage
// /backfill/start, middleware RequireAuth/RequireAdmin hérités du groupe).
func (h *BackfillHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/backfill/start", h.handleStartBackfill, humacore.Op("postBackfillStart", "Lance un backfill (job long)", "sync"),
		humacore.DefaultStatus(http.StatusAccepted))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// backfillStartInput : corps brut décodé maison. RawBody (pas Body typé) →
// préserve le contrat 400 {invalid_body} sur JSON invalide (un Body typé
// renverrait le 422 de validation Huma).
type backfillStartInput struct {
	RawBody []byte
}

// backfillStartOutput : 202 Accepted, corps = snapshot du job créé (le 202 est
// posé par humacore.DefaultStatus au montage — source unique, runtime ET document).
type backfillStartOutput struct {
	Body *domain.AsyncJobStatus
}

// ─── Endpoint ────────────────────────────────────────────────────────────────

// handleStartBackfill déclenche le pipeline backfill pour un joueur.
// POST /backfill/start → 202 AsyncJobStatus.
func (h *BackfillHandler) handleStartBackfill(ctx context.Context, in *backfillStartInput) (*backfillStartOutput, error) {
	var req domain.BackfillStartRequest
	if err := json.Unmarshal(in.RawBody, &req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "Corps de requête JSON invalide.")
	}
	if err := checkPlayerSlug(req.PlayerSlug); err != nil {
		return nil, err
	}
	job, err := h.svc.StartBackfill(ctx, syncCaller(ctx), req)
	if err != nil {
		return nil, syncJobHTTPError(ctx, err)
	}
	return &backfillStartOutput{Body: job}, nil
}
