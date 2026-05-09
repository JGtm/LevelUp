// backfill_registry_names — répare les noms d'assets stockés comme UUIDs bruts
// dans shared.match_registry (cas du fallback `coalesceStrPtr(name, id)` dans
// transforms.go ExtractRegistry, antérieur à EnrichRegistryFromMetadata).
//
// Pour chaque (asset_id) où *_name == *_id, lookup metadata.asset_translations
// [lang='en-US'] et UPDATE le nom canonique. Si la table de traduction n'a
// rien pour cet UUID, conserve la valeur (UUID) — pas de regression.
//
// Idempotent : re-run safe. Touche uniquement les rows pathologiques.
//
// Usage :
//
//	go run -tags cgo ./cmd/backfill_registry_names \
//	    --shared <path/to/shared_matches_v2.duckdb> \
//	    --metadata <path/to/metadata.duckdb> \
//	    [--dry-run]
//
// IMPORTANT : le serveur Go doit être stoppé pendant l'exécution (DuckDB
// requiert un lock écriture exclusif sur shared_matches_v2). Le backfill
// metadata est en lecture seule donc le serveur peut le partager.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/sync"
)

func main() {
	var (
		sharedPath = flag.String("shared", "", "Chemin vers shared_matches_v2.duckdb (RW)")
		metaPath   = flag.String("metadata", "", "Chemin vers metadata.duckdb (RO)")
		dryRun     = flag.Bool("dry-run", false, "N'effectue aucun UPDATE — affiche les compteurs")
	)
	flag.Parse()

	if *sharedPath == "" || *metaPath == "" {
		fmt.Fprintln(os.Stderr, "usage: backfill_registry_names --shared <path> --metadata <path> [--dry-run]")
		os.Exit(2)
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	ctx := context.Background()

	mode := "RW"
	if *dryRun {
		mode = "RO (dry-run)"
	}
	sharedDB, err := sql.Open("duckdb", *sharedPath+sharedAccessMode(*dryRun))
	if err != nil {
		fmt.Fprintf(os.Stderr, "open shared (%s): %v\n", mode, err)
		os.Exit(1)
	}
	defer sharedDB.Close()

	metaDB, err := sql.Open("duckdb", *metaPath+"?access_mode=read_only")
	if err != nil {
		fmt.Fprintf(os.Stderr, "open metadata: %v\n", err)
		os.Exit(1)
	}
	defer metaDB.Close()

	if *dryRun {
		// En mode dry-run on n'appelle pas BackfillRegistryNames (qui UPDATE).
		// On reproduit juste la phase 1 (scan) pour afficher les compteurs.
		fmt.Println("Mode dry-run : aucun UPDATE. Compteurs des UUIDs détectés :")
		for _, c := range []struct{ idCol, nameCol string }{
			{"playlist_id", "playlist_name"},
			{"map_id", "map_name"},
			{"pair_id", "pair_name"},
			{"game_variant_id", "game_variant_name"},
		} {
			var n int
			q := fmt.Sprintf(
				`SELECT COUNT(DISTINCT %s) FROM match_registry WHERE %s IS NOT NULL AND %s = %s`,
				c.idCol, c.idCol, c.nameCol, c.idCol,
			)
			if err := sharedDB.QueryRowContext(ctx, q).Scan(&n); err != nil {
				fmt.Printf("  %s: erreur scan: %v\n", c.idCol, err)
				continue
			}
			fmt.Printf("  %-20s : %d UUID(s) à backfiller\n", c.idCol, n)
		}
		return
	}

	stats, err := sync.BackfillRegistryNames(ctx, sharedDB, metaDB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "backfill: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Backfill terminé :")
	fmt.Printf("  playlists  : %d / %d UUIDs\n", stats.PlaylistsFixed, stats.PlaylistsScanned)
	fmt.Printf("  maps       : %d / %d UUIDs\n", stats.MapsFixed, stats.MapsScanned)
	fmt.Printf("  pairs      : %d / %d UUIDs\n", stats.PairsFixed, stats.PairsScanned)
	fmt.Printf("  variants   : %d / %d UUIDs\n", stats.VariantsFixed, stats.VariantsScanned)
	fmt.Printf("  TOTAL fixed: %d\n", stats.Total())
}

func sharedAccessMode(dryRun bool) string {
	if dryRun {
		return "?access_mode=read_only"
	}
	return ""
}
