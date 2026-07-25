// Package handlers — campaign.go : endpoints HTTP ImprovementCampaign
// (PLAN_PLAYER_PROFILE_ASCENSION §5.1).
//
// Routes (sous /api/v1/players/{player_slug}/) :
//
//	POST   /campaigns                  → StartCampaign
//	GET    /campaigns/active           → GetActiveCampaign (1 par titre)
//	GET    /campaigns/history          → ListEnded (campagnes closes, récentes d'abord)
//	GET    /campaigns/{id}             → GetByID (campagne + défis liés)
//	POST   /campaigns/{id}/pause       → PauseCampaign
//	POST   /campaigns/{id}/resume      → ResumeCampaign
//	POST   /campaigns/{id}/close       → CloseCampaign
//	POST   /campaigns/{id}/abandon     → AbandonCampaign
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (hérite ownership/title + lit {player_slug} parent) et enregistre via huma.*.
// Logique métier inchangée (campaign.Service), seul le wrapping HTTP change.
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
	"levelup/go-api/internal/campaign"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

// CampaignHandler regroupe les endpoints campagne.
type CampaignHandler struct {
	resolve   ProgressionResolver
	titleSlug string
}

// NewCampaignHandler construit le handler.
func NewCampaignHandler(resolve ProgressionResolver, titleSlug string) *CampaignHandler {
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	return &CampaignHandler{resolve: resolve, titleSlug: titleSlug}
}

// Mount enregistre les routes via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *CampaignHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/campaigns", h.handleStart)
	huma.Get(api, "/campaigns/active", h.handleGetActive)
	huma.Get(api, "/campaigns/history", h.handleListEnded)
	huma.Get(api, "/campaigns/{id}", h.handleGetByID)
	huma.Post(api, "/campaigns/{id}/pause", h.handlePause)
	huma.Post(api, "/campaigns/{id}/resume", h.handleResume)
	huma.Post(api, "/campaigns/{id}/close", h.handleClose)
	huma.Post(api, "/campaigns/{id}/abandon", h.handleAbandon)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// campaignPlayerInput : path param parent {player_slug} (active).
type campaignPlayerInput struct {
	PlayerSlug string `path:"player_slug"`
}

// campaignIDInput : {player_slug} + {id} (id passé tel quel au service).
type campaignIDInput struct {
	PlayerSlug string `path:"player_slug"`
	ID         string `path:"id"`
}

// startCampaignInput : {player_slug} + body de création.
type startCampaignInput struct {
	PlayerSlug string `path:"player_slug"`
	Body       startCampaignRequest
}

type startCampaignRequest struct {
	Axis          string `json:"axis"`
	AxisKind      string `json:"axis_kind"`
	PlaylistGroup string `json:"playlist_group,omitempty"`
}

// campaignCreatedOutput : 201 Created avec la campagne créée.
type campaignCreatedOutput struct {
	Status int
	Body   campaign.ImprovementCampaign
}

// campaignOutput : 200 avec la campagne (GetByID).
type campaignOutput struct {
	Body campaign.ImprovementCampaign
}

// campaignActiveOutput : 200 avec la campagne active OU null (ErrNotFound). Le
// pointeur reproduit le contrat d'origine (writeJSON nil → corps `null`).
type campaignActiveOutput struct {
	Body *campaign.ImprovementCampaign
}

// campaignHistoryItem : DTO dédié d'une campagne close pour la surface
// « Historique » (onglet Réalisations). N'expose que ce dont le front a besoin :
// axe, delta de progression (final − snapshot), playlist, dates, statut.
type campaignHistoryItem struct {
	ID            string     `json:"id"`
	Axis          string     `json:"axis"`
	AxisKind      string     `json:"axis_kind"`
	PlaylistGroup string     `json:"playlist_group"`
	Status        string     `json:"status"`
	StartedAt     time.Time  `json:"started_at"`
	EndedAt       *time.Time `json:"ended_at,omitempty"`
	SnapshotValue float64    `json:"snapshot_value"`
	FinalValue    *float64   `json:"final_value,omitempty"`
	Delta         *float64   `json:"delta,omitempty"`
}

