// Package handlers — sync_handler.go : lancement de la sync initiale (Sprint 17-18).
//
// POST /sync/initial → crée un job "initial_sync" et retourne 202 immédiatement.
// Règles :
//   - 403 si can_start_initial_sync=false dans app_settings.json
//   - 409 si un job initial_sync actif existe déjà pour ce player_slug
//   - Exclusivité stricte : 1 seule sync active par joueur
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/notifications"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
	go_sync "levelup/go-api/internal/sync"
)

// NotificationsEmitterFactory construit un emitter de notifications pour un slug
// joueur donné. Optionnel : si le SyncHandler n'a pas reçu de factory, aucun
// hook de notification n'est émis (pratique pour les tests et le bootstrap initial).
type NotificationsEmitterFactory func(ctx context.Context, slug string) (notifications.Emitter, error)

// SyncHandler gère les endpoints de synchronisation des données Halo.
type SyncHandler struct {
	cfg           *config.AppConfig
	settingsStore *settings_platform.Store
	jobStore      *jobs.Store
	provider      auth_platform.TokenProvider
	notifierFor   NotificationsEmitterFactory // optionnel : émission match_synced / sync_error
}

// NewSyncHandler crée un SyncHandler.
func NewSyncHandler(
	cfg *config.AppConfig,
	settingsStore *settings_platform.Store,
	jobStore *jobs.Store,
	provider auth_platform.TokenProvider,
) *SyncHandler {
	return &SyncHandler{
		cfg:           cfg,
		settingsStore: settingsStore,
		jobStore:      jobStore,
		provider:      provider,
	}
}

// WithNotificationsEmitterFactory branche la factory d'émetteurs de notifications.
// À appeler depuis server.go après création du ServiceRegistry.
//
// L'émission est best-effort : toute erreur est loguée et ne propage pas. Le
// hook est invoqué post-RunDelta (succès → match_synced, erreur → sync_error).
func (h *SyncHandler) WithNotificationsEmitterFactory(f NotificationsEmitterFactory) *SyncHandler {
	h.notifierFor = f
	return h
}

// PrestigeHook est la signature du hook post-sync injecté pour le module Prestige.
// Stub minimal — la logique métier vit dans internal/api/prestige_setup.go.
type PrestigeHook func(ctx context.Context, playerSlug, titleSlug string)

// WithPrestigeHook branche un hook Prestige post-sync. Stub temporaire pour
// satisfaire la référence de server.go en attendant la finalisation côté
// agent Prestige. No-op si pas de hook configuré.
func (h *SyncHandler) WithPrestigeHook(_ PrestigeHook) *SyncHandler {
	// TODO(prestige-agent): câbler l'invocation post-RunDelta.
	return h
}

// emitMatchSynced émet une notification agrégée match_synced si > 0 matchs insérés.
func (h *SyncHandler) emitMatchSynced(ctx context.Context, slug string, inserted int) {
	if h.notifierFor == nil || inserted <= 0 {
		return
	}
	em, err := h.notifierFor(ctx, slug)
	if err != nil || em == nil {
		slog.WarnContext(ctx, "notifications: emitter factory failed", "slug", slug, "err", err)
		return
	}
	if err := em.Emit(ctx, notifications.EmitInput{
		Category: notifications.CategoryMatchSynced,
		Severity: notifications.SeveritySuccess,
		TitleKey: "notif.match_synced.title",
		BodyKey:  "notif.match_synced.body",
		Params:   map[string]any{"count": inserted},
		Source:   "sync_handler",
	}); err != nil {
		slog.WarnContext(ctx, "notifications: match_synced emit", "err", err)
	}
}

