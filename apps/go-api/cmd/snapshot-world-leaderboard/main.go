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
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/halo"
)

func main() {
	season := flag.String("season", "", "saison CSR ciblée (ex: csrseason13-2) — requis")
	sharedDBPath := flag.String("shared-db", "data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb",
		"chemin shared_matches_v2.duckdb (RW — stopper le serveur)")
	playlistsCSV := flag.String("playlists", "",
		"playlist asset IDs séparés par des virgules ; vide = playlists classées actives")
	limit := flag.Int("limit", 200, "nombre max d'entrées par playlist (0 = échelle complète)")
	politeMs := flag.Int("polite-ms", 800, "délai poli entre deux pages (ms)")
	dryRun := flag.Bool("dry-run", false, "scrape et affiche les comptes sans écrire")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	ctx := context.Background()

	if strings.TrimSpace(*season) == "" {
		fatal("-season est requis (ex: csrseason13-2)")
	}
	playlists := resolvePlaylists(*playlistsCSV)
	if len(playlists) == 0 {
		fatal("aucune playlist à traiter")
	}
	fmt.Printf("Saison %s — %d playlist(s), limite %d/playlist%s\n",
		*season, len(playlists), *limit, dryRunSuffix(*dryRun))

	scraper := halo.NewLeaderboardScraper(time.Duration(*politeMs) * time.Millisecond)

	var db *sql.DB
	if !*dryRun {
		var err error
		db, err = openSharedRW(*sharedDBPath)
		if err != nil {
			fatal("open shared DB: %v", err)
		}
		defer db.Close()
		if err := migration.RunForDB(db, migration.TargetShared); err != nil {
			fatal("migration shared: %v", err)
		}
	}

	t0 := time.Now()
	totalRows, totalInserted := 0, 0
	for _, pl := range playlists {
		entries, err := scraper.FetchCSRLeaderboard(ctx, *season, pl, *limit)
		if err != nil {
			slog.ErrorContext(ctx, "scrape playlist échoué", "playlist", pl, "err", err)
			continue
		}
		totalRows += len(entries)
		fmt.Printf("  %s : %d entrées scrapées\n", pl, len(entries))
		if *dryRun || len(entries) == 0 {
			continue
		}
		n, err := duckdb.InsertWorldCSRSnapshot(ctx, db, entries)
		if err != nil {
			slog.ErrorContext(ctx, "insert snapshot échoué", "playlist", pl, "inserted", n, "err", err)
			continue
		}
		totalInserted += n
	}

	slog.InfoContext(ctx, "snapshot world leaderboard terminé",
		"season", *season, "playlists", len(playlists),
		"rows_scraped", totalRows, "rows_inserted", totalInserted,
		"dry_run", *dryRun, "duration", time.Since(t0).String())
	fmt.Printf("\nTerminé : %d entrées scrapées, %d insérées.\n", totalRows, totalInserted)
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
