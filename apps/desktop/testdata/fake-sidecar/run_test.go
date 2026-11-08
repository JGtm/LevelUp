package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

var testSecret = strings.Repeat("s", 32) // valeur de forme, pas un secret

var contractArgs = []string{"-mode", "desktop", "-arret-sur-stdin"}

// syncBuffer : stdout du faux serveur, lu pendant qu'il tourne.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// events rend les lignes JSON de stdout comme evenements, dans l'ordre ; les autres lignes sont comptees.
func (s *syncBuffer) events(t *testing.T) (events []map[string]any, noise int) {
	t.Helper()
	for _, line := range strings.Split(strings.TrimSpace(s.String()), "\n") {
		if line == "" {
			continue
		}
		var ev map[string]any
		if json.Unmarshal([]byte(line), &ev) != nil || ev["event"] == nil {
			noise++
			continue
		}
		events = append(events, ev)
	}
	return events, noise
}

// running : un faux serveur lance dans ce processus.
type running struct {
	stdin  *io.PipeWriter
	stdout *syncBuffer
	stderr *syncBuffer
	code   chan int
}

func launch(t *testing.T, args []string, env map[string]string) *running {
	t.Helper()
	pr, pw := io.Pipe()
	r := &running{stdin: pw, stdout: &syncBuffer{}, stderr: &syncBuffer{}, code: make(chan int, 1)}
	getenv := func(k string) string { return env[k] }
	go func() { r.code <- run(args, getenv, "/ici", pr, r.stdout, r.stderr) }()
	t.Cleanup(func() { _ = pw.Close() })
	return r
}

func baseEnv(t *testing.T, extra map[string]string) map[string]string {
	t.Helper()
	env := map[string]string{
		"LEVELUP_DESKTOP_SECRET": testSecret,
		"LEVELUP_DATA_DIR":       t.TempDir(),
		"LEVELUP_RESOURCES_DIR":  "/res",
		"LEVELUP_LOGS_DIR":       "/logs",
	}
	for k, v := range extra {
		env[k] = v
	}
	return env
}

// waitEvent attend qu'un evenement de ce nom soit sorti et le rend.
func (r *running) waitEvent(t *testing.T, name string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		events, _ := r.stdout.events(t)
		for _, ev := range events {
			if ev["event"] == name {
				return ev
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("evenement %q jamais emis ; stdout=%q stderr=%q", name, r.stdout.String(), r.stderr.String())
	return nil
}

func (r *running) wait(t *testing.T) int {
	t.Helper()
	select {
	case code := <-r.code:
		return code
	case <-time.After(10 * time.Second):
		t.Fatalf("le faux serveur ne s'est pas arrete ; stdout=%q stderr=%q", r.stdout.String(), r.stderr.String())
		return -1
	}
}

func portOf(t *testing.T, ev map[string]any) int {
	t.Helper()
	f, ok := ev["port"].(float64)
	if !ok || f <= 0 {
		t.Fatalf("port absent de %v", ev)
	}
	return int(f)
}

// get envoie GET path sans suivre les redirections, avec le Host donne.
func get(t *testing.T, port int, path, host string) *http.Response {
	t.Helper()
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:"+strconv.Itoa(port)+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if host != "" {
		req.Host = host
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET %s : %v", path, err)
	}
	_ = resp.Body.Close()
	return resp
}

func names(events []map[string]any) []string {
	var out []string
	for _, ev := range events {
		out = append(out, ev["event"].(string))
	}
	return out
}

func TestLancementRefuse(t *testing.T) {
	cases := []struct {
		name string
		args []string
		env  map[string]string
		want int
	}{
		{"secret dans les arguments", append(append([]string{}, contractArgs...), testSecret), nil, exitSecret},
		{"secret absent", contractArgs, map[string]string{"LEVELUP_DESKTOP_SECRET": ""}, exitNoSecret},
		{"argument inattendu", append(append([]string{}, contractArgs...), "-x"), nil, exitSetup},
		{"arguments manquants", []string{"-mode", "desktop"}, nil, exitSetup},
		{"variante inconnue", contractArgs, map[string]string{"FAKE_VARIANT": "rien"}, exitSetup},
		{"delai illisible", contractArgs, map[string]string{"FAKE_READY_DELAY_MS": "vite"}, exitSetup},
		{"arret propre illisible", contractArgs, map[string]string{"FAKE_STOP_CLEAN": "peut-etre"}, exitSetup},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := launch(t, tc.args, baseEnv(t, tc.env))
			if got := r.wait(t); got != tc.want {
				t.Errorf("code %d, attendu %d (stderr=%q)", got, tc.want, r.stderr.String())
			}
			if out := r.stdout.String(); out != "" {
				t.Errorf("stdout non vide apres un lancement refuse : %q", out)
			}
		})
	}
}

