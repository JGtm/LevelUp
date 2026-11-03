package handlers

// settings_desktop.go — l'intégration au poste de l'application de bureau dans les réglages (lot L21) :
// refus d'un réglage que le profil de déploiement n'offre pas, et état vivant exposé par
// GET /settings/desktop.

import (
	"context"
	"log/slog"
	"net/http"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/platform/gameproc"
	"levelup/go-api/internal/platform/schedtask"
)

// unknownTaskState : l'état de la tâche planifiée tant que la vérification du démarrage n'a rien observé.
const unknownTaskState schedtask.State = "unknown"

// codeDesktopNotOffered : le code du refus d'un réglage d'intégration au poste hors du profil desktop.
const codeDesktopNotOffered = "desktop_setting_not_offered"

// validateDesktop refuse (400) la modification d'un réglage d'intégration au poste dont la capacité
// de déploiement est absente : sur un serveur hébergé, ces clés ne commandent rien et un « enregistré »
// serait un mensonge. Chaque clé suit SA capacité (ADR 0044 D-4), jamais le nom du profil.
func (h *SettingsHandler) validateDesktop(ctx context.Context, req *domain.UpdateSettingsRequest) error {
	caps := h.cfg.Capabilities
	refused := ""
	switch {
	case req.GameProcessWatch != nil && !caps.GameProcessWatch:
		refused = "game_process_watch"
	case (req.NativeNotifications != nil || req.CloseToTray != nil || req.StartWithWindows != nil) && !caps.ShellIntegration:
		refused = "native_notifications|close_to_tray|start_with_windows"
	case req.ScheduledRefresh != nil && !caps.ScheduledRefresh:
		refused = "scheduled_refresh"
	}
	if refused == "" {
		return nil
	}
	slog.WarnContext(ctx, "settings: réglage d'intégration au poste refusé (capacité absente)", "cles", refused)
	return humacore.NewError(http.StatusBadRequest, codeDesktopNotOffered,
		i18n.Msg(i18n.FromContext(ctx), "settings.desktop.not_offered", nil))
}

// desktopStatusOutput : le corps de GET /settings/desktop.
type desktopStatusOutput struct {
	Body *domain.DesktopStatus
}

// handleGetDesktopStatus expose l'état vivant de l'intégration au poste : le jeu tourne-t-il, et la tâche
// planifiée de rafraîchissement est-elle présente.
// GET /settings/desktop
func (h *SettingsHandler) handleGetDesktopStatus(_ context.Context, _ *struct{}) (*desktopStatusOutput, error) {
	out := &domain.DesktopStatus{GameRunning: gameproc.Default().Running()}
	if h.cfg.Capabilities.ScheduledRefresh {
		state, code := schedtask.DefaultObserved().Get()
		if state == "" {
			state = unknownTaskState
		}
		out.ScheduledRefresh = &domain.ScheduledRefreshStatus{State: string(state), ErrorCode: code}
	}
	return &desktopStatusOutput{Body: out}, nil
}
