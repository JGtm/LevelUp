// Outil ops : sync initial LIVE Halo 5 pour un joueur (équivalent CLI de
// POST /sync/initial title_slug=halo_5). RÉUTILISE le câblage exact (livesync.
// RunnerForTitle) + la résolution de token testée (RefreshHaloTokensViaStoreFirst,
// comme cmd/probe-h5) — PAS de réinvention. Persiste dans le shared h5 + vérifie.
//
// Usage : LEVELUP_REPO_ROOT=<repo principal> go run ./cmd/h5-sync [Gamertag] [maxMatches]
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
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
	maxMatches := 25
	if len(os.Args) > 2 {
		if n, err := strconv.Atoi(os.Args[2]); err == nil && n > 0 {
			maxMatches = n
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

	// Token store-first (identique à probe-h5) → ctx (l'adapter h5 lit le token du ctx).
	store := auth.NewMultiUserTokenStore(titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	res, err := auth.RefreshHaloTokensViaStoreFirst(ctx, store, auth.NewSISUProvider(), xuid, gt, auth.LegacyAuthInputs{})
	if err != nil || res == nil || res.Tokens == nil {
		fatal("refresh tokens %s: err=%v", gt, err)
	}
	ctx = ctxkeys.WithHaloAuth(ctx, res.Tokens, xuid)
	fmt.Printf("owner=%s xuid=%s spartan_len=%d max_matches=%d\n", gt, xuid, len(res.Tokens.SpartanToken), maxMatches)

	// Provisionner le shared h5 (schéma complet : base + migrations title-owned),
	// comme provisionAdditionalActiveTitles au boot serveur. Idempotent.
	sharedPath := titlePkg.NewPathResolver(cfg.RepoRoot).SharedDBPath(halo5.TitleSlug)
	if err := provisionH5Shared(sharedPath); err != nil {
		fatal("provision shared h5: %v", err)
	}

	// Runner live h5 — EXACTEMENT ce que StartInitialSync sélectionne.
	runner := livesync.RunnerForTitle(halo5.TitleSlug, cfg, gt, xuid)
	if runner == nil {
		fatal("RunnerForTitle(halo_5) = nil (titre non câblé ?)")
	}

	syncRes, err := runner.RunDelta(ctx, domain.SyncOptions{MatchType: "all", MaxMatches: maxMatches})
	if err != nil {
		fatal("RunDelta: %v", err)
	}
	fmt.Printf("SYNC: inserted=%d skipped=%d participants=%d medals=%d events=%d status=%s\n",
		syncRes.MatchesInserted, syncRes.MatchesSkipped, syncRes.ParticipantsDone,
		syncRes.MedalsInserted, syncRes.EventsInserted, syncRes.Status())
	for _, e := range syncRes.Errors {
		fmt.Printf("  err: %s\n", e)
	}

	verifyShared(ctx, sharedPath)
}

// provisionH5Shared applique le schéma complet (base EnsureSharedSchema via
// OpenSharedDB + migrations title-owned via RunForTitleDB) au shared h5 — identique
// à provisionAdditionalActiveTitles du boot serveur. Idempotent.
func provisionH5Shared(sharedPath string) error {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	// Open RAW (comme provisionAdditionalTitle) — PAS OpenSharedDB/EnsureSharedSchema,
	// dont la base incomplète masquerait create_base_shared_schema (CREATE IF NOT EXISTS).
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
