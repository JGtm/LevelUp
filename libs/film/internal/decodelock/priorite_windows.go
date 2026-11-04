//go:build windows

package decodelock

import (
	"syscall"
	"testing"
)

// belowNormalPriorityClass : la constante Windows `BELOW_NORMAL_PRIORITY_CLASS` (winbase.h), que le
// paquet `syscall` n'expose pas.
const belowNormalPriorityClass = 0x00004000

// BaisserLaPriorite range le processus du test en priorite CPU « en dessous de la normale », pour
// que le poste reste utilisable pendant qu'un instrument decode. Un echec est dit au journal du
// test et n'arrete rien.
func BaisserLaPriorite(t testing.TB, outil string) {
	t.Helper()
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setPriorityClass := kernel32.NewProc("SetPriorityClass")
	getCurrentProcess := kernel32.NewProc("GetCurrentProcess")
	h, _, _ := getCurrentProcess.Call()
	if r, _, err := setPriorityClass.Call(h, uintptr(belowNormalPriorityClass)); r == 0 {
		t.Logf("%s : priorite basse non appliquee : %v", outil, err)
	}
}
