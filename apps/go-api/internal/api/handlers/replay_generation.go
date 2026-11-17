// Package handlers — replay_generation.go : la génération à la demande du rejeu d'un match qui n'en a pas
// (application de bureau et poste de développement).
//
//	POST /players/{player_slug}/matches/{match_id}/replay/generate    202 (200 si le rejeu est déjà rangé)
//	GET  /players/{player_slug}/matches/{match_id}/replay/generation  200
//
// Monté sous le groupe du rejeu (titre actif, auth des écritures, propriété du joueur, capacité de rejeu du titre,
// participation au match) et seulement sous la capacité GenerateReplays. Le handler décode, appelle le port et traduit ;
// les contrôles et la file vivent dans le service et dans `replayregen`. Le texte d'un refus ne part jamais au client :
// il reçoit un code (le web écrit les textes) et, pour un refus de prise en charge, sa raison dans `details.reason`.
package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// Codes des refus de la génération.
const (
	codeReplayFilmUnavailable         = "replay_film_unavailable"
	codeReplayGenerationNotSupported  = "replay_generation_not_supported"
	codeReplayGenerationQueueFull     = "replay_generation_queue_full"
	codeReplayGenerationUnavailable   = "replay_generation_unavailable"
	codeReplayGenerationError         = "replay_generation_error"
	probeReplayGeneration             = "replay.generation"
	detailKeyReason                   = "reason"
	errReplayGenerationInvalidMatchID = replayGenerationError("invalid match_id form")
)

// replayGenerationError : une erreur constante du handler.
type replayGenerationError string

func (e replayGenerationError) Error() string { return string(e) }

// ReplayGenerationHandler sert la demande et l'état de la génération du rejeu d'un match.
type ReplayGenerationHandler struct {
	newSvc ServiceFactory[port.ReplayGeneration]
}

// NewReplayGenerationHandler construit le handler avec sa factory de service (liée au joueur du chemin).
func NewReplayGenerationHandler(newSvc ServiceFactory[port.ReplayGeneration]) *ReplayGenerationHandler {
	return &ReplayGenerationHandler{newSvc: newSvc}
}

// Mount enregistre les deux routes sur le sous-routeur chi (préfixe /players/{player_slug}).
func (h *ReplayGenerationHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/matches/{match_id}/replay/generate", h.handleGenerate,
		humacore.Op("postPlayerMatchReplayGenerate", "Generate the replay of a match that has none (desktop app and development)", "replay"),
		humacore.DefaultStatus(http.StatusAccepted))
	huma.Get(api, "/matches/{match_id}/replay/generation", h.handleStatus,
		humacore.Op("getPlayerMatchReplayGeneration", "State of the generation of the replay of a match", "replay"))
}

type replayGenerationInput struct {
	PlayerSlug string `path:"player_slug"`
	MatchID    string `path:"match_id"`
}

// replayGenerationOutput : statut DYNAMIQUE (202 en file ou en cours, 200 rejeu rangé, 409 et 429 en refus avec leur
// enveloppe d'erreur : le champ Status l'emporte au runtime ; le statut nominal du document vient de DefaultStatus).
type replayGenerationOutput struct {
	Status int
	Body   any
}

type replayGenerationStatusOutput struct {
	Body *domain.ReplayGenerationStatus
}

// handleGenerate met la génération en file (ou rend l'état de la demande en vol).
func (h *ReplayGenerationHandler) handleGenerate(ctx context.Context, in *replayGenerationInput) (*replayGenerationOutput, error) {
	svc, err := h.service(ctx, in.PlayerSlug, in.MatchID)
	if err != nil {
		return nil, err
	}
	st, created, err := svc.RequestGeneration(ctx, ctxkeys.TitleSlug(ctx), in.MatchID)
	if err != nil {
		return refuseGeneration(ctx, err)
	}
	status := http.StatusAccepted
	if st.State == domain.ReplayGenerationReady {
		status = http.StatusOK
	}
	return &replayGenerationOutput{Status: status, Body: domain.ReplayGenerationRequestResponse{Created: created, Status: st}}, nil
}

// handleStatus rend l'état de la génération du match, sans rien lancer.
func (h *ReplayGenerationHandler) handleStatus(ctx context.Context, in *replayGenerationInput) (*replayGenerationStatusOutput, error) {
	svc, err := h.service(ctx, in.PlayerSlug, in.MatchID)
	if err != nil {
		return nil, err
	}
	st, err := svc.GenerationStatus(ctx, ctxkeys.TitleSlug(ctx), in.MatchID)
	if err != nil {
		if mapped, ok := MapCapabilityError(ctx, err, probeReplayGeneration); ok {
			return nil, mapped
		}
		return nil, humacore.NewErrorFrom(ctx, http.StatusInternalServerError, codeReplayGenerationError, err)
	}
	return &replayGenerationStatusOutput{Body: &st}, nil
}

// service valide l'identifiant du match et résout le service lié au joueur du chemin.
func (h *ReplayGenerationHandler) service(ctx context.Context, playerSlug, matchID string) (port.ReplayGeneration, error) {
	if !title.IsValidMatchID(matchID) {
		return nil, humacore.NewErrorFrom(ctx, http.StatusBadRequest, codeInvalidMatchID, errReplayGenerationInvalidMatchID)
	}
	svc, err := h.newSvc(ctx, playerSlug)
	if err != nil {
		return nil, humacore.NewErrorFrom(ctx, http.StatusNotFound, "player_not_found", err)
	}
	return svc, nil
}

// refuseGeneration traduit un refus du service : capacité absente (503), file pleine (429, réessayable), film
// indisponible et match non pris en charge (409, raison dans `details.reason`), file non reliée (503).
func refuseGeneration(ctx context.Context, err error) (*replayGenerationOutput, error) {
	if mapped, ok := MapCapabilityError(ctx, err, probeReplayGeneration); ok {
		return nil, mapped
	}
	var notSupported *domain.ReplayGenerationNotSupportedError
	switch {
	case errors.Is(err, domain.ErrReplayGenerationQueueFull):
		return refusal(ctx, http.StatusTooManyRequests, codeReplayGenerationQueueFull, "", true, err), nil
	case errors.Is(err, domain.ErrReplayFilmUnavailable):
		return refusal(ctx, http.StatusConflict, codeReplayFilmUnavailable, "", false, err), nil
	case errors.As(err, &notSupported):
		return refusal(ctx, http.StatusConflict, codeReplayGenerationNotSupported, notSupported.Reason, false, err), nil
	case errors.Is(err, domain.ErrReplayGenerationNotSupported):
		return refusal(ctx, http.StatusConflict, codeReplayGenerationNotSupported, "", false, err), nil
	case errors.Is(err, domain.ErrReplayGenerationUnbound):
		return nil, humacore.NewErrorFrom(ctx, http.StatusServiceUnavailable, codeReplayGenerationUnavailable, err)
	}
	return nil, humacore.NewErrorFrom(ctx, http.StatusInternalServerError, codeReplayGenerationError, err)
}

// refusal construit l'enveloppe d'erreur {code, message, retryable[, details]} d'un refus à statut dynamique ; le
// message est le message FIXE du code et la cause est journalisée.
func refusal(ctx context.Context, status int, code, reason string, retryable bool, cause error) *replayGenerationOutput {
	msg := humacore.FixedMessage(status, code)
	humacore.LogErrorResponse(ctx, status, code, msg, cause)
	body := map[string]any{"code": code, "message": msg, "retryable": retryable}
	if reason != "" {
		body["details"] = map[string]string{detailKeyReason: reason}
	}
	return &replayGenerationOutput{Status: status, Body: body}
}
