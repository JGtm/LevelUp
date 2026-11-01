// boot_shutdown.go — câblage de l'arrêt : attente de la demande d'arrêt, composition des
// dépendances de runShutdown (shutdown.go) à partir de l'état du démarrage, fermeture du pool.

package main

// awaitShutdown bloque jusqu'à la demande d'arrêt (signal, stdin, route HTTP) puis exécute la
// séquence d'arrêt.
func awaitShutdown(st *bootState) {
	<-st.bootSrv.stopRequested()
	runShutdown(st.toShutdownDeps())
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
	}
}

// closeAutoSyncPool ferme le pool de tokens s'il existe ; main le diffère jusqu'à son retour,
// donc après la séquence d'arrêt.
func (st *bootState) closeAutoSyncPool() {
	if st.autoSyncPool != nil {
		st.autoSyncPool.Close()
	}
}
