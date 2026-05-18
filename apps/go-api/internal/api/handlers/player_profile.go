// Package handlers — player_profile.go : endpoint HTTP du PlayerProfile V1
// (PLAN_PLAYER_PROFILE_ASCENSION §8.1).
//
// Route exposée (sous /api/v1/players/{player_slug}/) :
//   - GET /profile : PlayerProfile complet (Sections A1/A2/B/C).
//
// Le PlayerProfile est construit à la volée via progression/profile.BuildProfile
// (pas de cache en V1 — la fenêtre LOWESS est de 30 jours et le calcul tient
// en < 200ms sur 100 matchs).
package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/progression/profile"
)

// ProfileWindowDays : fenêtre par défaut pour les agrégats (Section A1/A2/B/C).
const ProfileWindowDays = 30

// PlayerProfileHandler regroupe le endpoint /profile.
type PlayerProfileHandler struct {
	resolve   ProgressionResolver
	titleSlug string
}

// NewPlayerProfileHandler construit le handler.
func NewPlayerProfileHandler(resolve ProgressionResolver, titleSlug string) *PlayerProfileHandler {
	if titleSlug == "" {
		titleSlug = "halo_infinite"
	}
	return &PlayerProfileHandler{resolve: resolve, titleSlug: titleSlug}
}

// Mount enregistre /profile sur un router chi sous-monté.
func (h *PlayerProfileHandler) Mount(r chi.Router) {
	r.Get("/profile", h.GetProfile)
}

// GetProfile : GET /profile → PlayerProfile complet.
//
// Query params optionnels :
//   - window_days : fenêtre d'analyse (défaut 30, min 7, max 120)
func (h *PlayerProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	pdb, ok := h.resolveOr404(w, r)
	if !ok {
		return
	}
	window := atoi(r.URL.Query().Get("window_days"))
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
	prof, err := svc.BuildProfile(r.Context(), pdb.XUID, h.titleSlug, window, time.Now().UTC())
	if err != nil {
		slog.WarnContext(r.Context(), "profile: build", "err", err)
		writeError(w, http.StatusInternalServerError, "build_profile_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, prof)
}

// resolveOr404 résout le slug courant ou écrit 404.
func (h *PlayerProfileHandler) resolveOr404(w http.ResponseWriter, r *http.Request) (*duckdb.PlayerDB, bool) {
	slug := chi.URLParam(r, "player_slug")
	pdb, err := h.resolve(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return nil, false
	}
	return pdb, true
}
