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

// squadChallengeCreatedOutput : 201 — objet SquadChallenge créé (statut posé par
// humacore.DefaultStatus au montage : source unique, runtime ET document).
type squadChallengeCreatedOutput struct {
	Body prestige.SquadChallenge
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
	if err := h.authorizeActor(ctx, body.CreatedBy); err != nil {
		return nil, err
	}
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
	return &squadChallengeCreatedOutput{Body: sc}, nil
}

// listSquadChallengesInput : path {squad_id} + le {player_slug} parent (l'acteur
// courant, déjà réconcilié avec la session par ownershipMW). player_slug sert de
// requestedBy pour la garde d'appartenance objet-level (les défis d'escouade sont
// en DB partagée, non isolés par player DB — cf. service.ListSquadChallenges).
type listSquadChallengesInput struct {
	SquadID    string `path:"squad_id"`
	PlayerSlug string `path:"player_slug"`
}

// ListSquadChallenges gère GET /squads/{squad_id}/challenges. La garde
// d'appartenance (requestedBy = player_slug du chemin, doit être membre-user)
// vit dans le service — un non-membre reçoit une erreur (BOLA objet-level clos).
func (h *PrestigeHandler) ListSquadChallenges(ctx context.Context, in *listSquadChallengesInput) (*mapOutput, error) {
	if in.SquadID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_squad_id", "squad_id requis")
	}
	list, err := h.svc.ListSquadChallenges(ctx, in.SquadID, in.PlayerSlug)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &mapOutput{Body: map[string]any{"squad_challenges": list, jsonKeyCount: len(list)}}, nil
}

type abandonSquadChallengeInput struct {
	ID          string `path:"id"`
	RequestedBy string `query:"requested_by"`
}

// AbandonSquadChallenge gère DELETE /squad-challenges/{id}?requested_by=slug —
// archive le défi (abandon volontaire) : il sort de la liste active. La garde
// d'appartenance (requestedBy membre-user) vit dans le service (BOLA objet-level).
func (h *PrestigeHandler) AbandonSquadChallenge(ctx context.Context, in *abandonSquadChallengeInput) (*noContentOutput, error) {
	if in.ID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_id", "id requis")
	}
	if in.RequestedBy == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_requested_by", "requested_by requis")
	}
	if err := h.authorizeActor(ctx, in.RequestedBy); err != nil {
		return nil, err
	}
	if err := h.svc.AbandonSquadChallenge(ctx, in.ID, in.RequestedBy); err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &noContentOutput{Status: http.StatusNoContent}, nil
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
	if err := h.authorizeActor(ctx, body.UserID); err != nil {
		return nil, err
	}
	if err := h.svc.JoinSquadChallenge(ctx, in.ID, body.UserID, prestige.Tier(body.ChosenTier), body.IsPrivate); err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &noContentOutput{Status: http.StatusNoContent}, nil
}
