package filelock

// filelock_windows.go — le verrou par `LockFileEx` (Win32). Un octet verrouille suffit : le
// fichier de verrou ne porte aucune donnee, et le verrou n'est qu'un jeton d'exclusion.

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// verrouiller prend le verrou exclusif sans attendre ; [ErrLocked] s'il est tenu ailleurs.
func verrouiller(f *os.File) error {
	var ov windows.Overlapped
	err := windows.LockFileEx(windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &ov)
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return ErrLocked
	}
	return err
}

// deverrouiller rend le verrou pose par [verrouiller].
func deverrouiller(f *os.File) error {
	var ov windows.Overlapped
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &ov)
}
