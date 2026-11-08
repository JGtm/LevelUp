// stdin_commands.go — les commandes que la coquille desktop écrit sur la stdin du serveur (ADR 0045 D-4,
// lot L21.c). La fermeture de stdin reste l'ordre d'arrêt (shutdown_stdin.go) ; les lignes écrites avant
// elle sont des commandes, une par ligne, que le serveur exécute sans réponse.
//
//	sync    demande un cycle de synchronisation immédiat (« Synchroniser maintenant » du menu de la zone de
//	        notification)
//	resources <json>
//	        demande de préparer le paquet de ressources que décrit le JSON (mise à jour de l'application : le paquet
//	        est téléchargé et vérifié AVANT l'installation du moteur qui l'exige) ; l'avancement revient par
//	        l'événement `resources`
//
// stdin est le tube privé que le parent tient : seul le processus qui a lancé le serveur peut y écrire, sans
// cookie ni secret. Une commande inconnue est journalisée et ignorée, jamais fatale.
package main

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
)

// stdinCommandSync : la commande de synchronisation immédiate.
const stdinCommandSync = "sync"

// maxStdinLine borne une ligne de commande : le tampon ne grossit jamais pour une ligne sans fin.
const maxStdinLine = 4096

// maxLoggedStdinLine borne ce qu'une ligne inconnue ajoute au journal.
const maxLoggedStdinLine = 40

// stdinCommands : les commandes branchées, par nom.
type stdinCommands struct {
	mu       sync.RWMutex
	handlers map[string]func()
	// argHandlers : les commandes qui portent un argument (le reste de la ligne après le nom).
	argHandlers map[string]func(arg string)
}

// Handle branche fn sur la commande name (un second appel remplace le premier).
func (c *stdinCommands) Handle(name string, fn func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.handlers == nil {
		c.handlers = make(map[string]func())
	}
	c.handlers[name] = fn
}

// HandleArg branche fn sur la commande name suivie d'un argument (le reste de la ligne, espaces de bord retirés).
func (c *stdinCommands) HandleArg(name string, fn func(arg string)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.argHandlers == nil {
		c.argHandlers = make(map[string]func(string))
	}
	c.argHandlers[name] = fn
}

// dispatch exécute la commande d'une ligne ; fn ne bloque pas (elle lance son travail à part).
func (c *stdinCommands) dispatch(ctx context.Context, line string) {
	name := strings.TrimSpace(line)
	if name == "" {
		return
	}
	word, arg, hasArg := strings.Cut(name, " ")
	c.mu.RLock()
	fn := c.handlers[name]
	argFn := c.argHandlers[word]
	c.mu.RUnlock()
	if fn == nil && hasArg && argFn != nil {
		// Le journal porte le nom de la commande, jamais son argument.
		slog.InfoContext(ctx, "stdin: commande reçue de la coquille", "commande", word)
		argFn(strings.TrimSpace(arg))
		return
	}
	if fn == nil {
		if len(name) > maxLoggedStdinLine {
			name = name[:maxLoggedStdinLine] + "…"
		}
		slog.WarnContext(ctx, "stdin: commande inconnue ignorée", "commande", name)
		return
	}
	slog.InfoContext(ctx, "stdin: commande reçue de la coquille", "commande", name)
	fn()
}

// drainStdin lit r ligne à ligne jusqu'à EOF ou erreur, exécute chaque ligne par cmds et rend la
// première erreur de lecture (nil à EOF). Une ligne plus longue que maxStdinLine est écartée en entier.
func drainStdin(ctx context.Context, r io.Reader, cmds *stdinCommands) error {
	br := bufio.NewReaderSize(r, maxStdinLine)
	var pending []byte
	discarding := false
	for {
		chunk, isPrefix, err := br.ReadLine()
		if len(chunk) > 0 && !discarding {
			pending = append(pending, chunk...)
			if len(pending) > maxStdinLine {
				discarding, pending = true, nil
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				if !discarding {
					cmds.dispatch(ctx, string(pending))
				}
				return nil
			}
			return err
		}
		if isPrefix {
			continue
		}
		if !discarding {
			cmds.dispatch(ctx, string(pending))
		}
		pending, discarding = nil, false
	}
}
