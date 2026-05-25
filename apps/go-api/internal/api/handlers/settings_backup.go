package handlers

import (
	"context"
	"net/http"
	"time"
)

// GetBackupStatus retourne l'état courant du scheduler de sauvegarde.
// GET /settings/backup/status
func (h *SettingsHandler) GetBackupStatus(w http.ResponseWriter, _ *http.Request) {
	if h.backupSched == nil {
		writeJSON(w, http.StatusOK, map[string]any{"enabled": false, "available": false})
		return
	}
	writeJSON(w, http.StatusOK, h.backupSched.Status())
}

// PostBackupRun déclenche un cycle de sauvegarde immédiat et attend son résultat.
// POST /settings/backup/run
// Synchrone avec timeout de 10 minutes — adapté à un usage admin occasionnel.
func (h *SettingsHandler) PostBackupRun(w http.ResponseWriter, r *http.Request) {
	if h.backupSched == nil {
		http.Error(w, "backup scheduler non initialisé", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	result, err := h.backupSched.RunOnce(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}
