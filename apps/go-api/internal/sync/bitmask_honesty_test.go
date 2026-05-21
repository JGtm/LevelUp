// Package sync — bitmask_honesty_test.go : tests garantissant que les bits
// `Mark*Loaded` ne sont pas positionnés quand l'opération sous-jacente a
// échoué. Régression de Phase 1bis du plan
// .ai/PLAN_HIGHLIGHT_EVENTS_BACKFILL.md (mai 2026) :
//
//   - insertHighlightEventsFromData appelait MarkKillerVictimLoaded
//     inconditionnellement après un InsertKillerVictimPairsFromEvents qui
//     pouvait échouer silencieusement (`_ =` swallow).
//   - healEventsForRecentMatches appelait MarkEventsLoaded sur tout retour
//     de ProcessHighlightEvents, y compris parse_anomaly (chunk présent
//     mais 0 events extraits — signe de format API évolué).
//
// Ces tests vérifient les invariants côté écriture : si l'insert/parse
// échoue ou produit une anomalie, le bit correspondant reste 0.
package sync

import (
	"bytes"
	"compress/zlib"
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// openHonestyShared ouvre un shared DB minimal pour exercer les fonctions
// Mark*Loaded et leurs call-sites. Le schéma killer_victim_pairs est
// volontairement BUGGÉ (manque de colonnes par rapport au schéma prod) pour
// faire échouer InsertKillerVictimPairsFromEvents et tester que le bit
// MBitKillerVictim reste 0.
func openHonestyShared(t *testing.T, withBrokenKVP bool) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	// Schéma killer_victim_pairs : version BUGGÉE = colonnes manquantes
	// (l'INSERT à 7 colonnes échouera) pour simuler un schéma désaligné prod
	// du temps où InsertKVP utilisait OR IGNORE.
	kvpDDL := `
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
	if withBrokenKVP {
		// Schéma incompatible : kill_count est obligatoire et notnull, et il
		// manque les autres colonnes. L'INSERT à 7 colonnes (sans kill_count)
		// échouera sur la contrainte NOT NULL.
		kvpDDL = `CREATE TABLE killer_victim_pairs (
			match_id    VARCHAR,
			kill_count  INTEGER NOT NULL
		);`
	}

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
	` + kvpDDL
	if err := execScript(t.Context(), db, ddl); err != nil {
		t.Fatal(err)
	}
	return db
}

// ─── 1.bis.a — InsertKillerVictimPairs failure does NOT mark MBitKillerVictim ─

func TestInsertHighlightEventsFromData_DoesNotMarkKVLoadedOnInsertFailure(t *testing.T) {
	db := openHonestyShared(t, true) // schéma kvp BUGGÉ
	if _, err := db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m1')`); err != nil {
		t.Fatal(err)
	}

	// On forge des events avec un kill ET un death pour que
	// ComputeKillerVictimPairs produise au moins une paire.
	events := []analysis.HighlightEvent{
		{XUID: 2_500_000_000_000_001, Gamertag: "PlayerA", EventType: "kill", TypeHint: 50, TimeMS: 5000},
		{XUID: 2_500_000_000_000_002, Gamertag: "PlayerB", EventType: "death", TypeHint: 20, TimeMS: 5003},
	}

	// Construire un chunk zlib qui ne sera PAS parsé directement (on bypass
	// ParseHighlightEvents en appelant insertHighlightEventsFromData avec
	// data nil ; mais cette fonction skippe si data est vide). Pour exercer
	// le chemin de KVP, on utilise plutôt InsertHighlightEvents +
	// InsertKillerVictimPairsFromEvents directement. Le test ci-dessous
	// cible directement la fonction Insert pour valider que le bit n'est
	// pas mis sur erreur.
	if _, err := InsertHighlightEvents(t.Context(), db, "m1", events); err != nil {
		t.Fatalf("InsertHighlightEvents: %v", err)
	}

	// Bit MBitKillerVictim doit être 0 avant — sanity.
	var bf int64
	_ = db.QueryRow(`SELECT backfill_completed FROM match_registry WHERE match_id = 'm1'`).Scan(&bf)
	if bf&MBitKillerVictim != 0 {
		t.Fatalf("précondition fausse : MBitKillerVictim déjà set avant InsertKVP")
	}

	// Tenter l'insert — doit échouer car le schéma kvp est buggé.
	if err := InsertKillerVictimPairsFromEvents(t.Context(), db, "m1", events); err == nil {
		t.Fatal("InsertKillerVictimPairsFromEvents devrait échouer sur schéma buggé")
	}

	// Le bit MBitKillerVictim doit toujours être à 0.
	_ = db.QueryRow(`SELECT backfill_completed FROM match_registry WHERE match_id = 'm1'`).Scan(&bf)
	if bf&MBitKillerVictim != 0 {
		t.Errorf("MBitKillerVictim positionné alors que l'insert a échoué (bf=%d)", bf)
	}
}

