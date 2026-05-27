//go:build cgo

// Package ops — media_reconcile_cgo_test.go : tests CGO pour
// ReconcileOrphanedMediaFiles. Couvre la détection d'entrées DB dont le
// fichier disque a changé d'extension (ex: conversion locale .mp4 → .mkv).
package ops

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

type reconcileSetup struct {
	db          *sql.DB
	playerSlug  string
	capturesDir string
	store       MediaPathStore
}

// setupReconcileTest crée une DB media_files vide, un dossier captures pour
// playerSlug, et insère une entrée DB pointant vers {slug}/clip.mp4 (fichier
// non créé par défaut — au test de placer ce qu'il veut sur disque).
func setupReconcileTest(t *testing.T) *reconcileSetup {
	t.Helper()
	_, db := openDiagDB(t)
	ctx := context.Background()
	if err := ensureMediaTables(ctx, db); err != nil {
		t.Fatalf("ensureMediaTables: %v", err)
	}

	capturesBase := t.TempDir()
	playerSlug := "TestPlayer"
	capturesDir := filepath.Join(capturesBase, playerSlug)
	if err := os.MkdirAll(capturesDir, 0o755); err != nil {
		t.Fatalf("mkdir captures: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO media_files (player_slug, file_path, file_name, file_stem, file_ext, kind)
		VALUES (?, ?, ?, ?, ?, ?)
	`, playerSlug, playerSlug+"/clip.mp4", "clip.mp4", "clip", ".mp4", "video"); err != nil {
		t.Fatalf("INSERT fixture: %v", err)
	}

	return &reconcileSetup{
		db:          db,
		playerSlug:  playerSlug,
		capturesDir: capturesDir,
		store:       MediaPathStore{CapturesBase: capturesBase},
	}
}

// currentEntry lit l'entrée DB du joueur courant (1 seule attendue).
func (s *reconcileSetup) currentEntry(t *testing.T) (filePath, fileExt string) {
	t.Helper()
	if err := s.db.QueryRowContext(context.Background(),
		"SELECT file_path, file_ext FROM media_files WHERE player_slug = ?",
		s.playerSlug,
	).Scan(&filePath, &fileExt); err != nil {
		t.Fatalf("SELECT entrée: %v", err)
	}
	return filePath, fileExt
}

// ─────────────────────────────────────────────────────────────────────────────

func TestReconcile_FilePresentNoChange(t *testing.T) {
	s := setupReconcileTest(t)
	if err := os.WriteFile(filepath.Join(s.capturesDir, "clip.mp4"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	n, err := ReconcileOrphanedMediaFiles(context.Background(), s.db, s.playerSlug, s.store)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if n != 0 {
		t.Errorf("updated = %d, want 0 (fichier toujours présent)", n)
	}
	fp, fe := s.currentEntry(t)
	if fp != s.playerSlug+"/clip.mp4" || fe != ".mp4" {
		t.Errorf("entrée modifiée par erreur: file_path=%q file_ext=%q", fp, fe)
	}
}

func TestReconcile_OrphanWithUniqueMkvMatch(t *testing.T) {
	s := setupReconcileTest(t)
	if err := os.WriteFile(filepath.Join(s.capturesDir, "clip.mkv"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	n, err := ReconcileOrphanedMediaFiles(context.Background(), s.db, s.playerSlug, s.store)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if n != 1 {
		t.Errorf("updated = %d, want 1", n)
	}
	fp, fe := s.currentEntry(t)
	if fe != ".mkv" {
		t.Errorf("file_ext = %q, want .mkv", fe)
	}
	if fp != s.playerSlug+"/clip.mkv" {
		t.Errorf("file_path = %q, want %s/clip.mkv", fp, s.playerSlug)
	}
}

func TestReconcile_OrphanNoMatch(t *testing.T) {
	s := setupReconcileTest(t)
	n, err := ReconcileOrphanedMediaFiles(context.Background(), s.db, s.playerSlug, s.store)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if n != 0 {
		t.Errorf("updated = %d, want 0 (pas de match)", n)
	}
	fp, fe := s.currentEntry(t)
	if fp != s.playerSlug+"/clip.mp4" || fe != ".mp4" {
		t.Errorf("entrée modifiée par erreur: file_path=%q file_ext=%q", fp, fe)
	}
}

func TestReconcile_AmbiguousMatchesSkipped(t *testing.T) {
	s := setupReconcileTest(t)
	if err := os.WriteFile(filepath.Join(s.capturesDir, "clip.mkv"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.capturesDir, "clip.mov"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	n, err := ReconcileOrphanedMediaFiles(context.Background(), s.db, s.playerSlug, s.store)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if n != 0 {
		t.Errorf("updated = %d, want 0 (ambigu)", n)
	}
	fp, fe := s.currentEntry(t)
	if fp != s.playerSlug+"/clip.mp4" || fe != ".mp4" {
		t.Errorf("entrée modifiée par erreur: file_path=%q file_ext=%q", fp, fe)
	}
}

func TestReconcile_Idempotent(t *testing.T) {
	s := setupReconcileTest(t)
	if err := os.WriteFile(filepath.Join(s.capturesDir, "clip.mkv"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	n1, err := ReconcileOrphanedMediaFiles(context.Background(), s.db, s.playerSlug, s.store)
	if err != nil {
		t.Fatalf("1er run: %v", err)
	}
	if n1 != 1 {
		t.Errorf("1er run updated = %d, want 1", n1)
	}

	n2, err := ReconcileOrphanedMediaFiles(context.Background(), s.db, s.playerSlug, s.store)
	if err != nil {
		t.Fatalf("2e run: %v", err)
	}
	if n2 != 0 {
		t.Errorf("2e run updated = %d, want 0 (idempotent)", n2)
	}
}

func TestReconcile_OnlyTargetsRequestedPlayer(t *testing.T) {
	_, db := openDiagDB(t)
	ctx := context.Background()
	if err := ensureMediaTables(ctx, db); err != nil {
		t.Fatalf("ensureMediaTables: %v", err)
	}
	capturesBase := t.TempDir()
	store := MediaPathStore{CapturesBase: capturesBase}

	for _, slug := range []string{"PlayerA", "PlayerB"} {
		dir := filepath.Join(capturesBase, slug)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		// Chacun a un .mkv sur disque, mais leur entrée DB pointe vers .mp4.
		fname := "clip_" + slug + ".mkv"
		if err := os.WriteFile(filepath.Join(dir, fname), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		stem := "clip_" + slug
		if _, err := db.ExecContext(ctx, `
			INSERT INTO media_files (player_slug, file_path, file_name, file_stem, file_ext, kind)
			VALUES (?, ?, ?, ?, ?, ?)
		`, slug, slug+"/"+stem+".mp4", stem+".mp4", stem, ".mp4", "video"); err != nil {
			t.Fatal(err)
		}
	}

	n, err := ReconcileOrphanedMediaFiles(ctx, db, "PlayerA", store)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if n != 1 {
		t.Errorf("updated = %d, want 1 (seulement PlayerA)", n)
	}

	// PlayerB doit rester intact.
	var fpB string
	if err := db.QueryRowContext(ctx,
		"SELECT file_path FROM media_files WHERE player_slug = ?", "PlayerB",
	).Scan(&fpB); err != nil {
		t.Fatal(err)
	}
	if fpB != "PlayerB/clip_PlayerB.mp4" {
		t.Errorf("PlayerB modifié par erreur: file_path=%q", fpB)
	}
}
