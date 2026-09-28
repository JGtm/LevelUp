package filmcache

// chunk.go — LE FICHIER D'UN CHUNK : son nom, et sa lecture VALIDEE (J2.3 du plan de suite de
// l'audit du decodeur de film, 2026-09-25).
//
// « PRESENT » N'EST PAS « COMPLET », A LA LECTURE NON PLUS. Le writer inscrit au manifeste la
// taille de chaque chunk (`size_bytes`, cf. write.go) ; tout lecteur du cache la compare au
// fichier et rend [ErrChunkTronque] quand elles different. Un manifeste historique sans taille
// reste lisible, sans controle (DT-4 : rien n'est rempli apres coup).
//
// UN SEUL LIEU. Le nom `chunk_NN.bin` et la validation vivent ici : le client Halo
// (`haloclient.LocalFilmCache`) et les outils lisent par [LireChunk] ou [CheminDuChunk] plutot
// que de recomposer le nom — garde-rail `internal/archlint/no_hardcoded_film_cache_dirs_test.go`.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// ErrChunkTronque : le fichier d'un chunk n'a pas la taille que son manifeste declare. Un
// lecteur en ligne le traite comme un chunk ABSENT du disque (reseau, puis reecriture par
// [Write], qui remplace un chunk de taille fausse).
type ErrChunkTronque struct {
	Index   int   // numero de fichier du chunk
	Attendu int64 // taille declaree au manifeste (`size_bytes`)
	Lu      int64 // taille du fichier sur disque
}

func (e *ErrChunkTronque) Error() string {
	return fmt.Sprintf("filmcache: chunk %d tronque (%d octets sur disque, %d au manifeste)", e.Index, e.Lu, e.Attendu)
}

// CheminDuChunk rend le chemin du fichier du chunk de NUMERO `numero` dans le repertoire de
// chunks `chunkDir` (celui que rend [ChunkDir], ou un jeu de donnees hors cache range pareil).
func CheminDuChunk(chunkDir string, numero int) string {
	return filepath.Join(chunkDir, chunkName(numero))
}

// LireChunk lit le chunk de NUMERO `numero` (le numero de fichier, pas la position au
// manifeste) d'un film du cache. Quand le manifeste du film porte la taille de ce chunk, un
// fichier d'une autre taille rend [ErrChunkTronque]. Un chunk absent rend une erreur qui
// enveloppe [os.ErrNotExist] ; un manifeste absent laisse lire sans controle.
func LireChunk(root, shortID string, numero int) ([]byte, error) {
	src, _, err := Open(root, shortID)
	if err != nil {
		return nil, err
	}
	var attendu int64
	if src != nil {
		attendu = src.tailleDuNumero(numero)
	}
	raw, err := lireChunkValide(CheminDuChunk(ChunkDir(root, shortID), numero), numero, attendu)
	if err != nil {
		return nil, fmt.Errorf("filmcache: chunk %d de %s : %w", numero, shortID, err)
	}
	return raw, nil
}

// lireChunkValide lit le fichier d'un chunk et le compare a la taille attendue (0 = inconnue).
func lireChunkValide(path string, numero int, attendu int64) ([]byte, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // chemin compose par CheminDuChunk
	if err != nil {
		return nil, err
	}
	if attendu > 0 && int64(len(raw)) != attendu {
		return nil, &ErrChunkTronque{Index: numero, Attendu: attendu, Lu: int64(len(raw))}
	}
	return raw, nil
}

// tailleDuNumero rend la taille que le manifeste declare pour le chunk de numero `numero`
// (0 = inconnue : manifeste historique, ou numero absent du manifeste).
func (s *Source) tailleDuNumero(numero int) int64 {
	for i, c := range s.chunks {
		if c.Index == numero {
			return s.tailles[i]
		}
	}
	return 0
}

// FilmComplet dit si un film du cache est COMPLET : manifeste FINALISE ([Finalise]), chaque chunk
// declare present sur disque, et a la taille declaree quand le manifeste la porte. « Present »
// n'est pas « complet » : un manifeste partiel ou un chunk tronque laissent le film a archiver
// (et [Write] le repare). Manifeste absent : false. Une erreur d'E/S ou un manifeste illisible
// remontent.
func FilmComplet(root, shortID string) (bool, error) {
	src, ok, err := Open(root, shortID)
	if err != nil || !ok {
		return false, err
	}
	if !Finalise(src.chunks, typeDeMeta) {
		return false, nil
	}
	manquant, err := src.controlerFichiers()
	var tronque *ErrChunkTronque
	if errors.As(err, &tronque) {
		return false, nil
	}
	return err == nil && !manquant, err
}

// controlerFichiers compare au manifeste chaque fichier de chunk : rend `manquant` quand un
// fichier declare est absent, et [ErrChunkTronque] au premier fichier d'une autre taille que
// celle declaree (0 = inconnue, aucun controle).
func (s *Source) controlerFichiers() (manquant bool, err error) {
	dir := ChunkDir(s.root, s.short)
	for i, c := range s.chunks {
		info, err := os.Stat(CheminDuChunk(dir, c.Index))
		if errors.Is(err, os.ErrNotExist) {
			manquant = true
			continue
		}
		if err != nil {
			return manquant, fmt.Errorf("filmcache: etat du chunk %d de %s : %w", c.Index, s.short, err)
		}
		if s.tailles[i] > 0 && info.Size() != s.tailles[i] {
			return manquant, &ErrChunkTronque{Index: c.Index, Attendu: s.tailles[i], Lu: info.Size()}
		}
	}
	return manquant, nil
}

// typeDeMeta : l'accesseur de type que [Finalise] recoit pour les entrees d'un manifeste lu.
func typeDeMeta(c types.ChunkMeta) int { return c.ChunkType }
