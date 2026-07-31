// tmp_killframe_sweep — THROWAWAY (FRONT 3) : pour CHAQUE kill du match (couvert
// par les chunks gameplay), localise la kill-FRAME (FRAME volumineuse coïncidant à
// ±10ms) et rapporte :
//   - sa taille vs médiane (signature de burst),
//   - l'opcode de tête (4 premiers octets),
//   - l'arme high-32 @bit44 (head-weapon hypothèse = arme du tueur figée),
//   - si une arme catalog 64-bit est isolée en tête (bit<128).
//
// But : valider que la kill-FRAME porte SYSTÉMATIQUEMENT une head-weapon, et la
// comparer à l'attribution v2 (held). On charge la liste des kills depuis l'oracle
// passé en dur (mêmes données que killer_victim_pairs).
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/analysis"
)

const cacheDir = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

// (ms, killerGT, v2weaponName)
var kills = []struct {
	ms       int
	killerGT string
	v2       string
}{
	{104995, "JGtm", "CQS48 Bulldog"},
	{111185, "Akatsuki", "Skewer"},
	{112869, "JGtm", "CQS48 Bulldog"},
	{115537, "IKE ILYA", "Disruptor"},
	{124713, "LORD PEINX13", "S7 Sniper"},
	{126949, "aldusbroncus", "S7 Sniper"},
	{128850, "JAVIERLOLITO540", "Needler"},
	{129000, "whiteknight2519", "Needler"},
	{141897, "Akatsuki", "(pi0/none)"},
	{147569, "LORD PEINX13", "M41 SPNKr"},
	{149471, "JGtm", "Disruptor"},
	{156529, "whiteknight2519", "S7 Sniper"},
	{156996, "aldusbroncus", "S7 Sniper"},
}

var chunkStart = []struct{ idx, startMS int }{
	{1, 0}, {2, 19995}, {3, 40000}, {4, 60003}, {5, 80006},
	{6, 100010}, {7, 120013}, {8, 140017}, {9, 160021},
}

func load(idx int) []byte {
	raw, _ := os.ReadFile(fmt.Sprintf("%s/chunk_%02d.bin", cacheDir, idx))
	if len(raw) >= 2 && raw[0] == 0x78 {
		zr, e := zlib.NewReader(bytes.NewReader(raw))
		if e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

type fp struct {
	us   uint64
	size int
	off  int
}

func walkFrames(data []byte) []fp {
	var out []fp
	off := 0
	for off+16 <= len(data) {
		typ := binary.LittleEndian.Uint16(data[off:])
		size := int(binary.LittleEndian.Uint32(data[off+4:]))
		us := binary.LittleEndian.Uint64(data[off+8:])
		if size < 0 || size > len(data) {
			break
		}
		if typ == 0 && off+16+size <= len(data) {
			out = append(out, fp{us, size, off + 16})
		}
		off += 16 + size
		if typ == 7 {
			break
		}
	}
	return out
}

func readU32BEAtBit(pl []byte, bit int) uint32 {
	var v uint32
	for i := 0; i < 32; i++ {
		bi := (bit + i) / 8
		if bi >= len(pl) {
			break
		}
		off := 7 - ((bit + i) % 8)
		v = v<<1 | uint32((pl[bi]>>uint(off))&1)
	}
	return v
}
func readU64BEAtBit(pl []byte, bit int) uint64 {
	var v uint64
	for i := 0; i < 64; i++ {
		bi := (bit + i) / 8
		if bi >= len(pl) {
			break
		}
		off := 7 - ((bit + i) % 8)
		v = v<<1 | uint64((pl[bi]>>uint(off))&1)
	}
	return v
}

func chunkForMS(ms int) int {
	for i := len(chunkStart) - 1; i >= 0; i-- {
		if ms >= chunkStart[i].startMS {
			return i
		}
	}
	return -1
}

func main() {
	high := map[uint32]string{}
	full := map[uint64]string{}
	for id, name := range analysis.WeaponIDToName {
		high[uint32(id>>32)] = name
		full[id] = name
	}
	// cache des frames+médiane par chunk
	type chunkData struct {
		data   []byte
		frames []fp
		first  uint64
		med    int
	}
	cache := map[int]*chunkData{}
	get := func(ci int) *chunkData {
		if c, ok := cache[ci]; ok {
			return c
		}
		d := load(chunkStart[ci].idx)
		fr := walkFrames(d)
		var first uint64
		med := 0
		if len(fr) > 0 {
			first = fr[0].us
			sizes := make([]int, len(fr))
			for i, f := range fr {
				sizes[i] = f.size
			}
			for i := 0; i < len(sizes); i++ {
				for j := i + 1; j < len(sizes); j++ {
					if sizes[j] < sizes[i] {
						sizes[i], sizes[j] = sizes[j], sizes[i]
					}
				}
			}
			med = sizes[len(sizes)/2]
		}
		c := &chunkData{d, fr, first, med}
		cache[ci] = c
		return c
	}

	fmt.Printf("%-9s %-16s %-14s %-7s %-9s %-7s %s\n", "kill_ms", "killer", "v2(held)", "Δms", "frameSz", "×méd", "head-weapon@bit44 / opcode")
	for _, k := range kills {
		ci := chunkForMS(k.ms)
		if ci < 0 {
			continue
		}
		c := get(ci)
		if len(c.frames) == 0 {
			continue
		}
		bestI, bestD := -1, 1<<30
		for i, f := range c.frames {
			ms := chunkStart[ci].startMS + int(int64(f.us-c.first)/1000)
			d := ms - k.ms
			if d < 0 {
				d = -d
			}
			if d < bestD {
				bestD, bestI = d, i
			}
		}
		f := c.frames[bestI]
		pl := c.data[f.off : f.off+f.size]
		op := pl[:4]
		hw := "-"
		if v := readU32BEAtBit(pl, 44); high[v] != "" {
			hw = fmt.Sprintf("%s(0x%08x)", high[v], v)
		}
		// arme 64-bit complète en tête (bit<200)
		headFull := ""
		for bit := 0; bit < 200 && bit+64 <= len(pl)*8; bit++ {
			if n, ok := full[readU64BEAtBit(pl, bit)]; ok {
				headFull = fmt.Sprintf(" [full %s @bit%d]", n, bit)
				break
			}
		}
		fmt.Printf("%-9d %-16s %-14s %-7d %-9d ×%-5.1f %-30s op=% x%s\n",
			k.ms, k.killerGT, k.v2, bestD, f.size, float64(f.size)/float64(max(c.med, 1)), hw, op, headFull)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
