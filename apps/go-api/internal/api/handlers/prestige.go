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
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/platform/dblease"
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
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
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
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

// ─────────── GetChallenge ───────────

func (h *PrestigeHandler) GetChallenge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_id", "id requis")
		return
	}
	c, err := h.svc.GetChallenge(r.Context(), id)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// ─────────── ListActiveChallenges ───────────

func (h *PrestigeHandler) ListActiveChallenges(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	titleSlug := r.URL.Query().Get("title_slug")
	if userID == "" || titleSlug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
		return
	}
	list, err := h.svc.ListActiveChallenges(r.Context(), userID, titleSlug)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"challenges": list, "count": len(list)})
}

// ─────────── UpdateChallenge ───────────

func (h *PrestigeHandler) UpdateChallenge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_id", "id requis")
		return
	}
	var body updateChallengeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	c, err := h.svc.UpdateChallenge(r.Context(), id, prestige.UpdateChallengePatch{
		Target: body.Target,
		Label:  body.Label,
	})
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// ─────────── AbandonChallenge ───────────

func (h *PrestigeHandler) AbandonChallenge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_id", "id requis")
		return
	}
	if err := h.svc.AbandonChallenge(r.Context(), id); err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─────────── SuggestNext ───────────

func (h *PrestigeHandler) SuggestNext(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_id", "id requis")
		return
	}
	templates, err := h.svc.SuggestNext(r.Context(), id)
	if err != nil {
		writeServiceError(r.Context(), w, err)
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
		writeError(r.Context(), w, http.StatusBadRequest, "missing_user_id", "user_id requis")
		return
	}
	titleSlug := r.URL.Query().Get("title_slug")
	up, err := h.svc.GetUserPrestige(r.Context(), userID, titleSlug)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, up)
}

// ─────────── SuggestTemplates ───────────

func (h *PrestigeHandler) SuggestTemplates(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	titleSlug := r.URL.Query().Get("title_slug")
	if userID == "" || titleSlug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
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
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"templates": templates})
}

// ─────────── Arcs ───────────

