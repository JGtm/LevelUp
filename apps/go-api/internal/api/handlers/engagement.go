// Package handlers — engagement.go : endpoints API EngagementScore (Phase 4 plan).
//
// Routes :
//
//   - GET /api/v1/players/{slug}/matches/{match_id}/engagement
//     Retourne le score d'engagement + la courbe pour un match (Mock 10 Match View).
//
//   - GET /api/v1/players/{slug}/engagement_profile
//     Retourne les coefficients d'engagement par categorie de mode.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// EngagementHandler gere les endpoints engagement.
type EngagementHandler struct {
	newSvc ServiceFactory[*service.PlayerEngagementService]
}

// NewEngagementHandler cree un EngagementHandler.
func NewEngagementHandler(newSvc ServiceFactory[*service.PlayerEngagementService]) *EngagementHandler {
	return &EngagementHandler{newSvc: newSvc}
}

// GetMatchEngagement : GET /matches/{match_id}/engagement
//
// Reponse JSON :
//
//	{
//	  "engagement_score": 62.0,
//	  "residual_brut": 1.45,
//	  "engagement_curve": [{"time_ms": ..., "pace_joueur": ..., "pace_team": ..., "pace_attendu": ..., "pace_lobby": ...}],
//	  "match_intensity": 14.2,
//	  "confidence": "full",
//	  "n_history_matches": 47
//	}
//
// Codes d'erreur :
//   - 404 player_not_found / match_not_found
//   - 400 invalid_request (matchID manquant)
//   - 422 pve_not_supported (match PvE - non couvert v1)
//   - 503 engagement_unavailable (migration Phase 2 non appliquee)
//   - 500 engagement_error (autre)
func (h *EngagementHandler) GetMatchEngagement(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	matchID := chi.URLParam(r, "match_id")
	if matchID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_request", "match_id est requis")
		return
	}

	result, err := svc.GetMatchEngagement(r.Context(), matchID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEngagementMatchNotFound):
			writeError(r.Context(), w, http.StatusNotFound, "match_not_found", "match introuvable pour ce joueur : "+matchID)
		case errors.Is(err, service.ErrEngagementPvENotSupported):
			writeError(r.Context(), w, http.StatusUnprocessableEntity, "pve_not_supported", "engagement non couvert pour les matchs PvE en v1")
		case errors.Is(err, port.ErrEngagementUnavailable):
			writeError(r.Context(), w, http.StatusServiceUnavailable, "engagement_unavailable", "migration EngagementScore non appliquee")
		default:
			writeError(r.Context(), w, http.StatusInternalServerError, "engagement_error", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetEngagementTimeseries : POST /engagement/timeseries
//
// Body JSON (optionnel) :
//
//	{
//	  "filters": { ...FilterContextInput... },
//	  "limit": 30
//	}
//
// Retourne les N derniers matchs PvP du scope avec leurs paces (Mock 11 Timeseries).
// Le scope est defini par `filters` (period, cascade, sessions, match_context)
// — aligne sur le contrat POST /pages/timeseries. `limit` est optionnel
// (defaut 50, max 500). Un body vide est tolere et equivaut a `{}` (compat
// integration tests / smoke).
func (h *EngagementHandler) GetEngagementTimeseries(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.EngagementTimeseriesRequest
	// Body optionnel : si vide ou absent, on garde la valeur zero (filters vide, limit 0 → 50).
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			writeError(r.Context(), w, http.StatusBadRequest, "invalid_json", "corps JSON invalide")
			return
		}
	}
	if err := req.Filters.Validate(); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_filters", err.Error())
		return
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	out, err := svc.GetTimeseries(r.Context(), req.Filters, limit)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "engagement_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// GetSquadEngagementSession : GET /pages/squad/v2/engagement?match_ids=m1,m2&teammates=xuid1,xuid2
//
// Retourne le payload SquadEngagementSession pour Mock 15 v2.
func (h *EngagementHandler) GetSquadEngagementSession(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	matchIDsParam := r.URL.Query().Get("match_ids")
	teammatesParam := r.URL.Query().Get("teammates")
	gamertgsParam := r.URL.Query().Get("teammate_gamertags")

	matchIDs := splitCSV(matchIDsParam)
	teammateXUIDs := splitCSV(teammatesParam)
	teammateGamertags := splitCSV(gamertgsParam)

	// Si aucun match_id fourni, utilise les matchs recents du joueur (limite a 15).
	// On reste sur la granularite "match" : avec limit=15 et workCap=200 le
	// service renvoie 1 point par match avec MatchID non vide.
	if len(matchIDs) == 0 {
		recent, err := svc.GetTimeseries(r.Context(), domain.FilterContextInput{}, 15)
		if err != nil {
			writeError(r.Context(), w, http.StatusInternalServerError, "engagement_error", err.Error())
			return
		}
		matchIDs = make([]string, 0, len(recent.Points))
		for _, m := range recent.Points {
			if m.MatchID == "" {
				continue
			}
			matchIDs = append(matchIDs, m.MatchID)
		}
	}

	// Convertir en EngagementCoefficient — zippe XUID et gamertag (index identique).
	teammates := make([]domain.EngagementCoefficient, 0, len(teammateXUIDs))
	for i, x := range teammateXUIDs {
		gt := ""
		if i < len(teammateGamertags) {
			gt = teammateGamertags[i]
		}
		teammates = append(teammates, domain.EngagementCoefficient{XUID: x, Gamertag: gt})
	}

	session, err := svc.GetSquadSession(r.Context(), matchIDs, teammates)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "engagement_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, session)
}

// splitCSV split "a,b,c" en ["a","b","c"], ignorant les vides.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// PostRecomputeCoefficients : POST /matches/.../engagement/recompute_coefficients
//
// Force le recalcul des coefficients d'engagement (coef_team_share /
// coef_lobby_share) pour toutes les categories de mode supportees, depuis
// les paces persistees dans player_match_enrichment. Utile en admin / debug
// quand un user veut rafraichir ses coefs sans attendre le prochain sync.
//
// Reponse JSON :
//
//	{
//	  "modes_updated": ["PvP_ranked", "PvP_unranked"],
//	  "n_coefs_persisted": 2,
//	  "modes_skipped": []
//	}
//
// Codes d'erreur :
//   - 404 player_not_found
//   - 503 engagement_unavailable (migration Phase 2 ou recompute non appliquee)
//   - 500 engagement_error (autre)
func (h *EngagementHandler) PostRecomputeCoefficients(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	report, err := svc.RecomputeCoefficients(r.Context())
	if err != nil {
		switch {
		case errors.Is(err, port.ErrEngagementUnavailable):
			writeError(r.Context(), w, http.StatusServiceUnavailable, "engagement_unavailable",
				"migration EngagementScore non appliquee")
		default:
			writeError(r.Context(), w, http.StatusInternalServerError, "engagement_error", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// GetEngagementProfile : GET /engagement_profile
//
// Reponse JSON : tableau de coefficients par categorie de mode.
//
//	[
//	  {"xuid": "...", "mode_category": "PvP_ranked", "coef_team_share": 1.12, "coef_lobby_share": 1.05, "n_matches": 187, "last_updated": "..."},
//	  ...
//	]
//
// Reponse vide si aucun coefficient n'a encore ete calcule (cold start).
func (h *EngagementHandler) GetEngagementProfile(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	coefs, err := svc.GetEngagementProfile(r.Context())
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "engagement_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, coefs)
}
