// Package handlers — prestige_arcs.go : handlers Huma des Arcs de prestige (CRUD +
// presets). Extrait de prestige.go (K3f god-file split, 2026-07-06), même package.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/prestige"
)

// ─────────── Arcs ───────────

type createArcBody struct {
	UserID      string `json:"user_id"`
	TitleSlug   string `json:"title_slug"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// arcOutput : 200 — objet Arc brut.
type arcOutput struct{ Body prestige.Arc }

// arcCreatedOutput : 201 — objet Arc créé (statut posé par humacore.DefaultStatus
// au montage : source unique, runtime ET document).
type arcCreatedOutput struct {
	Body prestige.Arc
}

// CreateArc gère POST /arcs.
func (h *PrestigeHandler) CreateArc(ctx context.Context, in *rawBodyInput) (*arcCreatedOutput, error) {
	var body createArcBody
	if err := json.Unmarshal(in.RawBody, &body); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	if err := h.authorizeActor(ctx, body.UserID); err != nil {
		return nil, err
	}
	a, err := h.svc.CreateArc(ctx, prestige.CreateArcRequest{
		UserID:      body.UserID,
		TitleSlug:   body.TitleSlug,
		Title:       body.Title,
		Description: body.Description,
	})
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &arcCreatedOutput{Body: a}, nil
}

// GetArc gère GET /arcs/{id}.
func (h *PrestigeHandler) GetArc(ctx context.Context, in *idInput) (*arcOutput, error) {
	if in.ID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_id", "id requis")
	}
	a, err := h.svc.GetArc(ctx, in.ID)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &arcOutput{Body: a}, nil
}

type deleteArcInput struct {
	ID         string `path:"id"`
	UserID     string `query:"user_id"`
	Objectives string `query:"objectives"`
}

// DeleteArc gère DELETE /arcs/{id}?user_id=&objectives=delete|detach.
//
//   - objectives=delete : supprime aussi les objectifs (abandon, ou hard delete
//     si l'arc a moins d'1h → zéro cooldown).
//   - objectives=detach : détache les objectifs (gardés, redeviennent libres).
func (h *PrestigeHandler) DeleteArc(ctx context.Context, in *deleteArcInput) (*noContentOutput, error) {
	if in.ID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_id", "id requis")
	}
	if in.UserID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_params", "user_id requis")
	}
	if in.Objectives != "delete" && in.Objectives != "detach" {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_input",
			"objectives doit valoir 'delete' ou 'detach'")
	}
	if err := h.authorizeActor(ctx, in.UserID); err != nil {
		return nil, err
	}
	opts := prestige.DeleteArcOptions{CascadeObjectives: in.Objectives == "delete"}
	if err := h.svc.DeleteArc(ctx, in.UserID, in.ID, opts); err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &noContentOutput{Status: http.StatusNoContent}, nil
}

type listArcsInput struct {
	UserID    string `query:"user_id"`
	TitleSlug string `query:"title_slug"`
}

// ListArcs gère GET /arcs?user_id=&title_slug=.
func (h *PrestigeHandler) ListArcs(ctx context.Context, in *listArcsInput) (*mapOutput, error) {
	if in.UserID == "" || in.TitleSlug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
	}
	if err := h.authorizeActor(ctx, in.UserID); err != nil {
		return nil, err
	}
	arcs, err := h.svc.ListArcs(ctx, in.UserID, in.TitleSlug)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &mapOutput{Body: map[string]any{"arcs": arcs, jsonKeyCount: len(arcs)}}, nil
}

// ListArcPresets gère GET /arcs/presets?user_id=&title_slug=.
func (h *PrestigeHandler) ListArcPresets(ctx context.Context, in *listArcsInput) (*mapOutput, error) {
	if in.UserID == "" || in.TitleSlug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
	}
	if err := h.authorizeActor(ctx, in.UserID); err != nil {
		return nil, err
	}
	presets, err := h.svc.ListArcPresets(ctx, in.UserID, in.TitleSlug)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &mapOutput{Body: map[string]any{"presets": presets, jsonKeyCount: len(presets)}}, nil
}

type adoptPresetArcBody struct {
	UserID    string `json:"user_id"`
	TitleSlug string `json:"title_slug"`
}

// AdoptPresetArc gère POST /arcs/presets/{id}/adopt.
func (h *PrestigeHandler) AdoptPresetArc(ctx context.Context, in *idBodyInput) (*arcCreatedOutput, error) {
	if in.ID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_id", "id requis")
	}
	var body adoptPresetArcBody
	if err := json.Unmarshal(in.RawBody, &body); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
	}
	if body.UserID == "" || body.TitleSlug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
	}
	if err := h.authorizeActor(ctx, body.UserID); err != nil {
		return nil, err
	}
	arc, err := h.svc.AdoptPresetArc(ctx, body.UserID, body.TitleSlug, in.ID)
	if err != nil {
		return nil, h.serviceError(ctx, err)
	}
	return &arcCreatedOutput{Body: arc}, nil
}
