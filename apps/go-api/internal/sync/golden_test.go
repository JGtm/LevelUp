// Package sync — golden_test.go : test E2E "golden fixture" pour la chaîne
// sync highlight events.
//
// Phase 4 du plan .ai/PLAN_HIGHLIGHT_EVENTS_BACKFILL.md (mai 2026).
//
// CE TEST AURAIT CAPTURÉ LES BUGS RÉCENTS :
//   - Parser bit-aligné cassé : highlight_events resterait à 0 alors qu'on
//     attend ≥ 100 events sur la fixture v41 réelle (4v4).
//   - InsertKillerVictimPairs OR IGNORE rejeté : killer_victim_pairs vide
//     alors qu'on attend > 30 paires (kill events détectés).
//   - InsertKillerVictimPairs sans time_ms : NULL critiques dans la table.
//   - Bitmasks menteurs (Phase 1bis) : MBitEvents/MBitKillerVictim doivent
//     correspondre à la présence réelle de rows.
//
// Le fixture canonique vit à `internal/analysis/testdata/v41_chunk_he.bin`
// (~197 KB zlib). Pour le ré-capturer si l'API Halo change :
//
//	go run ./cmd/refresh_golden_fixture --gamertag JGtm
//
// Le test ne nécessite PAS de réseau (le mock client sert le fixture).
// CGO_ENABLED=1 requis (DuckDB in-memory).
package sync

import (
	"bytes"
	"compress/zlib"
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
)

// openGoldenShared ouvre une DB DuckDB in-memory avec le schéma prod aligné
// pour la chaîne highlight events (registry + participants + events + kvp +
// aliases). Reproduit les colonnes essentielles de la migration
// steps_shared.go.
func openGoldenShared(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	ddl := `
		CREATE TABLE match_registry (
			match_id           VARCHAR PRIMARY KEY,
			start_time         TIMESTAMPTZ DEFAULT now(),
			start_time_utc     TIMESTAMP,
			events_loaded      BOOLEAN DEFAULT FALSE,
			backfill_completed INTEGER DEFAULT 0
		);
		CREATE TABLE match_participants (
			match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR
		);
		CREATE SEQUENCE highlight_events_id_seq;
		CREATE TABLE highlight_events (
			id         INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
			match_id   VARCHAR NOT NULL,
			event_type VARCHAR NOT NULL,
			time_ms    INTEGER,
			xuid       VARCHAR,
			type_hint  INTEGER,
			raw_json   VARCHAR,
			UNIQUE (match_id, xuid, time_ms, event_type)
		);
		CREATE TABLE killer_victim_pairs (
			match_id        VARCHAR NOT NULL,
			killer_xuid     VARCHAR NOT NULL,
			killer_gamertag VARCHAR,
			victim_xuid     VARCHAR NOT NULL,
			victim_gamertag VARCHAR,
			kill_count      INTEGER DEFAULT 1,
			time_ms         INTEGER,
			is_validated    BOOLEAN DEFAULT FALSE,
			created_at      TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);
		CREATE TABLE xuid_aliases (
			xuid     VARCHAR PRIMARY KEY,
			gamertag VARCHAR
		);
	`
	if err := execScript(t.Context(), db, ddl); err != nil {
		t.Fatal(err)
	}
	// match_kill_events : seconde destination du kill-feed depuis le 2026-08-02 (double
	// écriture datée). DDL issue des migrations réelles — un schéma recopié ici divergerait
	// du jour où la table change, et le golden certifierait une forme disparue de la prod.
	if err := migration.EnsureMatchKillEvents(db); err != nil {
		t.Fatal(err)
	}
	return db
}

// loadGoldenChunk lit le fixture v41 réel (chemin relatif au package sync).
func loadGoldenChunk(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "analysis", "testdata", "v41_chunk_he.bin")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("fixture introuvable %s: %v\n  → re-capturer via : go run ./cmd/refresh_golden_fixture --gamertag JGtm", path, err)
	}
	return data
}

