// Package himodule lit les archives .module Halo Infinite (mohd v53) et extrait
// les tags (décompression Kraken via internal/ooz). Offline-pur, universel.
//
// Layout VÉRIFIÉ (hand-decode catalyst/ds, cross-checké table des blocs) :
//
//	header 0x48 ; entrée fichier stride 0x58 :
//	  +0x0A blockCount u16 | +0x0C firstBlockIndex u32 | +0x14 group fourCC u32 LE
//	  +0x18 data_offset u32 | +0x20 compressedSize u32 | +0x24 uncompressedSize u32
//	bloc (20o) : +0 compOff +4 compSize +8 decompOff +12 decompSize +16 bCompressed
//	  compOff RELATIF au data_offset du fichier.
//
// En-tête (0x48 o) : 0x10 fileCount | 0x20 resourceIndex | 0x24 stringsSize
//
//	0x28 resourceCount | 0x2C blockCount | 0x38 hd1Delta (u64) | 0x40 dataSize (u64).
//
// dataBase (début de la section données) : `tailleFichier - dataSize` quand dataSize
// est renseigné, sinon fin des tables alignée sur 4 Ko. L'ancienne règle
// `tailleFichier - max(data_offset+compressedSize)` est FAUSSE dès qu'un module a un
// compagnon `_hd1` ou du remplissage final (mesuré : 1965/2047 entrées en échec sur
// any/levels/multi/catalyst, 100 % sur pc/levels/multi/ridgeline).
package himodule

import (
	"encoding/binary"
	"fmt"
	"os"

	"levelup/go-api/internal/ooz"
)

const (
	headerSize     = 0x48
	entryStride    = 0x58
	blockEntrySize = 20

	offBlockCount = 0x0A
	offFirstBlock = 0x0C
	offGroup      = 0x14
	offDataOffset = 0x18
	offGlobalID   = 0x28
	offCompSize   = 0x20
	offUncompSize = 0x24

	offHdrStringsSize = 0x24
	offHdrResCount    = 0x28
	offHdrDataSize    = 0x40

	// dataAlign : la section données commence sur une frontière de 4 Ko.
	dataAlign = 0x1000
)

// Module est une archive .module mappée en mémoire.
type Module struct {
	data      []byte
	fileCount int
	numBlocks int
	blockOff  int
	dataBase  int
	// hd1 est le contenu du compagnon `.module_hd1`, s'il existe. Les entrees dont le
	// drapeau UseHd1 est arme y pointent -- sur ridgeline c'est un fichier de 206 Mo que
	// personne n'avait jamais ouvert.
	hd1     []byte
	hd1Base int
}

// File décrit une entrée fichier (tag ou ressource).
type File struct {
	Index      int
	Group      string // fourCC (ex. "sbsp", "mode"), ou "" si non imprimable
	FirstBlock int
	BlockCount int
	// DataOffset est un entier 48 BITS, pas un u32 : les deux octets de poids fort du
	// mot de 64 bits a +0x18 sont des DRAPEAUX. Lire un u32 casse toute archive de plus
	// de 4 Go (globals-rtx-new fait 7,78 Go) -- symptome : ooz retourne -1 partout.
	DataOffset int64
	// Flags : bit 0 = UseHd1, la donnee vit dans le compagnon `.module_hd1`.
	Flags      uint16
	CompSize   int
	UncompSize int
	// GlobalID est l'identifiant global de tag (+0x28). C'est la cle de jointure avec les
	// references de tag d'un autre tag.
	GlobalID uint32
}

// UseHd1 dit si la donnee de cette entree vit dans le fichier compagnon `.module_hd1`.
func (f File) UseHd1() bool { return f.Flags&1 != 0 }

func u16(b []byte, o int) uint16 { return binary.LittleEndian.Uint16(b[o:]) }
func u32(b []byte, o int) uint32 { return binary.LittleEndian.Uint32(b[o:]) }
func u64(b []byte, o int) uint64 { return binary.LittleEndian.Uint64(b[o:]) }

func fourCC(v uint32) string {
	b := []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
	for _, c := range b {
		if c < 0x20 || c > 0x7e {
			return ""
		}
	}
	return string(b)
}

