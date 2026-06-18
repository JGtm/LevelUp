package handlers

import (
	"context"
	"net/http"
	"time"

	"levelup/go-api/internal/api/humacore"
)

// handleGetBackupStatus retourne l'état courant du scheduler de sauvegarde.
// GET /settings/backup/status
func (h *SettingsHandler) handleGetBackupStatus(ctx context.Context, _ *struct{}) (*settingsJSONOutput, error) {
	if h.backupSched == nil {
		return &settingsJSONOutput{Body: map[string]any{"enabled": false, "available": false}}, nil
	}
	return &settingsJSONOutput{Body: h.backupSched.Status()}, nil
}

// handlePostBackupRun déclenche un cycle de sauvegarde immédiat et attend son résultat.
// POST /settings/backup/run
// Synchrone avec timeout de 10 minutes — adapté à un usage admin occasionnel.
func (h *SettingsHandler) handlePostBackupRun(ctx context.Context, _ *struct{}) (*settingsJSONOutput, error) {
	if h.backupSched == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, "backup_scheduler_unavailable", "backup scheduler non initialisé")
	}
	runCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	result, err := h.backupSched.RunOnce(runCtx)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "backup_run_failed", err.Error())
	}
	return &settingsJSONOutput{Body: result}, nil
}
