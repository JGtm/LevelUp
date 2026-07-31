// tmp_sbsp_parse — PoC PHASE 3b : parser la STRUCTURE du tag sbsp (en-tête ucsh +
// tables AusarDocs) pour CIBLER les data-blocks de géométrie, au lieu de scanner.
//
// En-tête tag HI (CachedTag, AusarDocs) : magic@0 version@4 ... puis compteurs.
// Version 27 a un décalage -4 vs la doc (unknown8 = 4o). On AUTO-LOCK l'offset H
// du champ headerSize par l'invariant headerSize+dataSize == len(tag), puis :
//
//	dependencyCount@H-0x20 dataBlockCount@H-0x1C tagStructCount@H-0x18
//	dataReferenceCount@H-0x14 tagReferenceCount@H-0x10 stringIdCount@H-0x0C
//	stringTableSize@H-0x08 zonesetDataSize@H-0x04 headerSize@H dataSize@H+4 resData@H+8
//
// Tables (dans l'ordre) : deps(0x18) dataBlocks(0x10) structs(0x20) ...
// Data block : size@0 section@6(enum16) offset@8(u64). section 0=header, 1=tag data.
//
//	absolu dans le tag assemblé = (section==0 ? 0 : headerSize) + offset.
//
// Usage : CGO_ENABLED=1 go run ./cmd/tmp_sbsp_parse ["<module>"]
package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"sort"

	"levelup/go-api/internal/himodule"
)

const defMod = `D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/ds/levels/multi/catalyst/catalyst-rtx-new.module`

func u16(b []byte, o int) uint16 { return binary.LittleEndian.Uint16(b[o:]) }
func u32(b []byte, o int) int    { return int(binary.LittleEndian.Uint32(b[o:])) }
func u64(b []byte, o int) int    { return int(binary.LittleEndian.Uint64(b[o:])) }
func f32(b []byte, o int) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b[o:]))
}

type v3 struct{ x, y, z float64 }

type header struct {
	deps, blocks, structs, dataRefs, tagRefs, strIds int
	strTableSize, zonesetSize, headerSize, dataSize  int
	tablesStart                                      int
}

// lockHeader trouve H (offset de headerSize) tel que headerSize+dataSize==len(tag).
func lockHeader(tag []byte) (header, bool) {
	for H := 0x28; H <= 0x44; H += 4 {
		if H+8 > len(tag) {
			break
		}
		hs, ds := u32(tag, H), u32(tag, H+4)
		if hs > 0 && ds > 0 && hs+ds == len(tag) {
			h := header{
				deps:         u32(tag, H-0x20),
				blocks:       u32(tag, H-0x1C),
				structs:      u32(tag, H-0x18),
				dataRefs:     u32(tag, H-0x14),
				tagRefs:      u32(tag, H-0x10),
				strIds:       u32(tag, H-0x0C),
				strTableSize: u32(tag, H-0x08),
				zonesetSize:  u32(tag, H-0x04),
				headerSize:   hs,
				dataSize:     ds,
				tablesStart:  H + 0x18, // header AusarDocs = 0x50 (deps commencent là)
			}
			return h, true
		}
	}
	return header{}, false
}

