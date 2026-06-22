// Outil ops : backfill HISTORIQUE COMPLET LIVE Halo 5 pour un joueur — incrémental
// et résumable. Distinct de cmd/h5-sync (qui reste le sync DELTA, delta-stop au 1er
// connu). Ici on paginne TOUT l'historique (2016→aujourd'hui), on PERSISTE CHAQUE
// PAGE immédiatement (lease RW court) et on SAUTE les matchs déjà connus sans
// s'arrêter → relancer après interruption reprend sans doublon (INSERT-only).
//
// RÉUTILISE le câblage exact de cmd/h5-sync (RefreshHaloTokensViaStoreFirst →
// ctxkeys.WithHaloAuth → provisionH5Shared via RunForTitleDB) + les briques du runner
// (livesync.BuildBackfillDeps : known-set + persist + resolver). PAS de réinvention.
//
// Usage : LEVELUP_REPO_ROOT=<repo principal> go run ./cmd/h5-backfill [Gamertag] [pageSize]
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/games/halo_5/livesync"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
)

func main() {
	gt := "JGtm"
	if len(os.Args) > 1 {
		gt = os.Args[1]
	}
	pageSize := 25
	if len(os.Args) > 2 {
		if n, err := strconv.Atoi(os.Args[2]); err == nil && n > 0 {
			pageSize = n
		}
	}

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		fatal("config.Load: %v", err)
	}
	players, err := cfg.LoadPlayers()
	if err != nil {
		fatal("LoadPlayers: %v", err)
	}
	var xuid string
	for i := range players {
		if players[i].Gamertag == gt {
			xuid = players[i].XUID
		}
	}
	if xuid == "" {
		fatal("xuid introuvable pour %q dans db_profiles (déclarer le joueur)", gt)
	}

	// Token store-first (identique à h5-sync/probe-h5) → ctx (l'adapter h5 lit le token du ctx).
	store := auth.NewMultiUserTokenStore(titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	res, err := auth.RefreshHaloTokensViaStoreFirst(ctx, store, auth.NewMSALProvider(), xuid, gt, auth.LegacyAuthInputs{})
	if err != nil || res == nil || res.Tokens == nil {
		fatal("refresh tokens %s: err=%v", gt, err)
	}
	ctx = ctxkeys.WithHaloAuth(ctx, res.Tokens, xuid)
	fmt.Printf("owner=%s xuid=%s spartan_len=%d page_size=%d\n", gt, xuid, len(res.Tokens.SpartanToken), pageSize)

	// Provisionner le shared h5 (schéma complet) — identique à h5-sync. Idempotent.
	sharedPath := titlePkg.NewPathResolver(cfg.RepoRoot).SharedDBPath(halo5.TitleSlug)
	if err := provisionH5Shared(sharedPath); err != nil {
		fatal("provision shared h5: %v", err)
	}

	// Source live construite UNE fois (réseau différé au-delà de l'auth ; token déjà
	// dans le ctx). Le backfill réutilise la MÊME source sur toutes les pages.
	src, err := halo5.NewCaptureSource(ctx)
	if err != nil {
		fatal("NewCaptureSource: %v", err)
	}

	deps := livesync.BuildBackfillDeps(ctx, cfg, src, gt, xuid)
	stats, err := livesync.RunBackfill(ctx, deps, pageSize, nil)
	if err != nil {
		fatal("RunBackfill: %v", err)
	}
	fmt.Printf("BACKFILL: pages=%d seen=%d inserted=%d skipped=%d events_failed=%d carnage_failed=%d warzone=%d persist_errors=%d\n",
		stats.Pages, stats.MatchesSeen, stats.Inserted, stats.Skipped,
		stats.EventsFailed, stats.CarnageFailed, stats.Warzone, stats.PersistErrors)

	verifyShared(ctx, sharedPath)
}

// provisionH5Shared applique le schéma complet (base EnsureSharedSchema via OpenSharedDB
// + migrations title-owned via RunForTitleDB) au shared h5 — identique à h5-sync. Idempotent.
func provisionH5Shared(sharedPath string) error {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	db, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		return err
	}
	defer db.Close()
	return migration.RunForTitleDB(db.SQLDb(), halo5.TitleSlug, migration.TargetShared)
}

// verifyShared compte ce qui a atterri dans le shared h5 (preuve de persistance).
func verifyShared(ctx context.Context, path string) {
	db, release, err := syncpkg.AcquireSharedWriterStandalone(ctx, nil, path)
	if err != nil {
		fmt.Printf("verif shared (%s): %v\n", path, err)
		return
	}
	defer release()
	count := func(q string) int {
		var n int
		_ = db.QueryRowContext(ctx, q).Scan(&n)
		return n
	}
	fmt.Printf("DB h5 shared (%s):\n  match_registry=%d  match_participants=%d  killer_victim_pairs=%d  medals_earned=%d  weapon_kills=%d  kill_positions=%d\n",
		path,
		count("SELECT count(*) FROM match_registry"),
		count("SELECT count(*) FROM match_participants"),
		count("SELECT count(*) FROM killer_victim_pairs"),
		count("SELECT count(*) FROM medals_earned"),
		count("SELECT count(*) FROM weapon_kills"),
		count("SELECT count(*) FROM kill_positions"))
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
