//go:build windows

// Package diskfree — façade minimale « espace disque libre » (DC-4 : build tags
// windows/linux, zéro nouvelle dépendance — golang.org/x/sys est déjà dans le
// graphe). Utilisée par le monitoring ressources (A5).
package diskfree

import "golang.org/x/sys/windows"

// Free retourne (libre, total) en octets pour le volume contenant path.
func Free(path string) (free, total uint64, err error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, err
	}
	var freeBytesAvailable, totalBytes, totalFree uint64
	if err := windows.GetDiskFreeSpaceEx(p, &freeBytesAvailable, &totalBytes, &totalFree); err != nil {
		return 0, 0, err
	}
	return freeBytesAvailable, totalBytes, nil
}
