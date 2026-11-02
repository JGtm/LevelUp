package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Variantes (FAKE_VARIANT).
const (
	variantOK            = "ok"
	variantNeverReady    = "never_ready"
	variantFailed        = "failed"
	variantFailedNoPort  = "failed_no_port"
	variantLateStopped   = "late_stopped"
	variantCrash         = "crash"
	variantCrashAfterRdy = "crash_after_ready"
)

// defaultFailedState : etat annonce par les variantes d'echec.
const defaultFailedState = "migration_failed"

// stoppedMarker : fichier ecrit sous la racine de donnees juste avant `stopped`. Un serveur tue
// avant la fin de son arret ne l'ecrit pas : c'est ce qui distingue, cote tests, un arret
// termine d'un kill.
const stoppedMarker = "fake-sidecar-stopped.txt"

// chattyLine : une ligne de bavardage (non JSON) de 100 octets, saut de ligne compris.
var chattyLine = "journal egare qui n'est pas un evenement " + strings.Repeat("x", 58) + "\n"

type options struct {
	variant      string
	listenDelay  time.Duration
	readyDelay   time.Duration
	stopDelay    time.Duration
	crashAfter   time.Duration
	chattyBytes  int
	failedState  string
	dataDir      string
	resourcesDir string
	logsDir      string
	secret       string
}

// readOptions lit l'environnement ; une valeur illisible est une erreur (jamais un defaut muet).
func readOptions(getenv func(string) string) (options, error) {
	o := options{
		variant:      getenv("FAKE_VARIANT"),
		failedState:  getenv("FAKE_FAILED_STATE"),
		dataDir:      getenv("LEVELUP_DATA_DIR"),
		resourcesDir: getenv("LEVELUP_RESOURCES_DIR"),
		logsDir:      getenv("LEVELUP_LOGS_DIR"),
		secret:       getenv("LEVELUP_DESKTOP_SECRET"),
	}
	if o.variant == "" {
		o.variant = variantOK
	}
	if o.failedState == "" {
		o.failedState = defaultFailedState
	}
	switch o.variant {
	case variantOK, variantNeverReady, variantFailed, variantFailedNoPort, variantLateStopped, variantCrash, variantCrashAfterRdy:
	default:
		return o, fmt.Errorf("FAKE_VARIANT %q inconnue", o.variant)
	}
	for name, dst := range map[string]*time.Duration{
		"FAKE_LISTEN_DELAY_MS": &o.listenDelay, "FAKE_READY_DELAY_MS": &o.readyDelay,
		"FAKE_STOP_DELAY_MS": &o.stopDelay, "FAKE_CRASH_AFTER_MS": &o.crashAfter,
	} {
		ms, err := intEnv(getenv, name)
		if err != nil {
			return o, err
		}
		*dst = time.Duration(ms) * time.Millisecond
	}
	var err error
	o.chattyBytes, err = intEnv(getenv, "FAKE_CHATTY_BYTES")
	return o, err
}

func intEnv(getenv func(string) string, name string) (int, error) {
	raw := getenv(name)
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("%s %q n'est pas un entier positif", name, raw)
	}
	return n, nil
}

// emitter ecrit les evenements du contrat : une ligne JSON chacun, d'un seul Write, sous verrou.
type emitter struct {
	mu sync.Mutex
	w  io.Writer
}

func (e *emitter) emit(v any) {
	line, err := json.Marshal(v)
	if err != nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	_, _ = e.w.Write(append(line, '\n')) // une sortie fermee ne bloque pas le faux serveur
}

func (e *emitter) listening(port int) { e.emit(map[string]any{"event": "listening", "port": port}) }
func (e *emitter) ready(port int)     { e.emit(map[string]any{"event": "ready", "port": port}) }
func (e *emitter) failed(state string) {
	e.emit(map[string]any{"event": "failed", "state": state})
}
func (e *emitter) stopping() { e.emit(map[string]any{"event": "stopping"}) }
func (e *emitter) stopped(clean bool) {
	e.emit(map[string]any{"event": "stopped", "clean": clean})
}

// chatter ecrit n octets de lignes non JSON : un tube qui n'est pas vide en continu bloque ce
// processus, comme un serveur reel dont la coquille ne lirait plus la sortie.
func (e *emitter) chatter(n int) {
	for written := 0; written < n; written += len(chattyLine) {
		e.mu.Lock()
		_, _ = e.w.Write([]byte(chattyLine))
		e.mu.Unlock()
	}
}

