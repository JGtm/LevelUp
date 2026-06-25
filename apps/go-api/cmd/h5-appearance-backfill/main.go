// Outil ops : backfill de l'IDENTITÉ SPARTAN Halo 5 (service tag + rendu Spartan +
// emblème) pour un joueur, dans career_progression (player DB h5, append-only). Sert
// le bloc identitaire du Home (qui n'affichait AUCUNE image Spartan / emblème /
// service tag pour Halo 5 avant ce backfill).
//
// Fetch l'appearance + télécharge les PNG rendus (Spartan + emblème) depuis
// haloplayer via une source authentifiée — le token peut être EMPRUNTÉ
// (LEVELUP_H5_AUTH_AS) : les endpoints /h5/profiles/{gt}/{appearance,spartan,emblem}
// sont indexés par gamertag et servis pour n'importe quel SpartanToken v4 valide
// (comme /matches et /servicerecords).
//
//	Usage : LEVELUP_REPO_ROOT=<repo principal> [LEVELUP_H5_AUTH_AS=<compte sain>] \
//	        go run ./cmd/h5-appearance-backfill [Gamertag]
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/games/halo_5/livesync"
	"levelup/go-api/internal/platform/auth"
)

func main() {
	gt := "JGtm"
	if len(os.Args) > 1 {
		gt = os.Args[1]
	}
	authGT := gt
	if v := os.Getenv("LEVELUP_H5_AUTH_AS"); v != "" {
		authGT = v
	}

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		fatal("config.Load: %v", err)
	}

	// Résolution xuid robuste (halo_5 puis global) pour la cible ET le compte d'auth.
	findXUID := func(who string) string {
		for _, slug := range []string{halo5.TitleSlug, ""} {
			ps, e := cfg.LoadPlayers(slug)
			if e != nil {
				continue
			}
			for i := range ps {
				if ps[i].Gamertag == who {
					return ps[i].XUID
				}
			}
		}
		return ""
	}
	xuid := findXUID(gt)
	authXUID := findXUID(authGT)
	if xuid == "" {
		fatal("xuid introuvable pour %q dans db_profiles", gt)
	}
	if authXUID == "" {
		fatal("xuid auth introuvable pour %q (LEVELUP_H5_AUTH_AS)", authGT)
	}

	// Token store-first du COMPTE D'AUTH (emprunt possible). Appearance indexée par
	// gamertag cible → un token valide quelconque suffit.
	store := auth.NewMultiUserTokenStore(titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	res, err := auth.RefreshHaloTokensViaStoreFirst(ctx, store, auth.NewMSALProvider(), authXUID, authGT, auth.LegacyAuthInputs{})
	if err != nil || res == nil || res.Tokens == nil {
		fatal("refresh tokens %s (auth_as=%s): err=%v", gt, authGT, err)
	}
	ctx = ctxkeys.WithHaloAuth(ctx, res.Tokens, authXUID)

	src, err := halo5.NewAppearanceSource(ctx)
	if err != nil {
		fatal("NewAppearanceSource: %v", err)
	}

	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	playerPath := resolver.PlayerDBPath(halo5.TitleSlug, gt)
	cacheRoot := filepath.Join(resolver.RepoRoot(), "data", "cache")

	out, err := livesync.PersistAppearance(ctx, src, playerPath, cacheRoot, gt, xuid)
	if err != nil {
		fatal("PersistAppearance %s: %v", gt, err)
	}
	fmt.Printf("appearance h5 %s (auth_as=%s) : service_tag=%q spartan_render=%v emblem=%v persisted=%v\n",
		gt, authGT, out.ServiceTag, out.SpartanRendered, out.EmblemRendered, out.Persisted)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
