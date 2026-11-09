// Package instancelock : le verrou d'instance du serveur, une seule instance par racine de données, tous
// profils confondus (hébergé, développement, démo, desktop, refresh-only : ADR 0045 D-10). Une restauration de
// sauvegarde (`-mode restore`) le consulte avant de remplacer des bases.
//
// Le plugin « instance unique » de la coquille ne protège ni la tâche planifiée de rafraîchissement
// ni un `go run` lancé à la main : le verrou est donc tenu par le serveur lui-même.
//
// C'EST LE NOYAU QUI TIENT LE VERROU (`LockFileEx` sous Windows, `flock` ailleurs), pas la présence
// d'un fichier : le noyau le rend à la mort du processus, tué ou non, et jamais à un autre moment.
// Un fichier « présent = pris » demanderait un battement de coeur et une règle de reprise, qui se
// trompent dans les deux sens. Le motif est celui de libs/film/internal/filelock, qu'un paquet hors
// du module libs/film ne peut pas importer (ADR 0041) : il est recopié ici.
//
// CONTRAT :
//   - [Acquire] ne bloque jamais : verrou pris, ou [ErrInstanceDejaLancee] tout de suite ;
//   - deux [Acquire] sur le même chemin s'excluent, y compris dans un même processus (le verrou est
//     porté par le descripteur ouvert) ;
//   - le fichier n'est jamais supprimé : l'effacer pendant qu'un candidat l'a ouvert laisserait deux
//     détenteurs sur deux fichiers différents. Il reste, vide, à côté des données qu'il protège.
package instancelock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ErrInstanceDejaLancee : une autre instance tient le verrou. Un refus attendu, pas une panne ; le
// serveur le traduit en échec de démarrage « locked ».
var ErrInstanceDejaLancee = errors.New("instancelock: une autre instance tient déjà ce dossier de données")

// errVerrouTenu : le verrou du noyau est pris ailleurs. Posée par verrouiller (variante de la
// plateforme), traduite en [ErrInstanceDejaLancee] par [Acquire].
var errVerrouTenu = errors.New("instancelock: verrou noyau tenu")

// Lock : un verrou tenu. [Lock.Release] le rend ; l'appeler plusieurs fois est sûr.
type Lock struct {
	f    *os.File
	once sync.Once
	err  error
}

// Acquire prend le verrou exclusif de path (fichier et dossier créés au besoin), ou rend
// [ErrInstanceDejaLancee] sans attendre si une autre instance le tient.
func Acquire(path string) (*Lock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("instancelock: dossier de %s : %w", path, err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644) //nolint:gosec // chemin fourni par le résolveur de chemins
	if err != nil {
		return nil, fmt.Errorf("instancelock: ouverture de %s : %w", path, err)
	}
	if err := verrouiller(f); err != nil {
		_ = f.Close()
		if errors.Is(err, errVerrouTenu) {
			return nil, ErrInstanceDejaLancee
		}
		return nil, fmt.Errorf("instancelock: verrou de %s : %w", path, err)
	}
	return &Lock{f: f}, nil
}

// Release rend le verrou et ferme le fichier. Idempotent : les appels suivants rendent l'erreur du
// premier (nil s'il a réussi). Un *Lock nil est accepté.
func (l *Lock) Release() error {
	if l == nil {
		return nil
	}
	l.once.Do(func() {
		errDeverrou := deverrouiller(l.f)
		errFermeture := l.f.Close()
		if err := errors.Join(errDeverrou, errFermeture); err != nil {
			l.err = fmt.Errorf("instancelock: libération de %s : %w", l.f.Name(), err)
		}
	})
	return l.err
}
