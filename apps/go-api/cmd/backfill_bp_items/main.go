// cmd/backfill_bp_items — backfill one-shot des définitions d'items Battle Pass.
//
// Lit tous les JSONs d'items déjà présents dans asset_index (kind 'track-def' ou
// 'bp-item-def') et les persiste dans battlepass_item_definitions +
// battlepass_item_translations via UpsertItemDefinition.
//
// Les nouveaux items (fetchés après ce déploiement) sont persistés automatiquement
// par warmBPTrackAssets via le kind KindBPItemDefinition. Ce script est donc
// uniquement nécessaire une seule fois pour migrer les items existants.
//
// Usage:
//
//	backfill_bp_items [flags]
//
// Flags:
//
//	--title     Slug du titre (défaut: halo_infinite)
//	--dry-run   Affiche le nombre d'items sans écrire
//
// Exemple:
//
//	backfill_bp_items --title halo_infinite
//	backfill_bp_items --dry-run
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

func main() {
	titleFlag := flag.String("title", titlePkg.DefaultSlug, "Slug du titre")
	dryRun := flag.Bool("dry-run", false, "Simule sans écrire")
	flag.Parse()

	repoRoot, err := findRepoRoot()
	if err != nil {
		slog.Error("findRepoRoot", "err", err)
		os.Exit(1)
	}

	pr := titlePkg.NewPathResolver(repoRoot)
	metaPath := pr.MetadataDBPath(*titleFlag)

	slog.Info("backfill_bp_items: démarrage",
		"title", *titleFlag,
		"meta_path", metaPath,
		"dry_run", *dryRun,
	)

	ctx := context.Background()
	if err := run(ctx, metaPath, *dryRun); err != nil {
		slog.Error("backfill_bp_items: échec", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, metaPath string, dryRun bool) error {
	items, err := loadItemsFromAssetIndex(ctx, metaPath)
	if err != nil {
		return fmt.Errorf("loadItemsFromAssetIndex: %w", err)
	}

	slog.Info("backfill_bp_items: items trouvés dans asset_index", "count", len(items))

	if dryRun {
		slog.Info("backfill_bp_items: dry-run, aucune écriture effectuée")
		return nil
	}

	sink := &duckdb.PersistSink{MetaPath: metaPath}

	ok, skipped, failed := 0, 0, 0
	for _, item := range items {
		if err := sink.UpsertItemDefinition(ctx, item.path, item.raw); err != nil {
			slog.Warn("UpsertItemDefinition failed", "path", item.path, "err", err)
			failed++
			continue
		}
		ok++
		if ok%50 == 0 {
			slog.Info("backfill_bp_items: progression", "ok", ok, "skipped", skipped, "failed", failed)
		}
	}

	slog.Info("backfill_bp_items: terminé", "ok", ok, "skipped", skipped, "failed", failed)
	return nil
}

type assetIndexItem struct {
	path string
	raw  []byte
}

// loadItemsFromAssetIndex charge les JSONs d'items depuis asset_index.
// Cherche dans kind='bp-item-def' (nouveau) et kind='track-def' (ancien, fallback).
// Un item présent dans les deux kinds n'est retourné qu'une seule fois.
func loadItemsFromAssetIndex(ctx context.Context, metaPath string) ([]assetIndexItem, error) {
	db, err := duckdb.OpenReadOnly(metaPath)
	if err != nil {
		return nil, fmt.Errorf("open meta read-only: %w", err)
	}
	defer db.Close()

	const query = `
		SELECT id, raw_json
		FROM asset_index
		WHERE kind IN ('bp-item-def', 'track-def')
		  AND raw_json IS NOT NULL
		  AND raw_json LIKE '%CommonData%'
		  AND id NOT LIKE 'RewardTracks/%'`

	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("asset_index query: %w", err)
	}
	defer rows.Close()

	seen := make(map[string]struct{})
	var items []assetIndexItem

	for rows.Next() {
		var id sql.NullString
		var rawJSON sql.NullString
		if err := rows.Scan(&id, &rawJSON); err != nil {
			continue
		}
		if !id.Valid || id.String == "" || !rawJSON.Valid || rawJSON.String == "" {
			continue
		}
		if _, dup := seen[id.String]; dup {
			continue
		}
		// Vérification rapide que le JSON est un objet item (pas un track).
		if !isItemJSON(rawJSON.String) {
			continue
		}
		seen[id.String] = struct{}{}
		items = append(items, assetIndexItem{
			path: id.String,
			raw:  []byte(rawJSON.String),
		})
	}
	return items, rows.Err()
}

// isItemJSON retourne true si le JSON ressemble à un item (possède CommonData.Quality
// ou CommonData.DisplayPath), et non à un Reward Track (qui a Ranks[]).
func isItemJSON(raw string) bool {
	if strings.Contains(raw, `"Ranks"`) {
		return false
	}
	var check struct {
		CommonData *json.RawMessage `json:"CommonData"`
	}
	if err := json.Unmarshal([]byte(raw), &check); err != nil || check.CommonData == nil {
		return false
	}
	return true
}

// findRepoRoot remonte depuis le répertoire courant pour trouver la racine du repo.
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	// Remonte jusqu'à trouver go.mod à la racine du module Go.
	for {
		if _, err := os.Stat(dir + "/go.mod"); err == nil {
			return dir, nil
		}
		parent := dir[:strings.LastIndexByte(dir, '/')]
		if parent == dir || parent == "" {
			break
		}
		// Chercher aussi la racine du repo (contient apps/).
		if _, err := os.Stat(dir + "/apps"); err == nil {
			return dir, nil
		}
		dir = parent
	}
	return "", fmt.Errorf("impossible de trouver la racine du repo depuis %s", dir)
}
