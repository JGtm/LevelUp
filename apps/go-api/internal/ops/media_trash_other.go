//go:build !windows

package ops

import "errors"

// errTrashUnsupported : aucune Corbeille n'est pilotée hors de Windows.
var errTrashUnsupported = errors.New("corbeille non prise en charge sur ce système")

// systemDriveType rend 0 (type inconnu) : aucun lecteur n'est jugé fixe, donc TrashMediaFile rend
// false, nil et n'appelle jamais systemShellDelete.
func systemDriveType(string) uint32 { return 0 }

// systemShellDelete n'est jamais appelée (systemDriveType exclut tout lecteur) ; elle refuse si elle
// l'était, plutôt que de laisser croire à un envoi réussi.
func systemShellDelete(string) (int32, bool, error) { return 0, false, errTrashUnsupported }
