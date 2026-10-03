package filmproc

// solo.go — LE VERROU « UN SEUL DECODAGE DE FILM A LA FOIS SUR CETTE MACHINE ».
//
// # LE QUATRIEME SINISTRE, ET IL A TRAVERSE TOUTES LES SECURITES EXISTANTES (2026-08-31)
//
// Les trois premiers sinistres (cf. l'en-tete du paquet) ont produit trois remedes : un
// processus par film, un plafond memoire par processus, une priorite CPU basse. Le quatrieme a
// pris une porte qu'AUCUN des trois ne garde — l'OPERATEUR.
//
// `cmd/replay-build` est declare au ratchet `archlint.TestNoUnboundedFilmLoop` avec la
// justification « CLI unitaire : un film par invocation, le processus meurt ensuite. Aucune
// boucle ». C'est VRAI a l'interieur du processus, et c'est EXACTEMENT ce qui a manque : rien
// n'empechait de mettre ce CLI dans une boucle de shell, et encore moins d'en lancer plusieurs
// en parallele en arriere-plan. C'est ce qui s'est passe — une boucle au premier plan pendant
// qu'une seconde tournait en fond — et la machine de travail de l'utilisateur a de nouveau
// suffoque. Le binaire n'armait par ailleurs AUCUNE sentinelle : il etait le seul point
// d'entree de decodage du depot dans ce cas.
//
// La lecon, ecrite pour la prochaine lecture : **« un film par invocation » ne dit rien du
// nombre d'invocations.** Une garantie qui s'arrete a la frontiere du processus doit etre
// reprise a l'exterieur, par un verrou que TOUS les points d'entree respectent.
//
// # CE QUE CE VERROU FAIT, ET CE QU'IL NE FAIT PAS
//
// Il rend le decodage de film MUTUELLEMENT EXCLUSIF sur une machine : un second point d'entree
// qui demande le verrou ECHOUE en nommant le detenteur, plutot que de decoder en parallele. Il
// ne borne pas la memoire (c'est [Arm]) et n'ordonne pas une file d'attente : il REFUSE. Un
// refus est le bon comportement pour un outil d'operateur — il rend la main tout de suite, avec
// le nom du processus qui travaille deja.
//
// # UN VERROU DU NOYAU, PAS UN BATTEMENT DE COEUR (OPS-1, DT-2, 2026-09-26)
//
// Jusqu'ici le verrou etait un FICHIER : present = tenu, un horodatage reecrit toutes les deux
// secondes, et un verrou dont le battement datait de plus de six secondes etait tenu pour mort et
// repris. La regle se trompait dans les deux sens : un detenteur GELE (pause, veille, disque
// lent) se faisait deposseder pendant qu'il decodait encore, puis son Release EFFACAIT le fichier
// de son successeur — et un troisieme decodait en parallele du second.
//
// Le verrou est desormais celui du NOYAU (`platform/filelock` : `LockFileEx` / `flock`) sur
// `film_decode.lock` : exclusif tant que le processus vit, rendu par le systeme a sa mort — la
// sentinelle memoire et l'operateur tuent, c'est le cas nominal — et a aucun autre moment. Ni
// battement, ni reprise de verrou perime, ni intervention manuelle.
//
// Le DETENTEUR est decrit a part, dans `film_decode.lock.json` (outil, pid, match, debut),
// ecrit ATOMIQUEMENT une fois le verrou pris et retire AVANT qu'il soit rendu : tant que le
// verrou est tenu, la description est celle de son detenteur, et un refus nomme donc le
// detenteur ACTUEL. Le fichier de verrou lui-meme n'est jamais supprime (cf. `filelock`).

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"levelup/go-api/internal/platform/atomicfile"
	"levelup/go-api/internal/platform/filelock"
)

const (
	// soloFileName : le fichier du verrou OS, sous la racine du cache film. Il reste en place.
	soloFileName = "film_decode.lock"
	// soloHolderFileName : la description du detenteur, presente tant que le verrou est tenu.
	soloHolderFileName = soloFileName + ".json"
)

// ErrDecodeBusy : un autre decodage de film tient deja la machine. C'est un REFUS attendu, pas
// une panne — l'appelant le presente a l'operateur et sort proprement.
var ErrDecodeBusy = errors.New("un decodage de film est deja en cours sur cette machine")

// soloHolder : ce qu'un detenteur ecrit dans sa description.
type soloHolder struct {
	Tool      string `json:"tool"`
	PID       int    `json:"pid"`
	MatchID   string `json:"matchId,omitempty"`
	StartedAt string `json:"startedAt"`
}

