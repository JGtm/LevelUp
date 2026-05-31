//go:build cgo

// Package sync — bitmask_honesty_test.go : tests garantissant que les bits
// `Mark*Loaded` ne sont pas positionnés quand l'opération sous-jacente a
// échoué / produit une anomalie.
//
//   - healEventsForRecentMatches ne doit pas marquer events_loaded sur un
//     parse_anomaly (chunk présent mais 0 events extraits — format API évolué).
//   - MarkSkillLoaded ne marque que les participants dont team_mmr est non-NULL.
//
// NB : les tests historiques sur la complétion killer_victim (échec d'insert →
// bit non marqué) ont migré vers internal/persist/events_completion_persister_test.go
// — la complétion combat est désormais atomique via persist.EventsCompletionPersister
// (les fonctions sync legacy InsertKillerVictimPairsFromEvents / MarkKillerVictimLoaded
// ont été supprimées).
package sync

import (
	"bytes"
	"compress/zlib"
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openHonestyShared ouvre un shared DB minimal pour exercer les fonctions
// Mark*Loaded et leurs call-sites.
func openHonestyShared(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	ddl := `
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
		CREATE TABLE match_registry (
			match_id           VARCHAR PRIMARY KEY,
			start_time         TIMESTAMPTZ DEFAULT now(),
			start_time_utc     TIMESTAMP,
			events_loaded      BOOLEAN DEFAULT FALSE,
			backfill_completed INTEGER DEFAULT 0
		);
		CREATE TABLE match_participants (
			match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR,
			team_mmr DOUBLE,
			backfill_bits INTEGER DEFAULT 0
		);
		CREATE TABLE xuid_aliases (
			xuid VARCHAR PRIMARY KEY,
			gamertag VARCHAR
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
			created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
	if err := execScript(t.Context(), db, ddl); err != nil {
		t.Fatal(err)
	}
	return db
}

// ─── heal does NOT mark on parse_anomaly ────────────────────────────────────

func TestHealEventsForRecentMatches_DoesNotMarkOnParseAnomaly(t *testing.T) {
	db := openHonestyShared(t)
	if _, err := db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m-anomaly')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO match_participants (match_id, xuid) VALUES ('m-anomaly', 'xuid001')`); err != nil {
		t.Fatal(err)
	}

	// Mock client qui renvoie un chunk zlib non-vide mais qui décompresse en
	// zéros (= 0 events parsés = parse_anomaly).
	var benign bytes.Buffer
	w := zlib.NewWriter(&benign)
	_, _ = w.Write(make([]byte, 200))
	_ = w.Close()

	mock := &mockHaloClient{
		highlightChunkData:    benign.Bytes(),
		highlightChunkVersion: 41,
		highlightChunkFound:   true,
	}

	healed, noFilm, err := healEventsForRecentMatches(context.Background(), db, nil, mock, 10)
	if err != nil {
		t.Fatalf("healEventsForRecentMatches: %v", err)
	}
	if healed != 0 {
		t.Errorf("healed = %d, want 0 sur parse_anomaly", healed)
	}
	if noFilm != 0 {
		t.Errorf("noFilm = %d, want 0 (anomaly ne doit pas être compté comme no_film)", noFilm)
	}

	// L'invariant clé : events_loaded doit RESTER FALSE après un anomaly.
	var loaded bool
	if err := db.QueryRow(`SELECT events_loaded FROM match_registry WHERE match_id = 'm-anomaly'`).Scan(&loaded); err != nil {
		t.Fatal(err)
	}
	if loaded {
		t.Error("events_loaded est TRUE après parse_anomaly — bit menteur réintroduit")
	}
}

// ─── Phase 2 PLAN_BITMASKS_AUDIT_FIX — MarkSkillLoaded / MarkParticipantsDone ─

