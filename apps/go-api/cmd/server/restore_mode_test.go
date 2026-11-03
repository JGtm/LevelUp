package main

import (
	"context"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops/parquetbackup"
	"levelup/go-api/internal/platform/instancelock"
)

// TestRunRestoreMode_RefusesWhileAServerHoldsTheRoot : la restauration par le serveur livré ne passe pas outre le
// verrou d'instance d'une racine dont un serveur tourne, --force compris (S2, D-a).
func TestRunRestoreMode_RefusesWhileAServerHoldsTheRoot(t *testing.T) {
	data := filepath.Join(t.TempDir(), "data")
	lock, err := instancelock.Acquire(title.NewPathResolverRoots(title.Roots{Resources: t.TempDir(), Data: data, DataSet: true}).InstanceLockPath())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lock.Release() })
	f := serverFlags{Mode: serverModeRestore, Restore: parquetbackup.RestoreCommand{
		From: filepath.Join(t.TempDir(), "jeu"), To: data, Force: true}}
	if code := runRestoreMode(context.Background(), f); code != restoreModeFailure {
		t.Fatalf("code de sortie = %d, attendu %d : un serveur tient la racine", code, restoreModeFailure)
	}
}
