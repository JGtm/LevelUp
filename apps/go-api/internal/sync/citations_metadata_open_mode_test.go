package sync

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	duckdbpkg "levelup/go-api/internal/platform/duckdb"
)

// Les trois passes de citations ouvraient metadata.duckdb par OpenReadOnly (`ro:`). Le
// serveur tient ce fichier en `rw:` depuis le boot, et l orchestrateur de backfill de l admin
// les appelle dans ce process : « Can't open a connection to same database file with a
// different configuration » — le backfill citations lance depuis l admin echouait. Ce test
// tient le handle `rw:` comme le serveur et exige que RunCitationPostComputeChecks passe
// l ouverture de metadata (l erreur, si elle revient, porte « open metadata »). Il rougit si
// OpenReadOnly revient. Les deux autres passes partagent le meme motif ; le ratchet
// archlint no_metadata_readonly_in_sync_test.go interdit le retour du litteral.
func TestRunCitationPostComputeChecks_SousHandleRW_PasseLOuvertureMetadata(t *testing.T) {
	dir := t.TempDir()
	metaPath := filepath.Join(dir, "metadata.duckdb")
	held, err := duckdbpkg.OpenReadWriteShared(metaPath)
	if err != nil {
		t.Fatalf("ouverture rw metadata : %v", err)
	}
	defer held.Close()
	if _, err := held.SQLDb().ExecContext(context.Background(), `
		CREATE TABLE citation_mappings (
			citation_name_norm VARCHAR, citation_name_display VARCHAR, mapping_type VARCHAR,
			medal_id BIGINT, medal_ids VARCHAR, stat_name VARCHAR, award_name VARCHAR,
			custom_function VARCHAR, composite_children VARCHAR, tier_targets VARCHAR,
			enabled BOOLEAN
		)`); err != nil {
		t.Fatalf("seed citation_mappings : %v", err)
	}

	e := &SyncEngine{
		gamertag:       "TestPlayer",
		xuid:           "2533274800000000",
		playerDBPath:   filepath.Join(dir, "player", "stats.duckdb"),
		metadataDBPath: metaPath,
	}
	_, err = e.RunCitationPostComputeChecks(context.Background())
	if err != nil && strings.Contains(err.Error(), "open metadata") {
		t.Fatalf("l ouverture de metadata a echoue alors qu un handle rw est tenu par le process : %v", err)
	}
}
