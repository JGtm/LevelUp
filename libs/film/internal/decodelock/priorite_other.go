//go:build !windows

package decodelock

import "testing"

// BaisserLaPriorite : sans effet hors Windows, ou les instruments ne tournent pas sur un poste de
// travail interactif.
func BaisserLaPriorite(_ testing.TB, _ string) {}
