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
	"net/http"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
	go_sync "levelup/go-api/internal/sync"
)

// SyncHandler gère les endpoints de synchronisation des données Halo.
type SyncHandler struct {
	cfg           *config.AppConfig
	settingsStore *settings_platform.Store
	jobStore      *jobs.Store
}

// NewSyncHandler crée un SyncHandler.
func NewSyncHandler(cfg *config.AppConfig, settingsStore *settings_platform.Store, jobStore *jobs.Store) *SyncHandler {
	return &SyncHandler{
		cfg:           cfg,
		settingsStore: settingsStore,
		jobStore:      jobStore,
	}
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

	go func() {
		engine := go_sync.NewSyncEngine(h.cfg.RepoRoot, gamertag, xuid, tokens)
		opts := domain.DefaultSyncOptions()
		opts.MaxMatches = req.MaxMatches

		result, err := engine.RunDelta(context.Background(), opts)
		if err != nil {
			errMsg := err.Error()
			h.jobStore.SetStatus(job.JobID, domain.JobStatusFailed, &errMsg)
			return
		}
		summary := fmt.Sprintf("inserted=%d skipped=%d medals=%d duration=%.1fs status=%s",
			result.MatchesInserted, result.MatchesSkipped, result.MedalsInserted,
			result.DurationSeconds, result.Status())
		h.jobStore.SetStatus(job.JobID, domain.JobStatusSucceeded, &summary)
	}()

	writeJSON(w, http.StatusAccepted, job)
}
