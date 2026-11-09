package duckdbbackup

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func heldTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "held.duckdb"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, s := range []string{
		`CREATE TABLE a (id INTEGER)`, `INSERT INTO a VALUES (1),(2)`,
		`CREATE TABLE b (id INTEGER)`, `INSERT INTO b VALUES (3)`,
		`CREATE TABLE c (id INTEGER)`, `INSERT INTO c VALUES (4)`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

// spyRunner : un HeldRunner qui note chaque prise et chaque rendu du bail, et dit s'il est tenu.
type spyRunner struct {
	db     *sql.DB
	events []string
	held   bool
}

func (s *spyRunner) run(_ context.Context, fn func(*sql.DB) error) error {
	s.events = append(s.events, "prise")
	s.held = true
	err := fn(s.db)
	s.held = false
	s.events = append(s.events, "rendu")
	return err
}

// TestExportTargetTablesHeld_LeaseIsTakenAndReleasedPerTable : le bail est pris puis rendu pour le schéma, puis
// pour CHAQUE table : il est libre entre deux tables (S3).
func TestExportTargetTablesHeld_LeaseIsTakenAndReleasedPerTable(t *testing.T) {
	spy := &spyRunner{db: heldTestDB(t)}
	exports, failed, err := ExportTargetTablesHeld(context.Background(), spy.run, HeldExport{Key: "test", OutputDir: t.TempDir(), Level: 3, TableTimeout: time.Minute})
	if err != nil || len(failed) != 0 || len(exports) != 3 {
		t.Fatalf("export : %d table(s), échecs %v, erreur %v", len(exports), failed, err)
	}
	var want []string
	for i := 0; i < 4; i++ { // 1 prise pour le schéma + 3 tables
		want = append(want, "prise", "rendu")
	}
	if len(spy.events) != len(want) {
		t.Fatalf("événements du bail : %v, attendu %d prises et rendus alternés", spy.events, len(want)/2)
	}
	for i := range want {
		if spy.events[i] != want[i] {
			t.Fatalf("événements du bail : %v", spy.events)
		}
	}
	var rows int64
	for _, e := range exports {
		rows += e.Rows
	}
	if rows != 4 {
		t.Errorf("lignes exportées = %d, attendu 4", rows)
	}
}

// TestExportTargetTablesHeld_TableTimeoutMakesItPartial : une table dont l'échéance est dépassée est écartée
// (listée en échec), les autres sont exportées, et le bail n'est pas retenu (S3).
func TestExportTargetTablesHeld_TableTimeoutMakesItPartial(t *testing.T) {
	spy := &spyRunner{db: heldTestDB(t)}
	_, failed, err := ExportTargetTablesHeld(context.Background(), spy.run, HeldExport{Key: "test", OutputDir: t.TempDir(), Level: 3, TableTimeout: time.Nanosecond})
	if err == nil && len(failed) == 0 {
		t.Fatal("une échéance d'une nanoseconde par table doit écarter les tables")
	}
	if spy.held {
		t.Fatal("le bail est resté tenu après l'export")
	}
}

// TestExportTargetTablesHeld_RunnerFailureAbortsTheWholeExport : un échec du bail (base occupée) n'est pas une
// table en échec : il interrompt l'export et se rend tel quel, pour que la base entière soit reprise plus tard.
func TestExportTargetTablesHeld_RunnerFailureAbortsTheWholeExport(t *testing.T) {
	errBusy := errors.New("occupée")
	calls := 0
	run := func(_ context.Context, fn func(*sql.DB) error) error {
		calls++
		if calls == 3 { // schéma et première table passent, la deuxième trouve le bail tenu
			return errBusy
		}
		return fn(heldTestDB(t))
	}
	_, _, err := ExportTargetTablesHeld(context.Background(), run, HeldExport{Key: "test", OutputDir: t.TempDir(), Level: 3, TableTimeout: time.Minute})
	if !errors.Is(err, errBusy) {
		t.Fatalf("erreur %v, attendu l'erreur du bail", err)
	}
}
