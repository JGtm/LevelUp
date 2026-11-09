//go:build !windows

// Package diskfree — façade minimale « espace disque libre » (DC-4 : build tags
// windows/linux, zéro nouvelle dépendance — golang.org/x/sys est déjà dans le
// graphe). Utilisée par le monitoring ressources (A5). Variante unix (statfs).
package diskfree

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// Free retourne (libre, total) en octets pour le volume contenant path.
func Free(path string) (free, total uint64, err error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, 0, err
	}
	bsize := uint64(st.Bsize) //nolint:unconvert // Bsize est int64 sur certains OS
	return st.Bavail * bsize, st.Blocks * bsize, nil
}

// device rend le numéro de périphérique qui porte path.
func device(path string) (uint64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("diskfree: périphérique de %s indisponible", path)
	}
	return uint64(st.Dev), nil //nolint:unconvert // Dev est int32 ou uint64 selon la plateforme
}

// SameVolume dit si a et b (deux chemins EXISTANTS) sont sur le même périphérique.
func SameVolume(a, b string) (bool, error) {
	da, err := device(a)
	if err != nil {
		return false, err
	}
	db, err := device(b)
	if err != nil {
		return false, err
	}
	return da == db, nil
}
