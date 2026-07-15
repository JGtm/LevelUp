// cmd/diag_live_economy — Test one-shot des endpoints Economy player-gated
// (career progression XP + Spartan customization appearance) pour un joueur.
//
// Reproduit exactement la chaîne d'auth + appels HTTP utilisée par
// CareerLiveService en runtime, sans passer par le serveur HTTP. Utile pour
// auditer en local que la chaîne RT → Spartan → Economy fonctionne pour chaque
// joueur du parc, sans dépendre du frontend.
//
// Usage :
//
//	diag_live_economy <gamertag>
//
// Sortie : 3 lignes
//
//	[auth]          OK / FAIL (XSTS + Spartan token len)
//	[career]        OK rank=N xp=N / NIL / FAIL
//	[customization] OK / NIL / FAIL (NIL = joueur sans Spartan customizé)
package main

import (
	"context"
	"fmt"
	"os"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	authpkg "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: diag_live_economy <gamertag>")
		os.Exit(1)
	}
	gamertag := os.Args[1]

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config.Load: %v\n", err)
		os.Exit(1)
	}
	ctx := context.Background()

	pdb, err := config.ResolvePlayer(ctx, cfg, gamertag, titlePkg.DefaultSlug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ResolvePlayer: %v\n", err)
		os.Exit(1)
	}

	provider := authpkg.NewSISUProvider()
	store := authpkg.NewMultiUserTokenStore(titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	legacy := authpkg.LegacyAuthInputs{Source: "duckdb"}
	legacy.MSALCache, _ = duckdb.ReadMSALCacheJSON(ctx, pdb.Player)
	legacy.OAuthRT, _ = duckdb.ReadOAuthRefreshToken(ctx, pdb.Player)

	result, rerr := authpkg.RefreshHaloTokensViaStoreFirst(ctx, store, provider, pdb.XUID, pdb.Gamertag, legacy)
	if rerr != nil {
		fmt.Fprintf(os.Stderr, "AUTH FAIL: %v\n", rerr)
		os.Exit(1)
	}
	tokens := authpkg.HaloTokensFromExchange(result)
	if tokens == nil {
		fmt.Fprintln(os.Stderr, "AUTH FAIL: tokens nil")
		os.Exit(1)
	}
	fmt.Printf("[auth] OK xuid=%s spartan_len=%d clearance_len=%d\n",
		pdb.XUID, len(tokens.SpartanToken), len(tokens.ClearanceToken))

	client := syncpkg.NewHaloAPIClient(tokens.SpartanToken, tokens.ClearanceToken, 10)

	career, err := client.GetCareerProgress(ctx, pdb.XUID)
	if err != nil {
		fmt.Printf("[career] FAIL: %v\n", err)
	} else if career == nil {
		fmt.Println("[career] NIL (silent skip — 401/403 ou réponse vide)")
	} else {
		fmt.Printf("[career] OK rank=%d name=%q xp=%d (career rank XP live)\n",
			career.CurrentRank, career.CurrentRankName, career.CurrentXP)
	}

	custo, err := client.GetSpartanCustomization(ctx, pdb.XUID)
	if err != nil {
		fmt.Printf("[customization] FAIL: %v\n", err)
	} else if custo == nil {
		fmt.Println("[customization] NIL (silent skip — 401/403 ou réponse vide)")
	} else {
		fmt.Printf("[customization] OK spartan_id=%s banner=%t emblem=%t\n",
			custo.SpartanID, custo.BannerImageURL != "", custo.EmblemImageURL != "")
	}
}