// Open lit et indexe un .module.
func Open(path string) (*Module, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < headerSize || string(data[:4]) != "mohd" {
		return nil, fmt.Errorf("himodule: magic mohd absent (%s)", path)
	}
	m := &Module{
		data:      data,
		fileCount: int(u32(data, 0x10)),
		numBlocks: int(u32(data, 0x2C)),
	}
	m.dataBase = dataBase(data, m.fileCount, m.numBlocks)
	m.blockOff = lockBlockTable(data, m.numBlocks, m.dataBase, headerSize+m.fileCount*entryStride)
	if m.blockOff < 0 {
		return nil, fmt.Errorf("himodule: table des blocs introuvable")
	}
	m.loadHd1(path)
	return m, nil
}

// loadHd1 ouvre le compagnon `.module_hd1` et CALIBRE sa base de donnees.
//
// La base n'est pas postulee : on essaie les candidats plausibles et on retient celui qui
// maximise le nombre d'entrees hd1 dont la premiere lecture est DANS les bornes ET dont le
// premier bloc porte un en-tete Kraken (nibble haut 0x8). Une base fausse echoue en masse,
// donc le critere separe nettement -- et s'il ne separe pas, on n'active PAS le compagnon
// plutot que de servir des octets pris au hasard.
func (m *Module) loadHd1(path string) {
	raw, err := os.ReadFile(path + "_hd1")
	if err != nil || len(raw) == 0 {
		return
	}
	var ents []File
	for i := 0; i < m.fileCount && len(ents) < 400; i++ {
		if f := m.file(i); f.UseHd1() && f.CompSize > 0 {
			ents = append(ents, f)
		}
	}
	if len(ents) == 0 {
		return
	}
	best, bestN := -1, 0
	for _, base := range hd1BaseCandidates(m.data, len(raw)) {
		n := 0
		for _, f := range ents {
			off := base + int(f.DataOffset)
			if off >= 0 && off+f.CompSize <= len(raw) && raw[off]&0xF0 == 0x80 {
				n++
			}
		}
		if n > bestN {
			bestN, best = n, base
		}
	}
	// Seuil : la moitie des entrees testees. En dessous, la base n'est pas etablie.
	if best >= 0 && bestN*2 >= len(ents) {
		m.hd1, m.hd1Base = raw, best
	}
}

