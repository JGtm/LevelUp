// boot_shell.go — l'intégration de la coquille de bureau côté serveur (lot L21) : le relais des
// notifications de la cloche vers l'événement stdout `notification`. Ne s'installe que sous la capacité
// ShellIntegration (profil desktop) et quand l'écrivain d'événements existe (mode -mode desktop).
package main

import (
	"log/slog"

	"levelup/go-api/internal/notifications/native"
)

// launchShellIntegration branche le relais natif des notifications sur l'écrivain d'événements.
func launchShellIntegration(st *bootState) {
	if !st.cfg.Capabilities.ShellIntegration || st.events == nil {
		return
	}
	installNativeRelay(native.Default(), st.events, st.settingsStore)
	slog.Info("shell: relais des notifications natives branché (réglage native_notifications)")
}

// notificationWriter : ce que le relais appelle pour écrire l'événement (satisfait par *desktopEvents).
type notificationWriter interface {
	Notification(title, body, kind string)
}

// installNativeRelay installe sur relay un sink qui écrit l'événement `notification` de w.
func installNativeRelay(relay *native.Relay, w notificationWriter, store native.SettingsReader) {
	relay.Install(func(n native.Notice) {
		w.Notification(n.Title, n.Body, string(n.Kind))
	}, store)
}
