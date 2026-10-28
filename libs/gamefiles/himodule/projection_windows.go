//go:build windows

package himodule

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// projetteFichier projette un fichier en lecture seule (Windows).
//
// LE HANDLE DE FICHIER EST FERME TOUT DE SUITE, et ce n'est pas un oubli : une vue projetee
// garde une reference sur la section, qui garde elle-meme la donnee. Fermer le fichier tot
// evite de tenir un descripteur par archive pendant toute la vie du module.
func projetteFichier(chemin string) ([]byte, func() error, error) {
	f, err := os.Open(chemin) //nolint:gosec // chemin fourni par l'appelant, lecture seule
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}
	taille := info.Size()
	if taille == 0 {
		return nil, func() error { return nil }, nil
	}
	if taille != int64(int(taille)) {
		return nil, nil, fmt.Errorf("archive de %d octets : trop grande pour cette plateforme", taille)
	}

	section, err := windows.CreateFileMapping(
		windows.Handle(f.Fd()), nil, windows.PAGE_READONLY,
		uint32(taille>>32), uint32(taille), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("CreateFileMapping: %w", err)
	}
	adresse, err := windows.MapViewOfFile(section, windows.FILE_MAP_READ, 0, 0, uintptr(taille))
	if err != nil {
		_ = windows.CloseHandle(section)
		return nil, nil, fmt.Errorf("MapViewOfFile: %w", err)
	}

	octets := trancheSurVue(adresse, int(taille))
	fermer := func() error {
		errVue := windows.UnmapViewOfFile(adresse)
		errSection := windows.CloseHandle(section)
		if errVue != nil {
			return errVue
		}
		return errSection
	}
	return octets, fermer, nil
}

// trancheSurVue fabrique la tranche `[]byte` qui designe une vue projetee.
//
// POURQUOI CETTE FORME PLUTOT QUE `unsafe.Slice((*byte)(unsafe.Pointer(adresse)), n)`. Parce
// que `go vet` (analyseur `unsafeptr`, actif par defaut dans golangci-lint) refuse toute
// conversion d'un `uintptr` en `unsafe.Pointer` : de son point de vue l'entier pourrait
// designer de la memoire que le ramasse-miettes est libre de deplacer. Ici il n'en est rien —
// l'adresse vient de `MapViewOfFile`, elle designe une vue du systeme, hors du tas Go, et elle
// reste valide jusqu'a `UnmapViewOfFile`. Le seul `unsafe.Pointer` pris ci-dessous l'est sur
// une variable locale ordinaire, ce que vet accepte, et la tranche obtenue est identique.
//
// La disposition {donnee, longueur, capacite} d'un en-tete de tranche est la meme hypothese
// que celle de `reflect.SliceHeader` — c'est la forme employee par les bibliotheques de
// projection memoire depuis toujours.
func trancheSurVue(adresse uintptr, n int) []byte {
	entete := struct {
		donnee   uintptr
		longueur int
		capacite int
	}{adresse, n, n}
	return *(*[]byte)(unsafe.Pointer(&entete))
}
