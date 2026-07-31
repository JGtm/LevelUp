package killsource

// chunks.go — L ENTREE DU PAQUET : LES CHUNKS, ET RIEN D AUTRE.
//
// Un film Theater est une suite de chunks numerotes, chacun eventuellement compresse en zlib.
// Ce paquet ne sait pas les TELECHARGER : c est deliberement la responsabilite de l appelant
// (le telechargement est le vrai cout, il est batchable et il a deja son chemin store-first).
// Il ne connait qu une interface a deux methodes, ce qui rend le decodage testable a partir
// d octets en memoire, sans disque et sans reseau.
//
// PIEGE HISTORIQUE, ET IL A COUTE UN KILL-FEED ENTIER : tout l outillage de RE bornait la
// lecture au chunk 41. Un film BTB en compte 63, et son chunk HIGHLIGHT est le n62 — le
// kill-feed y etait purement introuvable (RE_LOG 7ter.52). Ici il n y a AUCUNE borne : la
// source declare son compte, et tout est lu.

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// ChunkSource : de quoi lire les chunks d un film. Deux methodes, aucune hypothese sur l origine
// des octets (disque, cache, memoire, objet distant deja materialise).
type ChunkSource interface {
	// NumChunks : nombre de chunks, index 0..NumChunks()-1.
	NumChunks() int
	// Chunk : les octets BRUTS du chunk `i`, compresses ou non — la decompression est faite ici.
	// Un chunk absent se signale par une erreur ; un chunk vide est accepte et ignore.
	Chunk(i int) ([]byte, error)
}

// MemoryChunks : implementation triviale sur une tranche deja chargee. C est la forme que prend
// naturellement un pipeline qui vient de telecharger les chunks.
type MemoryChunks [][]byte

func (m MemoryChunks) NumChunks() int { return len(m) }

func (m MemoryChunks) Chunk(i int) ([]byte, error) {
	if i < 0 || i >= len(m) {
		return nil, fmt.Errorf("killsource: chunk %d hors bornes (%d chunks)", i, len(m))
	}
	return m[i], nil
}

// dirChunks : les chunks d un repertoire `chunk_NN.bin`, lus a la demande.
type dirChunks struct {
	dir   string
	files []string
}

// DirChunks : source lisant `<dir>/chunk_NN.bin`, NN sur deux chiffres, sans borne haute.
// Utile pour rejouer un film mis en fixture ; un pipeline de production preferera
// [MemoryChunks].
func DirChunks(dir string) (ChunkSource, error) {
	names, err := filepath.Glob(filepath.Join(dir, "chunk_*.bin"))
	if err != nil {
		return nil, fmt.Errorf("killsource: lecture de %s: %w", dir, err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("killsource: aucun chunk_NN.bin dans %s", dir)
	}
	sort.Strings(names)
	return &dirChunks{dir: dir, files: names}, nil
}

func (d *dirChunks) NumChunks() int { return len(d.files) }

func (d *dirChunks) Chunk(i int) ([]byte, error) {
	if i < 0 || i >= len(d.files) {
		return nil, fmt.Errorf("killsource: chunk %d hors bornes (%d chunks)", i, len(d.files))
	}
	b, err := os.ReadFile(d.files[i])
	if err != nil {
		return nil, fmt.Errorf("killsource: %s: %w", d.files[i], err)
	}
	return b, nil
}

// inflate : decompresse un chunk zlib. Une entree deja decompressee traverse telle quelle, et
// un flux tronque rend ce qui a pu etre lu — un film Theater se termine parfois net.
func inflate(raw []byte) []byte {
	if len(raw) < 2 || raw[0] != 0x78 {
		return raw
	}
	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return raw
	}
	defer func() { _ = zr.Close() }()
	dec, err := io.ReadAll(zr)
	if err != nil && len(dec) == 0 {
		return raw
	}
	return dec
}

// packet : un paquet de replication, tel qu il se presente dans un chunk decompresse.
// En-tete de 16 octets LITTLE-ENDIAN : [u16 type][2 o][u32 taille][u64 horodatage].
type packet struct {
	chunk, idx int
	typ        int
	ts         uint64
	payload    []byte
}

// packetType0 / packetTypeKeyframe / packetTypeBotMeta : les trois seuls types lus ici.
const (
	packetType0        = 0  // replication : events + records ECS. Les morts y sont 93/93.
	packetTypeKeyframe = 2  // declaration complete du monde a un instant
	packetTypeBotMeta  = 12 // BOT_METADATA : nbBots, slot, identifiant, nom (RE_LOG 7ter.62)
)

// film : les octets d un film, deja decompresses, prets a decoder.
type film struct {
	chunks  [][]byte // par index de chunk, decompresses
	packets []packet
	t0      []packet // paquets type-0, tries par horodatage
	tsBase  uint64
}

// loadFilm : decompresse tous les chunks et decoupe les paquets. Une seule lecture de la source.
func loadFilm(src ChunkSource) (*film, error) {
	n := src.NumChunks()
	if n == 0 {
		return nil, ErrNoChunk
	}
	f := &film{chunks: make([][]byte, n)}
	for ch := 0; ch < n; ch++ {
		raw, err := src.Chunk(ch)
		if err != nil {
			return nil, err
		}
		d := inflate(raw)
		f.chunks[ch] = d
		f.packets = append(f.packets, splitPackets(d, ch)...)
	}
	for i := range f.packets {
		if f.packets[i].typ == packetType0 {
			f.t0 = append(f.t0, f.packets[i])
		}
	}
	if len(f.t0) == 0 {
		return nil, ErrNoPacket
	}
	sort.Slice(f.t0, func(i, j int) bool { return f.t0[i].ts < f.t0[j].ts })
	f.tsBase = f.t0[0].ts
	return f, nil
}

// splitPackets : decoupage d un chunk decompresse en paquets.
func splitPackets(d []byte, ch int) []packet {
	var out []packet
	off, k := 0, 0
	for off+16 <= len(d) {
		typ := int(binary.LittleEndian.Uint16(d[off:]))
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		out = append(out, packet{chunk: ch, idx: k, typ: typ, ts: ts, payload: d[off+16 : off+16+sz]})
		off += 16 + sz
		k++
	}
	return out
}

// ms : instant d un paquet, en millisecondes depuis le premier paquet type-0 du film.
func (f *film) ms(p *packet) int { return int((p.ts - f.tsBase) / 1000) }

// hasEvents : le paquet porte-t-il une liste d evenements ? Le bit 1 du payload le dit.
func hasEvents(p *packet) bool { return bitAt(p.payload, 1) != 0 }

// bitAt : lecture MSB-first d un bit. Hors tampon = 0, comme le lecteur de bits du moteur.
func bitAt(d []byte, p int) int {
	if p < 0 || p>>3 >= len(d) {
		return 0
	}
	return int(d[p>>3]>>uint(7-(p&7))) & 1
}

// bits32 : lecture MSB-first de 32 bits a la position `p`.
func bits32(d []byte, p int) uint32 {
	i, sh := p>>3, uint(p&7)
	var v uint64
	for k := 0; k < 5; k++ {
		v <<= 8
		if i+k < len(d) {
			v |= uint64(d[i+k])
		}
	}
	return uint32(v >> (8 - sh))
}

// bitsN : lecture MSB-first de n bits (n <= 8) a la position `p`.
func bitsN(d []byte, p, n int) int {
	v := 0
	for k := 0; k < n; k++ {
		v = v<<1 | bitAt(d, p+k)
	}
	return v
}
