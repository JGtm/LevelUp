// Package api — commendation_handler.go : handler pour les images de citations/commendations.
// Utilise os.Open + http.ServeContent pour éviter les redirects de http.ServeFile
// lorsque RawPath != Path (caractères spéciaux URL-encodés dans le nom de fichier).
package api

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// commendationHandler sert les fichiers sous /static/commendations/ avec fallback :
//  1. Nom décodé (cas nominal : apostrophes, accents littéraux sur disk)
//  2. RawPath URL-décodé explicitement via url.PathUnescape (cas où net/http
//     a laissé `%27` non décodé car `'` est "unreserved" RFC 3986)
//  3. Nom URL-encodé littéral sur disk (fichiers avec ? → %3F, Windows interdit ?)
type commendationHandler struct {
	dir string // répertoire static/ absolu
}

func newCommendationHandler(staticDir string) http.Handler {
	return &commendationHandler{dir: staticDir}
}

func (h *commendationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Essai 1 : chemin décodé (r.URL.Path est décodé par net/http)
	decodedRel := strings.TrimPrefix(r.URL.Path, "/static/")
	fullDecoded := filepath.Join(h.dir, filepath.FromSlash(decodedRel))
	if f, fi, ok := h.openUnder(fullDecoded); ok {
		defer f.Close()
		http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
		return
	}

	// Essai 2 : si RawPath présent, décoder explicitement (couvre les cas où
	// net/http a laissé certains caractères "unreserved" RFC 3986 encodés
	// dans Path — ex `%27` apostrophe, `%21` !, `%24` $).
	rawPath := r.URL.RawPath
	if rawPath != "" {
		if unescaped, err := url.PathUnescape(strings.TrimPrefix(rawPath, "/static/")); err == nil && unescaped != decodedRel {
			fullForcedDecoded := filepath.Join(h.dir, filepath.FromSlash(unescaped))
			if f, fi, ok := h.openUnder(fullForcedDecoded); ok {
				defer f.Close()
				http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
				return
			}
		}
	}

	// Essai 3 : nom URL-encodé littéral sur disk (? interdit Windows → %3F stocké tel quel)
	if rawPath == "" {
		rawPath = r.URL.Path
	}
	encodedRel := strings.TrimPrefix(rawPath, "/static/")
	fullEncoded := filepath.Join(h.dir, filepath.FromSlash(encodedRel))
	if f, fi, ok := h.openUnder(fullEncoded); ok {
		defer f.Close()
		http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
		return
	}

	http.NotFound(w, r)
}

// openUnder ouvre `path` seulement s'il reste sous le dossier des citations
// (`<static>/commendations`). Les trois essais de ServeHTTP construisent le chemin
// depuis l'URL, décodée ou brute : un segment `..` (en clair ou encodé `%2f`,
// `%2e%2e`, `%5c`) y ferait sortir le chemin du dossier servi et lirait n'importe quel
// fichier du serveur. Tout chemin qui sort du dossier est refusé comme absent.
func (h *commendationHandler) openUnder(path string) (*os.File, os.FileInfo, bool) {
	base := filepath.Clean(filepath.Join(h.dir, "commendations"))
	clean := filepath.Clean(path)
	rel, err := filepath.Rel(base, clean)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return nil, nil, false
	}
	f, err := os.Open(clean)
	if err != nil {
		return nil, nil, false
	}
	fi, err := f.Stat()
	if err != nil || fi.IsDir() {
		f.Close()
		return nil, nil, false
	}
	return f, fi, true
}
