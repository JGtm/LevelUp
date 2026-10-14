// Package handlers — tactical_sequence.go : la SEQUENCE D'UNE MORT de l'onglet Tactique (vue D).
//
//	POST /players/{player_slug}/tactical/{map_id}/sequence   (corps : match_id, victim_xuid, time_ms)
//
// UN HANDLER A PART, MONTE DANS LE GROUPE DU REJEU 2D. La sequence lit les TRAJECTOIRES decodees du
// film, comme `/replay` : elle doit suivre EXACTEMENT les memes gardes — `RequireCapability(CapReplay)`
// (un titre sans rejeu rend 503) et `LocalOnlyReplay` (le rejeu n'est servi qu'en local jusqu'au
// reexamen de la decision du 2026-07-28, cf. replay_local_gate.go). Activer le rejeu en production
// active donc la vue D sans autre changement. Les autres routes de l'onglet (`TacticalHandler`) restent
// hors de ce groupe : elles ne lisent que la base et les sidecars.
//
// C'EST UNE LECTURE EN POST (le corps porte la cle de la mort), declaree comme telle par le prefixe
// `/tactical/` de `middleware.readOnlyPostPrefixes`.
//
// ZERO LOGIQUE ICI. Le handler valide la forme des identifiants, delegue, traduit un refus en statut.
// Match inconnu, match d'un autre joueur, artefact absent, horloge non etablie, carte sans rayon :
// TOUS rendent le meme 404 `replay_not_available` — ils sont indiscernables par construction.
package handlers

import (
	"context"
	"errors"
	"net/http"
	"regexp"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// TacticalSequenceHandler sert la sequence d'une mort.
type TacticalSequenceHandler struct {
	newSvc ServiceFactory[port.TacticalService]
}

// NewTacticalSequenceHandler construit le handler.
func NewTacticalSequenceHandler(newSvc ServiceFactory[port.TacticalService]) *TacticalSequenceHandler {
	return &TacticalSequenceHandler{newSvc: newSvc}
}

// Mount enregistre la route sur le sous-routeur chi (groupe du rejeu).
func (h *TacticalSequenceHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/tactical/{map_id}/sequence", h.handleGetSequence,
		humacore.Op("getTacticalSequence", "Sequence d'une mort : traces de la victime, du tueur et des coequipiers de -8 a +3 s, et etapes du cercle de portee", "tactical"))
}

type tacticalSequenceBody struct {
	MatchID    string `json:"match_id" doc:"Le match de la mort."`
	VictimXUID string `json:"victim_xuid" doc:"La victime (xuid decimal) — celle de la contribution de face « mort »."`
	TimeMs     int64  `json:"time_ms" doc:"L'instant de la mort sur l'horloge du MATCH (TacticalContribution.instant_ms)."`
}

type tacticalSequenceInput struct {
	PlayerSlug string `path:"player_slug"`
	MapID      string `path:"map_id"`
	Body       tacticalSequenceBody
}

type tacticalSequenceOutput struct{ Body domain.TacticalSequence }

// messageSansSequence : LE message de tous les refus de la route (cf. l en-tete).
const messageSansSequence = "aucune sequence disponible pour cette mort"

// motif de forme : un match_id est un uuid (le xuid est valide par domain.XUIDValide). LISTE BLANCHE, comme
// `MapIDValide` — rien d'autre n'atteint le service ni, derriere lui, le chemin d'un artefact.
var (
	motifMatchSequence = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,63}$`)
)

// handleGetSequence rend la sequence d'une mort du joueur de la page.
func (h *TacticalSequenceHandler) handleGetSequence(ctx context.Context, in *tacticalSequenceInput) (*tacticalSequenceOutput, error) {
	mapID, okMap := MapIDValide(in.MapID)
	if !okMap || !motifMatchSequence.MatchString(in.Body.MatchID) || !domain.XUIDValide(in.Body.VictimXUID) {
		// UN REFUS DE FORME EST UN 404, jamais un 400 : meme statut, meme code, meme message que
		// l'absence (cf. MapIDValide) — la frontiere n'est pas un oracle.
		return nil, humacore.NewError(http.StatusNotFound, "replay_not_available", messageSansSequence)
	}
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	seq, err := svc.Sequence(ctx, domain.TacticalSequenceRequest{
		MapID: mapID, MatchID: in.Body.MatchID, VictimXUID: in.Body.VictimXUID, TimeMs: in.Body.TimeMs,
	})
	if errors.Is(err, port.ErrReplayNotAvailable) {
		return nil, humacore.NewError(http.StatusNotFound, "replay_not_available", messageSansSequence)
	}
	if err != nil {
		return nil, mapTacticalError(ctx, err, "tactical.sequence")
	}
	return &tacticalSequenceOutput{Body: seq}, nil
}
