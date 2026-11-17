// boot_replayregen.go — relie la file des rejeux (internal/replayregen, ADR 0045) au contexte et au groupe d'attente du
// serveur, reprend à l'ouverture la file que l'utilisateur avait lancée avant un arrêt (drapeau
// `replay_regen_pending`) et les demandes de génération d'un rejeu en vol. Deux capacités, deux entrées :
//   - ReplayRegen (bureau) : le lot de régénération (reprise par drapeau, décompte de démarrage). La file ne part
//     jamais seule : sans drapeau posé par un démarrage demandé, rien ne tourne ;
//   - GenerateReplays (bureau et poste de développement) : les demandes explicites d'un match (reprise du fichier).
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

// replayRegenModes : ce que le profil de déploiement active dans la file.
type replayRegenModes struct {
	// Batch : le lot de régénération des rejeux en retard (capacité ReplayRegen).
	Batch bool
	// Demands : la reprise des demandes explicites de génération (capacité GenerateReplays).
	Demands bool
}

// bindReplayRegen relie la file et reprend ce qui était en vol. Appelée après la construction du routeur
// (le registre de services porte la reconstruction d'un match).
func bindReplayRegen(st *bootState) {
	caps := st.cfg.Capabilities
	if (!caps.ReplayRegen && !caps.GenerateReplays) || st.reg == nil || st.settingsStore == nil {
		return
	}
	deps := st.reg.ReplayRegenDeps(st.settingsStore)
	if st.watcherDaemon != nil { // sans suivi, aucun coordinateur à qui céder la place : le verrou solo arbitre seul
		deps.SyncRunning = st.watcherDaemon.SyncRunning
	}
	startReplayRegen(st.schedulerCtx, &st.schedulerWG, replayregen.Default(), deps,
		replayRegenModes{Batch: caps.ReplayRegen, Demands: caps.GenerateReplays})
}

// startReplayRegen relie le service, puis selon les modes : reprend la file lancée avant l'arrêt (drapeau posé
// seulement) et lance le décompte de démarrage (sans effet si la file a repris : elle refait le décompte à sa fin), et
// reprend les demandes de génération en vol. Le décompte est le seul qui se lance seul : il ne lit que les petits
// fichiers d'en-tête et attend la porte du mode jeu.
func startReplayRegen(ctx context.Context, wg *sync.WaitGroup, svc *replayregen.Service, deps replayregen.Deps, modes replayRegenModes) {
	svc.Bind(ctx, wg, deps)
	if modes.Batch {
		svc.ResumeIfPending()
		if _, err := svc.Rescan(); err != nil {
			slog.WarnContext(ctx, "replay_regen: décompte de démarrage impossible", "err", err)
		}
	}
	if modes.Demands {
		svc.ResumeDemands()
	}
}
