// boot_gameproc.go — la détection du jeu en cours du serveur desktop (lot L21) : sonde les processus
// du poste (internal/platform/gameproc), tient le mode jeu à jour (cuisson des rejeux et tâches lourdes
// suspendues pendant la partie) et demande une synchronisation à la fermeture du jeu.
package main

import (
	"log/slog"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/gameproc"
	"levelup/go-api/internal/platform/settings"
)

// settingsLoader : la lecture des réglages vivants (satisfaite par *settings.Store).
type settingsLoader interface {
	Load() (*settings.AppSettings, error)
}

// syncRequester : la demande de synchronisation à l'ordonnanceur d'auto-sync, la même que celle de la
// séquence de réveil (*scheduler.AutoSyncScheduler.RequestCatchUp).
type syncRequester interface {
	RequestCatchUp(reason string) bool
}

// launchGameWatch lance la détection du jeu si le profil l'offre (capacité GameProcessWatch, vraie en
// desktop seulement). Elle tourne sous le contexte des tâches de fond et dans leur groupe d'attente :
// l'arrêt l'annule puis l'attend avant la fermeture des bases.
func launchGameWatch(st *bootState) {
	if !st.cfg.Capabilities.GameProcessWatch {
		return
	}
	var sync syncRequester
	if st.autoScheduler != nil { // un *AutoSyncScheduler nil ne doit pas devenir une interface non nil
		sync = st.autoScheduler
	}
	w := newGameWatcher(st.settingsStore, gameproc.NewSystemLister(), gameproc.Default(), sync)
	ctx := st.schedulerCtx
	st.schedulerWG.Add(1)
	go func() {
		defer st.schedulerWG.Done()
		w.Run(ctx)
	}()
	slog.InfoContext(ctx, "game_mode: détection du jeu armée", "intervalle", gameproc.DefaultInterval,
		"executables", title.DefaultRegistry().GameProcessNames())
}

// newGameWatcher assemble le sondeur de production : les noms d'exécutables viennent des descripteurs de
// titres (relus à chaque sondage), le réglage `game_process_watch` est relu à chaque sondage (un
// changement prend effet sans redémarrage), la fermeture du jeu demande un rattrapage de synchronisation.
func newGameWatcher(store settingsLoader, lister gameproc.Lister, mode *gameproc.Mode, sync syncRequester) *gameproc.Watcher {
	w := &gameproc.Watcher{
		Lister: lister,
		Mode:   mode,
		Names:  func() []string { return title.DefaultRegistry().GameProcessNames() },
		Enabled: func() bool {
			cfg, err := store.Load()
			if err != nil {
				// Réglage illisible : le défaut (actif). Journalisé, jamais muet.
				slog.Warn("game_mode: réglages illisibles, détection du jeu laissée active", "err", err)
				return true
			}
			return cfg.GameProcessWatch
		},
	}
	if sync != nil {
		w.OnGameClosed = func(reason string) {
			if !sync.RequestCatchUp(reason) {
				slog.Info("game_mode: une demande de synchronisation attend déjà, celle-ci est couverte")
			}
		}
	}
	return w
}
