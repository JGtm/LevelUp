// Package handlers — MatchEventsHandler : GET .../matches/{match_id}/events.
//
// Surface canonique de la timeline d'events d'un match (kill-feed / timeline),
// chargée on-demand. Huma (struct Output, jamais writeJSON). Monté sur le
// sous-routeur /players/{player_slug} → ownership + title hérités du groupe.
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// MatchEventsHandler gère GET /players/{player_slug}/matches/{match_id}/events.
type MatchEventsHandler struct {
	newSvc ServiceFactory[port.MatchEventsService]
}

// NewMatchEventsHandler crée un MatchEventsHandler.
func NewMatchEventsHandler(newSvc ServiceFactory[port.MatchEventsService]) *MatchEventsHandler {
	return &MatchEventsHandler{newSvc: newSvc}
}

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *MatchEventsHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/matches/{match_id}/events", h.handleGetMatchEvents, humacore.Op("getMatchEvents", "Timeline canonique d'events d'un match (kill-feed / timeline)", "match-view"))
}

// matchEventsInput : {player_slug} parent + {match_id} + filtre optionnel par
// type d'event (?types=kill,medal — répétable ou séparé par virgule).
type matchEventsInput struct {
	PlayerSlug string   `path:"player_slug"`
	MatchID    string   `path:"match_id"`
	Types      []string `query:"types" doc:"Filtre optionnel par type d'event (kill, medal, impulse, ...). Répétable ou CSV. Vide = tous."`
}

type matchEventsOutput struct{ Body canonical.MatchEventTimeline }

// handleGetMatchEvents retourne la timeline canonique d'events d'un match.
// GET /api/v1/players/{player_slug}/matches/{match_id}/events
func (h *MatchEventsHandler) handleGetMatchEvents(ctx context.Context, in *matchEventsInput) (*matchEventsOutput, error) {
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	matchID := strings.TrimSpace(in.MatchID)
	if matchID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_match_id", "match_id est requis")
	}

	opts, err := parseMatchEventOptions(in.Types)
	if err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_event_type", err.Error())
	}

	tl, err := svc.GetMatchEvents(ctx, matchID, opts)
	if err != nil {
		if mapped, ok := MapCapabilityError(ctx, err, "match.detail.events"); ok {
			return nil, mapped
		}
		return nil, humacore.NewError(http.StatusInternalServerError, "match_events_error", err.Error())
	}
	if tl == nil {
		tl = &canonical.MatchEventTimeline{MatchID: matchID}
	}
	return &matchEventsOutput{Body: *tl}, nil
}

// parseMatchEventOptions valide le filtre ?types. Chaque valeur (répétée ou CSV)
// doit être un MatchEventType connu, sinon 400. Vide → aucun filtre (tous les types).
func parseMatchEventOptions(raw []string) (canonical.MatchEventOptions, error) {
	if len(raw) == 0 {
		return canonical.MatchEventOptions{}, nil
	}
	out := make([]canonical.MatchEventType, 0, len(raw))
	for _, group := range raw {
		for _, tok := range strings.Split(group, ",") {
			tok = strings.TrimSpace(tok)
			if tok == "" {
				continue
			}
			mt := canonical.MatchEventType(tok)
			if !canonical.IsKnownMatchEventType(mt) {
				return canonical.MatchEventOptions{}, fmt.Errorf("type d'event inconnu: %q", tok)
			}
			out = append(out, mt)
		}
	}
	return canonical.MatchEventOptions{Types: out}, nil
}
