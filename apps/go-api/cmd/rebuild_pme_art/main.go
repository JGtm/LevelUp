//go:build cgo

// rebuild_pme_art — reconstruit l'index ART de player_match_enrichment d'une ou
// plusieurs player DB via swap CTAS (migration.RebuildPlayerMatchEnrichmentART),
// pour defaire la corruption qui fait crasher DuckDB sous pression d'UPDATE
// ("Failed to append to PRIMARY_player_match_enrichment_0: duplicate key").
//
// Pourquoi pas force_rebuild_art --player-db : ce dernier tente AUSSI de
// rebuild match_skill_rank, table devenue append-only (v5.3) qui ne peut plus
// porter de PRIMARY KEY -> echec + WAL non checkpointe. Ce CLI cible UNIQUEMENT
// player_match_enrichment et CHECKPOINT explicitement pour ne laisser aucun WAL.
//
// Non destructif : garde anti-perte (rollback si le nombre de rows change).
//
// Usage :
//
//	go run -tags cgo ./cmd/rebuild_pme_art <player_db_path> [<player_db_path>...]
//
// PRE-REQUIS : aucun writer (serveur, backfill) ne doit tenir les DB ciblees.
// Faire un backup des fichiers avant.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: rebuild_pme_art <player_db_path> [<player_db_path>...]")
		os.Exit(2)
	}
	ctx := context.Background()
	failed := 0
	for _, path := range os.Args[1:] {
		if err := rebuildOne(ctx, path); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", path, err)
			failed++
		}
	}
	if failed > 0 {
		os.Exit(1)
	}
}

func rebuildOne(ctx context.Context, path string) error {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return fmt.Errorf("open RW: %w (serveur/backfill encore actif ?)", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1) // DuckDB single-writer : tout serialise sur 1 connexion

	// Rejoue + flush tout WAL en attente (ex. crash backfill anterieur) AVANT le
	// rebuild, pour partir d'un etat checkpointe propre.
	if _, err := db.ExecContext(ctx, `CHECKPOINT`); err != nil {
		return fmt.Errorf("checkpoint initial: %w", err)
	}

	var before int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM player_match_enrichment`).Scan(&before); err != nil {
		return fmt.Errorf("count before: %w", err)
	}

	start := time.Now()
	if err := migration.RebuildPlayerMatchEnrichmentART(ctx, db); err != nil {
		return err
	}

	var after int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM player_match_enrichment`).Scan(&after); err != nil {
		return fmt.Errorf("count after: %w", err)
	}
	if before != after {
		return fmt.Errorf("INCOHERENCE rows %d -> %d (NE PAS utiliser, restaurer le backup)", before, after)
	}

	// CHECKPOINT final : integre le rebuild dans le fichier principal et vide le
	// WAL, pour qu'un open ulterieur (read_only inclus) n'ait rien a rejouer.
	if _, err := db.ExecContext(ctx, `CHECKPOINT`); err != nil {
		return fmt.Errorf("checkpoint final: %w", err)
	}

	fmt.Printf("OK %s : %d rows preservees (%.1fs)\n", path, after, time.Since(start).Seconds())
	return nil
}
