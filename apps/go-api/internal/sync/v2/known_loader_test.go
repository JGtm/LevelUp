// Package v2 — known_loader_test.go : tests KnownLoader V2-native avec
// DuckDB temp réelle (pas de mock — vérifie le SQL exact contre le moteur
// de prod).
//
// Pas de build tag : ces tests tournent par défaut dans `go test ./...`.
// Coût : ~200ms par test (init DuckDB + migrations minimales).
package v2

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/migration"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
)

// setupTestPlayerDB crée une stats.duckdb minimaliste avec
// player_match_enrichment + données.
func setupTestPlayerDB(t *testing.T, matchIDs []string) (*duckdbpkg.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "player.duckdb")
	db, err := duckdbpkg.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite player: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Schéma minimal pour player_match_enrichment : match_id + les colonnes de
	// base référencées par la vue _latest (la migration append-only ajoute les
	// colonnes engagement/psa mais suppose les colonnes de base préexistantes,
	// créées en prod par create_base_player_schema).
	if _, err := db.SQLDb().Exec(`
		CREATE TABLE player_match_enrichment (
			match_id VARCHAR PRIMARY KEY,
			performance_score FLOAT,
			session_id VARCHAR,
			session_label VARCHAR,
			is_with_friends BOOLEAN DEFAULT FALSE,
			teammates_signature VARCHAR
		)
	`); err != nil {
		t.Fatalf("create table player_match_enrichment: %v", err)
	}
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db.SQLDb()); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}
	for _, mID := range matchIDs {
		if _, err := db.SQLDb().Exec("INSERT INTO player_match_enrichment (match_id) VALUES (?)", mID); err != nil {
			t.Fatalf("insert match_id %s: %v", mID, err)
		}
	}
	return db, path
}

// setupTestSharedDB crée une shared_matches_v2.duckdb minimaliste avec
// match_participants + données.
// participantsByXUID : xuid → liste de match_ids.
func setupTestSharedDB(t *testing.T, participantsByXUID map[string][]string) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "shared.duckdb")
	db, err := duckdbpkg.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite shared: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.SQLDb().Exec(`
		CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid VARCHAR
		)
	`); err != nil {
		t.Fatalf("create table match_participants: %v", err)
	}
	for xuid, matchIDs := range participantsByXUID {
		for _, mID := range matchIDs {
			if _, err := db.SQLDb().Exec(
				"INSERT INTO match_participants (match_id, xuid) VALUES (?, ?)", mID, xuid,
			); err != nil {
				t.Fatalf("insert participant: %v", err)
			}
		}
	}
	return db.SQLDb()
}

// ─── Tests ────────────────────────────────────────────────────────────

func TestKnownLoaderV2_PlayerSourceOnly(t *testing.T) {
	// Seulement source 1 (player_match_enrichment), pas de shared.
	playerDB, _ := setupTestPlayerDB(t, []string{"m1", "m2", "m3"})
	opener := func(ctx context.Context, gt string) (*sql.DB, func(), error) {
		return playerDB.SQLDb(), func() {}, nil
	}
	loader := NewKnownLoader(opener, func() *sql.DB { return nil })

	known, err := loader.LoadKnown(context.Background(), PlayerProfile{
		Gamertag: "alice", XUID: "1234567890123456", PlayerSlug: "alice",
	})
	if err != nil {
		t.Fatalf("LoadKnown err = %v", err)
	}
	if len(known) != 3 {
		t.Errorf("known len = %d, want 3", len(known))
	}
	for _, mID := range []string{"m1", "m2", "m3"} {
		if !known[mID] {
			t.Errorf("known[%s] = false, want true", mID)
		}
	}
}

func TestKnownLoaderV2_PlayerAndSharedUnion(t *testing.T) {
	// Source 1 a m1, m2. Source 2 (shared pour alice xuid=999) a m2, m3, m4.
	// Union attendue : {m1, m2, m3, m4}.
	playerDB, _ := setupTestPlayerDB(t, []string{"m1", "m2"})
	sharedDB := setupTestSharedDB(t, map[string][]string{
		"999": {"m2", "m3", "m4"},
		"888": {"m_other"}, // autre joueur, ne doit pas être inclus
	})
	opener := func(ctx context.Context, gt string) (*sql.DB, func(), error) {
		return playerDB.SQLDb(), func() {}, nil
	}
	loader := NewKnownLoader(opener, func() *sql.DB { return sharedDB })

	known, err := loader.LoadKnown(context.Background(), PlayerProfile{
		Gamertag: "alice", XUID: "999",
	})
	if err != nil {
		t.Fatalf("LoadKnown err = %v", err)
	}
	if len(known) != 4 {
		t.Errorf("known len = %d, want 4 (m1,m2,m3,m4)", len(known))
	}
	for _, mID := range []string{"m1", "m2", "m3", "m4"} {
		if !known[mID] {
			t.Errorf("known[%s] = false", mID)
		}
	}
	if known["m_other"] {
		t.Error("known[m_other] = true (cross-xuid leak — bug !)")
	}
}

