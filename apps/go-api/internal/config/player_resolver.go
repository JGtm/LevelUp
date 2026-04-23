// Package config — player_resolver.go : résolution slug → PlayerDB.
package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

// ErrPlayerNotFound est retourné quand le slug ne correspond à aucun joueur configuré.
var ErrPlayerNotFound = fmt.Errorf("joueur introuvable")

// ResolvePlayer traduit un slug joueur en PlayerDB prêt à l'emploi.
// titleSlug est le titre courant (ex: "halo_infinite"). Si vide, DefaultSlug est utilisé.
// En mode démo, les fixtures de test sont utilisées (slug ignoré).
func ResolvePlayer(ctx context.Context, cfg *AppConfig, slug, titleSlug string) (*duckdb.PlayerDB, error) {
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	if cfg.DemoMode {
		return resolveDemoPlayer(ctx, cfg, titleSlug)
	}
	return resolveRealPlayer(ctx, cfg, slug, titleSlug)
}

// resolveDemoPlayer ouvre le joueur de démo depuis DemoFixturesDir.
func resolveDemoPlayer(ctx context.Context, cfg *AppConfig, titleSlug string) (*duckdb.PlayerDB, error) {
	dir := cfg.DemoFixturesDir
	xuidBytes, err := readXUIDFile(filepath.Join(dir, "xuid.txt"))
	if err != nil {
		// Fallback : XUID hardcodé pour la fixture Chocoboflor
		xuidBytes = "2535469190789936"
	}
	pcfg := duckdb.PlayerPoolConfig{
		Gamertag:           "DemoPlayer",
		XUID:               xuidBytes,
		TitleSlug:          titleSlug,
		PlayerDBPath:       filepath.Join(dir, "stats.duckdb"),
		SharedDBPath:       filepath.Join(dir, "shared_matches_v2.duckdb"),
		MetaDBPath:         filepath.Join(dir, "metadata.duckdb"),
		SharedSocialDBPath: title.NewPathResolver(cfg.RepoRoot).SharedSocialDBPath(titleSlug),
		UserTimezone:       cfg.UserTimezone,
	}
	return duckdb.GetOrOpen(ctx, pcfg)
}

// resolveRealPlayer trouve le joueur par slug dans db_profiles.json.
func resolveRealPlayer(ctx context.Context, cfg *AppConfig, slug, titleSlug string) (*duckdb.PlayerDB, error) {
	players, err := cfg.LoadPlayers(titleSlug)
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

	pcfg := buildPoolConfig(cfg, found, titleSlug)
	return duckdb.GetOrOpen(ctx, pcfg)
}

// buildPoolConfig construit un PlayerPoolConfig depuis un PlayerSummary.
// Utilise le PathResolver pour résoudre les chemins title-aware.
func buildPoolConfig(cfg *AppConfig, p *domain.PlayerSummary, titleSlug string) duckdb.PlayerPoolConfig {
	pr := title.NewPathResolver(cfg.RepoRoot)
	return duckdb.PlayerPoolConfig{
		Gamertag:           p.Gamertag,
		XUID:               p.XUID,
		TitleSlug:          titleSlug,
		PlayerDBPath:       pr.PlayerDBPath(titleSlug, p.Gamertag),
		SharedDBPath:       pr.SharedDBPath(titleSlug),
		MetaDBPath:         pr.MetadataDBPath(titleSlug),
		SharedSocialDBPath: pr.SharedSocialDBPath(titleSlug),
		UserTimezone:       cfg.UserTimezone,
	}
}

// SharedDBPath retourne le chemin vers la base DuckDB partagée.
// titleSlug : titre courant, vide = halo_infinite.
func SharedDBPath(cfg *AppConfig, titleSlug string) string {
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	if cfg.DemoMode {
		return filepath.Join(cfg.DemoFixturesDir, "shared_matches_v2.duckdb")
	}
	return title.NewPathResolver(cfg.RepoRoot).SharedDBPath(titleSlug)
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
