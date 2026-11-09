package schedtask

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"levelup/go-api/internal/platform/childproc"
)

// runTimeout borne un appel de schtasks : un planificateur qui ne répond pas ne retient pas le serveur.
const runTimeout = 30 * time.Second

// ExecRunner exécute le vrai schtasks.exe, par childproc.Command (jamais de fenêtre de console).
type ExecRunner struct{}

// Run lance schtasks avec args ; une sortie non nulle rend une erreur portant la fin de la sortie.
func (ExecRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()
	cmd := childproc.Command(ctx, schtasksPath(), args...)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		return out.Bytes(), fmt.Errorf("%w : %s", err, lastLine(errOut.String(), out.String()))
	}
	return out.Bytes(), nil
}

// schtasksPath rend le chemin de schtasks.exe dans System32 (jamais une résolution par le PATH, que le dossier
// de travail ou une variable pourraient détourner) ; à défaut de SystemRoot, le nom nu.
func schtasksPath() string {
	if root := os.Getenv("SystemRoot"); root != "" {
		return filepath.Join(root, "System32", "schtasks.exe")
	}
	return "schtasks.exe"
}

// lastLine rend la dernière ligne non vide du premier texte non vide (le message de schtasks).
func lastLine(texts ...string) string {
	for _, t := range texts {
		lines := strings.Split(strings.TrimSpace(t), "\n")
		if last := strings.TrimSpace(lines[len(lines)-1]); last != "" {
			return last
		}
	}
	return "aucun message"
}

// Observed garde la dernière observation de la tâche, que le réglage et la page des réglages relisent : l'état
// (présente, absente, erreur) et le code de l'erreur, jamais son texte.
type Observed struct {
	mu    sync.RWMutex
	state State
	code  string
}

// Code d'erreur exposé : le texte de schtasks va au journal, jamais au client.
const (
	// ErrorCodeSchtasks : un appel de schtasks a échoué (planificateur indisponible, accès refusé).
	ErrorCodeSchtasks = "schtasks_failed"
	// ErrorCodePath : un chemin de racine ne peut pas être posé dans la ligne de commande de la tâche.
	ErrorCodePath = "unsupported_path"
)

// Set pose l'observation. Un état sans erreur efface le code.
func (o *Observed) Set(state State, code string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.state, o.code = state, code
}

// Get rend l'état et le code ; l'état est vide tant que rien n'a été observé.
func (o *Observed) Get() (State, string) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.state, o.code
}

// defaultObserved est l'observation du processus : la boucle du boot l'écrit, la page des réglages la lit.
var defaultObserved = &Observed{}

// DefaultObserved rend l'observation du processus.
func DefaultObserved() *Observed { return defaultObserved }
