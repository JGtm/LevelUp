// Package handlers — handler HTTP du module Prestige (Phase 3).
//
// Couvre les endpoints REST :
//
//	POST   /api/v1/challenges           — créer un défi (avec quotas mode pilote)
//	GET    /api/v1/challenges/{id}      — détail d'un défi
//	GET    /api/v1/challenges           — liste filtrée
//	PATCH  /api/v1/challenges/{id}      — édition (cible recalcule palier en mode libre)
//	DELETE /api/v1/challenges/{id}      — abandon
//	POST   /api/v1/challenges/{id}/suggest-next — alternatives palier supérieur
//	GET    /api/v1/prestige/me          — total PP + niveau (par titre ou cross-titre)
//	GET    /api/v1/prestige/leaderboard — classement amis
//	GET    /api/v1/templates/suggest    — propositions catalogue
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/prestige"
)

// PrestigeHandler regroupe les routes du module Prestige.
//
// Une seule struct car le module a un service unique et toutes les routes
// sont closely related. Préférable à 5 handlers fragmentés pour la lisibilité.
type PrestigeHandler struct {
	svc prestige.Service
}

// NewPrestigeHandler construit le handler Prestige.
func NewPrestigeHandler(svc prestige.Service) *PrestigeHandler {
	return &PrestigeHandler{svc: svc}
}

// ─────────── DTOs requête/réponse ───────────

type createChallengeBody struct {
	UserID          string  `json:"user_id"`
	TitleSlug       string  `json:"title_slug"`
	ArcID           string  `json:"arc_id,omitempty"`
	TemplateID      string  `json:"template_id,omitempty"`
	Metric          string  `json:"metric"`
	Target          float64 `json:"target"`
	WindowType      string  `json:"window_type"`
	WindowValue     string  `json:"window_value,omitempty"`
	Cadence         string  `json:"cadence"`
	EvalType        string  `json:"eval_type"`
	Mode            string  `json:"mode"`
	Label           string  `json:"label,omitempty"`
	IsPrivate       bool    `json:"is_private,omitempty"`
	TargetPerMember float64 `json:"target_per_member,omitempty"`
	Position        int     `json:"position,omitempty"`
}

type updateChallengeBody struct {
	Target *float64 `json:"target,omitempty"`
	Label  *string  `json:"label,omitempty"`
}

// ─────────── CreateChallenge ───────────

// CreateChallenge gère POST /challenges.
func (h *PrestigeHandler) CreateChallenge(w http.ResponseWriter, r *http.Request) {
	var body createChallengeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	req := prestige.CreateChallengeRequest{
		UserID:          body.UserID,
		TitleSlug:       body.TitleSlug,
		ArcID:           body.ArcID,
		TemplateID:      body.TemplateID,
		Metric:          body.Metric,
		Target:          body.Target,
		WindowType:      prestige.WindowType(body.WindowType),
		WindowValue:     body.WindowValue,
		Cadence:         prestige.Cadence(body.Cadence),
		EvalType:        prestige.EvalType(body.EvalType),
		Mode:            prestige.ChallengeMode(body.Mode),
		Label:           body.Label,
		IsPrivate:       body.IsPrivate,
		TargetPerMember: body.TargetPerMember,
		Position:        body.Position,
	}
	c, err := h.svc.CreateChallenge(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

// ─────────── GetChallenge ───────────

func (h *PrestigeHandler) GetChallenge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing_id", "id requis")
		return
	}
	c, err := h.svc.GetChallenge(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// ─────────── ListActiveChallenges ───────────

func (h *PrestigeHandler) ListActiveChallenges(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	titleSlug := r.URL.Query().Get("title_slug")
	if userID == "" || titleSlug == "" {
		writeError(w, http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
		return
	}
	list, err := h.svc.ListActiveChallenges(r.Context(), userID, titleSlug)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"challenges": list, "count": len(list)})
}

// ─────────── UpdateChallenge ───────────

func (h *PrestigeHandler) UpdateChallenge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing_id", "id requis")
		return
	}
	var body updateChallengeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	c, err := h.svc.UpdateChallenge(r.Context(), id, prestige.UpdateChallengePatch{
		Target: body.Target,
		Label:  body.Label,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// ─────────── AbandonChallenge ───────────

func (h *PrestigeHandler) AbandonChallenge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing_id", "id requis")
		return
	}
	if err := h.svc.AbandonChallenge(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─────────── SuggestNext ───────────

func (h *PrestigeHandler) SuggestNext(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing_id", "id requis")
		return
	}
	templates, err := h.svc.SuggestNext(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"suggestions": templates})
}

// ─────────── GetMyPrestige ───────────

// GetMyPrestige gère GET /prestige/me.
//
// Si title_slug est fourni → vue par titre.
// Sinon → cross-titre (somme tous titres).
func (h *PrestigeHandler) GetMyPrestige(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "missing_user_id", "user_id requis")
		return
	}
	titleSlug := r.URL.Query().Get("title_slug")
	up, err := h.svc.GetUserPrestige(r.Context(), userID, titleSlug)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, up)
}

// ─────────── SuggestTemplates ───────────

func (h *PrestigeHandler) SuggestTemplates(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	titleSlug := r.URL.Query().Get("title_slug")
	if userID == "" || titleSlug == "" {
		writeError(w, http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
		return
	}
	count := 3
	if c := r.URL.Query().Get("count"); c != "" {
		if n, err := strconv.Atoi(c); err == nil && n > 0 && n <= 10 {
			count = n
		}
	}
	templates, err := h.svc.SuggestTemplates(r.Context(), userID, titleSlug, count)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"templates": templates})
}

// ─────────── Helper d'erreurs ───────────

// writeServiceError mappe les erreurs du service vers des codes HTTP.
//
// Centralise la traduction pour éviter la duplication dans chaque handler.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, prestige.ErrChallengeNotFound),
		errors.Is(err, prestige.ErrArcNotFound),
		errors.Is(err, prestige.ErrUserNotFound):
		writeError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, prestige.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, prestige.ErrNotEditable):
		writeError(w, http.StatusForbidden, "not_editable", err.Error())
	case errors.Is(err, prestige.ErrAlreadyTerminal):
		writeError(w, http.StatusConflict, "already_terminal", err.Error())
	case errors.Is(err, prestige.ErrCooldownActive):
		writeError(w, http.StatusTooManyRequests, "cooldown_active", err.Error())
	default:
		// Erreur masquée à l'extérieur — ne pas exposer les internals
		// si la cause n'est pas explicitement une de nos sentinelles.
		msg := err.Error()
		if strings.Contains(msg, "stretch") {
			// Cas particulier RejectTooEasy formaté avec stretch dans le message
			writeError(w, http.StatusBadRequest, "challenge_too_easy", msg)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", msg)
	}
}