func TestSecretJamaisEcrit(t *testing.T) {
	r := launch(t, contractArgs, baseEnv(t, nil))
	r.waitEvent(t, "ready")
	_ = r.stdin.Close()
	r.wait(t)
	if strings.Contains(r.stdout.String()+r.stderr.String(), testSecret) {
		t.Error("le secret apparait sur stdout ou stderr")
	}
}

func TestNominal(t *testing.T) {
	env := baseEnv(t, nil)
	r := launch(t, contractArgs, env)
	port := portOf(t, r.waitEvent(t, "listening"))
	if got := portOf(t, r.waitEvent(t, "ready")); got != port {
		t.Errorf("ready sur %d, listening annoncait %d", got, port)
	}
	host := "127.0.0.1:" + strconv.Itoa(port)

	if resp := get(t, port, "/health", ""); resp.StatusCode != http.StatusOK {
		t.Errorf("/health : %d", resp.StatusCode)
	}
	if resp := get(t, port, "/", ""); resp.StatusCode != http.StatusOK {
		t.Errorf("/ : %d", resp.StatusCode)
	}
	if resp := get(t, port, "/desktop/enter?secret=non", ""); resp.StatusCode != http.StatusForbidden {
		t.Errorf("entree avec un faux secret : %d, attendu 403", resp.StatusCode)
	}
	if resp := get(t, port, "/health", "autre.exemple:"+strconv.Itoa(port)); resp.StatusCode != http.StatusMisdirectedRequest {
		t.Errorf("Host etranger : %d, attendu 421", resp.StatusCode)
	}
	resp := get(t, port, "/desktop/enter?secret="+testSecret, host)
	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != "/" {
		t.Fatalf("entree : %d vers %q, attendu 302 vers /", resp.StatusCode, resp.Header.Get("Location"))
	}
	cookies := resp.Cookies()
	if len(cookies) != 1 || cookies[0].Name != "levelup_session_"+strconv.Itoa(port) || !cookies[0].HttpOnly {
		t.Errorf("cookie = %v", cookies)
	}

	_ = r.stdin.Close()
	if code := r.wait(t); code != exitOK {
		t.Errorf("code de sortie %d, attendu 0", code)
	}
	events, noise := r.stdout.events(t)
	if got := strings.Join(names(events), ","); got != "listening,ready,stopping,stopped" || noise != 0 {
		t.Errorf("evenements %s (bruit %d), attendu listening,ready,stopping,stopped", got, noise)
	}
	if events[3]["clean"] != true {
		t.Errorf("stopped = %v, attendu clean true", events[3])
	}
	if _, err := os.Stat(filepath.Join(env["LEVELUP_DATA_DIR"], stoppedMarker)); err != nil {
		t.Errorf("marqueur d'arret absent : %v", err)
	}
	for _, want := range []string{"cwd=/ici", "resources=/res", "data=" + env["LEVELUP_DATA_DIR"], "logs=/logs"} {
		if !strings.Contains(r.stderr.String(), want) {
			t.Errorf("stderr ne dit pas %q : %q", want, r.stderr.String())
		}
	}
}

func TestPortRepris(t *testing.T) {
	env := baseEnv(t, nil)
	first := launch(t, contractArgs, env)
	port := portOf(t, first.waitEvent(t, "ready"))
	_ = first.stdin.Close()
	first.wait(t)
	raw, err := os.ReadFile(filepath.Join(env["LEVELUP_DATA_DIR"], "desktop", "port"))
	if err != nil || strings.TrimSpace(string(raw)) != strconv.Itoa(port) {
		t.Fatalf("fichier de port = %q (%v), attendu %d", raw, err, port)
	}
	again := launch(t, contractArgs, env)
	if got := portOf(t, again.waitEvent(t, "listening")); got != port {
		t.Errorf("relancement sur %d, attendu le meme port %d", got, port)
	}
	_ = again.stdin.Close()
	again.wait(t)
}

