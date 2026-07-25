// Package openapigen construit le document OpenAPI PARTAGÉ du serveur (chantier
// V72-01 / H6) et le sérialise en YAML DÉTERMINISTE, fusionné avec le fragment
// manuel versionné (api/openapi_manual_fragment.yaml).
//
// Deux consommateurs, UN seul harnais :
//   - `cmd/openapi-gen` (make openapi-gen) écrit api/openapi.yaml ;
//   - le golden `TestOpenAPIYAMLIsUpToDate` (internal/api) régénère en mémoire et
//     diffe byte-à-byte avec le fichier commité.
//
// Le routeur est assemblé en MODE DÉMO (aucun accès DuckDB) : c'est exactement le
// routeur des tests de contrat (contract_helpers_test.go délègue ici), donc le
// document généré décrit la MÊME surface que celle vérifiée par contract_test.
package openapigen

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"levelup/go-api/internal/api"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/groupstore"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// Options paramètre l'assemblage du routeur de documentation.
type Options struct {
	// Root : répertoire racine ISOLÉ (app_settings.json, sessions/, data/cache/).
	Root string
	// GroupStorePath : chemin du store de groupes JSON (jamais lu en démo).
	GroupStorePath string
	// MultiTitleAPI / Prestige : flags d'exposition des routes conditionnelles.
	MultiTitleAPI bool
	Prestige      bool
}

// stubBootstrapRepo implémente port.BootstrapRepository sans DuckDB (mode démo).
type stubBootstrapRepo struct{}

func (stubBootstrapRepo) GetMatchCount(context.Context) (int, error)        { return 0, nil }
func (stubBootstrapRepo) GetDBVersion(context.Context) (string, error)      { return "test-mock", nil }
func (stubBootstrapRepo) GetPlayerCount(context.Context) (int, error)       { return 0, nil }
func (stubBootstrapRepo) GetLastSyncAt(context.Context) (*time.Time, error) { return nil, nil }

var _ port.BootstrapRepository = stubBootstrapRepo{}

// BuildDemoRouter assemble le routeur chi complet en mode démo et retourne le
// document OpenAPI PARTAGÉ (3e résultat de api.NewRouter, cf. H1).
func BuildDemoRouter(ctx context.Context, opts Options) (http.Handler, *huma.OpenAPI, error) {
	if opts.Root == "" {
		return nil, nil, fmt.Errorf("openapigen: Root vide")
	}
	appSettingsPath := filepath.Join(opts.Root, "app_settings.json")
	if err := os.WriteFile(appSettingsPath, []byte(`{}`), 0o600); err != nil {
		return nil, nil, fmt.Errorf("openapigen: écriture app_settings.json: %w", err)
	}
	sessionDir := filepath.Join(opts.Root, "sessions")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("openapigen: mkdir sessions: %w", err)
	}
	// NB : le cache jobs (RepoRoot/data/cache) est créé à la demande par
	// jobs.Store.save() — rien à pré-créer ici (et le ratchet L2-2 interdit un
	// filepath.Join(..., "data", ...) hors PathResolver).

	cfg := &config.AppConfig{
		RepoRoot:        opts.Root,
		DBProfilesPath:  filepath.Join(opts.Root, "db_profiles.json"), // absent → liste vide
		AppSettingsPath: appSettingsPath,
		SessionDir:      sessionDir,
		DemoMode:        true, // CRITIQUE : aucun accès DuckDB
		DemoFixturesDir: opts.Root,
		APIHost:         "127.0.0.1",
		APIPort:         8000,
		// Secret par défaut → isProduction=false (cookies non Secure, cf. NewRouter).
		SessionSecret:        "CHANGE_ME_IN_PRODUCTION", // pragma: allowlist secret
		CORSOrigins:          []string{},
		Lang:                 "fr",
		MultiTitleAPIEnabled: opts.MultiTitleAPI,
		PrestigeEnabled:      opts.Prestige,
	}

	repo := stubBootstrapRepo{}
	router, _, doc := api.NewRouter(ctx, cfg, repo, service.NewBootstrapService(cfg, repo),
		nil, nil, nil, nil, groupstore.NewGroupStore(opts.GroupStorePath))
	if doc == nil {
		return nil, nil, fmt.Errorf("openapigen: document OpenAPI partagé nil")
	}
	return router, doc, nil
}
