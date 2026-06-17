package handlers

// patterns.go — endpoint GET /patterns pour le Pattern Engine v3.
//
// Route : GET /api/v1/players/{player_slug}/patterns?n=50
// Query params :
//   - n : nombre de matchs récents à analyser (défaut 50, min 10, max 200)
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (hérite ownership/title + lit {player_slug} parent) et enregistre via huma.Get.
// Le param n reste pris en STRING + parsé maison pour préserver la tolérance
// d'origine (valeur hors plage/invalide → ignorée, défaut servi), au lieu du 422
// de validation Huma qu'un `int` produirait.
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

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis/patterns"
	"levelup/go-api/internal/api/humacore"
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

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *PatternsHandler) Mount(r chi.Router) {
	api := humacore.NewAPI(r)
	huma.Get(api, "/patterns", h.GetPatterns)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// patternsInput : path param parent {player_slug} + ?n= (string, parsé maison).
type patternsInput struct {
	PlayerSlug string `path:"player_slug"`
	N          string `query:"n"`
}

type patternsOutput struct{ Body patterns.PatternReport }

// ─── Endpoint ────────────────────────────────────────────────────────────────

// GetPatterns : GET /patterns?n=50
func (h *PatternsHandler) GetPatterns(ctx context.Context, in *patternsInput) (*patternsOutput, error) {
	repo, err := h.resolveRepo(ctx, in.PlayerSlug)
	if err != nil {
		slog.WarnContext(ctx, "patterns: player not found", "player_slug", in.PlayerSlug, "err", err)
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	n := patternDefaultN
	if in.N != "" {
		if v, parseErr := strconv.Atoi(in.N); parseErr == nil && v >= patternMinN && v <= patternMaxN {
			n = v
		} else {
			slog.DebugContext(ctx, "patterns: n param ignoré (hors plage ou invalide)", "raw", in.N, "default", patternDefaultN)
		}
	}

	slog.DebugContext(ctx, "patterns: chargement des rows", "player_slug", in.PlayerSlug, "n", n)

	rows, err := repo.LoadRows(ctx, n)
	if err != nil {
		slog.ErrorContext(ctx, "patterns: échec chargement rows", "player_slug", in.PlayerSlug, "n", n, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "load_error", err.Error())
	}
	if len(rows) == 0 {
		slog.InfoContext(ctx, "patterns: aucun row chargé — rapport vide retourné", "player_slug", in.PlayerSlug)
	}

	report := patterns.Analyze(patterns.AnalyzeInput{
		Rows:   rows,
		N:      n,
		Config: patterns.DefaultPatternConfig(),
		Now:    time.Now().UTC(),
	})
	slog.DebugContext(ctx, "patterns: analyse terminée",
		"player_slug", in.PlayerSlug,
		"rows", len(rows),
		"context_patterns", len(report.ContextPatterns),
		"behavior_patterns", len(report.BehaviorPatterns),
		"levers", len(report.Levers),
	)
	return &patternsOutput{Body: report}, nil
}
