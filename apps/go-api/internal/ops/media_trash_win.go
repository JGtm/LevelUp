//go:build windows

package ops

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// Fonction et drapeaux de SHFileOperationW (shellapi.h).
const (
	foDelete = 0x3 // FO_DELETE

	fofSilent         = 0x4   // FOF_SILENT : pas de boîte de progression
	fofNoConfirmation = 0x10  // FOF_NOCONFIRMATION : pas de question
	fofAllowUndo      = 0x40  // FOF_ALLOWUNDO : vers la Corbeille
	fofNoErrorUI      = 0x400 // FOF_NOERRORUI : pas de boîte d'erreur
	// FOF_WANTNUKEWARNING : Windows interroge l'utilisateur avant de détruire un fichier qu'il ne
	// peut pas recycler (plus gros que la Corbeille du lecteur). Sans lui, FOF_NOCONFIRMATION
	// répond oui à sa place. Un refus remonte comme une opération annulée.
	fofWantNukeWarning = 0x4000
)

// shFileOpStruct reproduit SHFILEOPSTRUCTW pour les architectures 64 bits (alignement naturel ;
// la variante 32 bits est compactée, et n'est pas livrée).
type shFileOpStruct struct {
	hwnd                  uintptr
	wFunc                 uint32
	pFrom                 *uint16
	pTo                   *uint16
	fFlags                uint16
	fAnyOperationsAborted int32
	hNameMappings         uintptr
	lpszProgressTitle     *uint16
}

var procSHFileOperation = windows.NewLazySystemDLL("shell32.dll").NewProc("SHFileOperationW")

// systemDriveType rend le type du lecteur dont root (« C:\ ») est la racine (GetDriveTypeW).
func systemDriveType(root string) uint32 {
	p, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return windows.DRIVE_UNKNOWN
	}
	return windows.GetDriveType(p)
}

// systemShellDelete envoie abs à la Corbeille par SHFileOperationW (FO_DELETE avec FOF_ALLOWUNDO,
// sans interface). pFrom est une liste terminée par DEUX zéros ; un seul chemin y figure. Le retour
// de SHFileOperationW est son propre code (0 = réussite), pas un errno : seul un échec de
// l'appel (DLL ou procédure introuvable) rend une erreur.
func systemShellDelete(abs string) (int32, bool, error) {
	from, err := windows.UTF16FromString(abs)
	if err != nil {
		return 0, false, err
	}
	from = append(from, 0) // second zéro terminal : fin de la liste
	op := shFileOpStruct{
		wFunc:  foDelete,
		pFrom:  &from[0],
		fFlags: fofAllowUndo | fofNoConfirmation | fofSilent | fofNoErrorUI | fofWantNukeWarning,
	}
	if err := procSHFileOperation.Find(); err != nil {
		return 0, false, err
	}
	ret, _, _ := procSHFileOperation.Call(uintptr(unsafe.Pointer(&op)))
	return int32(ret), op.fAnyOperationsAborted != 0, nil
}
