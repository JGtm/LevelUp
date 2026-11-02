//go:build !windows

package filmproc

import "os/exec"

// sansFenetre : sans effet hors Windows, ou aucun parent sans console ne fait naitre de fenetre.
func sansFenetre(_ *exec.Cmd) {}
