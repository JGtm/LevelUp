// boot_shutdown.go — câblage de l'arrêt : attente de la demande d'arrêt, composition des
// dépendances de runShutdown (shutdown.go) à partir de l'état du démarrage, fermeture du pool.

package main

// awaitShutdown bloque jusqu'à la demande d'arrêt (signal, stdin, route HTTP) puis exécute la
// séquence d'arrêt. Le pool de jetons se ferme et le verrou d'instance se rend ensuite, et
// l'événement `stopped` vient en TOUTE FIN : une instance relancée dès qu'elle le lit trouve les
// bases fermées, les journaux vidés et le verrou libre.
func awaitShutdown(st *bootState) {
	<-st.bootSrv.stopRequested()
	clean := runShutdown(st.toShutdownDeps())
	st.closeAutoSyncPool()
	st.releaseInstanceLock()
	st.events.Stopped(clean)
}

// toShutdownDeps réunit ce que la séquence d'arrêt arrête, vide ou ferme.
func (st *bootState) toShutdownDeps() shutdownDeps {
	return shutdownDeps{
		reason:           st.bootSrv.shutdownReason(),
		srv:              st.srv,
		cancelScheduler:  st.cancelScheduler,
		watcherDaemon:    st.watcherDaemon,
		autoScheduler:    st.autoScheduler,
		schedulerWG:      &st.schedulerWG,
		autoBatchQueue:   st.autoBatchQueue,
		workerWG:         &st.workerWG,
		sharedSocialPath: st.dbPaths.sharedSocial,
		reg:              st.reg,
		closeShared:      st.closeShared,
		metaDB:           st.metaDB,
		multiHandler:     st.multiHandler,
		events:           st.events,
	}
}

// closeAutoSyncPool ferme le pool de tokens s'il existe ; awaitShutdown l'appelle après la
// séquence d'arrêt.
func (st *bootState) closeAutoSyncPool() {
	if st.autoSyncPool != nil {
		st.autoSyncPool.Close()
	}
}
