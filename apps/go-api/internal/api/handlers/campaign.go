// Package handlers — campaign.go : endpoints HTTP ImprovementCampaign
// (PLAN_PLAYER_PROFILE_ASCENSION §5.1).
//
// Routes (sous /api/v1/players/{player_slug}/) :
//
//	POST   /campaigns                  → StartCampaign
//	GET    /campaigns/active           → GetActiveCampaign (1 par titre)
//	GET    /campaigns/{id}             → GetByID (campagne + défis liés)
//	POST   /campaigns/{id}/pause       → PauseCampaign
//	POST   /campaigns/{id}/resume      → ResumeCampaign
//	POST   /campaigns/{id}/close       → CloseCampaign
//	POST   /campaigns/{id}/abandon     → AbandonCampaign
package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

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

// Mount enregistre les routes sur un router chi sous-monté.
func (h *CampaignHandler) Mount(r chi.Router) {
	r.Post("/campaigns", h.Start)
	r.Get("/campaigns/active", h.GetActive)
	r.Get("/campaigns/{id}", h.GetByID)
	r.Post("/campaigns/{id}/pause", h.Pause)
	r.Post("/campaigns/{id}/resume", h.Resume)
	r.Post("/campaigns/{id}/close", h.Close)
	r.Post("/campaigns/{id}/abandon", h.Abandon)
}

// ─── DTOs ──────────────────────────────────────────────────────────────────

type startCampaignRequest struct {
	Axis          string `json:"axis"`
	AxisKind      string `json:"axis_kind"`
	PlaylistGroup string `json:"playlist_group,omitempty"`
}

// ─── Endpoints ─────────────────────────────────────────────────────────────

// Start : POST /campaigns → crée une nouvelle campagne sur axe ciblé.
func (h *CampaignHandler) Start(w http.ResponseWriter, r *http.Request) {
	pdb, ok := h.resolveOr404Campaign(w, r)
	if !ok {
		return
	}
	var req startCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	svc := h.serviceFromPDB(pdb)
	c, err := svc.StartCampaign(r.Context(), campaign.StartParams{
		UserID:        pdb.XUID,
		TitleSlug:     h.titleSlug,
		Axis:          req.Axis,
		AxisKind:      campaign.AxisKind(req.AxisKind),
		PlaylistGroup: req.PlaylistGroup,
	}, time.Now().UTC())
	if err != nil {
		switch {
		case errors.Is(err, campaign.ErrAlreadyActive):
			writeError(r.Context(), w, http.StatusConflict, "campaign_already_active", err.Error())
		case errors.Is(err, campaign.ErrInvalidAxis):
			writeError(r.Context(), w, http.StatusBadRequest, "invalid_axis", err.Error())
		default:
			slog.WarnContext(r.Context(), "campaign: start", "err", err)
			writeError(r.Context(), w, http.StatusInternalServerError, "start_campaign_error", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

// GetActive : GET /campaigns/active → campagne active du joueur.
func (h *CampaignHandler) GetActive(w http.ResponseWriter, r *http.Request) {
	pdb, ok := h.resolveOr404Campaign(w, r)
	if !ok {
		return
	}
	svc := h.serviceFromPDB(pdb)
	c, err := svc.GetActive(r.Context(), pdb.XUID, h.titleSlug)
	if err != nil {
		if errors.Is(err, campaign.ErrNotFound) {
			writeJSON(w, http.StatusOK, nil)
			return
		}
		slog.WarnContext(r.Context(), "campaign: get active", "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "get_active_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// GetByID : GET /campaigns/{id} → détail + défis liés.
func (h *CampaignHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	pdb, ok := h.resolveOr404Campaign(w, r)
	if !ok {
		return
	}
	svc := h.serviceFromPDB(pdb)
	c, err := svc.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, campaign.ErrNotFound) {
			writeError(r.Context(), w, http.StatusNotFound, "campaign_not_found", err.Error())
			return
		}
		slog.WarnContext(r.Context(), "campaign: get by id", "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "get_campaign_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// Pause : POST /campaigns/{id}/pause.
func (h *CampaignHandler) Pause(w http.ResponseWriter, r *http.Request) {
	h.runTransition(w, r, func(svc *campaign.Service, id string) error {
		return svc.PauseCampaign(r.Context(), id)
	})
}

// Resume : POST /campaigns/{id}/resume.
func (h *CampaignHandler) Resume(w http.ResponseWriter, r *http.Request) {
	h.runTransition(w, r, func(svc *campaign.Service, id string) error {
		return svc.ResumeCampaign(r.Context(), id)
	})
}

// Close : POST /campaigns/{id}/close.
func (h *CampaignHandler) Close(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	h.runTransition(w, r, func(svc *campaign.Service, id string) error {
		return svc.CloseCampaign(r.Context(), id, now)
	})
}

// Abandon : POST /campaigns/{id}/abandon.
func (h *CampaignHandler) Abandon(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	h.runTransition(w, r, func(svc *campaign.Service, id string) error {
		return svc.AbandonCampaign(r.Context(), id, now)
	})
}

// ─── Helpers ───────────────────────────────────────────────────────────────

func (h *CampaignHandler) resolveOr404Campaign(w http.ResponseWriter, r *http.Request) (*duckdb.PlayerDB, bool) {
	slug := chi.URLParam(r, "player_slug")
	pdb, err := h.resolve(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return nil, false
	}
	return pdb, true
}

func (h *CampaignHandler) serviceFromPDB(pdb *duckdb.PlayerDB) *campaign.Service {
	repo := duckdb.NewCampaignRepo(pdb.Player)
	samples := duckdb.NewCampaignSampleProvider(pdb)
	return campaign.NewService(repo, samples)
}

func (h *CampaignHandler) runTransition(w http.ResponseWriter, r *http.Request, fn func(*campaign.Service, string) error) {
	pdb, ok := h.resolveOr404Campaign(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	svc := h.serviceFromPDB(pdb)
	if err := fn(svc, id); err != nil {
		switch {
		case errors.Is(err, campaign.ErrNotFound):
			writeError(r.Context(), w, http.StatusNotFound, "campaign_not_found", err.Error())
		case errors.Is(err, campaign.ErrInvalidStatus):
			writeError(r.Context(), w, http.StatusConflict, "invalid_status_transition", err.Error())
		default:
			slog.WarnContext(r.Context(), "campaign: transition", "err", err)
			writeError(r.Context(), w, http.StatusInternalServerError, "campaign_transition_error", err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
