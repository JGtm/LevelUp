// Package v2 — known_loader_test.go : tests du KnownLoader V2 avec DuckDB temp réelle (pas de
// mock — le SQL exact de la règle knownset contre le moteur de prod). La règle elle-même est
// testée dans internal/sync/knownset ; ici, la délégation et les échecs.
//
// Pas de build tag : ces tests tournent par défaut dans `go test ./...`.
package v2

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/migration"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync/knownset"
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

// setupTestSharedDB crée une shared_matches_v2.duckdb minimaliste : match_registry +
// match_participants. Chaque match d'un participant est inscrit au registre (comme la
// persistance l'écrit, en une transaction) ; registryOnly ajoute des matchs au registre seul.
func setupTestSharedDB(t *testing.T, participantsByXUID map[string][]string, registryOnly ...string) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "shared.duckdb")
	db, err := duckdbpkg.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite shared: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.SQLDb().Exec(`
		CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY);
		CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR);
	`); err != nil {
		t.Fatalf("create shared tables: %v", err)
	}
	inRegistry := map[string]bool{}
	register := func(mID string) {
		if inRegistry[mID] {
			return
		}
		inRegistry[mID] = true
		if _, err := db.SQLDb().Exec("INSERT INTO match_registry (match_id) VALUES (?)", mID); err != nil {
			t.Fatalf("insert registry %s: %v", mID, err)
		}
	}
	for xuid, matchIDs := range participantsByXUID {
		for _, mID := range matchIDs {
			register(mID)
			if _, err := db.SQLDb().Exec(
				"INSERT INTO match_participants (match_id, xuid) VALUES (?, ?)", mID, xuid,
			); err != nil {
				t.Fatalf("insert participant: %v", err)
			}
		}
	}
	for _, mID := range registryOnly {
		register(mID)
	}
	return db.SQLDb()
}

// ─── Tests ────────────────────────────────────────────────────────────

// TestKnownLoaderV2_ConnuSeulementSiAuRegistre : le loader applique la règle knownset —
// participants du xuid au registre connus, enrichissement sans registre INCONNU (re-récupéré),
// participants d'un autre xuid absents.
func TestKnownLoaderV2_ConnuSeulementSiAuRegistre(t *testing.T) {
	playerDB, _ := setupTestPlayerDB(t, []string{"m1", "m2", "m_orphelin"})
	sharedDB := setupTestSharedDB(t, map[string][]string{
		"999": {"m1", "m2", "m3"},
		"888": {"m_other"},
	})
	opener := func(ctx context.Context, gt string) (*sql.DB, func(), error) {
		return playerDB.SQLDb(), func() {}, nil
	}
	loader := NewKnownLoader(opener, SharedBorrower(nil, func() *sql.DB { return sharedDB }))

	known, err := knownOf(loader.LoadKnown(context.Background(), PlayerProfile{Gamertag: "alice", XUID: "999"}))
	if err != nil {
		t.Fatalf("LoadKnown err = %v", err)
	}
	if len(known) != 3 {
		t.Errorf("known = %v, want {m1, m2, m3}", known)
	}
	for _, mID := range []string{"m1", "m2", "m3"} {
		if !known[mID] {
			t.Errorf("known[%s] = false", mID)
		}
	}
	if known["m_orphelin"] {
		t.Error("known[m_orphelin] = true : enrichissement sans registre traité comme connu (jamais re-récupéré)")
	}
	if known["m_other"] {
		t.Error("known[m_other] = true (fuite inter-xuid)")
	}
}

