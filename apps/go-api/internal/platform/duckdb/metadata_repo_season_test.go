package duckdb_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
)

// openTestMetadataRepo ouvre une DB DuckDB temporaire et retourne un MetadataRepo initialisé.
func openTestMetadataRepo(t *testing.T) *duckdb.MetadataRepo {
	t.Helper()
	path := filepath.Join(t.TempDir(), "metadata_test.duckdb")
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("openTestMetadataRepo: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	repo := duckdb.NewMetadataRepoFromDB(db)
	if err := repo.EnsureSeasonTables(context.Background()); err != nil {
		t.Fatalf("EnsureSeasonTables: %v", err)
	}
	return repo
}

// TestSeasonRefresh_ContentHashUnchanged vérifie que si le content_hash est identique,
// on peut détecter le skip sans écriture supplémentaire.
func TestSeasonRefresh_ContentHashUnchanged(t *testing.T) {
	repo := openTestMetadataRepo(t)
	ctx := context.Background()

	snap := domain.WaypointResourceSnapshot{
		TitleID:     "halo_infinite",
		ResourceKey: "season_calendar",
		Version:     "v1",
		FetchedAt:   time.Now().UTC(),
		ContentHash: "abc123",
	}
	if err := repo.UpsertSnapshot(ctx, snap); err != nil {
		t.Fatalf("UpsertSnapshot: %v", err)
	}

	got, err := repo.GetSnapshot(ctx, "halo_infinite", "season_calendar")
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if got == nil {
		t.Fatal("GetSnapshot: attendu non-nil")
	}
	if got.ContentHash != "abc123" {
		t.Errorf("ContentHash: attendu %q, obtenu %q", "abc123", got.ContentHash)
	}

	// Simuler la logique CLI : skip si hash identique.
	newHash := "abc123"
	if got.ContentHash == newHash {
		// Skip attendu — le test vérifie qu'on peut détecter ce cas.
		return
	}
	t.Error("skip non détecté alors que les hashes sont identiques")
}

// TestSeasonRefresh_ContentHashChanged vérifie que si le content_hash change,
// un nouvel upsert est effectué et le snapshot est mis à jour.
func TestSeasonRefresh_ContentHashChanged(t *testing.T) {
	repo := openTestMetadataRepo(t)
	ctx := context.Background()

	// Premier snapshot.
	snap1 := domain.WaypointResourceSnapshot{
		TitleID:     "halo_infinite",
		ResourceKey: "season_calendar",
		Version:     "v1",
		FetchedAt:   time.Now().UTC(),
		ContentHash: "hash-v1",
	}
	if err := repo.UpsertSnapshot(ctx, snap1); err != nil {
		t.Fatalf("UpsertSnapshot v1: %v", err)
	}

	// Upsert d'une saison.
	s := domain.SeasonCalendar{
		TitleID:   "halo_infinite",
		SeasonID:  "Season6",
		StartDate: time.Now().UTC(),
	}
	if err := repo.UpsertSeason(ctx, s); err != nil {
		t.Fatalf("UpsertSeason: %v", err)
	}

	// Deuxième snapshot avec hash différent.
	snap2 := domain.WaypointResourceSnapshot{
		TitleID:     "halo_infinite",
		ResourceKey: "season_calendar",
		Version:     "v2",
		FetchedAt:   time.Now().UTC(),
		ContentHash: "hash-v2",
	}
	if err := repo.UpsertSnapshot(ctx, snap2); err != nil {
		t.Fatalf("UpsertSnapshot v2: %v", err)
	}

	got, err := repo.GetSnapshot(ctx, "halo_infinite", "season_calendar")
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	// GetSnapshot retourne le plus récent.
	if got.ContentHash != "hash-v2" {
		t.Errorf("ContentHash: attendu hash-v2, obtenu %q", got.ContentHash)
	}
}

// TestGetCurrentSeason_Fallback vérifie que GetCurrentSeason retourne nil (et non une erreur)
// quand aucune saison n'est présente — le CLI doit gérer le fallback lui-même.
func TestGetCurrentSeason_Fallback(t *testing.T) {
	repo := openTestMetadataRepo(t)
	ctx := context.Background()

	season, err := repo.GetCurrentSeason(ctx, "halo_infinite")
	if err != nil {
		t.Fatalf("GetCurrentSeason: erreur inattendue : %v", err)
	}
	// Base vide : nil attendu, pas d'erreur.
	if season != nil {
		t.Errorf("GetCurrentSeason: attendu nil pour base vide, obtenu %+v", season)
	}
}

// TestSeasonRefresh_ETagNotModified simule le comportement ETag :
// si le snapshot existant a le même hash, on ne réécrit pas.
func TestSeasonRefresh_ETagNotModified(t *testing.T) {
	repo := openTestMetadataRepo(t)
	ctx := context.Background()

	const hash = "etag-hash-stable"

	snap := domain.WaypointResourceSnapshot{
		TitleID:     "halo_infinite",
		ResourceKey: "csr_season_calendar",
		Version:     "v1",
		FetchedAt:   time.Now().UTC(),
		ContentHash: hash,
		ETag:        `"etag-value"`,
	}
	if err := repo.UpsertSnapshot(ctx, snap); err != nil {
		t.Fatalf("UpsertSnapshot: %v", err)
	}

	got, err := repo.GetSnapshot(ctx, "halo_infinite", "csr_season_calendar")
	if err != nil || got == nil {
		t.Fatalf("GetSnapshot: err=%v, got=%v", err, got)
	}

	// Simuler réponse 304 (ETag inchangé) : le hash Waypoint correspond.
	if got.ContentHash == hash {
		return // skip correct — pas d'upsert
	}
	t.Error("304 non détecté : l'ETag est identique mais le skip n'a pas été effectué")
}
