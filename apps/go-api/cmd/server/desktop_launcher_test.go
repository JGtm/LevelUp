//go:build e2e_windows && windows

// desktop_launcher_test.go — le contrat coquille <-> serveur, côté serveur, contre un FAUX LANCEUR
// (ADR 0045 D-21, lot L12.5).
//
// Le test compile cmd/server et le lance comme le fera la coquille : `-mode desktop
// -arret-sur-stdin`, un tube de stdin tenu par le test, trois racines posées par l'environnement
// (ressources avec config/ copié, données et journaux vides), LEVELUP_DESKTOP_SECRET posé, stdout lu
// ligne à ligne. Il exige :
//
//   - la séquence `listening` (port > 0) puis `ready` (même port, et <DataRoot>/desktop/port le
//     contient), /health répondant sur ce port ;
//   - l'entrée de la coquille (lot L13) : GET / sans cookie répond 401, GET /desktop/enter?secret=
//     répond 302 et pose le cookie levelup_session_<port> (une seconde entrée : 403, le secret est à usage
//     unique), puis GET /api/v1/bootstrap avec ce cookie
//     répond 200 (et 401 sans) ;
//   - un second lancement sur les mêmes racines PENDANT le premier : `failed{state:"locked"}` puis
//     sortie non nulle, sans toucher au fichier de port du premier ;
//   - la fermeture de stdin : `stopping` puis `stopped{clean:true}`, sortie 0, aucun *.wal ;
//   - un relancement après l'arrêt : le MÊME port est repris ;
//   - stdout ne porte que des événements, et stderr aucune ligne JSON d'événement.
//
// Voie : job go-test-windows-cgo de la CI (`-tags=e2e_windows -run TestE2E`).
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	launcherListeningTimeout = 90 * time.Second  // jusqu'à `listening` (lancement du processus)
	launcherReadyTimeout     = 180 * time.Second // jusqu'à `ready` (migrations, ouverture des bases)
	launcherEventTimeout     = 60 * time.Second  // autres événements
	launcherSecret           = "secret-de-test-du-faux-lanceur"
)

// launcherRoots : les trois racines que la coquille pose par l'environnement.
type launcherRoots struct{ resources, data, logs string }

func newLauncherRoots(t *testing.T) launcherRoots {
	t.Helper()
	return launcherRoots{resources: e2eNewDataRoot(t), data: t.TempDir(), logs: t.TempDir()}
}

// launcherEnv : l'environnement du poste SANS LEVELUP_* ni SPNKR_*, plus ce que pose la coquille.
func launcherEnv(r launcherRoots) []string {
	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "LEVELUP_") || strings.HasPrefix(kv, "SPNKR_") {
			continue
		}
		env = append(env, kv)
	}
	return append(env,
		"LEVELUP_RESOURCES_DIR="+r.resources,
		"LEVELUP_DATA_DIR="+r.data,
		"LEVELUP_LOGS_DIR="+r.logs,
		"LEVELUP_DESKTOP_SECRET="+launcherSecret,
	)
}

// lockedBuffer : capture de stderr sûre en concurrence.
type lockedBuffer struct {
	mu sync.Mutex
	sb strings.Builder
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sb.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sb.String()
}

// launchedServer : un serveur lancé par le faux lanceur.
type launchedServer struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stderr  *lockedBuffer
	lines   chan string // lignes brutes de stdout, fermé à l'EOF de stdout
	exited  chan struct{}
	waitErr error // valide une fois exited fermé
	started time.Time
	seen    []string // toutes les lignes lues, dans l'ordre
}

// startLauncherServer lance le serveur en -mode desktop avec un stdin tenu. Un serveur encore vivant
// à la fin du test est tué.
func startLauncherServer(t *testing.T, bin string, r launcherRoots) *launchedServer {
	t.Helper()
	s := &launchedServer{stderr: &lockedBuffer{}, lines: make(chan string, 64), exited: make(chan struct{})}
	s.cmd = exec.Command(bin, "-mode", "desktop", "-arret-sur-stdin")
	s.cmd.Dir = r.resources
	s.cmd.Env = launcherEnv(r)
	s.cmd.Stderr = s.stderr
	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if s.stdin, err = s.cmd.StdinPipe(); err != nil {
		t.Fatal(err)
	}
	s.started = time.Now()
	if err := s.cmd.Start(); err != nil {
		t.Fatalf("lancement du serveur : %v", err)
	}
	// La coquille vide stdout en continu (un tube plein bloquerait le serveur) : une goroutine lit
	// tout, ligne à ligne.
	go func() {
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			s.lines <- sc.Text()
		}
		close(s.lines)
		s.waitErr = s.cmd.Wait()
		close(s.exited)
	}()
	t.Cleanup(func() {
		select {
		case <-s.exited:
		default:
			_ = s.cmd.Process.Kill()
			<-s.exited
		}
	})
	return s
}

