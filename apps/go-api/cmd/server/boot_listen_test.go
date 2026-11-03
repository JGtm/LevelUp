package main

import (
	"errors"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/instancelock"
)

// TestBindBootListener_EveryProfileTakesTheInstanceLock : le verrou d'instance n'est pas réservé au mode desktop.
// Un serveur ordinaire (hébergé, développement, démo) le tient aussi, de sorte qu'une restauration ou un second
// serveur sur la même racine le trouve pris (S2).
func TestBindBootListener_EveryProfileTakesTheInstanceLock(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	first := &bootState{cfg: cfg, flags: serverFlags{Mode: serverModeDefault}}
	ln := bindBootListener(first, "127.0.0.1:0")
	t.Cleanup(func() { _ = ln.Close(); first.releaseInstanceLock() })
	if first.instanceLockErr != nil || first.instanceLock == nil {
		t.Fatalf("le premier serveur doit tenir le verrou : %v", first.instanceLockErr)
	}

	if l, err := instancelock.Acquire(cfg.PathResolver().InstanceLockPath()); err == nil {
		_ = l.Release()
		t.Fatal("un serveur ordinaire en marche doit tenir le verrou de sa racine")
	} else if !errors.Is(err, instancelock.ErrInstanceDejaLancee) {
		t.Fatalf("verrou : %v, attendu ErrInstanceDejaLancee", err)
	}

	second := &bootState{cfg: cfg, flags: serverFlags{Mode: serverModeDefault}}
	ln2 := bindBootListener(second, "127.0.0.1:0")
	t.Cleanup(func() { _ = ln2.Close() })
	if !errors.Is(second.instanceLockErr, instancelock.ErrInstanceDejaLancee) {
		t.Fatalf("un second serveur sur la même racine doit trouver le verrou pris : %v", second.instanceLockErr)
	}
}
