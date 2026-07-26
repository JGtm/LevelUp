// Package middleware — static_assets.go : SOURCE UNIQUE des extensions de
// fichiers statiques servis par le front (dist Vite : `assets/` hashés +
// `public/` non hashés recopié à la racine).
//
// Trois consommateurs, une seule liste :
//   - `RateLimit` (rate_limit.go) : exempte ces chemins du bucket httprate —
//     une page peut tirer 100+ images/icônes en rafale.
//   - `serveStaticFile` (internal/api/server_apiv1.go) : pose le Cache-Control
//     des fichiers non fingerprintés.
//   - `mountSPA` (idem) : un chemin à extension statique absent du dist est un
//     404 franc, jamais un fallback index.html.
//
// La liste vivait initialement en dur dans server_apiv1.go ; elle est remontée
// ici parce que `internal/api` importe `internal/api/middleware` (jamais
// l'inverse) — le middleware est donc le point le plus bas commun aux trois
// consommateurs, sans cycle d'import possible.
//
// Garde-rail : `internal/archlint/no_duplicate_static_ext_list_test.go` interdit
// toute 2e liste littérale d'extensions statiques (CLAUDE.md règle 6).
package middleware

import (
	"path"
	"strings"
)

// staticAssetExts : extensions de fichiers statiques attendues dans le dist Vite.
// Un chemin qui porte l'une d'elles mais n'existe pas sur disque est un asset
// MANQUANT (404 franc + log), jamais une route SPA — avant ce correctif, le
// catch-all SPA servait index.html (200 text/html) pour TOUT asset absent du
// dist, quelle qu'en soit la cause (build amputé, fichier déplacé/renommé,
// déploiement partiel...) : le silence de ce fallback rendait ces manques
// invisibles côté client (<img> vide, zéro erreur réseau visible).
//
// `.html` en est volontairement ABSENT : index.html est l'entrée SPA, elle doit
// rester revalidable à chaque déploiement (aucun Cache-Control long).
var staticAssetExts = map[string]struct{}{
	".png": {}, ".jpg": {}, ".jpeg": {}, ".webp": {}, ".gif": {}, ".svg": {},
	".ico": {}, ".css": {}, ".js": {}, ".mjs": {}, ".map": {}, ".woff": {},
	".woff2": {}, ".ttf": {}, ".txt": {}, ".xml": {}, ".json": {},
}

// IsStaticAssetPath indique si le chemin URL porte une extension de fichier
// statique connue (insensible à la casse).
func IsStaticAssetPath(urlPath string) bool {
	_, ok := staticAssetExts[strings.ToLower(path.Ext(urlPath))]
	return ok
}
