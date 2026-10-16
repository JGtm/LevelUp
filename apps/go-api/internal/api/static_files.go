// Package api — static_files.go : service des fichiers de static/ sous /static/.
package api

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// registerStaticRoutes monte /static/* sur staticDir (images de cartes, médailles, armes,
// sons...). /static/commendations/* passe par son handler dédié (noms de fichiers encodés).
// Un dossier n'est jamais listé : il répond 404 (noDirListingFS), les fichiers restent
// servis.
func registerStaticRoutes(r chi.Router, staticDir string) {
	r.Handle("/static/commendations/*", newCommendationHandler(staticDir))
	r.Handle("/static/*", http.StripPrefix("/static/",
		http.FileServer(noDirListingFS{http.Dir(staticDir)})))
}

// noDirListingFS refuse l'ouverture d'un dossier comme s'il n'existait pas : http.FileServer
// répond alors 404 au lieu de rendre le listing du dossier (ou de rediriger vers lui).
type noDirListingFS struct {
	fs http.FileSystem
}

func (n noDirListingFS) Open(name string) (http.File, error) {
	f, err := n.fs.Open(name)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err == nil && !info.IsDir() {
		return f, nil
	}
	_ = f.Close() // ouvert en lecture seule : la fermeture ne peut rien perdre
	if err != nil {
		return nil, err
	}
	return nil, fs.ErrNotExist
}
