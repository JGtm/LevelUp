//go:build !windows

package ops

import (
	"context"
	"errors"
)

// errRevealUnsupported : aucun explorateur n'est piloté hors de Windows.
var errRevealUnsupported = errors.New("ouverture de l'explorateur non prise en charge sur ce système")

// systemRevealOpen refuse : la capacité RevealFilesInExplorer n'existe que sur le bureau Windows.
func systemRevealOpen(context.Context, string, bool) error { return errRevealUnsupported }
