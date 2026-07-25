// Package handlers — progression_backfill.go : endpoint admin/dev qui force une
// évaluation progression V2 (Ascension) pour un joueur, hors du chemin post-sync.
//
// Raison d'être : les tables streaks/records/milestones ne sont peuplées que par
// le hook post-sync EvaluateProgressionAfterSync. Pour un joueur dont l'historique
// existe déjà (BDD pleine) mais dont le pipeline n'a jamais abouti (cf. incident
// timeout shared reader, fix 2026-05-30), aucun backfill rétroactif n'existait.
// Cet endpoint déclenche une évaluation idempotente in-process et renvoie le
// diag (counts) résultant, ce qui permet de vérifier que les tables se peuplent.
//
// Usage :
//
//	curl -s -X POST http://127.0.0.1:8000/api/v1/_admin/progression/backfill/JGtm | jq
//
// L'évaluation est idempotente (PB déjà persisté → pas de doublon) : ré-appeler
// l'endpoint ne crée pas de données en double.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) au point de montage
// /api/v1 et enregistre le POST via huma.Post. Le {player_slug} est PRÉSENT dans
// le path de la route (path param). Le corps POST est OPTIONNEL et ignoré (le
// backfill ne lit pas le body) : RawBody []byte + MarkRequestBodyOptional préserve
// le 200 sur corps absent. Logique métier inchangée, seul le wrapping HTTP change.
package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
)

// ProgressionBackfiller exécute l'évaluation progression pour un joueur et
// retourne le diag (counts) post-exécution. Implémenté côté api par un adapter
// qui réutilise EvaluateProgressionAfterSync.
type ProgressionBackfiller interface {
	BackfillProgression(ctx context.Context, slug string) (*domain.ProgressionDiag, error)
}

// ProgressionBackfillFactory : résout slug → backfiller.
type ProgressionBackfillFactory func(ctx context.Context, slug string) (ProgressionBackfiller, error)

// ProgressionBackfillHandler expose l'endpoint de backfill progression.
type ProgressionBackfillHandler struct {
	newRunner ProgressionBackfillFactory
}

// NewProgressionBackfillHandler crée un handler avec une factory injectée.
func NewProgressionBackfillHandler(newRunner ProgressionBackfillFactory) *ProgressionBackfillHandler {
	return &ProgressionBackfillHandler{newRunner: newRunner}
}

// Mount enregistre la route via Huma au point de montage /api/v1 (le path complet
// /_admin/progression/backfill/{player_slug} est relatif à ce routeur). Le corps
// POST est OPTIONNEL (MarkRequestBodyOptional) — il n'est pas lu par le backfill.
func (h *ProgressionBackfillHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/_admin/progression/backfill/{player_slug}", h.handleRunBackfill)
	humacore.MarkRequestBodyOptional(api, http.MethodPost, "/_admin/progression/backfill/{player_slug}")
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// progressionBackfillInput : {player_slug} (présent dans le path de la route) +
// corps OPTIONNEL ignoré (RawBody, rendu non requis via MarkRequestBodyOptional).
type progressionBackfillInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}

type progressionBackfillOutput struct {
	Body *domain.ProgressionDiag
}

// ─── Endpoint ────────────────────────────────────────────────────────────────

// handleRunBackfill : POST /api/v1/_admin/progression/backfill/{player_slug}.
func (h *ProgressionBackfillHandler) handleRunBackfill(ctx context.Context, in *progressionBackfillInput) (*progressionBackfillOutput, error) {
	if in.PlayerSlug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_slug", "player_slug requis")
	}
	runner, err := h.newRunner(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	diag, err := runner.BackfillProgression(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "backfill_error", err.Error())
	}
	return &progressionBackfillOutput{Body: diag}, nil
}
