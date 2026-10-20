package wire

import (
	"net/http"

	"levelup/go-api/internal/games"
)

// registry_accessors.go — accesseurs exportés pour les membres de ServiceRegistry
// consommés par api/server.go (NewRouter) après l'extraction du package wire (K3d).
// server.go reste dans le package api (racine fine) et ne peut plus toucher les
// champs/méthodes non-exportés de ServiceRegistry.

// Resolve retourne le PlayerResolver (résolution player_slug -> *PlayerDB).
func (r *ServiceRegistry) Resolve() PlayerResolver { return r.resolve }

// HiCapabilities retourne les capabilities HI chargées au boot (nil possible).
func (r *ServiceRegistry) HiCapabilities() games.CapabilityMap { return r.hiCapabilities }

// ServeIndexWithOG sert l'index.html avec injection des meta Open Graph. Sûr sur un
// registre nil (carte générique) : la page servie pendant le démarrage du serveur, avant
// que le registre existe, passe par là (cf. api.NewBootPageHandler).
func (r *ServiceRegistry) ServeIndexWithOG(w http.ResponseWriter, req *http.Request, indexPath string) {
	r.serveIndexWithOG(w, req, indexPath)
}
