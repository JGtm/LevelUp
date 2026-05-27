//go:build integration

// Package sync — refresh_shared_views_boot_test.go : anti-régression pour
// le fix bug 2026-05-27 — création des VIEWs shared au boot via
// EnsureSharedSchema (au lieu du post-sync runtime).
//
// CONTEXTE :
//   - AVANT : refreshSharedViews tournait après chaque sync, recréait
//     v_gamertag_lookup et v_match_full sur le sharedDB writer.
//   - BUG : en runtime, le writer pouvait avoir un ATTACH RO implicite
//     (auto-attach DuckDB depuis l'instance RO précédente du Provider). Le
//     CREATE OR REPLACE VIEW tombait sur la DB attached RO → "Cannot
//     execute statement of type CREATE on database shared_matches_v2 which
//     is attached in read-only mode!" (logs/sync.log 23:57:51 et+).
//   - FIX : déplacer les VIEW dans EnsureSharedSchema (boot one-time, conn
//     RW pure sans auto-attach concurrent) + retirer le call runtime.
//
// Tests :
//  1. EnsureSharedSchema crée les VIEWs v_gamertag_lookup et v_match_full
//  2. Les VIEWs sont fonctionnelles (SELECT renvoie le bon shape)
//  3. EnsureSharedSchema est idempotent (recreate sans erreur)
//  4. Le post-sync NE doit PAS appeler refreshSharedViews (assertion code)
package sync

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestEnsureSharedSchema_CreatesVGamertagLookup vérifie qu'après
// EnsureSharedSchema, la vue v_gamertag_lookup existe et est queryable.
func TestEnsureSharedSchema_CreatesVGamertagLookup(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := EnsureSharedSchema(context.Background(), db); err != nil {
		t.Fatalf("EnsureSharedSchema: %v", err)
	}

	// La VIEW doit exister.
	var viewName string
	err = db.QueryRow(`
		SELECT view_name FROM duckdb_views()
		WHERE view_name = 'v_gamertag_lookup'
	`).Scan(&viewName)
	if err != nil {
		t.Fatalf("v_gamertag_lookup absente après EnsureSharedSchema: %v", err)
	}

	// Insert minimal puis SELECT depuis la VIEW.
	if _, err := db.Exec(`INSERT INTO xuid_aliases (xuid, gamertag) VALUES (?, ?)`,
		"xuid_test_001", "TestGT"); err != nil {
		t.Fatalf("insert xuid_aliases: %v", err)
	}
	var gotXUID, gotGamertag string
	err = db.QueryRow(`SELECT xuid, gamertag FROM v_gamertag_lookup WHERE xuid = ?`,
		"xuid_test_001").Scan(&gotXUID, &gotGamertag)
	if err != nil {
		t.Fatalf("SELECT v_gamertag_lookup: %v", err)
	}
	if gotGamertag != "TestGT" {
		t.Errorf("gamertag = %q, want TestGT", gotGamertag)
	}
}

// TestEnsureSharedSchema_CreatesVMatchFull vérifie v_match_full.
func TestEnsureSharedSchema_CreatesVMatchFull(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := EnsureSharedSchema(context.Background(), db); err != nil {
		t.Fatalf("EnsureSharedSchema: %v", err)
	}

	var viewName string
	err = db.QueryRow(`
		SELECT view_name FROM duckdb_views()
		WHERE view_name = 'v_match_full'
	`).Scan(&viewName)
	if err != nil {
		t.Fatalf("v_match_full absente après EnsureSharedSchema: %v", err)
	}
}

// TestEnsureSharedSchema_ViewsIdempotent : recreate views sans erreur
// (CREATE OR REPLACE VIEW) à travers plusieurs passes EnsureSharedSchema.
func TestEnsureSharedSchema_ViewsIdempotent(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for i := 0; i < 3; i++ {
		if err := EnsureSharedSchema(context.Background(), db); err != nil {
			t.Fatalf("EnsureSharedSchema passe %d: %v", i+1, err)
		}
	}

	// Les VIEWs doivent toujours être uniques (pas de duplicate).
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM duckdb_views()
		WHERE view_name IN ('v_gamertag_lookup', 'v_match_full')
	`).Scan(&count)
	if err != nil {
		t.Fatalf("count views: %v", err)
	}
	if count != 2 {
		t.Errorf("count views = %d, want 2 (idempotent)", count)
	}
}

// TestPostSyncDoesNotCallRefreshSharedViews : garde-rail anti-régression
// (assertion code, pas runtime). Vérifie que engine_postsync.go ne
// référence plus `refreshSharedViews(`. Si quelqu'un ré-introduit ce call
// runtime, le bug "attached in read-only mode" reviendra.
func TestPostSyncDoesNotCallRefreshSharedViews(t *testing.T) {
	content, err := os.ReadFile("engine_postsync.go")
	if err != nil {
		t.Fatalf("read engine_postsync.go: %v", err)
	}
	text := string(content)

	// On cherche uniquement les CALLs (parenthèse ouvrante après le nom).
	// Les commentaires qui mentionnent "refreshSharedViews" sans parenthèse
	// sont OK (documentation du fix).
	if strings.Contains(text, "refreshSharedViews(") {
		t.Error("REGRESSION : engine_postsync.go référence refreshSharedViews(...) — le call runtime est ré-introduit, le bug auto-attach RO reviendra (cf. logs 2026-05-27 23:57:51)")
	}
}
