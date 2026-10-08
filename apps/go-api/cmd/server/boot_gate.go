// boot_gate.go — le serveur écoute dès le démarrage, avant d'ouvrir ses bases.
//
// Le port s'ouvre juste après la configuration et les journaux. Tant que l'initialisation
// n'est pas finie, le serveur sert la page web (quand il la sert : LEVELUP_WEB_DIST) et
// répond au reste — API, sondes de santé, routes serveur — un 503 JSON
// {"code":"server_starting","retryable":true,"details":{"step":"<étape>"}} ; la page attend
// donc le serveur au lieu d'échouer. Le routeur est branché d'un seul échange atomique
// (bootServer.openRouter), exactement là où le port s'ouvrait auparavant : il ne reçoit
// aucune requête avant d'être entièrement câblé.
//
// Arrêt : le premier signal (Ctrl+C, SIGTERM) pendant l'initialisation est mémorisé — main
// finit l'initialisation puis enchaîne l'arrêt propre habituel (bases fermées) sans brancher
// le routeur ; un second signal pendant l'initialisation force la sortie immédiate.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"levelup/go-api/internal/api"
	"levelup/go-api/internal/config"
)

// bootStep : étape d'initialisation en cours. Code stable, en anglais, exposé tel quel aux
// clients (la page web en tire son libellé ; un code inconnu n'y affiche rien).
type bootStep string

const (
	// bootStepMigrations : outillage média, registre des titres, migrations des bases
	// partagées et des bases joueur.
	bootStepMigrations bootStep = "migrations"
	// bootStepDatabases : ouverture des bases, garde ART, contrôle des types, réglages.
	bootStepDatabases bootStep = "databases"
	// bootStepAccounts : connexion des comptes Xbox (pool de tokens).
	bootStepAccounts bootStep = "accounts"
	// bootStepServices : planificateurs, suivi en direct, routeur et tâches de fond.
	bootStepServices bootStep = "services"
)

// serverStartingCode : code d'erreur de la réponse servie pendant l'initialisation.
const serverStartingCode = "server_starting"

// bootRetryAfterSeconds : délai de réinterrogation suggéré (en-tête Retry-After) pendant
// l'initialisation.
const bootRetryAfterSeconds = "1"

// bootForcedExitCode : code de sortie d'un arrêt forcé (second signal pendant
// l'initialisation).
const bootForcedExitCode = 1

// startingBody : corps de la réponse 503 servie pendant l'initialisation — l'enveloppe
// d'erreur de l'API ({code, message, retryable, details}), l'étape dans details.
type startingBody struct {
	Code      string          `json:"code"`
	Message   string          `json:"message"`
	Retryable bool            `json:"retryable"`
	Details   startingDetails `json:"details"`
}

type startingDetails struct {
	Step bootStep `json:"step"`
}

// routerRef enveloppe le routeur pour l'échange atomique (atomic.Pointer exige un type).
type routerRef struct{ h http.Handler }

// bootGate : handler HTTP du serveur dès l'ouverture du port. Avant open, il sert la page
// web et répond 503 server_starting au reste (cf. api.NewBootPageHandler) ; après open,
// tout va au routeur.
type bootGate struct {
	router  atomic.Pointer[routerRef]
	step    atomic.Pointer[bootStep]
	booting http.Handler

	// Chronologie des étapes, lue et écrite par la seule goroutine d'initialisation (main).
	started     time.Time
	stepStarted time.Time
	durations   []string
}

// newBootGate crée la porte sur l'étape bootStepMigrations. cfg.WebDistDir : build de la page
// servi par le serveur ("" : la page n'est pas servie par le serveur, en dev Vite la sert).
func newBootGate(cfg *config.AppConfig) *bootGate {
	now := time.Now()
	g := &bootGate{started: now, stepStarted: now}
	first := bootStepMigrations
	g.step.Store(&first)
	g.booting = api.NewBootPageHandler(cfg, http.HandlerFunc(g.writeStarting))
	return g
}

// ServeHTTP route vers le routeur une fois ouvert, sinon vers la réponse de démarrage.
func (g *bootGate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if ref := g.router.Load(); ref != nil {
		ref.h.ServeHTTP(w, r)
		return
	}
	g.booting.ServeHTTP(w, r)
}

// currentStep rend l'étape en cours.
func (g *bootGate) currentStep() bootStep { return *g.step.Load() }

// isOpen indique si le routeur est branché.
func (g *bootGate) isOpen() bool { return g.router.Load() != nil }

// setStep passe à l'étape next et note la durée de la précédente. Appelée par main seul.
func (g *bootGate) setStep(ctx context.Context, next bootStep) {
	prev := g.closeStep()
	g.step.Store(&next)
	slog.DebugContext(ctx, "démarrage : étape", "step", next, "previous", prev)
}

// closeStep clôt l'étape en cours (durée notée) et la rend.
func (g *bootGate) closeStep() bootStep {
	prev := g.currentStep()
	now := time.Now()
	g.durations = append(g.durations, fmt.Sprintf("%s=%dms", prev, now.Sub(g.stepStarted).Milliseconds()))
	g.stepStarted = now
	return prev
}

