// Package handlers — static_cache.go : politique de cache des fichiers du build de la page
// (assets Vite hashés, fichiers publics) et service d'un fichier du dist.
package handlers

import (
	"net/http"
	"regexp"

	"levelup/go-api/internal/api/middleware"
)

// immutableAssetCacheControl : politique de cache des assets Vite au nom hashé.
// max-age 1 an + immutable — sûr car le hash de contenu change à chaque build,
// donc l'URL elle-même est un cache-buster (le navigateur ne revalide jamais).
const immutableAssetCacheControl = "public, max-age=31536000, immutable"

// publicAssetCacheControl : politique de cache des fichiers statiques NON
// fingerprintés du dist (recopiés depuis apps/web/public/ : /logo.png, icônes,
// /titles/**, polices...). 1 h, sans `immutable` : leur URL est stable d'un build
// à l'autre, donc un cache long empêcherait de livrer une image corrigée — mais
// sans aucun Cache-Control (état d'avant le 2026-07-26) chaque navigation
// re-tirait la centaine d'images d'une page, ce qui vidait le bucket rate limit.
// Le validateur Last-Modified posé par le FileServer permet un 304 peu coûteux
// après expiration.
const publicAssetCacheControl = "public, max-age=3600"

// viteHashedAssetPattern reconnaît les fichiers émis par Vite avec un hash de
// contenu : `/assets/<nom>-<hash>.<ext>`. Le hash par défaut de Vite fait 8
// caractères base64url ([A-Za-z0-9_-]) ; on exige >= 8 pour ne PAS confondre un
// simple nom à tirets (`/assets/mon-fichier.css`) avec un asset hashé. index.html
// et tout fichier non hashé (racine, /favicon.ico…) ne matchent pas → jamais
// marqués immutable (ils DOIVENT rester revalidables pour livrer un nouveau build).
var viteHashedAssetPattern = regexp.MustCompile(`^/assets/.+-[A-Za-z0-9_-]{8,}\.[A-Za-z0-9]+$`)

// isViteHashedAsset indique si le chemin URL cible un asset Vite au nom hashé.
func isViteHashedAsset(urlPath string) bool {
	return viteHashedAssetPattern.MatchString(urlPath)
}

// serveStaticFile délègue au FileServer (qui pose ETag/Last-Modified/Content-Type)
// après avoir posé le Cache-Control adapté :
//   - asset Vite hashé → immutable 1 an (l'URL EST le cache-buster) ;
//   - autre fichier statique (public/ recopié à la racine du dist) → 1 h ;
//   - index.html et tout ce qui n'est pas un fichier statique → aucun header,
//     donc revalidation systématique (un nouveau build doit être livré).
//
// Le header est posé AVANT ServeHTTP : il survit aussi bien au 200 qu'au 304.
func serveStaticFile(w http.ResponseWriter, req *http.Request, fileServer http.Handler) {
	switch {
	case isViteHashedAsset(req.URL.Path):
		w.Header().Set("Cache-Control", immutableAssetCacheControl)
	case middleware.IsStaticAssetPath(req.URL.Path):
		w.Header().Set("Cache-Control", publicAssetCacheControl)
	}
	fileServer.ServeHTTP(w, req)
}