func TestKnownLoaderV2_EmptyXUIDSkipsSharedSource(t *testing.T) {
	// xuid vide → source 2 désactivée. Devrait juste retourner source 1.
	playerDB, _ := setupTestPlayerDB(t, []string{"m1"})
	sharedDB := setupTestSharedDB(t, map[string][]string{
		"999": {"m_shared"},
	})
	opener := func(ctx context.Context, gt string) (*sql.DB, func(), error) {
		return playerDB.SQLDb(), func() {}, nil
	}
	loader := NewKnownLoader(opener, func() *sql.DB { return sharedDB })

	known, err := loader.LoadKnown(context.Background(), PlayerProfile{
		Gamertag: "alice", XUID: "  ", // espaces uniquement → trim vide
	})
	if err != nil {
		t.Fatalf("LoadKnown err = %v", err)
	}
	if len(known) != 1 {
		t.Errorf("known len = %d, want 1 (source 2 désactivée)", len(known))
	}
	if !known["m1"] {
		t.Error("known[m1] = false")
	}
	if known["m_shared"] {
		t.Error("known[m_shared] = true (source 2 ne devrait pas avoir tourné)")
	}
}

func TestKnownLoaderV2_NilSharedDBSkipsSource2(t *testing.T) {
	// sharedDB nil → source 2 désactivée, pas d'erreur.
	playerDB, _ := setupTestPlayerDB(t, []string{"m1"})
	opener := func(ctx context.Context, gt string) (*sql.DB, func(), error) {
		return playerDB.SQLDb(), func() {}, nil
	}
	loader := NewKnownLoader(opener, func() *sql.DB { return nil })

	known, err := loader.LoadKnown(context.Background(), PlayerProfile{
		Gamertag: "alice", XUID: "999",
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(known) != 1 {
		t.Errorf("known len = %d, want 1", len(known))
	}
}

func TestKnownLoaderV2_OpenPlayerDBFailureIsFatal(t *testing.T) {
	// Si openPlayerDB échoue (cas pathologique, ne devrait pas arriver
	// en prod), on retourne erreur. Discovery capture dans Errors.
	opener := func(ctx context.Context, gt string) (*sql.DB, func(), error) {
		return nil, nil, sql.ErrConnDone
	}
	loader := NewKnownLoader(opener, func() *sql.DB { return nil })
	_, err := loader.LoadKnown(context.Background(), PlayerProfile{Gamertag: "alice"})
	if err == nil {
		t.Fatal("LoadKnown should return err when openPlayerDB fails")
	}
}

func TestKnownLoaderV2_PlayerTableMissingIsTolerated(t *testing.T) {
	// Si player_match_enrichment n'existe pas (DB neuve), on log DEBUG
	// et continue avec source 2. Cas réaliste : 1er sync d'un joueur.
	path := filepath.Join(t.TempDir(), "fresh.duckdb")
	freshDB, err := duckdbpkg.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite fresh: %v", err)
	}
	t.Cleanup(func() { _ = freshDB.Close() })
	// PAS de CREATE TABLE — schéma vide.

	sharedDB := setupTestSharedDB(t, map[string][]string{
		"999": {"m_from_shared"},
	})
	opener := func(ctx context.Context, gt string) (*sql.DB, func(), error) {
		return freshDB.SQLDb(), func() {}, nil
	}
	loader := NewKnownLoader(opener, func() *sql.DB { return sharedDB })

	known, err := loader.LoadKnown(context.Background(), PlayerProfile{
		Gamertag: "newplayer", XUID: "999",
	})
	if err != nil {
		t.Fatalf("err = %v (tolérance schéma vide attendue)", err)
	}
	if len(known) != 1 || !known["m_from_shared"] {
		t.Errorf("known = %v, want {m_from_shared:true}", known)
	}
}

func TestKnownLoaderV2_ReleaseCalledEvenOnError(t *testing.T) {
	// Garde-rail : release() doit être appelé même si Source 1 fail.
	// Vérifié via flag dans la closure.
	playerDB, _ := setupTestPlayerDB(t, []string{"m1"})
	released := false
	opener := func(ctx context.Context, gt string) (*sql.DB, func(), error) {
		return playerDB.SQLDb(), func() { released = true }, nil
	}
	loader := NewKnownLoader(opener, func() *sql.DB { return nil })
	_, _ = loader.LoadKnown(context.Background(), PlayerProfile{Gamertag: "alice"})
	if !released {
		t.Error("release() not called — defer leak")
	}
}
