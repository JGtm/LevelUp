//go:build cgo

// cmd/snapshot-world-leaderboard — capture le classement CSR mondial par
// playlist/saison depuis les pages publiques Halo Waypoint et le persiste en
// append-only dans shared.world_csr_leaderboard_snapshots (vue _latest pour la
// lecture courante).
//
// Source : https://www.halowaypoint.com/halo-infinite/leaderboards/{season}/{playlist}
// (rendu côté serveur, public, sans authentification). NE PAS dépendre du proxy
// tiers sr-nextjs.vercel.app.
//
// IMPORTANT : stopper le serveur API avant de lancer (la shared DB est ouverte
// en RW ; DuckDB interdit deux writers sur Windows).
//
// Usage :
//
//	go run ./cmd/snapshot-world-leaderboard -season csrseason13-2
//	go run ./cmd/snapshot-world-leaderboard -season csrseason13-2 \
//	    -playlists 6233381c-fc96-40b9-b1ff-f6a4de72dd7a -limit 200 -dry-run
//
// Réparation (aucun scrape, cf. restore_best.go) — ré-insère le meilleur lot
// historique là où le lot servi est dégradé, sur UNE saison nommée ('all' refusé) :
//
//	go run ./cmd/snapshot-world-leaderboard -restore-best -season csrseason13-2
//	go run ./cmd/snapshot-world-leaderboard -restore-best -season csrseason13-2 -execute
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/platform/halo"
)

func main() {
	season := flag.String("season", "", "saison CSR ciblée (ex: csrseason13-2) ou 'all' (toutes les saisons Halo Waypoint) — requis")
	sharedDBPath := flag.String("shared-db", "data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb",
		"chemin shared_matches_v2.duckdb (RW — stopper le serveur)")
	playlistsCSV := flag.String("playlists", "",
		"playlist asset IDs séparés par des virgules ; vide = playlists classées actives")
	limit := flag.Int("limit", 100, "nombre max d'entrées par playlist (défaut 100 = profondeur affichée ; 0 = échelle complète)")
	politeMs := flag.Int("polite-ms", 800, "délai poli entre deux pages (ms)")
	dryRun := flag.Bool("dry-run", false, "scrape et affiche les comptes sans écrire")
	titleSlug := flag.String("title", "halo_infinite", "slug du titre (défaut halo_infinite)")
	restoreBest := flag.Bool("restore-best", false,
		"ne scrape RIEN : ré-insère le meilleur lot historique des playlists de -season (saison PRÉCISE requise, 'all' refusé) dont le lot servi est dégradé (dry-run sauf -execute)")
	execute := flag.Bool("execute", false, "avec -restore-best : écrit réellement (sinon simulation)")
	flag.Parse()

	// Enregistre les steps de migration title-owned (halo_infinite) : sans ça,
	// RunForDB n'applique QUE les migrations globales et rate p.ex. add_xuid /
	// create_season_catalog → InsertWorldCSRSnapshot échoue (colonne xuid absente)
	// sur une DB que le serveur n'a pas déjà migrée (backfill hors prod déployée).
	migration.SetTitleStepsProvider(halomigrations.StepsFor)

	closeLogs := logging.InstallCLI(os.Getenv("LEVELUP_REPO_ROOT"))
	defer closeLogs()
	log := slog.Default().With("module", logging.ModuleLeaderboard, "job", "snapshot-world-leaderboard")
	ctx := context.Background()

	// Mode réparation : aucun scrape, le balayage part de ce qui est DÉJÀ en base
	// (cf. restore_best.go). Le périmètre doit être NOMMÉ : une restauration écrit
	// dans la seule archive du classement mondial, on n'y touche pas « partout à la
	// fois » — d'où une saison précise exigée, et 'all' explicitement refusé.
	if *restoreBest {
		s := strings.TrimSpace(*season)
		if s == "" {
			fatal("-restore-best exige -season <saison> (ex: -season csrseason13-2) : la restauration écrit, son périmètre doit être explicite")
		}
		if strings.EqualFold(s, "all") {
			fatal("-restore-best refuse -season all : restaurer toutes les saisons d'un coup n'est pas un périmètre explicite ; relancer saison par saison")
		}
		runRestoreMode(ctx, log, restoreOptions{
			sharedDBPath: *sharedDBPath,
			titleSlug:    *titleSlug,
			season:       s,
			execute:      *execute,
		})
		return
	}

	if strings.TrimSpace(*season) == "" {
		fatal("-season est requis (ex: csrseason13-2 ou 'all' pour toutes les saisons)")
	}
	runSnapshotMode(ctx, log, snapshotOptions{
		season:       *season,
		sharedDBPath: *sharedDBPath,
		playlistsCSV: *playlistsCSV,
		limit:        *limit,
		politeMs:     *politeMs,
		dryRun:       *dryRun,
		titleSlug:    *titleSlug,
	})
}

// resolveSeasons : 'all' (ou vide) → toutes les saisons du catalogue Halo Waypoint
// (csrseason3-1 → active, récentes d'abord) via FetchCatalog ; sinon la saison
// fournie telle quelle. refPlaylist = une playlist classée valide quelconque (sert
// juste à rendre la page qui porte le menu des saisons).
func resolveSeasons(ctx context.Context, scraper *halo.LeaderboardScraper, season, refPlaylist string) ([]string, error) {
	if s := strings.TrimSpace(season); s != "" && s != "all" {
		return []string{s}, nil
	}
	refs, _, err := scraper.FetchCatalog(ctx, refPlaylist)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		out = append(out, r.ID)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("aucune saison exposée par Halo Waypoint (markup changé ?)")
	}
	return out, nil
}

// resolvePlaylists retourne les asset IDs depuis le CSV fourni, sinon les
// playlists classées actives.
func resolvePlaylists(csv string) []string {
	if s := strings.TrimSpace(csv); s != "" {
		var out []string
		for _, p := range strings.Split(s, ",") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	var out []string
	for _, pl := range rankedplaylists.Active() {
		out = append(out, pl.AssetID)
	}
	return out
}

// openSharedRW ouvre la shared DB en écriture (1 seule connexion).
func openSharedRW(path string) (*sql.DB, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

func dryRunSuffix(dry bool) string {
	if dry {
		return " [dry-run]"
	}
	return ""
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
	os.Exit(1)
}