// run est le serveur entier ; rend le code de sortie du processus.
func run(args []string, getenv func(string) string, cwd string, stdin io.Reader, stdout, stderr io.Writer) int {
	opts, err := readOptions(getenv)
	if err != nil {
		fmt.Fprintf(stderr, "fake: %v\n", err)
		return exitSetup
	}
	if code := checkLaunch(args, opts, stderr); code != exitOK {
		return code
	}
	// Les racines recues et le repertoire courant : la coquille les prouve par le journal relaye.
	// Le secret n'est JAMAIS ecrit.
	fmt.Fprintf(stderr, "fake: boot variant=%s cwd=%s resources=%s data=%s logs=%s\n",
		opts.variant, cwd, opts.resourcesDir, opts.dataDir, opts.logsDir)

	out := &emitter{w: stdout}
	eof := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, stdin) // lit jusqu'a l'EOF : fermeture du tube = ordre d'arret
		close(eof)
	}()

	switch opts.variant {
	case variantCrash:
		time.Sleep(opts.crashAfter)
		fmt.Fprintln(stderr, "fake: crash sans evenement")
		return exitCrash
	case variantFailedNoPort:
		out.failed(opts.failedState)
		<-eof
		return exitFailed
	}

	ln, port, err := bindStable(opts.dataDir, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "fake: %v\n", err)
		out.failed("port")
		<-eof
		return exitBind
	}
	srv := &http.Server{Handler: newMux(port, opts), ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = srv.Serve(ln) }() // Serve rend une erreur a la fermeture : attendue

	time.Sleep(opts.listenDelay)
	out.listening(port)
	out.chatter(opts.chattyBytes)
	time.Sleep(opts.readyDelay)
	return serve(opts, out, port, srv, eof, stderr)
}

// serve joue la fin de vie selon la variante, une fois `listening` emis.
func serve(opts options, out *emitter, port int, srv *http.Server, eof <-chan struct{}, stderr io.Writer) int {
	switch opts.variant {
	case variantNeverReady:
		fmt.Fprintln(stderr, "fake: jamais pret, seul l'EOF arrete")
	case variantFailed:
		out.failed(opts.failedState)
		<-eof
		_ = srv.Close()
		return exitFailed
	case variantCrashAfterRdy:
		out.ready(port)
		time.Sleep(opts.crashAfter)
		fmt.Fprintln(stderr, "fake: crash apres ready")
		return exitCrash
	default:
		out.ready(port)
	}
	<-eof
	return stop(opts, out, srv, stderr)
}

// stop : `stopping`, delai d'arret, marqueur, `stopped{clean:true}`, sortie 0.
func stop(opts options, out *emitter, srv *http.Server, stderr io.Writer) int {
	out.stopping()
	fmt.Fprintln(stderr, "fake: stdin ferme, arret demande")
	time.Sleep(opts.stopDelay)
	_ = srv.Close()
	if opts.dataDir != "" {
		marker := filepath.Join(opts.dataDir, stoppedMarker)
		if err := os.WriteFile(marker, []byte("clean\n"), 0o600); err != nil {
			fmt.Fprintf(stderr, "fake: marqueur d'arret : %v\n", err)
		}
	}
	out.stopped(true)
	return exitOK
}

// checkLaunch verifie la ligne de commande et le secret.
func checkLaunch(args []string, opts options, stderr io.Writer) int {
	if opts.secret == "" {
		fmt.Fprintln(stderr, "fake: LEVELUP_DESKTOP_SECRET absent de l'environnement")
		return exitNoSecret
	}
	for _, a := range args {
		if strings.Contains(a, opts.secret) {
			fmt.Fprintln(stderr, "fake: le secret apparait dans les arguments")
			return exitSecret
		}
	}
	if strings.Join(args, " ") != strings.Join(expectedArgs, " ") {
		fmt.Fprintf(stderr, "fake: arguments %q, attendus %q\n", args, expectedArgs)
		return exitSetup
	}
	return exitOK
}

// bindStable reprend le port de <data>/desktop/port s'il est libre, sinon en choisit un et
// l'ecrit : le comportement du serveur reel (ADR 0045 D-9), sans son code.
func bindStable(dataDir string, stderr io.Writer) (net.Listener, int, error) {
	portFile := ""
	if dataDir != "" {
		portFile = filepath.Join(dataDir, "desktop", "port")
		if raw, err := os.ReadFile(portFile); err == nil {
			if kept, perr := strconv.Atoi(strings.TrimSpace(string(raw))); perr == nil && kept > 0 {
				if ln, lerr := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(kept))); lerr == nil {
					return ln, kept, nil
				}
				fmt.Fprintf(stderr, "fake: port garde %d occupe, un port libre est choisi\n", kept)
			}
		}
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, 0, fmt.Errorf("ecoute : %w", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if portFile != "" {
		if err := os.MkdirAll(filepath.Dir(portFile), 0o755); err == nil {
			err = os.WriteFile(portFile, []byte(strconv.Itoa(port)+"\n"), 0o644)
		}
		if err != nil {
			fmt.Fprintf(stderr, "fake: ecriture du port : %v\n", err)
		}
	}
	return ln, port, nil
}

// newMux sert l'entree de la coquille, la page d'accueil et la sonde de sante. Comme le serveur
// reel : Host strict, secret compare a temps constant, cookie lie au port, 503 non reessayable
// sur `/` pour la variante d'echec.
func newMux(port int, opts options) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/desktop/enter", func(w http.ResponseWriter, r *http.Request) {
		got := r.URL.Query().Get("secret")
		if subtle.ConstantTimeCompare([]byte(got), []byte(opts.secret)) != 1 {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "levelup_session_" + strconv.Itoa(port), Value: "ok", Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
		http.Redirect(w, r, "/", http.StatusFound)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "ok") })
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if opts.variant == variantFailed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"code":"server_start_failed","retryable":false,"details":{"state":%q}}`, opts.failedState)
			return
		}
		_, _ = io.WriteString(w, "ok")
	})
	want := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != want {
			http.Error(w, "misdirected", http.StatusMisdirectedRequest)
			return
		}
		mux.ServeHTTP(w, r)
	})
}
