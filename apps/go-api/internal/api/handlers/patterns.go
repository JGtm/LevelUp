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
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis/patterns"
	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/ctxkeys"
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
	h.enrichContextLabels(ctx, repo, &report)
	slog.DebugContext(ctx, "patterns: analyse terminée",
		"player_slug", in.PlayerSlug,
		"rows", len(rows),
		"context_patterns", len(report.ContextPatterns),
		"behavior_patterns", len(report.BehaviorPatterns),
		"levers", len(report.Levers),
	)
	return &patternsOutput{Body: report}, nil
}

// enrichContextLabels résout les libellés lisibles des patterns by_map (nom de
// carte depuis le référentiel metadata du titre) et les pose sur le rapport. La
// résolution SQL vit dans le repo (port) : ici on ne fait que collecter les ids,
// appeler le port et appliquer le repli localisé. Best-effort : une résolution
// en échec dégrade vers le repli sans casser l'endpoint.
func (h *PatternsHandler) enrichContextLabels(ctx context.Context, repo port.PatternsRepository, report *patterns.PatternReport) {
	mapIDs := distinctMapKeys(report.ContextPatterns)
	if len(mapIDs) == 0 {
		return
	}
	resolved, err := repo.ResolveMapLabels(ctx, mapIDs)
	if err != nil {
		slog.WarnContext(ctx, "patterns: résolution des libellés de carte échouée — repli local", "err", err)
		resolved = nil
	}
	locale := ctxkeys.Locale(ctx)
	for i := range report.ContextPatterns {
		if report.ContextPatterns[i].Type == patterns.ContextByMap {
			report.ContextPatterns[i].Label = mapLabelOrFallback(resolved, report.ContextPatterns[i].Key, locale)
		}
	}
	rewriteMapLeverLabels(report.Levers, resolved, locale)
}

// rewriteMapLeverLabels remplace le GUID de carte présent dans le texte des
// leviers by_map (« Améliore ton win rate en {GUID} ») par le nom résolu, pour
// qu'aucun identifiant technique n'apparaisse dans une phrase servie (A2). Le
// GUID est identifié via SourcePattern (« by_map:{GUID} ») — substitution
// title-agnostic, aucun identifiant de carte n'est reconstruit à la main.
func rewriteMapLeverLabels(levers []patterns.Lever, resolved map[string]string, locale string) {
	const byMapPrefix = string(patterns.ContextByMap) + ":"
	for i := range levers {
		if !strings.HasPrefix(levers[i].SourcePattern, byMapPrefix) {
			continue
		}
		mapID := strings.TrimPrefix(levers[i].SourcePattern, byMapPrefix)
		if mapID == "" {
			continue
		}
		label := mapLabelOrFallback(resolved, mapID, locale)
		levers[i].Label = strings.ReplaceAll(levers[i].Label, mapID, label)
	}
}

// distinctMapKeys collecte les clés (map_id) distinctes des patterns by_map.
func distinctMapKeys(pats []patterns.ContextualPattern) []string {
	seen := make(map[string]struct{}, len(pats))
	var out []string
	for _, p := range pats {
		if p.Type != patterns.ContextByMap {
			continue
		}
		id := strings.TrimSpace(p.Key)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// mapLabelOrFallback retourne le nom résolu, sinon un repli localisé (« Carte
// inconnue » + id court) — jamais le GUID nu.
func mapLabelOrFallback(resolved map[string]string, key, locale string) string {
	if name := strings.TrimSpace(resolved[key]); name != "" {
		return name
	}
	return fallbackMapLabel(key, locale)
}

// fallbackMapLabel construit le repli localisé pour une carte non résolue :
// « Carte inconnue (xxxxxxxx) » / « Unknown map (xxxxxxxx) », l'id tronqué aux
// 8 premiers caractères (jamais le GUID complet). Exception assumée à la règle
// « pas de libellé FR/EN en dur » : ce repli est un état de dégradation (asset
// absent du référentiel), pas un libellé de titre — le back doit garantir un
// label non vide (contrat A1), et le front affiche le label tel quel.
func fallbackMapLabel(id, locale string) string {
	short := id
	if len(short) > 8 {
		short = short[:8]
	}
	if strings.HasPrefix(strings.ToLower(locale), "en") {
		return "Unknown map (" + short + ")"
	}
	return "Carte inconnue (" + short + ")"
}
