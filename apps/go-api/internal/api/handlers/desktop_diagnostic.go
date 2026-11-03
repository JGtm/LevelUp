// Package handlers — desktop_diagnostic.go : POST /api/v1/desktop/diagnostic, le rapport de diagnostic de
// l'application de bureau (lot L22, ADR 0045).
//
// Le rapport est une archive zip écrite sous <racine de données>/diagnostics/ : l'utilisateur la joint à un
// signalement. La route n'est montée que sous la capacité DiagnosticReport et exige le cookie de la coquille
// (garde racine). Rien de secret n'entre dans l'archive (internal/ops/diagnostic).
package handlers

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/ops/diagnostic"
	"levelup/go-api/internal/platform/schedtask"
)

// codeDiagnosticBusy : un rapport est déjà en cours d'écriture.
const codeDiagnosticBusy = "diagnostic_already_running"

// DesktopDiagnosticHandler produit le rapport de diagnostic.
type DesktopDiagnosticHandler struct {
	cfg *config.AppConfig
	mu  sync.Mutex // un rapport à la fois
}

// NewDesktopDiagnosticHandler crée le handler.
func NewDesktopDiagnosticHandler(cfg *config.AppConfig) *DesktopDiagnosticHandler {
	return &DesktopDiagnosticHandler{cfg: cfg}
}

// Mount enregistre la route au point de montage de l'appelant.
func (h *DesktopDiagnosticHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/desktop/diagnostic", h.handle,
		humacore.Op("postDesktopDiagnostic", "Produire le rapport de diagnostic (application de bureau)", "desktop"))
}

type desktopDiagnosticOutput struct {
	Body *domain.DiagnosticReport
}

func (h *DesktopDiagnosticHandler) handle(ctx context.Context, _ *struct{}) (*desktopDiagnosticOutput, error) {
	if !h.mu.TryLock() {
		return nil, humacore.NewError(http.StatusConflict, codeDiagnosticBusy, "diagnostic running")
	}
	defer h.mu.Unlock()
	path, err := diagnostic.Write(ctx, h.input())
	if err != nil {
		return nil, humacore.NewErrorFrom(ctx, http.StatusInternalServerError, "diagnostic_failed", err)
	}
	var size int64
	if st, err := os.Stat(path); err == nil {
		size = st.Size()
	}
	return &desktopDiagnosticOutput{Body: &domain.DiagnosticReport{Path: path, FileName: filepath.Base(path), Bytes: size}}, nil
}

// input rassemble ce que le rapport décrit : le profil, les capacités, les réglages (le fichier), l'état de
// la tâche planifiée et de la mise à jour, et les secrets du processus à effacer partout.
func (h *DesktopDiagnosticHandler) input() diagnostic.Input {
	cfg := h.cfg
	state, code := schedtask.DefaultObserved().Get()
	return diagnostic.Input{
		Paths:        cfg.PathResolver(),
		LogsRoot:     cfg.Roots().Logs,
		AppVersion:   cfg.AppVersion,
		Deployment:   string(cfg.Deployment),
		Capabilities: cfg.Capabilities,
		SettingsFile: cfg.AppSettingsPath,
		Extras: map[string]any{
			"scheduled_tasks": map[string]any{
				"task_name": schedtask.TaskNameFor(cfg.Roots().Data), "state": string(state), "error_code": code,
			},
			"updater": map[string]any{
				"app_version": cfg.AppVersion, "update_offered": cfg.Capabilities.UpdateApp,
				"note": "detailed update state is in logs/desktop.log (the shell journal)",
			},
		},
		Secrets: []string{cfg.DesktopSecret, cfg.SessionSecret},
	}
}
