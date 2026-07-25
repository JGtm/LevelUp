// Package handlers — engagement.go : endpoints API EngagementScore (Phase 4 plan).
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (hérite ownership/title + lit {player_slug} parent) et enregistre via huma.*.
// Logique métier inchangée (PlayerEngagementService), seul le wrapping HTTP change.
//
// Routes :
//
//   - GET /api/v1/players/{slug}/matches/{match_id}/engagement
//     Retourne le score d'engagement + la courbe pour un match (Mock 10 Match View).
//
//   - GET /api/v1/players/{slug}/engagement_profile
//     Retourne les coefficients d'engagement par categorie de mode.
//
//   - POST /api/v1/players/{slug}/engagement/timeseries
//     Retourne les N derniers matchs PvP du scope avec leurs paces (Mock 11).
//
//   - GET /api/v1/players/{slug}/pages/squad/v2/engagement
//     Retourne le payload SquadEngagementSession (Mock 15 v2).
//
//   - POST /api/v1/players/{slug}/engagement/recompute_coefficients
//     Force le recalcul des coefficients d'engagement.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
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

// Mount enregistre tous les endpoints via Huma sur le sous-routeur chi
// (préfixe /players/{player_slug} + middleware ownership/title hérités).
func (h *EngagementHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/matches/{match_id}/engagement", h.handleMatchEngagement, humacore.Op("getMatchEngagement", "Score d'engagement + courbe pour un match (P3.2)", "match-view"))
	huma.Get(api, "/engagement_profile", h.handleEngagementProfile, humacore.Op("getEngagementProfile", "Profil engagement long-terme du joueur", "career"))
	huma.Post(api, "/engagement/timeseries", h.handleEngagementTimeseries, humacore.Op("postEngagementTimeseries", "Série temporelle de l'engagement filtrée (Mock 11)", "career"))
	// Body {filters, limit} OPTIONNEL : un body absent équivaut à `{}` (compat
	// smoke/integration). Sans ça, Huma exige le RawBody → 400 "request body is
	// required" (régression vs l'ancien Body pointeur tolérant).
	humacore.MarkRequestBodyOptional(api, http.MethodPost, "/engagement/timeseries")
	huma.Get(api, "/pages/squad/v2/engagement", h.handleSquadEngagementSession, humacore.Op("getSquadEngagementSession", "Engagement squad pour la session courante", "squad"))
	huma.Post(api, "/engagement/recompute_coefficients", h.handleRecomputeCoefficients, humacore.Op("postRecomputeEngagementCoefficients", "Force le recalcul des coefficients d'engagement (admin)", "career"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// engPlayerInput : path param parent {player_slug} seul.
type engPlayerInput struct {
	PlayerSlug string `path:"player_slug"`
}

// engMatchInput : {player_slug} + {match_id} (string ; parse maison pour
// reproduire le 400 invalid_request quand match_id est vide).
type engMatchInput struct {
	PlayerSlug string `path:"player_slug"`
	MatchID    string `path:"match_id"`
}

// engTimeseriesInput : {player_slug} + corps BRUT décodé à la main.
//
// RawBody (et non un Body typé Huma) : le front envoie period.start_date/end_date
// = null. Huma traite *time.Time comme optionnel mais PAS nullable et rejette le
// null en 422 (validation_error). On décode donc manuellement via json.Unmarshal
// (permissif : null → *time.Time nil), comme les endpoints filtres (cf.
// decodeFiltersBody, handlers/filters.go). Un body absent/vide équivaut à `{}`
// (compat integration tests / smoke).
type engTimeseriesInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}

// decodeEngagementTimeseriesBody décode le corps brut {filters, limit} à la main.
// json.Unmarshal tolère period.start_date/end_date = null que le schéma Huma
// rejetterait en 422 (cf. decodeFiltersBody). Body absent/vide → requête zéro
// (filters vide, limit 0 → défaut côté handler). Applique la validation métier
// des filtres (400 invalid_filters le cas échéant).
func decodeEngagementTimeseriesBody(raw []byte) (domain.EngagementTimeseriesRequest, error) {
	var req domain.EngagementTimeseriesRequest
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &req); err != nil {
			return req, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
		}
	}
	if err := req.Filters.Validate(); err != nil {
		return req, humacore.NewError(http.StatusBadRequest, "invalid_filters", err.Error())
	}
	return req, nil
}

