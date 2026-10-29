//go:build !windows

package filmproc

import (
	"errors"
	"syscall"
)

// processAlive dit si le processus `pid` existe. Signal 0 : aucune livraison, seulement le
// controle d'existence ; EPERM veut dire qu'il existe sous un autre compte.
func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