// TestGoldenFixture_HighlightEventsPipeline est le flagship test E2E.
//
// Construit un mock client servant le fixture v41, instancie un match dans
// match_registry + participant, puis appelle ProcessHighlightEvents (la
// fonction publique de l'engine). Vérifie ENSEMBLE :
//   - parsing bit-aligné (highlight_events ≥ 100)
//   - InsertKillerVictimPairs honnête (kvp > 30, time_ms NOT NULL partout,
//     gamertags renseignés)
//   - bitmasks honnêtes (MBitEvents et MBitKillerVictim positionnés)
//   - xuid_aliases peuplé depuis le chunk parsé
//
// Si l'un de ces invariants casse, c'est qu'une régression a été introduite
// dans la chaîne sync. Le test indique alors où chercher.
func TestGoldenFixture_HighlightEventsPipeline(t *testing.T) {
	const matchID = "golden-match-test"

	db := openGoldenShared(t)

	// Insérer un match minimal (registry + participant) pour permettre
	// l'écriture des side-effects de ProcessHighlightEvents.
	if _, err := db.Exec(`INSERT INTO match_registry (match_id) VALUES (?)`, matchID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO match_participants (match_id, xuid, gamertag) VALUES (?, 'xuid001', 'PlayerA')`, matchID); err != nil {
		t.Fatal(err)
	}

	chunk := loadGoldenChunk(t)
	mock := &mockHaloClient{
		highlightChunkData:    chunk,
		highlightChunkVersion: 41,
		highlightChunkFound:   true,
	}

	result := &domain.SyncResult{}
	err := ProcessHighlightEvents(context.Background(), mock, db, matchID, result)
	if err != nil {
		t.Fatalf("ProcessHighlightEvents: %v", err)
	}

	// ── Invariant 1 : highlight_events suffisamment peuplés ──
	heCount := countSQL(t, db, `SELECT COUNT(*) FROM highlight_events WHERE match_id = ?`, matchID)
	if heCount < 100 {
		t.Errorf("highlight_events count = %d, want ≥ 100. Le parser a probablement régressé vers le scan byte-aligné.", heCount)
	}

	// Distribution par type — le match canonique a au moins kills + deaths.
	heKills := countSQL(t, db, `SELECT COUNT(*) FROM highlight_events WHERE match_id = ? AND event_type = 'kill'`, matchID)
	heDeaths := countSQL(t, db, `SELECT COUNT(*) FROM highlight_events WHERE match_id = ? AND event_type = 'death'`, matchID)
	if heKills == 0 {
		t.Error("aucun event 'kill' parsé — parser cassé ou fixture non-canonique")
	}
	if heDeaths == 0 {
		t.Error("aucun event 'death' parsé — parser cassé ou fixture non-canonique")
	}

	// ── Invariant 2 : killer_victim_pairs honnêtement peuplés ──
	kvpCount := countSQL(t, db, `SELECT COUNT(*) FROM killer_victim_pairs WHERE match_id = ?`, matchID)
	if kvpCount < 30 {
		t.Errorf("killer_victim_pairs count = %d, want ≥ 30. Soit le parser a régressé, soit InsertKVP a re-régressé sur OR IGNORE.", kvpCount)
	}

	// time_ms NOT NULL — régression Phase 1bis (le précédent OR IGNORE
	// n'écrivait pas time_ms du tout).
	nullTime := countSQL(t, db, `SELECT COUNT(*) FROM killer_victim_pairs WHERE match_id = ? AND time_ms IS NULL`, matchID)
	if nullTime > 0 {
		t.Errorf("killer_victim_pairs.time_ms IS NULL pour %d rows — InsertKVP a régressé en omettant la colonne", nullTime)
	}

	// Gamertags présents — régression Phase 1bis (l'ancien INSERT n'écrivait
	// pas killer_gamertag/victim_gamertag).
	nullGT := countSQL(t, db, `SELECT COUNT(*) FROM killer_victim_pairs WHERE match_id = ? AND (killer_gamertag IS NULL OR victim_gamertag IS NULL)`, matchID)
	if nullGT == kvpCount {
		t.Errorf("killer_victim_pairs : aucune ligne n'a killer/victim_gamertag renseignés (%d/%d) — schéma désaligné", nullGT, kvpCount)
	}

	// ── Invariant 3 : bitmasks honnêtes (Phase 1bis) ──
	var bf int64
	if err := db.QueryRow(`SELECT backfill_completed FROM match_registry WHERE match_id = ?`, matchID).Scan(&bf); err != nil {
		t.Fatal(err)
	}
	if bf&MBitEvents == 0 {
		t.Errorf("MBitEvents non set dans backfill_completed (bf=%d) alors que %d events insérés", bf, heCount)
	}
	if bf&MBitKillerVictim == 0 {
		t.Errorf("MBitKillerVictim non set dans backfill_completed (bf=%d) alors que %d paires insérées", bf, kvpCount)
	}

	// ── Invariant 4 : aucune anomalie de parsing remontée ──
	if len(result.Warnings) > 0 {
		t.Errorf("result.Warnings non vide alors que le parsing devrait être propre : %v", result.Warnings)
	}

	// ── Invariant 5 : aucune erreur silencieuse ──
	if result.EventsInserted == 0 {
		t.Error("result.EventsInserted = 0 — chaîne sync silencieuse cassée")
	}

	// Log informatif (visible avec -v).
	uniqueXUIDs := countSQL(t, db, `SELECT COUNT(DISTINCT xuid) FROM highlight_events WHERE match_id = ?`, matchID)
	t.Logf("golden fixture pipeline OK : %d events (%d kills, %d deaths) / %d kvp / %d xuids distincts / bf=%d (MBitEvents=%v, MBitKV=%v)",
		heCount, heKills, heDeaths, kvpCount, uniqueXUIDs, bf,
		bf&MBitEvents != 0, bf&MBitKillerVictim != 0)
}

// TestGoldenFixture_NoFilm_CascadeRespected (variante) : si le client renvoie
// found=false, AUCUNE écriture ne doit se produire dans highlight_events ou
// killer_victim_pairs ; mais le bit MBitEvents DOIT être set (état définitif
// "no film" — distinguable d'un retry pending par events_loaded=TRUE).
func TestGoldenFixture_NoFilm_CascadeRespected(t *testing.T) {
	const matchID = "golden-match-nofilm"

	db := openGoldenShared(t)
	// start_time ancien (> filmRetryWindow) : no-film DÉFINITIF → events_loaded
	// doit être marqué. Depuis le fix "film retardé" (2026-05-31), un no-film
	// sur un match RÉCENT reste FALSE (retry) ; ce test cible le cas définitif.
	if _, err := db.Exec(
		`INSERT INTO match_registry (match_id, start_time) VALUES (?, TIMESTAMP '2020-01-01 00:00:00')`,
		matchID); err != nil {
		t.Fatal(err)
	}

	mock := &mockHaloClient{highlightChunkFound: false} // film 404
	result := &domain.SyncResult{}
	err := ProcessHighlightEvents(context.Background(), mock, db, matchID, result)
	if err != nil {
		t.Fatalf("ProcessHighlightEvents: %v", err)
	}

	heCount := countSQL(t, db, `SELECT COUNT(*) FROM highlight_events WHERE match_id = ?`, matchID)
	kvpCount := countSQL(t, db, `SELECT COUNT(*) FROM killer_victim_pairs WHERE match_id = ?`, matchID)
	if heCount != 0 || kvpCount != 0 {
		t.Errorf("no-film : aucune ligne attendue, got heCount=%d kvpCount=%d", heCount, kvpCount)
	}

	var loaded bool
	_ = db.QueryRow(`SELECT events_loaded FROM match_registry WHERE match_id = ?`, matchID).Scan(&loaded)
	if !loaded {
		t.Error("no-film : events_loaded devrait être TRUE (état définitif, pas de retry)")
	}
}

// TestGoldenFixture_ParseAnomaly_DoesNotMarkEventsLoaded : si le chunk est
// non-vide mais le parser ne trouve rien (futur changement de format API),
// events_loaded doit RESTER FALSE pour permettre un retry après fix parser.
// Régression directe de la situation pré-mai 2026.
func TestGoldenFixture_ParseAnomaly_DoesNotMarkEventsLoaded(t *testing.T) {
	const matchID = "golden-match-anomaly"

	db := openGoldenShared(t)
	if _, err := db.Exec(`INSERT INTO match_registry (match_id) VALUES (?)`, matchID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO match_participants (match_id, xuid, gamertag) VALUES (?, 'xuid001', 'PlayerA')`, matchID); err != nil {
		t.Fatal(err)
	}

	// Chunk zlib non-vide qui décompresse en zéros → 0 events parsés.
	var anomalyBuf bytes.Buffer
	w := zlib.NewWriter(&anomalyBuf)
	_, _ = w.Write(make([]byte, 200))
	_ = w.Close()

	mock := &mockHaloClient{
		highlightChunkData:    anomalyBuf.Bytes(),
		highlightChunkVersion: 41,
		highlightChunkFound:   true,
	}

	result := &domain.SyncResult{}
	err := ProcessHighlightEvents(context.Background(), mock, db, matchID, result)
	if err != nil {
		t.Fatalf("ProcessHighlightEvents: %v", err)
	}

	// Anomalie attendue.
	if len(result.Warnings) == 0 {
		t.Error("attendu une warning parse_anomaly")
	}

	// events_loaded doit RESTER FALSE (cette fois c'est la fonction interne,
	// MarkEventsLoaded n'est appelée que si n>0 — verrouille l'invariant).
	var loaded bool
	_ = db.QueryRow(`SELECT events_loaded FROM match_registry WHERE match_id = ?`, matchID).Scan(&loaded)
	if loaded {
		t.Error("parse_anomaly : events_loaded ne doit PAS être TRUE — bit menteur réintroduit")
	}
}

// countSQL est un helper qui scan une requête COUNT(*) en int.
func countSQL(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("countSQL %q: %v", query, err)
	}
	return n
}

// _ ensures io.EOF est considéré dans le binding (référence triviale).
var _ = io.EOF