// TestInsertHighlightEventsFromData_OrchestrationDoesNotMarkOnInsertFailure
// teste le call-site COMPLET dans engine.go. On construit un mini chunk zlib
// qui produit 2 events (1 kill + 1 death) puis on vérifie que :
//   - MBitEvents EST set (l'insert highlight_events a réussi)
//   - MBitKillerVictim N'EST PAS set (l'insert kvp a échoué sur schéma buggé)
//   - result.Warnings contient un message "killer_victim_pairs"
func TestInsertHighlightEventsFromData_OrchestrationDoesNotMarkOnInsertFailure(t *testing.T) {
	db := openHonestyShared(t, true) // schéma kvp buggé
	if _, err := db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m1')`); err != nil {
		t.Fatal(err)
	}

	// Construire un chunk zlib réel qui produit 2 events (1 kill + 1 death)
	// sur xuids dans la plage Xbox Live valide. On utilise les helpers du
	// package analysis (test-package, pas accessible ici directement) — à
	// défaut, on construit le chunk binaire à la main.
	chunk := buildMinimalKillDeathChunk(t)

	result := &domain.SyncResult{}
	err := insertHighlightEventsFromData(context.Background(), db, nil, "m1", chunk, 42, result)
	if err != nil {
		t.Fatalf("insertHighlightEventsFromData: %v", err)
	}

	var bf int64
	_ = db.QueryRow(`SELECT backfill_completed FROM match_registry WHERE match_id = 'm1'`).Scan(&bf)

	if bf&MBitEvents == 0 {
		t.Errorf("MBitEvents devrait être set (highlight_events insérés OK), bf=%d", bf)
	}
	if bf&MBitKillerVictim != 0 {
		t.Errorf("MBitKillerVictim ne doit PAS être set (insert kvp échoué), bf=%d", bf)
	}

	// Une warning doit être présente pour signaler l'échec kvp.
	foundKvpWarn := false
	for _, w := range result.Warnings {
		if containsKVPWarning(w) {
			foundKvpWarn = true
			break
		}
	}
	if !foundKvpWarn {
		t.Errorf("aucune warning 'killer_victim_pairs' dans result.Warnings : %v", result.Warnings)
	}
}

// containsKVPWarning vérifie qu'une string contient "killer_victim_pairs".
func containsKVPWarning(s string) bool {
	return len(s) >= 19 && containsSubstr(s, "killer_victim_pairs")
}

func containsSubstr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// buildMinimalKillDeathChunk construit un chunk zlib qui décode (via le parser
// bit-aligné) en 1 kill + 1 death sur 2 XUIDs valides.
func buildMinimalKillDeathChunk(t *testing.T) []byte {
	t.Helper()
	// On reproduit le pattern de buildRawChunk (analysis test) : 2 blocs
	// successifs avec end-marker 0x00002ee0.
	raw := buildEventChunk(2_500_000_000_000_077, "PlayerA", 50 /*kill*/, 5000, false, 0)
	raw2 := buildEventChunk(2_500_000_000_000_078, "PlayerB", 20 /*death*/, 5003, false, 0)
	combined := append(raw, raw2...)
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, _ = w.Write(combined)
	_ = w.Close()
	return buf.Bytes()
}

