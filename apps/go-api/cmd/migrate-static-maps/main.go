package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/duckdb"
)

func main() {
	var (
		dryRun    bool
		staticDir string
	)
	flag.BoolVar(&dryRun, "dry-run", false, "Afficher les actions sans écrire en DB")
	flag.StringVar(&staticDir, "static-dir", "static/maps", "Répertoire contenant les images (relatif à la racine)")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	if err := run(dryRun, staticDir); err != nil {
		slog.Error("migrate-static-maps failed", "err", err)
		os.Exit(1)
	}
}

func run(dryRun bool, staticDir string) error {
	ctx := context.Background()

	// 1. Charger config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 2. Ouvrir metadata.duckdb
	metadataPath := filepath.Join(cfg.RepoRoot, "data", "warehouse", "metadata.duckdb")
	metaDB, err := duckdb.OpenReadWrite(metadataPath)
	if err != nil {
		return fmt.Errorf("open metadata DB: %w", err)
	}
	defer metaDB.Close()

	metaRepo := duckdb.NewMetadataRepoFromDB(metaDB)

	// 3. Scanner static/maps/
	rootDir := cfg.RepoRoot
	if rootDir == "" {
		rootDir = "."
	}
	mapsDir := filepath.Join(rootDir, staticDir)

	files, err := os.ReadDir(mapsDir)
	if err != nil {
		return fmt.Errorf("read maps dir: %w", err)
	}

	slog.Info("scan static maps", "dir", mapsDir, "count", len(files))

	// 4. Construire index asset_translations (map_name → map_id)
	nameIndex, err := metaRepo.GetAssetNameIndex(ctx, "map")
	if err != nil {
		return fmt.Errorf("build name index: %w", err)
	}
	slog.Info("name index built", "entries", len(nameIndex))

	// 5. Traiter chaque fichier
	var (
		matched   int
		unmatched []UnmatchedMap
	)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		ext := strings.ToLower(filepath.Ext(filename))
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
			continue
		}

		// Extraire nom de map (retirer extension + suffixes)
		mapName := extractMapName(filename)

		// Lookup map_id
		mapID, ok := nameIndex[mapName]
		if !ok {
			// Tentative fuzzy (retirer " - Ranked", "Heavies", etc.)
			mapID, ok = fuzzyLookup(mapName, nameIndex)
		}

		if !ok {
			slog.Warn("unmatched map file", "filename", filename, "extracted_name", mapName)
			unmatched = append(unmatched, UnmatchedMap{
				Filename:      filename,
				ExtractedName: mapName,
			})
			continue
		}

		// Upsert dans map_images_registry
		localPath := fmt.Sprintf("/static/maps/%s", filename)

		if dryRun {
			slog.Info("would upsert",
				"map_id", mapID,
				"map_name", mapName,
				"local_path", localPath,
			)
		} else {
			if err := metaRepo.UpsertMapImageRegistry(ctx, "halo_infinite", mapID, localPath); err != nil {
				slog.Warn("upsert failed", "map_id", mapID, "err", err)
				continue
			}
			slog.Info("upserted",
				"map_id", mapID,
				"map_name", mapName,
				"local_path", localPath,
			)
		}

		matched++
	}

	// 6. Rapport final
	slog.Info("=== RÉSUMÉ ===")
	slog.Info("matched", "count", matched)
	slog.Info("unmatched", "count", len(unmatched))

	// 7. Écrire CSV des maps non reconnues
	if len(unmatched) > 0 {
		csvPath := defaultUnmatchedCSVPath(rootDir)
		if err := os.MkdirAll(filepath.Dir(csvPath), 0o755); err != nil {
			slog.Warn("failed to create unmatched maps dir", "path", filepath.Dir(csvPath), "err", err)
		} else if err := writeUnmatchedCSV(csvPath, unmatched); err != nil {
			slog.Warn("failed to write CSV", "err", err)
		} else {
			slog.Info("unmatched maps written", "path", csvPath)
		}
	}

	return nil
}

func defaultUnmatchedCSVPath(repoRoot string) string {
	return filepath.Join(repoRoot, "data", "investigation", "maps", "unmatched_maps.csv")
}

// extractMapName retire l'extension + suffixes courants.
func extractMapName(filename string) string {
	// Retirer extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Normaliser espaces
	name = strings.TrimSpace(name)

	// Retirer suffixes fréquents (à adapter selon fichiers réels)
	// Ex: "Aquarius - Ranked.png" → "Aquarius"
	// Ex: "Breaker Heavies.png" → "Breaker" (si "Breaker" existe dans asset_translations)

	return name
}

// fuzzyLookup tente de trouver une correspondance en retirant suffixes courants.
func fuzzyLookup(mapName string, index map[string]string) (string, bool) {
	// Retirer " - Ranked", " Heavies", " Firefight", etc.
	suffixes := []string{
		" - Ranked",
		" Heavies",
		" Firefight",
		" (Beta)",
		" - Team Slayer",
	}

	for _, suffix := range suffixes {
		candidate := strings.TrimSuffix(mapName, suffix)
		if candidate != mapName {
			if mapID, ok := index[candidate]; ok {
				return mapID, true
			}
		}
	}

	return "", false
}

// UnmatchedMap représente un fichier non reconnu.
type UnmatchedMap struct {
	Filename      string
	ExtractedName string
}

// writeUnmatchedCSV écrit les maps non reconnues dans un CSV.
func writeUnmatchedCSV(path string, unmatched []UnmatchedMap) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header
	if err := w.Write([]string{"Filename", "ExtractedName", "Notes"}); err != nil {
		return err
	}

	// Rows
	for _, u := range unmatched {
		if err := w.Write([]string{u.Filename, u.ExtractedName, ""}); err != nil {
			return err
		}
	}

	return w.Error()
}
