//go:build integration

// Package sync — ensure_enrichment_rows_test.go : tests du backfill des
// rows player_match_enrichment manquantes pour les matchs déjà présents
// en shared (incident 2026-05-27).
//
// Voir ensure_enrichment_rows.go pour le contexte du bug.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// setupTwoDBs ouvre 2 DuckDB in-memory (player + shared) avec le schéma
// minimal pour tester ensurePlayerEnrichmentRows : shared.match_participants
// et player_match_enrichment.
func setupTwoDBs(t *testing.T) (player, shared *sql.DB) {
	t.Helper()

	playerDB, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open player: %v", err)
	}
	t.Cleanup(func() { _ = playerDB.Close() })
	if _, err := playerDB.Exec(`
		CREATE TABLE player_match_enrichment (
			match_id          VARCHAR PRIMARY KEY,
			performance_score FLOAT,
			session_id        VARCHAR,
			created_at        TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		)`); err != nil {
		t.Fatalf("create player_match_enrichment: %v", err)
	}
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(playerDB); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}

	sharedDB, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open shared: %v", err)
	}
	t.Cleanup(func() { _ = sharedDB.Close() })
	if _, err := sharedDB.Exec(`
		CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid     VARCHAR,
			gamertag VARCHAR
		)`); err != nil {
		t.Fatalf("create match_participants: %v", err)
	}

	return playerDB, sharedDB
}

// TestEnsurePlayerEnrichmentRows_CreatesMissingRows : cas Madina/Choco —
// le joueur apparaît dans 8 matchs en shared mais 0 row dans
// player_match_enrichment → 8 rows créées.
func TestEnsurePlayerEnrichmentRows_CreatesMissingRows(t *testing.T) {
	playerDB, sharedDB := setupTwoDBs(t)
	ctx := context.Background()

	const xuid = "2533274858283686" // Madina
	matchIDs := []string{"m1", "m2", "m3", "m4", "m5", "m6", "m7", "m8"}
	for _, id := range matchIDs {
		if _, err := sharedDB.Exec(`INSERT INTO match_participants VALUES (?, ?, ?)`,
			id, xuid, "Madina97294"); err != nil {
			t.Fatalf("insert shared: %v", err)
		}
	}

	n, err := ensurePlayerEnrichmentRows(ctx, playerDB, sharedDB, xuid)
	if err != nil {
		t.Fatalf("ensurePlayerEnrichmentRows: %v", err)
	}
	if n != 8 {
		t.Errorf("rows créées = %d, want 8", n)
	}

	// Vérifier que les 8 rows sont bien là.
	var count int
	if err := playerDB.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 8 {
		t.Errorf("count player_match_enrichment = %d, want 8", count)
	}
}

