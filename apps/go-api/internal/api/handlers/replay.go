// Package handlers — replay.go : endpoint
// GET /players/{player_slug}/matches/{match_id}/replay.
//
// Sert l'artefact de rejeu 2D pré-construit (trajectoires joueurs vue du dessus). Monté
// sous le groupe /players/{player_slug} (ownership transparent en mono-user) ; 404 propre
// quand aucun artefact n'existe pour le match — la feature s'allume par présence
// d'artefact, pas par flag global.
package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/port"
)

// ReplayHandler sert le rejeu 2D d'un match via un ReplayService résolu par joueur.
type ReplayHandler struct {
	newSvc ServiceFactory[port.ReplayService]
}

// NewReplayHandler construit le handler avec sa factory de service.
func NewReplayHandler(newSvc ServiceFactory[port.ReplayService]) *ReplayHandler {
	return &ReplayHandler{newSvc: newSvc}
}

// GetReplay retourne le document de rejeu 2D d'un match (404 si absent).
func (h *ReplayHandler) GetReplay(w http.ResponseWriter, r *http.Request) {
	// Garde local : le rejeu reste hors de portée d'un client distant tant que ses écarts
	// n'ont pas été confrontés à un second film (cf. replay_local_gate.go, qui porte la date
	// de bascule, la cible de retrait et le critère mesurable).
	if !allowReplay(r) {
		writeError(r.Context(), w, http.StatusNotFound, "replay_not_available",
			"le rejeu 2D n'est servi qu'en local")
		return
	}
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	matchID := chi.URLParam(r, "match_id")
	if matchID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_match_id", "match_id est requis")
		return
	}

	doc, err := svc.GetReplay(r.Context(), matchID)
	if errors.Is(err, port.ErrReplayNotAvailable) {
		writeError(r.Context(), w, http.StatusNotFound, "replay_not_available", "aucun rejeu 2D pour ce match")
		return
	}
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "replay_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, doc)
}
