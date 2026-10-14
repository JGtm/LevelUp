package main

// cmd_backfill_registry_names.go — sous-commande `levelup backfill-registry-names`.
//
// ELLE FAIT CONVERGER LES NOMS D'ASSETS DU REGISTRE (map_name, pair_name, playlist_name,
// game_variant_name) restés NULL ou égaux à leur identifiant vers leurs traductions en-US de
// metadata.asset_translations. Même mécanique que le balayage périodique du serveur et que
// l'action admin : sync.BackfillRegistryNames → persist.RegistryNamesPersister (un match à la
// fois, garde `x IS NULL OR x = x_id` ; la catégorie de mode suit le nom de paire écrit). Une paire sans traduction est
// construite « {variante} on {carte} » quand la variante et la carte ont un nom ; sinon la
// colonne reste en l'état et est comptée « sans source ».
//
// Idempotente : une seconde passe ne trouve plus que les colonnes sans source.
//
// SERVEUR ARRÊTÉ, Y COMPRIS POUR --dry-run : le serveur tient metadata.duckdb en écriture en
// permanence (et la base partagée pendant ses écritures) ; DuckDB refuse alors à tout autre
// processus d'ouvrir le fichier, même en lecture seule. Une base tenue fait échouer la commande
// avant toute lecture, par errServeurLance.
//
// Usage :
//
//	levelup backfill-registry-names --dry-run   # comptes par colonne, aucune écriture
//	levelup backfill-registry-names             # écriture

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
	go_sync "levelup/go-api/internal/sync"
)

// errConvergenceIncomplete : des matchs n'ont pas pu être écrits (journalisés un à un).
var errConvergenceIncomplete = errors.New("convergence des noms incomplète : écritures en échec")

// errServeurLance : une base de la commande est tenue par un autre processus
// (duckdb.ErrBaseTenueEnEcriture) — le serveur LevelUp, le plus souvent.
var errServeurLance = errors.New("base tenue par un autre processus : arrêter le serveur LevelUp, y compris pour --dry-run")

// erreurOuverture nomme l'échec d'ouverture de `path` ; une base tenue par un autre processus
// est rendue sous errServeurLance (message d'origine de DuckDB conservé : il nomme le détenteur).
func erreurOuverture(quoi, path string, err error) error {
	if errors.Is(err, duckdb.ErrBaseTenueEnEcriture) {
		return fmt.Errorf("%s (%s) : %w : %w", quoi, path, errServeurLance, err)
	}
	return fmt.Errorf("%s (%s) : %w", quoi, path, err)
}

func runBackfillRegistryNames(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("backfill-registry-names", flag.ExitOnError)
	titleSlug := fs.String("title", titlePkg.DefaultSlug, "slug du titre")
	dryRun := fs.Bool("dry-run", false, "compter par colonne sans rien écrire (serveur arrêté, comme l'écriture)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	sharedPath, metaPath := pr.SharedDBPath(*titleSlug), pr.MetadataDBPath(*titleSlug)
	for _, p := range []string{sharedPath, metaPath} {
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("base introuvable (%s): %w", p, err)
		}
	}
	metaDB, releaseMeta, err := duckdb.OpenReadForQuery(metaPath)
	if err != nil {
		return erreurOuverture("open metadata", metaPath, err)
	}
	defer releaseMeta()

	ctx := context.Background()
	var stats go_sync.BackfillRegistryStats
	if *dryRun {
		sharedDB, releaseShared, oerr := duckdb.OpenReadForQuery(sharedPath)
		if oerr != nil {
			return erreurOuverture("open shared lecture", sharedPath, oerr)
		}
		defer releaseShared()
		stats, err = go_sync.BackfillRegistryNames(ctx, sharedDB, metaDB, go_sync.RegistryNamesOptions{DryRun: true})
	} else {
		handle, oerr := duckdb.OpenReadWrite(sharedPath)
		if oerr != nil {
			return erreurOuverture("open shared RW", sharedPath, oerr)
		}
		defer handle.Close()
		stats, err = go_sync.BackfillRegistryNames(ctx, handle.SQLDb(), metaDB, go_sync.RegistryNamesOptions{})
	}
	if err != nil {
		return err
	}
	printRegistryNamesStats(stats)
	if stats.Errors > 0 {
		return fmt.Errorf("%w : %d matchs", errConvergenceIncomplete, stats.Errors)
	}
	return nil
}

// printRegistryNamesStats imprime, par colonne, les matchs candidats, réparés (ou réparables en
// simulation) et sans source de nom.
func printRegistryNamesStats(s go_sync.BackfillRegistryStats) {
	verbe := "réparés"
	if s.DryRun {
		verbe = "réparables"
	}
	mode := "écriture"
	if s.DryRun {
		mode = "simulation"
	}
	fmt.Printf("noms du registre (%s) — matchs par colonne :\n", mode)
	lignes := []struct {
		col          string
		scanned, fix int
	}{
		{"map_name", s.MapsScanned, s.MapsFixed},
		{"pair_name", s.PairsScanned, s.PairsFixed},
		{"playlist_name", s.PlaylistsScanned, s.PlaylistsFixed},
		{"game_variant_name", s.VariantsScanned, s.VariantsFixed},
	}
	for _, l := range lignes {
		fmt.Printf("  %-18s candidats %6d   %s %6d   sans source %6d\n", l.col, l.scanned, verbe, l.fix, l.scanned-l.fix)
	}
	fmt.Printf("  dont paires construites « {variante} on {carte} » : %d\n", s.PairsConstructed)
	fmt.Printf("  total : %d colonnes sur %d matchs   échecs d'écriture : %d\n", s.Total(), s.Matches, s.Errors)
}
