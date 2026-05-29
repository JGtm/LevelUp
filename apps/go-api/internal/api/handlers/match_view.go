// Package handlers — MatchViewHandler : GET .../matches/{match_id}.
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// playlistOrSessionPattern : whitelist de chars pour les valeurs de filtres
// passés en query param. Empêche injection même si le repo utilise des
// paramètres préparés. Limite 64 chars (cf. plan Phase 2b §5).
var playlistOrSessionPattern = regexp.MustCompile(`^[A-Za-z0-9 _:.\-]{1,64}$`)

// xuidPattern : XUID Halo = entier décimal (jusqu'à 32 chars). Phase 2c
// (with_player). Mêmes contraintes que côté front (parseFilterSpecFromSearch).
var xuidPattern = regexp.MustCompile(`^\d{1,32}$`)

// parseNeighborsFilterSpec : extrait MatchFilterSpec depuis r.URL.Query().
// Tout filtre invalide est silencieusement ignoré + log warning. Jamais 400.
//
// Les noms de query params utilisés (côté front, voir useNavigateToMatch
// Phase 2b) :
//   - playlist (PlaylistName)
//   - mode (ModeCategory)
//   - from / to (DateFrom / DateTo, ISO 8601 / RFC3339)
//   - session (SessionID)
//   - outcome (Outcome — whitelist win/loss/draw/dnf)
func parseNeighborsFilterSpec(r *http.Request) *domain.MatchFilterSpec {
	q := r.URL.Query()
	spec := &domain.MatchFilterSpec{}
	ctx := r.Context()

	// playlist / mode : multi-valeurs séparées par virgule (Phase 3).
	if vals := parseCsvFilterParam(ctx, q.Get("playlist"), "playlist"); len(vals) > 0 {
		spec.PlaylistNames = vals
	}
	if vals := parseCsvFilterParam(ctx, q.Get("mode"), "mode"); len(vals) > 0 {
		spec.ModeCategories = vals
	}
	if v := strings.TrimSpace(q.Get("session")); v != "" {
		if playlistOrSessionPattern.MatchString(v) {
			spec.SessionID = &v
		} else {
			slog.WarnContext(ctx, "neighbors: invalid filter param ignored",
				"param", "session", "value", v)
		}
	}
	if v := strings.TrimSpace(q.Get("outcome")); v != "" {
		if analysis.IsValidOutcomeLabel(v) {
			spec.Outcome = &v
		} else {
			slog.WarnContext(ctx, "neighbors: invalid filter param ignored",
				"param", "outcome", "value", v)
		}
	}
	if v := strings.TrimSpace(q.Get("from")); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			spec.DateFrom = &t
		} else {
			slog.WarnContext(ctx, "neighbors: invalid filter param ignored",
				"param", "from", "value", v, "err", err)
		}
	}
	if v := strings.TrimSpace(q.Get("to")); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			spec.DateTo = &t
		} else {
			slog.WarnContext(ctx, "neighbors: invalid filter param ignored",
				"param", "to", "value", v, "err", err)
		}
	}
	if v := strings.TrimSpace(q.Get("with_player")); v != "" {
		if xuidPattern.MatchString(v) {
			spec.WithPlayerXuid = &v
		} else {
			slog.WarnContext(ctx, "neighbors: invalid filter param ignored",
				"param", "with_player", "value", v)
		}
	}
	if spec.IsEmpty() {
		return nil
	}
	return spec
}

// parseCsvFilterParam : découpe une valeur de query param en valeurs multiples
// (séparateur virgule), trim + valide chacune via la whitelist. Les valeurs
// invalides sont ignorées individuellement (log warn). Retourne nil si aucune
// valeur valide — préserve le comportement mono-valeur (1 valeur → slice de 1).
func parseCsvFilterParam(ctx context.Context, raw, param string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}
		if playlistOrSessionPattern.MatchString(v) {
			out = append(out, v)
		} else {
			slog.WarnContext(ctx, "neighbors: invalid filter param ignored",
				"param", param, "value", v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// MatchViewHandler gère GET /players/{player_slug}/matches/{match_id}.
type MatchViewHandler struct {
	newSvc ServiceFactory[port.MatchViewService]
}

// NewMatchViewHandler crée un MatchViewHandler.
func NewMatchViewHandler(newSvc ServiceFactory[port.MatchViewService]) *MatchViewHandler {
	return &MatchViewHandler{newSvc: newSvc}
}

// GetMatchView retourne la vue détaillée d'un match pour un joueur.
// GET /api/v1/players/{player_slug}/matches/{match_id}
func (h *MatchViewHandler) GetMatchView(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	matchID := chi.URLParam(r, "match_id")
	if matchID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_match_id", "match_id est requis")
		return
	}

	resp, err := svc.GetMatchView(r.Context(), matchID)
	if err != nil {
		var apiErr *domain.APIError
		if errors.As(err, &apiErr) && apiErr.Code == "not_found" {
			writeError(r.Context(), w, http.StatusNotFound, "match_not_found", apiErr.Message)
			return
		}
		if strings.Contains(err.Error(), "no rows") || strings.Contains(err.Error(), "no rows in result set") {
			writeError(r.Context(), w, http.StatusNotFound, "match_not_found", "match introuvable : "+matchID)
			return
		}
		writeError(r.Context(), w, http.StatusInternalServerError, "match_view_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetMatchNeighbors retourne les matchs adjacents pour la navigation prev/next.
// GET /api/v1/players/{player_slug}/matches/{match_id}/neighbors
func (h *MatchViewHandler) GetMatchNeighbors(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	matchID := chi.URLParam(r, "match_id")
	if matchID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_match_id", "match_id est requis")
		return
	}

	spec := parseNeighborsFilterSpec(r)
	var (
		resp domain.MatchNeighbors
	)
	if spec != nil {
		resp, err = svc.GetMatchNeighborsFiltered(r.Context(), matchID, spec)
	} else {
		resp, err = svc.GetMatchNeighbors(r.Context(), matchID)
	}
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "neighbors_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
