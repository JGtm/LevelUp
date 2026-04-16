// Package handlers — settings.go : endpoints de configuration (Sprint 16).
//
// GET  /settings              → retourne la configuration courante
// PATCH /settings             → mise à jour partielle (403 demo_mode)
// POST /settings/media/reset-index → réinitialise l'index médias (job asynchrone)
package handlers

import (
	"encoding/json"
	"net/http"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
)

// SettingsHandler gère les endpoints de configuration.
type SettingsHandler struct {
	cfg           *config.AppConfig
	settingsStore *settings_platform.Store
	jobStore      *jobs.Store
}

// NewSettingsHandler crée un SettingsHandler.
func NewSettingsHandler(cfg *config.AppConfig, settingsStore *settings_platform.Store, jobStore *jobs.Store) *SettingsHandler {
	return &SettingsHandler{
		cfg:           cfg,
		settingsStore: settingsStore,
		jobStore:      jobStore,
	}
}

// GetSettings retourne la configuration courante.
// GET /settings
func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	if h.cfg.DemoMode {
		// Mode démo : retourner les valeurs par défaut sans lecture disque
		defaults := settings_platform.ToResponse(settings_platform.Defaults())
		writeJSON(w, http.StatusOK, defaults)
		return
	}

	cfg, err := h.settingsStore.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "settings_load_error", "Impossible de charger la configuration.")
		return
	}
	writeJSON(w, http.StatusOK, settings_platform.ToResponse(cfg))
}

// PatchSettings met à jour partiellement la configuration.
// PATCH /settings — 422 en mode démo.
func (h *SettingsHandler) PatchSettings(w http.ResponseWriter, r *http.Request) {
	if h.cfg.DemoMode {
		writeError(w, http.StatusUnprocessableEntity, "demo_mode_unsupported",
			"La modification des settings n'est pas disponible en mode démo.")
		return
	}

	var req domain.UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Corps de requête JSON invalide.")
		return
	}

	cfg, err := h.settingsStore.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "settings_load_error", "Impossible de charger la configuration.")
		return
	}

	settings_platform.Apply(cfg, &req)

	if err := h.settingsStore.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "settings_save_error", "Impossible de sauvegarder la configuration.")
		return
	}

	writeJSON(w, http.StatusOK, settings_platform.ToResponse(cfg))
}

// PostMediaResetIndex réinitialise l'index des médias (opération destructive).
// POST /settings/media/reset-index — retourne un AsyncJobStatus (202).
// confirm_destructive doit être true.
func (h *SettingsHandler) PostMediaResetIndex(w http.ResponseWriter, r *http.Request) {
	var req domain.MediaResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Corps de requête JSON invalide.")
		return
	}

	if !req.ConfirmDestructive {
		writeError(w, http.StatusBadRequest, "confirmation_required",
			"confirm_destructive doit être true pour autoriser la réinitialisation de l'index.")
		return
	}

	job := h.jobStore.Create(domain.JobTypeReindexMedia, "")

	// Goroutine stub : dans Phase 4, le vrai reset sera implémenté ici.
	go func() {
		step := "Reset index"
		h.jobStore.SetStatus(job.JobID, domain.JobStatusRunning, &step)
		// TODO Sprint 19 : implémenter la réinitialisation réelle de l'index médias.
		done := "Terminé (stub)"
		pct := 100
		h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
			j.Status = domain.JobStatusSucceeded
			j.ProgressPct = &pct
			j.CurrentStep = &done
		})
	}()

	writeJSON(w, http.StatusAccepted, job)
}
