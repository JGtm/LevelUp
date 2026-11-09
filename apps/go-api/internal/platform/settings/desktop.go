package settings

import "levelup/go-api/internal/domain"

// desktop.go — les réglages d'intégration au poste de l'application de bureau (lot L21).
//
// Défauts : `game_process_watch` vaut OUI quand la clé est absente (applyAbsentDefaults : la détection
// seule est sans effet visible) ; les quatre autres valent NON, le zéro du type. Aucune ne se pose sans
// que l'utilisateur l'ait demandé.

// applyDesktop applique les clés d'intégration au poste d'une écriture partielle.
func applyDesktop(cfg *AppSettings, req *domain.DesktopUpdate) {
	if req.GameProcessWatch != nil {
		cfg.GameProcessWatch = *req.GameProcessWatch
	}
	if req.NativeNotifications != nil {
		cfg.NativeNotifications = *req.NativeNotifications
	}
	if req.CloseToTray != nil {
		cfg.CloseToTray = *req.CloseToTray
	}
	if req.StartWithWindows != nil {
		cfg.StartWithWindows = *req.StartWithWindows
	}
	if req.ScheduledRefresh != nil {
		cfg.ScheduledRefresh = *req.ScheduledRefresh
	}
}
