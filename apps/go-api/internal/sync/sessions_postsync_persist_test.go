//go:build integration

// Tests pour sessions_postsync_persist.go : delta filter + idempotence
// + INSERT-only path. Reference D.2 du plan de tests.
package sync

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
)

func openSessionTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	// Bootstrap : EnsurePlayerSchema applique le schema complet (PME + autres).
	if err := EnsurePlayerSchema(context.Background(), db); err != nil {
		t.Fatalf("EnsurePlayerSchema: %v", err)
	}
	// Puis migrations player (engagement_score, dominance_flag, etc.).
	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("migrate player: %v", err)
	}
	return db
}

// insertPMERow insere une ligne minimale dans player_match_enrichment avec un
// session_id / session_label donne. Si sessionID == -1, ne set pas les colonnes
// (NULL en DB).
func insertPMERow(t *testing.T, db *sql.DB, matchID string, sessionID int, sessionLabel string) {
	t.Helper()
	if sessionID < 0 {
		_, err := db.Exec("INSERT INTO player_match_enrichment (match_id) VALUES (?)", matchID)
		if err != nil {
			t.Fatalf("insert PME %s: %v", matchID, err)
		}
		return
	}
	_, err := db.Exec(`
		INSERT INTO player_match_enrichment (match_id, session_id, session_label)
		VALUES (?, ?, ?)`, matchID, sessionID, sessionLabel)
	if err != nil {
		t.Fatalf("insert PME %s: %v", matchID, err)
	}
}

