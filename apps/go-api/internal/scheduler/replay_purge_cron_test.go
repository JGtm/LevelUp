package scheduler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
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

// TestPurge_ArtefactPurgeEmporteSonSidecar — DECISION PRODUIT DU 2026-09-06 (constat C12).
//
// Le raster tactique est un DERIVE de l'artefact : sans lui il ne peut plus etre ni relu
// utilement (le match sort de la fenetre de retention, donc du perimetre) ni refait. Le
// laisser en place accumulerait des orphelins que rien ne collecterait jamais — la purge
// saute les repertoires, et `rasters/` en est un.
//
// Le sidecar d'un artefact GARDE, lui, reste : c'est la moitie qui prouve que la
// suppression suit la purge et n'est pas un balayage.
func TestPurge_ArtefactPurgeEmporteSonSidecar(t *testing.T) {
	sharedPath, artifactsDir := prepareReplayPurgeFixture(t)
	rasters := filepath.Join(artifactsDir, titlePkg.SousDossierRasters)
	if err := os.MkdirAll(rasters, 0o755); err != nil {
		t.Fatalf("mkdir rasters: %v", err)
	}
	purgeable := filepath.Join(rasters, "aaaa0001.json") // artefact date AVANT le seuil
	garde := filepath.Join(rasters, "bbbb0002.json")     // artefact date APRES le seuil
	for _, p := range []string{purgeable, garde} {
		if err := os.WriteFile(p, []byte(`{"schema_version":2}`), 0o644); err != nil {
			t.Fatalf("write sidecar: %v", err)
		}
	}

	purged, kept, _, err := purgeReplayArtifactsForTitle(
		context.Background(), sharedPath, artifactsDir, time.Now().UTC().AddDate(0, -6, 0))
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if purged != 1 || kept != 1 {
		t.Fatalf("purged=%d kept=%d, attendu 1 et 1", purged, kept)
	}
	if _, err := os.Stat(purgeable); !os.IsNotExist(err) {
		t.Fatalf("le sidecar de l'artefact purge survit (err = %v) : il est orphelin a jamais", err)
	}
	if _, err := os.Stat(garde); err != nil {
		t.Fatalf("le sidecar d'un artefact GARDE a ete supprime : %v", err)
	}
}

// TestPurge_DossierSansArtefactNOuvrePasLaBase — le court-circuit compte les ARTEFACTS,
// pas les entrees.
//
// Depuis que les sidecars vivent dans un sous-dossier, un titre sans le moindre artefact
// rend tout de meme une entree — le repertoire `rasters/`. Le court-circuit d'origine
// (`len(entries) == 0`) ne mordait donc plus, et chaque tick du cron ouvrait la shared
// pour n'y trouver rien a purger. LA PREUVE EST QUE LA BASE N'EXISTE MEME PAS : si elle
// etait ouverte, la passe rendrait une erreur au lieu de (0,0,0,nil).
func TestPurge_DossierSansArtefactNOuvrePasLaBase(t *testing.T) {
	artifactsDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(artifactsDir, titlePkg.SousDossierRasters), 0o755); err != nil {
		t.Fatalf("mkdir rasters: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(artifactsDir, titlePkg.SousDossierRasters, "aaaa0001.json"),
		[]byte(`{}`), 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}
	sharedInexistant := filepath.Join(t.TempDir(), "pas_de_shared.duckdb")

	purged, kept, unknown, err := purgeReplayArtifactsForTitle(
		context.Background(), sharedInexistant, artifactsDir, time.Now().UTC())
	if err != nil {
		t.Fatalf("un dossier sans aucun artefact ne doit rien ouvrir : %v", err)
	}
	if purged != 0 || kept != 0 || unknown != 0 {
		t.Fatalf("(%d, %d, %d), attendu (0, 0, 0)", purged, kept, unknown)
	}
}
