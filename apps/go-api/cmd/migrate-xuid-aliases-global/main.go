// cmd/migrate-xuid-aliases-global — migration one-shot des `xuid_aliases`
// par-titre vers la DB globale `data/global/xbox_aliases.duckdb`.
//
// Le mapping xuid → gamertag est un identifiant Microsoft Xbox Services qui ne
// dépend pas du titre, donc DB globale partagée par tous les titres (P5, ADR
// 0008).
//
// Stratégie : pour chaque titre dans `data/titles/<slug>/`, lire
// `shared.xuid_aliases` et insérer dans la DB globale avec dédup sur xuid
// (UPSERT : on garde le `last_seen` max). Les tables locales ne sont PAS
// droppées par ce script — usage `--drop-local` pour le faire dans une seconde
// passe une fois la migration validée.
//
// Idempotent : exécuter plusieurs fois ne casse rien.
//
// Usage :
//
//	cd apps/go-api && go run ./cmd/migrate-xuid-aliases-global [--repo-root /path/to/repo] [--dry-run]
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	titlePkg "levelup/go-api/internal/domain/title"
)

func main() {
	repoRoot := flag.String("repo-root", autoDetectRepoRoot(), "Racine du repo LevelUp")
	dryRun := flag.Bool("dry-run", false, "Afficher les actions sans écrire")
	dropLocal := flag.Bool("drop-local", false, "Drop les tables xuid_aliases locales (post-validation)")
	flag.Parse()

	ctx := context.Background()
	if err := run(ctx, *repoRoot, *dryRun, *dropLocal); err != nil {
		slog.ErrorContext(ctx, "xuid migration failed", "err", err)
		os.Exit(1)
	}
}

// autoDetectRepoRoot remonte 2 niveaux depuis apps/go-api.
func autoDetectRepoRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return filepath.Join(cwd, "..", "..")
}

// titleAlias est la projection d'une ligne xuid_aliases.
type titleAlias struct {
	XUID     string
	Gamertag string
	LastSeen time.Time
}