// next lit la prochaine ligne de stdout, la décode comme événement et rend son nom et ses champs.
func (s *launchedServer) next(t *testing.T, timeout time.Duration) (string, map[string]any) {
	t.Helper()
	select {
	case line, ok := <-s.lines:
		if !ok {
			t.Fatalf("stdout fermé sans l'événement attendu (lignes lues : %q)\n%s", s.seen, e2eTail2(s.stderr))
		}
		s.seen = append(s.seen, line)
		var ev map[string]any
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("stdout porte autre chose qu'un événement JSON : %q (%v)", line, err)
		}
		name, _ := ev["event"].(string)
		if name == "" {
			t.Fatalf("ligne de stdout sans champ event : %q", line)
		}
		return name, ev
	case <-time.After(timeout):
		t.Fatalf("aucun événement en %s (lignes lues : %q)\n%s", timeout, s.seen, e2eTail2(s.stderr))
	}
	return "", nil
}

// expect lit la prochaine ligne, exige l'événement want et rend ses champs.
func (s *launchedServer) expect(t *testing.T, want string, timeout time.Duration) map[string]any {
	t.Helper()
	name, ev := s.next(t, timeout)
	if name != want {
		t.Fatalf("événement %q, attendu %q (lignes lues : %q)\n%s", name, want, s.seen, e2eTail2(s.stderr))
	}
	return ev
}

// portOf lit le champ port d'un événement.
func portOf(t *testing.T, ev map[string]any) int {
	t.Helper()
	f, ok := ev["port"].(float64)
	if !ok || f <= 0 {
		t.Fatalf("port absent ou nul dans %v", ev)
	}
	return int(f)
}

// awaitExit attend la sortie du processus et rend son code.
func (s *launchedServer) awaitExit(t *testing.T) int {
	t.Helper()
	select {
	case <-s.exited:
	case <-time.After(e2eExitTimeout):
		t.Fatalf("le serveur n'est pas sorti en %s\n%s", e2eExitTimeout, e2eTail2(s.stderr))
	}
	var exitErr *exec.ExitError
	switch {
	case s.waitErr == nil:
		return 0
	case errors.As(s.waitErr, &exitErr):
		return exitErr.ExitCode()
	default:
		t.Fatalf("attente du serveur : %v", s.waitErr)
		return -1
	}
}

// e2eTail2 rend la fin de la capture de stderr, pour les messages d'échec.
func e2eTail2(b *lockedBuffer) string {
	lines := strings.Split(strings.TrimRight(b.String(), "\r\n"), "\n")
	if len(lines) > 40 {
		lines = lines[len(lines)-40:]
	}
	return "--- stderr (fin) ---\n" + strings.Join(lines, "\n")
}

// assertStderrSansEvenement : aucune ligne de stderr n'est un événement JSON du contrat (tout
// journal console va sur stderr, mais les événements ne vont QUE sur stdout).
func assertStderrSansEvenement(t *testing.T, b *lockedBuffer) {
	t.Helper()
	for _, line := range strings.Split(b.String(), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var ev map[string]any
		if json.Unmarshal([]byte(line), &ev) == nil {
			if _, isEvent := ev["event"]; isEvent {
				t.Errorf("événement du contrat sur stderr : %q", line)
			}
		}
	}
}

// getHealth interroge /health sur le port lié.
func getHealth(t *testing.T, port int) {
	t.Helper()
	url := "http://127.0.0.1:" + strconv.Itoa(port) + "/health"
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET %s : %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET %s : %d, attendu 200", url, resp.StatusCode)
	}
}

