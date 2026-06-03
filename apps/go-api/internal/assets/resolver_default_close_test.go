package assets

import (
	"context"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/platform/duckdb"
)

// TestDefaultResolver_Close_ReleasesIndexStoreHandle vérifie que Close ferme le
// handle RW que le DuckDBIndexStore tient sur metadata.duckdb. Non-régression
// de la fuite refCount diagnostiquée le 2026-06-02 : l'index store restait
// orphelin au shutdown (3e détenteur du leak, cf. INCIDENT_2026-05-21). Avant
// le fix, DefaultResolver.Close ne flushait que la WriteQueue.
func TestDefaultResolver_Close_ReleasesIndexStoreHandle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metadata.duckdb")

	store := NewDuckDBIndexStore(path)
	// EnsureTable ouvre le handle RW partagé (clé cache "rw:"+path) en lazy.
	if err := store.EnsureTable(context.Background()); err != nil {
		t.Fatalf("EnsureTable: %v", err)
	}
	if duckdb.DumpCachedLeaks()["rw:"+path] == 0 {
		t.Fatalf("précondition: le handle index store devrait être ouvert après EnsureTable")
	}

	r := NewDefaultResolver(nil, store, nil, nil, nil)
	if err := r.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if refs := duckdb.DumpCachedLeaks()["rw:"+path]; refs != 0 {
		t.Fatalf("handle index store non libéré après Close: refCount=%d (attendu 0)", refs)
	}
}
