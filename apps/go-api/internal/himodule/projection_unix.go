//go:build !windows

package himodule

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// projetteFichier projette un fichier en lecture seule (Linux, macOS — la CI tourne dessus).
//
// MAP_SHARED plutot que MAP_PRIVATE : la projection est en lecture seule, et le mode partage
// laisse le systeme servir les MEMES pages a tous les processus qui lisent l'archive. En
// prive, chaque projection paierait ses propres pages des la premiere lecture.
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

	octets, err := unix.Mmap(int(f.Fd()), 0, int(taille), unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		return nil, nil, fmt.Errorf("mmap: %w", err)
	}
	return octets, func() error { return unix.Munmap(octets) }, nil
}
