package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"levelup/go-api/internal/analysis"
	"math"
	"os"
	"sort"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const dc = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/dmgcapture_run2.bin`
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
func idxD(h uint32) int {
	if h < 0xEC500000 || h > 0xEC600000 {
		return -1
	}
	return int((h - 0xEC500000) / 0x10002)
}

type pt struct{ x, y float64 }

func main() {
	for id := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = analysis.WeaponIDToName[id]
	}
	// offline: key=atk|weapon -> sorted packet_ts
	type ow struct {
		atk int
		w   string
		ts  uint64
	}
	var offs []ow
	for ch := 0; ch <= 27; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		o := 0
		for o+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[o:])
			sz := int(binary.LittleEndian.Uint32(d[o+4:]))
			if sz <= 0 || o+16+sz > len(d) {
				break
			}
			pkts := binary.LittleEndian.Uint64(d[o+8:])
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
			if nm, ok := h32[f]; ok && uint32(bitsAt(pl, bp+32, 32)) == sfx {
				offs = append(offs, ow{int(bitsAt(pl, 36, 5)) >> 1, nm, pkts})
			}
		}
	}
	// live: key=atk|weapon -> sorted tick
	dd, _ := os.ReadFile(dc)
	type lw struct {
		atk int
		w   string
		tk  uint32
	}
	var lvs []lw
	for o := 0; o+32 <= len(dd); o += 32 {
		a := idxD(binary.LittleEndian.Uint32(dd[o:]))
		if a < 0 {
			continue
		}
		w := h32[binary.LittleEndian.Uint32(dd[o+8:])]
		if w == "" {
			continue
		}
		lvs = append(lvs, lw{a, w, binary.LittleEndian.Uint32(dd[o+20:])})
	}
	offM := map[string][]uint64{}
	lvM := map[string][]uint32{}
	for _, r := range offs {
		k := fmt.Sprintf("%d|%s", r.atk, r.w)
		offM[k] = append(offM[k], r.ts)
	}
	for _, r := range lvs {
		k := fmt.Sprintf("%d|%s", r.atk, r.w)
		lvM[k] = append(lvM[k], r.tk)
	}
	var pts []pt
	for k, os_ := range offM {
		ls := lvM[k]
		if len(ls) == 0 {
			continue
		}
		sort.Slice(os_, func(i, j int) bool { return os_[i] < os_[j] })
		sort.Slice(ls, func(i, j int) bool { return ls[i] < ls[j] })
		n := len(os_)
		if len(ls) < n {
			n = len(ls)
		}
		for i := 0; i < n; i++ {
			pts = append(pts, pt{float64(os_[i]), float64(ls[i])})
		}
	}
	// linear fit y=a*x+b + R2
	var sx, sy, sxx, sxy float64
	n := float64(len(pts))
	for _, p := range pts {
		sx += p.x
		sy += p.y
		sxx += p.x * p.x
		sxy += p.x * p.y
	}
	a := (n*sxy - sx*sy) / (n*sxx - sx*sx)
	b := (sy - a*sx) / n
	var ssr, sst, my float64
	my = sy / n
	for _, p := range pts {
		pr := a*p.x + b
		ssr += (p.y - pr) * (p.y - pr)
		sst += (p.y - my) * (p.y - my)
	}
	r2 := 1 - ssr/sst
	fmt.Printf("=== %d couples (packet_ts, tick-jeu) apparies par attaquant+arme ===\n", len(pts))
	fmt.Printf("  fit lineaire tick = %.6g * packet_ts + %.6g\n", a, b)
	fmt.Printf("  R2 = %.5f  (1.0=parfaitement lineaire ; <0.99 = warp non-lineaire)\n", r2)
	fmt.Printf("  residu RMS = %.2f ticks (%.1f s a ~1 tick/s)\n", math.Sqrt(ssr/n), math.Sqrt(ssr/n))
}
