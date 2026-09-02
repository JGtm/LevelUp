package filmsource

// source.go — L'ENTREE DU PAQUET : LES CHUNKS BRUTS, ET RIEN D'AUTRE.
//
// Ce paquet ne sait pas TELECHARGER un film : c'est deliberement la responsabilite de l'appelant
// (le telechargement est le vrai cout, il est batchable et il a deja son chemin store-first). Il
// ne connait qu'une interface a deux methodes, ce qui rend le decodage testable a partir d'octets
// en memoire, sans disque et sans reseau. La forme est celle de `killsource.ChunkSource`, dont ce
// fichier reprend la doctrine — y compris la lecon ci-dessous.
//
// PIEGE HISTORIQUE, ET IL A COUTE UN KILL-FEED ENTIER : tout l'outillage de retro-ingenierie
// bornait la lecture au chunk 41. Un film BTB en compte 63, et son chunk HIGHLIGHT est le n62 —
// le kill-feed y etait purement introuvable (RE_LOG 7ter.52). Ici il n'y a AUCUNE borne haute :
// la source declare son compte, et tout est lu.

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Source : de quoi lire les chunks d'un film. Deux methodes, aucune hypothese sur l'origine des
// octets (disque, cache, memoire, objet distant deja materialise).
type Source interface {
	// NumChunks : nombre de chunks, index 0..NumChunks()-1.
	NumChunks() int
	// Chunk : les octets BRUTS du chunk `i`, compresses (zlib) ou non — la decompression est
	// faite ici, par [Load]. Un chunk absent se signale par une erreur ; un chunk vide est
	// accepte et ne rend aucun paquet.
	Chunk(i int) ([]byte, error)
}

// MemoryChunks : implementation triviale sur une tranche deja chargee. C'est la forme que prend
// naturellement un pipeline qui vient de telecharger les chunks.
type MemoryChunks [][]byte

// NumChunks implemente [Source].
func (m MemoryChunks) NumChunks() int { return len(m) }

// Chunk implemente [Source].
func (m MemoryChunks) Chunk(i int) ([]byte, error) {
	if i < 0 || i >= len(m) {
		return nil, fmt.Errorf("filmsource: chunk %d hors bornes (%d chunks)", i, len(m))
	}
	return m[i], nil
}

// dirSource : les chunks d'un repertoire `chunk_NN.bin`, lus a la demande.
//
// `nums[i]` est le NUMERO DE FICHIER du chunk a l'indice `i` (le NN de `chunk_NN.bin`), ou
// [ChunkNumberUnknown] quand le nom ne porte pas de numero lisible. C'est ce numero, et pas la
// position, que [LoadDir] publie en [ChunkMeta.Index] — cf. l'en-tete de paquet, section
// « L'INDEXATION DES CHUNKS ».
type dirSource struct {
	files []string
	nums  []int
}

// ChunkNumberUnknown : le numero d'un chunk dont le nom de fichier n'en porte pas de lisible.
// Ni le registre (0) ni un chunk de donnees (>= 1) : un consommateur qui trie par numero le
// laisse de cote plutot que de le confondre avec le chunk 0.
const ChunkNumberUnknown = -1

// DirSource : source lisant `<dir>/chunk_*.bin`, SANS BORNE HAUTE (cf. le piege du chunk 41/63 en
// tete de fichier). L'indice de la source est la POSITION DANS L'ORDRE TRIE, pas le numero du
// fichier : sur le cache complet, l'indice 0 est `chunk_00.bin`, le REGISTRE — mais une bobine de
// fixture qui commence a `chunk_01.bin` a son PREMIER chunk de donnees a l'indice 0. Le numero,
// lui, se retrouve dans les metadonnees que [LoadDir] synthetise.
//
// Utile pour rejouer un film mis en fixture, et pour la cuisson qui lit le cache disque ; un
// pipeline qui a deja les octets en main preferera [MemoryChunks].
func DirSource(dir string) (Source, error) { return newDirSource(dir) }

// newDirSource : [DirSource] rendant le type concret, pour que [LoadDir] atteigne les numeros.
//
// LE TRI EST NUMERIQUE, PAS LEXICOGRAPHIQUE, et c'est le meme ordre a une exception pres : au-dela
// de 99 chunks le nom passe a trois chiffres (`chunk_100.bin`) et l'ordre des octets placerait
// `chunk_100.bin` avant `chunk_11.bin`. Les noms sans numero lisible ferment la marche, tries
// entre eux par nom — l'ordre reste TOTAL et deterministe dans tous les cas.
func newDirSource(dir string) (*dirSource, error) {
	names, err := filepath.Glob(filepath.Join(dir, "chunk_*.bin"))
	if err != nil {
		return nil, fmt.Errorf("filmsource: lecture de %s: %w", dir, err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("filmsource: aucun chunk_NN.bin dans %s", dir)
	}
	sort.Slice(names, func(i, j int) bool {
		ni, nj := chunkNumberOf(names[i]), chunkNumberOf(names[j])
		if ni != nj {
			return sortKeyOfChunkNumber(ni) < sortKeyOfChunkNumber(nj)
		}
		return names[i] < names[j]
	})
	src := &dirSource{files: names, nums: make([]int, len(names))}
	for i, n := range names {
		src.nums[i] = chunkNumberOf(n)
	}
	return src, nil
}

// sortKeyOfChunkNumber : les chunks sans numero lisible ferment la marche.
func sortKeyOfChunkNumber(n int) int {
	if n == ChunkNumberUnknown {
		return math.MaxInt
	}
	return n
}

// chunkNumberOf lit le NN de `chunk_NN.bin`, ou [ChunkNumberUnknown]. Le numero est celui du
// FICHIER : c'est lui que le manifeste du film appelle `index`, et lui que les balayages
// publient dans leurs sorties (`Chunk: c`).
func chunkNumberOf(path string) int {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, ".bin")
	digits, ok := strings.CutPrefix(base, "chunk_")
	if !ok || digits == "" {
		return ChunkNumberUnknown
	}
	n, err := strconv.Atoi(digits)
	if err != nil || n < 0 {
		return ChunkNumberUnknown
	}
	return n
}

// NumChunks implemente [Source].
func (d *dirSource) NumChunks() int { return len(d.files) }

// Chunk implemente [Source].
func (d *dirSource) Chunk(i int) ([]byte, error) {
	if i < 0 || i >= len(d.files) {
		return nil, fmt.Errorf("filmsource: chunk %d hors bornes (%d chunks)", i, len(d.files))
	}
	b, err := os.ReadFile(d.files[i])
	if err != nil {
		return nil, fmt.Errorf("filmsource: %s: %w", d.files[i], err)
	}
	return b, nil
}