// campaignHistoryResponse : corps de réponse de GET /campaigns/history.
type campaignHistoryResponse struct {
	Campaigns []campaignHistoryItem `json:"campaigns"`
	Count     int                   `json:"count"`
}

// campaignHistoryOutput : 200 avec la liste des campagnes closes + total.
type campaignHistoryOutput struct {
	Body campaignHistoryResponse
}

// campaignNoContent : réponse 204 sans corps (transitions pause/resume/close/abandon).
type campaignNoContent struct {
	Status int
}

// ─── Endpoints ─────────────────────────────────────────────────────────────

// handleStart : POST /campaigns → crée une nouvelle campagne sur axe ciblé (201).
// NB contrat (Phase 3b) : un corps JSON malformé/invalide est désormais rejeté par
// la validation Huma en 422 validation_error (au lieu de l'ancien 400 invalid_body) —
// seul écart délibéré de la migration, sans incidence pour un client bien formé.
func (h *CampaignHandler) handleStart(ctx context.Context, in *startCampaignInput) (*campaignCreatedOutput, error) {
	pdb, err := h.resolvePlayer(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	svc := h.serviceFromPDB(pdb)
	c, err := svc.StartCampaign(ctx, campaign.StartParams{
		UserID:        pdb.XUID,
		TitleSlug:     requestTitleSlug(ctx, h.titleSlug),
		Axis:          in.Body.Axis,
		AxisKind:      campaign.AxisKind(in.Body.AxisKind),
		PlaylistGroup: in.Body.PlaylistGroup,
	}, time.Now().UTC())
	if err != nil {
		switch {
		case errors.Is(err, campaign.ErrAlreadyActive):
			return nil, humacore.NewError(http.StatusConflict, "campaign_already_active", err.Error())
		case errors.Is(err, campaign.ErrInvalidAxis):
			return nil, humacore.NewError(http.StatusBadRequest, "invalid_axis", err.Error())
		default:
			slog.WarnContext(ctx, "campaign: start", "err", err)
			return nil, humacore.NewError(http.StatusInternalServerError, "start_campaign_error", err.Error())
		}
	}
	return &campaignCreatedOutput{Status: http.StatusCreated, Body: c}, nil
}

// handleGetActive : GET /campaigns/active → campagne active du joueur (ou null).
func (h *CampaignHandler) handleGetActive(ctx context.Context, in *campaignPlayerInput) (*campaignActiveOutput, error) {
	pdb, err := h.resolvePlayer(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	svc := h.serviceFromPDB(pdb)
	c, err := svc.GetActive(ctx, pdb.XUID, requestTitleSlug(ctx, h.titleSlug))
	if err != nil {
		if errors.Is(err, campaign.ErrNotFound) {
			return &campaignActiveOutput{Body: nil}, nil
		}
		slog.WarnContext(ctx, "campaign: get active", "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "get_active_error", err.Error())
	}
	return &campaignActiveOutput{Body: &c}, nil
}

// handleListEnded : GET /campaigns/history → campagnes closes du joueur, les
// plus récentes d'abord. Mappe vers un DTO dédié (delta snapshot→final calculé).
func (h *CampaignHandler) handleListEnded(ctx context.Context, in *campaignPlayerInput) (*campaignHistoryOutput, error) {
	pdb, err := h.resolvePlayer(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	svc := h.serviceFromPDB(pdb)
	list, err := svc.ListEnded(ctx, pdb.XUID, requestTitleSlug(ctx, h.titleSlug))
	if err != nil {
		slog.WarnContext(ctx, "campaign: list ended", "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "list_campaigns_error", err.Error())
	}
	items := make([]campaignHistoryItem, 0, len(list))
	for i := range list {
		items = append(items, toCampaignHistoryItem(&list[i]))
	}
	return &campaignHistoryOutput{Body: campaignHistoryResponse{Campaigns: items, Count: len(items)}}, nil
}

// toCampaignHistoryItem projette une campagne close vers son DTO d'historique.
func toCampaignHistoryItem(c *campaign.ImprovementCampaign) campaignHistoryItem {
	return campaignHistoryItem{
		ID:            c.ID,
		Axis:          c.Axis,
		AxisKind:      string(c.AxisKind),
		PlaylistGroup: c.PlaylistGroup,
		Status:        string(c.Status),
		StartedAt:     c.StartedAt,
		EndedAt:       c.EndedAt,
		SnapshotValue: c.SnapshotValue,
		FinalValue:    c.FinalValue(),
		Delta:         c.Delta(),
	}
}

// handleGetByID : GET /campaigns/{id} → détail + défis liés.
func (h *CampaignHandler) handleGetByID(ctx context.Context, in *campaignIDInput) (*campaignOutput, error) {
	pdb, err := h.resolvePlayer(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	svc := h.serviceFromPDB(pdb)
	c, err := svc.GetByID(ctx, in.ID)
	if err != nil {
		if errors.Is(err, campaign.ErrNotFound) {
			return nil, humacore.NewError(http.StatusNotFound, "campaign_not_found", err.Error())
		}
		slog.WarnContext(ctx, "campaign: get by id", "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "get_campaign_error", err.Error())
	}
	return &campaignOutput{Body: c}, nil
}

// handlePause : POST /campaigns/{id}/pause → 204.
func (h *CampaignHandler) handlePause(ctx context.Context, in *campaignIDInput) (*campaignNoContent, error) {
	return h.runTransition(ctx, in, func(svc *campaign.Service, id string) error {
		return svc.PauseCampaign(ctx, id)
	})
}

// handleResume : POST /campaigns/{id}/resume → 204.
func (h *CampaignHandler) handleResume(ctx context.Context, in *campaignIDInput) (*campaignNoContent, error) {
	return h.runTransition(ctx, in, func(svc *campaign.Service, id string) error {
		return svc.ResumeCampaign(ctx, id)
	})
}

// handleClose : POST /campaigns/{id}/close → 204.
func (h *CampaignHandler) handleClose(ctx context.Context, in *campaignIDInput) (*campaignNoContent, error) {
	now := time.Now().UTC()
	return h.runTransition(ctx, in, func(svc *campaign.Service, id string) error {
		return svc.CloseCampaign(ctx, id, now)
	})
}

// handleAbandon : POST /campaigns/{id}/abandon → 204.
func (h *CampaignHandler) handleAbandon(ctx context.Context, in *campaignIDInput) (*campaignNoContent, error) {
	now := time.Now().UTC()
	return h.runTransition(ctx, in, func(svc *campaign.Service, id string) error {
		return svc.AbandonCampaign(ctx, id, now)
	})
}

// ─── Helpers ───────────────────────────────────────────────────────────────

// resolvePlayer résout le slug courant ou renvoie une erreur Huma 404
// (contrat préservé : {code:player_not_found}).
func (h *CampaignHandler) resolvePlayer(ctx context.Context, slug string) (*duckdb.PlayerDB, error) {
	pdb, err := h.resolve(ctx, slug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	return pdb, nil
}

func (h *CampaignHandler) serviceFromPDB(pdb *duckdb.PlayerDB) *campaign.Service {
	repo := duckdb.NewCampaignRepo(pdb.Player)
	samples := duckdb.NewCampaignSampleProvider(pdb)
	return campaign.NewService(repo, samples)
}

// runTransition factorise les 4 transitions d'état (pause/resume/close/abandon),
// chacune renvoyant 204 No Content en succès. Contrat d'erreur identique à
// l'ancien handler chi (404 campaign_not_found, 409 invalid_status_transition).
func (h *CampaignHandler) runTransition(ctx context.Context, in *campaignIDInput, fn func(*campaign.Service, string) error) (*campaignNoContent, error) {
	pdb, err := h.resolvePlayer(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	svc := h.serviceFromPDB(pdb)
	if err := fn(svc, in.ID); err != nil {
		switch {
		case errors.Is(err, campaign.ErrNotFound):
			return nil, humacore.NewError(http.StatusNotFound, "campaign_not_found", err.Error())
		case errors.Is(err, campaign.ErrInvalidStatus):
			return nil, humacore.NewError(http.StatusConflict, "invalid_status_transition", err.Error())
		default:
			slog.WarnContext(ctx, "campaign: transition", "err", err)
			return nil, humacore.NewError(http.StatusInternalServerError, "campaign_transition_error", err.Error())
		}
	}
	return &campaignNoContent{Status: http.StatusNoContent}, nil
}
