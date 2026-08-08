// Package filmcache — LE cache film sur disque, lu par un seul morceau de code.
//
// Disposition, tenue par les repertoires deja presents :
//
//	<racine>/film_manifests/<short8>.json      le manifeste (index, type et debut des chunks)
//	<racine>/film_chunks/<short8>/chunk_NN.bin les chunks BRUTS
//
// `short8` est la forme courte du match_id (title.FilmShortMatchID) : c'est la cle du
// cache, pas l'uuid complet que manipule le reste de l'application.
//
// # Pourquoi ce paquet existe
//
// La meme source disque etait ecrite une fois dans `cmd/diag_weapons_v3` et une fois dans
// les tests d'`objectiveevents`. Un troisieme outil en avait besoin : a la troisieme
// copie, la regle du depot impose de centraliser ET de poser un garde-rail
// (filmcache_guard_test.go). Une disposition de cache dupliquee derive en silence — le
// jour ou le nom des chunks change, deux lecteurs sur trois cessent de trouver le film et
// se contentent de rendre « rien a decoder ».
//
// # Ce paquet ne decode rien
//
// Il rend des octets et un index. Le decodage vit dans `analysis/objectiveevents` et
// `analysis/filmdec` — d'ou l'absence de dependance dans l'autre sens.
package filmcache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// manifestsDir et chunksDir sont les deux sous-dossiers du cache. Nommes ici, et nulle
// part ailleurs.
const (
	manifestsDir = "film_manifests"
	chunksDir    = "film_chunks"
)

// Source implemente objectiveevents.FilmSource sur le cache disque.
type Source struct {
	root   string
	short  string
	chunks []objectiveevents.ChunkMeta
}

type manifestJSON struct {
	Chunks []struct {
		Index     int `json:"index"`
		ChunkType int `json:"chunk_type"`
		StartMS   int `json:"start_ms"`
	} `json:"chunks"`
}

// Open charge le manifeste d'un film.
//
// (nil, false, nil) quand le manifeste n'est pas la : un film absent du cache est le cas
// NOMINAL (le cache est partiel et local), pas une panne. Une erreur non nulle signale un
// manifeste present mais illisible — celle-la, il faut la voir.
func Open(root, shortID string) (*Source, bool, error) {
	path := ManifestPath(root, shortID)
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("manifeste de film illisible (%s) : %w", path, err)
	}
	var mf manifestJSON
	if err := json.Unmarshal(raw, &mf); err != nil {
		return nil, false, fmt.Errorf("manifeste de film invalide (%s) : %w", path, err)
	}
	src := &Source{root: root, short: shortID, chunks: make([]objectiveevents.ChunkMeta, 0, len(mf.Chunks))}
	for _, c := range mf.Chunks {
		src.chunks = append(src.chunks, objectiveevents.ChunkMeta{
			Index: c.Index, ChunkType: c.ChunkType, StartMS: c.StartMS,
		})
	}
	return src, true, nil
}

// Chunks rend l'index du manifeste.
func (s *Source) Chunks() []objectiveevents.ChunkMeta { return s.chunks }

// ChunkData rend les octets BRUTS d'un chunk, ou (nil, false) s'il manque.
func (s *Source) ChunkData(index int) ([]byte, bool) {
	raw, err := os.ReadFile(filepath.Join(ChunkDir(s.root, s.short), chunkName(index)))
	if err != nil {
		return nil, false
	}
	return raw, true
}

// ChunkDir rend le repertoire des chunks d'un film. C'est ce chemin qu'attendent les
// balayages hors ligne de `analysis/filmdec` et `analysis/replay`, qui lisent le dossier
// directement plutot que par l'interface.
func ChunkDir(root, shortID string) string {
	return filepath.Join(root, chunksDir, shortID)
}

// ChunksRoot rend le repertoire qui contient UN dossier par film cache. Sert a enumerer
// ce qui est disponible localement.
func ChunksRoot(root string) string { return filepath.Join(root, chunksDir) }

// ManifestPath rend le chemin du manifeste d'un film.
func ManifestPath(root, shortID string) string {
	return filepath.Join(root, manifestsDir, shortID+".json")
}

func chunkName(index int) string { return fmt.Sprintf("chunk_%02d.bin", index) }
