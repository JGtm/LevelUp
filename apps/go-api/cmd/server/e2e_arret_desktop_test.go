//go:build e2e_windows && windows

// e2e_arret_desktop_test.go — bout en bout de l'arrêt propre du serveur compilé.
//
// Le test compile cmd/server, le lance comme le fait la coquille desktop (drapeau
// -arret-sur-stdin, stdin tenu par le test) sur une racine de données VIDE, attend /health,
// ferme stdin, puis exige la définition unique d'un arrêt propre : code de sortie 0, ligne de
// journal "arrêt: terminé" avec clean=true et reason=stdin, et aucun fichier *.wal sous la
// racine. Il relance ensuite le même binaire sur la même racine (le second démarrage relit
// les bases laissées par le premier) et rejoue les mêmes vérifications.
//
// Voie : job go-test-windows-cgo de la CI (`-tags=e2e_windows -run TestE2E`). Le coût d'un
// run (compilation exclue du cache chaud) est de l'ordre de la minute : il reste derrière un
// tag pour ne pas alourdir `go test ./cmd/server/`.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	e2eHealthTimeout = 120 * time.Second // démarrage : migrations, ouverture des bases
	e2eExitTimeout   = 60 * time.Second  // arrêt : budget du serveur (40 s) + marge
	e2eShutdownMsg   = "arrêt: terminé"
)

// e2eServer est un serveur lancé par le test.
type e2eServer struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  string // fichier de capture
	stderr  string // fichier de capture
	exited  chan struct{}
	waitErr error // valide une fois exited fermé
	started time.Time
}

// TestE2EArretPropre : démarrage, arrêt par fermeture de stdin, relance sur la même racine.
func TestE2EArretPropre(t *testing.T) {
	bin := e2eBuildServer(t)
	root := e2eNewDataRoot(t)
	port := e2eFreePort(t)

	for run := 1; run <= 2; run++ {
		t.Run(fmt.Sprintf("demarrage_%d", run), func(t *testing.T) {
			srv := e2eStart(t, bin, root, port)
			e2eAwaitHealth(t, srv, port)
			t.Logf("démarrage %d : /health 200 après %s", run, time.Since(srv.started).Round(time.Millisecond))

			stopAt := time.Now()
			if err := srv.stdin.Close(); err != nil {
				t.Fatalf("fermeture de stdin : %v", err)
			}
			e2eAwaitExit(t, srv)
			t.Logf("arrêt %d : sortie après %s", run, time.Since(stopAt).Round(time.Millisecond))

			e2eAssertCleanStop(t, root, run)
		})
		if t.Failed() {
			return
		}
	}
}

// e2eBuildServer compile cmd/server (CGO) dans un dossier temporaire et rend le chemin du binaire.
func e2eBuildServer(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "levelup-server.exe")
	start := time.Now()
	// Mêmes options de symboles que la release (`-s -w`, sans `-X main.version` : la version n'entre pas
	// dans le contrat testé) : le test joue le binaire tel que l'installeur le livre.
	cmd := exec.Command("go", "build", "-ldflags", "-s -w", "-o", bin, ".") // cwd du test = cmd/server
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build ./cmd/server a échoué : %v\n%s", err, out)
	}
	t.Logf("compilation : %s", time.Since(start).Round(time.Millisecond))
	return bin
}

// e2eNewDataRoot prépare une racine de données vide : le dossier config/ du dépôt copié, rien
// d'autre (ni .env.local, ni db_profiles.json, ni data/). Le serveur crée seul data/ et logs/.
func e2eNewDataRoot(t *testing.T) string {
	t.Helper()
	repo := e2eRepoRoot(t)
	root := t.TempDir()
	if err := os.CopyFS(filepath.Join(root, "config"), os.DirFS(filepath.Join(repo, "config"))); err != nil {
		t.Fatalf("copie de config/ : %v", err)
	}
	return root
}

// e2eRepoRoot remonte depuis le répertoire du test jusqu'au dossier qui porte config/locales.toml.
func e2eRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "config", "locales.toml")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("racine du dépôt introuvable (config/locales.toml)")
	return ""
}

func e2eFreePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

// e2eEnv rend l'environnement du serveur : celui du poste SANS aucune variable LEVELUP_* ni
// SPNKR_* (le résultat ne doit pas dépendre du poste), plus les réglages explicites du test.
func e2eEnv(root string, port int) []string {
	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "LEVELUP_") || strings.HasPrefix(kv, "SPNKR_") {
			continue
		}
		env = append(env, kv)
	}
	return append(env,
		"LEVELUP_AUTH_MODE=none",
		"LEVELUP_REPO_ROOT="+root,
		"LEVELUP_API_PORT="+strconv.Itoa(port),
		"LEVELUP_DESKTOP_SECRET=",
	)
}

