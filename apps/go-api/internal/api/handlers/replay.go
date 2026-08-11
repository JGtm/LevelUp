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
	"log/slog"
	"net/http"
	"strconv"

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
//
// L'IMAGE DU FOND reste une route chi NUE, comme l'export CSV de l'historique : Huma décrit
// des charges utiles JSON typées, et forcer 700 Kio de PNG dans ce moule n'apporterait qu'un
// schéma qui ment. Elle hérite des mêmes middlewares (ownership, titre, garde local) puisque
// c'est le même sous-routeur.
func (h *ReplayHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/matches/{match_id}/replay", h.handleGetReplay,
		humacore.Op("getMatchReplay", "Rejeu 2D pré-construit d'un match", "match-view"))
	huma.Get(api, "/matches/{match_id}/replay/background", h.handleGetBackground,
		humacore.Op("getMatchReplayBackground",
			"Calage du fond de carte du rejeu 2D d'un match", "match-view"))
	r.Get("/matches/{match_id}/replay/background.png", h.handleGetBackgroundImage)
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

type backgroundOutput struct{ Body replay.MapBackground }

// handleGetBackground retourne le CALAGE du fond de carte du match : mètres par pixel,
// origine monde, taille de l'image, plus les statistiques de cuisson et les dégradations
// déclarées. 404 quand la carte du match n'a pas d'image figée — une absence normale
// (21 cartes en ont), que le client traduit en repli sur le sol reconstruit.
func (h *ReplayHandler) handleGetBackground(ctx context.Context, in *replayInput) (*backgroundOutput, error) {
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	if in.MatchID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_match_id", "match_id est requis")
	}
	bg, err := svc.MapBackground(ctx, in.MatchID)
	if errors.Is(err, port.ErrMapBackgroundNotAvailable) {
		return nil, humacore.NewError(http.StatusNotFound, "map_background_not_available",
			"aucun fond de carte pour ce match")
	}
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "replay_error", err.Error())
	}
	return &backgroundOutput{Body: *bg}, nil
}

// handleGetBackgroundImage sert le PNG du fond de carte.
//
// Route chi nue (cf. Mount) : la charge utile est binaire. Elle rend les mêmes 404 que le
// calage, et pour les mêmes raisons — l'image et son calage vont toujours ensemble.
func (h *ReplayHandler) handleGetBackgroundImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slug := chi.URLParam(r, "player_slug")
	matchID := chi.URLParam(r, "match_id")
	svc, err := h.newSvc(ctx, slug)
	if err != nil {
		writeError(ctx, w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	if matchID == "" {
		writeError(ctx, w, http.StatusBadRequest, "missing_match_id", "match_id est requis")
		return
	}
	blob, err := svc.MapBackgroundImage(ctx, matchID)
	if errors.Is(err, port.ErrMapBackgroundNotAvailable) {
		writeError(ctx, w, http.StatusNotFound, "map_background_not_available",
			"aucun fond de carte pour ce match")
		return
	}
	if err != nil {
		writeError(ctx, w, http.StatusInternalServerError, "replay_error", err.Error())
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(blob)))
	// Donnée de RÉFÉRENCE versionnée : elle ne change qu'à une re-cuisson, jamais en
	// cours de session. `private` parce que la route est derrière l'ownership joueur.
	w.Header().Set("Cache-Control", "private, max-age=3600")
	if _, err := w.Write(blob); err != nil {
		slog.WarnContext(ctx, "rejeu 2D : écriture de l'image de fond interrompue",
			"err", err, "match_id", matchID, "player", slug)
	}
}
