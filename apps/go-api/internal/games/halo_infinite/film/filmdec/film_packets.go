package filmdec

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"levelup/go-api/internal/analysis/filmsource"
)

// packetHeaderSize est la taille de l'en-tête d'un paquet de chunk film :
// [u16 type][u16 pad][u32 taille payload][u64 timestamp microsecondes].
const packetHeaderSize = 16

// PacketTypeDelta est le type de paquet portant les enregistrements delta d'entités
// (positions incluses) ; PacketTypeKeyframe porte l'état monde full-state.
const (
	PacketTypeDelta    uint16 = 0
	PacketTypeKeyframe uint16 = 2
)

// FilmPacket décrit un paquet dans un chunk film décompressé : bornes du payload dans le
// buffer du chunk et horodatage moteur.
type FilmPacket struct {
	// Index est le rang du paquet dans le chunk (0-based).
	Index int
	// Type est le type de paquet (cf. PacketType*).
	Type uint16
	// Start / Size délimitent le payload dans le buffer du chunk.
	Start, Size int
	// TimestampUS est l'horodatage moteur du paquet, en microsecondes (horloge du film).
	TimestampUS uint64
}

// Payload renvoie la tranche de payload du paquet dans le buffer du chunk.
func (p FilmPacket) Payload(chunk []byte) []byte { return chunk[p.Start : p.Start+p.Size] }

// ReadFilmChunk lit chunk_NN.bin dans dir et le décompresse si nécessaire (les chunks
// sont stockés en zlib brut ; certains dumps sont déjà décompressés).
//
// ENVELOPPE D2, HORS PRODUCTION (lot 1 de PLAN_CUISSON_PERF, 2026-09-02). Le chemin de cuisson
// charge le film UNE fois par `filmsource.LoadDir` et lit ses chunks par [FilmChunkAt] : plus
// aucune relecture disque par balayage. Cette fonction ne survit que pour les LECTEURS D'UN SEUL
// CHUNK — instruments de recherche et tests de `filmdec`, `replay`, `objectiveevents` — et pour
// `FindPackets`, qui balaye une RACINE de films et n'a pas de film a charger. L'inflate lui-meme
// vit desormais dans `filmsource` : un seul decompresseur dans le depot.
func ReadFilmChunk(dir string, chunk int) ([]byte, error) {
	path := filepath.Join(dir, fmt.Sprintf("chunk_%02d.bin", chunk))
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return filmsource.Inflate(raw), nil
}

// WalkPackets énumère les paquets d'un chunk décompressé. S'arrête au premier en-tête
// incohérent (fin de chunk ou padding) — les paquets déjà lus restent valides.
//
// MARCHEUR HORS PRODUCTION depuis le lot 1 (2026-09-02) : la grammaire de la chaine de cuisson
// vit dans `filmsource` (une seule, mesuree sur 1 378 films), et [FilmChunkAt] rend ses paquets.
// Celui-ci survit pour les lecteurs d'un chunk ISOLE (`FindPackets` et les instruments de
// recherche) et comme TEMOIN : `TestFilmChunkAtEgaleWalkPackets` compare les deux vues sur un
// vrai chunk, et c'est cette comparaison qui autorise la migration.
func WalkPackets(chunk []byte) []FilmPacket {
	var out []FilmPacket
	off := 0
	for off+packetHeaderSize <= len(chunk) {
		size := int(binary.LittleEndian.Uint32(chunk[off+4:]))
		if size < 0 || off+packetHeaderSize+size > len(chunk) {
			break
		}
		out = append(out, FilmPacket{
			Index:       len(out),
			Type:        binary.LittleEndian.Uint16(chunk[off:]),
			Start:       off + packetHeaderSize,
			Size:        size,
			TimestampUS: binary.LittleEndian.Uint64(chunk[off+8:]),
		})
		off += packetHeaderSize + size
	}
	return out
}

// CountFilmChunks compte les chunk_NN.bin présents dans dir (chunk_00 = registre exclu),
// en s'arrêtant au premier index manquant.
//
// ENVELOPPE D2, HORS PRODUCTION (lot 1, 2026-09-02) : la cuisson enumere les chunks d'un film
// DEJA CHARGE par [FilmChunkNumbers], qui reproduit exactement cette regle d'arret sans toucher
// le disque. Survit pour `FindPackets` (qui balaye une racine de films, pas un film) et pour les
// tests qui comparent les deux chemins.
func CountFilmChunks(dir string) int {
	n := 0
	for i := 1; ; i++ {
		if _, err := os.Stat(filepath.Join(dir, fmt.Sprintf("chunk_%02d.bin", i))); err != nil {
			return n
		}
		n = i
	}
}
