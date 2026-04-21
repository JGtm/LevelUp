package assets

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// TestDuckDBIndexStore_RoundTrip valide le cycle complet EnsureTable + PersistIndex + LookupIndex.
func TestDuckDBIndexStore_RoundTrip(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_assets.duckdb")
	store := NewDuckDBIndexStore(dbPath)
	t.Cleanup(func() { store.Close() })

	ctx := context.Background()

	if !store.Available(ctx) {
		t.Skip("DuckDB non disponible dans cet environnement de test")
	}

	if err := store.EnsureTable(ctx); err != nil {
		t.Fatalf("EnsureTable: %v", err)
	}

	ref := Ref{Kind: KindMedalImage, TitleID: "halo_infinite", ID: "42"}
	entry := IndexEntry{
		Ref:         ref,
		URL:         "https://cdn.example.com/medal.png",
		ContentHash: "abc123",
		FetchedAt:   time.Now().UTC().Truncate(time.Second),
	}

	if err := store.PersistIndex(ctx, ref, entry); err != nil {
		t.Fatalf("PersistIndex: %v", err)
	}

	got, err := store.LookupIndex(ctx, ref)
	if err != nil {
		t.Fatalf("LookupIndex: %v", err)
	}
	if got == nil {
		t.Fatal("attendu un résultat non-nil")
	}
	if got.URL != entry.URL {
		t.Errorf("URL: got %q, want %q", got.URL, entry.URL)
	}
	if got.ContentHash != entry.ContentHash {
		t.Errorf("ContentHash: got %q, want %q", got.ContentHash, entry.ContentHash)
	}
}

func TestDuckDBIndexStore_LookupIndex_Miss(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_assets_miss.duckdb")
	store := NewDuckDBIndexStore(dbPath)
	t.Cleanup(func() { store.Close() })

	ctx := context.Background()

	if !store.Available(ctx) {
		t.Skip("DuckDB non disponible dans cet environnement de test")
	}

	if err := store.EnsureTable(ctx); err != nil {
		t.Fatalf("EnsureTable: %v", err)
	}

	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "nonexistent"}
	got, err := store.LookupIndex(ctx, ref)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if got != nil {
		t.Error("attendu nil pour ref absente")
	}
}

func TestDuckDBIndexStore_PersistIndex_Upsert(t *testing.T) {
	// Upsert : une seconde écriture avec le même ref doit mettre à jour l'URL.
	dbPath := filepath.Join(t.TempDir(), "test_assets_upsert.duckdb")
	store := NewDuckDBIndexStore(dbPath)
	t.Cleanup(func() { store.Close() })

	ctx := context.Background()
	if !store.Available(ctx) {
		t.Skip("DuckDB non disponible")
	}
	if err := store.EnsureTable(ctx); err != nil {
		t.Fatalf("EnsureTable: %v", err)
	}

	ref := Ref{Kind: KindChallengeBadge, TitleID: "hi", ID: "weekly"}
	e1 := IndexEntry{Ref: ref, URL: "https://cdn.example.com/v1.png", ContentHash: "hash1", FetchedAt: time.Now()}
	e2 := IndexEntry{Ref: ref, URL: "https://cdn.example.com/v2.png", ContentHash: "hash2", FetchedAt: time.Now()}

	if err := store.PersistIndex(ctx, ref, e1); err != nil {
		t.Fatalf("PersistIndex e1: %v", err)
	}
	if err := store.PersistIndex(ctx, ref, e2); err != nil {
		t.Fatalf("PersistIndex e2: %v", err)
	}

	got, err := store.LookupIndex(ctx, ref)
	if err != nil {
		t.Fatalf("LookupIndex: %v", err)
	}
	if got == nil {
		t.Fatal("attendu non-nil")
	}
	if got.URL != e2.URL {
		t.Errorf("URL après upsert: got %q, want %q", got.URL, e2.URL)
	}
}

func TestDuckDBIndexStore_Available_InvalidPath(t *testing.T) {
	// Un chemin invalide doit retourner Available=false.
	store := NewDuckDBIndexStore("/invalid/path/nonexistent.duckdb")
	ctx := context.Background()
	if store.Available(ctx) {
		t.Error("Available devrait être false pour un chemin invalide")
	}
}

func TestDuckDBIndexStore_PersistIndex_WithLocalPath(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_assets_lp.duckdb")
	store := NewDuckDBIndexStore(dbPath)
	t.Cleanup(func() { store.Close() })

	ctx := context.Background()
	if !store.Available(ctx) {
		t.Skip("DuckDB non disponible")
	}
	if err := store.EnsureTable(ctx); err != nil {
		t.Fatalf("EnsureTable: %v", err)
	}

	ref := Ref{Kind: KindMapImage, TitleID: "hi", ID: "LiveFire"}
	entry := IndexEntry{
		Ref:       ref,
		LocalPath: "/static/maps/livefire.png",
		FetchedAt: time.Now(),
	}

	if err := store.PersistIndex(ctx, ref, entry); err != nil {
		t.Fatalf("PersistIndex: %v", err)
	}

	got, err := store.LookupIndex(ctx, ref)
	if err != nil {
		t.Fatalf("LookupIndex: %v", err)
	}
	if got == nil {
		t.Fatal("attendu non-nil")
	}
	if got.LocalPath != entry.LocalPath {
		t.Errorf("LocalPath: got %q, want %q", got.LocalPath, entry.LocalPath)
	}
}
