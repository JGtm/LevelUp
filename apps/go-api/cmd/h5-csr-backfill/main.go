// Outil ops : backfill des snapshots CSR CARRIÈRE Halo 5 (par playlist arena) pour un
// joueur, dans player_csr_snapshots. Fetch le service record arena via une source
// authentifiée — le token peut être EMPRUNTÉ (LEVELUP_H5_AUTH_AS) pour les joueurs aux
// RT morts (le service record /h5/servicerecords/arena?players={gt} est servi pour
// n'importe quel gamertag avec n'importe quel SpartanToken v4 valide, comme /matches).
//
// Distinct du LUSR (rating social) et de l'enrichment par match. C'est le TIER classé
// officiel (Bronze..Onyx/Champion) par playlist, affiché en carrière/identité.
//
//	Usage : LEVELUP_REPO_ROOT=<repo principal> [LEVELUP_H5_AUTH_AS=<compte sain>] \
//	        go run ./cmd/h5-csr-backfill [Gamertag]
package main

import (
	"context"
	"fmt"
	"os"

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

	// Token store-first du COMPTE D'AUTH (emprunt possible). Le service record est
	// indexé par gamertag cible → un token valide quelconque suffit.
	store := auth.NewMultiUserTokenStore(titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	res, err := auth.RefreshHaloTokensViaStoreFirst(ctx, store, auth.NewSISUProvider(), authXUID, authGT)
	if err != nil || res == nil || res.Tokens == nil {
		fatal("refresh tokens %s (auth_as=%s): err=%v", gt, authGT, err)
	}
	ctx = ctxkeys.WithHaloAuth(ctx, res.Tokens, authXUID)

	src, err := halo5.NewCaptureSource(ctx)
	if err != nil {
		fatal("NewCaptureSource: %v", err)
	}

	playerPath := titlePkg.NewPathResolver(cfg.RepoRoot).PlayerDBPath(halo5.TitleSlug, gt)
	n, err := livesync.PersistArenaCSR(ctx, src, playerPath, gt)
	if err != nil {
		fatal("PersistArenaCSR %s: %v", gt, err)
	}
	fmt.Printf("CSR carrière h5 %s (auth_as=%s) : %d playlists classées persistées (player_csr_snapshots)\n", gt, authGT, n)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
