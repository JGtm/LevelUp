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
const live = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/dmgcapture_run2.bin`
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
func main() {
	for id, n := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = n
	}
	off := map[string]int{}
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
			lo := uint32(bitsAt(pl, bp+32, 32))
			if nm, ok := h32[f]; ok && lo == sfx {
				off[nm]++
			}
		}
	}
	lv := map[string]int{}
	d, _ := os.ReadFile(live)
	var tmin, tmax uint32
	for o := 0; o+32 <= len(d); o += 32 {
		a := binary.LittleEndian.Uint32(d[o:])
		if a == 0 {
			continue
		}
		f := binary.LittleEndian.Uint32(d[o+8:])
		nm := h32[f]
		if nm == "" {
			nm = fmt.Sprintf("0x%08x", f)
		}
		lv[nm]++
		tk := binary.LittleEndian.Uint32(d[o+20:])
		if tmin == 0 || tk < tmin {
			tmin = tk
		}
		if tk > tmax {
			tmax = tk
		}
	}
	fmt.Printf("TICK [20] live : min=%d max=%d amplitude=%d\n", tmin, tmax, tmax-tmin)
	allk := map[string]bool{}
	for k := range off {
		allk[k] = true
	}
	for k := range lv {
		allk[k] = true
	}
	var ks []string
	for k := range allk {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	to, tl, tm := 0, 0, 0
	fmt.Printf("=== par arme : LIVE vs 0xd2 offline ===\n")
	for _, k := range ks {
		o, l := off[k], lv[k]
		to += o
		tl += l
		d := ""
		if l > o {
			tm += l - o
			d = fmt.Sprintf("  <<< MANQUE %d", l-o)
		}
		fmt.Printf("  %-26s live=%-3d 0xd2=%-3d%s\n", k, l, o, d)
	}
	fmt.Printf("\n=== TOTAL live=%d ; 0xd2=%d ; total manquant offline=%d ===\n", tl, to, tm)
}
