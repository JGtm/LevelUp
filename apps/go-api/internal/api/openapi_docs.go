// openapi_docs.go — UI de documentation OpenAPI, montée UNIQUEMENT hors production
// (reliquat V72-01, item backlog archi/contrat).
//
// POURQUOI PAS `huma.Config.DocsPath`. Le document OpenAPI est PARTAGÉ par une
// dizaine de sous-routeurs (NewSubrouterAPI) : un DocsPath non vide ferait
// enregistrer la route de docs par CHAQUE humachi.New, donc une fois par
// sous-routeur et sous son préfixe de montage (`/api/v1/players/{player_slug}/docs`…).
// La route est donc posée ICI, une seule fois, sur le routeur RACINE.
//
// CE QUI EST SERVI : le document VIVANT du serveur (routes + schémas dérivés des
// types Go), pas le contrat publié `api/openapi.yaml`. Les deux diffèrent par ce
// que le fragment manuel apporte (corps `RawBody`, routes chi-brut, descriptions
// de schéma racine). C'est assumé : cette UI est un outil de dev qui doit refléter
// le BINAIRE qui tourne ; le contrat publié, lui, est versionné et vérifié par le
// golden TestOpenAPIYAMLIsUpToDate.
//
// GATE : les trois routes n'existent PAS sur un déploiement exposé — production
// (LEVELUP_ENV=production) ET démo. La démo (demo.lvelup.info) est publique et ne
// pose pas LEVELUP_ENV : la gater sur la seule production l'aurait exposée. Sur un
// déploiement gaté, /docs tombe dans le catch-all SPA (404 côté front).
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/config"
)

// Chemins de l'UI de documentation (routeur racine, hors /api/v1).
const (
	docsUIPath   = "/docs"
	docsJSONPath = "/docs/openapi.json"
	docsYAMLPath = "/docs/openapi.yaml"
)

// docsCSP — Content-Security-Policy de la SEULE page /docs (le reste de l'app n'en
// a pas, cf. middleware.SecurityHeaders). Autorise strictement les deux ressources
// Stoplight Elements épinglées par version, rien d'autre.
var docsCSP = strings.Join([]string{
	"default-src 'none'",
	"base-uri 'none'",
	"connect-src 'self'",
	"form-action 'none'",
	"frame-ancestors 'none'",
	"img-src 'self' data:",
	"script-src https://unpkg.com/@stoplight/elements@9.0.15/web-components.min.js",
	"style-src 'unsafe-inline' https://unpkg.com/@stoplight/elements@9.0.15/styles.min.css",
}, "; ")

// docsHTML — page Stoplight Elements pointant sur le document vivant.
const docsHTML = `<!doctype html>
<html lang="fr">
  <head>
    <meta charset="utf-8">
    <meta name="referrer" content="no-referrer">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>LevelUp API — référence (dev)</title>
    <link rel="stylesheet" href="https://unpkg.com/@stoplight/elements@9.0.15/styles.min.css" crossorigin integrity="sha384-iVQBHadsD+eV0M5+ubRCEVXrXEBj+BqcuwjUwPoVJc0Pb1fmrhYSAhL+BFProHdV">
    <script src="https://unpkg.com/@stoplight/elements@9.0.15/web-components.min.js" crossorigin integrity="sha384-xjOcq9PZ/k+pGtPS/xcsCRXGjKKfTlIa4H1IYEnC+97jNa6sAMWTNrV6hY08W3GL"></script>
  </head>
  <body style="height: 100vh;">
    <elements-api
      apiDescriptionUrl="` + docsYAMLPath + `"
      router="hash"
      layout="sidebar"
      tryItCredentialsPolicy="same-origin"
    ></elements-api>
  </body>
</html>
`

// mountOpenAPIDocs monte /docs (+ ses deux sources) sur le routeur racine, sauf
// sur un déploiement exposé (production ou démo). No-op si le document est nil.
func mountOpenAPIDocs(r chi.Router, cfg *config.AppConfig, doc *huma.OpenAPI) {
	if doc == nil {
		return
	}
	if cfg != nil && (cfg.IsProduction() || cfg.DemoMode) {
		slog.Debug("openapi docs: UI non montée (déploiement exposé)",
			"path", docsUIPath, "production", cfg.IsProduction(), "demo", cfg.DemoMode)
		return
	}
	src := &docsSource{doc: doc}

	r.Get(docsUIPath, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", docsCSP)
		_, _ = w.Write([]byte(docsHTML))
	})
	r.Get(docsJSONPath, func(w http.ResponseWriter, _ *http.Request) {
		body, err := src.jsonBody()
		if err != nil {
			http.Error(w, "openapi encode failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})
	r.Get(docsYAMLPath, func(w http.ResponseWriter, _ *http.Request) {
		body, err := src.yamlBody()
		if err != nil {
			http.Error(w, "openapi encode failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/openapi+yaml")
		_, _ = w.Write(body)
	})
}

// docsSource sérialise le document une seule fois par format (le document ne bouge
// plus après le montage des routes).
type docsSource struct {
	doc      *huma.OpenAPI
	jsonOnce sync.Once
	jsonRaw  []byte
	jsonErr  error
	yamlOnce sync.Once
	yamlRaw  []byte
	yamlErr  error
}

func (s *docsSource) jsonBody() ([]byte, error) {
	s.jsonOnce.Do(func() {
		s.jsonRaw, s.jsonErr = json.Marshal(s.doc)
		if s.jsonErr != nil {
			slog.Error("openapi docs: sérialisation JSON du document", "err", s.jsonErr)
		}
	})
	return s.jsonRaw, s.jsonErr
}

func (s *docsSource) yamlBody() ([]byte, error) {
	s.yamlOnce.Do(func() {
		s.yamlRaw, s.yamlErr = s.doc.YAML()
		if s.yamlErr != nil {
			slog.Error("openapi docs: sérialisation YAML du document", "err", s.yamlErr)
		}
	})
	return s.yamlRaw, s.yamlErr
}
