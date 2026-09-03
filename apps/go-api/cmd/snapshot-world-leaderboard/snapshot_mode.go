//go:build cgo

// snapshot_mode.go — mode CAPTURE du job : scrape les pages Halo Waypoint pour une ou
// toutes les saisons et persiste les relevés en append-only. Extrait de main.go, qui
// ne garde que les drapeaux et l'aiguillage entre les deux modes (capture ici,
// réparation dans restore_best.go).
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/halo"
)

// snapshotOptions rassemble les réglages du mode capture. Une struct plutôt qu'une
// liste d'arguments : sept réglages en paramètres rendraient chaque signature de la
// chaîne illisible.
type snapshotOptions struct {
	season       string
	sharedDBPath string
	playlistsCSV string
	limit        int
	politeMs     int
	dryRun       bool
	titleSlug    string
}

// runSnapshotMode exécute la capture : résolution des playlists et des saisons,
// ouverture de la base (sauf en dry-run), puis scrape saison par saison.
func runSnapshotMode(ctx context.Context, log *slog.Logger, opt snapshotOptions) {
	playlists := resolvePlaylists(opt.playlistsCSV)
	if len(playlists) == 0 {
		fatal("aucune playlist à traiter")
	}
	scraper := halo.NewLeaderboardScraper(time.Duration(opt.politeMs) * time.Millisecond)

	// 'all' → snapshot TOUTES les saisons exposées par Halo Waypoint (csrseason3-1
	// jusqu'à l'active), pas seulement la courante. Miroir du -season all du backfill.
	seasons, err := resolveSeasons(ctx, scraper, opt.season, playlists[0])
	if err != nil {
		fatal("résolution des saisons: %v", err)
	}
	log.InfoContext(ctx, "snapshot world leaderboard démarré",
		"seasons", len(seasons), "playlists", len(playlists), "limit", opt.limit, "dry_run", opt.dryRun)
	fmt.Printf("Saisons (%d) : %v — %d playlist(s), limite %d/playlist%s\n",
		len(seasons), seasons, len(playlists), opt.limit, dryRunSuffix(opt.dryRun))

	var db *sql.DB
	if !opt.dryRun {
		db, err = openSharedRW(opt.sharedDBPath)
		if err != nil {
			fatal("open shared DB: %v", err)
		}
		defer db.Close()
		if err := migration.RunForDB(db, migration.TargetShared); err != nil {
			fatal("migration shared: %v", err)
		}
	}

	runner := &snapshotRunner{log: log, db: db, scraper: scraper, playlists: playlists, opt: opt}
	t0 := time.Now()
	grandRows, grandInserted := 0, 0
	for i, s := range seasons {
		if i > 0 && opt.politeMs > 0 {
			time.Sleep(time.Duration(opt.politeMs) * time.Millisecond)
		}
		r, ins := runner.season(ctx, s)
		grandRows += r
		grandInserted += ins
	}

	log.InfoContext(ctx, "snapshot world leaderboard terminé",
		"seasons", len(seasons), "playlists", len(playlists),
		"rows_scraped", grandRows, "rows_inserted", grandInserted,
		"dry_run", opt.dryRun, "duration", time.Since(t0).String())
	fmt.Printf("\nTerminé : %d saison(s), %d entrées scrapées, %d insérées.\n", len(seasons), grandRows, grandInserted)
}

// snapshotRunner porte le contexte commun à toutes les saisons d'un run (base,
// scraper, playlists, réglages) : sans lui, chaque saison scrapée demanderait de
// repasser six arguments.
type snapshotRunner struct {
	log       *slog.Logger
	db        *sql.DB
	scraper   *halo.LeaderboardScraper
	playlists []string
	opt       snapshotOptions
}

// season scrape et persiste une saison (toutes ses playlists). Best-effort par
// playlist : une page en échec ou un lot refusé n'interrompt pas les suivantes.
func (r *snapshotRunner) season(ctx context.Context, s string) (rows, inserted int) {
	fmt.Printf("Saison %s :\n", s)
	for i, pl := range r.playlists {
		// Délai poli ENTRE playlists (pas seulement entre pages) : Halo Waypoint
		// throttle au-delà de quelques requêtes rapprochées (429). Sans ça, un
		// -season all tire ~14 requêtes d'affilée et se fait couper.
		if i > 0 && r.opt.politeMs > 0 {
			time.Sleep(time.Duration(r.opt.politeMs) * time.Millisecond)
		}
		entries, ferr := r.scraper.FetchCSRLeaderboard(ctx, s, pl, r.opt.limit)
		if ferr != nil {
			// 404 = playlist non classée cette saison : skip NOMINAL d'un backfill
			// multi-saisons × toutes playlists (pas une erreur). Les vraies erreurs
			// (429, 5xx, réseau) restent en ERROR.
			if errors.Is(ferr, halo.ErrLeaderboardPageNotFound) {
				fmt.Printf("  %s : —  (non classée cette saison)\n", pl)
				r.log.InfoContext(ctx, "playlist non classée cette saison — ignorée", "season", s, "playlist", pl)
			} else {
				r.log.ErrorContext(ctx, "scrape playlist échoué", "season", s, "playlist", pl, "err", ferr)
			}
			continue
		}
		rows += len(entries)
		fmt.Printf("  %s : %d entrées\n", pl, len(entries))
		if r.opt.dryRun || len(entries) == 0 {
			continue
		}
		// Même garde-fou que le cron : un lot effondré (volume ou identification)
		// ne doit pas masquer le relevé servi — la vue _latest sert le dernier
		// lot écrit, et ces snapshots sont la seule archive.
		key := duckdb.WorldCSRBatchKey{TitleSlug: r.opt.titleSlug, SeasonID: s, PlaylistID: pl}
		if reason := cliBatchRefusalReason(ctx, r.db, key, entries); reason != "" {
			fmt.Printf("  %s : lot REFUSE — %s (relevé servi conservé)\n", pl, reason)
			r.log.WarnContext(ctx, "lot dégradé refusé — aucune écriture pour cette playlist",
				"season", s, "playlist", pl, "raison", reason, "candidat_lignes", len(entries))
			continue
		}
		n, ierr := duckdb.InsertWorldCSRSnapshot(ctx, r.db, r.opt.titleSlug, entries)
		if ierr != nil {
			r.log.ErrorContext(ctx, "insert snapshot échoué", "season", s, "playlist", pl, "err", ierr)
			continue
		}
		inserted += n
	}
	return rows, inserted
}