// engSquadInput : {player_slug} + query params CSV (parse maison comme avant).
type engSquadInput struct {
	PlayerSlug        string `path:"player_slug"`
	MatchIDs          string `query:"match_ids"`
	Teammates         string `query:"teammates"`
	TeammateGamertags string `query:"teammate_gamertags"`
}

type engMatchOutput struct{ Body *domain.EngagementScoreResult }
type engProfileOutput struct {
	Body []domain.EngagementProfile
}
type engTimeseriesOutput struct {
	Body *domain.EngagementTimeseriesResponse
}
type engSquadOutput struct {
	Body *domain.SquadEngagementSession
}
type engRecomputeOutput struct{ Body *service.RecomputeReport }

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleMatchEngagement : GET /matches/{match_id}/engagement
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
func (h *EngagementHandler) handleMatchEngagement(ctx context.Context, in *engMatchInput) (*engMatchOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}

	if in.MatchID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_request", "match_id est requis")
	}

	result, err := svc.GetMatchEngagement(ctx, in.MatchID)
	if err != nil {
		// Porte statique F7 : titre sans engagement.score (not_exposed) → 503 propre
		// centralise (jamais un 500 ni un score faux), pattern B15.
		if mapped, ok := MapCapabilityError(ctx, err, "engagement.score"); ok {
			return nil, mapped
		}
		switch {
		case errors.Is(err, service.ErrEngagementMatchNotFound):
			return nil, humacore.NewError(http.StatusNotFound, "match_not_found", "match introuvable pour ce joueur : "+in.MatchID)
		case errors.Is(err, service.ErrEngagementPvENotSupported):
			return nil, humacore.NewError(http.StatusUnprocessableEntity, "pve_not_supported", "engagement non couvert pour les matchs PvE en v1")
		case errors.Is(err, service.ErrEngagementInsufficient):
			return nil, humacore.NewError(http.StatusUnprocessableEntity, "engagement_insufficient", "engagement indisponible pour ce match (trop court ou peu d'action)")
		case errors.Is(err, port.ErrEngagementUnavailable):
			return nil, humacore.NewError(http.StatusServiceUnavailable, "engagement_unavailable", "migration EngagementScore non appliquee")
		default:
			return nil, humacore.NewError(http.StatusInternalServerError, "engagement_error", err.Error())
		}
	}

	return &engMatchOutput{Body: result}, nil
}

// handleEngagementTimeseries : POST /engagement/timeseries
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
func (h *EngagementHandler) handleEngagementTimeseries(ctx context.Context, in *engTimeseriesInput) (*engTimeseriesOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}

	req, err := decodeEngagementTimeseriesBody(in.RawBody)
	if err != nil {
		// Observabilité : le 422 historique (period.start_date null rejeté par la
		// validation Huma du body typé) était SILENCIEUX — visible seulement dans
		// http.log (status 422), jamais expliqué. On logge désormais tout rejet de
		// body engagement dans handlers.log pour rendre le cas diagnostiquable.
		slog.WarnContext(ctx, "engagement_timeseries: body invalide",
			"player", in.PlayerSlug, "err", err)
		return nil, err
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	out, err := svc.GetTimeseries(ctx, req.Filters, limit)
	if err != nil {
		slog.ErrorContext(ctx, "engagement_timeseries: GetTimeseries a échoué",
			"player", in.PlayerSlug, "limit", limit, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "engagement_error", err.Error())
	}
	return &engTimeseriesOutput{Body: out}, nil
}

