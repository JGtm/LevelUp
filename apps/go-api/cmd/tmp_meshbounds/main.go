// tmp_meshbounds — trouve le bloc "meshes import info" (bbox par mesh, render_geometry)
// et rend les bounding-boxes en coords MONDE → plan 2D de la carte.
//
// Record (def IRTV mode.xml "meshes import info", stride 0x2C) :
//
//	+0x00 CRC int32 | +0x04 position bounds 0 vec3 (min) | +0x10 position bounds 1 vec3 (max)
//	+0x1C texcoord bounds 0 vec2 | +0x24 texcoord bounds 1 vec2
//
// On classe les data-blocks du tag : celui qui est un array propre de ces records.
//
// Usage : CGO_ENABLED=1 go run ./cmd/tmp_meshbounds ["<module>"]
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
const recSize = 0x2C

func u16(b []byte, o int) uint16 { return binary.LittleEndian.Uint16(b[o:]) }
func u32(b []byte, o int) int    { return int(binary.LittleEndian.Uint32(b[o:])) }
func u64(b []byte, o int) int    { return int(binary.LittleEndian.Uint64(b[o:])) }
func f32(b []byte, o int) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b[o:]))
}
func fin(v float32, lim float64) bool {
	a := float64(v)
	return !math.IsNaN(a) && !math.IsInf(a, 0) && a >= -lim && a <= lim
}

type bbox struct{ min, max [3]float32 }

// validRec teste un record @o+ao : min vec3 @ao, max vec3 @ao+12 ; fini, min<=max,
// range monde (tolère les meshes plats : extent 0 OK).
func validRec(d []byte, o, ao int) (bbox, bool) {
	var b bbox
	for a := 0; a < 3; a++ {
		mn, mx := f32(d, o+ao+a*4), f32(d, o+ao+12+a*4)
		if !fin(mn, 5000) || !fin(mx, 5000) || mx < mn {
			return b, false
		}
		b.min[a], b.max[a] = mn, mx
	}
	return b, true
}

// nonFlat : la bbox a une étendue réelle sur au moins un axe (exclut les records nuls).
func nonFlat(b bbox) bool {
	e := float32(0)
	for a := 0; a < 3; a++ {
		if b.max[a]-b.min[a] > e {
			e = b.max[a] - b.min[a]
		}
	}
	return e > 0.5
}

// ---- parse en-tête + data-blocks (cf tmp_sbsp_parse) ----
func parseBlocks(tag []byte) []struct{ abs, size int } {
	var H int
	for h := 0x28; h <= 0x44; h += 4 {
		if h+8 <= len(tag) && u32(tag, h) > 0 && u32(tag, h+4) > 0 && u32(tag, h)+u32(tag, h+4) == len(tag) {
			H = h
			break
		}
	}
	if H == 0 {
		return nil
	}
	deps := u32(tag, H-0x20)
	nBlocks := u32(tag, H-0x1C)
	headerSize := u32(tag, H)
	tablesStart := H + 0x18
	blockTab := tablesStart + deps*0x18
	var out []struct{ abs, size int }
	for i := 0; i < nBlocks; i++ {
		b := blockTab + i*0x10
		if b+0x10 > len(tag) {
			break
		}
		size := u32(tag, b)
		section := int(u16(tag, b+6))
		offset := u64(tag, b+8)
		abs := offset
		if section != 0 {
			abs = headerSize + offset
		}
		out = append(out, struct{ abs, size int }{abs, size})
	}
	return out
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
	blocks := parseBlocks(tag)
	fmt.Printf("tag %d o, %d data-blocks\n", len(tag), len(blocks))

	// compression info (sbsp.xml) = 0x54/record : flags2+pad2 + posMin@4 + posMax@0x10 + 4×uv + 2×unused.
	// On cherche le data-block qui est un array de tels records (tous valides, ≥1 non-plat).
	const st, ao = 0x54, 4
	var boxes []bbox
	bestBlock := -1
	for bi, b := range blocks {
		if b.size < st*2 || b.abs+b.size > len(tag) || b.size%st > 8 {
			continue
		}
		n := b.size / st
		ok, nf := 0, 0
		var bs []bbox
		for i := 0; i < n; i++ {
			if bb, v := validRec(tag, b.abs+i*st, ao); v {
				ok++
				bs = append(bs, bb)
				if nonFlat(bb) {
					nf++
				}
			}
		}
		// array propre = tous records valides, au moins la moitié non-plats, plus gros trouvé.
		if ok == n && nf >= n/2 && n > len(boxes) {
			boxes, bestBlock = bs, bi
		}
	}
	if len(boxes) < 2 {
		fmt.Println("bloc compression-info introuvable — TOUS les data-blocks par taille (top 40) :")
		sort.Slice(blocks, func(i, j int) bool { return blocks[i].size > blocks[j].size })
		for i, b := range blocks {
			if i >= 40 {
				break
			}
			div54 := ""
			if b.size%0x54 == 0 {
				div54 = " ×0x54"
			}
			fmt.Printf("  abs=%#-8x size=%#-7x (%d)%s\n", b.abs, b.size, b.size, div54)
		}
		return
	}
	fmt.Printf("compression info : block#%d, %d meshes (stride 0x54)\n", bestBlock, len(boxes))
	for i := 0; i < len(boxes) && i < 5; i++ {
		b := boxes[i]
		fmt.Printf("  mesh[%d] X[%.1f,%.1f] Y[%.1f,%.1f] Z[%.1f,%.1f]\n",
			i, b.min[0], b.max[0], b.min[1], b.max[1], b.min[2], b.max[2])
	}
	renderBoxes(boxes)
}

