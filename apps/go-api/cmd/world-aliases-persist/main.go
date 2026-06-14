//go:build cgo

// cmd/world-aliases-persist — persiste le cache gamertag->xuid du backfill world
// (checkpoint resolved_xuids) dans la table globale xuid_aliases.
//
// Le backfill world résout les gamertags du classement mondial en xuid via PeopleHub
// et garde le mapping dans data/world_backfill_checkpoint.json (champ resolved_xuids).
// Ces associations ne vivaient que dans ce JSON ; ce one-shot les rend DURABLES et
// app-wide en les insérant dans xuid_aliases (source=world_leaderboard) → elles
// alimentent la source unique d'affichage des gamertags (GamertagLookupView) et on ne
// re-résout plus ces joueurs.
//
// INSERT-only : ON CONFLICT (xuid) DO NOTHING. On n'ÉCRASE JAMAIS un alias existant
// (alimenté par le sync depuis les matchs, potentiellement plus frais) — on ne fait
// qu'AJOUTER les xuid qu'on ne connaissait pas. Idempotent (re-lançable sans risque).
//
// IMPORTANT : stopper le serveur API avant (shared DB ouverte en RW ; DuckDB interdit
// deux writers). Serveur arrêté = zéro concurrence → ON CONFLICT sans risque ART.
//
// Usage (depuis apps/go-api/, ou avec LEVELUP_REPO_ROOT) :
//
//	go run ./cmd/world-aliases-persist -dry-run   # compte sans écrire
//	go run ./cmd/world-aliases-persist            # upsert effectif
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/config"
	titlepkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability/logging"
)

func main() {
	checkpoint := flag.String("checkpoint", "", "checkpoint JSON du backfill world ; vide = <RepoRoot>/data/world_backfill_checkpoint.json")
	sharedDB := flag.String("shared-db", "", "chemin shared_matches_v2.duckdb (RW — stopper le serveur) ; vide = dérivé de RepoRoot")
	source := flag.String("source", "world_leaderboard", "valeur de la colonne source pour les alias insérés")
	dryRun := flag.Bool("dry-run", false, "compte les nouveaux vs existants sans écrire")
	flag.Parse()

	closeLogs := logging.InstallCLI(os.Getenv("LEVELUP_REPO_ROOT"))
	defer closeLogs()

	cfg, err := config.Load()
	if err != nil {
		fatal("chargement config: %v", err)
	}
	cpPath := strings.TrimSpace(*checkpoint)
	if cpPath == "" {
		cpPath = filepath.Join(cfg.RepoRoot, "data", "world_backfill_checkpoint.json")
	}
	dbPath := strings.TrimSpace(*sharedDB)
	if dbPath == "" {
		dbPath = titlepkg.NewPathResolver(cfg.RepoRoot).SharedDBPath(titlepkg.DefaultSlug)
	}
	if err := run(cpPath, dbPath, *source, *dryRun); err != nil {
		fatal("%v", err)
	}
}

func run(checkpointPath, sharedDBPath, source string, dryRun bool) error {
	resolved, err := loadResolvedXUIDs(checkpointPath)
	if err != nil {
		return err
	}
	fmt.Printf("checkpoint: %s\n", checkpointPath)
	fmt.Printf("shared DB: %s\n", sharedDBPath)
	fmt.Printf("associations gamertag->xuid dans le cache: %d%s\n", len(resolved), dryRunSuffix(dryRun))
	if len(resolved) == 0 {
		fmt.Println("Rien à persister.")
		return nil
	}

	db, err := sql.Open("duckdb", sharedDBPath)
	if err != nil {
		return fmt.Errorf("open shared DB (serveur stoppé ?): %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ctx := context.Background()

	existing, err := loadExistingXUIDs(ctx, db)
	if err != nil {
		return fmt.Errorf("lecture xuid_aliases existants: %w", err)
	}
	nNew := 0
	for _, xuid := range resolved {
		if xuid != "" && !existing[xuid] {
			nNew++
		}
	}
	fmt.Printf("→ %d nouveaux xuid (à insérer), %d déjà présents (conservés tels quels)\n", nNew, len(resolved)-nNew)
	if dryRun {
		fmt.Println("[dry-run] aucune écriture.")
		return nil
	}

	inserted, err := upsertAliases(ctx, db, resolved, source)
	if err != nil {
		return err
	}
	fmt.Printf("✓ %d alias insérés dans xuid_aliases (source=%s) ; %d déjà présents ignorés.\n",
		inserted, source, len(resolved)-inserted)
	return nil
}

// loadResolvedXUIDs lit le champ resolved_xuids (gamertag->xuid) du checkpoint.
func loadResolvedXUIDs(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lecture checkpoint %s: %w", path, err)
	}
	var cp struct {
		ResolvedXUIDs map[string]string `json:"resolved_xuids"`
	}
	if err := json.Unmarshal(b, &cp); err != nil {
		return nil, fmt.Errorf("parse checkpoint: %w", err)
	}
	return cp.ResolvedXUIDs, nil
}

// loadExistingXUIDs charge l'ensemble des xuid déjà présents dans xuid_aliases.
func loadExistingXUIDs(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT xuid FROM xuid_aliases`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var x string
		if err := rows.Scan(&x); err != nil {
			return nil, err
		}
		out[x] = true
	}
	return out, rows.Err()
}

// upsertAliases insère les paires (gamertag, xuid) en INSERT-only (ON CONFLICT DO
// NOTHING) dans une transaction. Retourne le nombre de lignes réellement insérées.
func upsertAliases(ctx context.Context, db *sql.DB, resolved map[string]string, source string) (int, error) {
	now := time.Now().UTC()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO xuid_aliases (xuid, gamertag, last_seen, source, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (xuid) DO NOTHING`)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	defer stmt.Close()
	inserted := 0
	for gamertag, xuid := range resolved {
		if xuid == "" || gamertag == "" {
			continue
		}
		res, err := stmt.ExecContext(ctx, xuid, gamertag, now, source, now)
		if err != nil {
			_ = tx.Rollback()
			return 0, fmt.Errorf("upsert %s (%s): %w", gamertag, xuid, err)
		}
		if n, _ := res.RowsAffected(); n > 0 {
			inserted++
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return inserted, nil
}

func dryRunSuffix(dry bool) string {
	if dry {
		return " [dry-run]"
	}
	return ""
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\nFATAL: "+format+"\n", args...)
	os.Exit(1)
}
