package adminstate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestActionJournal_RecordRehydrateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.json")
	j := NewActionJournal(NewFileStore(path))

	j.Record(context.Background(), ActionSyncCycle, OutcomeOK, TriggerCron)
	j.Record(context.Background(), ActionDataHealth, OutcomeError, TriggerManual)

	rec, ok := j.Entry(ActionSyncCycle)
	if !ok || rec.Outcome != OutcomeOK || rec.Trigger != TriggerCron {
		t.Fatalf("Entry sync_cycle: ok=%v rec=%+v", ok, rec)
	}
	if rec.LastRunAt.IsZero() {
		t.Fatalf("Entry sync_cycle: last_run_at zéro")
	}

	// Réhydratation dans un nouveau journal (survit au « reboot »).
	j2 := NewActionJournal(NewFileStore(path))
	if err := j2.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	dh, ok := j2.Entry(ActionDataHealth)
	if !ok || dh.Outcome != OutcomeError || dh.Trigger != TriggerManual {
		t.Fatalf("réhydratation data_health: ok=%v rec=%+v", ok, dh)
	}
	if len(j2.Snapshot()) != 2 {
		t.Fatalf("snapshot réhydraté: %d entrées (attendu 2)", len(j2.Snapshot()))
	}
}

func TestActionJournal_LoadCorruptDegrades(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.json")
	if err := os.WriteFile(path, []byte("}{ broken"), 0o644); err != nil {
		t.Fatalf("seed corrompu: %v", err)
	}
	j := NewActionJournal(NewFileStore(path))
	if err := j.Load(context.Background()); err == nil {
		t.Fatalf("Load corrompu: err=nil (attendu erreur pour dégradation loggée par le caller)")
	}
	// Le journal reste utilisable (démarrage à vide) et Record refonctionne.
	j.Record(context.Background(), ActionCatalogRefresh, OutcomeOK, TriggerManual)
	if _, ok := j.Entry(ActionCatalogRefresh); !ok {
		t.Fatalf("Record après Load corrompu: entrée absente")
	}
}

func TestActionJournal_MissingFileIsNotAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "never-written.json")
	j := NewActionJournal(NewFileStore(path))
	if err := j.Load(context.Background()); err != nil {
		t.Fatalf("Load fichier absent: err=%v (attendu nil — premier boot)", err)
	}
	if len(j.Snapshot()) != 0 {
		t.Fatalf("journal absent: snapshot non vide")
	}
}

func TestActionJournal_NilSafe(t *testing.T) {
	var j *ActionJournal
	// Aucune de ces méthodes ne doit paniquer sur un journal nil (feature off).
	j.Record(context.Background(), ActionSyncCycle, OutcomeOK, TriggerManual)
	if _, ok := j.Entry(ActionSyncCycle); ok {
		t.Fatalf("Entry sur journal nil: ok=true")
	}
	if len(j.Snapshot()) != 0 {
		t.Fatalf("Snapshot sur journal nil: non vide")
	}
	if err := j.Load(context.Background()); err != nil {
		t.Fatalf("Load sur journal nil: %v", err)
	}
}

func TestOutcomeMapping(t *testing.T) {
	if Outcome(nil) != OutcomeOK {
		t.Fatalf("Outcome(nil) != ok")
	}
	if Outcome(errors.New("boom")) != OutcomeError {
		t.Fatalf("Outcome(err) != error")
	}
}
