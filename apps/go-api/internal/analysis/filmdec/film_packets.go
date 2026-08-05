package filmdec

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
func ReadFilmChunk(dir string, chunk int) ([]byte, error) {
	path := filepath.Join(dir, fmt.Sprintf("chunk_%02d.bin", chunk))
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return inflateChunk(raw), nil
}

// inflateChunk décompresse un chunk zlib ; renvoie l'entrée telle quelle si elle n'est pas
// compressée (ou si le flux est tronqué mais a produit des octets exploitables).
func inflateChunk(raw []byte) []byte {
	if len(raw) < 2 || raw[0] != 0x78 {
		return raw
	}
	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return raw
	}
	defer func() { _ = zr.Close() }()
	out, err := io.ReadAll(zr)
	if len(out) == 0 && err != nil {
		return raw
	}
	return out
}

// WalkPackets énumère les paquets d'un chunk décompressé. S'arrête au premier en-tête
// incohérent (fin de chunk ou padding) — les paquets déjà lus restent valides.
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
func CountFilmChunks(dir string) int {
	n := 0
	for i := 1; ; i++ {
		if _, err := os.Stat(filepath.Join(dir, fmt.Sprintf("chunk_%02d.bin", i))); err != nil {
			return n
		}
		n = i
	}
}
