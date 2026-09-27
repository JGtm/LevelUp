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
//
// REFUSÉE EN DÉMO (403, lot B5.6 du backlog 2026-09-26) : RequireAdmin est transparent en
// démo, donc n'importe quel visiteur atteint cette route, et le scheduler est construit sur
// le PathResolver du dépôt — une démo lancée sur un poste de dev sauvegardait les bases réelles.
func (h *SettingsHandler) handlePostBackupRun(ctx context.Context, _ *struct{}) (*settingsJSONOutput, error) {
	if err := refuseInDemo(h.cfg != nil && h.cfg.DemoMode, "backup"); err != nil {
		return nil, err
	}
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
