// Package handlers — tactical_sequence.go : la SEQUENCE D'UNE MORT de l'onglet Tactique (vue D).
//
//	POST /players/{player_slug}/tactical/{map_id}/sequence   (corps : match_id, victim_xuid, time_ms)
//
// UN HANDLER A PART, MONTE DANS LE GROUPE DU REJEU 2D. La sequence lit les TRAJECTOIRES decodees du
// film, comme `/replay` : elle doit suivre EXACTEMENT les memes gardes — `RequireCapability(CapReplay)`
// (un titre sans rejeu rend 503) et `LocalOnlyReplay` (le rejeu n'est servi qu'en local jusqu'au
// reexamen de la decision du 2026-07-28, cf. replay_local_gate.go). Activer le rejeu en production
// active donc la vue D sans autre changement. En demo, `ReplayGate` lit le match dans le CORPS
// (`match_id`) : la route n'a pas de `{match_id}` dans son chemin. Les autres routes de l'onglet (`TacticalHandler`) restent
// hors de ce groupe : elles ne lisent que la base et les sidecars.
//
// C'EST UNE LECTURE EN POST (le corps porte la cle de la mort), declaree comme telle par le prefixe
// `/tactical/` de `middleware.readOnlyPostPrefixes`.
//
// ZERO LOGIQUE ICI. Le handler valide la forme des identifiants, delegue, traduit un refus ou une
// cause en statut et en code.
//
// DEUX FAMILLES D'ABSENCE (le web doit pouvoir distinguer une fonction absente d'une donnee absente) :
//
//	REFUS    match inconnu ou d'un autre joueur, forme invalide, garde du rejeu : TOUS rendent le meme
//	         404 `replay_not_available` — indiscernables par construction (ADR 0029). Le web n'y lit
//	         rien : fonction absente, la vue de sequence n'est pas rendue (comme le lien de rejeu).
//	CAUSE    le match est celui de l'appelant, la sequence manque pour une raison de DONNEES : 404 a
//	         code propre (`sequenceCauses`), que le web lit pour ecrire lui-meme le texte.
package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
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

// sequenceCauses : les causes « donnees » d'une sequence absente et leur code machine STABLE. Le web
// (`features/tactical/sequenceCause.logic.ts`) les lit pour ecrire le texte de l'etat vide : ne pas
// renommer un code sans le miroir. Statut 404 pour toutes : la sequence demandee n'existe pas (encore
// ou plus), le code dit pourquoi.
var sequenceCauses = []struct {
	err  error
	code string
}{
	{port.ErrSequenceAnalyseEnCours, "sequence_analysis_pending"},
	{port.ErrSequenceFilmNonAnalyse, "sequence_film_not_analysed"},
	{port.ErrSequenceFilmSansHorloge, "sequence_film_clock_missing"},
	{port.ErrSequenceCarteNonMesuree, "sequence_map_unmeasured"},
	{port.ErrSequenceSansPosition, "sequence_no_death_position"},
}

// erreurDeCause rend l'erreur HTTP d'une cause « donnees », ou nil si err n'en est pas une.
func erreurDeCause(ctx context.Context, err error) error {
	for _, c := range sequenceCauses {
		if errors.Is(err, c.err) {
			return humacore.NewErrorFrom(ctx, http.StatusNotFound, c.code, err)
		}
	}
	return nil
}

// handleGetSequence rend la sequence d'une mort du joueur de la page.
//
// Formes en LISTE BLANCHE, comme `MapIDValide` : le match_id par title.IsValidMatchID, le
// xuid par domain.XUIDValide. Rien d'autre n'atteint le service ni, derriere lui, le chemin
// d'un artefact.
func (h *TacticalSequenceHandler) handleGetSequence(ctx context.Context, in *tacticalSequenceInput) (*tacticalSequenceOutput, error) {
	mapID, okMap := MapIDValide(in.MapID)
	if !okMap || !title.IsValidMatchID(in.Body.MatchID) || !domain.XUIDValide(in.Body.VictimXUID) {
		// UN REFUS DE FORME EST UN 404, jamais un 400 : meme statut, meme code, meme message que
		// l'absence (cf. MapIDValide) — la frontiere n'est pas un oracle.
		return nil, humacore.NewError(http.StatusNotFound, "replay_not_available", messageSansSequence)
	}
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewErrorFrom(ctx, http.StatusNotFound, "player_not_found", err)
	}
	seq, err := svc.Sequence(ctx, domain.TacticalSequenceRequest{
		MapID: mapID, MatchID: in.Body.MatchID, VictimXUID: in.Body.VictimXUID, TimeMs: in.Body.TimeMs,
	})
	if errors.Is(err, port.ErrReplayNotAvailable) {
		return nil, humacore.NewError(http.StatusNotFound, "replay_not_available", messageSansSequence)
	}
	if httpErr := erreurDeCause(ctx, err); httpErr != nil {
		return nil, httpErr
	}
	if err != nil {
		return nil, mapTacticalError(ctx, err, "tactical.sequence")
	}
	return &tacticalSequenceOutput{Body: seq}, nil
}
