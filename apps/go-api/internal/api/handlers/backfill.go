// Package handlers — backfill.go : lancement du pipeline de backfill (Sprint 51-B3).
//
// POST /backfill/start → crée un job "backfill" et retourne 202 immédiatement.
// Règles :
//   - 400 si player_slug absent ou scope entièrement vide
//   - 401 si tokens Halo absents (requis pour weapon kills)
//   - 404 si le joueur est introuvable dans db_profiles.json
//   - 409 si un job backfill actif existe déjà pour ce player_slug
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le routeur chi
// (point de montage /backfill/start, middleware RequireAuth/RequireAdmin hérités)
// et enregistre le POST via huma.Post. Le corps est lu via RawBody (pas de Body
// typé) pour reproduire EXACTEMENT le contrat de décodage d'origine : un JSON
// invalide (corps absent inclus) renvoie 400 {invalid_body} et non le 422 de
// validation Huma. Logique métier inchangée, seul le wrapping HTTP change.
//
// K1f (2026-07-06) : l'orchestration multi-phase (détection + N backfills) est
// extraite dans service.BackfillOrchestrator ; ce handler ne garde que validation,
// wiring du SyncEngine, création du job et 202.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/platform/jobs"
	"levelup/go-api/internal/service"
	go_sync "levelup/go-api/internal/sync"
)

// BackfillHandler gère le pipeline de backfill des données manquantes.
type BackfillHandler struct {
	cfg      *config.AppConfig
	jobStore *jobs.Store
}

// NewBackfillHandler crée un BackfillHandler.
func NewBackfillHandler(cfg *config.AppConfig, jobStore *jobs.Store) *BackfillHandler {
	return &BackfillHandler{cfg: cfg, jobStore: jobStore}
}

// Mount enregistre la route via Huma sur le routeur chi (point de montage
// /backfill/start, middleware RequireAuth/RequireAdmin hérités du groupe).
func (h *BackfillHandler) Mount(r chi.Router) {
	api := humacore.NewAPI(r)
	huma.Post(api, "/backfill/start", h.handleStartBackfill)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// backfillStartInput : corps brut décodé maison. RawBody (pas Body typé) →
// préserve le contrat 400 {invalid_body} sur JSON invalide ou corps absent (un
// Body typé renverrait le 422 de validation Huma).
type backfillStartInput struct {
	RawBody []byte
}

// backfillStartOutput : 202 Accepted, corps = snapshot du job créé.
type backfillStartOutput struct {
	Status int
	Body   *domain.AsyncJobStatus
}

// ─── Endpoint ────────────────────────────────────────────────────────────────

// handleStartBackfill déclenche le pipeline backfill pour un joueur.
// POST /backfill/start → 202 AsyncJobStatus. Validation + wiring uniquement :
// l'orchestration vit dans service.BackfillOrchestrator (lancé en goroutine).
func (h *BackfillHandler) handleStartBackfill(ctx context.Context, in *backfillStartInput) (*backfillStartOutput, error) {
	var req domain.BackfillStartRequest
	if err := json.Unmarshal(in.RawBody, &req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "Corps de requête JSON invalide.")
	}

	if req.PlayerSlug == "" || len(req.PlayerSlug) > 50 {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_player_slug", "player_slug vide ou trop long.")
	}

	// Chercher le joueur dans db_profiles.json.
	players, err := h.cfg.LoadPlayers()
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "profiles_load_error",
			"Impossible de charger db_profiles.json.")
	}
	var gamertag, xuid string
	for _, p := range players {
		if p.PlayerSlug == req.PlayerSlug {
			gamertag = p.Gamertag
			xuid = p.XUID
			break
		}
	}
	if gamertag == "" {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found",
			fmt.Sprintf("Joueur %q introuvable dans db_profiles.json.", req.PlayerSlug))
	}

	// 409 si un job backfill est déjà actif pour ce joueur.
	if active := h.jobStore.FindActiveJob(domain.JobTypeBackfill, req.PlayerSlug); active != nil {
		return nil, humacore.NewError(http.StatusConflict, "backfill_already_active",
			"Un backfill est déjà en cours pour ce joueur.")
	}

	// Sprint B1 commit 17 : event_id pour tracer le backfill HTTP-triggered.
	// Log immédiat sans muter le contexte requête (data race avec chi middleware).
	_, evID := logging.WithEvent(ctx, "http.backfill:"+req.PlayerSlug)
	slog.InfoContext(ctx, "backfill_handler: StartBackfill démarré",
		"player_slug", req.PlayerSlug, "gamertag", gamertag, "event", evID)

	// Les weapons backfill nécessitent les tokens Halo.
	sess := middleware.GetSession(ctx)
	var tokens *domain.HaloTokens
	if sess != nil {
		tokens = sess.HaloTokens
	}

	// Construire le SyncScope depuis la requête.
	scope := buildSyncScope(req)

	// Wiring du SyncEngine (DI). Sprint B1 commit 11b : aligner le backfill
	// HTTP-triggered sur auto_sync via Provider (le double-dblease dans
	// acquireSharedWriter rend ce wire sûr) — sans Provider, RunBackfill*
	// utiliseraient OpenSharedDB direct (mode legacy) en parallèle des syncs auto
	// → conflit "different configuration" pour les readers HTTP.
	engine := go_sync.NewSyncEngine(h.cfg.RepoRoot, gamertag, xuid, tokens, nil)
	if h.cfg.SharedProvider != nil {
		engine = engine.WithSharedProvider(h.cfg.SharedProvider)
	}

	job := h.jobStore.Create(domain.JobTypeBackfill, req.PlayerSlug)
	// Snapshot avant le lancement : la goroutine modifie in-place le job dans le store.
	jobSnapshot := *job

	orch := service.NewBackfillOrchestrator(engine, h.jobStore, scope, tokens, req.DryRun)
	go orch.Run(job.JobID)

	return &backfillStartOutput{Status: http.StatusAccepted, Body: &jobSnapshot}, nil
}