func main() {
	modPath := defMod
	if len(os.Args) > 1 {
		modPath = os.Args[1]
	}
	m, err := himodule.Open(modPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	sbsps := m.Files("sbsp")
	sort.Slice(sbsps, func(i, j int) bool { return sbsps[i].UncompSize > sbsps[j].UncompSize })
	tag, err := m.Extract(sbsps[0])
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("=== sbsp décompressé %d o ===\n", len(tag))

	h, ok := lockHeader(tag)
	if !ok {
		fmt.Println("header non verrouillé (invariant headerSize+dataSize).")
		return
	}
	fmt.Printf("deps=%d dataBlocks=%d structs=%d dataRefs=%d tagRefs=%d strIds=%d\n",
		h.deps, h.blocks, h.structs, h.dataRefs, h.tagRefs, h.strIds)
	fmt.Printf("strTableSize=%#x zonesetSize=%#x headerSize=%#x dataSize=%#x tablesStart=%#x\n",
		h.strTableSize, h.zonesetSize, h.headerSize, h.dataSize, h.tablesStart)

	// tables : deps puis dataBlocks.
	depsEnd := h.tablesStart + h.deps*0x18
	blockTab := depsEnd
	fmt.Printf("table dataBlocks @ %#x (%d entrées × 0x10)\n", blockTab, h.blocks)

	type blk struct {
		idx, size, section, offset, abs int
	}
	var blocks []blk
	for i := 0; i < h.blocks; i++ {
		b := blockTab + i*0x10
		if b+0x10 > len(tag) {
			break
		}
		size := u32(tag, b)
		section := int(u16(tag, b+6))
		offset := u64(tag, b+8)
		abs := offset
		if section != 0 {
			abs = h.headerSize + offset
		}
		blocks = append(blocks, blk{i, size, section, offset, abs})
	}
	// les plus gros = candidats vertex/index buffers.
	sort.Slice(blocks, func(i, j int) bool { return blocks[i].size > blocks[j].size })
	fmt.Println("\n=== 12 plus gros data-blocks ===")
	fmt.Println("  idx     size  section   offset      abs")
	for i := 0; i < 12 && i < len(blocks); i++ {
		b := blocks[i]
		fmt.Printf("  %5d %8x   s%d    %8x  %8x\n", b.idx, b.size, b.section, b.offset, b.abs)
	}

	// le PLUS GROS block = buffer de géométrie. Plot des vertices int16 (stride 8).
	big := blocks[0]
	if big.abs >= 0 && big.abs+big.size <= len(tag) {
		seg := tag[big.abs : big.abs+big.size]
		fmt.Printf("\n=== block géométrie #%d (size=%#x abs=%#x) ===\n", big.idx, big.size, big.abs)
		analyzeFloat(seg)
		analyzeInt16(seg)
		plotInt16Verts(seg)
	}
}

// plotInt16Verts : trouve le sous-buffer de POSITIONS (int16×4, w≈constant) le plus
// long, et projette les positions brutes (int16, transform linéaire = forme préservée).
func plotInt16Verts(d []byte) {
	const stride = 8 // int16 x,y,z,w
	// trouve le plus long run où le 4e int16 (w) est ~constant ET (x,y,z) non dégénérés.
	bestRun, bestStart := 0, -1
	p := 0
	for p+stride <= len(d) {
		w0 := int16(u16(d, p+6))
		cur, start := 0, p
		q := p
		for q+stride <= len(d) {
			x, y, z, w := int16(u16(d, q)), int16(u16(d, q+2)), int16(u16(d, q+4)), int16(u16(d, q+6))
			if w != w0 || (x == 0 && y == 0 && z == 0) {
				break
			}
			cur++
			q += stride
		}
		if cur > bestRun {
			bestRun, bestStart = cur, start
		}
		if cur > 0 {
			p = q
		} else {
			p += stride
		}
	}
	fmt.Printf("  positions int16×4 : run=%d @%#x (w constant)\n", bestRun, bestStart)
	if bestRun < 100 {
		fmt.Println("  (pas de buffer de positions net)")
		return
	}
	var vs []v3
	for i := 0; i < bestRun; i++ {
		q := bestStart + i*stride
		vs = append(vs, v3{float64(int16(u16(d, q))), float64(int16(u16(d, q+2))), float64(int16(u16(d, q+4)))})
	}
	render2Dint(vs)
}

func render2Dint(vs []v3) {
	if len(vs) == 0 {
		return
	}
	mn := [3]float64{vs[0].x, vs[0].y, vs[0].z}
	mx := mn
	get := func(v v3, a int) float64 { return [3]float64{v.x, v.y, v.z}[a] }
	for _, v := range vs {
		for a := 0; a < 3; a++ {
			c := get(v, a)
			if c < mn[a] {
				mn[a] = c
			}
			if c > mx[a] {
				mx[a] = c
			}
		}
	}
	axn := []string{"X", "Y", "Z"}
	sp := []struct {
		a int
		s float64
	}{{0, mx[0] - mn[0]}, {1, mx[1] - mn[1]}, {2, mx[2] - mn[2]}}
	sort.Slice(sp, func(i, j int) bool { return sp[i].s > sp[j].s })
	ax, ay := sp[0].a, sp[1].a
	fmt.Printf("  bbox int16 X[%.0f,%.0f] Y[%.0f,%.0f] Z[%.0f,%.0f] → plan %s/%s\n",
		mn[0], mx[0], mn[1], mx[1], mn[2], mx[2], axn[ax], axn[ay])
	const W, H = 90, 44
	grid := make([][]int, H)
	for i := range grid {
		grid[i] = make([]int, W)
	}
	spanX, spanY := mx[ax]-mn[ax], mx[ay]-mn[ay]
	if spanX == 0 || spanY == 0 {
		return
	}
	for _, v := range vs {
		gx := int((get(v, ax) - mn[ax]) / spanX * (W - 1))
		gy := H - 1 - int((get(v, ay)-mn[ay])/spanY*(H-1))
		if gx >= 0 && gx < W && gy >= 0 && gy < H {
			grid[gy][gx]++
		}
	}
	ramp := []byte(" .:-=+*#%@")
	maxc := 0
	for _, r := range grid {
		for _, c := range r {
			if c > maxc {
				maxc = c
			}
		}
	}
	line := "+"
	for i := 0; i < W; i++ {
		line += "-"
	}
	line += "+"
	fmt.Println(line)
	for _, r := range grid {
		s := make([]byte, W)
		for x, c := range r {
			if c == 0 {
				s[x] = ' '
				continue
			}
			idx := 1 + int(float64(c)/float64(maxc)*float64(len(ramp)-2))
			if idx >= len(ramp) {
				idx = len(ramp) - 1
			}
			s[x] = ramp[idx]
		}
		fmt.Println("|" + string(s) + "|")
	}
	fmt.Println(line)
}

// analyzeFloat : meilleur stride de triplets float32 map-like, bbox.
func analyzeFloat(d []byte) {
	bestRun, bestStride, bestStart := 0, 0, 0
	for _, s := range []int{12, 16, 24, 32, 48, 64} {
		for phase := 0; phase < s && phase < 64; phase++ {
			cur, curStart := 0, 0
			for p := phase; p+12 <= len(d); p += s {
				x, y, z := f32(d, p), f32(d, p+4), f32(d, p+8)
				if real3(x, y, z, 4000) {
					if cur == 0 {
						curStart = p
					}
					cur++
					if cur > bestRun {
						bestRun, bestStride, bestStart = cur, s, curStart
					}
				} else {
					cur = 0
				}
			}
		}
	}
	fmt.Printf("  float32: meilleur run=%d stride=%d @%#x", bestRun, bestStride, bestStart)
	if bestRun > 20 {
		var mn, mx [3]float32
		first := true
		for i := 0; i < bestRun; i++ {
			p := bestStart + i*bestStride
			x, y, z := f32(d, p), f32(d, p+4), f32(d, p+8)
			if first {
				mn, mx = [3]float32{x, y, z}, [3]float32{x, y, z}
				first = false
			}
			for a, v := range []float32{x, y, z} {
				if v < mn[a] {
					mn[a] = v
				}
				if v > mx[a] {
					mx[a] = v
				}
			}
		}
		fmt.Printf("  bbox X[%.1f,%.1f] Y[%.1f,%.1f] Z[%.1f,%.1f]", mn[0], mx[0], mn[1], mx[1], mn[2], mx[2])
	}
	fmt.Println()
}

// analyzeInt16 : variance des colonnes int16 stride 6/8 (détecte buffer quantifié).
func analyzeInt16(d []byte) {
	// heuristique : compte les int16 non-nuls de magnitude moyenne (vertex quantif).
	nonzero := 0
	for p := 0; p+2 <= len(d); p += 2 {
		v := int16(u16(d, p))
		if v != 0 && v != -1 {
			nonzero++
		}
	}
	frac := float64(nonzero) / float64(len(d)/2)
	fmt.Printf("  int16 : %.0f%% non-nuls (buffer dense quantifié si élevé)\n", frac*100)
}

func real3(x, y, z float32, bound float64) bool {
	for _, v := range []float32{x, y, z} {
		a := float64(v)
		if math.IsNaN(a) || math.IsInf(a, 0) || a < -bound || a > bound {
			return false
		}
	}
	m := math.Max(math.Max(math.Abs(float64(x)), math.Abs(float64(y))), math.Abs(float64(z)))
	return m > 2.0
}
