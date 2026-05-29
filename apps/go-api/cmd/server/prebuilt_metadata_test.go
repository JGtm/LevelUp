// Package main — prebuilt_metadata_test.go : extraction de metadata-prebuilt.zip.
//
// Tag cgo pour cohérence avec les autres tests du package (qui compilent main,
// lequel dépend de duckdb/cgo). L'extraction elle-même n'utilise pas duckdb.
//
//go:build cgo

package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
)

// writeFakePrebuiltZip écrit un zip contenant une entrée metadata.duckdb avec le
// contenu fourni, dans le dossier warehouse du titre.
func writeFakePrebuiltZip(t *testing.T, pr *titlePkg.PathResolver, slug, content string) {
	t.Helper()
	if err := os.MkdirAll(pr.WarehouseDir(slug), 0o755); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(pr.WarehouseDir(slug), prebuiltMetadataZipName)
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	w, err := zw.Create(prebuiltMetadataEntry)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestExtractPrebuiltMetadataIfAbsent(t *testing.T) {
	slug := titlePkg.DefaultSlug

	// --- Cas 1 : metadata.duckdb absent + zip présent → extraction ---
	t.Run("absent_extracts_from_zip", func(t *testing.T) {
		pr := titlePkg.NewPathResolver(t.TempDir())
		writeFakePrebuiltZip(t, pr, slug, "PREBUILT-CONTENT")

		if err := extractPrebuiltMetadataIfAbsent(pr, slug); err != nil {
			t.Fatalf("extraction: %v", err)
		}
		got, err := os.ReadFile(pr.MetadataDBPath(slug))
		if err != nil {
			t.Fatalf("metadata.duckdb devrait exister: %v", err)
		}
		if string(got) != "PREBUILT-CONTENT" {
			t.Errorf("contenu = %q, want PREBUILT-CONTENT", got)
		}
	})

	// --- Cas 2 : metadata.duckdb existe déjà → no-op (jamais écrasé) ---
	t.Run("existing_db_not_overwritten", func(t *testing.T) {
		pr := titlePkg.NewPathResolver(t.TempDir())
		writeFakePrebuiltZip(t, pr, slug, "PREBUILT-CONTENT")

		// Pré-créer une base existante avec un contenu différent.
		metaPath := pr.MetadataDBPath(slug)
		if err := os.WriteFile(metaPath, []byte("EXISTING-DB"), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := extractPrebuiltMetadataIfAbsent(pr, slug); err != nil {
			t.Fatalf("extraction: %v", err)
		}
		got, _ := os.ReadFile(metaPath)
		if string(got) != "EXISTING-DB" {
			t.Errorf("la base existante a été écrasée: %q", got)
		}
	})

	// --- Cas 3 : pas de zip + pas de base → no-op silencieux (pas d'erreur) ---
	t.Run("no_zip_noop", func(t *testing.T) {
		pr := titlePkg.NewPathResolver(t.TempDir())
		if err := os.MkdirAll(pr.WarehouseDir(slug), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := extractPrebuiltMetadataIfAbsent(pr, slug); err != nil {
			t.Fatalf("devrait être un no-op sans erreur: %v", err)
		}
		if _, err := os.Stat(pr.MetadataDBPath(slug)); !os.IsNotExist(err) {
			t.Errorf("metadata.duckdb ne devrait pas exister (stat err = %v)", err)
		}
	})
}
