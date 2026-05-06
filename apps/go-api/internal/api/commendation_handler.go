// Package api — commendation_handler.go : handler pour les images de citations/commendations.
// Gère le cas où les fichiers sur disk ont des noms URL-encodés littéraux (ex: %3F pour ?)
// quand les caractères correspondants sont interdits dans les noms de fichiers Windows.
package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// commendationHandler sert les fichiers sous /static/commendations/ avec fallback :
//  1. Cherche le fichier avec le nom décodé (cas nominal post-renommage)
//  2. Si absent, cherche le fichier avec le nom URL-encodé littéral sur disk
//     (cas des noms avec ? ou autres caractères interdits Windows)
type commendationHandler struct {
	dir string // répertoire static/ absolu
}

func newCommendationHandler(staticDir string) http.Handler {
	return &commendationHandler{dir: staticDir}
}

func (h *commendationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// r.URL.Path est déjà décodé par net/http (ex: "h5g/H5G_citation_Can't_we_get_along?.png")
	// r.URL.RawPath est le path tel que reçu dans la requête HTTP (peut être vide si == Path)
	rawPath := r.URL.RawPath
	if rawPath == "" {
		rawPath = r.URL.Path
	}

	// Extraire le chemin relatif après /static/
	rel := strings.TrimPrefix(rawPath, "/static/")
	// rel = "commendations/halo_5_guardians/H5G_citation_Can%27t_we_get_along%3F.png"

	// Essai 1 : chemin décodé (fichiers renommés avec accents littéraux)
	decodedRel := strings.TrimPrefix(r.URL.Path, "/static/")
	// decodedRel = "commendations/halo_5_guardians/H5G_citation_Can't_we_get_along?.png"

	fullDecoded := filepath.Join(h.dir, filepath.FromSlash(decodedRel))
	if _, err := os.Stat(fullDecoded); err == nil {
		http.ServeFile(w, r, fullDecoded)
		return
	}

	// Essai 2 : chemin avec URL-encoding littéral sur disk (fichiers non renommés)
	fullEncoded := filepath.Join(h.dir, filepath.FromSlash(rel))
	if _, err := os.Stat(fullEncoded); err == nil {
		http.ServeFile(w, r, fullEncoded)
		return
	}

	http.NotFound(w, r)
}
