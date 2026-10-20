//go:build windows

package filmproc

import (
	"errors"

	"golang.org/x/sys/windows"
)

// stillActive : code de sortie rendu par GetExitCodeProcess pour un processus qui tourne.
const stillActive = 259

// processAlive dit si le processus `pid` existe et n'est pas termine. Un pid refuse en acces
// (autre compte) existe ; un pid inconnu n'existe pas.
func processAlive(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return errors.Is(err, windows.ERROR_ACCESS_DENIED)
	}
	defer func() { _ = windows.CloseHandle(h) }()
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == stillActive
}
