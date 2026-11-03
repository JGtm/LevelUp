// boot_replayregen.go — relie la file de régénération des rejeux (internal/replayregen, ADR 0045) au contexte et au
// groupe d'attente du serveur, et reprend à l'ouverture la file que l'utilisateur avait lancée avant un arrêt
// (drapeau `replay_regen_pending`). Sous la capacité ReplayRegen seulement ; la file ne part jamais seule : sans
// drapeau posé par un démarrage demandé, rien ne tourne.
//
// ORDRE : après `armGameWatch` (main.go). La porte du mode jeu doit être armée (premier relevé du jeu fait) avant que la
// file reprenne : une file reprise avant lui décoderait un film pendant une partie déjà lancée.
package main

import (
	"context"
	"log/slog"
	"sync"

	"levelup/go-api/internal/replayregen"
)

// bindReplayRegen relie la file et reprend celle qui était active. Appelée après la construction du routeur
// (le registre de services porte la reconstruction d'un match).
func bindReplayRegen(st *bootState) {
	if !st.cfg.Capabilities.ReplayRegen || st.reg == nil || st.settingsStore == nil {
		return
	}
	deps := st.reg.ReplayRegenDeps(st.settingsStore)
	if st.watcherDaemon != nil { // sans suivi, aucun coordinateur à qui céder la place : le verrou solo arbitre seul
		deps.SyncRunning = st.watcherDaemon.SyncRunning
	}
	startReplayRegen(st.schedulerCtx, &st.schedulerWG, replayregen.Default(), deps)
}

// startReplayRegen relie le service, reprend la file lancée avant l'arrêt (drapeau posé seulement), puis lance le
// décompte de démarrage (sans effet si la file a repris : elle refait le décompte à sa fin). Le décompte est le seul qui
// se lance seul : il ne lit que les petits fichiers d'en-tête et attend la porte du mode jeu.
func startReplayRegen(ctx context.Context, wg *sync.WaitGroup, svc *replayregen.Service, deps replayregen.Deps) {
	svc.Bind(ctx, wg, deps)
	svc.ResumeIfPending()
	if _, err := svc.Rescan(); err != nil {
		slog.WarnContext(ctx, "replay_regen: décompte de démarrage impossible", "err", err)
	}
}
