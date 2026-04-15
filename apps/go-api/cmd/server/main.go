// cmd/server — point d'entrée du backend Go LevelUp.
//
// Sprint 0 : POC DuckDB + HTTP.
// Ce binaire démarre un serveur HTTP minimal qui :
//   - ouvre metadata.duckdb et shared_matches_v2.duckdb en read-only
//   - expose GET /health (nb de matchs + version DuckDB)
//   - expose GET /api/v1/bootstrap (réponse structurée, parité Python cible)
//   - expose GET /api/v1/players
//
// Variables d'environnement utiles (sprint 0) :
//   LEVELUP_REPO_ROOT    — racine du repo (par défaut : auto-détection)
//   LEVELUP_API_PORT     — port d'écoute (défaut : 8000)
//   LEVELUP_DEMO_MODE    — "true" pour activer le mode démo
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"levelup/go-api/internal/api"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/service"
)

func main() {
	// --- 1. Logging structuré ---
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	// --- 2. Configuration ---
	cfg, err := config.Load()
	if err != nil {
		slog.Error("chargement config", "err", err)
		os.Exit(1)
	}
	slog.Info("config chargée",
		"repo_root", cfg.RepoRoot,
		"demo_mode", cfg.DemoMode,
		"addr", cfg.ServerAddr(),
	)

	// --- 3. Connexions DuckDB ---
	sharedPath := filepath.Join(cfg.RepoRoot, "data", "warehouse", "shared_matches_v2.duckdb")
	metaPath := filepath.Join(cfg.RepoRoot, "data", "warehouse", "metadata.duckdb")

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

	slog.Info("ouverture DuckDB", "shared", sharedPath, "metadata", metaPath)

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
	slog.Info("DuckDB ouvert ✓")

	// --- 4. Repositories + services ---
	bootRepo := duckdb.NewBootstrapRepo(sharedDB, metaDB)
	bootSvc := service.NewBootstrapService(cfg, bootRepo)

	// --- 5. Sprint 0 : validation des types critiques ---
	ctx := context.Background()
	if err := bootRepo.ValidateTypes(ctx); err != nil {
		slog.Error("validation types DuckDB échouée", "err", err)
		os.Exit(1)
	}
	slog.Info("types DuckDB validés (UBIGINT/TIMESTAMPTZ/BOOLEAN) ✓")

	careerCount, err := bootRepo.GetCareerRanksSample(ctx)
	if err != nil {
		slog.Warn("lecture career_ranks échouée", "err", err)
	} else {
		slog.Info("metadata.duckdb lisible ✓", "career_ranks_count", careerCount)
	}

	// --- 6. Routeur HTTP ---
	router := api.NewRouter(bootRepo, bootSvc)

	srv := &http.Server{
		Addr:         cfg.ServerAddr(),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// --- 7. Démarrage + graceful shutdown ---
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("serveur démarré", "addr", cfg.ServerAddr())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("ListenAndServe", "err", err)
			os.Exit(1)
		}
	}()

	<-sigCh
	slog.Info("arrêt gracieux en cours…")

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
	slog.Info("bye.")
	fmt.Println()
}

func strPtr(s string) *string { return &s }
