// Package pool — scan_snapshot_test.go : snapshot mémoire des sources de scan.
package pool

import "testing"

func TestScanSnapshot_RecordAndLookup(t *testing.T) {
	// Jamais scanné → absent.
	if _, ok := LastScanSource("halo_infinite", "SnapshotGT-jamais-scanne"); ok {
		t.Error("gamertag jamais scanné : attendu absent")
	}

	recordScanSource("halo_infinite", "SnapshotGT", "watcher_oauth")
	s, ok := LastScanSource("halo_infinite", "SnapshotGT")
	if !ok {
		t.Fatal("source enregistrée introuvable")
	}
	if s.Source != "watcher_oauth" {
		t.Errorf("Source = %q, want watcher_oauth", s.Source)
	}
	if s.At.IsZero() {
		t.Error("At doit être horodaté")
	}

	// Isolation par titre : pas de fuite cross-titre.
	if _, ok := LastScanSource("halo_5", "SnapshotGT"); ok {
		t.Error("la clé doit être isolée par titre")
	}

	// Un nouveau scan écrase la valeur.
	recordScanSource("halo_infinite", "SnapshotGT", "duckdb_oauth+env_oauth")
	s2, _ := LastScanSource("halo_infinite", "SnapshotGT")
	if s2.Source != "duckdb_oauth+env_oauth" {
		t.Errorf("Source après re-scan = %q, want duckdb_oauth+env_oauth", s2.Source)
	}
}