// TestKnownLoaderV2_BasePartageeAbsenteEstFatale : getSharedDB rend nil (connexion non ouverte,
// swap en cours) → erreur typée, aucun ensemble. Un ensemble « enrichissements seuls » ferait
// sauter des matchs que la base partagée n'a pas ; un ensemble vide ferait tout retélécharger.
func TestKnownLoaderV2_BasePartageeAbsenteEstFatale(t *testing.T) {
	playerDB, _ := setupTestPlayerDB(t, []string{"m1"})
	opener := func(ctx context.Context, gt string) (*sql.DB, func(), error) {
		return playerDB.SQLDb(), func() {}, nil
	}
	loader := NewKnownLoader(opener, SharedBorrower(nil, func() *sql.DB { return nil }))

	known, err := knownOf(loader.LoadKnown(context.Background(), PlayerProfile{Gamertag: "alice", XUID: "999"}))
	if !errors.Is(err, knownset.ErrSharedUnreadable) {
		t.Fatalf("err = %v, want knownset.ErrSharedUnreadable", err)
	}
	if known != nil {
		t.Errorf("known = %v, want nil", known)
	}
}

// TestKnownLoaderV2_XUIDVideEstFatal : sans xuid, rien ne borne les participants au joueur.
func TestKnownLoaderV2_XUIDVideEstFatal(t *testing.T) {
	playerDB, _ := setupTestPlayerDB(t, []string{"m1"})
	sharedDB := setupTestSharedDB(t, map[string][]string{"999": {"m1"}})
	opener := func(ctx context.Context, gt string) (*sql.DB, func(), error) {
		return playerDB.SQLDb(), func() {}, nil
	}
	loader := NewKnownLoader(opener, SharedBorrower(nil, func() *sql.DB { return sharedDB }))

	_, err := loader.LoadKnown(context.Background(), PlayerProfile{Gamertag: "alice", XUID: "  "})
	if !errors.Is(err, knownset.ErrNoXUID) {
		t.Fatalf("err = %v, want knownset.ErrNoXUID", err)
	}
}

func TestKnownLoaderV2_OpenPlayerDBFailureIsFatal(t *testing.T) {
	// Si openPlayerDB échoue (cas pathologique, ne devrait pas arriver
	// en prod), on retourne erreur. Discovery capture dans Errors.
	opener := func(ctx context.Context, gt string) (*sql.DB, func(), error) {
		return nil, nil, sql.ErrConnDone
	}
	loader := NewKnownLoader(opener, SharedBorrower(nil, func() *sql.DB { return nil }))
	_, err := loader.LoadKnown(context.Background(), PlayerProfile{Gamertag: "alice"})
	if err == nil {
		t.Fatal("LoadKnown should return err when openPlayerDB fails")
	}
}

func TestKnownLoaderV2_PlayerTableMissingIsTolerated(t *testing.T) {
	// Base joueur neuve (vue d'enrichissement absente) : la règle ne dépend que de la base
	// partagée, le chargement aboutit. Cas réaliste : 1er sync d'un joueur.
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
	loader := NewKnownLoader(opener, SharedBorrower(nil, func() *sql.DB { return sharedDB }))

	known, err := knownOf(loader.LoadKnown(context.Background(), PlayerProfile{
		Gamertag: "newplayer", XUID: "999",
	}))
	if err != nil {
		t.Fatalf("err = %v (tolérance schéma vide attendue)", err)
	}
	if len(known) != 1 || !known["m_from_shared"] {
		t.Errorf("known = %v, want {m_from_shared:true}", known)
	}
}

func TestKnownLoaderV2_ReleaseCalledEvenOnError(t *testing.T) {
	// Garde-rail : release() doit être appelé même si le chargement échoue
	// (ici : base partagée absente). Vérifié via flag dans la closure.
	playerDB, _ := setupTestPlayerDB(t, []string{"m1"})
	released := false
	opener := func(ctx context.Context, gt string) (*sql.DB, func(), error) {
		return playerDB.SQLDb(), func() { released = true }, nil
	}
	loader := NewKnownLoader(opener, SharedBorrower(nil, func() *sql.DB { return nil }))
	if _, err := loader.LoadKnown(context.Background(), PlayerProfile{Gamertag: "alice", XUID: "999"}); err == nil {
		t.Fatal("LoadKnown sans base partagée doit échouer")
	}
	if !released {
		t.Error("release() not called — defer leak")
	}
}

// knownOf projette le résultat de LoadKnown sur l'ensemble connu (Known nil en erreur).
func knownOf(set knownset.Set, err error) (map[string]bool, error) {
	return set.Known, err
}
