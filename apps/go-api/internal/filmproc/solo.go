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
// # POURQUOI UN BATTEMENT DE COEUR PLUTOT QU'UN TEST DE PID
//
// Un verrou pose par un processus TUE (c'est le cas nominal ici : la sentinelle memoire tue, et
// l'operateur aussi) doit pouvoir etre repris. Tester si un PID est vivant n'est pas portable —
// sur Windows `os.FindProcess` reussit toujours. Le detenteur reecrit donc un horodatage toutes
// les [soloHeartbeat] ; un verrou dont le battement date de plus de [soloStale] est MORT et se
// reprend, en le journalisant. Aucune intervention manuelle n'est jamais requise.

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// soloHeartbeat : cadence de reecriture de l'horodatage par le detenteur.
	soloHeartbeat = 2 * time.Second
	// soloStale : au-dela, le verrou est tenu pour MORT et repris. Trois battements — de quoi
	// absorber une pause de GC ou un disque lent sans laisser un verrou orphelin bloquer
	// l'operateur plus de quelques secondes.
	soloStale = 3 * soloHeartbeat
	// soloFileName : le nom du fichier de verrou, sous la racine du cache film.
	soloFileName = "film_decode.lock"
)

// ErrDecodeBusy : un autre decodage de film tient deja la machine. C'est un REFUS attendu, pas
// une panne — l'appelant le presente a l'operateur et sort proprement.
var ErrDecodeBusy = errors.New("un decodage de film est deja en cours sur cette machine")

// soloHolder : ce qu'un detenteur ecrit dans le fichier de verrou.
type soloHolder struct {
	Tool      string `json:"tool"`
	PID       int    `json:"pid"`
	MatchID   string `json:"matchId,omitempty"`
	StartedAt string `json:"startedAt"`
	BeatAt    string `json:"beatAt"`
}

// SoloLock : le verrou tenu. [SoloLock.Release] le rend ; il est sur d'appeler Release deux fois.
type SoloLock struct {
	path string
	stop chan struct{}
	once sync.Once
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
	path := filepath.Join(cacheRoot, soloFileName)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if !os.IsExist(err) {
			return nil, fmt.Errorf("filmproc: ouverture du verrou %s: %w", path, err)
		}
		if !soloStealIfDead(path) {
			return nil, soloBusyError(path)
		}
		if f, err = os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644); err != nil {
			// Course perdue contre un autre candidat qui vient de reprendre le verrou mort :
			// c'est un refus ordinaire, pas une panne.
			return nil, soloBusyError(path)
		}
	}
	l := &SoloLock{path: path, stop: make(chan struct{})}
	if err := soloWrite(f, tool, matchID); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, err
	}
	_ = f.Close()
	slog.Info("verrou de decodage pris", "outil", tool, "match_id", matchID, "verrou", path)
	go l.beat(tool, matchID)
	return l, nil
}

// Release rend le verrou. Sur d'etre appele plusieurs fois (defer + chemin d'erreur).
func (l *SoloLock) Release() {
	if l == nil {
		return
	}
	l.once.Do(func() {
		close(l.stop)
		if err := os.Remove(l.path); err != nil && !os.IsNotExist(err) {
			slog.Warn("verrou de decodage non rendu — il expirera de lui-meme",
				"err", err, "verrou", l.path, "expiration_s", int(soloStale.Seconds()))
		}
	})
}

// beat reecrit l'horodatage tant que le verrou est tenu.
func (l *SoloLock) beat(tool, matchID string) {
	t := time.NewTicker(soloHeartbeat)
	defer t.Stop()
	for {
		select {
		case <-l.stop:
			return
		case <-t.C:
			f, err := os.OpenFile(l.path, os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				continue // le verrou a ete repris ou efface : rien a sauver, le decodage finit
			}
			_ = soloWrite(f, tool, matchID)
			_ = f.Close()
		}
	}
}

// soloWrite ecrit l'etat du detenteur dans un fichier deja ouvert en ecriture.
func soloWrite(f *os.File, tool, matchID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	h := soloHolder{Tool: tool, PID: os.Getpid(), MatchID: matchID, StartedAt: now, BeatAt: now}
	b, err := json.Marshal(h)
	if err != nil {
		return fmt.Errorf("filmproc: serialisation du verrou: %w", err)
	}
	if _, err := f.Write(b); err != nil {
		return fmt.Errorf("filmproc: ecriture du verrou: %w", err)
	}
	return nil
}

// soloStealIfDead reprend un verrou dont le battement est perime. Rend vrai s'il a ete efface.
func soloStealIfDead(path string) bool {
	st, err := os.Stat(path)
	if err != nil {
		return os.IsNotExist(err) // deja parti : la voie est libre
	}
	if time.Since(st.ModTime()) < soloStale {
		return false
	}
	slog.Warn("verrou de decodage PERIME — repris",
		"verrou", path, "dernier_battement", st.ModTime().UTC().Format(time.RFC3339),
		"seuil_s", int(soloStale.Seconds()))
	return os.Remove(path) == nil
}

// soloBusyError nomme le detenteur dans le message — sans lui, l'operateur ne sait pas quoi
// attendre ni quoi arreter.
func soloBusyError(path string) error {
	var h soloHolder
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &h)
	}
	return fmt.Errorf("%w : %s (pid %d) decode %q depuis %s. Attendre la fin, ou arreter ce "+
		"processus. Verrou : %s", ErrDecodeBusy, h.Tool, h.PID, h.MatchID, h.StartedAt, path)
}