// buildSyncScope construit un SyncScope depuis le payload de requête.
func buildSyncScope(req domain.BackfillStartRequest) *go_sync.SyncScope {
	scope := &go_sync.SyncScope{
		DetectionMode: "or",
		MaxMatches:    req.MaxMatches,
	}

	noExplicitScope := !req.Medals && !req.Events && !req.Skill &&
		!req.PersonalScores && !req.PerformanceScores &&
		!req.Aliases && !req.Weapons && !req.LUSR && !req.CSR && !req.EngagementScores &&
		!req.EngagementCoefficients && !req.ComebackBadges && !req.Citations
	if req.AllData || noExplicitScope {
		// Aucun scope explicite → activer tout
		scope.AllData = true
		scope.Medals = true
		scope.Events = true
		scope.Skill = true
		scope.PersonalScores = true
		scope.PerformanceScores = true
		scope.Aliases = true
		scope.Weapons = true
		scope.LUSR = true
		scope.CSR = true
		scope.EngagementScores = true
		scope.ComebackBadges = true
		scope.Citations = true
		// EngagementCoefficients implicite : les coefs sont recomputes en
		// queue de RunBackfillEngagementScores, pas besoin d'activer le flag.
	} else {
		scope.Medals = req.Medals
		scope.Events = req.Events
		scope.Skill = req.Skill
		scope.PersonalScores = req.PersonalScores
		scope.PerformanceScores = req.PerformanceScores
		scope.Aliases = req.Aliases
		scope.Weapons = req.Weapons
		scope.LUSR = req.LUSR
		scope.CSR = req.CSR
		scope.EngagementScores = req.EngagementScores
		scope.EngagementCoefficients = req.EngagementCoefficients
		scope.ComebackBadges = req.ComebackBadges
		scope.Citations = req.Citations
	}

	if req.ForceRescan {
		scope.ForceMedals = req.Medals || req.AllData
		scope.ForceEvents = req.Events || req.AllData
		scope.ForceSkill = req.Skill || req.AllData
		scope.ForceWeapons = req.Weapons || req.AllData
		scope.ForcePersonalScores = req.PersonalScores || req.AllData
		scope.ForcePerformanceScores = req.PerformanceScores || req.AllData
		scope.ForceAliases = req.Aliases || req.AllData
		scope.ForceLUSR = req.LUSR || req.AllData
		scope.ForceCSR = req.CSR || req.AllData
		scope.ForceEngagementScores = req.EngagementScores || req.AllData
		scope.ForceEngagementCoefficients = req.EngagementCoefficients
		scope.ForceComebackBadges = req.ComebackBadges || req.AllData
		scope.ForceCitations = req.Citations || req.AllData
	}

	scope.Resolve()
	return scope
}
