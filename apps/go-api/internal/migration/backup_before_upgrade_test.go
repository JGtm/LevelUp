package migration

// backup_before_upgrade_test.go — la copie avant montée : fichiers et noms, WAL présent ou
// absent, rétention, refus faute de place.

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func writeBase(t *testing.T, dir, name string, size int) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, bytes.Repeat([]byte{'d'}, size), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func plentyOfSpace(string) (uint64, error) { return 1 << 40, nil }

func listNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	slices.Sort(names)
	return names
}

func TestBackupBeforeUpgrade_CopieBaseEtWal(t *testing.T) {
	src, dest := t.TempDir(), filepath.Join(t.TempDir(), "migrations")
	db := writeBase(t, src, "metadata.duckdb", 100)
	if err := os.WriteFile(db+".wal", []byte("wal-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 1, 14, 5, 9, 0, time.UTC)

	res, err := BackupBeforeUpgrade(context.Background(), BackupParams{
		DBPath: db, DestDir: dest, Version: 62, Now: now, FreeBytes: plentyOfSpace,
	})
	if err != nil {
		t.Fatalf("BackupBeforeUpgrade: %v", err)
	}
	want := []string{"metadata.v62.20261101-140509.duckdb", "metadata.v62.20261101-140509.duckdb.wal"}
	if got := listNames(t, dest); !slices.Equal(got, want) {
		t.Fatalf("fichiers = %v, attendu %v", got, want)
	}
	if len(res.Copied) != 2 {
		t.Errorf("Copied = %v", res.Copied)
	}
	if b, _ := os.ReadFile(filepath.Join(dest, want[1])); string(b) != "wal-bytes" {
		t.Errorf("contenu du WAL copié = %q", b)
	}
	if st, _ := os.Stat(db); st == nil || st.Size() != 100 {
		t.Error("la base source a été modifiée")
	}
}

func TestBackupBeforeUpgrade_SansWal_CopieLaBaseSeule(t *testing.T) {
	src, dest := t.TempDir(), t.TempDir()
	db := writeBase(t, src, "shared_social.duckdb", 10)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	if _, err := BackupBeforeUpgrade(context.Background(), BackupParams{
		DBPath: db, DestDir: dest, Version: 0, Now: now, FreeBytes: plentyOfSpace,
	}); err != nil {
		t.Fatal(err)
	}
	if got, want := listNames(t, dest), []string{"shared_social.v0.20260102-030405.duckdb"}; !slices.Equal(got, want) {
		t.Errorf("fichiers = %v, attendu %v", got, want)
	}
}

func TestBackupBeforeUpgrade_RetentionDeux_AutresBasesIntactes(t *testing.T) {
	src, dest := t.TempDir(), t.TempDir()
	db := writeBase(t, src, "metadata.duckdb", 10)
	other := writeBase(t, dest, "shared.v1.20200101-000000.duckdb", 5)
	base := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)

	for i := range 4 {
		if err := os.WriteFile(db+".wal", []byte("w"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := BackupBeforeUpgrade(context.Background(), BackupParams{
			DBPath: db, DestDir: dest, Version: i, Now: base.Add(time.Duration(i) * time.Hour), FreeBytes: plentyOfSpace,
		}); err != nil {
			t.Fatalf("copie %d: %v", i, err)
		}
	}
	want := []string{
		"metadata.v2.20261101-120000.duckdb", "metadata.v2.20261101-120000.duckdb.wal",
		"metadata.v3.20261101-130000.duckdb", "metadata.v3.20261101-130000.duckdb.wal",
		filepath.Base(other),
	}
	slices.Sort(want)
	if got := listNames(t, dest); !slices.Equal(got, want) {
		t.Errorf("après 4 copies : %v, attendu %v", got, want)
	}
}

func TestBackupBeforeUpgrade_EspaceInsuffisant_RefusSansCopie(t *testing.T) {
	src, dest := t.TempDir(), t.TempDir()
	db := writeBase(t, src, "metadata.duckdb", 100)
	if err := os.WriteFile(db+".wal", bytes.Repeat([]byte{'w'}, 50), 0o644); err != nil {
		t.Fatal(err)
	}
	free := func(string) (uint64, error) { return 149, nil } // 150 octets à copier

	_, err := BackupBeforeUpgrade(context.Background(), BackupParams{
		DBPath: db, DestDir: dest, Version: 1, Now: time.Now(), FreeBytes: free,
	})
	if !errors.Is(err, ErrInsufficientBackupSpace) {
		t.Fatalf("err = %v, attendu ErrInsufficientBackupSpace", err)
	}
	if got := listNames(t, dest); len(got) != 0 {
		t.Errorf("fichiers copiés malgré le refus : %v", got)
	}

	// Exactement la place nécessaire : accepté.
	free = func(string) (uint64, error) { return 150, nil }
	if _, err := BackupBeforeUpgrade(context.Background(), BackupParams{
		DBPath: db, DestDir: dest, Version: 1, Now: time.Now(), FreeBytes: free,
	}); err != nil {
		t.Errorf("place exacte refusée: %v", err)
	}
}

func TestBackupBeforeUpgrade_ErreursDeSource(t *testing.T) {
	dest := t.TempDir()
	_, err := BackupBeforeUpgrade(context.Background(), BackupParams{
		DBPath: filepath.Join(t.TempDir(), "absente.duckdb"), DestDir: dest, Now: time.Now(), FreeBytes: plentyOfSpace,
	})
	if err == nil {
		t.Error("base absente : erreur attendue")
	}
	db := writeBase(t, t.TempDir(), "metadata.duckdb", 1)
	_, err = BackupBeforeUpgrade(context.Background(), BackupParams{
		DBPath: db, DestDir: dest, Now: time.Now(),
		FreeBytes: func(string) (uint64, error) { return 0, errors.New("volume illisible") },
	})
	if err == nil || errors.Is(err, ErrInsufficientBackupSpace) {
		t.Errorf("espace libre illisible : err = %v, attendu une erreur distincte du refus d'espace", err)
	}
}