func run(ctx context.Context, repoRoot string, dryRun, dropLocal bool) error {
	pr := titlePkg.NewPathResolver(repoRoot)
	titlesDir := filepath.Join(repoRoot, "data", "titles")

	entries, err := os.ReadDir(titlesDir)
	if err != nil {
		return fmt.Errorf("lecture data/titles: %w", err)
	}

	var slugs []string
	for _, e := range entries {
		if e.IsDir() {
			slugs = append(slugs, e.Name())
		}
	}
	slog.InfoContext(ctx, "xuid migration started",
		"repo_root", repoRoot,
		"titles_count", len(slugs),
		"dry_run", dryRun,
		"drop_local", dropLocal,
	)
	start := time.Now()

	// Charger toutes les rows par titre.
	allByXUID := make(map[string]titleAlias)
	totalRowsRead := 0
	totalDuplicatesCollapsed := 0
	for _, slug := range slugs {
		sharedPath := pr.SharedDBPath(slug)
		if _, err := os.Stat(sharedPath); err != nil {
			slog.InfoContext(ctx, "shared db absent skipping", "title", slug, "path", sharedPath)
			continue
		}
		rows, err := readTitleAliases(ctx, sharedPath)
		if err != nil {
			slog.ErrorContext(ctx, "xuid migration failed", "err", err, "title", slug)
			return fmt.Errorf("title %s: %w", slug, err)
		}
		duplicates := 0
		for _, r := range rows {
			existing, ok := allByXUID[r.XUID]
			if !ok || r.LastSeen.After(existing.LastSeen) {
				allByXUID[r.XUID] = r
				if ok {
					duplicates++
				}
			} else {
				duplicates++
			}
		}
		totalRowsRead += len(rows)
		totalDuplicatesCollapsed += duplicates
		slog.InfoContext(ctx, "xuid migration title processed",
			"title", slug,
			"rows_read", len(rows),
			"rows_kept", len(rows)-duplicates,
			"duplicates_collapsed", duplicates,
		)
	}

	// Écrire dans la DB globale (UPSERT).
	globalPath := pr.GlobalXuidAliasesDBPath()
	if dryRun {
		slog.InfoContext(ctx, "xuid migration dry-run completed",
			"would_write_rows", len(allByXUID),
			"global_path", globalPath,
			"total_rows_read", totalRowsRead,
			"total_duplicates_collapsed", totalDuplicatesCollapsed,
		)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
		return fmt.Errorf("mkdir global dir: %w", err)
	}
	written, err := writeGlobalAliases(ctx, globalPath, allByXUID)
	if err != nil {
		return fmt.Errorf("write global: %w", err)
	}

	if dropLocal {
		dropped := 0
		for _, slug := range slugs {
			sharedPath := pr.SharedDBPath(slug)
			if _, err := os.Stat(sharedPath); err != nil {
				continue
			}
			if err := dropLocalAliases(ctx, sharedPath); err != nil {
				slog.ErrorContext(ctx, "drop local failed", "title", slug, "err", err)
				continue
			}
			dropped++
		}
		slog.InfoContext(ctx, "xuid migration drop-local completed", "dropped_titles", dropped)
	}

	slog.InfoContext(ctx, "xuid migration completed",
		"total_rows", written,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return nil
}

// readTitleAliases ouvre la DB shared d'un titre en read-only et lit les
// xuid_aliases existants. Tolère l'absence de la table.
func readTitleAliases(ctx context.Context, sharedPath string) ([]titleAlias, error) {
	db, err := sql.Open("duckdb", sharedPath+"?access_mode=read_only")
	if err != nil {
		return nil, fmt.Errorf("open shared: %w", err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `
		SELECT xuid, gamertag, COALESCE(last_seen, now())
		FROM xuid_aliases
	`)
	if err != nil {
		// Table absente — tolérer.
		return nil, nil
	}
	defer rows.Close()

	var out []titleAlias
	for rows.Next() {
		var a titleAlias
		if err := rows.Scan(&a.XUID, &a.Gamertag, &a.LastSeen); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// writeGlobalAliases ouvre/crée la DB globale et insère les rows en UPSERT.
// Retourne le nombre de rows écrites.
func writeGlobalAliases(ctx context.Context, globalPath string, byXUID map[string]titleAlias) (int, error) {
	db, err := sql.Open("duckdb", globalPath)
	if err != nil {
		return 0, fmt.Errorf("open global: %w", err)
	}
	defer db.Close()

	// Schéma.
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS xuid_aliases (
			xuid VARCHAR PRIMARY KEY,
			gamertag VARCHAR NOT NULL,
			last_seen TIMESTAMP NOT NULL
		)
	`); err != nil {
		return 0, fmt.Errorf("create table: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO xuid_aliases (xuid, gamertag, last_seen)
		VALUES (?, ?, ?)
		ON CONFLICT (xuid) DO UPDATE SET
			gamertag = excluded.gamertag,
			last_seen = GREATEST(xuid_aliases.last_seen, excluded.last_seen)
	`)
	if err != nil {
		return 0, fmt.Errorf("prepare upsert: %w", err)
	}
	defer stmt.Close()

	written := 0
	for _, a := range byXUID {
		if _, err := stmt.ExecContext(ctx, a.XUID, a.Gamertag, a.LastSeen); err != nil {
			return written, fmt.Errorf("upsert %s: %w", a.XUID, err)
		}
		written++
	}
	if err := tx.Commit(); err != nil {
		return written, fmt.Errorf("commit: %w", err)
	}
	return written, nil
}

// dropLocalAliases supprime la table xuid_aliases d'une DB shared.
// Idempotent (DROP TABLE IF EXISTS).
func dropLocalAliases(ctx context.Context, sharedPath string) error {
	db, err := sql.Open("duckdb", sharedPath)
	if err != nil {
		return fmt.Errorf("open shared rw: %w", err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS xuid_aliases`); err != nil {
		return fmt.Errorf("drop table: %w", err)
	}
	return nil
}
