// Package config — player_resolver.go : résolution slug → PlayerDB.
package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
)

// ErrPlayerNotFound est retourné quand le slug ne correspond à aucun joueur configuré.
var ErrPlayerNotFound = fmt.Errorf("joueur introuvable")

// ResolvePlayer traduit un slug joueur en PlayerDB prêt à l'emploi.
// En mode démo, les fixtures de test sont utilisées (slug ignoré).
// Les connexions sont mise en cache dans le pool global duckdb.globalPool.
func ResolvePlayer(ctx context.Context, cfg *AppConfig, slug string) (*duckdb.PlayerDB, error) {
	if cfg.DemoMode {
		return resolveDemoPlayer(ctx, cfg)
	}
	return resolveRealPlayer(ctx, cfg, slug)
}

// resolveDemoPlayer ouvre le joueur de démo depuis DemoFixturesDir.
func resolveDemoPlayer(ctx context.Context, cfg *AppConfig) (*duckdb.PlayerDB, error) {
	dir := cfg.DemoFixturesDir
	xuidBytes, err := readXUIDFile(filepath.Join(dir, "xuid.txt"))
	if err != nil {
		// Fallback : XUID hardcodé pour la fixture Chocoboflor
		xuidBytes = "2535469190789936"
	}
	pcfg := duckdb.PlayerPoolConfig{
		Gamertag:     "DEMO",
		XUID:         xuidBytes,
		PlayerDBPath: filepath.Join(dir, "stats.duckdb"),
		SharedDBPath: filepath.Join(dir, "shared_matches_v2.duckdb"),
		MetaDBPath:   filepath.Join(dir, "metadata.duckdb"),
	}
	return duckdb.GetOrOpen(ctx, pcfg)
}

// resolveRealPlayer trouve le joueur par slug dans db_profiles.json.
func resolveRealPlayer(ctx context.Context, cfg *AppConfig, slug string) (*duckdb.PlayerDB, error) {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return nil, fmt.Errorf("ResolvePlayer: %w", err)
	}

	var found *domain.PlayerSummary
	for i := range players {
		if players[i].PlayerSlug == slug {
			found = &players[i]
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf("%w: slug=%q", ErrPlayerNotFound, slug)
	}

	pcfg := buildPoolConfig(cfg, found)
	return duckdb.GetOrOpen(ctx, pcfg)
}

// buildPoolConfig construit un PlayerPoolConfig depuis un PlayerSummary.
func buildPoolConfig(cfg *AppConfig, p *domain.PlayerSummary) duckdb.PlayerPoolConfig {
	playerDir := filepath.Join(cfg.RepoRoot, "data", "players", p.Gamertag)
	warehouseDir := filepath.Join(cfg.RepoRoot, "data", "warehouse")
	return duckdb.PlayerPoolConfig{
		Gamertag:     p.Gamertag,
		XUID:         p.XUID,
		PlayerDBPath: filepath.Join(playerDir, "stats.duckdb"),
		SharedDBPath: filepath.Join(warehouseDir, "shared_matches_v2.duckdb"),
		MetaDBPath:   filepath.Join(warehouseDir, "metadata.duckdb"),
	}
}

// SharedDBPath retourne le chemin vers la base DuckDB partagée.
func SharedDBPath(cfg *AppConfig) string {
	if cfg.DemoMode {
		return filepath.Join(cfg.DemoFixturesDir, "shared_matches_v2.duckdb")
	}
	return filepath.Join(cfg.RepoRoot, "data", "warehouse", "shared_matches_v2.duckdb")
}

// readXUIDFile lit le XUID depuis un fichier texte (1 ligne).
func readXUIDFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	xuid := strings.TrimSpace(string(data))
	if xuid == "" {
		return "", fmt.Errorf("xuid.txt vide : %s", path)
	}
	return xuid, nil
}