func TestWriteSessionAssignmentsBatch_EmptyInput_NoOp(t *testing.T) {
	db := openSessionTestDB(t)
	n, err := writeSessionAssignmentsBatch(context.Background(), db, nil, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if n != 0 {
		t.Errorf("changed = %d, want 0", n)
	}
}

func TestWriteSessionAssignmentsBatch_NewAssignments_AllChanged(t *testing.T) {
	db := openSessionTestDB(t)
	// Pre-insert 3 rows sans session — toutes doivent etre considerees "changed".
	insertPMERow(t, db, "m1", -1, "")
	insertPMERow(t, db, "m2", -1, "")
	insertPMERow(t, db, "m3", -1, "")

	assignments := []domain.SessionAssignment{
		{MatchID: "m1", SessionID: 1, SessionLabel: "Session 1"},
		{MatchID: "m2", SessionID: 1, SessionLabel: "Session 1"},
		{MatchID: "m3", SessionID: 2, SessionLabel: "Session 2"},
	}
	n, err := writeSessionAssignmentsBatch(context.Background(), db, assignments, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if n != 3 {
		t.Errorf("changed = %d, want 3 (toutes nouvelles)", n)
	}
}

func TestWriteSessionAssignmentsBatch_NoChange_Idempotent(t *testing.T) {
	db := openSessionTestDB(t)
	// Pre-insert avec les memes session_id / session_label que ce qu'on va passer.
	insertPMERow(t, db, "m1", 1, "Session 1")
	insertPMERow(t, db, "m2", 1, "Session 1")

	assignments := []domain.SessionAssignment{
		{MatchID: "m1", SessionID: 1, SessionLabel: "Session 1"},
		{MatchID: "m2", SessionID: 1, SessionLabel: "Session 1"},
	}
	n, err := writeSessionAssignmentsBatch(context.Background(), db, assignments, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if n != 0 {
		t.Errorf("changed = %d, want 0 (idempotence : aucun changement)", n)
	}
}

func TestWriteSessionAssignmentsBatch_PartialChange_OnlyDeltasWritten(t *testing.T) {
	db := openSessionTestDB(t)
	// m1 inchange, m2 session_label change, m3 session_id change, m4 nouveau.
	insertPMERow(t, db, "m1", 1, "Session 1")
	insertPMERow(t, db, "m2", 1, "Session 1")
	insertPMERow(t, db, "m3", 1, "Session 1")
	insertPMERow(t, db, "m4", -1, "")

	assignments := []domain.SessionAssignment{
		{MatchID: "m1", SessionID: 1, SessionLabel: "Session 1"}, // pas de changement
		{MatchID: "m2", SessionID: 1, SessionLabel: "Renommee"},  // label change
		{MatchID: "m3", SessionID: 2, SessionLabel: "Session 1"}, // sid change
		{MatchID: "m4", SessionID: 1, SessionLabel: "Session 1"}, // nouveau
	}
	n, err := writeSessionAssignmentsBatch(context.Background(), db, assignments, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if n != 3 {
		t.Errorf("changed = %d, want 3 (m2 label + m3 sid + m4 new ; m1 unchanged)", n)
	}
}

func TestDeltaSessionAssignments_EmptyDB_AllNew(t *testing.T) {
	db := openSessionTestDB(t)
	assignments := []domain.SessionAssignment{
		{MatchID: "m1", SessionID: 1, SessionLabel: "S1"},
		{MatchID: "m2", SessionID: 1, SessionLabel: "S1"},
	}
	changed, err := deltaSessionAssignments(context.Background(), db, assignments, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(changed) != 2 {
		t.Errorf("changed = %d, want 2 (DB vide : toutes nouvelles)", len(changed))
	}
}

func TestWriteSessionAssignmentsBatch_DoubleRunIsIdempotent(t *testing.T) {
	db := openSessionTestDB(t)
	insertPMERow(t, db, "m1", -1, "")

	assignments := []domain.SessionAssignment{
		{MatchID: "m1", SessionID: 1, SessionLabel: "Session 1"},
	}

	// 1er run : ecrit
	n1, _ := writeSessionAssignmentsBatch(context.Background(), db, assignments, false)
	if n1 != 1 {
		t.Errorf("1er run : changed = %d, want 1", n1)
	}

	// 2eme run avec les memes assignments : 0 change
	n2, _ := writeSessionAssignmentsBatch(context.Background(), db, assignments, false)
	if n2 != 0 {
		t.Errorf("2eme run : changed = %d, want 0 (idempotent)", n2)
	}
}

func TestWriteSessionAssignmentsBatch_SubsequentChange(t *testing.T) {
	db := openSessionTestDB(t)
	insertPMERow(t, db, "m1", -1, "")

	// 1er run
	a1 := []domain.SessionAssignment{
		{MatchID: "m1", SessionID: 1, SessionLabel: "Session 1"},
	}
	_, _ = writeSessionAssignmentsBatch(context.Background(), db, a1, false)

	// Verif que la valeur est bien ecrite
	var sid sql.NullInt64
	var label sql.NullString
	if err := db.QueryRow("SELECT session_id, session_label FROM player_match_enrichment WHERE match_id = 'm1'").Scan(&sid, &label); err != nil {
		t.Fatalf("query: %v", err)
	}
	if !sid.Valid || sid.Int64 != 1 {
		t.Errorf("session_id = %v, want 1", sid)
	}
	if !label.Valid || label.String != "Session 1" {
		t.Errorf("session_label = %v, want 'Session 1'", label)
	}

	// 2eme run avec changement
	a2 := []domain.SessionAssignment{
		{MatchID: "m1", SessionID: 2, SessionLabel: "Session 2"},
	}
	n, err := writeSessionAssignmentsBatch(context.Background(), db, a2, false)
	if err != nil {
		t.Fatalf("2eme run err: %v", err)
	}
	if n != 1 {
		t.Errorf("2eme run : changed = %d, want 1 (changement de session)", n)
	}

	// Verif valeur mise a jour
	if err := db.QueryRow("SELECT session_id, session_label FROM player_match_enrichment WHERE match_id = 'm1'").Scan(&sid, &label); err != nil {
		t.Fatalf("query 2: %v", err)
	}
	if !sid.Valid || sid.Int64 != 2 {
		t.Errorf("apres update : session_id = %v, want 2", sid)
	}
	if !label.Valid || label.String != "Session 2" {
		t.Errorf("apres update : session_label = %v, want 'Session 2'", label)
	}
}

// TestWriteSessionAssignmentsBatch_OnlyNewRows_SkipsExisting valide le comportement
// onlyNewRows=true : seules les rows avec session_id IS NULL sont mises à jour.
// Les rows existantes (session_id != NULL) sont ignorées même si le computed
// session_id diffère — évite le bulk-UPDATE → ART bug (crash JGtm 2026-05-26).
func TestWriteSessionAssignmentsBatch_OnlyNewRows_SkipsExisting(t *testing.T) {
	db := openSessionTestDB(t)
	// m1 a déjà un session_id (existant) — ne doit PAS être mis à jour.
	// m2 a session_id NULL (nouveau) — doit être assigné.
	insertPMERow(t, db, "m1", 1, "Session 1")
	insertPMERow(t, db, "m2", -1, "")

	assignments := []domain.SessionAssignment{
		{MatchID: "m1", SessionID: 2, SessionLabel: "Session 2"}, // valeur different, mais existant → skip
		{MatchID: "m2", SessionID: 2, SessionLabel: "Session 2"}, // NULL → doit etre ecrit
	}
	n, err := writeSessionAssignmentsBatch(context.Background(), db, assignments, true)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if n != 1 {
		t.Errorf("changed = %d, want 1 (seul m2 NULL doit etre ecrit)", n)
	}

	// m1 inchangé
	var sid1 int
	if err := db.QueryRow("SELECT session_id FROM player_match_enrichment WHERE match_id = 'm1'").Scan(&sid1); err != nil {
		t.Fatalf("query m1: %v", err)
	}
	if sid1 != 1 {
		t.Errorf("m1 session_id = %d, want 1 (ne doit pas etre modifie)", sid1)
	}

	// m2 assigné
	var sid2 int
	if err := db.QueryRow("SELECT session_id FROM player_match_enrichment WHERE match_id = 'm2'").Scan(&sid2); err != nil {
		t.Fatalf("query m2: %v", err)
	}
	if sid2 != 2 {
		t.Errorf("m2 session_id = %d, want 2", sid2)
	}
}

// keep time import alive even when not directly used (for future extensions).
var _ = time.Now
