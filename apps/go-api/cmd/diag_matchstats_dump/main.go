// cmd/diag_matchstats_dump — dump du JSON BRUT de GetMatchStats pour un ou plusieurs
// match_id, sur disque, sans toucher a la moindre base.
//
// POURQUOI CET OUTIL EXISTE. Les outils qui re-fetchent GetMatchStats
// (`backfill_objective_stats`, `backfill_kda_accuracy`, `backfill_participation_info`)
// ouvrent tous la shared DB en LECTURE-ECRITURE et ecrivent : ils exigent le serveur
// arrete et ils modifient les donnees. Pour une question de qualite de donnees — « quel
// champ de l'API porte le score reellement affiche par le jeu ? » — il faut le contraire :
// voir le payload ENTIER, ne rien ecrire, et pouvoir tourner pendant que le serveur tient
// les bases. C'est le seul chemin qui repond a ce besoin.
//
// CE QU'IL NE FAIT PAS : aucune ouverture DuckDB, aucun INSERT, aucun UPDATE. Le seul
// effet de bord est celui, inevitable, du chemin d'auth canonique (ADR 0023) : un
// refresh token Microsoft est a USAGE UNIQUE, donc `RefreshHaloTokensViaStoreFirst` en
// obtient un nouveau et le persiste dans le store. Ne pas le persister casserait la
// chaine du porteur. Aucune re-capture de token n'est tentee : si le refresh echoue,
// l'outil s'arrete en le disant.
//
// Usage (depuis apps/go-api/ d'un worktree qui n'a pas de data/ : pointer LEVELUP_REPO_ROOT
// sur le depot qui porte les donnees) :
//
//	LEVELUP_REPO_ROOT=/chemin/vers/LevelUp-go-migration \
//	  go run ./cmd/diag_matchstats_dump --gamertag JGtm --out /tmp/dumps <match_id>...
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	auth_platform "levelup/go-api/internal/platform/auth"
	go_sync "levelup/go-api/internal/sync"
)

func main() {
	gamertag := flag.String("gamertag", "JGtm", "Gamertag dont les tokens servent a l'auth API (doit etre dans db_profiles.json)")
	out := flag.String("out", ".", "Dossier de destination des JSON")
	rps := flag.Int("rps", 3, "Requetes API par seconde")
	flag.Parse()

	matchIDs := flag.Args()
	if len(matchIDs) == 0 {
		fmt.Fprintln(os.Stderr, "usage: diag_matchstats_dump [--gamertag GT] [--out DIR] <match_id>...")
		os.Exit(2)
	}

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)

	xuid, err := resolveXUID(cfg, *gamertag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve xuid %s: %v\n", *gamertag, err)
		os.Exit(1)
	}

	// Auth store-first (ADR 0023) — JAMAIS de re-capture ici.
	store := auth_platform.NewMultiUserTokenStore(pr.WatcherTokensDir())
	provider := auth_platform.NewSISUProvider()
	exch, err := auth_platform.RefreshHaloTokensViaStoreFirst(ctx, store, provider, xuid, *gamertag)
	if err != nil || exch == nil {
		fmt.Fprintf(os.Stderr, "auth %s (xuid %s): %v\n", *gamertag, xuid, err)
		fmt.Fprintln(os.Stderr, "→ diagnostiquer la cause (RT tourne perdu, mauvais xuid) — ne PAS re-capturer de token.")
		os.Exit(1)
	}
	client := go_sync.NewHaloAPIClient(exch.Tokens.SpartanToken, exch.Tokens.ClearanceToken, *rps)

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir %s: %v\n", *out, err)
		os.Exit(1)
	}

	failures := 0
	for _, mid := range matchIDs {
		mid = strings.TrimSpace(mid)
		if mid == "" {
			continue
		}
		mj, ferr := client.GetMatchStats(ctx, mid)
		if ferr != nil {
			slog.ErrorContext(ctx, "diag_matchstats_dump: GetMatchStats echoue", "match_id", mid, "err", ferr)
			failures++
			continue
		}
		blob, merr := json.MarshalIndent(mj, "", "  ")
		if merr != nil {
			slog.ErrorContext(ctx, "diag_matchstats_dump: encodage JSON echoue", "match_id", mid, "err", merr)
			failures++
			continue
		}
		path := filepath.Join(*out, mid+".json")
		if werr := os.WriteFile(path, blob, 0o644); werr != nil {
			slog.ErrorContext(ctx, "diag_matchstats_dump: ecriture echouee", "path", path, "err", werr)
			failures++
			continue
		}
		fmt.Printf("OK %s (%d octets)\n", path, len(blob))
	}
	if failures > 0 {
		os.Exit(1)
	}
}

// resolveXUID lit le xuid d'un gamertag dans db_profiles.json.
func resolveXUID(cfg *config.AppConfig, gamertag string) (string, error) {
	players, err := cfg.LoadPlayers(titlePkg.DefaultSlug)
	if err != nil {
		return "", err
	}
	for i := range players {
		if players[i].Gamertag == gamertag {
			return players[i].XUID, nil
		}
	}
	return "", fmt.Errorf("gamertag %q absent de db_profiles.json", gamertag)
}
