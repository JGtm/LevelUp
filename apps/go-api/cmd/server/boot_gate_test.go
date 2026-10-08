// boot_gate_test.go — le serveur écoute pendant l'initialisation (boot_gate.go) : réponse
// 503 server_starting avant le branchement du routeur, routeur seul après, page web servie
// pendant l'initialisation, échange sûr sous requêtes concurrentes, arrêt par signal.
package main

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"levelup/go-api/internal/config"
)

// routerStub : routeur de test qui signe ses réponses.
func routerStub() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("router"))
	})
}

func gateGet(h http.Handler, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

// assertStarting vérifie une réponse 503 server_starting portant l'étape want.
func assertStarting(t *testing.T, w *httptest.ResponseRecorder, want bootStep) {
	t.Helper()
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("statut %d, attendu 503", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, attendu application/json", ct)
	}
	if ra := w.Header().Get("Retry-After"); ra != bootRetryAfterSeconds {
		t.Errorf("Retry-After = %q, attendu %q", ra, bootRetryAfterSeconds)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, attendu no-store", cc)
	}
	if xfo := w.Header().Get("X-Frame-Options"); xfo != "DENY" {
		t.Errorf("X-Frame-Options = %q, attendu DENY (en-têtes de sécurité du routeur)", xfo)
	}
	var body struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		Retryable bool   `json:"retryable"`
		Details   struct {
			Step string `json:"step"`
		} `json:"details"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("corps illisible %q : %v", w.Body.String(), err)
	}
	if body.Code != serverStartingCode || !body.Retryable || body.Message == "" || body.Details.Step != string(want) {
		t.Errorf("corps = %+v, attendu code %q, retryable, message, étape %q", body, serverStartingCode, want)
	}
}

func TestBootGate_AvantOuverture_503ServerStartingAvecEtape(t *testing.T) {
	g := newBootGate(&config.AppConfig{})
	for _, path := range []string{"/health", "/healthz", "/api/v1/bootstrap", "/"} {
		assertStarting(t, gateGet(g, path), bootStepMigrations)
	}
	g.setStep(context.Background(), bootStepAccounts)
	assertStarting(t, gateGet(g, "/api/v1/bootstrap"), bootStepAccounts)
	if g.isOpen() {
		t.Error("porte ouverte avant open")
	}
}

func TestBootGate_ApresOuverture_ToutVaAuRouteur(t *testing.T) {
	g := newBootGate(&config.AppConfig{})
	g.setStep(context.Background(), bootStepDatabases)
	g.setStep(context.Background(), bootStepAccounts)
	g.setStep(context.Background(), bootStepServices)
	g.open(context.Background(), routerStub())

	for _, path := range []string{"/health", "/api/v1/bootstrap", "/players/x/home"} {
		if w := gateGet(g, path); w.Code != http.StatusOK || w.Body.String() != "router" {
			t.Errorf("GET %s après ouverture : %d %q, attendu le routeur", path, w.Code, w.Body.String())
		}
	}
	// Une durée par étape traversée, dans l'ordre.
	var steps []string
	for _, d := range g.durations {
		steps = append(steps, d[:strings.IndexByte(d, '=')])
	}
	if got, want := strings.Join(steps, ","), "migrations,databases,accounts,services"; got != want {
		t.Errorf("chronologie des étapes = %s, attendu %s", got, want)
	}
}

func TestBootGate_PageServiePendantLInitialisation(t *testing.T) {
	dist := t.TempDir()
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte(`<div id="root"></div>`), 0o644); err != nil {
		t.Fatal(err)
	}
	g := newBootGate(&config.AppConfig{WebDistDir: dist})

	if w := gateGet(g, "/players/x/home"); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `id="root"`) {
		t.Errorf("route de la page pendant l'initialisation : %d, attendu index.html", w.Code)
	}
	assertStarting(t, gateGet(g, "/api/v1/bootstrap"), bootStepMigrations)
	assertStarting(t, gateGet(g, "/health"), bootStepMigrations)

	g.open(context.Background(), routerStub())
	if w := gateGet(g, "/players/x/home"); w.Body.String() != "router" {
		t.Errorf("après ouverture, la page vient du routeur (%q)", w.Body.String())
	}
}

// TestBootGate_EchangeSousRequetesConcurrentes : pendant l'échange, chaque réponse est
// soit la réponse de démarrage, soit celle du routeur — jamais autre chose ; une fois
// l'échange fait, plus aucune réponse de démarrage.
func TestBootGate_EchangeSousRequetesConcurrentes(t *testing.T) {
	g := newBootGate(&config.AppConfig{})
	const workers, afterOpenPerWorker = 8, 50
	var wg sync.WaitGroup
	var mu sync.Mutex
	var bad []string
	var starting, routed int
	opened := make(chan struct{})
	started := make(chan struct{}, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Chaque travailleur tourne AVANT l'échange puis encore afterOpenPerWorker requêtes
			// parties APRÈS : la fenêtre de l'échange est forcément traversée.
			for after := 0; after < afterOpenPerWorker; {
				afterOpen := false
				select {
				case <-opened:
					afterOpen = true
					after++
				default:
				}
				w := gateGet(g, "/api/v1/bootstrap")
				isRouter := w.Body.String() == "router"
				isStarting := w.Code == http.StatusServiceUnavailable && strings.Contains(w.Body.String(), serverStartingCode)
				mu.Lock()
				switch {
				case isRouter:
					routed++
				case isStarting && !afterOpen: // une requête partie après l'échange va au routeur
					starting++
				default:
					bad = append(bad, w.Body.String())
				}
				mu.Unlock()
				if isStarting && !afterOpen {
					select {
					case started <- struct{}{}:
					default:
					}
				}
			}
		}()
	}
	<-started // au moins une réponse de démarrage servie avant l'échange
	g.open(context.Background(), routerStub())
	close(opened)
	wg.Wait()
	if len(bad) > 0 {
		t.Errorf("%d réponse(s) inattendue(s), ex. %q", len(bad), bad[0])
	}
	if starting == 0 || routed < workers*afterOpenPerWorker {
		t.Errorf("démarrage=%d routeur=%d : l'échange n'a pas été traversé sous charge", starting, routed)
	}
}

// TestBootServer_EcouteReelle : sur un vrai port, 503 server_starting avant l'ouverture,
// routeur après, arrêt sans sortie forcée.
func TestBootServer_EcouteReelle(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	g := newBootGate(&config.AppConfig{})
	srv := &http.Server{Handler: g, ReadHeaderTimeout: 5 * time.Second}
	exits := make(chan int, 1)
	b := newBootServer(srv, g, func(code int) { exits <- code })
	served := make(chan struct{})
	go func() { b.serve(ln); close(served) }()

	url := "http://" + ln.Addr().String() + "/api/v1/bootstrap"
	if code, body := httpGet(t, url); code != http.StatusServiceUnavailable || !strings.Contains(body, serverStartingCode) {
		t.Errorf("avant ouverture : %d %q, attendu 503 server_starting", code, body)
	}
	b.openRouter(context.Background(), routerStub())
	if code, body := httpGet(t, url); code != http.StatusOK || body != "router" {
		t.Errorf("après ouverture : %d %q, attendu le routeur", code, body)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown : %v", err)
	}
	<-served
	select {
	case code := <-exits:
		t.Errorf("sortie forcée (%d) sur un arrêt normal", code)
	default:
	}
}

func httpGet(t *testing.T, url string) (int, string) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s : %v", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, string(body)
}

// startSignalWatch lance watchSignals sur un canal de test ; exits reçoit les sorties forcées.
func startSignalWatch(t *testing.T) (b *bootServer, sigCh chan os.Signal, exits chan int) {
	t.Helper()
	g := newBootGate(&config.AppConfig{})
	exits = make(chan int, 1)
	b = newBootServer(&http.Server{Handler: g, ReadHeaderTimeout: time.Second}, g, func(code int) { exits <- code })
	sigCh = make(chan os.Signal, 2)
	go b.watchSignals(sigCh)
	return b, sigCh, exits
}

func waitClosed(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatalf("%s : canal jamais fermé", what)
	}
}

func TestBootServer_SignalPendantLInitialisation_ArretPropreDiffere(t *testing.T) {
	b, sigCh, exits := startSignalWatch(t)
	sigCh <- os.Interrupt
	waitClosed(t, b.stopRequested(), "arrêt demandé")

	// Fin de l'initialisation : le routeur n'est PAS branché, main enchaîne l'arrêt propre.
	b.openRouter(context.Background(), routerStub())
	if b.gate.isOpen() {
		t.Error("routeur branché malgré l'arrêt demandé pendant l'initialisation")
	}
	assertStarting(t, gateGet(b.gate, "/api/v1/bootstrap"), bootStepMigrations)
	select {
	case code := <-exits:
		t.Errorf("sortie forcée (%d) sur un seul signal", code)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestBootServer_SecondSignalPendantLInitialisation_SortieImmediate(t *testing.T) {
	b, sigCh, exits := startSignalWatch(t)
	sigCh <- os.Interrupt
	waitClosed(t, b.stopRequested(), "arrêt demandé")
	sigCh <- syscall.SIGTERM
	select {
	case code := <-exits:
		if code != bootForcedExitCode {
			t.Errorf("code de sortie %d, attendu %d", code, bootForcedExitCode)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("aucune sortie forcée au second signal pendant l'initialisation")
	}
}

func TestBootServer_SignalApresOuverture_PasDeSortieForcee(t *testing.T) {
	b, sigCh, exits := startSignalWatch(t)
	b.openRouter(context.Background(), routerStub())
	if !b.gate.isOpen() {
		t.Fatal("routeur non branché sans arrêt demandé")
	}
	sigCh <- os.Interrupt
	waitClosed(t, b.stopRequested(), "arrêt demandé")
	sigCh <- syscall.SIGTERM
	select {
	case code := <-exits:
		t.Errorf("sortie forcée (%d) alors que l'arrêt propre est en cours", code)
	case <-time.After(50 * time.Millisecond):
	}
}
