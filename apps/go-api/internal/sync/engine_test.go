//go:build integration

// Package sync — engine_test.go : tests pour les fonctions utilitaires du moteur de sync.
//
// Sprint 47 T15 — couvrir loadKnownMatchIDs et les fonctions de cycle sync.
package sync

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// ── Tests loadKnownMatchIDs ──────────────────────────────────────────────────

func TestLoadKnownMatchIDs_EmptyTable(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	// Append-only #23046 : loadKnownMatchIDs lit la vue player_match_enrichment_latest.
	// La vue référence TOUTES les colonnes métier de la table → schéma complet
	// (EnsurePlayerSchema) requis avant la création de la vue (un CREATE TABLE
	// minimal manuel casse le binder de la vue).
	if err := EnsurePlayerSchema(t.Context(), db); err != nil {
		t.Fatalf("EnsurePlayerSchema: %v", err)
	}
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}

	known, err := loadKnownMatchIDs(t.Context(), db, nil, "")
	if err != nil {
		t.Fatalf("loadKnownMatchIDs: %v", err)
	}
	if len(known) != 0 {
		t.Errorf("attendu map vide, obtenu %d entrées", len(known))
	}
}

func TestLoadKnownMatchIDs_WithMatches(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	// Append-only #23046 : schéma complet requis avant la vue _latest (lue par
	// loadKnownMatchIDs) — un CREATE TABLE minimal casse le binder de la vue.
	if err := EnsurePlayerSchema(t.Context(), db); err != nil {
		t.Fatalf("EnsurePlayerSchema: %v", err)
	}
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}
	_, _ = db.Exec(`INSERT INTO player_match_enrichment (match_id) VALUES ('aabbccdd-0000-0000-0000-000000000001')`)
	_, _ = db.Exec(`INSERT INTO player_match_enrichment (match_id) VALUES ('aabbccdd-0000-0000-0000-000000000002')`)

	known, err := loadKnownMatchIDs(t.Context(), db, nil, "")
	if err != nil {
		t.Fatalf("loadKnownMatchIDs: %v", err)
	}
	if len(known) != 2 {
		t.Errorf("attendu 2 match_id, obtenu %d", len(known))
	}
	if !known["aabbccdd-0000-0000-0000-000000000001"] {
		t.Error("match_id #1 manquant dans la map")
	}
}

func TestLoadKnownMatchIDs_MissingTable(t *testing.T) {
	// Si la table n'existe pas → retourne map vide sans erreur
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	known, err := loadKnownMatchIDs(t.Context(), db, nil, "")
	if err != nil {
		t.Fatalf("attendu nil, obtenu %v", err)
	}
	if known == nil {
		t.Error("map ne devrait pas être nil")
	}
}

// TestLoadKnownMatchIDs_UnionWithSharedParticipants : régression du fix
// 2026-05-22 (cross-player dedup). Quand un autre joueur du cycle a déjà
// inséré le match en shared.match_participants pour notre xuid, on doit le
// voir comme "known" et skipper le fetch API, sinon on duplique 21 calls
// Halo par cycle multi-joueurs.
func TestLoadKnownMatchIDs_UnionWithSharedParticipants(t *testing.T) {
	playerDB, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open player: %v", err)
	}
	defer playerDB.Close()
	sharedDB, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open shared: %v", err)
	}
	defer sharedDB.Close()

	// Player DB : 1 enrichment row (match "self-1") déjà connu localement.
	// Append-only #23046 : schéma complet + vue _latest (lue par loadKnownMatchIDs).
	if err := EnsurePlayerSchema(t.Context(), playerDB); err != nil {
		t.Fatalf("EnsurePlayerSchema: %v", err)
	}
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(playerDB); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}
	_, _ = playerDB.Exec(`INSERT INTO player_match_enrichment (match_id) VALUES ('self-1')`)

	// Shared DB : 3 participants rows pour notre xuid (match "shared-1/2/3")
	// + 1 row pour un AUTRE xuid (match "other-1") qui ne doit PAS être inclus.
	_, _ = sharedDB.Exec(`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR)`)
	_, _ = sharedDB.Exec(`INSERT INTO match_participants VALUES
		('shared-1', '2533274823110022'),
		('shared-2', '2533274823110022'),
		('shared-3', '2533274823110022'),
		('other-1',  '9999999999999999')`)

	known, err := loadKnownMatchIDs(t.Context(), playerDB, sharedDB, "2533274823110022")
	if err != nil {
		t.Fatalf("loadKnownMatchIDs: %v", err)
	}

	// Attendu : self-1 (player) + shared-1/2/3 (shared pour notre xuid) = 4.
	// other-1 doit être absent (xuid différent).
	if len(known) != 4 {
		t.Errorf("len(known) = %d, want 4 (1 player + 3 shared)", len(known))
	}
	for _, want := range []string{"self-1", "shared-1", "shared-2", "shared-3"} {
		if !known[want] {
			t.Errorf("match %q manquant dans known", want)
		}
	}
	if known["other-1"] {
		t.Error("match 'other-1' (autre xuid) ne devrait PAS être dans known")
	}
}

// TestLoadKnownMatchIDs_NilSharedFallsBackToPlayer : sharedDB=nil doit
// fonctionner (cas tests minimaux / boot avant shared init).
func TestLoadKnownMatchIDs_NilSharedFallsBackToPlayer(t *testing.T) {
	playerDB, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer playerDB.Close()
	// Append-only #23046 : schéma complet + vue _latest (lue par loadKnownMatchIDs).
	if err := EnsurePlayerSchema(t.Context(), playerDB); err != nil {
		t.Fatalf("EnsurePlayerSchema: %v", err)
	}
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(playerDB); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}
	_, _ = playerDB.Exec(`INSERT INTO player_match_enrichment (match_id) VALUES ('only-self')`)

	known, err := loadKnownMatchIDs(t.Context(), playerDB, nil, "2533274823110022")
	if err != nil {
		t.Fatalf("loadKnownMatchIDs: %v", err)
	}
	if len(known) != 1 || !known["only-self"] {
		t.Errorf("known = %v, want {only-self: true}", known)
	}
}
