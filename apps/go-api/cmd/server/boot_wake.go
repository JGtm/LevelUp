// boot_wake.go — la séquence de réveil du serveur desktop (ADR 0045 D-10) : au lancement et à chaque
// sortie de veille, vérifier les jetons des profils suivis puis tirer un cycle de sync si le dernier
// est plus vieux que l'intervalle d'auto-sync. La logique vit dans internal/wakeup ; ce fichier la
// branche sur l'ordonnanceur et le magasin de jetons du processus.
package main

import (
	"log/slog"

	"levelup/go-api/internal/clock"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/scheduler"
	"levelup/go-api/internal/sync/deadtoken"
	"levelup/go-api/internal/wakeup"
)

// launchWakeSequence lance la séquence de réveil si le profil l'offre (capacité WakeSequence, vraie
// en desktop seulement : un serveur hébergé redémarre à chaque déploiement et ne dort jamais). Elle
// démarre une fois le routeur branché, sous le contexte des tâches de fond et dans leur groupe
// d'attente : l'arrêt l'annule puis l'attend avant la fermeture des bases.
func launchWakeSequence(st *bootState) {
	cfg := st.cfg
	if !cfg.Capabilities.WakeSequence {
		return
	}
	seq := newWakeSequence(cfg, st.tokenProvider, st.autoScheduler)
	ctx := st.schedulerCtx
	st.schedulerWG.Add(1)
	go func() {
		defer st.schedulerWG.Done()
		seq.Run(ctx)
	}()
	slog.InfoContext(ctx, "wakeup: séquence de réveil armée (lancement, sortie de veille)",
		"tic", wakeup.SleepTick, "seuil_de_veille", wakeup.SleepGap)
}

// newWakeSequence assemble la séquence de production : vérification des jetons de db_profiles.json
// contre le magasin de jetons (source unique, ADR 0023), rattrapage par l'ordonnanceur d'auto-sync.
func newWakeSequence(cfg *config.AppConfig, provider auth.TokenProvider, sched *scheduler.AutoSyncScheduler) *wakeup.Sequence {
	return &wakeup.Sequence{Tokens: newTokenCheck(cfg, provider), Sync: sched, Clock: clock.System{}}
}

// newTokenCheck rend la vérification des jetons des profils suivis, partagée par la séquence de
// réveil et le mode refresh-only : une seule définition de « vérifier les jetons ».
func newTokenCheck(cfg *config.AppConfig, provider auth.TokenProvider) *wakeup.TokenCheck {
	return &wakeup.TokenCheck{
		Players:  func() ([]domain.PlayerSummary, error) { return cfg.LoadPlayers() },
		Store:    auth.NewMultiUserTokenStore(cfg.WatcherTokensDir()),
		Provider: provider,
		Gate:     deadtoken.New(),
	}
}
