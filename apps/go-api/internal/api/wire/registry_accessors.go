package wire

import (
	"net/http"

	"levelup/go-api/internal/games"
	session_platform "levelup/go-api/internal/platform/session"
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
// que le registre existe, passe par là (cf. handlers.NewBootPageHandler).
func (r *ServiceRegistry) ServeIndexWithOG(w http.ResponseWriter, req *http.Request, indexPath string) {
	r.serveIndexWithOG(w, req, indexPath)
}

// WithSessionStore attache le magasin des sessions HTTP du routeur : la tâche de fond de purge
// des sessions expirées (cmd/server) le lit par SessionStore, pour agir sur le MÊME magasin que
// le routeur (ses marques d'écriture en mémoire sont propres à l'instance).
func (r *ServiceRegistry) WithSessionStore(store *session_platform.Store) *ServiceRegistry {
	r.sessionStore = store
	return r
}

// SessionStore rend le magasin des sessions du routeur (nil tant qu'il n'est pas attaché).
func (r *ServiceRegistry) SessionStore() *session_platform.Store { return r.sessionStore }
