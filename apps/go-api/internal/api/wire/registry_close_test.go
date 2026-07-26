package wire

import (
	"context"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/ops"
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

// TestServiceRegistry_Close_ReleasesMonitoringStore vérifie que le handle
// monitoring.duckdb ouvert au boot (ops.NewMonitoringStore → OpenReadWrite) est
// bien fermé par Close(). Non-régression de l'anomalie D (prod 2026-07-26) : le
// champ monitoringStore était ignoré par Close(), et duckdb.CloseAll() ne
// parcourt que le pool joueur — refCount=1 résiduel à CHAQUE arrêt
// (WARN shutdown_db_leak) + WAL non checkpointé.
func TestServiceRegistry_Close_ReleasesMonitoringStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "monitoring.duckdb")

	store, err := ops.NewMonitoringStore(context.Background(), path)
	if err != nil {
		t.Fatalf("NewMonitoringStore: %v", err)
	}
	if refs := duckdb.DumpCachedLeaks()["rw:"+path]; refs != 1 {
		t.Fatalf("précondition: refCount attendu 1 après ouverture, got %d", refs)
	}

	reg := &ServiceRegistry{}
	reg.WithMonitoringStore(store)

	reg.Close()

	if refs := duckdb.DumpCachedLeaks()["rw:"+path]; refs != 0 {
		t.Fatalf("store monitoring non libéré après reg.Close(): refCount=%d (attendu 0)", refs)
	}
	// Idempotence : un 2e Close ne doit ni paniquer ni re-fermer.
	reg.Close()
}
