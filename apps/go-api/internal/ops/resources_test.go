// Package ops — resources_test.go : helpers purs du monitoring ressources (A5).
package ops

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
)

func TestEvaluateDiskStatus(t *testing.T) {
	cases := []struct {
		name string
		free uint64
		want string
	}{
		{"confortable (10 Go)", 10 << 30, domain.FreshnessStatusOK},
		{"juste au seuil warn (2 Go)", DiskFreeWarnBytes, domain.FreshnessStatusOK},
		{"sous le seuil warn (1 Go)", 1 << 30, domain.FreshnessStatusWarn},
		{"juste au seuil critical (500 Mo)", DiskFreeCriticalBytes, domain.FreshnessStatusWarn},
		{"sous le seuil critical (100 Mo)", 100 << 20, domain.FreshnessStatusCritical},
	}
	for _, tc := range cases {
		if got := EvaluateDiskStatus(tc.free); got != tc.want {
			t.Errorf("%s : EvaluateDiskStatus(%d) = %q, attendu %q", tc.name, tc.free, got, tc.want)
		}
	}
}

func TestDBFileSize_WithWAL(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.duckdb")
	if err := os.WriteFile(dbPath, make([]byte, 1234), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dbPath+".wal", make([]byte, 56), 0o600); err != nil {
		t.Fatal(err)
	}
	got := DBFileSize("t/test", dbPath)
	if got.SizeBytes != 1234 || got.WalBytes != 56 {
		t.Errorf("DBFileSize = %+v, attendu size=1234 wal=56", got)
	}
	// Base absente : zéros, pas d'erreur.
	missing := DBFileSize("t/missing", filepath.Join(dir, "absent.duckdb"))
	if missing.SizeBytes != 0 || missing.WalBytes != 0 {
		t.Errorf("base absente : %+v, attendu zéros", missing)
	}
}

func TestDirTotalSize(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "player1")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "stats.duckdb"), make([]byte, 100), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "root.bin"), make([]byte, 20), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := DirTotalSize(dir); got != 120 {
		t.Errorf("DirTotalSize = %d, attendu 120", got)
	}
	if got := DirTotalSize(filepath.Join(dir, "absent")); got != 0 {
		t.Errorf("répertoire absent : %d, attendu 0", got)
	}
}