// desktopGet envoie GET path au serveur du port donné, sans suivre les redirections, avec le cookie
// donné (nil : aucun), et rend la réponse (corps fermé).
func desktopGet(t *testing.T, port int, path string, cookie *http.Cookie) *http.Response {
	t.Helper()
	client := &http.Client{
		Timeout:       10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		"http://127.0.0.1:"+strconv.Itoa(port)+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET %s : %v", path, err)
	}
	_ = resp.Body.Close()
	return resp
}

// assertDesktopEntry rejoue le parcours de la coquille sur le vrai binaire : sans cookie 401, entrée
// par secret 302 avec le cookie lié au port, puis une route de l'API répond 200 avec ce cookie.
func assertDesktopEntry(t *testing.T, port int) {
	t.Helper()
	if resp := desktopGet(t, port, "/", nil); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("GET / sans cookie : %d, attendu 401", resp.StatusCode)
	}
	if resp := desktopGet(t, port, "/desktop/enter?secret=pas-le-bon", nil); resp.StatusCode != http.StatusForbidden {
		t.Errorf("GET /desktop/enter avec un faux secret : %d, attendu 403", resp.StatusCode)
	}
	resp := desktopGet(t, port, "/desktop/enter?secret="+launcherSecret, nil)
	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != "/" {
		t.Fatalf("GET /desktop/enter?secret= : %d vers %q, attendu 302 vers /", resp.StatusCode, resp.Header.Get("Location"))
	}
	var cookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "levelup_session_"+strconv.Itoa(port) {
			cookie = c
		}
	}
	if cookie == nil {
		t.Fatalf("aucun cookie levelup_session_%d posé (%v)", port, resp.Cookies())
	}
	// Le secret sert une fois : son URL reste dans l'historique de la WebView, rejouée elle ne donne rien.
	if resp := desktopGet(t, port, "/desktop/enter?secret="+launcherSecret, nil); resp.StatusCode != http.StatusForbidden || len(resp.Cookies()) != 0 {
		t.Errorf("seconde entrée avec le bon secret : %d (%d cookie), attendu 403 sans cookie", resp.StatusCode, len(resp.Cookies()))
	}
	if resp := desktopGet(t, port, "/api/v1/bootstrap", nil); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("GET /api/v1/bootstrap sans cookie : %d, attendu 401", resp.StatusCode)
	}
	if resp := desktopGet(t, port, "/api/v1/bootstrap", cookie); resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/v1/bootstrap avec le cookie : %d, attendu 200", resp.StatusCode)
	}
	if resp := desktopGet(t, port, "/api/v1/desktop/strings", cookie); resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/v1/desktop/strings avec le cookie : %d, attendu 200", resp.StatusCode)
	}
}

// readPortFile lit <DataRoot>/desktop/port.
func readPortFile(t *testing.T, r launcherRoots) int {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(r.data, "desktop", "port"))
	if err != nil {
		t.Fatalf("fichier de port illisible : %v", err)
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatalf("fichier de port invalide %q : %v", raw, err)
	}
	return port
}

