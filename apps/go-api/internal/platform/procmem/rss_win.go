//go:build windows

package procmem

import (
	"log/slog"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// processMemoryCounters : PROCESS_MEMORY_COUNTERS de psapi.h.
type processMemoryCounters struct {
	cb                         uint32
	pageFaultCount             uint32
	peakWorkingSetSize         uintptr
	workingSetSize             uintptr
	quotaPeakPagedPoolUsage    uintptr
	quotaPagedPoolUsage        uintptr
	quotaPeakNonPagedPoolUsage uintptr
	quotaNonPagedPoolUsage     uintptr
	pagefileUsage              uintptr
	peakPagefileUsage          uintptr
}

var (
	psapi                = windows.NewLazySystemDLL("psapi.dll")
	procGetProcessMemory = psapi.NewProc("GetProcessMemoryInfo")
	warnOnce             sync.Once
)

// residentBytes lit l'ensemble de travail du processus courant.
func residentBytes() (uint64, bool) {
	var c processMemoryCounters
	c.cb = uint32(unsafe.Sizeof(c))
	h := windows.CurrentProcess()
	r, _, err := procGetProcessMemory.Call(uintptr(h), uintptr(unsafe.Pointer(&c)), uintptr(c.cb))
	if r == 0 {
		warnOnce.Do(func() { slog.Warn("procmem: GetProcessMemoryInfo a échoué, mémoire non mesurée", "err", err) })
		return 0, false
	}
	return uint64(c.workingSetSize), true
}
