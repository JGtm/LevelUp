package handlers

// patterns.go — endpoint GET /patterns pour le Pattern Engine v3.
//
// Route : GET /api/v1/players/{player_slug}/patterns?n=50
// Query params :
//   - n : nombre de matchs récents à analyser (défaut 50, min 10, max 200)
//
// L'accès données vit dans duckdb.PatternsRepo (port.PatternsRepository) :
// ce handler ne connaît ni le SQL ni le moteur de stockage (refactor Axe 1).
//
// Ref : .ai/PLAN_PATTERN_ENGINE_V3.md

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis/patterns"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

const (
	patternDefaultN = 50
	patternMinN     = 10
	patternMaxN     = 200
)

// PatternsRepoResolver retourne le PatternsRepository d'un joueur pour un slug.
// Implémenté côté composition (server.go) en wrappant le ProgressionResolver.
type PatternsRepoResolver func(ctx context.Context, slug string) (port.PatternsRepository, error)

// PatternsHandler gère l'endpoint GET /patterns.
type PatternsHandler struct {
	resolveRepo PatternsRepoResolver
	titleSlug   string
}

// NewPatternsHandler construit le handler.
func NewPatternsHandler(resolveRepo PatternsRepoResolver, titleSlug string) *PatternsHandler {
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	return &PatternsHandler{resolveRepo: resolveRepo, titleSlug: titleSlug}
}

// Mount enregistre la route sur le router chi.
func (h *PatternsHandler) Mount(r chi.Router) {
	r.Get("/patterns", h.GetPatterns)
}

// GetPatterns : GET /patterns?n=50
func (h *PatternsHandler) GetPatterns(w http.ResponseWriter, r *http.Request) {
	playerSlug := chi.URLParam(r, "player_slug")
	repo, err := h.resolveRepo(r.Context(), playerSlug)
	if err != nil {
		slog.WarnContext(r.Context(), "patterns: player not found", "player_slug", playerSlug, "err", err)
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	n := patternDefaultN
	if nStr := r.URL.Query().Get("n"); nStr != "" {
		if v, parseErr := strconv.Atoi(nStr); parseErr == nil && v >= patternMinN && v <= patternMaxN {
			n = v
		} else {
			slog.DebugContext(r.Context(), "patterns: n param ignoré (hors plage ou invalide)", "raw", nStr, "default", patternDefaultN)
		}
	}

	slog.DebugContext(r.Context(), "patterns: chargement des rows", "player_slug", playerSlug, "n", n)

	rows, err := repo.LoadRows(r.Context(), n)
	if err != nil {
		slog.ErrorContext(r.Context(), "patterns: échec chargement rows", "player_slug", playerSlug, "n", n, "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "load_error", err.Error())
		return
	}
	if len(rows) == 0 {
		slog.InfoContext(r.Context(), "patterns: aucun row chargé — rapport vide retourné", "player_slug", playerSlug)
	}

	report := patterns.Analyze(patterns.AnalyzeInput{
		Rows:   rows,
		N:      n,
		Config: patterns.DefaultPatternConfig(),
		Now:    time.Now().UTC(),
	})
	slog.DebugContext(r.Context(), "patterns: analyse terminée",
		"player_slug", playerSlug,
		"rows", len(rows),
		"context_patterns", len(report.ContextPatterns),
		"behavior_patterns", len(report.BehaviorPatterns),
		"levers", len(report.Levers),
	)
	writeJSON(w, http.StatusOK, report)
}
