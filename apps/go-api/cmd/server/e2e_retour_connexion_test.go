//go:build e2e_windows && windows

// e2e_retour_connexion_test.go — bout en bout du retour de la connexion par le navigateur sur le
// binaire compilé, en -mode desktop (ADR 0045 D-22).
//
// Le retour de Microsoft arrive dans le navigateur par défaut, sans cookie de la coquille, avec un
// Host « localhost:<port> ». Deux contrôles de Host se succèdent devant le routeur : la porte de
// démarrage (StrictHost, qui reste devant le routeur une fois ouvert) et le garde du routeur. Le
// 2026-11-11, la porte refusait ce retour en 421 « desktop_host_refused » avant le garde : la
// connexion par le navigateur n'aboutissait jamais sur l'application installée. Ce test lance le
// serveur comme la coquille, attend `ready`, puis exige que GET /auth/xbox/callback avec le Host
// « localhost » atteigne le handler (jamais 421 ni 401), et que toute autre route au même Host reste
// refusée (421).
//
// Voie : job go-test-windows-cgo de la CI (`-tags=e2e_windows -run TestE2E`).
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// e2eStartDesktop lance le serveur en -mode desktop -arret-sur-stdin avec les quatre variables du
// contrat coquille <-> serveur (racines et secret), sur une racine de données vide.
func e2eStartDesktop(t *testing.T, bin, root string) *e2eServer {
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
	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "LEVELUP_") || strings.HasPrefix(kv, "SPNKR_") {
			continue
		}
		env = append(env, kv)
	}
	// Un binaire installé (LEVELUP_E2E_SERVER_BIN) lit config/ et son paquet de ressources dans son propre dossier.
	resources := root
	if os.Getenv("LEVELUP_E2E_SERVER_BIN") != "" {
		resources = filepath.Dir(bin)
	}
	env = append(env,
		"LEVELUP_RESOURCES_DIR="+resources,
		"LEVELUP_DATA_DIR="+filepath.Join(root, "data"),
		"LEVELUP_LOGS_DIR="+filepath.Join(root, "logs"),
		"LEVELUP_DESKTOP_SECRET=e2e-secret-du-lanceur",
	)
	srv.cmd = exec.Command(bin, "-mode", "desktop", "-arret-sur-stdin")
	srv.cmd.Dir = root
	srv.cmd.Env = env
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

// e2eAwaitEvent attend sur la sortie standard l'événement `event` et rend son port.
func e2eAwaitEvent(t *testing.T, srv *e2eServer, event string) int {
	t.Helper()
	deadline := time.After(e2eHealthTimeout)
	for {
		select {
		case <-srv.exited:
			t.Fatalf("le serveur est sorti avant l'événement %q (%v)\n%s", event, srv.waitErr, e2eTail(srv.stderr))
		case <-deadline:
			t.Fatalf("événement %q absent après %s\n%s", event, e2eHealthTimeout, e2eTail(srv.stderr))
		case <-time.After(250 * time.Millisecond):
		}
		f, err := os.Open(srv.stdout)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			var ev struct {
				Event string `json:"event"`
				Port  int    `json:"port"`
			}
			if json.Unmarshal(sc.Bytes(), &ev) == nil && ev.Event == event {
				_ = f.Close()
				return ev.Port
			}
		}
		_ = f.Close()
	}
}

// e2eGetHost fait GET path sur le port avec l'en-tête Host donné et rend le statut et le corps.
func e2eGetHost(t *testing.T, port int, path, host string) (int, string) {
	t.Helper()
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:"+strconv.Itoa(port)+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = host
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET %s (Host %s) : %v", path, host, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

// e2eServerBinary rend le binaire à éprouver : celui que LEVELUP_E2E_SERVER_BIN désigne (témoin « avant » sur un
// binaire installé), sinon cmd/server compilé.
func e2eServerBinary(t *testing.T) string {
	t.Helper()
	if bin := os.Getenv("LEVELUP_E2E_SERVER_BIN"); bin != "" {
		return bin
	}
	return e2eBuildServer(t)
}

// TestE2ERetourConnexionNavigateur : en -mode desktop, routeur ouvert, le retour de connexion au Host
// « localhost » passe les deux contrôles de Host ; les autres routes au même Host restent refusées.
func TestE2ERetourConnexionNavigateur(t *testing.T) {
	bin := e2eServerBinary(t)
	root := e2eNewDataRoot(t)
	srv := e2eStartDesktop(t, bin, root)
	port := e2eAwaitEvent(t, srv, "listening")
	if got := e2eAwaitEvent(t, srv, "ready"); got != port {
		t.Fatalf("port de ready %d != port de listening %d", got, port)
	}
	loopbackName := "localhost:" + strconv.Itoa(port)

	code, body := e2eGetHost(t, port, "/auth/xbox/callback?code=c&state=s", loopbackName)
	if code == http.StatusMisdirectedRequest || code == http.StatusUnauthorized || strings.Contains(body, "desktop_host_refused") {
		t.Errorf("retour de connexion, Host %q : %d %q — refusé avant le handler", loopbackName, code, e2eHead(body))
	}
	for _, path := range []string{"/", "/api/v1/bootstrap", "/health"} {
		if code, body := e2eGetHost(t, port, path, loopbackName); code != http.StatusMisdirectedRequest || !strings.Contains(body, "desktop_host_refused") {
			t.Errorf("%s, Host %q : %d %q, attendu 421 desktop_host_refused", path, loopbackName, code, e2eHead(body))
		}
	}
	if code, _ := e2eGetHost(t, port, "/health", "127.0.0.1:"+strconv.Itoa(port)); code != http.StatusOK {
		t.Errorf("/health au Host exact : %d, attendu 200", code)
	}

	if err := srv.stdin.Close(); err != nil {
		t.Fatal(err)
	}
	e2eAwaitExit(t, srv)
}

// e2eHead rend le début du corps pour un message d'échec lisible.
func e2eHead(body string) string {
	if len(body) > 160 {
		return body[:160] + "…"
	}
	return body
}
