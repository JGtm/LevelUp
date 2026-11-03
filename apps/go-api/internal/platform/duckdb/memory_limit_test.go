package duckdb

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// readMemoryLimit lit le plafond mémoire que DuckDB applique à la base (PRAGMA memory_limit).
func readMemoryLimit(t *testing.T, db *DB) string {
	t.Helper()
	var v string
	if err := db.SQLDb().QueryRow(`SELECT current_setting('memory_limit')`).Scan(&v); err != nil {
		t.Fatalf("lecture du plafond : %v", err)
	}
	return v
}

// TestMemoryLimit_AppliedOnOpenAndLive : le réglage pose le memory_limit de chaque base à l'ouverture et,
// quand il change à chaud, sur les bases déjà ouvertes ; 0 rend la valeur par défaut du serveur.
func TestMemoryLimit_AppliedOnOpenAndLive(t *testing.T) {
	t.Setenv("LEVELUP_DUCKDB_MEMORY_LIMIT", "")
	SetMemoryLimitMB(0)
	t.Cleanup(func() { SetMemoryLimitMB(0) })

	base, err := OpenReadWrite(filepath.Join(t.TempDir(), "defaut.duckdb"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = base.Close() }()
	if got := EffectiveMemoryLimit(); got != "512MB" {
		t.Fatalf("plafond par défaut = %q, attendu 512MB", got)
	}

	SetMemoryLimitMB(4096)
	fresh, err := OpenReadWrite(filepath.Join(t.TempDir(), "plafonnee.duckdb"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = fresh.Close() }()
	if got := readMemoryLimit(t, fresh); !strings.HasPrefix(got, "4.0 GiB") {
		t.Fatalf("PRAGMA memory_limit d'une base ouverte après le réglage = %q, attendu 4 GiB", got)
	}
	if got := EffectiveMemoryLimit(); got != "4096MiB" {
		t.Fatalf("plafond effectif = %q, attendu 4096MiB", got)
	}

	// À chaud : la base ouverte AVANT le réglage reçoit le nouveau plafond.
	before := readMemoryLimit(t, base)
	done, failed := ApplyMemoryLimitToOpen(context.Background())
	if failed != 0 || done < 2 {
		t.Fatalf("ApplyMemoryLimitToOpen : %d bases, %d échecs ; attendu au moins 2 bases, 0 échec", done, failed)
	}
	after := readMemoryLimit(t, base)
	if before == after {
		t.Fatalf("la base ouverte avant le réglage garde son plafond (%s)", before)
	}
	if after != readMemoryLimit(t, fresh) {
		t.Fatalf("les deux bases doivent partager le plafond : %s contre %s", after, readMemoryLimit(t, fresh))
	}

	SetMemoryLimitMB(0)
	if got := EffectiveMemoryLimit(); got != "512MB" {
		t.Fatalf("plafond après retour à 0 = %q, attendu 512MB", got)
	}
}