// buildEventChunk produit le binaire pour 1 event (XUID + marqueur 0x2d 0xc0
// + 60 bytes event + end-marker 0x00002ee0 + padding). Reproduction du
// helper test du package analysis (non-exporté → réimplémenté ici).
func buildEventChunk(xuid uint64, gamertag string, typeHint int, timeMS int, isMedal bool, medal int) []byte {
	var buf bytes.Buffer
	buf.Write(make([]byte, 20)) // pad pour i >= 9

	// XUID 8 bytes LE
	xb := make([]byte, 8)
	for i := 0; i < 8; i++ {
		xb[i] = byte(xuid >> (uint(i) * 8))
	}
	buf.Write(xb)
	buf.WriteByte(0x2d)
	buf.WriteByte(0xc0)
	buf.Write(make([]byte, 30)) // pad

	// 60 bytes event data : gamertag UTF-16LE [0:32] + pad + type_hint @47
	// + time_ms BE [48:52] + pad + is_medal @55 + pad + medal @59.
	ev := make([]byte, 60)
	for i, r := range []rune(gamertag) {
		if i >= 16 {
			break
		}
		ev[i*2] = byte(r)
		ev[i*2+1] = byte(r >> 8)
	}
	ev[47] = byte(typeHint)
	ev[48] = byte(timeMS >> 24)
	ev[49] = byte(timeMS >> 16)
	ev[50] = byte(timeMS >> 8)
	ev[51] = byte(timeMS)
	if isMedal {
		ev[55] = 1
	}
	ev[59] = byte(medal)
	buf.Write(ev)

	// End-marker
	buf.Write([]byte{0x00, 0x00, 0x2e, 0xe0})
	buf.Write(make([]byte, 10)) // tail pad
	return buf.Bytes()
}

// ─── 1.bis.b — heal does NOT mark on parse_anomaly ──────────────────────────

func TestHealEventsForRecentMatches_DoesNotMarkOnParseAnomaly(t *testing.T) {
	db := openHonestyShared(t, false) // schéma kvp valide
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
// PBitTeamMMR que sur les rows où team_mmr IS NOT NULL (pas un mensonge sur
// un participant skipped par l'API skill).
func TestMarkSkillLoaded_FiltersByTeamMMR(t *testing.T) {
	db := openHonestyShared(t, false)
	if _, err := db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m-skill')`); err != nil {
		t.Fatal(err)
	}
	// 3 participants : 2 avec team_mmr, 1 sans.
	_, _ = db.Exec(`INSERT INTO match_participants (match_id, xuid, team_mmr) VALUES ('m-skill', 'a', 1500.0)`)
	_, _ = db.Exec(`INSERT INTO match_participants (match_id, xuid, team_mmr) VALUES ('m-skill', 'b', 1600.0)`)
	_, _ = db.Exec(`INSERT INTO match_participants (match_id, xuid, team_mmr) VALUES ('m-skill', 'c', NULL)`)

	if err := MarkSkillLoaded(t.Context(), db, "m-skill"); err != nil {
		t.Fatalf("MarkSkillLoaded: %v", err)
	}

	// Bits PBit sur les 2 premiers, 0 sur le 3ème.
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

	// Bit BackfillFlags["skill"]=4 sur match_registry
	var bf int64
	_ = db.QueryRow(`SELECT backfill_completed FROM match_registry WHERE match_id = 'm-skill'`).Scan(&bf)
	if bf&backfillFlagSkill == 0 {
		t.Errorf("BackfillFlags[skill] non positionné (bf=%d)", bf)
	}
}

// TestMarkSkillLoaded_Idempotent : ré-exécution ne change rien (|=).
func TestMarkSkillLoaded_Idempotent(t *testing.T) {
	db := openHonestyShared(t, false)
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
	db := openHonestyShared(t, false)
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

// ─── Phase 1bis (existing) — heal honesty ───

func TestHealEventsForRecentMatches_DoesMarkOnNoFilm(t *testing.T) {
	db := openHonestyShared(t, false)
	if _, err := db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m-nofilm')`); err != nil {
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

	// Sur no_film, ProcessHighlightEvents fait MarkEventsLoaded en interne ;
	// le heal n'a plus rien à faire. Vérifier que le bit est bien set.
	var loaded bool
	_ = db.QueryRow(`SELECT events_loaded FROM match_registry WHERE match_id = 'm-nofilm'`).Scan(&loaded)
	if !loaded {
		t.Error("events_loaded devrait être TRUE après no_film")
	}
}
