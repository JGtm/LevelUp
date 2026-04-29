// Package git — implémentation port.GitProvider basée sur exec.Command("git", ...).
//
// P8.10 (revue 2026-04-29 gap #4) : l'exec.Command vit désormais ici, hors
// des handlers HTTP. Les services qui consomment l'historique des release
// notes peuvent être mockés via port.GitProvider sans dépendance binaire.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// CLI implémente port.GitProvider via le binaire git du système.
type CLI struct{}

// NewCLI retourne une instance de CLI.
func NewCLI() *CLI { return &CLI{} }

// LogSHAs retourne les SHAs (descendants → ancêtres) ayant modifié relPath.
func (g *CLI) LogSHAs(repoRoot, relPath string) ([]string, error) {
	cmd := exec.Command("git", "log", "--all", "--format=%H", "--", relPath)
	cmd.Dir = repoRoot
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git.LogSHAs %q: %w", relPath, err)
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			out = append(out, l)
		}
	}
	return out, nil
}

// ShowFile retourne le contenu de relPath au commit sha.
func (g *CLI) ShowFile(repoRoot, sha, relPath string) (string, error) {
	cmd := exec.Command("git", "show", sha+":"+relPath)
	cmd.Dir = repoRoot
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git.ShowFile %s:%s: %w", sha, relPath, err)
	}
	return stdout.String(), nil
}
