// Package filelock — un verrou EXCLUSIF entre processus, tenu par le NOYAU (DT-2 du plan de
// suite de l'audit du decodeur de film, 2026-09-26).
//
// POURQUOI LE NOYAU. Un verrou « fichier present = verrou tenu » ne sait pas si son detenteur vit
// encore : il lui faut un battement de coeur et une regle de reprise d'un verrou perime, et cette
// regle se trompe dans les deux sens (un detenteur GELE — pause, veille — se fait deposseder ; un
// detenteur TUE bloque jusqu'a l'expiration). Un verrou OS (`LockFileEx` sous Windows, `flock`
// ailleurs) est rendu par le noyau a la mort du processus qui le tient, et JAMAIS a un autre
// moment : ni battement, ni reprise.
//
// CONTRAT :
//   - [TryLock] ne bloque jamais : verrou pris, ou [ErrLocked] tout de suite ;
//   - deux [TryLock] sur le meme chemin s'excluent, y compris dans un meme processus (le verrou
//     est porte par le descripteur ouvert, pas par le processus) ;
//   - le FICHIER de verrou n'est jamais supprime : l'effacer pendant qu'un candidat l'a ouvert
//     laisserait deux detenteurs sur deux inodes differents. Il reste, vide, a cote des donnees
//     qu'il protege.
//
// `gofrs/flock` ecarte : une dependance pour la quarantaine de lignes ci-dessous, au-dessus de
// `golang.org/x/sys`, deja dependance directe du module.
package filelock

import (
	"errors"
	"fmt"
	"os"
	"sync"
)

// ErrLocked : le verrou est tenu par un autre detenteur. Un REFUS attendu, pas une panne.
var ErrLocked = errors.New("filelock: verrou tenu par un autre detenteur")

// Lock : un verrou tenu. [Lock.Unlock] le rend ; l'appeler plusieurs fois est sur.
type Lock struct {
	f    *os.File
	once sync.Once
	err  error
}

// TryLock prend le verrou exclusif du fichier `path` (cree au besoin), ou rend [ErrLocked] sans
// attendre si un autre detenteur le tient.
func TryLock(path string) (*Lock, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644) //nolint:gosec // chemin fourni par l'appelant
	if err != nil {
		return nil, fmt.Errorf("filelock: ouverture de %s : %w", path, err)
	}
	if err := verrouiller(f); err != nil {
		_ = f.Close()
		if errors.Is(err, ErrLocked) {
			return nil, err
		}
		return nil, fmt.Errorf("filelock: verrou de %s : %w", path, err)
	}
	return &Lock{f: f}, nil
}

// Unlock rend le verrou et ferme le fichier. Idempotent : les appels suivants rendent l'erreur
// du premier (nil s'il a reussi).
func (l *Lock) Unlock() error {
	if l == nil {
		return nil
	}
	l.once.Do(func() {
		errDeverrou := deverrouiller(l.f)
		errFermeture := l.f.Close()
		if err := errors.Join(errDeverrou, errFermeture); err != nil {
			l.err = fmt.Errorf("filelock: liberation de %s : %w", l.f.Name(), err)
		}
	})
	return l.err
}
