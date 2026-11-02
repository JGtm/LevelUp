// boot_listen.go — ouverture du port du serveur : la porte de démarrage (boot_gate.go) écoute avant
// toute base. Hors -mode desktop, sur le port configuré ; en -mode desktop, sous le verrou
// d'instance et sur le port stable (desktop_port.go), avec les événements du contrat coquille <->
// serveur (desktop_events.go).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/instancelock"
)

// openBootGate ouvre le port sur la porte de démarrage et branche les demandes d'arrêt. En
// -mode desktop, il prend d'abord le verrou d'instance, lie le port stable, puis annonce
// l'événement `listening` (avant les migrations).
func openBootGate(st *bootState) {
	cfg, flags := st.cfg, st.flags
	// --- 2 bis. Écoute immédiate (boot_gate.go) : la page est servie et l'API répond 503
	// server_starting jusqu'à bootSrv.openRouter ; un port indisponible arrête ici, avant les bases.
	gate := newBootGate(cfg)
	srv := newBootHTTPServer(cfg, gate)
	if flags.Mode == serverModeDesktop {
		st.events = newDesktopEvents(os.Stdout)
	}
	ln := bindBootListener(st, srv.Addr)
	if flags.Mode == serverModeDesktop {
		// Le port réel n'est connu qu'une fois lié : le serveur HTTP et la configuration le voient.
		srv.Addr = ln.Addr().String()
		cfg.SetBoundPort(listenerPort(ln))
		srv.Handler = bootHandler(cfg, gate)
	}
	bootSrv := startBootServer(srv, gate, ln)
	bootSrv.events = st.events
	st.events.Listening(bootSrv.port)
	// Arrêt demandé hors signal : route POST /api/v1/internal/shutdown (posée avant la
	// construction du routeur, même patron que TitleReadyNotifier) et fermeture de stdin.
	cfg.ShutdownRequester = func(reason string) { bootSrv.requestShutdown(reason) }
	if flags.StopOnStdin {
		bootSrv.watchStdin(os.Stdin)
	}
	st.gate, st.srv, st.bootSrv = gate, srv, bootSrv
}

// bootHandler rend le handler du serveur HTTP de démarrage. Sous l'entrée de la coquille desktop,
// la porte répond avant le routeur et son garde : le même contrôle de Host (421) l'enveloppe, sans
// cookie (la coquille charge la page de démarrage ou d'échec avant de l'avoir, ADR 0045 D-3). Ailleurs,
// la porte seule.
func bootHandler(cfg *config.AppConfig, gate *bootGate) http.Handler {
	if cfg.Capabilities.ShellEntryGate {
		return middleware.StrictHost(cfg.DesktopAuthority())(gate)
	}
	return gate
}

// bindBootListener lie le port du serveur. Hors -mode desktop : le port configuré (addr), qui doit
// être libre, sinon sortie immédiate. En -mode desktop : le verrou d'instance d'abord (une autre
// instance ne doit pas toucher au fichier de port de celle qui tourne), puis le port stable
// (desktop_port.go) ; l'échec d'écoute y est annoncé `failed{state:"port"}` avant la sortie.
func bindBootListener(st *bootState, addr string) net.Listener {
	ctx := context.Background()
	if st.flags.Mode != serverModeDesktop {
		ln, err := bindConfiguredPort(ctx, addr)
		if err != nil {
			slog.ErrorContext(ctx, "démarrage : port indisponible", "addr", addr, "err", err)
			fmt.Fprintf(os.Stderr, "  [ERR] Port %s deja occupe -- fermez l'ancien processus\n", addr)
			os.Exit(1)
		}
		return ln
	}
	pr := st.cfg.PathResolver()
	st.instanceLock, st.instanceLockErr = instancelock.Acquire(pr.InstanceLockPath())
	if st.instanceLockErr != nil {
		slog.ErrorContext(ctx, "démarrage : verrou d'instance non pris", "path", pr.InstanceLockPath(), "err", st.instanceLockErr)
	}
	ln, err := bindDesktopPort(ctx, st.cfg.APIHost, pr.DesktopPortPath(), st.instanceLockErr == nil)
	if err != nil {
		slog.ErrorContext(ctx, "démarrage : aucun port disponible", "host", st.cfg.APIHost, "err", err)
		st.events.Failed(bootFailurePort)
		os.Exit(1)
	}
	return ln
}

// releaseInstanceLock rend le verrou d'instance tenu par ce processus (sans effet s'il n'en tient
// pas). Appelé à la fin de l'arrêt, AVANT l'événement `stopped` : une instance relancée dès qu'elle
// lit `stopped` ne doit pas retrouver le verrou tenu par un processus qui sort.
func (st *bootState) releaseInstanceLock() {
	if err := st.instanceLock.Release(); err != nil {
		slog.Warn("arrêt : libération du verrou d'instance échouée", "err", err)
	}
}