func TestVarianteEchecAvecPort(t *testing.T) {
	r := launch(t, contractArgs, baseEnv(t, map[string]string{"FAKE_VARIANT": variantFailed, "FAKE_FAILED_STATE": "locked"}))
	port := portOf(t, r.waitEvent(t, "listening"))
	if ev := r.waitEvent(t, "failed"); ev["state"] != "locked" {
		t.Errorf("failed = %v", ev)
	}
	if resp := get(t, port, "/", ""); resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("/ apres un echec : %d, attendu 503", resp.StatusCode)
	}
	_ = r.stdin.Close()
	if code := r.wait(t); code != exitFailed {
		t.Errorf("code %d, attendu %d", code, exitFailed)
	}
	if events, _ := r.stdout.events(t); strings.Contains(strings.Join(names(events), ","), "stopped") {
		t.Errorf("un serveur qui refuse de demarrer n'annonce pas stopped : %v", names(events))
	}
}

func TestVarianteEchecSansPort(t *testing.T) {
	r := launch(t, contractArgs, baseEnv(t, map[string]string{"FAKE_VARIANT": variantFailedNoPort, "FAKE_FAILED_STATE": "port"}))
	if ev := r.waitEvent(t, "failed"); ev["state"] != "port" {
		t.Errorf("failed = %v", ev)
	}
	_ = r.stdin.Close()
	r.wait(t)
	if events, _ := r.stdout.events(t); len(events) != 1 {
		t.Errorf("evenements %v, attendu failed seul", names(events))
	}
}

func TestVarianteCrash(t *testing.T) {
	r := launch(t, contractArgs, baseEnv(t, map[string]string{"FAKE_VARIANT": variantCrash}))
	if code := r.wait(t); code != exitCrash {
		t.Errorf("code %d, attendu %d", code, exitCrash)
	}
	if out := r.stdout.String(); out != "" {
		t.Errorf("un crash n'emet aucun evenement, stdout=%q", out)
	}
}

func TestVarianteCrashApresReady(t *testing.T) {
	r := launch(t, contractArgs, baseEnv(t, map[string]string{"FAKE_VARIANT": variantCrashAfterRdy}))
	r.waitEvent(t, "ready")
	if code := r.wait(t); code != exitCrash {
		t.Errorf("code %d, attendu %d", code, exitCrash)
	}
}

func TestVarianteJamaisPret(t *testing.T) {
	r := launch(t, contractArgs, baseEnv(t, map[string]string{"FAKE_VARIANT": variantNeverReady}))
	r.waitEvent(t, "listening")
	time.Sleep(100 * time.Millisecond)
	if events, _ := r.stdout.events(t); len(events) != 1 {
		t.Fatalf("evenements %v, attendu listening seul", names(events))
	}
	_ = r.stdin.Close()
	if code := r.wait(t); code != exitOK {
		t.Errorf("code %d, attendu 0 a l'EOF", code)
	}
}

func TestVarianteArretTardif(t *testing.T) {
	env := baseEnv(t, map[string]string{"FAKE_VARIANT": variantLateStopped, "FAKE_STOP_DELAY_MS": "300"})
	r := launch(t, contractArgs, env)
	r.waitEvent(t, "ready")
	start := time.Now()
	_ = r.stdin.Close()
	r.waitEvent(t, "stopping")
	if _, err := os.Stat(filepath.Join(env["LEVELUP_DATA_DIR"], stoppedMarker)); err == nil {
		t.Error("le marqueur existe avant la fin du delai d'arret")
	}
	r.waitEvent(t, "stopped")
	if elapsed := time.Since(start); elapsed < 250*time.Millisecond {
		t.Errorf("stopped apres %s, attendu au moins le delai d'arret", elapsed)
	}
	r.wait(t)
}

func TestVarianteBavarde(t *testing.T) {
	r := launch(t, contractArgs, baseEnv(t, map[string]string{"FAKE_CHATTY_BYTES": "200000"}))
	r.waitEvent(t, "ready")
	events, noise := r.stdout.events(t)
	if got := strings.Join(names(events), ","); got != "listening,ready" {
		t.Errorf("evenements %s", got)
	}
	if noise < 200000/len(chattyLine) {
		t.Errorf("%d lignes de bavardage, attendu au moins %d", noise, 200000/len(chattyLine))
	}
	_ = r.stdin.Close()
	r.wait(t)
}

func TestArretNonPropre(t *testing.T) {
	env := baseEnv(t, map[string]string{"FAKE_STOP_CLEAN": "false"})
	r := launch(t, contractArgs, env)
	r.waitEvent(t, "ready")
	_ = r.stdin.Close()
	stopped := r.waitEvent(t, "stopped")
	if stopped["clean"] != false {
		t.Errorf("stopped = %v, attendu clean false", stopped)
	}
	if _, err := os.Stat(filepath.Join(env["LEVELUP_DATA_DIR"], stoppedMarker)); err == nil {
		t.Error("le marqueur d'arret propre existe apres un arret non propre")
	}
	if code := r.wait(t); code != exitOK {
		t.Errorf("code %d, attendu %d", code, exitOK)
	}
}