// TestEnsurePlayerEnrichmentRows_Idempotent : 2ème appel ne crée pas de
// doublons (cas stationnaire — sync subséquent après backfill).
func TestEnsurePlayerEnrichmentRows_Idempotent(t *testing.T) {
	playerDB, sharedDB := setupTwoDBs(t)
	ctx := context.Background()

	const xuid = "xuid_test"
	for _, id := range []string{"m1", "m2", "m3"} {
		if _, err := sharedDB.Exec(`INSERT INTO match_participants VALUES (?, ?, ?)`,
			id, xuid, "TestPlayer"); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	// 1er appel : 3 rows créées.
	n1, err := ensurePlayerEnrichmentRows(ctx, playerDB, sharedDB, xuid)
	if err != nil {
		t.Fatalf("call 1: %v", err)
	}
	if n1 != 3 {
		t.Errorf("call 1 rows = %d, want 3", n1)
	}

	// 2ème appel : 0 row créée (idempotence).
	n2, err := ensurePlayerEnrichmentRows(ctx, playerDB, sharedDB, xuid)
	if err != nil {
		t.Fatalf("call 2: %v", err)
	}
	if n2 != 0 {
		t.Errorf("call 2 rows = %d, want 0 (idempotence)", n2)
	}
}

// TestEnsurePlayerEnrichmentRows_PreservesExistingRows : si certaines rows
// existent déjà (cas mixte JGtm où PlayerPersister a inséré 5 rows et 3
// manquent), seul le delta est créé et les anciennes ne sont pas touchées.
func TestEnsurePlayerEnrichmentRows_PreservesExistingRows(t *testing.T) {
	playerDB, sharedDB := setupTwoDBs(t)
	ctx := context.Background()

	const xuid = "xuid_jgtm"

	// 8 matchs en shared.
	allMatches := []string{"m1", "m2", "m3", "m4", "m5", "m6", "m7", "m8"}
	for _, id := range allMatches {
		if _, err := sharedDB.Exec(`INSERT INTO match_participants VALUES (?, ?, ?)`,
			id, xuid, "JGtm"); err != nil {
			t.Fatalf("insert shared: %v", err)
		}
	}
	// 5 rows déjà en player avec des values significatives (à préserver).
	for i, id := range allMatches[:5] {
		score := float64(80 + i)
		if _, err := playerDB.Exec(
			`INSERT INTO player_match_enrichment (match_id, performance_score) VALUES (?, ?)`,
			id, score); err != nil {
			t.Fatalf("insert player: %v", err)
		}
	}

	n, err := ensurePlayerEnrichmentRows(ctx, playerDB, sharedDB, xuid)
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	// Seules les 3 manquantes (m6, m7, m8) doivent être créées.
	if n != 3 {
		t.Errorf("rows créées = %d, want 3 (delta only)", n)
	}

	// Les 5 anciennes doivent toujours avoir leur performance_score.
	for i, id := range allMatches[:5] {
		var score sql.NullFloat64
		if err := playerDB.QueryRow(
			`SELECT performance_score FROM player_match_enrichment WHERE match_id = ?`,
			id).Scan(&score); err != nil {
			t.Fatalf("select %s: %v", id, err)
		}
		expected := float64(80 + i)
		if !score.Valid || score.Float64 != expected {
			t.Errorf("%s: performance_score = %v, want %f (ne doit pas être touché)", id, score, expected)
		}
	}

	// Les 3 nouvelles doivent exister avec performance_score NULL.
	for _, id := range allMatches[5:] {
		var score sql.NullFloat64
		if err := playerDB.QueryRow(
			`SELECT performance_score FROM player_match_enrichment WHERE match_id = ?`,
			id).Scan(&score); err != nil {
			t.Fatalf("select %s: %v", id, err)
		}
		if score.Valid {
			t.Errorf("%s: performance_score = %v, want NULL (row vierge)", id, score)
		}
	}
}

// TestEnsurePlayerEnrichmentRows_FiltersByXUID : si la table shared contient
// des matchs où d'AUTRES joueurs participent (mais pas le xuid demandé), ces
// matchs ne doivent PAS créer de row pour le xuid demandé.
func TestEnsurePlayerEnrichmentRows_FiltersByXUID(t *testing.T) {
	playerDB, sharedDB := setupTwoDBs(t)
	ctx := context.Background()

	const myXUID = "my_xuid"
	const otherXUID = "other_xuid"

	// 3 matchs où je participe.
	for _, id := range []string{"mine1", "mine2", "mine3"} {
		_, _ = sharedDB.Exec(`INSERT INTO match_participants VALUES (?, ?, ?)`, id, myXUID, "Me")
	}
	// 4 matchs où SEUL un autre joueur participe.
	for _, id := range []string{"other1", "other2", "other3", "other4"} {
		_, _ = sharedDB.Exec(`INSERT INTO match_participants VALUES (?, ?, ?)`, id, otherXUID, "Other")
	}

	n, err := ensurePlayerEnrichmentRows(ctx, playerDB, sharedDB, myXUID)
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if n != 3 {
		t.Errorf("rows créées = %d, want 3 (seuls les match de myXUID)", n)
	}

	// Vérifier que les 4 "other" matchs ne sont PAS en player_match_enrichment.
	for _, id := range []string{"other1", "other2", "other3", "other4"} {
		var present int
		_ = playerDB.QueryRow(
			`SELECT COUNT(*) FROM player_match_enrichment WHERE match_id = ?`, id).Scan(&present)
		if present > 0 {
			t.Errorf("match %s (other_xuid) ne devrait PAS être inséré pour myXUID", id)
		}
	}
}

// TestEnsurePlayerEnrichmentRows_NilSharedDB : nil sharedDB → no-op sans
// erreur (cas tests / boot avant shared init).
func TestEnsurePlayerEnrichmentRows_NilSharedDB(t *testing.T) {
	playerDB, _ := setupTwoDBs(t)

	n, err := ensurePlayerEnrichmentRows(context.Background(), playerDB, nil, "xuid")
	if err != nil {
		t.Errorf("err non nil avec sharedDB nil: %v", err)
	}
	if n != 0 {
		t.Errorf("rows = %d, want 0", n)
	}
}

// TestEnsurePlayerEnrichmentRows_PlayerHistoryLargerThanShared : cas réel
// observé en panic 2026-05-27 — un joueur a un GROS historique
// player_match_enrichment (1000 matchs anciens) mais seuls quelques matchs
// récents en shared.match_participants pour son xuid. Avant le fix du cap,
// `make([]string, 0, len(shared)-len(player))` calculait un cap négatif
// → panic `makeslice: cap out of range`.
func TestEnsurePlayerEnrichmentRows_PlayerHistoryLargerThanShared(t *testing.T) {
	playerDB, sharedDB := setupTwoDBs(t)
	ctx := context.Background()

	const xuid = "xuid_madina"

	// 1000 rows en player (gros historique).
	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("ancient_%04d", i)
		if _, err := playerDB.Exec(
			`INSERT INTO player_match_enrichment (match_id) VALUES (?)`, id); err != nil {
			t.Fatalf("seed player: %v", err)
		}
	}

	// 5 matchs en shared (récents) — 3 chevauchent player + 2 réellement nouveaux.
	allShared := []string{"ancient_0500", "ancient_0501", "ancient_0502", "new_1", "new_2"}
	for _, id := range allShared {
		if _, err := sharedDB.Exec(`INSERT INTO match_participants VALUES (?, ?, ?)`,
			id, xuid, "Madina"); err != nil {
			t.Fatalf("seed shared: %v", err)
		}
	}

	// L'ancien code paniquait ici sur `make([]string, 0, 5-1000)`.
	n, err := ensurePlayerEnrichmentRows(ctx, playerDB, sharedDB, xuid)
	if err != nil {
		t.Fatalf("panic ou erreur : %v", err)
	}
	if n != 2 {
		t.Errorf("rows créées = %d, want 2 (new_1, new_2)", n)
	}
}

// TestEnsurePlayerEnrichmentRows_EmptyXUID : xuid vide → no-op sans erreur.
func TestEnsurePlayerEnrichmentRows_EmptyXUID(t *testing.T) {
	playerDB, sharedDB := setupTwoDBs(t)

	n, err := ensurePlayerEnrichmentRows(context.Background(), playerDB, sharedDB, "")
	if err != nil {
		t.Errorf("err non nil avec xuid vide: %v", err)
	}
	if n != 0 {
		t.Errorf("rows = %d, want 0", n)
	}
}
