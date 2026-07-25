// Package api_test — helpers de test pour contract_test.go (Sprint 29).
//
// buildTestRouter crée un routeur chi minimal en mode démo.
// Nécessite CGO=1 (dépendance transitive platform/duckdb via handlers).

//go:build cgo

package api_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"levelup/go-api/internal/api"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/groupstore"
	"levelup/go-api/internal/service"
)

// ---------------------------------------------------------------------------
// mockBootstrapRepo — implémente port.BootstrapRepository pour les tests de contrat.
// Retourne des valeurs fixes sans accès DuckDB.
// ---------------------------------------------------------------------------

type mockBootstrapRepo struct{}

func (m *mockBootstrapRepo) GetMatchCount(_ context.Context) (int, error) {
	return 0, nil
}

func (m *mockBootstrapRepo) GetDBVersion(_ context.Context) (string, error) {
	return "test-mock", nil
}

func (m *mockBootstrapRepo) GetPlayerCount(_ context.Context) (int, error)       { return 0, nil }
func (m *mockBootstrapRepo) GetLastSyncAt(_ context.Context) (*time.Time, error) { return nil, nil }

// ---------------------------------------------------------------------------
// buildTestRouter — construit le routeur chi en mode démo pour les tests de contrat.
//
// Caractéristiques :
//   - DemoMode = true → pas d'accès DuckDB
//   - SessionDir pointe vers un répertoire temporaire (nettoyé après le test)
//   - AppSettingsPath pointe vers un fichier JSON minimal
//   - mockBootstrapRepo → les endpoints /health et /bootstrap n'ont pas besoin de DB
// ---------------------------------------------------------------------------

func buildTestRouter(t *testing.T) http.Handler {
	t.Helper()
	router, _ := buildTestRouterWithDoc(t)
	return router
}

// buildTestRouterWithDoc construit le routeur de test ET retourne le document
// OpenAPI PARTAGÉ (V72-01 / H1) pour les tests qui vérifient la fidélité du
// document fusionné (cf. shared_openapi_doc_test.go).
func buildTestRouterWithDoc(t *testing.T) (http.Handler, *huma.OpenAPI) {
	t.Helper()

	// Répertoire temporaire isolé pour les I/O du routeur (nettoyé automatiquement)
	tmpDir := t.TempDir()

	// app_settings.json minimal — LoadAppSettings tolère un objet vide
	appSettingsPath := filepath.Join(tmpDir, "app_settings.json")
	writeTestFile(t, appSettingsPath, []byte(`{}`))

	// Répertoire de sessions (session_platform.NewStore requiert un répertoire valide)
	sessionDir := filepath.Join(tmpDir, "sessions")
	mkdirTest(t, sessionDir)

	// Répertoire data/cache pour jobs_platform.NewStore
	cacheDir := filepath.Join(tmpDir, "data", "cache")
	mkdirTest(t, cacheDir)

	cfg := &config.AppConfig{
		RepoRoot:        tmpDir,
		DBProfilesPath:  filepath.Join(tmpDir, "db_profiles.json"), // absent intentionnellement → liste vide
		AppSettingsPath: appSettingsPath,
		SessionDir:      sessionDir,
		DemoMode:        true, // CRITIQUE : évite tout accès DuckDB
		DemoFixturesDir: tmpDir,
		APIHost:         "127.0.0.1",
		APIPort:         8000,
		// SessionSecret différent de "CHANGE_ME_IN_PRODUCTION" → isProduction=true dans NewRouter,
		// ce qui force des cookies Secure. Pour les tests httptest on reste en non-production.
		SessionSecret:        "CHANGE_ME_IN_PRODUCTION", // pragma: allowlist secret
		CORSOrigins:          []string{},
		Lang:                 "fr",
		MultiTitleAPIEnabled: parseBoolEnvFlag("MULTI_TITLE_API_ENABLED", false),
		PrestigeEnabled:      parseBoolEnvFlag("PRESTIGE_ENABLED", true),
	}

	bootRepo := &mockBootstrapRepo{}
	bootSvc := service.NewBootstrapService(cfg, bootRepo)

	gs := groupstore.NewGroupStore(filepath.Join(os.TempDir(), "levelup_contract_groups.json"))
	router, _, doc := api.NewRouter(context.Background(), cfg, bootRepo, bootSvc, nil, nil, nil, nil, gs)
	return router, doc
}

// ---------------------------------------------------------------------------
// Helpers de setup I/O
// ---------------------------------------------------------------------------

func writeTestFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("writeTestFile(%s): %v", path, err)
	}
}

func mkdirTest(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdirTest(%s): %v", path, err)
	}
}

// parseBoolEnvFlag lit une variable d'environnement comme booléen.
// Retourne defaultVal si la variable est absente.
// Valeurs falsy reconnues : "0", "false", "no", "off".
// Valeurs truthy reconnues : "1", "true", "yes" (et tout le reste si defaultVal=true).
func parseBoolEnvFlag(key string, defaultVal bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return defaultVal
	}
	if defaultVal {
		switch v {
		case "0", "false", "no", "off":
			return false
		}
		return true
	}
	return v == "1" || v == "true" || v == "yes"
}
