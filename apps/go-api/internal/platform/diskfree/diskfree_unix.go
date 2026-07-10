//go:build !windows

// Package diskfree — façade minimale « espace disque libre » (DC-4 : build tags
// windows/linux, zéro nouvelle dépendance — golang.org/x/sys est déjà dans le
// graphe). Utilisée par le monitoring ressources (A5). Variante unix (statfs).
package diskfree

import "golang.org/x/sys/unix"

// Free retourne (libre, total) en octets pour le volume contenant path.
func Free(path string) (free, total uint64, err error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, 0, err
	}
	bsize := uint64(st.Bsize) //nolint:unconvert // Bsize est int64 sur certains OS
	return st.Bavail * bsize, st.Blocks * bsize, nil
}
