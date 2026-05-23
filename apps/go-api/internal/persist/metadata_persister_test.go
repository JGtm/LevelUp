//go:build integration

// Package persist — metadata_persister_test.go : tests TDD pour MetadataPersister.

package persist

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func openMetadataTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migration.RunForDB(db, migration.TargetMetadata); err != nil {
		t.Fatalf("migrate metadata: %v", err)
	}
	return db
}

// ─── Test 1 : Persist N translations ──────────────────────────────────────

func TestMetadataPersister_Persist_InsertsTranslations(t *testing.T) {
	db := openMetadataTestDB(t)
	p := NewMetadataPersister(db)

	b := NewBatchBuilder("halo_infinite", "Alice", "1111", "test")
	b.AddModeNameTranslations([]ModeNameTranslationInsert{
		{ModeEN: "NewSlayer3000", Lang: "fr", Name: "Carnage 3000"},
		{ModeEN: "NewSlayer3000", Lang: "en", Name: "Slayer 3000"},
		{ModeEN: "NewCTF", Lang: "fr", Name: "Capture du drapeau"},
	})

	if err := p.Persist(context.Background(), b.Build()); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM mode_name_tr WHERE mode_en LIKE 'New%'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("mode_name_tr : %d rows insérées, want 3", n)
	}

	var got string
	_ = db.QueryRow(`SELECT name FROM mode_name_tr WHERE mode_en = ? AND lang = ?`,
		"NewSlayer3000", "fr").Scan(&got)
	if got != "Carnage 3000" {
		t.Errorf("name fr = %q, want 'Carnage 3000'", got)
	}
}

// ─── Test 2 : INSERT OR IGNORE — préserve l'existant ──────────────────────

func TestMetadataPersister_Persist_PreservesExistingTranslations(t *testing.T) {
	db := openMetadataTestDB(t)

	// La migration pré-seed les modes Halo classiques (Slayer, CTF, etc.).
	// Lire la traduction FR actuelle de "Slayer" comme valeur de référence.
	var existing string
	if err := db.QueryRow(`SELECT name FROM mode_name_tr WHERE mode_en = ? AND lang = ?`,
		"Slayer", "fr").Scan(&existing); err != nil {
		t.Skipf("Slayer/fr non pré-seedé, skip : %v", err)
	}

	p := NewMetadataPersister(db)
	b := NewBatchBuilder("halo_infinite", "Alice", "1111", "test")
	b.AddModeNameTranslations([]ModeNameTranslationInsert{
		// Le sync essaie d'écraser avec une traduction auto-générée.
		{ModeEN: "Slayer", Lang: "fr", Name: "AUTO_OVERRIDE_NEVER"},
	})

	if err := p.Persist(context.Background(), b.Build()); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	// La traduction existante doit avoir été préservée (INSERT OR IGNORE).
	var got string
	if err := db.QueryRow(`SELECT name FROM mode_name_tr WHERE mode_en = ? AND lang = ?`,
		"Slayer", "fr").Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != existing {
		t.Errorf("name = %q, want %q (INSERT OR IGNORE préserve l'existant)", got, existing)
	}
	if got == "AUTO_OVERRIDE_NEVER" {
		t.Errorf("INSERT OR IGNORE a échoué — la traduction a été écrasée")
	}
}

// ─── Test 3 : No metadata in batch → no-op ─────────────────────────────────

func TestMetadataPersister_Persist_NoMetadata_NoOp(t *testing.T) {
	db := openMetadataTestDB(t)
	p := NewMetadataPersister(db)

	builder := NewBatchBuilder("halo_infinite", "Alice", "1111", "test")
	if err := p.Persist(context.Background(), builder.Build()); err != nil {
		t.Fatalf("Persist no-op: %v", err)
	}
	// Pas d'assertion : on vérifie juste que ça ne panic pas.
}

// ─── Test 4 : NilBatch → erreur ────────────────────────────────────────────

func TestMetadataPersister_Persist_NilBatch_ReturnsError(t *testing.T) {
	db := openMetadataTestDB(t)
	p := NewMetadataPersister(db)
	if err := p.Persist(context.Background(), nil); err == nil {
		t.Error("Persist(nil) devrait retourner une erreur")
	}
}
