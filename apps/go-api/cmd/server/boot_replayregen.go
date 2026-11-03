// boot_replayregen.go — relie la file de régénération des rejeux (internal/replayregen, ADR 0045) au contexte et au
// groupe d'attente du serveur, et reprend à l'ouverture la file que l'utilisateur avait lancée avant un arrêt
// (drapeau `replay_regen_pending`). Sous la capacité ReplayRegen seulement ; la file ne part jamais seule : sans
// drapeau posé par un démarrage demandé, rien ne tourne.
package main

import (
	"levelup/go-api/internal/replayregen"
)

// bindReplayRegen relie la file et reprend celle qui était active. Appelée après la construction du routeur
// (le registre de services porte la reconstruction d'un match).
func bindReplayRegen(st *bootState) {
	if !st.cfg.Capabilities.ReplayRegen || st.reg == nil || st.settingsStore == nil {
		return
	}
	svc := replayregen.Default()
	svc.Bind(st.schedulerCtx, &st.schedulerWG, st.reg.ReplayRegenDeps(st.settingsStore))
	svc.ResumeIfPending()
}