// assertNoWAL : aucun *.wal sous la racine de données hors des copies de sécurité de migration.
func assertNoWAL(t *testing.T, r launcherRoots) {
	t.Helper()
	backups := filepath.Join(r.data, "backups")
	err := filepath.WalkDir(r.data, func(p string, d os.DirEntry, err error) error {
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
		t.Fatalf("parcours de la racine de données : %v", err)
	}
}

// TestE2ELanceurDesktop : la séquence complète du contrat, deux instances, un relancement.
func TestE2ELanceurDesktop(t *testing.T) {
	bin := e2eBuildServer(t)
	roots := newLauncherRoots(t)

	// --- Premier lancement : listening -> ready, port écrit, /health. ---
	first := startLauncherServer(t, bin, roots)
	port := portOf(t, first.expect(t, "listening", launcherListeningTimeout))
	t.Logf("listening après %s (port %d)", time.Since(first.started).Round(time.Millisecond), port)
	if ready := first.expect(t, "ready", launcherReadyTimeout); portOf(t, ready) != port {
		t.Errorf("ready sur le port %d, listening annonçait %d", portOf(t, ready), port)
	}
	t.Logf("ready après %s", time.Since(first.started).Round(time.Millisecond))
	if got := readPortFile(t, roots); got != port {
		t.Errorf("fichier de port = %d, attendu le port annoncé %d", got, port)
	}
	getHealth(t, port)
	assertDesktopEntry(t, port)
	if raw, err := os.ReadFile(filepath.Join(roots.data, "auth", "session_secret")); err != nil {
		t.Errorf("secret de session absent après le premier lancement : %v", err)
	} else if secret := strings.TrimSpace(string(raw)); len(secret) != 64 || secret == "CHANGE_ME_IN_PRODUCTION" {
		t.Errorf("secret de session = %d caractères, attendu 64 tirés au hasard", len(secret))
	}

	// --- Second lancement pendant le premier : failed locked, sortie non nulle, port intact. ---
	second := startLauncherServer(t, bin, roots)
	secondPort := portOf(t, second.expect(t, "listening", launcherListeningTimeout))
	if secondPort == port {
		t.Errorf("le second serveur écoute sur le port %d du premier", port)
	}
	failed := second.expect(t, "failed", launcherEventTimeout)
	if failed["state"] != "locked" {
		t.Errorf("failed = %v, attendu state locked", failed)
	}
	if err := second.stdin.Close(); err != nil {
		t.Fatalf("fermeture du stdin du second serveur : %v", err)
	}
	if code := second.awaitExit(t); code == 0 {
		t.Error("le second serveur est sorti en 0 après failed locked, attendu un code non nul")
	}
	assertStderrSansEvenement(t, second.stderr)
	if got := readPortFile(t, roots); got != port {
		t.Errorf("fichier de port = %d après le second lancement, attendu %d intact", got, port)
	}
	getHealth(t, port) // le premier n'a rien perdu

	// --- Arrêt par fermeture de stdin : stopping -> stopped clean, sortie 0, aucun WAL. ---
	stopAt := time.Now()
	if err := first.stdin.Close(); err != nil {
		t.Fatalf("fermeture du stdin : %v", err)
	}
	first.expect(t, "stopping", launcherEventTimeout)
	stopped := first.expect(t, "stopped", launcherEventTimeout)
	if stopped["clean"] != true {
		t.Errorf("stopped = %v, attendu clean true", stopped)
	}
	if code := first.awaitExit(t); code != 0 {
		t.Errorf("code de sortie %d, attendu 0\n%s", code, e2eTail2(first.stderr))
	}
	t.Logf("arrêt : stopped après %s", time.Since(stopAt).Round(time.Millisecond))
	if extra, ok := <-first.lines; ok {
		t.Errorf("ligne de stdout après stopped : %q", extra)
	}
	assertStderrSansEvenement(t, first.stderr)
	assertNoWAL(t, roots)
	if lines := e2eShutdownLines(t, roots.logs); len(lines) != 1 {
		t.Fatalf("%d ligne(s) %q dans %s, attendu 1", len(lines), e2eShutdownMsg, roots.logs)
	} else if clean, _ := lines[0]["clean"].(bool); !clean {
		t.Errorf("ligne d'arrêt non propre : %v", lines[0])
	}

	// --- Relancement après l'arrêt : le MÊME port est repris, le verrou est libre. ---
	again := startLauncherServer(t, bin, roots)
	if got := portOf(t, again.expect(t, "listening", launcherListeningTimeout)); got != port {
		t.Errorf("relancement : port %d, attendu le même port %d", got, port)
	}
	if got := portOf(t, again.expect(t, "ready", launcherReadyTimeout)); got != port {
		t.Errorf("relancement : ready sur le port %d, attendu %d", got, port)
	}
	if err := again.stdin.Close(); err != nil {
		t.Fatalf("fermeture du stdin : %v", err)
	}
	again.expect(t, "stopping", launcherEventTimeout)
	if stopped := again.expect(t, "stopped", launcherEventTimeout); stopped["clean"] != true {
		t.Errorf("relancement : stopped = %v, attendu clean true", stopped)
	}
	if code := again.awaitExit(t); code != 0 {
		t.Errorf("relancement : code de sortie %d, attendu 0", code)
	}
	assertStderrSansEvenement(t, again.stderr)
	assertNoWAL(t, roots)
}
