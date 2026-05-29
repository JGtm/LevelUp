package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/domain/title"
)

const (
	prebuiltMetadataZipName = "metadata-prebuilt.zip"
	prebuiltMetadataEntry   = "metadata.duckdb"
)

// extractPrebuiltMetadataIfAbsent décompresse metadata-prebuilt.zip vers
// metadata.duckdb si cette base n'existe pas encore.
//
// Sur un clone frais, metadata.duckdb est absent : sans ce pas, les référentiels
// (career_ranks, playlists, citations…) seraient vides jusqu'à un fetch live.
// Le zip (tracké git, dans le dossier warehouse) contient une base metadata prête
// à l'emploi ; on l'extrait AVANT runMigrations, qui applique ensuite tout schéma
// plus récent (idempotent).
//
// No-op si :
//   - metadata.duckdb existe déjà (install existant — on ne l'écrase jamais) ;
//   - le zip est absent (les migrations créeront une base vide).
//
// Non-fatal pour le caller : une extraction échouée laisse les migrations créer
// une base vide, le serveur démarre quand même.
func extractPrebuiltMetadataIfAbsent(pr *title.PathResolver, titleSlug string) error {
	metaPath := pr.MetadataDBPath(titleSlug)
	if _, err := os.Stat(metaPath); err == nil {
		return nil // base déjà présente — ne pas écraser
	}

	zipPath := filepath.Join(pr.WarehouseDir(titleSlug), prebuiltMetadataZipName)
	if _, err := os.Stat(zipPath); err != nil {
		slog.Debug("prebuilt metadata: zip absent, skip (migrations créeront une base vide)",
			"zip", zipPath)
		return nil
	}

	if err := unzipSingleEntry(zipPath, prebuiltMetadataEntry, metaPath); err != nil {
		return fmt.Errorf("extractPrebuiltMetadata: %w", err)
	}
	slog.Info("prebuilt metadata: base extraite depuis le zip", "dest", metaPath)
	return nil
}

// unzipSingleEntry extrait l'entrée `entryName` de l'archive `zipPath` vers `dest`
// (write-to-temp + rename atomique pour éviter une base partielle si interruption).
func unzipSingleEntry(zipPath, entryName, dest string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("ouverture zip: %w", err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		if f.Name != entryName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("ouverture entrée %s: %w", entryName, err)
		}
		defer rc.Close()

		tmp := dest + ".tmp"
		out, err := os.Create(tmp)
		if err != nil {
			return fmt.Errorf("création fichier temp: %w", err)
		}
		// Source = zip interne tracké git, pas une entrée user → pas de risque
		// de decompression bomb.
		if _, err := io.Copy(out, rc); err != nil { //nolint:gosec // G110: source interne de confiance
			_ = out.Close()
			_ = os.Remove(tmp)
			return fmt.Errorf("copie de %s: %w", entryName, err)
		}
		if err := out.Close(); err != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("fermeture fichier temp: %w", err)
		}
		if err := os.Rename(tmp, dest); err != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("rename %s → %s: %w", tmp, dest, err)
		}
		return nil
	}
	return fmt.Errorf("entrée %q introuvable dans %s", entryName, zipPath)
}
