package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/weaponv3"
	"os"
	"sort"
)

const root = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`
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
func u64LE(d []byte, bp int) uint64 {
	var b [8]byte
	for i := 0; i < 8; i++ {
		var by byte
		for j := 0; j < 8; j++ {
			q := bp + i*8 + j
			if q>>3 < len(d) {
				by |= ((d[q>>3] >> uint(7-(q&7))) & 1) << uint(7-j)
			}
		}
		b[i] = by
	}
	return binary.LittleEndian.Uint64(b[:])
}

type dmg struct {
	slot int
	fam  string
	ts   uint64
}
type seg struct {
	start int
	ts    uint64
}

func main() {
	m := "00ba2e1c"
	if len(os.Args) > 1 {
		m = os.Args[1]
	}
	cache := root + "/" + m
	for id := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = analysis.WeaponIDToName[id]
	}
	var chunks [][]byte
	chSegs := map[int][]seg{}
	var dmgs []dmg
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		chunks = append(chunks, d)
		if len(d) == 0 {
			continue
		}
		off := 0
		var segs []seg
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			if typ == 0 {
				segs = append(segs, seg{off + 16, ts})
				if len(pl) > 0 && pl[0] == 0xd2 {
					bp := 41
					if bitsAt(pl, bp, 1) == 1 {
						bp++
					} else {
						bp += 3
					}
					f := uint32(bitsAt(pl, bp, 32))
					if nm, ok := h32[f]; ok && uint32(bitsAt(pl, bp+32, 32)) == sfx {
						dmgs = append(dmgs, dmg{int(bitsAt(pl, 36, 5)) >> 1, nm, ts})
					}
				}
			}
			off += 16 + sz
		}
		chSegs[ch] = segs
	}
	sort.Slice(dmgs, func(i, j int) bool { return dmgs[i].ts < dmgs[j].ts })
	var kills, deaths []struct {
		x uint64
		t int
	}
	for ch := 41; ch >= 18; ch-- {
		b := chunks[ch]
		if len(b) == 0 {
			continue
		}
		ev, _ := analysis.ParseHighlightEvents(b, 0)
		var kk, dd []struct {
			x uint64
			t int
		}
		for _, e := range ev {
			switch e.EventType {
			case analysis.EventTypeKill:
				kk = append(kk, struct {
					x uint64
					t int
				}{e.XUID, e.TimeMS})
			case analysis.EventTypeDeath:
				dd = append(dd, struct {
					x uint64
					t int
				}{e.XUID, e.TimeMS})
			}
		}
		if len(kk) > len(kills) {
			kills, deaths = kk, dd
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].t < kills[j].t })
	var t8 []byte
	for _, d := range chunks {
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if typ == 8 && len(pl) > len(t8) {
				t8 = pl
			}
		}
	}
	want := map[uint64]bool{}
	for _, k := range kills {
		want[k.x] = true
	}
	for _, d := range deaths {
		want[d.x] = true
	}
	fb := map[uint64]int{}
	for bp := 0; bp <= len(t8)*8-64; bp++ {
		if v := u64LE(t8, bp); want[v] {
			if _, ok := fb[v]; !ok {
				fb[v] = bp
			}
		}
	}
	type xb struct {
		x  uint64
		bp int
	}
	var xbs []xb
	for x, bp := range fb {
		xbs = append(xbs, xb{x, bp})
	}
	sort.Slice(xbs, func(i, j int) bool { return xbs[i].bp < xbs[j].bp })
	slotOf := map[uint64]int{}
	for s, e := range xbs {
		slotOf[e.x] = s
	}
	type kk struct{ slot, t int }
	var ks []kk
	for _, k := range kills {
		if s, ok := slotOf[k.x]; ok {
			ks = append(ks, kk{s, k.t})
		}
	}
	if len(ks) == 0 || len(dmgs) == 0 {
		fmt.Println("insuffisant")
		return
	}
	tsMin, tsMax := dmgs[0].ts, dmgs[len(dmgs)-1].ts
	tMin, tMax := ks[0].t, ks[0].t
	for _, k := range ks {
		if k.t < tMin {
			tMin = k.t
		}
		if k.t > tMax {
			tMax = k.t
		}
	}
	a := float64(tMax-tMin) / float64(tsMax-tsMin)
	b := float64(tMin) - a*float64(tsMin)
	warp := func(ts uint64) float64 { return a*float64(ts) + b }
	fit := func(lb bool) {
		var sx, sy, sxx, sxy, n float64
		for _, k := range ks {
			var bts uint64
			bd := 1e18
			f := false
			for _, d := range dmgs {
				if d.slot != k.slot {
					continue
				}
				wt := warp(d.ts)
				if lb {
					if wt <= float64(k.t) && d.ts > bts {
						bts = d.ts
						f = true
						bd = float64(k.t) - wt
					}
				} else {
					di := wt - float64(k.t)
					if di < 0 {
						di = -di
					}
					if di < bd {
						bd = di
						bts = d.ts
						f = true
					}
				}
			}
			if f && bd < 8000 {
				x := float64(bts)
				y := float64(k.t)
				sx += x
				sy += y
				sxx += x * x
				sxy += x * y
				n++
			}
		}
		if n > 2 {
			na := (n*sxy - sx*sy) / (n*sxx - sx*sx)
			if na > 0 {
				a = na
				b = (sy - na*sx) / n
			}
		}
	}
	for i := 0; i < 3; i++ {
		fit(false)
	}
	for i := 0; i < 3; i++ {
		fit(true)
	}
	// RE-SCAN MÊLÉE avec est=warp (temps-jeu)
	var melees []weaponv3.MeleeHit
	for ch := 0; ch <= 41; ch++ {
		d := chunks[ch]
		if len(d) == 0 {
			continue
		}
		segs := chSegs[ch]
		est := func(bp int) float64 {
			var tt uint64
			for _, s := range segs {
				if s.start <= bp {
					tt = s.ts
				} else {
					break
				}
			}
			return warp(tt)
		}
		for _, h := range weaponv3.ScanMeleeHits(d, est) {
			if weaponv3.MeleeHitLethal(h.HitType) {
				melees = append(melees, h)
			}
		}
	}
	fire, mel, neither := 0, 0, 0
	for _, k := range ks {
		best := ""
		var bts uint64
		for _, d := range dmgs {
			if d.slot != k.slot {
				continue
			}
			if warp(d.ts) <= float64(k.t) && d.ts >= bts {
				bts = d.ts
				best = d.fam
			}
		}
		if best != "" {
			fire++
			continue
		}
		hasMel := false
		for _, h := range melees {
			if h.PI == k.slot {
				dt := h.TimeMS - k.t
				if dt < 0 {
					dt = -dt
				}
				if dt < 2500 {
					hasMel = true
					break
				}
			}
		}
		if hasMel {
			mel++
		} else {
			neither++
		}
	}
	anyAlign := 0
	for _, h := range melees {
		for _, k := range ks {
			dt := h.TimeMS - k.t
			if dt < 0 {
				dt = -dt
			}
			if dt < 2500 {
				anyAlign++
				break
			}
		}
	}
	fmt.Printf("=== %s : %d kills ; %d dégâts 0xd2 ; %d mêlées (temps-jeu warpé) ===\n", m, len(ks), len(dmgs), len(melees))
	fmt.Printf("  FIREARM %d (%.0f%%) | MÊLÉE même-slot<2.5s %d (%.0f%%) | NI %d (%.0f%%)\n", fire, 100*float64(fire)/float64(len(ks)), mel, 100*float64(mel)/float64(len(ks)), neither, 100*float64(neither)/float64(len(ks)))
	fmt.Printf("  [diag] mêlées alignées à un kill (tout slot) : %d/%d\n", anyAlign, len(melees))
}
