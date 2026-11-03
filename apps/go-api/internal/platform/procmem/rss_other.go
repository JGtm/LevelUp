//go:build !windows

package procmem

import "runtime"

// residentBytes approche la mémoire résidente par ce que le runtime Go a obtenu du système.
func residentBytes() (uint64, bool) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Sys, true
}
