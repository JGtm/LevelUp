package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/schedtask"
	"levelup/go-api/internal/platform/startupentry"
	"levelup/go-api/internal/resourcepack"
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

// cleanupRunner : un schtasks simulé (liste CSV de /Query, définition XML de /Query /XML) qui note ses commandes.
type cleanupRunner struct {
	listing string
	xml     string
	calls   [][]string
	failOn  string
}

// taskDefinition : la définition XML d'une tâche de rafraîchissement dont l'action lance le serveur exe.
func taskDefinition(exe string) string {
	return `<Task><Actions><Exec><Command>cmd.exe</Command><Arguments>/d /s /c &quot;set &quot;LEVELUP_DATA_DIR=C:\d&quot; ` +
		`&amp;&amp; &quot;` + exe + `&quot; -mode refresh-only&quot;</Arguments></Exec></Actions></Task>`
}

func (r *cleanupRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string(nil), args...))
	if r.failOn != "" && args[0] == r.failOn {
		return nil, errors.New("accès refusé")
	}
	if args[0] == "/Query" && len(args) > 3 && args[3] == "/XML" {
		return []byte(r.xml), nil
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
	run.xml = taskDefinition(deps.InstallDir + `\levelup-server.exe`)
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
	for _, want := range []string{"demarrage=removed", "tache=removed", "ok=true", "exit=0"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("bilan sans %q : %s", want, out)
		}
	}
	got, err := os.ReadFile(witness)
	if err != nil || !strings.Contains(string(got), "uninstall-cleanup:") || !strings.Contains(string(got), "exit=0") ||
		!strings.Contains(string(got), "tache=removed") {
		t.Fatalf("fichier témoin : %q, %v", got, err)
	}
}

// Revue V4 : deux installations du même utilisateur partagent la racine de données par défaut, donc le nom de la
// tâche ; désinstaller A ne retire pas la tâche dont l'action lance le serveur de B.
func TestUninstallCleanup_TaskOfAnotherInstallationIsKept(t *testing.T) {
	reg := &cleanupRegistry{}
	run := &cleanupRunner{}
	deps, out := newCleanup(t, reg, run)
	run.listing = `"\` + deps.Tasks.TaskName() + `","N/A","Prêt"` + "\n"
	run.xml = taskDefinition(`D:\autre\LevelUp\levelup-server.exe`)
	if code := uninstallCleanup(context.Background(), deps); code != 0 {
		t.Fatalf("code = %d, sortie : %s", code, out)
	}
	for _, c := range run.calls {
		if c[0] == "/Delete" {
			t.Fatalf("la tâche d'une autre installation a été supprimée : %v", run.calls)
		}
	}
	if !strings.Contains(out.String(), "tache=foreign") || !strings.Contains(out.String(), "exit=0") {
		t.Errorf("bilan : %s", out)
	}
}

// Revue V7 : le câblage du mode. La tâche visée est celle de la racine de données de l'environnement (son nom),
// le dossier du programme est celui de l'exécutable, et l'exécuteur et le registre injectés sont les seuls touchés.
func TestNewUninstallCleanupDeps_WiresRootAndExecutable(t *testing.T) {
	programs := filepath.Join(t.TempDir(), "Programs", "LevelUp")
	exe := filepath.Join(programs, "levelup-server.exe")
	roots := title.Roots{
		Resources: programs,
		Data:      filepath.Join(t.TempDir(), "LevelUp"),
		Logs:      filepath.Join(t.TempDir(), "logs"),
		DataSet:   true,
	}
	reg := &cleanupRegistry{present: true, command: `"` + exe + `" --minimized`}
	run := &cleanupRunner{}
	deps := newUninstallCleanupDeps(exe, roots, reg, run)
	out := &bytes.Buffer{}
	deps.Out = out

	want := schedtask.TaskNameFor(roots.Data)
	if deps.InstallDir != programs || deps.DataRoot != roots.Data || deps.Tasks == nil || deps.Tasks.TaskName() != want {
		t.Fatalf("câblage : dossier %q, racine %q, tâche %v (attendu %q)", deps.InstallDir, deps.DataRoot, deps.Tasks, want)
	}
	run.listing = `"\` + want + `","N/A","Prêt"` + "\n"
	run.xml = taskDefinition(exe)
	if code := uninstallCleanup(context.Background(), deps); code != 0 {
		t.Fatalf("code = %d, sortie : %s", code, out)
	}
	last := run.calls[len(run.calls)-1]
	if strings.Join(last, " ") != "/Delete /TN "+want+" /F" || strings.Join(reg.deleted, ",") != "run,approved" {
		t.Fatalf("dernier appel %v, registre %v, sortie : %s", last, reg.deleted, out)
	}

	roots.DataSet = false
	if unset := newUninstallCleanupDeps(exe, roots, reg, run); unset.Tasks != nil || unset.DataRoot != "" || unset.InstallDir != programs {
		t.Fatalf("racine inconnue : la tâche ne doit pas être visée : %+v", unset)
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

// TestUninstallCleanup_RemovesTheResourcePackOfTheInstallDir : le paquet de ressources installé sous le dossier du
// programme est retiré (l'installeur « moteur seul » d'une mise à jour ne le liste pas), les fichiers du moteur
// restent à l'installeur, et le bilan le dit.
func TestUninstallCleanup_RemovesTheResourcePackOfTheInstallDir(t *testing.T) {
	dir := t.TempDir()
	for rel, content := range map[string]string{"static/a.png": "A", "ffmpeg.exe": "F", "web/titles/x/b.png": "B"} {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m, err := resourcepack.Build(dir, "")
	if err != nil || resourcepack.WriteManifest(dir, m) != nil {
		t.Fatalf("manifeste : %v", err)
	}
	engine := filepath.Join(dir, "levelup-server.exe")
	if err := os.WriteFile(engine, []byte("E"), 0o644); err != nil {
		t.Fatal(err)
	}
	deps, out := newCleanup(t, &cleanupRegistry{}, &cleanupRunner{})
	deps.InstallDir = dir
	if code := uninstallCleanup(context.Background(), deps); code != 0 {
		t.Fatalf("code %d : %s", code, out)
	}
	if !strings.Contains(out.String(), "ressources=3-fichiers") {
		t.Fatalf("bilan sans le paquet : %s", out)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "levelup-server.exe" {
		t.Fatalf("seul le moteur doit rester : %v", entries)
	}
}
