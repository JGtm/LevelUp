package filmproc

// solo_observe.go — LIRE QUI DECODE, SANS JAMAIS PRENDRE LE VERROU.
//
// Prendre le verrou pour « voir s'il est libre » ferait refuser un vrai decodeur qui arrive au
// meme instant : la lecture passe donc uniquement par la description du detenteur
// (`film_decode.lock.json`), ecrite apres la prise du verrou et retiree avant sa liberation. Une
// description laissee par un processus tue est reconnue par son pid, qui ne vit plus.

import (
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
)

// DecodeHolder : le detenteur decrit du verrou de decodage.
type DecodeHolder struct {
	Tool      string
	PID       int
	MatchID   string
	StartedAt string
}

// ReadDecodeHolder rend le detenteur decrit sous `cacheRoot` quand son processus vit. Fichier
// absent, illisible, vide, ou pid qui ne vit plus : (zero, false). Ne prend jamais le verrou.
func ReadDecodeHolder(cacheRoot string) (DecodeHolder, bool) {
	if cacheRoot == "" {
		return DecodeHolder{}, false
	}
	path := filepath.Join(cacheRoot, soloHolderFileName)
	b, err := os.ReadFile(path) //nolint:gosec // chemin compose sous la racine du cache
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.Warn("description du detenteur du verrou de decodage illisible", "err", err, "fichier", path)
		}
		return DecodeHolder{}, false
	}
	var h soloHolder
	if err := json.Unmarshal(b, &h); err != nil {
		slog.Warn("description du detenteur du verrou de decodage invalide", "err", err, "fichier", path)
		return DecodeHolder{}, false
	}
	if h.PID <= 0 || !processAlive(h.PID) {
		return DecodeHolder{}, false
	}
	return DecodeHolder(h), true
}

// DecodeObserver lit le detenteur du verrou de decodage sous une racine de cache fixe.
type DecodeObserver struct {
	cacheRoot string
}

// NewDecodeObserver construit l'observateur de la racine de cache film `cacheRoot`
// (`PathResolver.CacheRootDir()`).
func NewDecodeObserver(cacheRoot string) *DecodeObserver {
	return &DecodeObserver{cacheRoot: cacheRoot}
}

// DecodingMatchID rend le match que decode un processus vivant, "" sinon.
func (o *DecodeObserver) DecodingMatchID() string {
	if o == nil {
		return ""
	}
	h, ok := ReadDecodeHolder(o.cacheRoot)
	if !ok {
		return ""
	}
	return h.MatchID
}
