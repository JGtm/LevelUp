// Package duckdb_test — privacy_state_repo_test.go : tests PrivacyStateRepo.
//
// Sprint 55 E5 — Utilise un fichier DuckDB temporaire (même pattern que metadata_repo_season_test.go).
// Lancer avec : go test ./internal/platform/duckdb/ -v (sans tag intégration)
package duckdb_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
)

// newPrivacyTestDB crée un fichier DuckDB temporaire avec la table player_privacy_state.
// Retourne le chemin du fichier pour permettre de rouvrir les connexions.
func newPrivacyTestDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "stats.duckdb")

	rw, err := duckdb.OpenReadWrite(dbPath)
	if err != nil {
		t.Fatalf("newPrivacyTestDB OpenReadWrite: %v", err)
	}
	defer rw.Close()

	_, err = rw.Exec(context.Background(), `
		CREATE TABLE player_privacy_state (
			xuid        VARCHAR PRIMARY KEY,
			is_private  BOOLEAN NOT NULL DEFAULT FALSE,
			observed_at TIMESTAMPTZ NOT NULL,
			source      VARCHAR NOT NULL DEFAULT 'api'
		)`)
	if err != nil {
		t.Fatalf("newPrivacyTestDB CREATE TABLE: %v", err)
	}
	return dbPath
}

// openPlayerDB ouvre une connexion read-write sur le fichier et construit un PlayerDB.
// La connexion est fermée automatiquement en fin de test.
func openPlayerDB(t *testing.T, dbPath string) *duckdb.PlayerDB {
	t.Helper()
	db, err := duckdb.OpenReadWrite(dbPath)
	if err != nil {
		t.Fatalf("openPlayerDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return &duckdb.PlayerDB{
		Player:   db,
		XUID:     "test-xuid-privacy",
		Gamertag: "PrivacyTestPlayer",
	}
}

// TestPrivacyStateRepo_Upsert_Then_Load vérifie l'écriture puis la lecture.
// UpsertPrivacyState ouvre sa propre connexion rw (ferme la précédente pour éviter le conflit).
func TestPrivacyStateRepo_Upsert_Then_Load(t *testing.T) {
	dbPath := newPrivacyTestDB(t)
	ctx := context.Background()

	state := domain.PlayerPrivacyState{
		XUID:       "test-xuid-privacy",
		IsPrivate:  true,
		ObservedAt: time.Now().UTC().Truncate(time.Second),
		Source:     "api",
	}

	// Upsert avec connexion fermée avant pour éviter conflit rw+rw
	{
		pdb := openPlayerDB(t, dbPath)
		repo := duckdb.NewPrivacyStateRepo(pdb)
		pdb.Player.Close()
		if err := repo.UpsertPrivacyState(ctx, state); err != nil {
			t.Fatalf("UpsertPrivacyState: %v", err)
		}
	}

	// Load avec une nouvelle connexion (après fermeture de la rw du Upsert)
	pdb := openPlayerDB(t, dbPath)
	repo := duckdb.NewPrivacyStateRepo(pdb)

	loaded, err := repo.LoadPrivacyState(ctx, "test-xuid-privacy")
	if err != nil {
		t.Fatalf("LoadPrivacyState: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadPrivacyState: got nil, want state")
	}
	if loaded.XUID != state.XUID {
		t.Errorf("xuid = %q, want %q", loaded.XUID, state.XUID)
	}
	if loaded.IsPrivate != state.IsPrivate {
		t.Errorf("is_private = %v, want %v", loaded.IsPrivate, state.IsPrivate)
	}
	if loaded.Source != state.Source {
		t.Errorf("source = %q, want %q", loaded.Source, state.Source)
	}
}

// TestPrivacyStateRepo_Load_NotFound vérifie que nil est retourné si absent.
func TestPrivacyStateRepo_Load_NotFound(t *testing.T) {
	dbPath := newPrivacyTestDB(t)
	pdb := openPlayerDB(t, dbPath)
	repo := duckdb.NewPrivacyStateRepo(pdb)

	loaded, err := repo.LoadPrivacyState(context.Background(), "unknown-xuid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded != nil {
		t.Errorf("expected nil for unknown xuid, got %+v", loaded)
	}
}

// TestPrivacyStateRepo_Upsert_Idempotent vérifie que l'upsert remplace l'existant.
func TestPrivacyStateRepo_Upsert_Idempotent(t *testing.T) {
	dbPath := newPrivacyTestDB(t)
	ctx := context.Background()

	first := domain.PlayerPrivacyState{
		XUID:       "test-xuid-privacy",
		IsPrivate:  false,
		ObservedAt: time.Now().UTC().Add(-time.Hour),
		Source:     "api",
	}
	second := domain.PlayerPrivacyState{
		XUID:       "test-xuid-privacy",
		IsPrivate:  true,
		ObservedAt: time.Now().UTC(),
		Source:     "sync",
	}

	// Premier upsert
	{
		pdb := openPlayerDB(t, dbPath)
		repo := duckdb.NewPrivacyStateRepo(pdb)
		pdb.Player.Close()
		if err := repo.UpsertPrivacyState(ctx, first); err != nil {
			t.Fatalf("first upsert: %v", err)
		}
	}

	// Deuxième upsert
	{
		pdb := openPlayerDB(t, dbPath)
		repo := duckdb.NewPrivacyStateRepo(pdb)
		pdb.Player.Close()
		if err := repo.UpsertPrivacyState(ctx, second); err != nil {
			t.Fatalf("second upsert: %v", err)
		}
	}

	// Load — doit refléter le deuxième état
	pdb := openPlayerDB(t, dbPath)
	repo := duckdb.NewPrivacyStateRepo(pdb)
	loaded, err := repo.LoadPrivacyState(ctx, "test-xuid-privacy")
	if err != nil {
		t.Fatalf("LoadPrivacyState: %v", err)
	}
	if loaded == nil {
		t.Fatal("got nil after upsert")
	}
	if !loaded.IsPrivate {
		t.Error("expected is_private=true after second upsert")
	}
	if loaded.Source != "sync" {
		t.Errorf("source = %q, want sync", loaded.Source)
	}
}
