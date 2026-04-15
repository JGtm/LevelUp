// Package handlers — sync_handler.go : lancement de la sync initiale (Sprint 17).
//
// POST /sync/initial → crée un job "initial_sync" et retourne 202 immédiatement.
// Règles :
//   - 403 si can_start_initial_sync=false dans app_settings.json
//   - 409 si un job initial_sync actif existe déjà pour ce player_slug
//   - Exclusivité stricte : 1 seule sync active par joueur
package handlers

import (
	"encoding/json"
	"net/http"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
)

// SyncHandler gère les endpoints de synchronisation des données Halo.
type SyncHandler struct {
	cfg          *config.AppConfig
	settingsStore *settings_platform.Store
	jobStore     *jobs.Store
}

// NewSyncHandler crée un SyncHandler.
func NewSyncHandler(cfg *config.AppConfig, settingsStore *settings_platform.Store, jobStore *jobs.Store) *SyncHandler {
	return &SyncHandler{
		cfg:          cfg,
		settingsStore: settingsStore,
		jobStore:     jobStore,
	}
}

// StartInitialSync lance la sync initiale pour un joueur.
// POST /sync/initial → 202 AsyncJobStatus.
func (h *SyncHandler) StartInitialSync(w http.ResponseWriter, r *http.Request) {
	// Guard : can_start_initial_sync
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
		writeError(w, http.StatusBadRequest, "invalid_body", "Corps de requête JSON invalide.")
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
		writeError(w, http.StatusBadRequest, "invalid_max_matches", "max_matches doit être entre 1 et 2000.")
		return
	}

	// Guard : single-flight par joueur
	if active := h.jobStore.FindActiveInitialSync(req.PlayerSlug); active != nil {
		writeError(w, http.StatusConflict, "sync_already_active",
			"Une sync initiale est déjà en cours pour ce joueur.")
		return
	}

	// Créer le job
	job := h.jobStore.Create(domain.JobTypeInitialSync, req.PlayerSlug)

	// Stub goroutine — Phase 4 implémentera le vrai moteur sync (Sprint 18).
	// Pour l'instant on met juste le job en file d'attente.
	go func() {
		// Note : le vrai moteur sync sera branché ici en Sprint 18.
		// Pour l'instant : marquer queued → succeeded après un stub minimal.
		_ = req.MaxMatches // utilisé en Phase 4
	}()

	writeJSON(w, http.StatusAccepted, job)
}
