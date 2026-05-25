// Package handlers — coach_proposals.go : endpoints HTTP du coach_advisor.
//
// Routes (sous /api/v1/players/{player_slug}/coach/) :
//   - GET    /proposals?status=pending   : liste les proposals coach
//   - POST   /proposals/{id}/accept      : matérialise la proposal via Prestige
//   - POST   /proposals/{id}/dismiss     : marque la proposal comme dismissed
//
// Le pipeline de génération s'exécute côté post-sync hook (ADR 0020 Phase 8).
// Ces handlers sont purement lecture (GET) + action joueur (POST).
//
// Réf : ADR 0020 §"Architecture proposée".
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/progression/coach_advisor"
)

// CoachAdvisorResolver retourne le coach_advisor.Service + le userID (xuid)
// pour un joueur donné. Le userID est aligné sur la colonne coach_proposal.user_id
// (xuid du PlayerDB, cf. EvaluateProgressionAfterSync). Implémenté côté
// server.go (composé via les bundles + reg.resolve).
//
// Erreur si le joueur n'existe pas ou si le coach_advisor n'est pas
// configuré (bundles manquants).
type CoachAdvisorResolver func(ctx context.Context, playerSlug string) (svc coach_advisor.Service, userID string, err error)

// CoachProposalsHandler regroupe les endpoints HTTP du coach_advisor.
type CoachProposalsHandler struct {
	resolve   CoachAdvisorResolver
	titleSlug string
}

// NewCoachProposalsHandler construit le handler.
func NewCoachProposalsHandler(resolve CoachAdvisorResolver, titleSlug string) *CoachProposalsHandler {
	if titleSlug == "" {
		titleSlug = "halo_infinite"
	}
	return &CoachProposalsHandler{resolve: resolve, titleSlug: titleSlug}
}

// Mount enregistre les 3 routes sur un sous-routeur déjà préfixé par
// /players/{player_slug}.
func (h *CoachProposalsHandler) Mount(r chi.Router) {
	r.Route("/coach", func(r chi.Router) {
		r.Get("/proposals", h.ListProposals)
		r.Post("/proposals/{id}/accept", h.AcceptProposal)
		r.Post("/proposals/{id}/dismiss", h.DismissProposal)
	})
}

// ─── DTOs ───

type proposalDTO struct {
	ID            string     `json:"id"`
	Kind          string     `json:"kind"` // "challenge" | "arc"
	TemplateID    string     `json:"template_id,omitempty"`
	SuggestedTier string     `json:"suggested_tier,omitempty"`
	SourceSignal  string     `json:"source_signal"`
	SourceMetric  string     `json:"source_metric,omitempty"`
	RadarAxis     string     `json:"radar_axis,omitempty"`
	Strength      float64    `json:"strength"`
	Origin        string     `json:"origin"` // "catalog" | "synthesized"
	ReasonKeyEN   string     `json:"reason_key_en,omitempty"`
	ReasonKeyFR   string     `json:"reason_key_fr,omitempty"`
	ReasonParams  string     `json:"reason_params,omitempty"` // JSON raw
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	ResolvedAt    *time.Time `json:"resolved_at,omitempty"`
	ResolvedRef   string     `json:"resolved_ref,omitempty"`
}

type proposalsListResponse struct {
	Items []proposalDTO `json:"items"`
}

type acceptResponse struct {
	Status       string   `json:"status"` // "accepted"
	ChallengeID  string   `json:"challenge_id,omitempty"`
	ArcID        string   `json:"arc_id,omitempty"`
	ChallengeIDs []string `json:"challenge_ids,omitempty"`
}

type dismissResponse struct {
	Status string `json:"status"` // "dismissed"
}

// ─── Endpoints ───

