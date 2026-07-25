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
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (hérite ownership/title + lit {player_slug} parent) et enregistre via huma.*.
// Logique métier inchangée (coach_advisor.Service), seul le wrapping HTTP change.
//
// Réf : ADR 0020 §"Architecture proposée".
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain/title"
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
		titleSlug = title.DefaultSlug
	}
	return &CoachProposalsHandler{resolve: resolve, titleSlug: titleSlug}
}

// Mount enregistre les 3 routes via Huma (Phase 3b) sur un sous-routeur déjà
// préfixé par /players/{player_slug} (humacore.NewAPI lit le path param parent
// et hérite du middleware ownership/title du sous-groupe).
func (h *CoachProposalsHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/coach/proposals", h.handleListProposals)
	huma.Post(api, "/coach/proposals/{id}/accept", h.handleAcceptProposal)
	huma.Post(api, "/coach/proposals/{id}/dismiss", h.handleDismissProposal)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// coachListInput : {player_slug} + ?status= (filtre par status, optionnel).
type coachListInput struct {
	PlayerSlug string `path:"player_slug"`
	Status     string `query:"status"`
}

// coachActionInput : {player_slug} + {id} (accept/dismiss). L'id reste en STRING
// (identifiant opaque côté service) ; le guard missing_id du contrat d'origine est
// reproduit dans le handler.
type coachActionInput struct {
	PlayerSlug string `path:"player_slug"`
	ID         string `path:"id"`
}

type proposalsListOutput struct{ Body proposalsListResponse }
type acceptOutput struct{ Body acceptResponse }
type dismissOutput struct{ Body dismissResponse }

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

// handleListProposals : GET /proposals?status=pending
//
// Query param `status` filtre par status (pending|accepted|dismissed|
// superseded|obsoleted|stale). Si vide, retourne toutes les proposals.
func (h *CoachProposalsHandler) handleListProposals(ctx context.Context, in *coachListInput) (*proposalsListOutput, error) {
	svc, userID, err := h.resolveOr404(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}

	status := coach_advisor.ProposalStatus(in.Status)
	props, err := svc.ListProposals(ctx, userID, requestTitleSlug(ctx, h.titleSlug), status)
	if err != nil {
		slog.WarnContext(ctx, "coach_proposals: list", "err", err, "player", in.PlayerSlug)
		return nil, humacore.NewError(http.StatusInternalServerError, "list_proposals_error", err.Error())
	}
	items := make([]proposalDTO, 0, len(props))
	for _, p := range props {
		items = append(items, toProposalDTO(p))
	}
	return &proposalsListOutput{Body: proposalsListResponse{Items: items}}, nil
}

// handleAcceptProposal : POST /proposals/{id}/accept
//
// Matérialise la proposal via prestige.CreateChallenge ou CreateArc.
// 409 Conflict si la proposal n'est plus pending.
func (h *CoachProposalsHandler) handleAcceptProposal(ctx context.Context, in *coachActionInput) (*acceptOutput, error) {
	svc, _, err := h.resolveOr404(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	if in.ID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_id", "proposal id required")
	}

	result, err := svc.AcceptProposal(ctx, in.ID)
	switch {
	case errors.Is(err, coach_advisor.ErrProposalNotFound):
		return nil, humacore.NewError(http.StatusNotFound, "proposal_not_found", err.Error())
	case errors.Is(err, coach_advisor.ErrProposalNotAcceptable):
		return nil, humacore.NewError(http.StatusConflict, "proposal_not_acceptable", err.Error())
	case err != nil:
		slog.WarnContext(ctx, "coach_proposals: accept", "err", err, "id", in.ID)
		return nil, humacore.NewError(http.StatusInternalServerError, "accept_error", err.Error())
	}
	return &acceptOutput{Body: acceptResponse{
		Status:       "accepted",
		ChallengeID:  result.ChallengeID,
		ArcID:        result.ArcID,
		ChallengeIDs: result.ChallengeIDs,
	}}, nil
}

// handleDismissProposal : POST /proposals/{id}/dismiss
//
// Idempotent — pas d'erreur si déjà dismissed.
func (h *CoachProposalsHandler) handleDismissProposal(ctx context.Context, in *coachActionInput) (*dismissOutput, error) {
	svc, _, err := h.resolveOr404(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	if in.ID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_id", "proposal id required")
	}
	if err := svc.DismissProposal(ctx, in.ID); err != nil {
		slog.WarnContext(ctx, "coach_proposals: dismiss", "err", err, "id", in.ID)
		return nil, humacore.NewError(http.StatusInternalServerError, "dismiss_error", err.Error())
	}
	return &dismissOutput{Body: dismissResponse{Status: "dismissed"}}, nil
}

// ─── Helpers ───

// resolveOr404 résout le service coach + userID pour le slug courant, ou renvoie
// l'erreur Huma au contrat d'origine (503 coach_advisor_disabled si resolve nil,
// 404 player_not_found si introuvable, 503 coach_advisor_unavailable si service nil).
func (h *CoachProposalsHandler) resolveOr404(ctx context.Context, playerSlug string) (coach_advisor.Service, string, error) {
	if h.resolve == nil {
		return nil, "", humacore.NewError(http.StatusServiceUnavailable, "coach_advisor_disabled",
			"coach_advisor not configured")
	}
	svc, userID, err := h.resolve(ctx, playerSlug)
	if err != nil {
		return nil, "", humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	if svc == nil {
		return nil, "", humacore.NewError(http.StatusServiceUnavailable, "coach_advisor_unavailable",
			"coach_advisor service could not be built for this player")
	}
	return svc, userID, nil
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