// renderBoxes remplit le rectangle 2D de chaque bbox (plan = 2 axes les plus larges).
func renderBoxes(boxes []bbox) {
	var mn, mx [3]float32
	mn = boxes[0].min
	mx = boxes[0].max
	for _, b := range boxes {
		for a := 0; a < 3; a++ {
			if b.min[a] < mn[a] {
				mn[a] = b.min[a]
			}
			if b.max[a] > mx[a] {
				mx[a] = b.max[a]
			}
		}
	}
	axn := []string{"X", "Y", "Z"}
	sp := []struct {
		a int
		s float32
	}{{0, mx[0] - mn[0]}, {1, mx[1] - mn[1]}, {2, mx[2] - mn[2]}}
	sort.Slice(sp, func(i, j int) bool { return sp[i].s > sp[j].s })
	ax, ay := sp[0].a, sp[1].a
	fmt.Printf("\nétendue monde X[%.1f,%.1f] Y[%.1f,%.1f] Z[%.1f,%.1f] ; plan %s/%s (up=%s)\n",
		mn[0], mx[0], mn[1], mx[1], mn[2], mx[2], axn[ax], axn[ay], axn[sp[2].a])

	const W, H = 96, 48
	grid := make([][]int, H)
	for i := range grid {
		grid[i] = make([]int, W)
	}
	spanX, spanY := mx[ax]-mn[ax], mx[ay]-mn[ay]
	if spanX == 0 || spanY == 0 {
		return
	}
	cell := func(v, lo, span float32, n int) int {
		return int(float64(v-lo) / float64(span) * float64(n-1))
	}
	for _, b := range boxes {
		x0 := cell(b.min[ax], mn[ax], spanX, W)
		x1 := cell(b.max[ax], mn[ax], spanX, W)
		y0 := cell(b.min[ay], mn[ay], spanY, H)
		y1 := cell(b.max[ay], mn[ay], spanY, H)
		for gy := y0; gy <= y1; gy++ {
			for gx := x0; gx <= x1; gx++ {
				ry := H - 1 - gy
				if gx >= 0 && gx < W && ry >= 0 && ry < H {
					grid[ry][gx]++
				}
			}
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
	bar := "+"
	for i := 0; i < W; i++ {
		bar += "-"
	}
	bar += "+"
	fmt.Println(bar)
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
	fmt.Println(bar)
}
