//go:build cgo

// restore_one_player — one-shot : reconstruit le stats.duckdb d'UN seul joueur
// depuis un staging Parquet déjà extrait (pas de pull restic, juste l'import).
//
// Usage : go run -tags cgo ./cmd/restore_one_player --staging /tmp/staging_choco --player Chocoboflor
//
// Le staging doit avoir la structure : <staging>/halo_infinite/player/<gt>/*.parquet
// La DB cible est résolue via PathResolver. Aucune autre DB n'est touchée.
//
// Créé pour restaurer Chocoboflor après corruption causée par taskkill /F
// durant un cmd qui ne devait pas tourner alors que le serveur tenait la DB
// (incident 2026-05-27, cf. thought_log).
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/pkg/duckdbbackup"
)

func main() {
	staging := flag.String("staging", "", "racine staging avec /halo_infinite/player/<gt>/*.parquet")
	player := flag.String("player", "", "gamertag à restaurer")
	dryRun := flag.Bool("dry-run", false, "simuler sans modifier")
	flag.Parse()

	if *staging == "" || *player == "" {
		fmt.Fprintln(os.Stderr, "usage: restore_one_player --staging <dir> --player <gt> [--dry-run]")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	pr := title.NewPathResolver(cfg.RepoRoot)
	targetPath := pr.PlayerDBPath("halo_infinite", *player)

	expectedKey := "halo_infinite:player:" + *player
	keyToPath := func(key string) string {
		if key == expectedKey {
			return targetPath
		}
		return "" // skip toutes les autres clés
	}

	slog.Info("restore_one_player: démarrage",
		"staging", *staging, "player", *player, "target", targetPath, "dry_run", *dryRun)

	// Sécurité : vérifier que rien d'autre que le joueur cible n'est dans le staging
	playersDir := filepath.Join(*staging, "halo_infinite", "player")
	if entries, err := os.ReadDir(playersDir); err == nil {
		for _, e := range entries {
			if e.IsDir() && e.Name() != *player {
				slog.Warn("restore_one_player: autre joueur trouvé dans staging, sera skip via keyToPath",
					"other_player", e.Name())
			}
		}
	}

	result, err := duckdbbackup.ImportFromStaging(context.Background(), *staging, keyToPath, *dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "import: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Restauration %s\n", *player)
	fmt.Printf("  duration_ms : %d\n", result.DurationMs)
	fmt.Printf("  restored    : %v\n", result.Restored)
	fmt.Printf("  skipped     : %v\n", result.Skipped)
	if *dryRun {
		fmt.Println("(dry-run, aucune modif)")
		return
	}
	if err := verifyDBHealth(context.Background(), targetPath); err != nil {
		fmt.Fprintf(os.Stderr, "verify: %v\n", err)
		os.Exit(2)
	}
	fmt.Println("verify : OK")
}

// verifyDBHealth ouvre la DB restaurée en RO, liste les tables et fait un
// SELECT COUNT(*) sur chacune. Si DuckDB crash → la DB est corrompue ;
// si tout passe → restauration validée.
func verifyDBHealth(ctx context.Context, dbPath string) error {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		return fmt.Errorf("open RO: %w", err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `SELECT table_name FROM information_schema.tables WHERE table_schema='main' ORDER BY 1`)
	if err != nil {
		return fmt.Errorf("list tables: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	tables := make([]string, 0, 32)
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return err
		}
		tables = append(tables, t)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(tables) == 0 {
		return fmt.Errorf("aucune table trouvée — DB vide ?")
	}

	fmt.Printf("verify : %d tables, comptage row par row :\n", len(tables))
	for _, t := range tables {
		var n int
		if err := db.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, t)).Scan(&n); err != nil {
			return fmt.Errorf("count %s: %w", t, err)
		}
		fmt.Printf("  %-40s %10d rows\n", t, n)
	}
	return nil
}
