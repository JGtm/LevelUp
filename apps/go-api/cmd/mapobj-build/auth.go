// cmd/mapobj-build — auth.go : obtention des jetons Halo par le chemin canonique
// du projet (ADR 0023) — auth.RefreshHaloTokensViaStoreFirst sur le
// MultiUserTokenStore. AUCUNE re-capture de jeton : si le refresh token est mort,
// on échoue bruyamment et on diagnostique.
//
// Note : contrairement à cmd/refresh-metadata, on NE passe PAS par
// config.ResolvePlayer. Cet outil n'a besoin que du xuid, et ResolvePlayer ouvre la
// DB joueur en écriture — interdit tant que le serveur la tient (modèle mono-writer,
// ADR 0013). db_profiles.json suffit et ne verrouille rien.
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
		slog.InfoContext(ctx, "mapobj: jeton Spartan fourni par l'environnement")
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

	store := authpkg.NewMultiUserTokenStore(titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	legacy := authpkg.LegacyAuthInputs{
		OAuthRT: oauthRefreshEnvForPlayer(playerSlug),
		Source:  "env_var",
	}

	result, err := authpkg.RefreshHaloTokensViaStoreFirst(
		ctx, store, authpkg.NewMSALProvider(), xuid, gamertag, legacy)
	if err != nil {
		return nil, err
	}
	tokens := authpkg.HaloTokensFromExchange(result)
	if tokens == nil || tokens.SpartanToken == "" {
		return nil, fmt.Errorf(
			"aucun jeton exploitable pour %q (xuid %s) — diagnostiquer la chaîne de refresh, ne PAS re-capturer",
			playerSlug, xuid)
	}
	slog.InfoContext(ctx, "mapobj: jetons obtenus", "xuid", xuid, "gamertag", gamertag,
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

// oauthRefreshEnvForPlayer construit SPNKR_OAUTH_REFRESH_TOKEN_<GT_NORM>.
func oauthRefreshEnvForPlayer(gamertag string) string {
	key := strings.ToUpper(gamertag)
	key = strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' || r == '.' {
			return '_'
		}
		return r
	}, key)
	return os.Getenv("SPNKR_OAUTH_REFRESH_TOKEN_" + key)
}