// TestMarkSkillLoaded_FiltersByTeamMMR : MarkSkillLoaded ne positionne
// PBitTeamMMR que sur les rows où team_mmr IS NOT NULL.
func TestMarkSkillLoaded_FiltersByTeamMMR(t *testing.T) {
	db := openHonestyShared(t)
	if _, err := db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m-skill')`); err != nil {
		t.Fatal(err)
	}
	_, _ = db.Exec(`INSERT INTO match_participants (match_id, xuid, team_mmr) VALUES ('m-skill', 'a', 1500.0)`)
	_, _ = db.Exec(`INSERT INTO match_participants (match_id, xuid, team_mmr) VALUES ('m-skill', 'b', 1600.0)`)
	_, _ = db.Exec(`INSERT INTO match_participants (match_id, xuid, team_mmr) VALUES ('m-skill', 'c', NULL)`)

	if err := MarkSkillLoaded(t.Context(), db, "m-skill"); err != nil {
		t.Fatalf("MarkSkillLoaded: %v", err)
	}

	rows, err := db.Query(`SELECT xuid, COALESCE(backfill_bits, 0) FROM match_participants WHERE match_id = 'm-skill' ORDER BY xuid`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	results := map[string]int64{}
	for rows.Next() {
		var xuid string
		var bits int64
		_ = rows.Scan(&xuid, &bits)
		results[xuid] = bits
	}
	expected := skillBitsCombined
	if results["a"] != int64(expected) {
		t.Errorf("a : got bits=%d want %d", results["a"], expected)
	}
	if results["b"] != int64(expected) {
		t.Errorf("b : got bits=%d want %d", results["b"], expected)
	}
	if results["c"] != 0 {
		t.Errorf("c (team_mmr NULL) : devrait rester à 0, got %d", results["c"])
	}

	var bf int64
	_ = db.QueryRow(`SELECT backfill_completed FROM match_registry WHERE match_id = 'm-skill'`).Scan(&bf)
	if bf&backfillFlagSkill == 0 {
		t.Errorf("BackfillFlags[skill] non positionné (bf=%d)", bf)
	}
}

// TestMarkSkillLoaded_Idempotent : ré-exécution ne change rien (|=).
func TestMarkSkillLoaded_Idempotent(t *testing.T) {
	db := openHonestyShared(t)
	if _, err := db.Exec(`INSERT INTO match_registry (match_id, backfill_completed) VALUES ('m-idemp', 4)`); err != nil {
		t.Fatal(err)
	}

	if err := MarkSkillLoaded(t.Context(), db, "m-idemp"); err != nil {
		t.Fatalf("MarkSkillLoaded: %v", err)
	}
	var bf int64
	_ = db.QueryRow(`SELECT backfill_completed FROM match_registry WHERE match_id = 'm-idemp'`).Scan(&bf)
	if bf&backfillFlagSkill == 0 {
		t.Error("BackfillFlags[skill] perdu après MarkSkillLoaded idempotent")
	}
}

// TestMarkParticipantsDone_SetsBit : positionnement standard du bit 1<<9.
func TestMarkParticipantsDone_SetsBit(t *testing.T) {
	db := openHonestyShared(t)
	if _, err := db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m-parts')`); err != nil {
		t.Fatal(err)
	}
	if err := MarkParticipantsDone(t.Context(), db, "m-parts"); err != nil {
		t.Fatalf("MarkParticipantsDone: %v", err)
	}
	var bf int64
	_ = db.QueryRow(`SELECT backfill_completed FROM match_registry WHERE match_id = 'm-parts'`).Scan(&bf)
	if bf&backfillFlagParticipants == 0 {
		t.Errorf("BackfillFlags[participants] non positionné (bf=%d)", bf)
	}
}

// ─── heal honesty : no-film définitif (match ancien) → events_loaded marqué ──

func TestHealEventsForRecentMatches_DoesMarkOnNoFilm(t *testing.T) {
	db := openHonestyShared(t)
	// start_time ancien (> filmRetryWindow) : le film est jugé définitivement
	// absent → events_loaded marqué (sort du retry set). Cf. isNoFilmDefinitive
	// + politique film-retardé (golden_test applique le même start_time ancien).
	if _, err := db.Exec(`INSERT INTO match_registry (match_id, start_time) VALUES ('m-nofilm', TIMESTAMP '2020-01-01 00:00:00')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO match_participants (match_id, xuid) VALUES ('m-nofilm', 'xuid001')`); err != nil {
		t.Fatal(err)
	}

	mock := &mockHaloClient{
		highlightChunkFound: false, // film absent
	}

	healed, noFilm, err := healEventsForRecentMatches(context.Background(), db, nil, mock, 10)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if healed != 0 {
		t.Errorf("healed = %d, want 0", healed)
	}
	if noFilm != 1 {
		t.Errorf("noFilm = %d, want 1", noFilm)
	}

	// no_film définitif → ProcessHighlightEvents marque events_loaded en interne.
	var loaded bool
	_ = db.QueryRow(`SELECT events_loaded FROM match_registry WHERE match_id = 'm-nofilm'`).Scan(&loaded)
	if !loaded {
		t.Error("events_loaded devrait être TRUE après no_film définitif")
	}
}