// handleSquadEngagementSession : GET /pages/squad/v2/engagement?match_ids=m1,m2&teammates=xuid1,xuid2
//
// Retourne le payload SquadEngagementSession pour Mock 15 v2.
func (h *EngagementHandler) handleSquadEngagementSession(ctx context.Context, in *engSquadInput) (*engSquadOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}

	matchIDs := splitCSV(in.MatchIDs)
	teammateXUIDs := splitCSV(in.Teammates)
	teammateGamertags := splitCSV(in.TeammateGamertags)

	// Si aucun match_id fourni, utilise les matchs recents du joueur (limite a 15).
	//
	// PIEGE CONNU (2026-06-10) : GetTimeseries applique un binning adaptatif —
	// sur un gros historique il renvoie des agregats session/semaine/saison
	// avec MatchID vide → 0 match derive → session vide silencieuse. Le front
	// (SquadContributionsPage) passe desormais match_ids explicitement ; ce
	// fallback ne sert que d'eventuels autres consommateurs. On logge en WARN
	// quand la derivation echoue pour rendre le cas observable.
	if len(matchIDs) == 0 {
		recent, err := svc.GetTimeseries(ctx, domain.FilterContextInput{}, 15)
		if err != nil {
			return nil, humacore.NewError(http.StatusInternalServerError, "engagement_error", err.Error())
		}
		matchIDs = make([]string, 0, len(recent.Points))
		for _, m := range recent.Points {
			if m.MatchID == "" {
				continue
			}
			matchIDs = append(matchIDs, m.MatchID)
		}
		if len(matchIDs) == 0 && len(recent.Points) > 0 {
			slog.WarnContext(ctx, "engagement_squad: derivation match_ids vide (binning agrege, MatchID absent des points)",
				"player", in.PlayerSlug, "points", len(recent.Points), "granularity", recent.Granularity)
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

	session, err := svc.GetSquadSession(ctx, matchIDs, teammates)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "engagement_error", err.Error())
	}
	return &engSquadOutput{Body: session}, nil
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

// handleRecomputeCoefficients : POST /engagement/recompute_coefficients
//
// Force le recalcul du coef lobby global + des bins de reponse (modele
// lobby-anchored v2) pour toutes les categories de mode supportees, depuis les
// paces persistees dans player_match_enrichment. Utile en admin / debug quand un
// user veut rafraichir ses coefs sans attendre le prochain sync.
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
//   - 503 db_busy (database occupée, Retry-After: 5)
//   - 500 engagement_error (autre)
func (h *EngagementHandler) handleRecomputeCoefficients(ctx context.Context, in *engPlayerInput) (*engRecomputeOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}

	report, err := svc.RecomputeCoefficients(ctx)
	if err != nil {
		switch {
		case errors.Is(err, port.ErrEngagementUnavailable):
			return nil, humacore.NewError(http.StatusServiceUnavailable, "engagement_unavailable",
				"migration EngagementScore non appliquee")
		case errors.Is(err, dblease.ErrDBLocked):
			return nil, huma.ErrorWithHeaders(
				humacore.NewError(http.StatusServiceUnavailable, "db_busy", "database is currently busy, please retry"),
				http.Header{"Retry-After": []string{"5"}},
			)
		default:
			return nil, humacore.NewError(http.StatusInternalServerError, "engagement_error", err.Error())
		}
	}
	return &engRecomputeOutput{Body: report}, nil
}

// handleEngagementProfile : GET /engagement_profile
//
// Reponse JSON : tableau de profils par categorie de mode (modele lobby-anchored
// v2). Chaque profil porte le coef lobby global + les bins de reponse par bin
// d'intensite. coef_team_share n'est plus expose (D5).
//
//	[
//	  {"xuid": "...", "mode_category": "PvP_ranked", "coef_lobby_share": 1.05, "n_matches": 187,
//	   "last_updated": "...", "bins": [{"bin": "calme", "lower_bound": 0, "upper_bound": 4.2,
//	   "coef_lobby": 1.3, "n_matches": 62}, ...]},
//	  ...
//	]
//
// Reponse vide si aucun coefficient n'a encore ete calcule (cold start).
func (h *EngagementHandler) handleEngagementProfile(ctx context.Context, in *engPlayerInput) (*engProfileOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}

	coefs, err := svc.GetEngagementProfile(ctx)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "engagement_error", err.Error())
	}
	return &engProfileOutput{Body: coefs}, nil
}

// resolve récupère le service pour le slug courant ou renvoie une erreur Huma 404.
func (h *EngagementHandler) resolve(ctx context.Context, slug string) (*service.PlayerEngagementService, error) {
	svc, err := h.newSvc(ctx, slug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	return svc, nil
}
