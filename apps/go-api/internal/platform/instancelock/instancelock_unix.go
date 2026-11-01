//go:build unix

package instancelock

// instancelock_unix.go : le verrou par `flock`, porté par la description de fichier ouverte (deux
// ouvertures du même fichier s'excluent, même dans un seul processus).

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// verrouiller prend le verrou exclusif sans attendre ; errVerrouTenu s'il est tenu ailleurs.
func verrouiller(f *os.File) error {
	err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) {
		return errVerrouTenu
	}
	return err
}

// deverrouiller rend le verrou posé par verrouiller.
func deverrouiller(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN)
}
