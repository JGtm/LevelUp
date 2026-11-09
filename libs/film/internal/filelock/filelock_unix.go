//go:build unix

package filelock

// filelock_unix.go — le verrou par `flock` (porte par la description de fichier ouverte : deux
// ouvertures du meme fichier s'excluent, meme dans un seul processus).

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// verrouiller prend le verrou exclusif sans attendre ; [ErrLocked] s'il est tenu ailleurs.
func verrouiller(f *os.File) error {
	err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) {
		return ErrLocked
	}
	return err
}

// deverrouiller rend le verrou pose par [verrouiller].
func deverrouiller(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN)
}
