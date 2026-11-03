//go:build cgo

package main

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops/parquetbackup"
	"levelup/go-api/internal/platform/instancelock"
)

func testRoots(t *testing.T) (*title.PathResolver, string) {
	t.Helper()
	root := t.TempDir()
	data := filepath.Join(root, "data")
	return title.NewPathResolverRoots(title.Roots{Resources: root, Data: data, DataSet: true}), data
}

func TestBackupRestore_RefusesWhileInstanceLockHeld(t *testing.T) {
	_, data := testRoots(t)
	lock, err := instancelock.Acquire(filepath.Join(data, "levelup.lock"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = lock.Release() }()
	cfg := &config.AppConfig{}
	err = runBackupRestore(cfg, []string{"--from", filepath.Join(t.TempDir(), "jeu"), "--to", data}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "tourne") {
		t.Fatalf("restauration sous un verrou d'instance tenu : %v, attendu un refus", err)
	}
}

func TestBackupRestore_RequiresFromAndTo(t *testing.T) {
	if err := runBackupRestore(&config.AppConfig{}, nil, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "usage") {
		t.Fatalf("sans arguments : %v, attendu l'usage", err)
	}
}

// TestBackupRestore_EndToEnd : un jeu écrit par la sauvegarde périodique se restaure par la commande dans
// une racine neuve, mêmes lignes.
func TestBackupRestore_EndToEnd(t *testing.T) {
	src, srcData := testRoots(t)
	dbPath := src.MetadataDBPath("halo_infinite")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{`CREATE TABLE medal_definitions (medal_id BIGINT, name VARCHAR)`,
		`INSERT INTO medal_definitions VALUES (1,'Killing Spree'),(2,'Double Kill')`} {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	_ = db.Close()
	dest := t.TempDir()
	m, err := parquetbackup.Run(context.Background(), parquetbackup.Options{Paths: src, AuthDir: filepath.Join(srcData, "auth"),
		Dest: dest, Keep: 3, AppVersion: "test", LeaseWait: 50 * time.Millisecond, RetryWait: 50 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}

	dstPR, dstData := testRoots(t)
	var out bytes.Buffer
	if err := runBackupRestore(&config.AppConfig{}, []string{"--from", filepath.Join(dest, m.Name), "--to", dstData}, &out); err != nil {
		t.Fatalf("restauration : %v\n%s", err, out.String())
	}
	got, err := sql.Open("duckdb", dstPR.MetadataDBPath("halo_infinite")+"?access_mode=read_only")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = got.Close() }()
	var n int
	if err := got.QueryRow(`SELECT count(*) FROM medal_definitions`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("lignes restaurées = %d (err=%v), attendu 2", n, err)
	}
	if !strings.Contains(out.String(), "halo_infinite:metadata") {
		t.Fatalf("le bilan doit nommer la base restaurée :\n%s", out.String())
	}
	// Seconde restauration sur la racine désormais peuplée : refusée sans --force.
	if err := runBackupRestore(&config.AppConfig{}, []string{"--from", filepath.Join(dest, m.Name), "--to", dstData}, &out); err == nil {
		t.Fatal("restauration sur des bases existantes sans --force : un refus était attendu")
	}
}
