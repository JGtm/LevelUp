//go:build windows

package gameproc

import (
	"errors"
	"fmt"
	"log/slog"
	"unsafe"

	"golang.org/x/sys/windows"
)

// systemLister énumère les processus par un instantané ToolHelp32 : une liste de noms de fichiers,
// aucune ouverture de processus ni lecture de sa mémoire.
type systemLister struct{}

// NewSystemLister rend l'énumération du système.
func NewSystemLister() Lister { return systemLister{} }

// Names prend un instantané des processus et rend le nom d'exécutable de chacun.
func (systemLister) Names() ([]string, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("instantané des processus: %w", err)
	}
	defer func() {
		if cerr := windows.CloseHandle(snap); cerr != nil {
			slog.Warn("game_mode: instantané des processus non fermé", "err", cerr)
		}
	}()
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snap, &entry); err != nil {
		return nil, fmt.Errorf("premier processus: %w", err)
	}
	var names []string
	for {
		names = append(names, windows.UTF16ToString(entry.ExeFile[:]))
		if err := windows.Process32Next(snap, &entry); err != nil {
			if errors.Is(err, windows.ERROR_NO_MORE_FILES) {
				return names, nil
			}
			return names, fmt.Errorf("processus suivant: %w", err)
		}
	}
}
