package main

// cmd_rebuild_pme.go — sous-commande `levelup rebuild-pme-art` : reconstruit
// l'index ART de player_match_enrichment (swap CTAS transactionnel, garde
// anti-perte, recrée les indexes) pour défaire la corruption DuckDB 1.5.x
// (issue amont #23046) qui fait crasher les backfills UPDATE-lourds
// (ex. engagement-coefs --with-scores).
//
// Non destructif : refuse de détruire l'original si le nombre de rows change.
// CHECKPOINT initial (rejoue un WAL de crash éventuel) + final (vide le WAL).
//
// Pré-requis : aucun writer concurrent (serveur arrêté). En prod via
// `docker compose stop levelup` puis `docker compose run --rm levelup
// levelup rebuild-pme-art --all`.

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
)

func runRebuildPME(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("rebuild-pme-art", flag.ExitOnError)
	gamertag := fs.String("gamertag", "", "Gamertag du joueur (mutuellement exclusif avec --all)")
	allPlayers := fs.Bool("all", false, "Reconstruit pour tous les joueurs de db_profiles.json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *allPlayers == (strings.TrimSpace(*gamertag) != "") {
		return fmt.Errorf("spécifier soit --gamertag <gt>, soit --all")
	}

	ctx := context.Background()
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

	if *allPlayers {
		players, err := cfg.LoadPlayers()
		if err != nil {
			return fmt.Errorf("chargement db_profiles.json: %w", err)
		}
		failed := 0
		for _, p := range players {
			dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, p.Gamertag)
			if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
				fmt.Printf("rebuild-pme-art SKIP: gamertag=%s reason=no_player_db\n", p.Gamertag)
				continue
			}
			if err := rebuildPMEOne(ctx, dbPath, p.Gamertag); err != nil {
				failed++
				fmt.Fprintf(os.Stderr, "rebuild-pme-art FAIL: gamertag=%s err=%v\n", p.Gamertag, err)
			}
		}
		if failed > 0 {
			return fmt.Errorf("rebuild-pme-art: %d joueur(s) en échec", failed)
		}
		return nil
	}

	player, err := loadPlayerSummary(cfg, *gamertag)
	if err != nil {
		return err
	}
	dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, player.Gamertag)
	if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
		return fmt.Errorf("player DB introuvable: %s", dbPath)
	}
	return rebuildPMEOne(ctx, dbPath, player.Gamertag)
}

// rebuildPMEOne ouvre la player DB en RW exclusif, rejoue/vide le WAL, lance le
// rebuild, vérifie qu'aucune ligne n'est perdue, puis CHECKPOINT.
func rebuildPMEOne(ctx context.Context, dbPath, gamertag string) error {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("open RW %s: %w (serveur encore actif ?)", dbPath, err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1) // DuckDB single-writer

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
		return fmt.Errorf("INCOHÉRENCE rows %d -> %d (NE PAS utiliser, restaurer le backup)", before, after)
	}
	if _, err := db.ExecContext(ctx, `CHECKPOINT`); err != nil {
		return fmt.Errorf("checkpoint final: %w", err)
	}
	fmt.Printf("rebuild-pme-art OK: gamertag=%s rows=%d preservees (%.1fs)\n", gamertag, after, time.Since(start).Seconds())
	return nil
}
