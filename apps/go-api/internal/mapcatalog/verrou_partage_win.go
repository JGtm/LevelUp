//go:build windows

package mapcatalog

import (
	"errors"
	"syscall"
)

// errSharingViolation : ERROR_SHARING_VIOLATION (32). Go ne l'associe pas à fs.ErrPermission ; Windows la
// rend quand un autre écrivain tient encore le fichier de verrou ouvert au moment où il le supprime.
const errSharingViolation syscall.Errno = 32

// partageRefuse dit si err est un partage refusé du fichier de verrou.
func partageRefuse(err error) bool { return errors.Is(err, errSharingViolation) }
