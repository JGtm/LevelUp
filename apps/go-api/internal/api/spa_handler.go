package api

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/api/wire"
	"levelup/go-api/internal/config"
)

// indexServer sert index.html (chemin donné) pour une route client-side de la page.
type indexServer func(w http.ResponseWriter, req *http.Request, indexPath string)

// newSPAHandler rend le catch-all du build Vite servi depuis dist : un fichier du dist est
// servi tel quel (Cache-Control selon serveStaticFile) ; un chemin à extension statique
// absent du dist rend un 404 franc (cf. middleware.IsStaticAssetPath) ; tout autre chemin
// est une route client-side React et reçoit index.html par serveIndex. ok=false si dist est
// vide ou sans index.html.
func newSPAHandler(dist string, serveIndex indexServer) (h http.HandlerFunc, ok bool) {
	if dist == "" {
		return nil, false
	}
	indexPath := filepath.Join(dist, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return nil, false
	}
	fileServer := http.FileServer(http.Dir(dist))
	return func(w http.ResponseWriter, req *http.Request) {
		if fi, err := os.Stat(filepath.Join(dist, filepath.Clean(req.URL.Path))); err == nil && !fi.IsDir() {
			serveStaticFile(w, req, fileServer)
			return
		}
		if middleware.IsStaticAssetPath(req.URL.Path) {
			// Log volontairement non throttle : c'est le silence de l'ancien fallback
			// (200 text/html sur tout asset absent) qui rendait ce genre de manque
			// invisible en prod.
			slog.WarnContext(req.Context(), "static asset not found", "path", req.URL.Path)
			http.NotFound(w, req)
			return
		}
		serveIndex(w, req, indexPath)
	}, true
}

// serverRoutePrefixes : racines des routes que le routeur sert LUI-MÊME (hors page web).
// Une requête sous l'une d'elles n'est jamais une route de la page : pendant le démarrage,
// elle reçoit la réponse « serveur en démarrage », jamais index.html (un retour OAuth
// servi en page perdrait son code). Tenue par TestServerRoutePath_CouvreLeRouteur, qui
// parcourt le vrai routeur.
var serverRoutePrefixes = []string{"/api/", "/auth/", "/static/", "/debug/", "/docs/"}

// serverRoutePaths : routes racine exactes servies par le routeur (sondes de santé, docs).
var serverRoutePaths = map[string]struct{}{
	"/health": {}, "/healthz": {}, "/readyz": {}, "/docs": {},
}

// isServerRoutePath indique si urlPath relève d'une route du routeur et non de la page web.
func isServerRoutePath(urlPath string) bool {
	if _, ok := serverRoutePaths[urlPath]; ok {
		return true
	}
	for _, prefix := range serverRoutePrefixes {
		if strings.HasPrefix(urlPath, prefix) {
			return true
		}
	}
	return false
}

// NewBootPageHandler rend le handler servi PENDANT le démarrage du serveur, avant que le
// routeur soit prêt : les fichiers du build Vite (cfg.WebDistDir) et index.html pour les
// routes de la page, pour que la page s'affiche et attende l'API. Toute autre requête —
// route du routeur (isServerRoutePath), méthode autre que GET/HEAD — va à starting. Sans
// page à servir (dist vide ou sans index.html : en dev, Vite sert la page), tout va à
// starting. Toutes ses réponses portent les en-têtes de sécurité du routeur
// (middleware.SecurityHeaders).
//
// index.html y reçoit la carte Open Graph générique (registre nil : cf. serveIndexWithOG).
func NewBootPageHandler(cfg *config.AppConfig, starting http.Handler) http.Handler {
	secure := middleware.SecurityHeaders(cfg.TrustProxyHeaders)
	var noRegistry *wire.ServiceRegistry
	spa, ok := newSPAHandler(cfg.WebDistDir, noRegistry.ServeIndexWithOG)
	if !ok {
		return secure(starting)
	}
	return secure(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if (req.Method != http.MethodGet && req.Method != http.MethodHead) || isServerRoutePath(req.URL.Path) {
			starting.ServeHTTP(w, req)
			return
		}
		spa(w, req)
	}))
}
