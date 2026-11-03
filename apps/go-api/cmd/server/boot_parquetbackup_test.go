package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/ops/parquetbackup"
	"levelup/go-api/internal/platform/gameproc"
	"levelup/go-api/internal/platform/settings"
)

type fakeBackupRunner struct {
	due  bool
	runs int
	err  error
	dest string
}

func (f *fakeBackupRunner) Due(dest string, _, _ time.Duration, _ time.Time) bool {
	f.dest = dest
	return f.due
}

func (f *fakeBackupRunner) RunNow(context.Context) (*parquetbackup.Manifest, error) {
	f.runs++
	return nil, f.err
}

func backupSettings(dir string) fakeSettings {
	s := settings.Defaults()
	s.BackupDir = dir
	return fakeSettings{cfg: s}
}

func TestBackupTick_SansDossierNeFaitRien(t *testing.T) {
	r := &fakeBackupRunner{due: true}
	backupTick(context.Background(), r, backupSettings("  "))
	if r.runs != 0 {
		t.Fatalf("aucun dossier choisi : %d passage(s), attendu 0", r.runs)
	}
}

func TestBackupTick_DuLanceLePassageEtNonDuNon(t *testing.T) {
	r := &fakeBackupRunner{due: true}
	backupTick(context.Background(), r, backupSettings(`D:\Sauvegardes`))
	if r.runs != 1 || r.dest != `D:\Sauvegardes` {
		t.Fatalf("dû : %d passage(s) vers %q, attendu 1 vers le dossier choisi", r.runs, r.dest)
	}
	r2 := &fakeBackupRunner{due: false}
	backupTick(context.Background(), r2, backupSettings(`D:\Sauvegardes`))
	if r2.runs != 0 {
		t.Fatalf("pas dû : %d passage(s), attendu 0", r2.runs)
	}
}

func TestBackupTick_ReglagesIllisiblesIgnoreLeTic(t *testing.T) {
	r := &fakeBackupRunner{due: true}
	backupTick(context.Background(), r, fakeSettings{err: errors.New("illisible")})
	if r.runs != 0 {
		t.Fatalf("réglages illisibles : %d passage(s), attendu 0", r.runs)
	}
}

// TestBackupTick_PendantUnePartieAttendLaFin : un jeu qui tourne ferme la porte du travail lourd ; le
// passage part à la fermeture du jeu, jamais pendant la partie.
func TestBackupTick_PendantUnePartieAttendLaFin(t *testing.T) {
	mode := gameproc.NewModeRetrying(10 * time.Millisecond)
	prev := gameproc.SetDefault(mode)
	defer gameproc.SetDefault(prev)
	running := true
	l := gameproc.ListerFunc(func() ([]string, error) {
		if running {
			return []string{"HaloInfinite.exe"}, nil
		}
		return nil, nil
	})
	cfg := settings.Defaults()
	w := newGameWatcher(fakeSettings{cfg: cfg}, l, mode, nil)
	w.Poll(context.Background())
	if !mode.Running() {
		t.Skip("le nom d'exécutable de test n'est pas celui du registre")
	}

	r := &fakeBackupRunner{due: true}
	done := make(chan struct{})
	go func() {
		backupTick(context.Background(), r, backupSettings(`D:\Sauvegardes`))
		close(done)
	}()
	select {
	case <-done:
		t.Fatal("le passage est parti pendant la partie")
	case <-time.After(100 * time.Millisecond):
	}
	running = false
	w.Poll(context.Background())
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("le passage n'est pas parti à la fermeture du jeu")
	}
	if r.runs != 1 {
		t.Fatalf("passages = %d, attendu 1", r.runs)
	}
}
