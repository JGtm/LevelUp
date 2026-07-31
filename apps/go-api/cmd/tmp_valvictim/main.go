package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"levelup/go-api/internal/analysis"
	"os"
	"sort"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const kc = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/killcapture.bin`
const sfx = uint32(0x42c9679f)

var h32 = map[uint32]string{}

func inflate(p string) []byte {
	raw, _ := os.ReadFile(p)
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}
func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}
func idxOf(h uint32) int {
	if h < 0xE1500000 || h > 0xE1600000 {
		return -1
	}
	return int((h - 0xE1500000) / 0x10002)
}
func main() {
	for id := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = analysis.WeaponIDToName[id]
	}
	type R struct {
		pl  []byte
		atk int
	}
	var recs []R
	for ch := 0; ch <= 27; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		o := 0
		for o+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[o:])
			sz := int(binary.LittleEndian.Uint32(d[o+4:]))
			if sz <= 0 || o+16+sz > len(d) {
				break
			}
			pl := d[o+16 : o+16+sz]
			o += 16 + sz
			if typ != 0 || len(pl) == 0 || pl[0] != 0xd2 {
				continue
			}
			bp := 41
			if bitsAt(pl, bp, 1) == 1 {
				bp++
			} else {
				bp += 3
			}
			f := uint32(bitsAt(pl, bp, 32))
			if _, ok := h32[f]; ok && uint32(bitsAt(pl, bp+32, 32)) == sfx {
				recs = append(recs, R{pl, int(bitsAt(pl, 36, 5)) >> 1})
			}
		}
	}
	kd, _ := os.ReadFile(kc)
	type kp struct{ k, v int }
	var kills []kp
	for o := 0; o+16 <= len(kd); o += 16 {
		ki := idxOf(binary.LittleEndian.Uint32(kd[o:]))
		vi := idxOf(binary.LittleEndian.Uint32(kd[o+4:]))
		if ki < 0 || vi < 0 || ki == vi {
			continue
		}
		kills = append(kills, kp{ki, vi})
	}
	type res struct{ bp, hit int }
	var rs []res
	for vbp := 0; vbp+5 <= 200; vbp++ {
		pairs := map[[2]int]bool{}
		for _, r := range recs {
			vic := int(bitsAt(r.pl, vbp, 5)) >> 1
			pairs[[2]int{r.atk, vic}] = true
		}
		hit := 0
		for _, k := range kills {
			if pairs[[2]int{k.k, k.v}] {
				hit++
			}
		}
		rs = append(rs, res{vbp, hit})
	}
	sort.Slice(rs, func(i, j int) bool { return rs[i].hit > rs[j].hit })
	fmt.Printf("=== %d kills valides ; couverture paire (attaquant,victime@bp) ===\n", len(kills))
	for i := 0; i < 12; i++ {
		fmt.Printf("  bp=%-3d : %d/%d (%.0f%%)\n", rs[i].bp, rs[i].hit, len(kills), 100*float64(rs[i].hit)/float64(len(kills)))
	}
}