// emitSyncError émet une notification sync_error sur échec d'une sync.
func (h *SyncHandler) emitSyncError(ctx context.Context, slug, jobID, message string) {
	if h.notifierFor == nil {
		return
	}
	em, err := h.notifierFor(ctx, slug)
	if err != nil || em == nil {
		return
	}
	if err := em.Emit(ctx, notifications.EmitInput{
		Category:    notifications.CategorySyncError,
		Severity:    notifications.SeverityError,
		TitleKey:    "notif.sync_error.title",
		BodyKey:     "notif.sync_error.body",
		Params:      map[string]any{"message": truncate(message, 200), "job_id": jobID},
		TargetRoute: fmt.Sprintf("/players/%s/sync", slug),
		Source:      "sync_handler",
	}); err != nil {
		slog.WarnContext(ctx, "notifications: sync_error emit", "err", err)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// StartInitialSync lance la sync initiale pour un joueur.
// POST /sync/initial -> 202 AsyncJobStatus.
func (h *SyncHandler) StartInitialSync(w http.ResponseWriter, r *http.Request) {
	appCfg, err := h.settingsStore.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "settings_load_error", "Impossible de charger la configuration.")
		return
	}
	if !appCfg.CanStartInitialSync {
		writeError(w, http.StatusForbidden, "initial_sync_disabled",
			"Le lancement d'une sync initiale est désactivé sur cette instance.")
		return
	}

	var req domain.InitialSyncStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Corps de requete JSON invalide.")
		return
	}

	if req.PlayerSlug == "" || len(req.PlayerSlug) > 50 {
		writeError(w, http.StatusBadRequest, "invalid_player_slug", "player_slug vide ou trop long.")
		return
	}
	if req.MaxMatches == 0 {
		req.MaxMatches = 200
	}
	if req.MaxMatches < 1 || req.MaxMatches > 2000 {
		writeError(w, http.StatusBadRequest, "invalid_max_matches", "max_matches doit etre entre 1 et 2000.")
		return
	}

	if active := h.jobStore.FindActiveInitialSync(req.PlayerSlug); active != nil {
		writeError(w, http.StatusConflict, "sync_already_active",
			"Une sync initiale est deja en cours pour ce joueur.")
		return
	}

	sess := middleware.GetSession(r.Context())
	if sess == nil || sess.HaloTokens == nil {
		writeError(w, http.StatusUnauthorized, "auth_required",
			"Tokens Halo absents.")
		return
	}
	tokens := sess.HaloTokens

	players, err := h.cfg.LoadPlayers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "profiles_load_error", "Impossible de charger db_profiles.json.")
		return
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
		writeError(w, http.StatusNotFound, "player_not_found",
			fmt.Sprintf("Joueur %q introuvable dans db_profiles.json.", req.PlayerSlug))
		return
	}

	job := h.jobStore.Create(domain.JobTypeInitialSync, req.PlayerSlug)
	// Snapshot avant le go func() : la goroutine modifie in-place le job dans le store.
	jobSnapshot := *job

	go func() {
		bgCtx := context.Background()
		engine := go_sync.NewSyncEngine(h.cfg.RepoRoot, gamertag, xuid, tokens, h.provider)
		opts := domain.DefaultSyncOptions()
		opts.MaxMatches = req.MaxMatches

		result, err := engine.RunDelta(bgCtx, opts)
		if err != nil {
			errMsg := err.Error()
			h.jobStore.SetStatus(job.JobID, domain.JobStatusFailed, &errMsg)
			h.emitSyncError(bgCtx, req.PlayerSlug, job.JobID, errMsg)
			return
		}
		summary := fmt.Sprintf("inserted=%d skipped=%d medals=%d duration=%.1fs status=%s",
			result.MatchesInserted, result.MatchesSkipped, result.MedalsInserted,
			result.DurationSeconds, result.Status())
		h.jobStore.SetStatus(job.JobID, domain.JobStatusSucceeded, &summary)
		h.emitMatchSynced(bgCtx, req.PlayerSlug, result.MatchesInserted)
	}()

	writeJSON(w, http.StatusAccepted, &jobSnapshot)
}

