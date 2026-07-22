//go:build windows

// Package session — store_purge_windows_test.go : reproduction DÉTERMINISTE de la
// sharing violation Windows qui, avec l'ancienne impl, faisait supprimer une session
// VIVANTE par PurgeExpired (erreur de lecture transitoire → os.Remove → déconnexion
// permanente). Windows-only : la course n'existe pas sur les OS POSIX (pas de sharing
// violation à l'ouverture concurrente).
package session_test

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"levelup/go-api/internal/platform/session"
)

// TestStore_PurgeExpired_SharingViolation_NeverDeletes ouvre le fichier de session
// avec un share mode EXCLUSIF (dwShareMode = 0) : tant que ce handle est ouvert,
// l'os.ReadFile de PurgeExpired échoue avec ERROR_SHARING_VIOLATION — exactement la
// course d'un Save concurrent (doublon `air`). Le fichier NE DOIT PAS être supprimé
// et un WARN « conservé » doit être tracé.
func TestStore_PurgeExpired_SharingViolation_NeverDeletes(t *testing.T) {
	sessDir := filepath.Join(t.TempDir(), "sessions")
	store := session.NewStore(sessDir, time.Hour, "test-secret-32-bytesXXXXXXXXXX")

	sess := store.New()
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// L'ID est un UUID v4 (uniquement [0-9a-f-]) → sanitizeID le préserve, donc le
	// chemin sur disque est déterministe.
	target := filepath.Join(sessDir, sess.SessionID+".json")

	ptr, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		t.Fatalf("UTF16PtrFromString: %v", err)
	}
	// Handle EXCLUSIF : dwShareMode = 0 → aucune ouverture concurrente autorisée.
	h, err := syscall.CreateFile(ptr, syscall.GENERIC_READ, 0, nil,
		syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatalf("CreateFile exclusif: %v", err)
	}
	// Fermé AVANT le cleanup de t.TempDir (defer au retour) pour ne pas bloquer la
	// suppression du répertoire temporaire.
	defer func() { _ = syscall.CloseHandle(h) }()

	buf := captureWarnLogs(t)
	if removed := store.PurgeExpired(); removed != 0 {
		t.Errorf("PurgeExpired = %d, attendu 0 (une sharing violation ne supprime pas)", removed)
	}
	if _, statErr := os.Stat(target); statErr != nil {
		t.Errorf("une sharing violation ne doit pas supprimer le fichier de session: %v", statErr)
	}
	if !strings.Contains(buf.String(), "conservé") {
		t.Errorf("PurgeExpired doit tracer un WARN 'conservé' sur erreur de lecture, got: %q", buf.String())
	}
}
