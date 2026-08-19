// cmd/variant-probe — auth.go : jetons Halo par le chemin canonique du projet
// (ADR 0023) — auth.RefreshHaloTokensViaStoreFirst sur le MultiUserTokenStore.
// AUCUNE re-capture : si le refresh token est mort, on échoue bruyamment.
//
// PORTAGE ASSUMÉ de cmd/mapobj-build/auth.go (2e copie du patron ; à la 3e,
// centraliser dans un helper partagé — noté en découverte du plan de sonde).
// Comme là-bas, on NE passe PAS par config.ResolvePlayer : ResolvePlayer ouvre la
// DB joueur en écriture, interdit tant que le serveur la tient (ADR 0013).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	authpkg "levelup/go-api/internal/platform/auth"
)

func resolveTokens(ctx context.Context, cfg *config.AppConfig, playerSlug, titleSlug string) (*domain.HaloTokens, error) {
	if envToken := os.Getenv("SPARTAN_TOKEN"); envToken != "" {
		slog.InfoContext(ctx, "variant-probe: jeton Spartan fourni par l'environnement")
		return &domain.HaloTokens{
			SpartanToken:   envToken,
			ClearanceToken: os.Getenv("CLEARANCE_TOKEN"),
		}, nil
	}
	if playerSlug == "" {
		return nil, fmt.Errorf("SPARTAN_TOKEN absent ET --player non fourni")
	}

	xuid, gamertag, err := lookupPlayer(cfg, playerSlug, titleSlug)
	if err != nil {
		return nil, err
	}

	// Store seul, sans repli sur la variable d'environnement legacy : le sentinel
	// ADR 0023 interdit tout NOUVEAU lecteur de cette variable.
	store := authpkg.NewMultiUserTokenStore(titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())

	result, err := authpkg.RefreshHaloTokensViaStoreFirst(
		ctx, store, authpkg.NewSISUProvider(), xuid, gamertag, authpkg.LegacyAuthInputs{})
	if err != nil {
		return nil, err
	}
	tokens := authpkg.HaloTokensFromExchange(result)
	if tokens == nil || tokens.SpartanToken == "" {
		return nil, fmt.Errorf(
			"aucun jeton exploitable pour %q (xuid %s) — diagnostiquer la chaîne de refresh, ne PAS re-capturer",
			playerSlug, xuid)
	}
	slog.InfoContext(ctx, "variant-probe: jetons obtenus", "xuid", xuid, "gamertag", gamertag,
		"clearance", tokens.ClearanceToken != "")
	return tokens, nil
}

// lookupPlayer résout gamertag → (xuid, gamertag) via db_profiles.json, sans
// ouvrir la moindre base.
func lookupPlayer(cfg *config.AppConfig, slug, titleSlug string) (string, string, error) {
	players, err := cfg.LoadPlayers(titleSlug)
	if err != nil {
		return "", "", fmt.Errorf("lecture db_profiles.json: %w", err)
	}
	for _, p := range players {
		if strings.EqualFold(p.PlayerSlug, slug) || strings.EqualFold(p.Gamertag, slug) {
			if p.XUID == "" {
				return "", "", fmt.Errorf("joueur %q déclaré sans xuid dans db_profiles.json", slug)
			}
			return p.XUID, p.Gamertag, nil
		}
	}
	return "", "", fmt.Errorf("joueur %q absent de db_profiles.json (titre %s)", slug, titleSlug)
}
