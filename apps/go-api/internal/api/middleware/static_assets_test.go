package middleware_test

import (
	"testing"

	"levelup/go-api/internal/api/middleware"
)

// TestIsStaticAssetPath couvre la source unique consommée par le rate limiter,
// serveStaticFile et mountSPA.
func TestIsStaticAssetPath(t *testing.T) {
	cases := map[string]bool{
		"/icons/halowaypoint-white.png": true,
		"/logo.PNG":                     true, // insensible à la casse
		"/photo.jpg":                    true,
		"/photo.jpeg":                   true,
		"/anim.webp":                    true,
		"/anim.gif":                     true,
		"/titles/halo_5/emblem.svg":     true,
		"/favicon.ico":                  true,
		"/assets/app-Bx1Yq9Zk.css":      true,
		"/assets/index-DiwrgTda.js":     true,
		"/module.mjs":                   true,
		"/assets/index-DiwrgTda.js.map": true,
		"/fonts/inter.woff":             true,
		"/fonts/inter.WOFF2":            true,
		"/fonts/inter.ttf":              true,
		"/robots.txt":                   true,
		"/sitemap.xml":                  true,
		"/manifest.json":                true,
		"/index.html":                   false, // entrée SPA — jamais un asset
		"/":                             false, // racine → SPA
		"/players/demo-player/home":     false, // route SPA sans extension
		"/explorer":                     false,
		"/api/v1/players/Guillaume":     false,
		"/manifest.webmanifest":         false, // extension inconnue → SPA (statu quo)
		"/players/gamer.tag/home":       false, // point dans un segment != extension connue
		"/media/clip.mp4":               false, // hors périmètre front
	}
	for urlPath, want := range cases {
		if got := middleware.IsStaticAssetPath(urlPath); got != want {
			t.Errorf("IsStaticAssetPath(%q) = %v, attendu %v", urlPath, got, want)
		}
	}
}
