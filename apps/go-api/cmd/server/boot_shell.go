// boot_shell.go — l'intégration de la coquille de bureau côté serveur (lot L21) : le relais des
// notifications de la cloche vers l'événement stdout `notification`, les réglages que la coquille applique
// (événement `shell_settings`) et la commande `sync` de sa stdin. Ne s'installe que sous la capacité
// ShellIntegration (profil desktop) et quand l'écrivain d'événements existe (mode -mode desktop).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/notifications/native"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/scheduler"
)

// shellSettingsEvery : l'intervalle auquel le serveur relit les réglages que la coquille applique. Un
// réglage change par un PATCH /settings ou par l'édition du fichier ; la coquille le voit dans ce délai.
const shellSettingsEvery = 5 * time.Second

// launchShellIntegration branche, sous la capacité ShellIntegration : le relais natif des notifications,
// la boucle des réglages de la coquille et la commande `sync` de sa stdin.
func launchShellIntegration(st *bootState) {
	if !st.cfg.Capabilities.ShellIntegration || st.events == nil {
		return
	}
	installNativeRelay(native.Default(), st.events, st.settingsStore)
	native.Default().InstallOwnerFilter(localPlayerFilter(auth.NewMultiUserTokenStore(st.cfg.WatcherTokensDir())))
	if st.autoScheduler != nil {
		st.bootSrv.commands.Handle(stdinCommandSync, newManualSync(st, st.autoScheduler))
	}
	ctx := st.schedulerCtx
	st.schedulerWG.Add(1)
	go func() {
		defer st.schedulerWG.Done()
		runShellSettingsLoop(ctx, st.settingsStore, st.events, nil, shellSettingsEvery)
	}()
	slog.InfoContext(ctx, "shell: intégration branchée (notifications natives, réglages de la coquille, commande sync)")
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

// tokenLoader : la lecture des jetons d'un joueur (satisfaite par *auth.MultiUserTokenStore).
type tokenLoader interface {
	Load(xuid string) (*auth.UserTokens, error)
}

// localPlayerFilter rend le filtre du joueur local du relais natif : un joueur est local quand le magasin
// de jetons porte un jeton de rafraîchissement pour son xuid, c'est-à-dire quand l'utilisateur de la machine
// s'est connecté avec ce compte. Un profil suivi sans jeton (un ami, un adversaire) ne l'est pas.
func localPlayerFilter(store tokenLoader) native.OwnerFilter {
	return func(xuid string) bool {
		tokens, err := store.Load(xuid)
		if err != nil {
			if !errors.Is(err, auth.ErrUserTokensNotFound) {
				slog.Warn("shell: jetons du joueur illisibles, notification native non relayée", "xuid", xuid, "err", err)
			}
			return false
		}
		return tokens != nil && tokens.OAuthRefreshToken != ""
	}
}

// shellSettings : ce que la coquille lit des réglages (événement `shell_settings`) : les trois réglages
// d'intégration au poste, et la langue de ses libellés (menu de la zone de notification), qui est celle du joueur.
type shellSettings struct {
	NativeNotifications bool   `json:"native_notifications"`
	CloseToTray         bool   `json:"close_to_tray"`
	StartWithWindows    bool   `json:"start_with_windows"`
	Locale              string `json:"locale"`
}

// shellSettingsOf extrait de cfg ce que la coquille applique.
func shellSettingsOf(cfg *settings.AppSettings) shellSettings {
	return shellSettings{
		NativeNotifications: cfg.NativeNotifications,
		CloseToTray:         cfg.CloseToTray,
		StartWithWindows:    cfg.StartWithWindows,
		Locale:              string(i18n.Parse(cfg.Lang)),
	}
}

// shellSettingsWriter : ce que la boucle appelle pour écrire l'événement (satisfait par *desktopEvents).
type shellSettingsWriter interface {
	ShellSettings(s shellSettings)
}

// runShellSettingsLoop écrit les réglages de la coquille une première fois, puis à chaque changement lu
// aux tics (ticks nil : un ticker de every, arrêté avec la boucle). Des réglages illisibles ne changent
// rien et se journalisent une fois par panne. Bloquant : à lancer dans une goroutine suivie.
func runShellSettingsLoop(ctx context.Context, store settingsLoader, w shellSettingsWriter, ticks <-chan time.Time, every time.Duration) {
	if ticks == nil {
		ticker := time.NewTicker(every)
		defer ticker.Stop()
		ticks = ticker.C
	}
	var last *shellSettings
	failing := false
	poll := func() {
		cfg, err := store.Load()
		if err != nil {
			if !failing {
				slog.WarnContext(ctx, "shell: réglages illisibles, la coquille garde les précédents", "err", err)
			}
			failing = true
			return
		}
		failing = false
		current := shellSettingsOf(cfg)
		if last != nil && *last == current {
			return
		}
		last = &current
		w.ShellSettings(current)
	}
	poll()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			poll()
		}
	}
}

// manualSyncer : le déclencheur manuel de l'ordonnanceur d'auto-sync (*scheduler.AutoSyncScheduler).
type manualSyncer interface {
	RunOnceTrigger(ctx context.Context, trigger string) *scheduler.RunOnceResult
}

// newManualSync rend la commande `sync` : un cycle immédiat par le déclencheur manuel, à part, jamais deux
// à la fois (une commande reçue pendant un cycle est ignorée et journalisée). Le cycle compte dans le
// groupe d'attente des tâches de fond : l'arrêt l'attend avant la fermeture des bases.
func newManualSync(st *bootState, sched manualSyncer) func() {
	var busy atomic.Bool
	return func() {
		ctx := st.schedulerCtx
		if ctx.Err() != nil {
			return // arrêt en cours : plus de cycle
		}
		if !busy.CompareAndSwap(false, true) {
			slog.InfoContext(ctx, "shell: synchronisation demandée alors qu'un cycle manuel tourne déjà, ignorée")
			return
		}
		st.schedulerWG.Add(1)
		go func() {
			defer st.schedulerWG.Done()
			defer busy.Store(false)
			defer func() {
				if rec := recover(); rec != nil {
					slog.ErrorContext(ctx, "shell: panic pendant la synchronisation manuelle", "recover", fmt.Sprint(rec))
				}
			}()
			res := sched.RunOnceTrigger(ctx, "manual")
			slog.InfoContext(ctx, "shell: synchronisation manuelle terminée", "total", res.Total,
				"synced", res.Synced, "failed", res.Failed, "duration", res.Duration.Round(time.Millisecond))
		}()
	}
}
