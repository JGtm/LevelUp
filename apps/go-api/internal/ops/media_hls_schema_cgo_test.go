package ops

import (
	"context"
	"testing"
)

// TestEnsureMediaTables_HLSColumns vérifie que la migration idempotente ajoute
// les colonnes HLS (hls_path, transcode_status, transcode_started_at) requises
// par le worker de transcoding et le balayage. Garde-rail : ces colonnes sont
// dédiées — `status` (sémantique 'active' du rail home) n'est pas réutilisé.
func TestEnsureMediaTables_HLSColumns(t *testing.T) {
	_, db := openDiagDB(t)
	if err := ensureMediaTables(context.Background(), db); err != nil {
		t.Fatalf("ensureMediaTables: %v", err)
	}
	for _, col := range []string{"hls_path", "transcode_status", "transcode_started_at"} {
		var cnt int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='main' AND table_name='media_files' AND column_name=?",
			col,
		).Scan(&cnt)
		if err != nil {
			t.Fatalf("information_schema columns %q: %v", col, err)
		}
		if cnt == 0 {
			t.Errorf("colonne %q absente après ensureMediaTables", col)
		}
	}
}
