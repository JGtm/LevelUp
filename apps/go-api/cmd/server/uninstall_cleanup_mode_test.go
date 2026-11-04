package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/platform/schedtask"
	"levelup/go-api/internal/platform/startupentry"
)

// cleanupRegistry : un registre de démarrage simulé (une valeur Run) qui note ses suppressions.
type cleanupRegistry struct {
	command string
	present bool
	deleted []string
	readErr error
}

func (r *cleanupRegistry) RunValue(string) (string, bool, error) {
	return r.command, r.present, r.readErr
}
func (r *cleanupRegistry) DeleteRunValue(string) error {
	r.present = false
	r.deleted = append(r.deleted, "run")
	return nil
}
func (r *cleanupRegistry) DeleteApprovedValue(string) error {
	r.deleted = append(r.deleted, "approved")
	return nil
}

// cleanupRunner : un schtasks simulé (liste CSV de /Query) qui note ses commandes.
type cleanupRunner struct {
	listing string
	calls   [][]string
	failOn  string
}

func (r *cleanupRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string(nil), args...))
	if r.failOn != "" && args[0] == r.failOn {
		return nil, errors.New("accès refusé")
	}
	if args[0] == "/Query" {
		return []byte(r.listing), nil
	}
	return nil, nil
}

func newCleanup(t *testing.T, reg *cleanupRegistry, run *cleanupRunner) (uninstallCleanupDeps, *bytes.Buffer) {
	t.Helper()
	data := filepath.Join(t.TempDir(), "donnees")
	out := &bytes.Buffer{}
	return uninstallCleanupDeps{
		Startup:    reg,
		Tasks:      &schedtask.Manager{Runner: run, Spec: schedtask.Spec{DataRoot: data}},
		InstallDir: `C:\Users\Eve\AppData\Local\Programs\LevelUp`,
		DataRoot:   data,
		Out:        out,
	}, out
}

func TestUninstallCleanup_RemovesStartupEntryAndTaskOfThisRoot(t *testing.T) {
	reg := &cleanupRegistry{present: true, command: `"C:\Users\Eve\AppData\Local\Programs\LevelUp\levelup-desktop.exe" --minimized`}
	run := &cleanupRunner{}
	deps, out := newCleanup(t, reg, run)
	run.listing = `"\` + deps.Tasks.TaskName() + `","N/A","Prêt"` + "\n"
	witness := filepath.Join(t.TempDir(), "temoin.txt")
	deps.Witness = witness

	if code := uninstallCleanup(context.Background(), deps); code != 0 {
		t.Fatalf("code = %d, sortie : %s", code, out)
	}
	if strings.Join(reg.deleted, ",") != "run,approved" {
		t.Fatalf("suppressions du registre = %v", reg.deleted)
	}
	last := run.calls[len(run.calls)-1]
	if strings.Join(last, " ") != "/Delete /TN "+deps.Tasks.TaskName()+" /F" {
		t.Fatalf("dernier appel de schtasks = %v", last)
	}
	for _, want := range []string{"demarrage=removed", "tache=retiree-ou-absente", "ok=true"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("bilan sans %q : %s", want, out)
		}
	}
	got, err := os.ReadFile(witness)
	if err != nil || !strings.Contains(string(got), "uninstall-cleanup:") {
		t.Fatalf("fichier témoin : %q, %v", got, err)
	}
}

func TestUninstallCleanup_NothingToRemoveExitsZero(t *testing.T) {
	reg := &cleanupRegistry{}
	run := &cleanupRunner{}
	deps, out := newCleanup(t, reg, run)
	if code := uninstallCleanup(context.Background(), deps); code != 0 {
		t.Fatalf("code = %d, sortie : %s", code, out)
	}
	if len(reg.deleted) != 0 {
		t.Fatalf("suppressions sans entrée : %v", reg.deleted)
	}
	for _, c := range run.calls {
		if c[0] == "/Delete" {
			t.Fatalf("/Delete sans tâche : %v", run.calls)
		}
	}
	if !strings.Contains(out.String(), "demarrage=absent") {
		t.Errorf("bilan : %s", out)
	}
}

func TestUninstallCleanup_ForeignStartupEntryIsKept(t *testing.T) {
	reg := &cleanupRegistry{present: true, command: `"D:\dev\LevelUp\levelup-desktop.exe" --minimized`}
	deps, out := newCleanup(t, reg, &cleanupRunner{})
	if code := uninstallCleanup(context.Background(), deps); code != 0 {
		t.Fatalf("code = %d, sortie : %s", code, out)
	}
	if len(reg.deleted) != 0 || !strings.Contains(out.String(), "demarrage="+string(startupentry.OutcomeForeign)) {
		t.Fatalf("l'entrée d'un autre dossier a été touchée : %v, %s", reg.deleted, out)
	}
}

func TestUninstallCleanup_UnknownDataRootSkipsTheTask(t *testing.T) {
	reg := &cleanupRegistry{present: true, command: `C:\Users\Eve\AppData\Local\Programs\LevelUp\levelup-desktop.exe`}
	deps, out := newCleanup(t, reg, &cleanupRunner{})
	deps.Tasks, deps.DataRoot = nil, ""
	if code := uninstallCleanup(context.Background(), deps); code != 0 {
		t.Fatalf("code = %d, sortie : %s", code, out)
	}
	if !strings.Contains(out.String(), "tache=racine-inconnue") || len(reg.deleted) != 2 {
		t.Fatalf("racine inconnue : bilan %s, suppressions %v", out, reg.deleted)
	}
}

// Un échec d'un côté n'empêche pas l'autre : l'entrée est retirée même si schtasks refuse, et la sortie est 1.
func TestUninstallCleanup_FailureIsReportedAndTheOtherStepStillRuns(t *testing.T) {
	reg := &cleanupRegistry{present: true, command: `"C:\Users\Eve\AppData\Local\Programs\LevelUp\levelup-desktop.exe"`}
	run := &cleanupRunner{failOn: "/Query"}
	deps, out := newCleanup(t, reg, run)
	if code := uninstallCleanup(context.Background(), deps); code != uninstallCleanupFailure {
		t.Fatalf("code = %d, attendu %d", code, uninstallCleanupFailure)
	}
	if len(reg.deleted) != 2 {
		t.Fatalf("l'entrée de démarrage n'a pas été retirée : %v", reg.deleted)
	}
	if !strings.Contains(out.String(), "ok=false") || !strings.Contains(out.String(), "schtasks /Query") {
		t.Errorf("bilan : %s", out)
	}

	reg2 := &cleanupRegistry{readErr: errors.New("registre illisible")}
	run2 := &cleanupRunner{}
	deps2, out2 := newCleanup(t, reg2, run2)
	if code := uninstallCleanup(context.Background(), deps2); code != uninstallCleanupFailure {
		t.Fatalf("code = %d, attendu %d", code, uninstallCleanupFailure)
	}
	if len(run2.calls) == 0 {
		t.Errorf("la tâche n'a pas été tentée malgré l'échec du registre : %s", out2)
	}
}
