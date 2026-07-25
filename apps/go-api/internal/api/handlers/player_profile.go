// Package handlers — player_profile.go : endpoint HTTP du PlayerProfile V1
// (PLAN_PLAYER_PROFILE_ASCENSION §8.1).
//
// Route exposée (sous /api/v1/players/{player_slug}/) :
//   - GET /profile : PlayerProfile complet (Sections A1/A2/B/C).
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (hérite ownership/title + lit {player_slug} parent) et enregistre via huma.Get.
// Logique métier inchangée (profile.BuildProfile), seul le wrapping HTTP change.
//
// Le PlayerProfile est construit à la volée via progression/profile.BuildProfile
// (pas de cache en V1 — la fenêtre LOWESS est de 30 jours et le calcul tient
// en < 200ms sur 100 matchs).
package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/progression/profile"
)

// ProfileWindowDays : fenêtre par défaut pour les agrégats (Section A1/A2/B/C).
const ProfileWindowDays = 30

// ActivityCalendarWindowDays : fenêtre par défaut du calendrier d'activité (DEC-5/D3).
const ActivityCalendarWindowDays = 90

// PlayerProfileHandler regroupe le endpoint /profile.
type PlayerProfileHandler struct {
	resolve   ProgressionResolver
	titleSlug string
	// V2 §2 : mapping awards → axes (optionnel, enrichit la Section A1 radar).
	// Sans mapping, Objective reste à 0 (fallback V1).
	awards *mappings.AwardMappingSet
}

// NewPlayerProfileHandler construit le handler.
func NewPlayerProfileHandler(resolve ProgressionResolver, titleSlug string) *PlayerProfileHandler {
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	return &PlayerProfileHandler{resolve: resolve, titleSlug: titleSlug}
}

// WithAwardMapping injecte le mapping awards → axes (V2 §2). Chainable.
func (h *PlayerProfileHandler) WithAwardMapping(set *mappings.AwardMappingSet) *PlayerProfileHandler {
	h.awards = set
	return h
}

// Mount enregistre /profile via Huma (Phase 3b) sur le sous-routeur chi
// (préfixe /players/{player_slug} + middleware ownership/title hérités).
func (h *PlayerProfileHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/profile", h.handleGetProfile, humacore.Op("getPlayerProfile", "Profil joueur complet (radar 6 axes, style FK/FD, tier LUSR + 8 composantes, leviers, suggestions)", "progression"))
	huma.Get(api, "/activity-calendar", h.handleGetActivityCalendar, humacore.Op("getActivityCalendar", "Calendrier d'activité : nombre de matchs par jour UTC sur la fenêtre (jours vides omis)", "progression"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// profileInput : {player_slug} parent + ?window_days= (parse tolérant maison).
// window_days reste en STRING pour reproduire le contrat d'origine : une valeur
// invalide retombe sur le défaut (atoi → 0 → ProfileWindowDays), PAS le 422 de
// validation Huma qu'un `int` produirait.
type profileInput struct {
	PlayerSlug string `path:"player_slug"`
	WindowDays string `query:"window_days"`
}

type profileOutput struct{ Body *profile.PlayerProfile }

// activityCalendarInput : {player_slug} parent + ?days= (parse tolérant maison,
// même contrat que window_days : valeur invalide → défaut).
type activityCalendarInput struct {
	PlayerSlug string `path:"player_slug"`
	Days       string `query:"days" doc:"Fenêtre en jours (clampé 7..180, défaut 90)"`
}

type activityCalendarOutput struct{ Body *profile.ActivityCalendar }

// ─── Endpoints ─────────────────────────────────────────────────────────────

// handleGetProfile : GET /profile → PlayerProfile complet.
//
// Query params optionnels :
//   - window_days : fenêtre d'analyse (défaut 30, min 7, max 120)
func (h *PlayerProfileHandler) handleGetProfile(ctx context.Context, in *profileInput) (*profileOutput, error) {
	pdb, err := h.resolvePlayer(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	window := atoi(in.WindowDays)
	if window <= 0 {
		window = ProfileWindowDays
	}
	if window < 7 {
		window = 7
	}
	if window > 120 {
		window = 120
	}

	svc := profile.NewServiceFromPlayerDB(pdb)
	if h.awards != nil {
		svc = svc.WithAwardMapping(h.awards)
	}
	prof, err := svc.BuildProfile(ctx, pdb.XUID, requestTitleSlug(ctx, h.titleSlug), window, time.Now().UTC())
	if err != nil {
		slog.WarnContext(ctx, "profile: build", "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "build_profile_error", err.Error())
	}
	return &profileOutput{Body: prof}, nil
}

// handleGetActivityCalendar : GET /activity-calendar → jours joués sur la fenêtre
// (DEC-5/D3). Query param optionnel `days` (défaut 90, min 7, max 180).
func (h *PlayerProfileHandler) handleGetActivityCalendar(ctx context.Context, in *activityCalendarInput) (*activityCalendarOutput, error) {
	pdb, err := h.resolvePlayer(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	days := atoi(in.Days)
	if days <= 0 {
		days = ActivityCalendarWindowDays
	}
	if days < 7 {
		days = 7
	}
	if days > 180 {
		days = 180
	}
	now := time.Now().UTC()
	cal, err := profile.NewServiceFromPlayerDB(pdb).LoadActivityCalendar(ctx, pdb.XUID, now.AddDate(0, 0, -days), now)
	if err != nil {
		slog.WarnContext(ctx, "profile: activity calendar", "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "activity_calendar_error", err.Error())
	}
	return &activityCalendarOutput{Body: &cal}, nil
}

// resolvePlayer résout le slug courant ou renvoie une erreur Huma 404
// (contrat préservé : {code:player_not_found}).
func (h *PlayerProfileHandler) resolvePlayer(ctx context.Context, slug string) (*duckdb.PlayerDB, error) {
	pdb, err := h.resolve(ctx, slug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	return pdb, nil
}