// StartDeltaSync lance une synchronisation delta pour un joueur donné.
// POST /api/v1/players/{player_slug}/sync → 202 AsyncJobStatus.
// Contrairement à StartInitialSync, cette route n'est pas protégée par can_start_initial_sync.
func (h *SyncHandler) StartDeltaSync(w http.ResponseWriter, r *http.Request) {
	playerSlug := r.PathValue("player_slug")
	if playerSlug == "" {
		writeError(w, http.StatusBadRequest, "invalid_player_slug", "player_slug manquant.")
		return
	}

	if active := h.jobStore.FindActiveInitialSync(playerSlug); active != nil {
		writeError(w, http.StatusConflict, "sync_already_active",
			"Une synchronisation est déjà en cours pour ce joueur.")
		return
	}

	sess := middleware.GetSession(r.Context())
	if sess == nil || sess.HaloTokens == nil {
		writeError(w, http.StatusUnauthorized, "auth_required",
			"Tokens Halo absents.")
		return
	}
	tokens := sess.HaloTokens

	players, err := h.cfg.LoadPlayers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "profiles_load_error", "Impossible de charger db_profiles.json.")
		return
	}
	var gamertag, xuid string
	for _, p := range players {
		if p.PlayerSlug == playerSlug {
			gamertag = p.Gamertag
			xuid = p.XUID
			break
		}
	}
	if gamertag == "" {
		writeError(w, http.StatusNotFound, "player_not_found",
			fmt.Sprintf("Joueur %q introuvable dans db_profiles.json.", playerSlug))
		return
	}

	job := h.jobStore.Create(domain.JobTypeInitialSync, playerSlug)
	// Snapshot avant le go func() : la goroutine modifie in-place le job dans le store.
	jobSnapshot2 := *job

	go func() {
		bgCtx := context.Background()
		engine := go_sync.NewSyncEngine(h.cfg.RepoRoot, gamertag, xuid, tokens, h.provider)
		opts := domain.DefaultSyncOptions()

		result, err := engine.RunDelta(bgCtx, opts)
		if err != nil {
			errMsg := err.Error()
			h.jobStore.SetStatus(job.JobID, domain.JobStatusFailed, &errMsg)
			h.emitSyncError(bgCtx, playerSlug, job.JobID, errMsg)
			return
		}
		summary := fmt.Sprintf("inserted=%d skipped=%d medals=%d duration=%.1fs status=%s",
			result.MatchesInserted, result.MatchesSkipped, result.MedalsInserted,
			result.DurationSeconds, result.Status())
		h.jobStore.SetStatus(job.JobID, domain.JobStatusSucceeded, &summary)
		h.emitMatchSynced(bgCtx, playerSlug, result.MatchesInserted)
	}()

	writeJSON(w, http.StatusAccepted, &jobSnapshot2)
}

// StartSyncAll lance une synchronisation delta pour tous les joueurs configurés.
// POST /api/v1/sync/all → 202 AsyncJobStatus.
func (h *SyncHandler) StartSyncAll(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil || sess.HaloTokens == nil {
		writeError(w, http.StatusUnauthorized, "auth_required", "Tokens Halo absents.")
		return
	}
	tokens := sess.HaloTokens

	players, err := h.cfg.LoadPlayers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "profiles_load_error", "Impossible de charger db_profiles.json.")
		return
	}
	if len(players) == 0 {
		writeError(w, http.StatusNotFound, "no_players", "Aucun joueur configuré dans db_profiles.json.")
		return
	}

	job := h.jobStore.Create(domain.JobTypeDeltaSyncAll, "all")
	// Snapshot avant le go func() : la goroutine modifie in-place le job dans le store.
	jobSnapshot3 := *job

	go func() {
		total := len(players)
		var succeeded, failed int
		for i, p := range players {
			step := fmt.Sprintf("%s (%d/%d)", p.Gamertag, i+1, total)
			h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
				j.CurrentStep = &step
				pct := (i * 100) / total
				j.ProgressPct = &pct
			})

			engine := go_sync.NewSyncEngine(h.cfg.RepoRoot, p.Gamertag, p.XUID, tokens, h.provider)
			opts := domain.DefaultSyncOptions()
			if _, err := engine.RunDelta(context.Background(), opts); err != nil {
				failed++
			} else {
				succeeded++
			}
		}

		summary := fmt.Sprintf("players=%d succeeded=%d failed=%d", total, succeeded, failed)
		if failed > 0 {
			h.jobStore.SetStatus(job.JobID, domain.JobStatusFailed, &summary)
		} else {
			h.jobStore.SetStatus(job.JobID, domain.JobStatusSucceeded, &summary)
		}
	}()

	writeJSON(w, http.StatusAccepted, &jobSnapshot3)
}
