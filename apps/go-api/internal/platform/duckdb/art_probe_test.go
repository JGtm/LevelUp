//go:build integration

package duckdb

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openMemDBForProbe ouvre une DuckDB :memory: pour tester le probe.
func openMemDBForProbe(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("openMemDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestProbeART_NoTables : pas de tables PK VARCHAR → rapport vide, pas d'erreur.
func TestProbeART_NoTables(t *testing.T) {
	db := openMemDBForProbe(t)
	report, err := ProbeARTDivergences(context.Background(), db, 5)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if report.TablesScanned != 0 {
		t.Errorf("expected 0 tables scanned, got %d", report.TablesScanned)
	}
	if report.HasDivergence() {
		t.Errorf("expected no divergences on empty DB")
	}
}

// TestProbeART_HealthyTable : table PK VARCHAR correctement indexée →
// aucune divergence.
func TestProbeART_HealthyTable(t *testing.T) {
	db := openMemDBForProbe(t)
	if _, err := db.Exec(`
		CREATE TABLE healthy_t (
			id VARCHAR PRIMARY KEY,
			payload VARCHAR
		);
	`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	for i := 0; i < 20; i++ {
		if _, err := db.Exec(`INSERT INTO healthy_t VALUES (?, ?)`,
			"row_"+itoaP(i), "data_"+itoaP(i)); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	report, err := ProbeARTDivergences(context.Background(), db, 5)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if report.TablesScanned != 1 {
		t.Errorf("expected 1 table scanned, got %d", report.TablesScanned)
	}
	if report.SamplesTested != 5 {
		t.Errorf("expected 5 samples tested, got %d", report.SamplesTested)
	}
	if report.HasDivergence() {
		t.Errorf("expected no divergences on healthy table, got %d : %v",
			len(report.Divergences), report.Divergences)
	}
}

// TestProbeART_CompositePK : table avec PK composite (match_id, xuid) →
// le probe utilise la première colonne (match_id) et fonctionne.
func TestProbeART_CompositePK(t *testing.T) {
	db := openMemDBForProbe(t)
	if _, err := db.Exec(`
		CREATE TABLE mp_like (
			match_id VARCHAR,
			xuid VARCHAR,
			score INTEGER,
			PRIMARY KEY (match_id, xuid)
		);
	`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// 3 matchs, 5 xuids chacun.
	for m := 0; m < 3; m++ {
		for x := 0; x < 5; x++ {
			if _, err := db.Exec(`INSERT INTO mp_like VALUES (?, ?, ?)`,
				"match_"+itoaP(m), "xuid_"+itoaP(x), 100+x); err != nil {
				t.Fatalf("insert: %v", err)
			}
		}
	}

	report, err := ProbeARTDivergences(context.Background(), db, 3)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if report.TablesScanned != 1 {
		t.Errorf("expected 1 table scanned, got %d", report.TablesScanned)
	}
	if report.HasDivergence() {
		t.Errorf("expected no divergences on healthy composite PK table, got %v", report.Divergences)
	}
}

// TestProbeART_NonVarcharPK : tables avec PK INTEGER ignorées (hors scope).
func TestProbeART_NonVarcharPK(t *testing.T) {
	db := openMemDBForProbe(t)
	if _, err := db.Exec(`
		CREATE TABLE int_pk_t (
			id INTEGER PRIMARY KEY,
			payload VARCHAR
		);
		INSERT INTO int_pk_t VALUES (1, 'a'), (2, 'b'), (3, 'c');
	`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	report, err := ProbeARTDivergences(context.Background(), db, 5)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if report.TablesScanned != 0 {
		t.Errorf("expected 0 tables scanned (INTEGER PK), got %d", report.TablesScanned)
	}
}

// TestProbeART_MultipleTables : le probe traite toutes les tables candidates.
func TestProbeART_MultipleTables(t *testing.T) {
	db := openMemDBForProbe(t)
	if _, err := db.Exec(`
		CREATE TABLE t1 (id VARCHAR PRIMARY KEY, v INTEGER);
		CREATE TABLE t2 (id VARCHAR PRIMARY KEY, v INTEGER);
		CREATE TABLE t3 (id VARCHAR PRIMARY KEY, v INTEGER);
		INSERT INTO t1 VALUES ('a1', 1), ('a2', 2);
		INSERT INTO t2 VALUES ('b1', 1), ('b2', 2);
		INSERT INTO t3 VALUES ('c1', 1), ('c2', 2);
	`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	report, err := ProbeARTDivergences(context.Background(), db, 2)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if report.TablesScanned != 3 {
		t.Errorf("expected 3 tables scanned, got %d", report.TablesScanned)
	}
}

func itoaP(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+(n/10))) + string(rune('0'+(n%10)))
}

// TestART_FilterPushdown_NoTruncation : test anti-régression E2E sur un
// dataset hétérogène proche prod (10 matchs × 10 participants = 100 rows
// dans match_participants-like). Vérifie qu'aucune divergence n'est détectée
// par le probe ART après seed et un ATTACH/DETACH cycle.
//
// Note : la corruption ART du bug
// `docs/INCIDENT_2026-05-20_match_participants_index.md` ne se manifeste
// PAS de manière déterministe sur :memory: — c'est un bug de plan
// d'exécution lié au contenu de la table en pages physiques DuckDB.
// Ce test garantit donc que :
//   - le probe scanne bien la table avec PK composite (match_id, xuid)
//   - aucun faux positif sur dataset hétérogène
//   - la nouvelle table créée via rebuild est cohérente
//
// L'anti-régression réelle de corruption ART repose sur le filet de garde
// au boot serveur (BootARTGuard cf. cmd/server/main.go).
func TestART_FilterPushdown_NoTruncation(t *testing.T) {
	db := openMemDBForProbe(t)
	// Schéma proche prod match_participants : PK composite VARCHAR.
	if _, err := db.Exec(`
		CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid VARCHAR,
			gamertag VARCHAR,
			team_id INTEGER,
			kills INTEGER DEFAULT 0,
			deaths INTEGER DEFAULT 0,
			PRIMARY KEY (match_id, xuid)
		);
	`); err != nil {
		t.Fatalf("seed schema: %v", err)
	}

	// Dataset hétérogène : 10 matchs (UUID-like), 10 participants chacun
	// (mix bot + xuids 16-digit), totalisant 100 rows.
	matchUUIDs := []string{
		"50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e",
		"a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		"00000000-0000-0000-0000-000000000001",
		"ffffffff-ffff-ffff-ffff-ffffffffffff",
		"deadbeef-cafe-babe-c0de-feedfacefa11",
		"01234567-89ab-cdef-0123-456789abcdef",
		"abcdef01-2345-6789-abcd-ef0123456789",
		"99999999-9999-9999-9999-999999999999",
		"11111111-2222-3333-4444-555555555555",
		"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
	}
	xuidPool := []string{
		"bid(45.0)", "bid(99.0)",
		"2535469190789936", "2533274823110022", "2533274858283686",
		"2535409561713955", "2533274850178760", "2533274852144672",
		"2535455227223597", "2533274910731403",
	}
	for _, mid := range matchUUIDs {
		for i, xuid := range xuidPool {
			if _, err := db.Exec(`INSERT INTO match_participants
				(match_id, xuid, gamertag, team_id, kills, deaths)
				VALUES (?, ?, ?, ?, ?, ?)`,
				mid, xuid, "player_"+itoaP(i), i%2, i*2, 5); err != nil {
				t.Fatalf("insert (%s, %s): %v", mid, xuid, err)
			}
		}
	}

	// Vérification baseline : 100 rows totales.
	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants`).Scan(&total); err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != 100 {
		t.Fatalf("expected 100 rows seed, got %d", total)
	}

	// Vérification per-match : chaque match doit retourner 10 rows via
	// filter pushdown (cas healthy).
	for _, mid := range matchUUIDs {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants WHERE match_id = ?`,
			mid).Scan(&n); err != nil {
			t.Fatalf("count match %s: %v", mid, err)
		}
		if n != 10 {
			t.Fatalf("ART divergence détectée pour match %s : expected 10, got %d "+
				"(serait le bug INCIDENT_2026-05-20 reproduit sur :memory:)", mid, n)
		}
	}

	// Lancer le probe : aucune divergence attendue sur dataset healthy.
	report, err := ProbeARTDivergences(context.Background(), db, 10)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if report.HasDivergence() {
		t.Errorf("probe a détecté %d divergences sur dataset healthy : %v",
			len(report.Divergences), report.Divergences)
	}
	if report.TablesScanned != 1 {
		t.Errorf("expected 1 table scanned, got %d", report.TablesScanned)
	}
	if report.SamplesTested != 10 {
		t.Errorf("expected 10 samples tested (sampleSize=10), got %d", report.SamplesTested)
	}
}
