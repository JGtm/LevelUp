// cmd/server — point d'entrée du backend Go LevelUp.
//
// Sprint 0 : POC DuckDB + HTTP.
// Sprint 4 : CORS, rate-limit, slog JSON, oapi-codegen types.
// Ce binaire démarre un serveur HTTP qui :
//   - ouvre metadata.duckdb et shared_matches_v2.duckdb en read-only
//   - expose GET /health (nb de matchs + version DuckDB)
//   - expose GET /api/v1/bootstrap (réponse structurée, parité Python cible)
//   - expose GET /api/v1/players
//
// Variables d'environnement :
//
//	LEVELUP_REPO_ROOT    — racine du repo (par défaut : auto-détection)
//	LEVELUP_API_PORT     — port d'écoute (défaut : 8000)
//	LEVELUP_DEMO_MODE    — "true" pour activer le mode démo
//	LEVELUP_LOG_JSON     — "true" pour JSON logging (prod), défaut: text (dev)
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"levelup/go-api/internal/api"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/halo"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/service"
)

// version est injectée au build via -ldflags "-X main.version=X.Y.Z".
var version = "dev"

func main() {
	// --- 0. Health-check mode (Docker HEALTHCHECK) ---
	if len(os.Args) == 2 && os.Args[1] == "-health-check" {
		hcClient := &http.Client{Timeout: 5 * time.Second}
		hcDo := func(url string) (*http.Response, error) {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
			if err != nil {
				return nil, err
			}
			return hcClient.Do(req) //nolint:bodyclose // body fermé par le caller via defer resp.Body.Close()
		}
		port := os.Getenv("LEVELUP_API_PORT_OR_DEFAULT")
		resp, err := hcDo("http://127.0.0.1:" + port + "/health") //nolint:bodyclose // body fermé via defer resp.Body.Close()
		if err != nil {
			// Fallback sur le port par défaut 8000
			resp, err = hcDo("http://127.0.0.1:8000/health") //nolint:bodyclose // body fermé via defer resp.Body.Close()
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "health-check failed:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "health-check status: %d\n", resp.StatusCode)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// --- 1. Logging structuré ---
	// En production (LEVELUP_LOG_JSON=true) : JSON. En dev : texte lisible.
	// Niveau par défaut : INFO. Passer LEVELUP_LOG_LEVEL=debug pour activer les logs HTTP 2xx.
	logJSON := strings.ToLower(os.Getenv("LEVELUP_LOG_JSON")) == "true"
	logLevelStr := strings.ToLower(os.Getenv("LEVELUP_LOG_LEVEL"))
	logLevel := slog.LevelInfo
	if logLevelStr == "debug" {
		logLevel = slog.LevelDebug
	} else if logLevelStr == "warn" {
		logLevel = slog.LevelWarn
	} else if logLevelStr == "error" {
		logLevel = slog.LevelError
	}
	var logHandler slog.Handler
	if logJSON {
		logHandler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
	} else {
		logHandler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
	}
	slog.SetDefault(slog.New(logHandler))

	// --- 2. Configuration ---
	cfg, err := config.Load()
	if err != nil {
		slog.Error("chargement config", "err", err)
		os.Exit(1)
	}
	// Injecter la version buildée (ldflags) si LEVELUP_APP_VERSION non défini.
	if cfg.AppVersion == "dev" && version != "dev" {
		cfg.AppVersion = version
	}
	slog.Debug("config chargée",
		"repo_root", cfg.RepoRoot,
		"demo_mode", cfg.DemoMode,
		"version", cfg.AppVersion,
		"addr", cfg.ServerAddr(),
	)

	// --- 3. Connexions DuckDB ---
	sharedPath := filepath.Join(cfg.RepoRoot, "data", "warehouse", "shared_matches_v2.duckdb")
	metaPath := filepath.Join(cfg.RepoRoot, "data", "warehouse", "metadata.duckdb")
	sharedSocialPath := filepath.Join(cfg.RepoRoot, "data", "warehouse", "shared_social.duckdb")

	// En DEMO_MODE, utiliser les fixtures de démo si les DBs prod n'existent pas.
	if cfg.DemoMode {
		demoPaths := []struct{ name, path *string }{
			{name: strPtr("shared"), path: &sharedPath},
			{name: strPtr("metadata"), path: &metaPath},
		}
		for _, dp := range demoPaths {
			if _, err := os.Stat(*dp.path); os.IsNotExist(err) {
				demo := filepath.Join(cfg.DemoFixturesDir, "warehouse", filepath.Base(*dp.path))
				if _, err2 := os.Stat(demo); err2 == nil {
					*dp.path = demo
					slog.Info("demo_mode: utilisation fixture", "db", *dp.name, "path", demo)
				}
			}
		}
	}

	slog.Debug("ouverture DuckDB", "shared", sharedPath, "metadata", metaPath, "shared_social", sharedSocialPath)

	// --- 3a. Migrations (read-write, avant l'ouverture read-only) ---
	if err := runMigrations(metaPath, sharedPath, sharedSocialPath, cfg); err != nil {
		slog.Debug("migrations ignorées (DB verrouillée), démarrage sans migration")
	} else {
		slog.Debug("migrations appliquées")
	}

	sharedDB, err := duckdb.OpenReadOnly(sharedPath)
	if err != nil {
		slog.Error("ouverture shared_matches_v2 échouée", "err", err)
		os.Exit(1)
	}
	metaDB, err := duckdb.OpenReadOnly(metaPath)
	if err != nil {
		slog.Error("ouverture metadata échouée", "err", err)
		os.Exit(1)
	}
	slog.Debug("DuckDB ouvert")

	// --- 4. Repositories + services ---
	bootRepo := duckdb.NewBootstrapRepo(sharedDB, metaDB)
	bootSvc := service.NewBootstrapService(cfg, bootRepo).
		WithPrivacyProvider(halo.DefaultHaloProvider)

	// Auth locale : injecter le check "first launch" si mode password.
	if cfg.AuthMode == "password" {
		usersPath := filepath.Join(cfg.AuthDir, "users.json")
		us := userstore.NewStore(usersPath)
		bootSvc = bootSvc.WithUserStoreEmpty(us.IsEmpty)
	}

	// --- 5. Sprint 0 : validation des types critiques ---
	ctx := context.Background()
	if err := bootRepo.ValidateTypes(ctx); err != nil {
		slog.Error("validation types DuckDB échouée", "err", err)
		os.Exit(1)
	}
	if _, err := bootRepo.GetCareerRanksSample(ctx); err != nil {
		slog.Warn("lecture career_ranks échouée", "err", err)
	}

	// --- 6. Routeur HTTP ---
	router := api.NewRouter(cfg, bootRepo, bootSvc)

	srv := &http.Server{
		Addr:         cfg.ServerAddr(),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// --- 7. Démarrage + graceful shutdown ---
	// On bind le port en premier pour détecter immédiatement un conflit.
	ln, err := net.Listen("tcp", cfg.ServerAddr())
	if err != nil {
		fmt.Fprintf(os.Stderr, "  [ERR] Port %s deja occupe -- fermez l'ancien processus\n", cfg.ServerAddr())
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	fmt.Fprintf(os.Stderr, "\n  [OK] LevelUp API ready -> http://%s\n\n", cfg.ServerAddr())
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-sigCh
	fmt.Fprint(os.Stderr, "\n  [..] Arret en cours...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "err", err)
	}
	if err := sharedDB.Close(); err != nil {
		slog.Warn("fermeture shared DB", "err", err)
	}
	if err := metaDB.Close(); err != nil {
		slog.Warn("fermeture metadata DB", "err", err)
	}
	fmt.Fprintln(os.Stderr, " terminé.")
}

func strPtr(s string) *string { return &s }

// runMigrations applique les migrations DuckDB dans l'ordre :
// metadata → shared → shared_pve → shared_social.
// Les migrations player sont gérées à l'ouverture de chaque player DB.
func runMigrations(metaPath, sharedPath, sharedSocialPath string, cfg *config.AppConfig) error {
	// Ensure all step init() have been registered (side-effect imports).
	_ = migration.All()

	// 1. metadata.duckdb
	metaDB, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		return fmt.Errorf("open metadata rw: %w", err)
	}
	if err := migration.RunForDB(metaDB.SQLDb(), migration.TargetMetadata); err != nil {
		metaDB.Close()
		return fmt.Errorf("metadata migrations: %w", err)
	}
	metaDB.Close()

	// 2. shared_matches_v2.duckdb
	sharedDB, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		return fmt.Errorf("open shared rw: %w", err)
	}
	if err := migration.RunForDB(sharedDB.SQLDb(), migration.TargetShared); err != nil {
		sharedDB.Close()
		return fmt.Errorf("shared migrations: %w", err)
	}
	sharedDB.Close()

	// 3. shared_pve.duckdb (optionnel, fichier peut ne pas exister)
	pvePath := filepath.Join(cfg.RepoRoot, "data", "warehouse", "shared_pve.duckdb")
	if _, err := os.Stat(pvePath); err == nil {
		pveDB, err := duckdb.OpenReadWrite(pvePath)
		if err != nil {
			return fmt.Errorf("open shared_pve rw: %w", err)
		}
		if err := migration.RunForDB(pveDB.SQLDb(), migration.TargetSharedPvE); err != nil {
			pveDB.Close()
			return fmt.Errorf("shared_pve migrations: %w", err)
		}
		pveDB.Close()
	}

	// 4. shared_social.duckdb (créé automatiquement s'il n'existe pas)
	socialDB, err := duckdb.OpenReadWrite(sharedSocialPath)
	if err != nil {
		return fmt.Errorf("open shared_social rw: %w", err)
	}
	if err := migration.RunForDB(socialDB.SQLDb(), migration.TargetSharedSocial); err != nil {
		socialDB.Close()
		return fmt.Errorf("shared_social migrations: %w", err)
	}
	socialDB.Close()

	return nil
}

// RunPlayerMigrations applique les migrations player pour une DB individuelle.
// Appelé lors de l'ouverture d'une player DB.
func RunPlayerMigrations(playerDBPath string) error {
	db, err := duckdb.OpenReadWrite(playerDBPath)
	if err != nil {
		return fmt.Errorf("open player rw: %w", err)
	}
	defer db.Close()
	return migration.RunForDB(db.SQLDb(), migration.TargetPlayer)
}
