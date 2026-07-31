// Package handlers — replay.go : endpoint
// GET /players/{player_slug}/matches/{match_id}/replay.
//
// Sert l'artefact de rejeu 2D pré-construit (trajectoires joueurs vue du dessus). Monté
// sous le groupe /players/{player_slug} (ownership transparent en mono-user) ; 404 propre
// quand aucun artefact n'existe pour le match — la feature s'allume par présence
// d'artefact, pas par flag global.
//
// Le garde local (le rejeu n'est servi qu'en local tant que ses écarts n'ont pas été
// confrontés à un second film) est un middleware de TRANSPORT, pas une branche du
// handler : il lit l'adresse de la connexion TCP, que la couche Huma n'expose pas.
// Cf. replay_local_gate.go, qui porte la date de bascule, la cible de retrait et le
// critère mesurable.
package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/api/humacore"
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

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middlewares ownership/titre/garde local hérités).
func (h *ReplayHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/matches/{match_id}/replay", h.handleGetReplay,
		humacore.Op("getMatchReplay", "Rejeu 2D pré-construit d'un match", "match-view"))
}

// replayInput : {player_slug} parent + {match_id}. match_id pris en STRING pour
// reproduire le contrat d'origine — un match_id vide renvoie 400 missing_match_id.
type replayInput struct {
	PlayerSlug string `path:"player_slug"`
	MatchID    string `path:"match_id"`
}

type replayOutput struct{ Body replay.ReplayDocument }

// handleGetReplay retourne le document de rejeu 2D d'un match (404 si absent).
func (h *ReplayHandler) handleGetReplay(ctx context.Context, in *replayInput) (*replayOutput, error) {
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	if in.MatchID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_match_id", "match_id est requis")
	}

	doc, err := svc.GetReplay(ctx, in.MatchID)
	if errors.Is(err, port.ErrReplayNotAvailable) {
		return nil, humacore.NewError(http.StatusNotFound, "replay_not_available",
			"aucun rejeu 2D pour ce match")
	}
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "replay_error", err.Error())
	}
	return &replayOutput{Body: doc}, nil
}