// TestEvenementsApresReady : FAKE_EVENTS_AFTER_READY ecrit ses lignes JSON telles quelles, dans
// l'ordre, juste apres `ready` ; une ligne qui n'est pas du JSON refuse le lancement.
func TestEvenementsApresReady(t *testing.T) {
	lines := `{"event":"notification","title":"t","body":"b","kind":"sync"}|{"event":"shell_settings","close_to_tray":true}`
	r := launch(t, contractArgs, baseEnv(t, map[string]string{"FAKE_EVENTS_AFTER_READY": lines}))
	r.waitEvent(t, "shell_settings")
	events, noise := r.stdout.events(t)
	if got := strings.Join(names(events), ","); got != "listening,ready,notification,shell_settings" || noise != 0 {
		t.Errorf("evenements %s (bruit %d)", got, noise)
	}
	_ = r.stdin.Close()
	r.wait(t)

	if code := run(contractArgs, func(k string) string {
		return map[string]string{
			"LEVELUP_DESKTOP_SECRET": testSecret, "FAKE_EVENTS_AFTER_READY": "pas du json",
		}[k]
	}, "/", strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}); code != exitSetup {
		t.Errorf("code %d, attendu %d pour une ligne qui n'est pas du JSON", code, exitSetup)
	}
}

// TestCommandesDeLaCoquille : une ligne ecrite sur stdin avant la fermeture est une commande, ajoutee au
// fichier des commandes ; elle n'arrete pas le serveur (seul l'EOF le fait).
func TestCommandesDeLaCoquille(t *testing.T) {
	env := baseEnv(t, nil)
	r := launch(t, contractArgs, env)
	r.waitEvent(t, "ready")
	if _, err := io.WriteString(r.stdin, "sync\n"); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(env["LEVELUP_DATA_DIR"], commandsFile)
	deadline := time.Now().Add(10 * time.Second)
	for {
		if raw, err := os.ReadFile(file); err == nil && strings.TrimSpace(string(raw)) == "sync" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("la commande sync n'est jamais arrivee dans %s", file)
		}
		time.Sleep(10 * time.Millisecond)
	}
	select {
	case code := <-r.code:
		t.Fatalf("une commande ne doit pas arreter le serveur (code %d)", code)
	case <-time.After(100 * time.Millisecond):
	}
	_ = r.stdin.Close()
	if code := r.wait(t); code != exitOK {
		t.Errorf("code %d, attendu 0", code)
	}
}

// TestReponseALaCommandeResources : FAKE_RESOURCES_REPLY ecrit ses lignes JSON a chaque commande
// `resources <json>`, jamais pour une autre commande ; une ligne qui n'est pas du JSON refuse le lancement.
func TestReponseALaCommandeResources(t *testing.T) {
	reply := `{"event":"resources","phase":"downloading","fingerprint":"f"}|{"event":"resources","phase":"ready","fingerprint":"f"}`
	r := launch(t, contractArgs, baseEnv(t, map[string]string{"FAKE_RESOURCES_REPLY": reply}))
	r.waitEvent(t, "ready")
	_, _ = io.WriteString(r.stdin, "sync\n")
	_, _ = io.WriteString(r.stdin, `resources {"fingerprint":"f"}`+"\n")
	r.waitEvent(t, "resources")
	deadline := time.Now().Add(10 * time.Second)
	for {
		events, _ := r.stdout.events(t)
		if got := strings.Join(names(events), ","); got == "listening,ready,resources,resources" {
			break
		}
		if time.Now().After(deadline) {
			events, _ := r.stdout.events(t)
			t.Fatalf("evenements %s : deux `resources` attendus, une seule fois (la commande sync n'en provoque aucun)", strings.Join(names(events), ","))
		}
		time.Sleep(5 * time.Millisecond)
	}
	_ = r.stdin.Close()
	r.wait(t)

	if code := run(contractArgs, func(k string) string {
		return map[string]string{"LEVELUP_DESKTOP_SECRET": testSecret, "FAKE_RESOURCES_REPLY": "pas du json"}[k]
	}, "/", strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}); code != exitSetup {
		t.Errorf("code %d, attendu %d pour une ligne qui n'est pas du JSON", code, exitSetup)
	}
}
