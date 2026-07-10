// Package handlers — prestige_squad_challenges.go : handlers Huma des défis d escouade.
// Extrait de prestige.go (K3f god-file split, 2026-07-06), même package.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/prestige"
)

// ─────────── Squad challenges ───────────

type createSquadChallengeBody struct {
	SquadID         string  `json:"squad_id"`
	TemplateID      string  `json:"template_id,omitempty"`
	TitleSlug       string  `json:"title_slug"`
	Mode            string  `json:"mode"`
	EvalType        string  `json:"eval_type"`
	WindowType      string  `json:"window_type"`
	WindowValue     string  `json:"window_value,omitempty"`
	TargetPerMember float64 `json:"target_per_member,omitempty"`
	CreatedBy       string  `json:"created_by"`
}

// squadChallengeCreatedOutput : 201 — objet SquadChallenge créé.
type squadChallengeCreatedOutput struct {
	Status int
	Body   prestige.SquadChallenge
}

// CreateSquadChallenge gère POST /squads/{squad_id}/challenges.
func (h *PrestigeHandler) CreateSquadChallenge(ctx context.Context, in *squadIDBodyInput) (*squadChallengeCreatedOutput, error) {
	if in.SquadID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_squad_id", "squad_id requis")
	}
	var body createSquadChallengeBody
	if err := json.Unmarshal(in.RawBody, &body); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	body.SquadID = in.SquadID
	sc, err := h.svc.CreateSquadChallenge(ctx, prestige.CreateSquadChallengeRequest{
		SquadID:         body.SquadID,
		TemplateID:      body.TemplateID,
		TitleSlug:       body.TitleSlug,
		Mode:            prestige.SquadMode(body.Mode),
		EvalType:        prestige.EvalType(body.EvalType),
		WindowType:      prestige.WindowType(body.WindowType),
		WindowValue:     body.WindowValue,
		TargetPerMember: body.TargetPerMember,
		CreatedBy:       body.CreatedBy,
	})
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &squadChallengeCreatedOutput{Status: http.StatusCreated, Body: sc}, nil
}

// ListSquadChallenges gère GET /squads/{squad_id}/challenges.
func (h *PrestigeHandler) ListSquadChallenges(ctx context.Context, in *squadIDInput) (*mapOutput, error) {
	if in.SquadID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_squad_id", "squad_id requis")
	}
	list, err := h.svc.ListSquadChallenges(ctx, in.SquadID)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &mapOutput{Body: map[string]any{"squad_challenges": list, jsonKeyCount: len(list)}}, nil
}

type joinSquadChallengeBody struct {
	UserID     string `json:"user_id"`
	ChosenTier string `json:"chosen_tier,omitempty"`
	IsPrivate  bool   `json:"is_private,omitempty"`
}

// JoinSquadChallenge gère POST /squad-challenges/{id}/join.
func (h *PrestigeHandler) JoinSquadChallenge(ctx context.Context, in *idBodyInput) (*noContentOutput, error) {
	if in.ID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_id", "id requis")
	}
	var body joinSquadChallengeBody
	if err := json.Unmarshal(in.RawBody, &body); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	if err := h.svc.JoinSquadChallenge(ctx, in.ID, body.UserID, prestige.Tier(body.ChosenTier), body.IsPrivate); err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &noContentOutput{Status: http.StatusNoContent}, nil
}
