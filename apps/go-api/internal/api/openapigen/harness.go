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
	// Prestige : flag d'exposition des routes conditionnelles restantes.
	// (MultiTitleAPI retiré le 2026-08-02 — routes multi-titres inconditionnelles.)
	Prestige bool
	// BuildWorkerToken : jeton du protocole ouvrier vu par le routeur assemblé.
	// VIDE PAR DÉFAUT — le document OpenAPI et les tests de contrat sont inchangés
	// (les routes /internal/build-queue/* sont montées quel qu'en soit le jeton ;
	// seule la RÉPONSE du garde change : 503 sans jeton, 401 sur jeton refusé).
	// Renseigné uniquement par le test de traversée CSRF de la pile transverse
	// (csrf_transverse_stack_cgo_test.go, 2026-08-25), qui a besoin d'un serveur
	// « protocole ouvert » pour prouver que l'exemption CSRF ne court-circuite PAS
	// l'authentification par jeton.
	BuildWorkerToken string
	// DesktopSecret : secret d'arrêt propre (POST /api/v1/internal/shutdown). VIDE PAR DÉFAUT : la
	// route n'est pas montée et le contrat est inchangé ; renseigné par le ratchet des routes nues.
	DesktopSecret string
	// ShutdownRequester : reçoit la demande d'arrêt de POST /api/v1/internal/shutdown (nil : la route
	// répond 404 même avec un secret). Renseigné par les tests de la route d'arrêt.
	ShutdownRequester func(reason string)
	// ShellEntryPort : non nul, le routeur est assemblé SOUS l'entrée de la coquille desktop (Host
	// strict, cookie exigé, /desktop/enter, /api/v1/desktop/strings) pour ce port, avec
	// ShellCookieKey comme clé du cookie et DesktopSecret comme secret d'entrée. ZÉRO PAR DÉFAUT :
	// le document OpenAPI et les tests de contrat sont inchangés ; renseigné par les tests de
	// l'entrée de la coquille.
	ShellEntryPort int
	// ShellCookieKey : clé de signature du cookie de la coquille (le secret de session du fichier).
	ShellCookieKey string
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
//
// Surface COMPLÈTE (V721-04) : la racine isolée n'a aucune base DuckDB, ce qui
// faisait auparavant disparaître du contrat 35 routes pourtant montées en
// production (29 Prestige, 3 catalog, 3 diag auto-sync) — leurs dépendances
// échouaient silencieusement et le gate de montage tombait à faux. Le montage est
// désormais satisfaisable en démo via des dépendances de REPLI décidées dans
// api.mountAPIV1 (bundle Prestige nil toléré, handlers.EmptyCatalogRepo,
// ordonnanceur nil + garde 503) — même esprit que stubBootstrapRepo ci-dessus, et
// strictement borné à cfg.DemoMode : la production garde son wiring conditionnel.
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
		SessionSecret:    "CHANGE_ME_IN_PRODUCTION", // pragma: allowlist secret
		CORSOrigins:      []string{},
		Lang:             "fr",
		PrestigeEnabled:  opts.Prestige,
		BuildWorkerToken: opts.BuildWorkerToken,
		DesktopConfig:    config.DesktopConfig{DesktopSecret: opts.DesktopSecret, ShutdownRequester: opts.ShutdownRequester},
	}

	if opts.ShellEntryPort != 0 {
		cfg.APIPort = opts.ShellEntryPort
		cfg.SessionSecret = opts.ShellCookieKey
		cfg.Capabilities.ShellEntryGate = true
		cfg.Capabilities.LoginViaBrowser = true
		cfg.SetBoundPort(opts.ShellEntryPort)
	}

	repo := stubBootstrapRepo{}
	router, _, doc := api.NewRouter(ctx, cfg, repo, service.NewBootstrapService(cfg, repo),
		nil, nil, nil, nil, groupstore.NewGroupStore(opts.GroupStorePath))
	if doc == nil {
		return nil, nil, fmt.Errorf("openapigen: document OpenAPI partagé nil")
	}
	return router, doc, nil
}