// e2eStart lance le serveur avec -arret-sur-stdin et un tube de stdin tenu par le test. Un
// serveur encore vivant à la fin du test est tué.
func e2eStart(t *testing.T, bin, root string, port int) *e2eServer {
	t.Helper()
	dir := t.TempDir()
	srv := &e2eServer{
		stdout: filepath.Join(dir, "stdout.txt"),
		stderr: filepath.Join(dir, "stderr.txt"),
		exited: make(chan struct{}),
	}
	outF, err := os.Create(srv.stdout)
	if err != nil {
		t.Fatal(err)
	}
	errF, err := os.Create(srv.stderr)
	if err != nil {
		t.Fatal(err)
	}
	srv.cmd = exec.Command(bin, "-arret-sur-stdin")
	srv.cmd.Dir = root
	srv.cmd.Env = e2eEnv(root, port)
	srv.cmd.Stdout, srv.cmd.Stderr = outF, errF
	if srv.stdin, err = srv.cmd.StdinPipe(); err != nil {
		t.Fatal(err)
	}
	srv.started = time.Now()
	if err := srv.cmd.Start(); err != nil {
		t.Fatalf("lancement du serveur : %v", err)
	}
	go func() {
		srv.waitErr = srv.cmd.Wait()
		_ = outF.Close()
		_ = errF.Close()
		close(srv.exited)
	}()
	t.Cleanup(func() {
		select {
		case <-srv.exited:
		default:
			_ = srv.cmd.Process.Kill()
			<-srv.exited
		}
	})
	return srv
}

// e2eAwaitHealth attend GET /health = 200 ; échoue si le serveur sort avant ou au délai.
func e2eAwaitHealth(t *testing.T, srv *e2eServer, port int) {
	t.Helper()
	url := "http://127.0.0.1:" + strconv.Itoa(port) + "/health"
	client := &http.Client{Timeout: 3 * time.Second}
	deadline := time.After(e2eHealthTimeout)
	for {
		select {
		case <-srv.exited:
			t.Fatalf("le serveur est sorti avant /health 200 (%v)\n%s", srv.waitErr, e2eTail(srv.stderr))
		case <-deadline:
			t.Fatalf("/health n'a pas répondu 200 en %s\n%s", e2eHealthTimeout, e2eTail(srv.stderr))
		case <-time.After(250 * time.Millisecond):
		}
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return
		}
	}
}

// e2eAwaitExit attend la sortie du serveur et exige le code 0.
func e2eAwaitExit(t *testing.T, srv *e2eServer) {
	t.Helper()
	select {
	case <-srv.exited:
	case <-time.After(e2eExitTimeout):
		t.Fatalf("le serveur n'est pas sorti %s après la fermeture de stdin\n%s", e2eExitTimeout, e2eTail(srv.stderr))
	}
	var exitErr *exec.ExitError
	switch {
	case srv.waitErr == nil:
	case errors.As(srv.waitErr, &exitErr):
		t.Fatalf("code de sortie %d, attendu 0\n%s", exitErr.ExitCode(), e2eTail(srv.stderr))
	default:
		t.Fatalf("attente du serveur : %v", srv.waitErr)
	}
}

// e2eAssertCleanStop vérifie, après le run numéro `run` (à partir de 1) : exactement `run`
// lignes "arrêt: terminé" dans les journaux de la racine, la dernière avec clean=true et
// reason=stdin, et aucun *.wal sous la racine hors data/backups/.
func e2eAssertCleanStop(t *testing.T, root string, run int) {
	t.Helper()
	lines := e2eShutdownLines(t, filepath.Join(root, "logs"))
	if len(lines) != run {
		t.Fatalf("%d ligne(s) %q dans %s, attendu %d", len(lines), e2eShutdownMsg, filepath.Join(root, "logs"), run)
	}
	last := lines[len(lines)-1]
	if clean, _ := last["clean"].(bool); !clean {
		t.Errorf("arrêt non propre : %v", last)
	}
	if reason, _ := last["reason"].(string); reason != shutdownReasonStdin {
		t.Errorf("reason = %q, attendu %q (ligne : %v)", reason, shutdownReasonStdin, last)
	}
	backups := filepath.Join(root, "data", "backups")
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && p == backups {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".wal") {
			t.Errorf("fichier WAL résiduel après l'arrêt : %s", p)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de la racine : %v", err)
	}
}

// e2eShutdownLines rend les lignes JSON "arrêt: terminé" de tous les *.log sous dir.
func e2eShutdownLines(t *testing.T, dir string) []map[string]any {
	t.Helper()
	var found []map[string]any
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".log") {
			return err
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			var rec map[string]any
			if json.Unmarshal(sc.Bytes(), &rec) != nil {
				continue
			}
			if rec["msg"] == e2eShutdownMsg {
				found = append(found, rec)
			}
		}
		return sc.Err()
	})
	if err != nil {
		t.Fatalf("lecture des journaux de %s : %v", dir, err)
	}
	return found
}

// e2eTail rend les dernières lignes d'un fichier de capture, pour les messages d'échec.
func e2eTail(path string) string {
	buf, err := os.ReadFile(path)
	if err != nil {
		return "(capture illisible : " + err.Error() + ")"
	}
	lines := strings.Split(strings.TrimRight(string(buf), "\r\n"), "\n")
	if len(lines) > 40 {
		lines = lines[len(lines)-40:]
	}
	return "--- stderr (fin) ---\n" + strings.Join(lines, "\n")
}