// open branche le routeur : toutes les requêtes suivantes vont à router. Appelée par main
// seul, une fois.
func (g *bootGate) open(ctx context.Context, router http.Handler) {
	g.closeStep()
	g.router.Store(&routerRef{h: router})
	slog.InfoContext(ctx, "démarrage : serveur prêt",
		"boot_ms", time.Since(g.started).Milliseconds(),
		"steps", strings.Join(g.durations, " "))
}

// writeStarting écrit la réponse 503 server_starting (étape en cours dans details.step).
func (g *bootGate) writeStarting(w http.ResponseWriter, r *http.Request) {
	body, err := json.Marshal(startingBody{
		Code:      serverStartingCode,
		Message:   "server starting",
		Retryable: true,
		Details:   startingDetails{Step: g.currentStep()},
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "démarrage : encodage de la réponse server_starting échoué", "err", err)
		http.Error(w, serverStartingCode, http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Retry-After", bootRetryAfterSeconds)
	w.WriteHeader(http.StatusServiceUnavailable)
	if _, err := w.Write(body); err != nil {
		// Client parti pendant l'écriture : rien à rattraper, la page réinterroge.
		slog.DebugContext(r.Context(), "démarrage : écriture de la réponse server_starting interrompue", "err", err)
	}
}

// bootServer : serveur HTTP ouvert pendant l'initialisation, et l'arrêt demandé par signal.
type bootServer struct {
	srv  *http.Server
	gate *bootGate

	// stop est fermé au premier signal d'arrêt (cf. stopRequested).
	stop chan struct{}
	// initDone passe à vrai quand main a fini l'initialisation (openRouter) : un second
	// signal ne force plus la sortie, l'arrêt propre est en cours.
	initDone atomic.Bool
	// forceExit termine le processus (os.Exit ; remplacé par les tests).
	forceExit func(code int)
}

func newBootServer(srv *http.Server, gate *bootGate, forceExit func(code int)) *bootServer {
	return &bootServer{srv: srv, gate: gate, stop: make(chan struct{}), forceExit: forceExit}
}

// startBootServer ouvre le port de srv (son handler doit être gate), le sert en arrière-plan
// et surveille les signaux d'arrêt. Port indisponible : sortie immédiate, avant toute
// ouverture de base.
func startBootServer(srv *http.Server, gate *bootGate) *bootServer {
	var lc net.ListenConfig
	ln, err := lc.Listen(context.Background(), "tcp", srv.Addr)
	if err != nil {
		slog.ErrorContext(context.Background(), "démarrage : port indisponible", "addr", srv.Addr, "err", err)
		fmt.Fprintf(os.Stderr, "  [ERR] Port %s deja occupe -- fermez l'ancien processus\n", srv.Addr)
		os.Exit(1)
	}
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	b := newBootServer(srv, gate, os.Exit)
	go b.watchSignals(sigCh)
	go b.serve(ln)
	return b
}

// serve sert ln jusqu'à l'arrêt ; une erreur de service (hors arrêt) termine le processus.
func (b *bootServer) serve(ln net.Listener) {
	if err := b.srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.ErrorContext(context.Background(), "server error", "err", err)
		b.forceExit(1)
	}
}

// watchSignals ferme stop au premier signal. Si l'initialisation est en cours, il annonce
// l'arrêt différé et force la sortie au second signal reçu avant la fin de l'initialisation.
func (b *bootServer) watchSignals(sigCh <-chan os.Signal) {
	first := <-sigCh
	close(b.stop)
	if b.initDone.Load() {
		return
	}
	slog.WarnContext(context.Background(), "démarrage : arrêt demandé pendant l'initialisation — arrêt propre dès la fin "+
		"de l'initialisation ; un second signal force l'arrêt immédiat",
		"signal", first.String(), "step", b.gate.currentStep())
	second := <-sigCh
	if b.initDone.Load() {
		return
	}
	slog.ErrorContext(context.Background(), "démarrage : second signal pendant l'initialisation — arrêt immédiat, bases non fermées",
		"signal", second.String(), "step", b.gate.currentStep())
	b.forceExit(bootForcedExitCode)
}

// stopRequested est fermé au premier signal d'arrêt.
func (b *bootServer) stopRequested() <-chan struct{} { return b.stop }

// openRouter marque la fin de l'initialisation et branche le routeur, sauf si un arrêt a été
// demandé entre-temps : main enchaîne alors l'arrêt propre sans avoir servi le routeur.
func (b *bootServer) openRouter(ctx context.Context, router http.Handler) {
	b.initDone.Store(true)
	select {
	case <-b.stop:
		slog.InfoContext(ctx, "démarrage : initialisation terminée, arrêt demandé — routeur non branché")
		return
	default:
	}
	b.gate.open(ctx, router)
	fmt.Fprintf(os.Stderr, "\n  [OK] LevelUp API ready -> http://%s\n\n", b.srv.Addr)
}
