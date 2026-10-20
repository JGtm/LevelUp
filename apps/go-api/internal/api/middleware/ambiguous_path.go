// Package middleware — ambiguous_path.go : forme d'un chemin que les gardes ne
// peuvent pas lire comme chi le route.
//
// Toute garde qui décide sur le chemin (préfixe, exemption, classification lecture /
// écriture) appelle AmbiguousPath avant de lire `r.URL.Path` : garde-rail
// `archlint/ambiguous_path_middleware_test.go`. La table des formes ambiguës passée à
// chaque garde vit dans `internal/api/ambiguous_path_table_test.go`.
package middleware

import (
	"net/http"
	"strings"
)

// AmbiguousPath dit si le chemin que lisent les gardes peut désigner une AUTRE route que
// celle que chi résout. chi route sur le chemin BRUT (`URL.RawPath`) ; les gardes lisent le
// chemin décodé (`URL.Path`). Les deux divergent dès qu'un séparateur ou un point est ENCODÉ
// (%2F, %5C, %2E) — « coach/proposals/..%2F..%2Fpages/dismiss » se lit « …/pages/dismiss »
// mais se route vers dismiss — ou qu'un segment « . », « .. » ou vide apparaît. Une telle
// requête n'est JAMAIS classée en lecture, et aucun nettoyage (path.Clean) ne la rend
// lisible : la classification ne peut pas s'écarter du routage.
func AmbiguousPath(r *http.Request) bool {
	raw := strings.ToLower(r.URL.EscapedPath())
	for _, enc := range []string{"%2f", "%5c", "%2e"} {
		if strings.Contains(raw, enc) {
			return true
		}
	}
	p := r.URL.Path
	if strings.ContainsRune(p, '\\') || strings.Contains(p, "//") {
		return true
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "." || seg == ".." {
			return true
		}
	}
	return false
}
