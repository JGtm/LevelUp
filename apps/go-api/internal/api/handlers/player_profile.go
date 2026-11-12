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
	"levelup/go-api/internal/progression/profile"
)

// ProfileWindowDays : fenêtre par défaut pour les agrégats (Section A1/A2/B/C).
const ProfileWindowDays = 30

// ActivityCalendarWindowDays : fenêtre par défaut du calendrier d'activité (DEC-5/D3).
const ActivityCalendarWindowDays = 90

// PlayerProfileHandler regroupe le endpoint /profile.
type PlayerProfileHandler struct {
	svc       profile.PlayerProfileService
	titleSlug string
}

// NewPlayerProfileHandler construit le handler.
func NewPlayerProfileHandler(svc profile.PlayerProfileService, titleSlug string) *PlayerProfileHandler {
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	return &PlayerProfileHandler{svc: svc, titleSlug: titleSlug}
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

	prof, err := h.svc.Build(ctx, in.PlayerSlug, requestTitleSlug(ctx, h.titleSlug), window, time.Now().UTC())
	if err != nil {
		if isPlayerNotFound(err) {
			return nil, progressionHTTPError(ctx, err)
		}
		slog.WarnContext(ctx, "profile: build", "err", err)
		return nil, humacore.NewErrorFrom(ctx, http.StatusInternalServerError, "build_profile_error", err)
	}
	return &profileOutput{Body: prof}, nil
}

// handleGetActivityCalendar : GET /activity-calendar → jours joués sur la fenêtre
// (DEC-5/D3). Query param optionnel `days` (défaut 90, min 7, max 180).
func (h *PlayerProfileHandler) handleGetActivityCalendar(ctx context.Context, in *activityCalendarInput) (*activityCalendarOutput, error) {
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
	cal, err := h.svc.ActivityCalendar(ctx, in.PlayerSlug, now.AddDate(0, 0, -days), now)
	if err != nil {
		if isPlayerNotFound(err) {
			return nil, progressionHTTPError(ctx, err)
		}
		slog.WarnContext(ctx, "profile: activity calendar", "err", err)
		return nil, humacore.NewErrorFrom(ctx, http.StatusInternalServerError, "activity_calendar_error", err)
	}
	return &activityCalendarOutput{Body: &cal}, nil
}