type createArcBody struct {
	UserID      string `json:"user_id"`
	TitleSlug   string `json:"title_slug"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// CreateArc gère POST /arcs.
func (h *PrestigeHandler) CreateArc(w http.ResponseWriter, r *http.Request) {
	var body createArcBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	a, err := h.svc.CreateArc(r.Context(), prestige.CreateArcRequest{
		UserID:      body.UserID,
		TitleSlug:   body.TitleSlug,
		Title:       body.Title,
		Description: body.Description,
	})
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

// GetArc gère GET /arcs/{id}.
func (h *PrestigeHandler) GetArc(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_id", "id requis")
		return
	}
	a, err := h.svc.GetArc(r.Context(), id)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// ListArcs gère GET /arcs?user_id=&title_slug=.
func (h *PrestigeHandler) ListArcs(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	titleSlug := r.URL.Query().Get("title_slug")
	if userID == "" || titleSlug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
		return
	}
	arcs, err := h.svc.ListArcs(r.Context(), userID, titleSlug)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"arcs": arcs, "count": len(arcs)})
}

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

// CreateSquadChallenge gère POST /squads/{squad_id}/challenges.
func (h *PrestigeHandler) CreateSquadChallenge(w http.ResponseWriter, r *http.Request) {
	squadID := chi.URLParam(r, "squad_id")
	if squadID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_squad_id", "squad_id requis")
		return
	}
	var body createSquadChallengeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	body.SquadID = squadID
	sc, err := h.svc.CreateSquadChallenge(r.Context(), prestige.CreateSquadChallengeRequest{
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
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sc)
}

// ListSquadChallenges gère GET /squads/{squad_id}/challenges.
func (h *PrestigeHandler) ListSquadChallenges(w http.ResponseWriter, r *http.Request) {
	squadID := chi.URLParam(r, "squad_id")
	if squadID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_squad_id", "squad_id requis")
		return
	}
	list, err := h.svc.ListSquadChallenges(r.Context(), squadID)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"squad_challenges": list, "count": len(list)})
}

type joinSquadChallengeBody struct {
	UserID     string `json:"user_id"`
	ChosenTier string `json:"chosen_tier,omitempty"`
	IsPrivate  bool   `json:"is_private,omitempty"`
}

// JoinSquadChallenge gère POST /squad-challenges/{id}/join.
func (h *PrestigeHandler) JoinSquadChallenge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_id", "id requis")
		return
	}
	var body joinSquadChallengeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if err := h.svc.JoinSquadChallenge(r.Context(), id, body.UserID, prestige.Tier(body.ChosenTier), body.IsPrivate); err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─────────── Mode pilote ───────────

type pilotModeBody struct {
	UserID    string `json:"user_id"`
	TitleSlug string `json:"title_slug"`
}

// EnablePilotMode gère POST /pilot-mode/enable.
//
// Active le mode pilote pour un joueur : auto-attribue 1 quotidien + 1 hebdo
// forcé + propose 3 hebdo au choix. Idempotent : si des défis pilote actifs
// existent déjà sur ces cadences, ils sont conservés.
func (h *PrestigeHandler) EnablePilotMode(w http.ResponseWriter, r *http.Request) {
	var body pilotModeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if body.UserID == "" || body.TitleSlug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
		return
	}
	out, err := h.svc.EnablePilotMode(r.Context(), body.UserID, body.TitleSlug)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// DisablePilotMode gère POST /pilot-mode/disable.
//
// Désactive le mode pilote. Les défis pilote en cours sont conservés (le
// joueur peut les terminer), aucune nouvelle auto-attribution ne se fera.
func (h *PrestigeHandler) DisablePilotMode(w http.ResponseWriter, r *http.Request) {
	var body pilotModeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if body.UserID == "" || body.TitleSlug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_params", "user_id et title_slug requis")
		return
	}
	if err := h.svc.DisablePilotMode(r.Context(), body.UserID, body.TitleSlug); err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─────────── Pool collectif squad ───────────

type refreshSquadPoolBody struct {
	TitleSlug   string `json:"title_slug"`
	RequestedBy string `json:"requested_by"`
}

// RefreshSquadPool gère POST /squads/{squad_id}/challenges/pool/refresh.
//
// Génère un pool de 6-9 templates thématiques pour l'escouade. Le membre
// qui requête doit être dans l'escouade. Le pool est ensuite consommé par
// les membres qui peuvent en proposer un défi à l'équipe.
func (h *PrestigeHandler) RefreshSquadPool(w http.ResponseWriter, r *http.Request) {
	squadID := chi.URLParam(r, "squad_id")
	if squadID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_squad_id", "squad_id requis")
		return
	}
	var body refreshSquadPoolBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if body.TitleSlug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_title_slug", "title_slug requis")
		return
	}
	pool, err := h.svc.RefreshSquadPool(r.Context(), squadID, body.TitleSlug, body.RequestedBy)
	if err != nil {
		writeServiceError(r.Context(), w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"pool": pool, "count": len(pool)})
}

// ─────────── Helper d'erreurs ───────────

// writeServiceError mappe les erreurs du service vers des codes HTTP.
//
// Centralise la traduction pour éviter la duplication dans chaque handler.
func writeServiceError(ctx context.Context, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dblease.ErrDBLocked):
		// Le sync engine ou un autre handler tient le lease — on demande au
		// client de retry sous 5 s. Cf. plan db-concurrency commit 2.
		w.Header().Set("Retry-After", "5")
		writeError(ctx, w, http.StatusServiceUnavailable, "db_busy",
			"database is currently busy, please retry")
	case errors.Is(err, prestige.ErrChallengeNotFound),
		errors.Is(err, prestige.ErrArcNotFound),
		errors.Is(err, prestige.ErrUserNotFound):
		writeError(ctx, w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, prestige.ErrInvalidInput):
		writeError(ctx, w, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, prestige.ErrNotEditable):
		writeError(ctx, w, http.StatusForbidden, "not_editable", err.Error())
	case errors.Is(err, prestige.ErrAlreadyTerminal):
		writeError(ctx, w, http.StatusConflict, "already_terminal", err.Error())
	case errors.Is(err, prestige.ErrCooldownActive):
		writeError(ctx, w, http.StatusTooManyRequests, "cooldown_active", err.Error())
	default:
		// Erreur masquée à l'extérieur — ne pas exposer les internals
		// si la cause n'est pas explicitement une de nos sentinelles.
		msg := err.Error()
		if strings.Contains(msg, "stretch") {
			// Cas particulier RejectTooEasy formaté avec stretch dans le message
			writeError(ctx, w, http.StatusBadRequest, "challenge_too_easy", msg)
			return
		}
		writeError(ctx, w, http.StatusInternalServerError, "internal_error", msg)
	}
}
