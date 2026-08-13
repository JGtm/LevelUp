package scheduler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/platform/duckdb"
)

// prepareReplayPurgeFixture construit une shared DB minimale (match_registry avec les
// deux colonnes de temps du fragment canonique) et un dossier d'artefacts.
func prepareReplayPurgeFixture(t *testing.T) (sharedPath, artifactsDir string) {
	t.Helper()
	root := t.TempDir()
	sharedPath = filepath.Join(root, "shared_matches_v2.duckdb")
	db, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("open shared RW: %v", err)
	}
	defer func() { _ = db.Close() }()
	sqlDB := db.SQLDb()
	if _, err := sqlDB.Exec(`CREATE TABLE match_registry (
		match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP, start_time_utc TIMESTAMPTZ)`); err != nil {
		t.Fatalf("create match_registry: %v", err)
	}
	vieux := time.Now().UTC().AddDate(0, -8, 0).Format("2006-01-02 15:04:05")
	recent := time.Now().UTC().AddDate(0, -1, 0).Format("2006-01-02 15:04:05")
	for _, row := range [][2]string{
		{"aaaa0001-1111-4abc-9def-000000000001", vieux},
		{"bbbb0002-1111-4abc-9def-000000000002", recent},
	} {
		if _, err := sqlDB.Exec(
			`INSERT INTO match_registry (match_id, start_time_utc) VALUES (?, ?::TIMESTAMPTZ)`,
			row[0], row[1]); err != nil {
			t.Fatalf("insert %s: %v", row[0], err)
		}
	}

	artifactsDir = filepath.Join(root, "replays", "halo_infinite")
	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"aaaa0001.json", "bbbb0002.json", "cccc0003.json"} {
		if err := os.WriteFile(filepath.Join(artifactsDir, name), []byte(`{"schemaVersion":3}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return sharedPath, artifactsDir
}

// TestPurgeReplayArtifacts — le contrat de la purge : SEUL l'artefact d'un match plus
// vieux que la fenêtre part ; le récent reste ; l'indatable (sans ligne de registre)
// n'est JAMAIS détruit.
func TestPurgeReplayArtifacts(t *testing.T) {
	sharedPath, artifactsDir := prepareReplayPurgeFixture(t)
	cutoff := time.Now().UTC().AddDate(0, -6, 0)

	purged, kept, unknown, err := purgeReplayArtifactsForTitle(
		context.Background(), sharedPath, artifactsDir, cutoff)
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if purged != 1 || kept != 1 || unknown != 1 {
		t.Errorf("purge = (purged %d, kept %d, unknown %d), attendu (1, 1, 1)", purged, kept, unknown)
	}
	if _, err := os.Stat(filepath.Join(artifactsDir, "aaaa0001.json")); !os.IsNotExist(err) {
		t.Error("l'artefact du match de 8 mois devait être purgé")
	}
	for _, garde := range []string{"bbbb0002.json", "cccc0003.json"} {
		if _, err := os.Stat(filepath.Join(artifactsDir, garde)); err != nil {
			t.Errorf("%s ne devait PAS être purgé : %v", garde, err)
		}
	}
}

// TestPurgeReplayArtifacts_DossierAbsent — aucun artefact construit = cycle nominal.
func TestPurgeReplayArtifacts_DossierAbsent(t *testing.T) {
	sharedPath, _ := prepareReplayPurgeFixture(t)
	purged, kept, unknown, err := purgeReplayArtifactsForTitle(
		context.Background(), sharedPath, filepath.Join(t.TempDir(), "inexistant"), time.Now())
	if err != nil || purged != 0 || kept != 0 || unknown != 0 {
		t.Errorf("dossier absent = (%d, %d, %d, %v), attendu (0, 0, 0, nil)", purged, kept, unknown, err)
	}
}
