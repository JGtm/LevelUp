package wire

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/platform/duckdb"
)

// TestServiceRegistry_Close_ReleasesTrackedMetadataHandles vérifie que les
// handles RW "annexes" sur metadata.duckdb enregistrés via TrackMetadataHandle
// (seasons catalog, playlists catalog ouverts hors pool joueur dans NewRouter)
// sont bien fermés par Close(). Non-régression de la fuite refCount
// diagnostiquée le 2026-06-02 : ces handles restaient orphelins au shutdown
// car reg.Close() ne fermait que le PrestigeBundle (cf. INCIDENT_2026-05-21).
func TestServiceRegistry_Close_ReleasesTrackedMetadataHandles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metadata.duckdb")

	// Simule les 2 ouvertures "annexes" (seasons + catalog) : même clé cache
	// "rw:"+path → refCount=2 sur une seule entrée.
	h1, err := duckdb.OpenReadWriteShared(path)
	if err != nil {
		t.Fatalf("OpenReadWriteShared #1: %v", err)
	}
	h2, err := duckdb.OpenReadWriteShared(path)
	if err != nil {
		t.Fatalf("OpenReadWriteShared #2: %v", err)
	}

	reg := &ServiceRegistry{}
	reg.TrackMetadataHandle(h1)
	reg.TrackMetadataHandle(h2)

	reg.Close()

	if refs := duckdb.DumpCachedLeaks()["rw:"+path]; refs != 0 {
		t.Fatalf("handle metadata non libéré après reg.Close(): refCount=%d (attendu 0)", refs)
	}
}