// ListProposals : GET /proposals?status=pending
//
// Query param `status` filtre par status (pending|accepted|dismissed|
// superseded|obsoleted|stale). Si vide, retourne toutes les proposals.
func (h *CoachProposalsHandler) ListProposals(w http.ResponseWriter, r *http.Request) {
	playerSlug := chi.URLParam(r, "player_slug")
	svc, userID, ok := h.resolveOr404(w, r, playerSlug)
	if !ok {
		return
	}

	status := coach_advisor.ProposalStatus(r.URL.Query().Get("status"))
	props, err := svc.ListProposals(r.Context(), userID, h.titleSlug, status)
	if err != nil {
		slog.WarnContext(r.Context(), "coach_proposals: list", "err", err, "player", playerSlug)
		writeError(r.Context(), w, http.StatusInternalServerError, "list_proposals_error", err.Error())
		return
	}
	items := make([]proposalDTO, 0, len(props))
	for _, p := range props {
		items = append(items, toProposalDTO(p))
	}
	writeJSON(w, http.StatusOK, proposalsListResponse{Items: items})
}

// AcceptProposal : POST /proposals/{id}/accept
//
// Matérialise la proposal via prestige.CreateChallenge ou CreateArc.
// 409 Conflict si la proposal n'est plus pending.
func (h *CoachProposalsHandler) AcceptProposal(w http.ResponseWriter, r *http.Request) {
	playerSlug := chi.URLParam(r, "player_slug")
	svc, _, ok := h.resolveOr404(w, r, playerSlug)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_id", "proposal id required")
		return
	}

	result, err := svc.AcceptProposal(r.Context(), id)
	switch {
	case errors.Is(err, coach_advisor.ErrProposalNotFound):
		writeError(r.Context(), w, http.StatusNotFound, "proposal_not_found", err.Error())
		return
	case errors.Is(err, coach_advisor.ErrProposalNotAcceptable):
		writeError(r.Context(), w, http.StatusConflict, "proposal_not_acceptable", err.Error())
		return
	case err != nil:
		slog.WarnContext(r.Context(), "coach_proposals: accept", "err", err, "id", id)
		writeError(r.Context(), w, http.StatusInternalServerError, "accept_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, acceptResponse{
		Status:       "accepted",
		ChallengeID:  result.ChallengeID,
		ArcID:        result.ArcID,
		ChallengeIDs: result.ChallengeIDs,
	})
}

// DismissProposal : POST /proposals/{id}/dismiss
//
// Idempotent — pas d'erreur si déjà dismissed.
func (h *CoachProposalsHandler) DismissProposal(w http.ResponseWriter, r *http.Request) {
	playerSlug := chi.URLParam(r, "player_slug")
	svc, _, ok := h.resolveOr404(w, r, playerSlug)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_id", "proposal id required")
		return
	}
	if err := svc.DismissProposal(r.Context(), id); err != nil {
		slog.WarnContext(r.Context(), "coach_proposals: dismiss", "err", err, "id", id)
		writeError(r.Context(), w, http.StatusInternalServerError, "dismiss_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dismissResponse{Status: "dismissed"})
}

// ─── Helpers ───

func (h *CoachProposalsHandler) resolveOr404(w http.ResponseWriter, r *http.Request, playerSlug string) (coach_advisor.Service, string, bool) {
	if h.resolve == nil {
		writeError(r.Context(), w, http.StatusServiceUnavailable, "coach_advisor_disabled",
			"coach_advisor not configured")
		return nil, "", false
	}
	svc, userID, err := h.resolve(r.Context(), playerSlug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return nil, "", false
	}
	if svc == nil {
		writeError(r.Context(), w, http.StatusServiceUnavailable, "coach_advisor_unavailable",
			"coach_advisor service could not be built for this player")
		return nil, "", false
	}
	return svc, userID, true
}

func toProposalDTO(p coach_advisor.Proposal) proposalDTO {
	return proposalDTO{
		ID:            p.ID,
		Kind:          string(p.Kind),
		TemplateID:    p.TemplateID,
		SuggestedTier: string(p.SuggestedTier),
		SourceSignal:  string(p.SourceSignal),
		SourceMetric:  p.SourceMetric,
		RadarAxis:     p.RadarAxis,
		Strength:      p.Strength,
		Origin:        string(p.Origin),
		ReasonKeyEN:   p.ReasonKeyEN,
		ReasonKeyFR:   p.ReasonKeyFR,
		ReasonParams:  p.ReasonParams,
		Status:        string(p.Status),
		CreatedAt:     p.CreatedAt,
		ResolvedAt:    p.ResolvedAt,
		ResolvedRef:   p.ResolvedRef,
	}
}
