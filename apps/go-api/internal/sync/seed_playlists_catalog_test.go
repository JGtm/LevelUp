package sync

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openCatalogMemDB crée une DB in-memory avec la table playlists_catalog.
func openCatalogMemDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open mem db: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE playlists_catalog (
			title_slug         VARCHAR NOT NULL,
			playlist_asset_id  VARCHAR NOT NULL,
			current_version_id VARCHAR NOT NULL DEFAULT '',
			name_canonical     VARCHAR,
			experience         VARCHAR,
			is_ranked          BOOLEAN DEFAULT FALSE,
			is_active          BOOLEAN DEFAULT TRUE,
			first_seen_at      TIMESTAMPTZ,
			last_seen_at       TIMESTAMPTZ,
			PRIMARY KEY (title_slug, playlist_asset_id)
		)`)
	if err != nil {
		t.Fatalf("create playlists_catalog: %v", err)
	}
	return db
}

func countCatalogRows(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM playlists_catalog`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

// TestSeedPlaylistsCatalog_NoTableNoOp : sur une metadata SANS playlists_catalog
// (cas halo_5, dont la metadata.duckdb est isolée des référentiels HINF), le seed
// doit no-op proprement — ni panique, ni tentative d'UPDATE/INSERT (qui spammait un
// WARN par playlist à chaque cycle de post-sync CSR). Régression 2026-06-26.
func TestSeedPlaylistsCatalog_NoTableNoOp(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open mem db: %v", err)
	}
	defer db.Close()
	// Aucune table playlists_catalog créée (metadata h5 isolée).
	csrs := []PlayerPlaylistCSR{
		{PlaylistID: "bbbbbbbb-0000-0000-0000-000000000001", PlaylistName: "Arena classée"},
	}
	// Le guard (existence de table) doit court-circuiter avant tout accès à la table
	// absente : aucune erreur SQL ne remonte, aucune panique.
	seedPlaylistsCatalog(context.Background(), db, csrs, "halo_5")

	// La table ne doit pas avoir été créée par effet de bord.
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'playlists_catalog'`,
	).Scan(&n); err != nil {
		t.Fatalf("count tables: %v", err)
	}
	if n != 0 {
		t.Errorf("playlists_catalog ne devrait pas exister (table absente, guard actif), got %d", n)
	}
}

func TestSeedPlaylistsCatalog_InsertsNewPlaylists(t *testing.T) {
	db := openCatalogMemDB(t)
	defer db.Close()

	csrs := []PlayerPlaylistCSR{
		{PlaylistID: "aaaaaaaa-0000-0000-0000-000000000001", PlaylistName: "Ranked Arena"},
		{PlaylistID: "aaaaaaaa-0000-0000-0000-000000000002", PlaylistName: "Ranked Snipers"},
	}

	seedPlaylistsCatalog(context.Background(), db, csrs, "halo_infinite")

	if got := countCatalogRows(t, db); got != 2 {
		t.Errorf("attendu 2 lignes, got %d", got)
	}
}

func TestSeedPlaylistsCatalog_Idempotent(t *testing.T) {
	db := openCatalogMemDB(t)
	defer db.Close()

	csrs := []PlayerPlaylistCSR{
		{PlaylistID: "aaaaaaaa-0000-0000-0000-000000000001", PlaylistName: "Ranked Arena"},
	}

	seedPlaylistsCatalog(context.Background(), db, csrs, "halo_infinite")
	seedPlaylistsCatalog(context.Background(), db, csrs, "halo_infinite") // doit être no-op

	if got := countCatalogRows(t, db); got != 1 {
		t.Errorf("ON CONFLICT DO NOTHING violé : attendu 1, got %d", got)
	}
}

func TestSeedPlaylistsCatalog_SkipsEmptyPlaylistID(t *testing.T) {
	db := openCatalogMemDB(t)
	defer db.Close()

	csrs := []PlayerPlaylistCSR{
		{PlaylistID: "", PlaylistName: "Should be ignored"},
		{PlaylistID: "   ", PlaylistName: "Also ignored"},
		{PlaylistID: "aaaaaaaa-0000-0000-0000-000000000001", PlaylistName: "Valid"},
	}

	seedPlaylistsCatalog(context.Background(), db, csrs, "halo_infinite")

	if got := countCatalogRows(t, db); got != 1 {
		t.Errorf("attendu 1 (IDs vides ignorés), got %d", got)
	}
}

func TestSeedPlaylistsCatalog_UUIDAsFallbackName(t *testing.T) {
	db := openCatalogMemDB(t)
	defer db.Close()

	id := "bbbbbbbb-1111-1111-1111-111111111111"
	csrs := []PlayerPlaylistCSR{
		{PlaylistID: id, PlaylistName: id}, // nom = UUID → fallback sur l'ID
	}

	seedPlaylistsCatalog(context.Background(), db, csrs, "halo_infinite")

	var name string
	if err := db.QueryRow(`SELECT name_canonical FROM playlists_catalog WHERE playlist_asset_id = ?`, id).Scan(&name); err != nil {
		t.Fatalf("select: %v", err)
	}
	if name != id {
		t.Errorf("name_canonical attendu %q, got %q", id, name)
	}
}

func TestSeedPlaylistsCatalog_SetsRankedAndTimestamps(t *testing.T) {
	db := openCatalogMemDB(t)
	defer db.Close()

	before := time.Now().UTC().Add(-time.Second)
	csrs := []PlayerPlaylistCSR{
		{PlaylistID: "cccccccc-0000-0000-0000-000000000001", PlaylistName: "Ranked Doubles"},
	}
	seedPlaylistsCatalog(context.Background(), db, csrs, "halo_infinite")

	var isRanked bool
	var firstSeen, lastSeen time.Time
	err := db.QueryRow(`
		SELECT is_ranked, first_seen_at, last_seen_at FROM playlists_catalog
		WHERE playlist_asset_id = 'cccccccc-0000-0000-0000-000000000001'
	`).Scan(&isRanked, &firstSeen, &lastSeen)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if !isRanked {
		t.Error("is_ranked doit être TRUE")
	}
	if firstSeen.Before(before) {
		t.Errorf("first_seen_at trop ancien: %v", firstSeen)
	}
	if lastSeen.Before(before) {
		t.Errorf("last_seen_at trop ancien: %v", lastSeen)
	}
}

func TestIsUUIDLike(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", true},
		{"Ranked Arena", false},
		{"", false},
		{"too-short", false},
		{"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeeX", true}, // 36 chars, 4 tirets
	}
	for _, c := range cases {
		if got := isUUIDLike(c.input); got != c.want {
			t.Errorf("isUUIDLike(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}

func openCSRSnapshotsMemDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open mem db: %v", err)
	}
	_, err = db.Exec(`
		CREATE SEQUENCE IF NOT EXISTS pcs_seq START 1;
		CREATE TABLE player_csr_snapshots (
			id                              INTEGER PRIMARY KEY DEFAULT nextval('pcs_seq'),
			playlist_id                     VARCHAR,
			playlist_name                   VARCHAR,
			queue                           VARCHAR,
			input                           VARCHAR,
			season_id                       VARCHAR,
			current_value                   DOUBLE,
			current_tier                    VARCHAR,
			current_sub_tier                INTEGER,
			current_measurement_remaining   INTEGER,
			season_value                    DOUBLE,
			season_tier                     VARCHAR,
			season_sub_tier                 INTEGER,
			alltime_value                   DOUBLE,
			alltime_tier                    VARCHAR,
			alltime_sub_tier                INTEGER,
			fetched_at                      TIMESTAMPTZ,
			written_at                      TIMESTAMPTZ DEFAULT now()
		)`)
	if err != nil {
		t.Fatalf("create player_csr_snapshots: %v", err)
	}
	return db
}

func TestSyncPlayerCSRs_ReturnsCsrsOnSuccess(t *testing.T) {
	db := openCSRSnapshotsMemDB(t)
	defer db.Close()

	mock := &mockCSRClient{csrs: []PlayerPlaylistCSR{
		{PlaylistID: "aaaaaaaa-0000-0000-0000-000000000001", PlaylistName: "Ranked Arena"},
		{PlaylistID: "aaaaaaaa-0000-0000-0000-000000000002", PlaylistName: "Ranked Snipers"},
	}}

	got, err := syncPlayerCSRs(context.Background(), mock, db, "xuid(123)", "CsrSeason13-1")
	if err != nil {
		t.Fatalf("syncPlayerCSRs: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("attendu 2 CSRs retournés, got %d", len(got))
	}
}

func TestSyncPlayerCSRs_ReturnsNilOnEmptySeason(t *testing.T) {
	mock := &mockCSRClient{}
	got, err := syncPlayerCSRs(context.Background(), mock, nil, "xuid(123)", "")
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if got != nil {
		t.Errorf("attendu nil pour saison vide, got %v", got)
	}
}

// mockCSRClient implémente HaloClient pour les tests de syncPlayerCSRs.
type mockCSRClient struct {
	mockHaloClient
	csrs []PlayerPlaylistCSR
	err  error
}

func (m *mockCSRClient) GetPlayerCSRs(_ context.Context, _, _ string) ([]PlayerPlaylistCSR, error) {
	return m.csrs, m.err
}

func (m *mockCSRClient) GetPlaylistCsr(_ context.Context, _, _, _ string) (*PlayerPlaylistCSR, error) {
	return nil, nil
}