// SoloLock : le verrou tenu. [SoloLock.Release] le rend ; il est sur d'appeler Release deux fois.
type SoloLock struct {
	lock       *filelock.Lock
	holderPath string
	once       sync.Once
}

// AcquireSolo prend le verrou de decodage pour cette machine, ou rend [ErrDecodeBusy].
//
// `cacheRoot` est la racine du cache film (`PathResolver.CacheRootDir()`) : le verrou vit a cote
// des donnees qu'il protege, donc deux depots de travail distincts ne se bloquent pas l'un
// l'autre — ils ne decodent pas les memes fichiers et ne se disputent que la RAM, ce que la
// sentinelle borne deja de son cote.
//
// `tool` et `matchID` ne servent qu'au MESSAGE : quand un operateur se voit refuser le verrou,
// il doit lire QUI travaille et SUR QUOI, sinon il ira tuer le mauvais processus.
func AcquireSolo(cacheRoot, tool, matchID string) (*SoloLock, error) {
	if cacheRoot == "" {
		return nil, fmt.Errorf("filmproc: racine de cache vide — le verrou n'aurait pas d'endroit ou vivre")
	}
	if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
		return nil, fmt.Errorf("filmproc: creation de %s: %w", cacheRoot, err)
	}
	holderPath := filepath.Join(cacheRoot, soloHolderFileName)
	lock, err := filelock.TryLock(filepath.Join(cacheRoot, soloFileName))
	if errors.Is(err, filelock.ErrLocked) {
		return nil, soloBusyError(holderPath)
	}
	if err != nil {
		return nil, fmt.Errorf("filmproc: verrou de decodage: %w", err)
	}
	if err := soloWriteHolder(holderPath, tool, matchID); err != nil {
		if uErr := lock.Unlock(); uErr != nil {
			slog.Warn("verrou de decodage non rendu apres un echec d'ecriture du detenteur",
				"err", uErr, "verrou", cacheRoot)
		}
		return nil, err
	}
	slog.Info("verrou de decodage pris", "outil", tool, "match_id", matchID, "verrou", holderPath)
	return &SoloLock{lock: lock, holderPath: holderPath}, nil
}

// Release rend le verrou. Sur d'etre appele plusieurs fois (defer + chemin d'erreur).
//
// La description est retiree AVANT que le verrou soit rendu : tant qu'on le tient, personne
// d'autre n'a pu l'ecrire, donc c'est bien la notre qu'on retire.
func (l *SoloLock) Release() {
	if l == nil {
		return
	}
	l.once.Do(func() {
		if err := os.Remove(l.holderPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.Warn("description du detenteur non retiree — le prochain detenteur la reecrira",
				"err", err, "fichier", l.holderPath)
		}
		if err := l.lock.Unlock(); err != nil {
			slog.Warn("verrou de decodage non rendu — le systeme le rendra a la fin du processus",
				"err", err, "fichier", l.holderPath)
		}
	})
}

// soloWriteHolder decrit le detenteur, ATOMIQUEMENT (`atomicfile.WriteFileStrict`) : un refus ne
// lit jamais une description a moitie ecrite. `StartedAt` est pose ici, une seule fois.
func soloWriteHolder(path, tool, matchID string) error {
	h := soloHolder{Tool: tool, PID: os.Getpid(), MatchID: matchID,
		StartedAt: time.Now().UTC().Format(time.RFC3339)}
	b, err := json.Marshal(h)
	if err != nil {
		return fmt.Errorf("filmproc: serialisation du detenteur: %w", err)
	}
	if err := atomicfile.WriteFileStrict(path, b, 0o644); err != nil {
		return fmt.Errorf("filmproc: ecriture du detenteur: %w", err)
	}
	return nil
}

// soloBusyError nomme le detenteur dans le message — sans lui, l'operateur ne sait pas quoi
// attendre ni quoi arreter. Une description illisible (detenteur qui vient de prendre le verrou
// et ne l'a pas encore ecrite) est dite comme telle.
func soloBusyError(holderPath string) error {
	var h soloHolder
	b, err := os.ReadFile(holderPath) //nolint:gosec // chemin compose sous la racine du cache
	if err == nil {
		err = json.Unmarshal(b, &h)
	}
	if err != nil {
		return fmt.Errorf("%w : detenteur non decrit (%v). Attendre la fin. Description : %s",
			ErrDecodeBusy, err, holderPath)
	}
	return fmt.Errorf("%w : %s (pid %d) decode %q depuis %s. Attendre la fin, ou arreter ce "+
		"processus. Description : %s", ErrDecodeBusy, h.Tool, h.PID, h.MatchID, h.StartedAt, holderPath)
}
