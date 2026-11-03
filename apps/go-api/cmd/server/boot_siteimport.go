// boot_siteimport.go — relie l'import depuis le site (lot L22.b) au contexte et au groupe d'attente du serveur :
// un import en cours est annulé à l'arrêt, puis attendu avant la fermeture des bases.
package main

import (
	"levelup/go-api/internal/api/wire"
	"levelup/go-api/internal/service"
)

// bindSiteImport relie le service de l'import depuis le site sous la capacité ImportFromSite.
func bindSiteImport(st *bootState) {
	if st.cfg.Capabilities.ImportFromSite {
		service.DefaultSiteImport().Bind(st.schedulerCtx, &st.schedulerWG, st.cfg, wire.PlayerReadCaches{})
	}
}
