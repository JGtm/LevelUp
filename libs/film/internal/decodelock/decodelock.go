// Package decodelock : le verrou « un decodage de film a la fois sur cette machine », pour les
// tests et les instruments de la brique qui decodent des films du cache local.
//
// La brique ne decide pas QUAND on decode : le verrou de production vit chez l'hote, avec son
// lanceur de processus. Un test de la brique qui decode un
// film reel doit pourtant respecter le meme verrou, sinon il decoderait en parallele d'une passe
// de l'hote et la machine suffoquerait. Ce paquet pose donc le MEME verrou, sur le MEME fichier :
// verrou du noyau sur `film_decode.lock` sous la racine du cache, detenteur decrit dans
// `film_decode.lock.json`. Les declarations qui fixent ce protocole (noms de fichier, forme du
// detenteur) sont identiques a celles de l'hote, texte pour texte (garde-rail de
// parite de l'hote).
//
// Chaque prise est liee a un test : un refus fait echouer le test en nommant le detenteur, et le
// verrou est rendu a la fin du test.
package decodelock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JGtm/LevelUp/libs/film/internal/atomicfile"
	"github.com/JGtm/LevelUp/libs/film/internal/filelock"
)

const (
	// soloFileName : le fichier du verrou OS, sous la racine du cache film. Il reste en place.
	soloFileName = "film_decode.lock"
	// soloHolderFileName : la description du detenteur, presente tant que le verrou est tenu.
	soloHolderFileName = soloFileName + ".json"
)

// soloHolder : ce qu'un detenteur ecrit dans sa description.
type soloHolder struct {
	Tool      string `json:"tool"`
	PID       int    `json:"pid"`
	MatchID   string `json:"matchId,omitempty"`
	StartedAt string `json:"startedAt"`
}

// errOccupe : un autre decodage tient le verrou.
var errOccupe = errors.New("un decodage de film est deja en cours sur cette machine")

// attente : cadence des nouvelles tentatives de [PrendreEnAttendant].
const attente = 500 * time.Millisecond

// Prendre prend le verrou de decodage sous `cacheRoot` pour la duree du test, ou fait echouer le
// test en nommant le detenteur. `outil` et `matchID` decrivent le detenteur.
func Prendre(t testing.TB, cacheRoot, outil, matchID string) {
	t.Helper()
	l, err := tenter(cacheRoot, outil, matchID)
	if err != nil {
		t.Fatalf("verrou de decodage : %v", err)
	}
	t.Cleanup(func() { rendre(t, l) })
}

// PrendreEnAttendant est [Prendre] qui attend jusqu'a `max` qu'un autre decodage rende le verrou.
func PrendreEnAttendant(ctx context.Context, t testing.TB, cacheRoot, outil, matchID string, max time.Duration) {
	t.Helper()
	limite := time.Now().Add(max)
	for {
		l, err := tenter(cacheRoot, outil, matchID)
		if err == nil {
			t.Cleanup(func() { rendre(t, l) })
			return
		}
		if !errors.Is(err, errOccupe) || !time.Now().Before(limite) {
			t.Fatalf("verrou de decodage (apres %s d'attente au plus) : %v", max, err)
		}
		select {
		case <-ctx.Done():
			t.Fatalf("attente du verrou de decodage interrompue (%v) : %v", ctx.Err(), err)
		case <-time.After(attente):
		}
	}
}

// verrou : un verrou tenu et la description de son detenteur.
type verrou struct {
	lock       *filelock.Lock
	holderPath string
}

// tenter prend le verrou sans attendre et decrit le detenteur, ou rend [errOccupe] (detenteur
// nomme) ou l'erreur de la prise.
func tenter(cacheRoot, outil, matchID string) (*verrou, error) {
	if cacheRoot == "" {
		return nil, errors.New("racine de cache vide")
	}
	if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
		return nil, fmt.Errorf("creation de %s : %w", cacheRoot, err)
	}
	holderPath := filepath.Join(cacheRoot, soloHolderFileName)
	lock, err := filelock.TryLock(filepath.Join(cacheRoot, soloFileName))
	if errors.Is(err, filelock.ErrLocked) {
		return nil, occupe(holderPath)
	}
	if err != nil {
		return nil, err
	}
	h := soloHolder{Tool: outil, PID: os.Getpid(), MatchID: matchID,
		StartedAt: time.Now().UTC().Format(time.RFC3339)}
	b, err := json.Marshal(h)
	if err == nil {
		err = atomicfile.WriteFileStrict(holderPath, b, 0o644)
	}
	if err != nil {
		return nil, errors.Join(fmt.Errorf("description du detenteur : %w", err), lock.Unlock())
	}
	return &verrou{lock: lock, holderPath: holderPath}, nil
}

// rendre retire la description puis rend le verrou ; un echec est dit au journal du test (le
// systeme rend de toute facon le verrou a la fin du processus).
func rendre(t testing.TB, l *verrou) {
	if err := os.Remove(l.holderPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Logf("description du detenteur non retiree : %v", err)
	}
	if err := l.lock.Unlock(); err != nil {
		t.Logf("verrou de decodage non rendu : %v", err)
	}
}

// occupe nomme le detenteur decrit, ou dit qu'il ne l'est pas encore.
func occupe(holderPath string) error {
	var h soloHolder
	b, err := os.ReadFile(holderPath) //nolint:gosec // chemin compose sous la racine du cache
	if err == nil {
		err = json.Unmarshal(b, &h)
	}
	if err != nil {
		return fmt.Errorf("%w : detenteur non decrit (%v), %s", errOccupe, err, holderPath)
	}
	return fmt.Errorf("%w : %s (pid %d) decode %q depuis %s", errOccupe, h.Tool, h.PID, h.MatchID, h.StartedAt)
}
