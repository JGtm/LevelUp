// desktop_port.go — le port stable du mode desktop (ADR 0045 D-9).
//
// Le stockage du navigateur de la coquille (localStorage, IndexedDB) est indexé par origine, port
// compris : un port tiré au hasard à chaque lancement le viderait à chaque fois. Le serveur garde
// donc son port dans <DataRoot>/desktop/port (PathResolver.DesktopPortPath). Si le port gardé est
// libre, il le reprend ; sinon (fichier absent, illisible, port occupé), il en choisit un libre et
// l'écrit. Jamais `:0` sans cette écriture. Hors mode desktop, ce fichier n'existe pas : le port
// configuré doit être libre (bindConfiguredPort).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"levelup/go-api/internal/platform/atomicfile"
)

// maxTCPPort est le plus grand numéro de port TCP.
const maxTCPPort = 65535

// portFilePerm : mode du fichier de port, lisible par la coquille (même utilisateur).
const portFilePerm = 0o644

// bindConfiguredPort lie addr tel quel : le port configuré d'un serveur hébergé ou de
// développement doit être libre, sans repli.
func bindConfiguredPort(ctx context.Context, addr string) (net.Listener, error) {
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("écoute sur %s : %w", addr, err)
	}
	return ln, nil
}

// bindDesktopPort lie le port du mode desktop sur host : celui de portPath s'il est libre, sinon un
// port libre choisi par le système. persist dit si le nouveau port est écrit dans portPath ; faux
// pour une instance qui n'a pas le verrou d'instance (elle ne doit pas changer le port de celle qui
// tourne).
func bindDesktopPort(ctx context.Context, host, portPath string, persist bool) (net.Listener, error) {
	var lc net.ListenConfig
	kept, hadKept := readStablePort(ctx, portPath)
	if hadKept {
		ln, err := lc.Listen(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(kept)))
		if err == nil {
			return ln, nil
		}
		slog.WarnContext(ctx, "port de repli : le port gardé est occupé, un port libre est choisi "+
			"(le stockage local du navigateur de cet utilisateur sera perdu une fois)",
			"port_garde", kept, "err", err)
	}
	ln, err := lc.Listen(ctx, "tcp", net.JoinHostPort(host, "0"))
	if err != nil {
		return nil, fmt.Errorf("écoute sur un port libre de %s : %w", host, err)
	}
	if !persist {
		return ln, nil
	}
	port := listenerPort(ln)
	if werr := writeStablePort(portPath, port); werr != nil {
		slog.WarnContext(ctx, "port stable : écriture impossible, le port ne sera pas repris au prochain lancement",
			"path", portPath, "port", port, "err", werr)
	}
	return ln, nil
}

// listenerPort rend le port TCP lié par ln (0 si ln n'est pas une écoute TCP).
func listenerPort(ln net.Listener) int {
	if addr, ok := ln.Addr().(*net.TCPAddr); ok {
		return addr.Port
	}
	return 0
}

// readStablePort lit le port gardé dans path. Faux si le fichier est absent (premier lancement) ou
// si son contenu n'est pas un port valide (journalisé).
func readStablePort(ctx context.Context, path string) (int, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.WarnContext(ctx, "port stable : lecture impossible, un port libre sera choisi",
				"path", path, "err", err)
		}
		return 0, false
	}
	text := strings.TrimSpace(string(raw))
	port, perr := strconv.Atoi(text)
	if perr != nil || port < 1 || port > maxTCPPort {
		slog.WarnContext(ctx, "port stable : contenu invalide, un port libre sera choisi",
			"path", path, "contenu", text)
		return 0, false
	}
	return port, true
}

// writeStablePort écrit port dans path, atomiquement (une seule ligne, le port en décimal).
func writeStablePort(path string, port int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("dossier de %s : %w", path, err)
	}
	return atomicfile.WriteFileStrict(path, []byte(strconv.Itoa(port)+"\n"), portFilePerm)
}
