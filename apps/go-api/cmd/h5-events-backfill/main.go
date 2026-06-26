// Outil ops : backfill des surfaces DÉRIVÉES de la timeline d'events Halo 5
// (weapon_accuracy + highlight_events assist/objectif) sur les matchs DÉJÀ collectés
// avant l'ajout de ces surfaces. Le collect normal les saute (idempotence
// match_registry) ; comme personne ne (re)joue ces matchs, il faut re-fetch /events
// (endpoint PAR-MATCH → un seul token sain couvre tout l'historique partagé) et écrire
// uniquement les surfaces dérivées, idempotent (DELETE-puis-INSERT ciblé).
//
// Token EMPRUNTÉ (LEVELUP_H5_AUTH_AS, défaut JGtm = RT sain). Écriture offline
// single-writer (serveur arrêté). Idempotent (re-DELETE+INSERT les mêmes lignes).
//
//	Usage : LEVELUP_REPO_ROOT=<repo principal> [LEVELUP_H5_AUTH_AS=<sain>] \
//	        go run ./cmd/h5-events-backfill [Gamertag-auth] [maxMatches]
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/canonical"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/games/halo_5/livesync"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/duckdb"
)

func main() {
	authGT := "JGtm"
	if len(os.Args) > 1 {
		authGT = os.Args[1]
	}
	if v := os.Getenv("LEVELUP_H5_AUTH_AS"); v != "" {
		authGT = v
	}
	maxMatches := 0
	if len(os.Args) > 2 {
		if n, err := strconv.Atoi(os.Args[2]); err == nil {
			maxMatches = n
		}
	}

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		fatal("config.Load: %v", err)
	}

	authXUID := resolveAuthXUID(cfg, authGT)
	if authXUID == "" {
		fatal("xuid auth introuvable pour %q", authGT)
	}

	store := auth.NewMultiUserTokenStore(titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	res, err := auth.RefreshHaloTokensViaStoreFirst(ctx, store, auth.NewMSALProvider(), authXUID, authGT, auth.LegacyAuthInputs{})
	if err != nil || res == nil || res.Tokens == nil {
		fatal("refresh tokens (auth_as=%s): %v", authGT, err)
	}
	ctx = ctxkeys.WithHaloAuth(ctx, res.Tokens, authXUID)
	src, err := halo5.NewCaptureSource(ctx)
	if err != nil {
		fatal("NewCaptureSource: %v", err)
	}

	sharedPath := titlePkg.NewPathResolver(cfg.RepoRoot).SharedDBPath(halo5.TitleSlug)
	// Provisionner le schéma shared h5 (incl. weapon_accuracy) — idempotent, identique à
	// h5-backfill. Le shared local peut ne pas avoir la migration récente si le serveur
	// n'a pas tourné avec le nouveau code. Ouvre+migre+ferme AVANT le backfill RW.
	if err := provisionH5Shared(sharedPath); err != nil {
		fatal("provision shared h5: %v", err)
	}
	shared, err := sql.Open("duckdb", sharedPath)
	if err != nil {
		fatal("open shared RW: %v", err)
	}
	defer shared.Close()
	shared.SetMaxOpenConns(1)

	fetch := func(ctx context.Context, matchID string) ([]canonical.MatchEvent, error) {
		return halo5.FetchCanonicalEvents(ctx, src, matchID)
	}
	fmt.Printf("events-backfill h5 (auth_as=%s, max=%d) : démarrage\n", authGT, maxMatches)
	stats, err := livesync.RunEventsBackfill(ctx, shared, fetch, maxMatches, nil)
	if err != nil {
		fatal("RunEventsBackfill: %v", err)
	}
	fmt.Printf("events-backfill h5 : matchs=%d maj=%d fetch_err=%d weapon_rows=%d event_rows=%d\n",
		stats.Matches, stats.Updated, stats.FetchErr, stats.WeaponRows, stats.EventRows)
}

// provisionH5Shared applique le schéma shared complet (base + migrations title-owned
// via RunForTitleDB, incl. add_weapon_accuracy) au shared h5 — identique à h5-backfill.
// Idempotent. Ouvre/migre/ferme : le backfill rouvre ensuite le shared (now-migré).
func provisionH5Shared(sharedPath string) error {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	db, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		return err
	}
	defer db.Close()
	return migration.RunForTitleDB(db.SQLDb(), halo5.TitleSlug, migration.TargetShared)
}

// resolveAuthXUID trouve le xuid du gamertag d'auth dans db_profiles (titre h5 puis
// global). "" si introuvable.
func resolveAuthXUID(cfg *config.AppConfig, authGT string) string {
	for _, slug := range []string{halo5.TitleSlug, ""} {
		ps, e := cfg.LoadPlayers(slug)
		if e != nil {
			continue
		}
		for i := range ps {
			if ps[i].Gamertag == authGT {
				return ps[i].XUID
			}
		}
	}
	return ""
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
