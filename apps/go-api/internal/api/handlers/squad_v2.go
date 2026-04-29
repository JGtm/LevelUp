// Package handlers — squad_v2.go : handler HTTP de la nouvelle page Squad V2
// (multi-coéquipiers, fondations Phase 0).
//
// Endpoint :
//
//	GET /api/v1/players/{player_slug}/pages/squad/v2?teammates=gt1,gt2,gt3&period=1y
//
// Vit en parallèle de l'endpoint legacy /pages/squad jusqu'à migration complète
// du frontend (cf. PLAN_SQUAD_GO_PORTAGE).
package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// squadV2MaxTeammates est la borne haute du nombre de coéquipiers acceptés
// par l'endpoint (cohérent avec service.MaxTeammates et la version Python).
const squadV2MaxTeammates = 3

// SquadV2Handler gère l'endpoint Squad V2.
type SquadV2Handler struct {
	newSvc ContextFactory[port.SquadV2Service]
}

// NewSquadV2Handler construit un SquadV2Handler avec sa factory de service.
func NewSquadV2Handler(newSvc ContextFactory[port.SquadV2Service]) *SquadV2Handler {
	return &SquadV2Handler{newSvc: newSvc}
}

// GetSquadPage traite GET /api/v1/players/{player_slug}/pages/squad/v2.
//
// Query params :
//   - teammates        : CSV de gamertags (max 3, optionnel)
//   - period           : "all" | "2y" | "1y" | "1m" | "1w" (défaut "all")
//   - experience_types : CSV de types d'expérience (ex: "PVP classé,PVE")
//   - playlists        : CSV de noms de playlist (ex: "Ranked Arena")
//
// Statuts :
//   - 200 : payload valide
//   - 400 : params invalides (period inconnu, trop de coéquipiers)
//   - 404 : joueur principal introuvable (factory)
//   - 503 : capability match.history absente pour le titre courant
func (h *SquadV2Handler) GetSquadPage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, _, gamertag, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	teammates, err := parseSquadV2Teammates(r.URL.Query().Get("teammates"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_teammates", err.Error())
		return
	}
	period, err := parseSquadV2Period(r.URL.Query().Get("period"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_period", err.Error())
		return
	}

	experienceTypes := parseCSVFilter(r.URL.Query().Get("experience_types"))
	playlists := parseCSVFilter(r.URL.Query().Get("playlists"))

	titleSlug := titleSlugFromContext(r)

	slog.InfoContext(r.Context(), "squad_v2: lecture page",
		"player", gamertag,
		"title_slug", titleSlug,
		"teammates_count", len(teammates),
		"period", string(period),
		"experience_types", experienceTypes,
		"playlists", playlists,
	)

	resp, err := svc.GetSquadPage(r.Context(), titleSlug, gamertag, teammates, period, experienceTypes, playlists)
	if err != nil {
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			slog.WarnContext(r.Context(), "squad_v2: capability match.history absente",
				"player", gamertag, "title_slug", titleSlug, "err", err)
			writeError(w, http.StatusServiceUnavailable, "capability_not_supported", err.Error())
			return
		}
		slog.ErrorContext(r.Context(), "squad_v2: erreur service",
			"player", gamertag, "title_slug", titleSlug, "err", err)
		writeError(w, http.StatusInternalServerError, "squad_v2_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// parseSquadV2Teammates valide et découpe la liste CSV de coéquipiers.
// Retourne nil (et pas d'erreur) si le paramètre est absent ou vide.
func parseSquadV2Teammates(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		gt := strings.TrimSpace(p)
		if gt == "" {
			continue
		}
		out = append(out, gt)
	}
	if len(out) > squadV2MaxTeammates {
		return nil, fmt.Errorf("max %d coéquipiers, %d fournis", squadV2MaxTeammates, len(out))
	}
	return out, nil
}

// parseSquadV2Period valide la période en s'appuyant sur temporal.Period.
// Vide → temporal.PeriodAll (pas de filtrage temporel).
func parseSquadV2Period(raw string) (temporal.Period, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return temporal.PeriodAll, nil
	}
	p := temporal.Period(raw)
	if !p.IsValid() {
		return "", fmt.Errorf("période %q inconnue (admises : all, 2y, 1y, 1m, 1w)", raw)
	}
	return p, nil
}

// parseCSVFilter découpe un paramètre CSV en slice de strings non-vides.
// Retourne nil si le paramètre est absent ou entièrement vide.
func parseCSVFilter(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// titleSlugFromContext lit le titre courant depuis le contexte (middleware
// title.go) ; fallback chaîne vide laissé au service.
func titleSlugFromContext(r *http.Request) string {
	return ctxkeys.TitleSlug(r.Context())
}