// hd1BaseCandidates enumere les bases plausibles du compagnon : 0, la valeur derivee de
// hd1Delta (+0x38 de l'en-tete), et l'alignement 4 Ko de cette derniere.
func hd1BaseCandidates(head []byte, hd1Len int) []int {
	delta := int(u64(head, 0x38))
	c := []int{0, delta, hd1Len - delta}
	if delta > 0 {
		c = append(c, (delta+dataAlign-1)&^(dataAlign-1))
	}
	out := c[:0]
	seen := map[int]bool{}
	for _, v := range c {
		if v >= 0 && v < hd1Len && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// srcFor rend le tampon et la base a employer pour une entree : le compagnon si son
// drapeau UseHd1 est arme et que le compagnon a ete calibre, sinon l'archive principale.
func (m *Module) srcFor(f File) ([]byte, int, error) {
	if f.UseHd1() {
		if m.hd1 == nil {
			return nil, 0, fmt.Errorf("himodule: entree en hd1 mais compagnon absent ou non calibre")
		}
		return m.hd1, m.hd1Base, nil
	}
	return m.data, m.dataBase, nil
}

// dataBase : début de la section données. `tailleFichier - dataSize` quand l'en-tête
// renseigne dataSize ; sinon fin des tables (entrées + strings + resources + blocs)
// alignée sur 4 Ko. Les deux règles coïncident quand les deux sont applicables.
func dataBase(data []byte, fileCount, numBlocks int) int {
	if ds := int(u64(data, offHdrDataSize)); ds > 0 && ds <= len(data) {
		return len(data) - ds
	}
	end := headerSize + fileCount*entryStride +
		int(u32(data, offHdrStringsSize)) + int(u32(data, offHdrResCount))*4 +
		numBlocks*blockEntrySize
	return (end + dataAlign - 1) &^ (dataAlign - 1)
}

// lockBlockTable : la table des blocs précède immédiatement la section données. On part
// donc de (dataBase - numBlocks*20) et on descend par pas de 4 jusqu'à la fin de la
// table des entrées fichier, en validant bCompressed∈{0,1} et compSize plausible.
func lockBlockTable(data []byte, numBlocks, base, floor int) int {
	if numBlocks <= 0 {
		return floor
	}
	start := base - numBlocks*blockEntrySize
	if start > len(data)-numBlocks*blockEntrySize {
		start = len(data) - numBlocks*blockEntrySize
	}
	for off := start; off >= floor; off -= 4 {
		if plausibleBlockTable(data, numBlocks, off) {
			return off
		}
	}
	return -1
}

func plausibleBlockTable(data []byte, numBlocks, off int) bool {
	if off < 0 || off+numBlocks*blockEntrySize > len(data) {
		return false
	}
	for i := 0; i < numBlocks; i++ {
		b := off + i*blockEntrySize
		if u32(data, b+16) > 1 {
			return false
		}
		if cs := u32(data, b+4); cs == 0 || cs > 0x4000000 {
			return false
		}
	}
	return true
}

func (m *Module) file(i int) File {
	e := headerSize + i*entryStride
	return File{
		Index:      i,
		Group:      fourCC(u32(m.data, e+offGroup)),
		FirstBlock: int(u32(m.data, e+offFirstBlock)),
		BlockCount: int(u16(m.data, e+offBlockCount)),
		DataOffset: int64(u64(m.data, e+offDataOffset) & 0x0000FFFFFFFFFFFF),
		Flags:      uint16(u64(m.data, e+offDataOffset) >> 48),
		CompSize:   int(u32(m.data, e+offCompSize)),
		UncompSize: int(u32(m.data, e+offUncompSize)),
		GlobalID:   u32(m.data, e+offGlobalID),
	}
}

// Files renvoie toutes les entrées d'un groupe (ex. "sbsp"). Group="" → toutes.
func (m *Module) Files(group string) []File {
	var out []File
	for i := 0; i < m.fileCount; i++ {
		f := m.file(i)
		if group == "" || f.Group == group {
			out = append(out, f)
		}
	}
	return out
}

// Extract décompresse le contenu d'un fichier (assemblage de ses blocs, ou blob
// unique pour les resources à blockCount==0).
func (m *Module) Extract(f File) ([]byte, error) {
	// Resource : pas de blocs → un seul blob compressé à data_offset.
	buf, dbase, err := m.srcFor(f)
	if err != nil {
		return nil, err
	}
	if f.BlockCount == 0 {
		base := dbase + int(f.DataOffset)
		if base < 0 || base+f.CompSize > len(buf) {
			return nil, fmt.Errorf("himodule: resource hors borne")
		}
		src := buf[base : base+f.CompSize]
		if f.CompSize == f.UncompSize {
			out := make([]byte, f.UncompSize)
			copy(out, src)
			return out, nil
		}
		return ooz.Decompress(src, f.UncompSize)
	}
	out := make([]byte, f.UncompSize)
	for i := f.FirstBlock; i < f.FirstBlock+f.BlockCount; i++ {
		b := m.blockOff + i*blockEntrySize
		co, cs := int(u32(m.data, b)), int(u32(m.data, b+4))
		dorel, ds := int(u32(m.data, b+8)), int(u32(m.data, b+12))
		comp := u32(m.data, b+16)
		base := dbase + int(f.DataOffset) + co
		if base < 0 || base+cs > len(buf) || dorel+ds > len(out) {
			return nil, fmt.Errorf("himodule: bloc %d hors borne", i)
		}
		src := buf[base : base+cs]
		if comp == 0 {
			copy(out[dorel:dorel+ds], src)
			continue
		}
		dec, err := ooz.Decompress(src, ds)
		if err != nil {
			return nil, fmt.Errorf("himodule: bloc %d: %w", i, err)
		}
		copy(out[dorel:dorel+ds], dec)
	}
	return out, nil
}
