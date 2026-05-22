//go:build integration

package duckdb

import (
	"path/filepath"
	"testing"
)

// TestDumpCachedLeaks_EmptyMap : aucune DB ouverte → DumpCachedLeaks retourne nil.
func TestDumpCachedLeaks_EmptyMap(t *testing.T) {
	// Vider explicitement le cache pour isoler le test des autres
	// (les tests précédents peuvent avoir laissé des entrées).
	openDBsMu.Lock()
	openDBs = map[string]*cachedDB{}
	openDBsMu.Unlock()

	leaks := DumpCachedLeaks()
	if leaks != nil {
		t.Errorf("expected nil leaks on empty cache, got %v", leaks)
	}
}

// TestDumpCachedLeaks_DetectsLeak : ouvre une DB sans la fermer → DumpCachedLeaks
// reporte refCount=1 sur la clé.
func TestDumpCachedLeaks_DetectsLeak(t *testing.T) {
	openDBsMu.Lock()
	openDBs = map[string]*cachedDB{}
	openDBsMu.Unlock()

	path := filepath.Join(t.TempDir(), "leak_test.duckdb")
	db, err := OpenReadWriteShared(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	// Volontairement PAS de db.Close() pour simuler une fuite.

	leaks := DumpCachedLeaks()
	if len(leaks) != 1 {
		t.Errorf("expected 1 leak, got %d : %v", len(leaks), leaks)
	}
	key := "rw:" + path
	if refs, ok := leaks[key]; !ok {
		t.Errorf("expected key %q in leaks, got keys %v", key, leaks)
	} else if refs != 1 {
		t.Errorf("expected refCount=1 for %q, got %d", key, refs)
	}

	// Nettoyer pour ne pas polluer les autres tests.
	_ = db.Close()
}

// TestDumpCachedLeaks_DoubleOpenSameKey : ouverture double sans 2 closes →
// refCount=2 detected.
func TestDumpCachedLeaks_DoubleOpenSameKey(t *testing.T) {
	openDBsMu.Lock()
	openDBs = map[string]*cachedDB{}
	openDBsMu.Unlock()

	path := filepath.Join(t.TempDir(), "double_open.duckdb")
	db1, err := OpenReadWriteShared(path)
	if err != nil {
		t.Fatalf("open1: %v", err)
	}
	db2, err := OpenReadWriteShared(path)
	if err != nil {
		t.Fatalf("open2: %v", err)
	}

	leaks := DumpCachedLeaks()
	if len(leaks) != 1 {
		t.Errorf("expected 1 cache entry, got %d", len(leaks))
	}
	if refs := leaks["rw:"+path]; refs != 2 {
		t.Errorf("expected refCount=2, got %d", refs)
	}

	// Si on ferme 1 fois, refCount = 1 (toujours un leak)
	_ = db1.Close()
	leaks = DumpCachedLeaks()
	if refs := leaks["rw:"+path]; refs != 1 {
		t.Errorf("after 1 close: expected refCount=1, got %d", refs)
	}

	// Si on ferme 2 fois, refCount = 0 et la clé est supprimée du cache.
	_ = db2.Close()
	leaks = DumpCachedLeaks()
	if leaks != nil {
		t.Errorf("after 2 closes: expected nil leaks, got %v", leaks)
	}
}
