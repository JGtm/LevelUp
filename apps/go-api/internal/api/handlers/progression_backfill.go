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
package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

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

// RunBackfill : POST /api/v1/_admin/progression/backfill/{player_slug}.
func (h *ProgressionBackfillHandler) RunBackfill(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	if slug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_slug", "player_slug requis")
		return
	}
	runner, err := h.newRunner(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	diag, err := runner.BackfillProgression(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "backfill_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, diag)
}
