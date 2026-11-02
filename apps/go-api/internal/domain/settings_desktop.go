package domain

// settings_desktop.go — les réglages d'INTÉGRATION AU POSTE de l'application de bureau (lot L21,
// ADR 0045). Les trois types de lecture/écriture des réglages les EMBARQUENT (clés aplaties dans le
// JSON) : la forme de ces clés se déclare ici, une fois. Chacune n'a d'effet que sous la capacité de
// déploiement correspondante (GameProcessWatch, ShellIntegration, ScheduledRefresh) ; hors desktop
// le serveur refuse de les modifier.

// DesktopSettings : ce que app_settings.json et GET /settings portent de l'intégration au poste.
type DesktopSettings struct {
	// GameProcessWatch : sonder les processus pour savoir si le jeu tourne. Défaut OUI (clé absente =
	// vrai) : la détection seule n'a aucun effet visible ; elle suspend la cuisson des rejeux et les
	// tâches lourdes pendant la partie et déclenche une synchronisation à sa fermeture.
	GameProcessWatch bool `json:"game_process_watch" doc:"Détecter le jeu en cours : pendant une partie, cuisson des rejeux et tâches lourdes en attente ; synchronisation à la fermeture du jeu. Application de bureau."`
	// NativeNotifications : relayer les notifications de la cloche en notifications Windows quand la
	// fenêtre n'est pas au premier plan. Défaut NON.
	NativeNotifications bool `json:"native_notifications" doc:"Afficher les notifications de la cloche comme notifications Windows quand la fenêtre n'est pas au premier plan. Application de bureau seulement."`
	// CloseToTray : la croix réduit l'application dans la zone de notification au lieu de la quitter.
	// Défaut NON (la croix quitte).
	CloseToTray bool `json:"close_to_tray" doc:"La croix de la fenêtre réduit l'application dans la zone de notification au lieu de la quitter ; le serveur continue. Application de bureau seulement."`
	// StartWithWindows : lancer l'application, réduite dans la zone de notification, à l'ouverture de la
	// session Windows. Défaut NON.
	StartWithWindows bool `json:"start_with_windows" doc:"Lancer l'application, réduite dans la zone de notification, à l'ouverture de la session Windows. Application de bureau seulement."`
	// ScheduledRefresh : poser une tâche planifiée Windows qui rafraîchit les jetons quand l'application
	// est fermée. Défaut NON.
	ScheduledRefresh bool `json:"scheduled_refresh" doc:"Poser une tâche planifiée Windows qui rafraîchit les jetons quand l'application est fermée. Application de bureau seulement."`
}

// DesktopUpdate : les cinq clés, en écriture partielle (PATCH /settings).
type DesktopUpdate struct {
	GameProcessWatch    *bool `json:"game_process_watch,omitempty"`
	NativeNotifications *bool `json:"native_notifications,omitempty"`
	CloseToTray         *bool `json:"close_to_tray,omitempty"`
	StartWithWindows    *bool `json:"start_with_windows,omitempty"`
	ScheduledRefresh    *bool `json:"scheduled_refresh,omitempty"`
}

// DesktopStatus est la réponse de GET /settings/desktop : l'état vivant de l'intégration au poste (ce
// que les réglages ne disent pas).
type DesktopStatus struct {
	// GameRunning : le jeu tourne en ce moment (mode jeu actif : cuisson des rejeux et tâches lourdes
	// suspendues).
	GameRunning bool `json:"game_running" doc:"Le jeu tourne en ce moment : la cuisson des rejeux et les tâches lourdes sont suspendues."`
}
