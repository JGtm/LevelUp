// Outil ops : backfill de la MÉCANIQUE de kill Halo 5 (weapon_kills.kill_kind) sur les
// matchs DÉJÀ collectés AVANT la capture kill_kind (colonne NULL sur tout l'historique).
// Re-fetch /events PAR MATCH (endpoint par-match → un seul token sain couvre tout
// l'historique partagé), re-dérive weapon_kills COMPLET (tous les kills du match, avec
// kill_kind) et l'INSÈRE en NOUVELLE GÉNÉRATION (INSERT-only, la vue v_weapon_kills ne
// lit que la génération MAX par (match_id, xuid) → ART-safe, ADR 0026/0030).
//
// Token EMPRUNTÉ (LEVELUP_H5_AUTH_AS, défaut JGtm = RT sain). Écriture offline
// single-writer (serveur arrêté). Idempotent : h5KillKind renvoie toujours une valeur non
// vide → après backfill le match sort de la sélection (2e passe = skip).
//
//	Usage : LEVELUP_REPO_ROOT=<repo principal> [LEVELUP_H5_AUTH_AS=<sain>] [LEVELUP_KK_FORCE=1] \
//	        go run ./cmd/h5-kill-kind-backfill [Gamertag-auth] [maxMatches] [force]
//
// force (3e arg "force"/"1"/"true" ou LEVELUP_KK_FORCE=1) : re-dérive TOUS les matchs (déjà
// backfillés inclus) — requis pour capter une NOUVELLE valeur kill_kind (ex. assassination)
// sur les matchs déjà passés en 4 valeurs. Sans force : seulement les kill_kind NULL.
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
	// force : re-dérive TOUS les matchs (déjà backfillés inclus) pour capter une nouvelle
	// valeur kill_kind (ex. assassination). 3e arg "force"/"1"/"true" ou LEVELUP_KK_FORCE=1.
	force := false
	if len(os.Args) > 3 {
		switch os.Args[3] {
		case "force", "1", "true":
			force = true
		}
	}
	if v := os.Getenv("LEVELUP_KK_FORCE"); v == "1" || v == "true" {
		force = true
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
	res, err := auth.RefreshHaloTokensViaStoreFirst(ctx, store, auth.NewSISUProvider(), authXUID, authGT, auth.LegacyAuthInputs{})
	if err != nil || res == nil || res.Tokens == nil {
		fatal("refresh tokens (auth_as=%s): %v", authGT, err)
	}
	ctx = ctxkeys.WithHaloAuth(ctx, res.Tokens, authXUID)
	src, err := halo5.NewCaptureSource(ctx)
	if err != nil {
		fatal("NewCaptureSource: %v", err)
	}

	sharedPath := titlePkg.NewPathResolver(cfg.RepoRoot).SharedDBPath(halo5.TitleSlug)
	// Provisionner le schéma shared h5 (incl. weapon_kills.kill_kind + vue v_weapon_kills)
	// — idempotent, identique à h5-events-backfill. Ouvre+migre+ferme AVANT le RW.
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
	fmt.Printf("kill-kind-backfill h5 (auth_as=%s, max=%d, force=%v) : démarrage\n", authGT, maxMatches, force)
	stats, err := livesync.RunKillKindBackfill(ctx, shared, fetch, halo5.TitleSlug, maxMatches, force, nil)
	if err != nil {
		fatal("RunKillKindBackfill: %v", err)
	}
	fmt.Printf("kill-kind-backfill h5 : matchs=%d maj=%d vides=%d fetch_err=%d write_err=%d kill_rows=%d\n",
		stats.Matches, stats.Updated, stats.Empty, stats.FetchErr, stats.WriteErr, stats.KillRows)
}

// provisionH5Shared applique le schéma shared complet (base + migrations title-owned via
// RunForTitleDB, incl. weapon_kills append-only + kill_kind + vue v_weapon_kills) au
// shared h5 — identique à h5-events-backfill. Idempotent. Ouvre/migre/ferme : le backfill
// rouvre ensuite le shared (now-migré).
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
