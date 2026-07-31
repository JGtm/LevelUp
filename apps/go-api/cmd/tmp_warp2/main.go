// tmp_warp2 : convertit le temps des dégâts via l'INDEX DE FRAME (paquets 0xa0 = ticks de jeu),
// pour annuler la dérive horloge-flux↔temps-jeu. Test alignement whiteknight + couverture.
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
func mustRead(p string) []byte { b, _ := os.ReadFile(p); return b }
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
func main() {
	for id, n := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = n
	}
	m := os.Args[1]
	cache := root + "/" + m
	type rr struct {
		slot  int
		fam   string
		frame int
	}
	var raw []rr
	frameIdx := 0
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		if len(d) == 0 {
			continue
		}
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if typ != 0 || len(pl) == 0 {
				continue
			}
			if pl[0] == 0xa0 {
				frameIdx++
				continue
			}
			if pl[0] != 0xd2 {
				continue
			}
			r5 := int(bitsAt(pl, 36, 5))
			bp := 41
			if bitsAt(pl, bp, 1) == 1 {
				bp++
			} else {
				bp += 3
			}
			fam32 := uint32(bitsAt(pl, bp, 32))
			low := uint32(bitsAt(pl, bp+32, 32))
			if nm, ok := h32[fam32]; ok && low == sfx {
				raw = append(raw, rr{r5 >> 1, nm, frameIdx})
			}
		}
	}
	totalFrames := frameIdx
	// kill-feed
	var kills, deaths []struct {
		x uint64
		t int
	}
	for ch := 41; ch >= 18; ch-- {
		b := mustRead(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		if len(b) == 0 {
			continue
		}
		ev, _ := analysis.ParseHighlightEvents(b, 0)
		nk := 0
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
				nk++
			case analysis.EventTypeDeath:
				dd = append(dd, struct {
					x uint64
					t int
				}{e.XUID, e.TimeMS})
			}
		}
		if nk > len(kills) {
			kills, deaths = kk, dd
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].t < kills[j].t })
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].t < deaths[j].t })
	used := make([]bool, len(deaths))
	type kf struct {
		killer, victim uint64
		t              int
	}
	var feed []kf
	for _, k := range kills {
		best, bd := -1, 400
		for i, d := range deaths {
			if used[i] || d.x == k.x {
				continue
			}
			dt := k.t - d.t
			if dt < 0 {
				dt = -dt
			}
			if dt < bd {
				bd, best = dt, i
			}
		}
		if best >= 0 {
			used[best] = true
			feed = append(feed, kf{k.x, deaths[best].x, k.t})
		}
	}
	// tick_ms = temps-jeu max (dernier kill) / total frames
	maxT := 0
	for _, k := range feed {
		if k.t > maxT {
			maxT = k.t
		}
	}
	tickMs := float64(maxT) / float64(totalFrames)
	type dmg struct {
		slot int
		fam  string
		tms  int
	}
	var dmgs []dmg
	for _, r := range raw {
		dmgs = append(dmgs, dmg{r.slot, r.fam, int(float64(r.frame) * tickMs)})
	}
	fmt.Printf("=== %s : %d frames-jeu (0xa0) ; tick=%.2fms ; %d dégâts ; %d kills ===\n", m, totalFrames, tickMs, len(dmgs), len(feed))
	// roster
	var t8 []byte
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
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
	first := map[uint64]int{}
	for bp := 0; bp <= len(t8)*8-64; bp++ {
		if v := u64LE(t8, bp); want[v] {
			if _, ok := first[v]; !ok {
				first[v] = bp
			}
		}
	}
	type xb struct {
		x  uint64
		bp int
	}
	var xbs []xb
	for x, bp := range first {
		xbs = append(xbs, xb{x, bp})
	}
	sort.Slice(xbs, func(i, j int) bool { return xbs[i].bp < xbs[j].bp })
	slotOf := map[uint64]int{}
	for s, e := range xbs {
		slotOf[e.x] = s
	}
	// offset scan (fenêtre 3s)
	cov := func(B int) int {
		c := 0
		for _, k := range feed {
			s, ok := slotOf[k.killer]
			if !ok {
				continue
			}
			for _, d := range dmgs {
				if d.slot != s {
					continue
				}
				dd := d.tms - (k.t - B)
				if dd < 0 {
					dd = -dd
				}
				if dd < 3000 {
					c++
					break
				}
			}
		}
		return c
	}
	bB, bC := 0, -1
	for B := -30000; B <= 30000; B += 100 {
		if c := cov(B); c > bC {
			bC, bB = c, B
		}
	}
	att := 0
	for _, k := range feed {
		s, ok := slotOf[k.killer]
		if !ok {
			continue
		}
		bd, fam := 4000, "?"
		for _, d := range dmgs {
			if d.slot != s {
				continue
			}
			dd := d.tms - (k.t - bB)
			if dd < 0 {
				dd = -dd
			}
			if dd < bd {
				bd, fam = dd, d.fam
			}
		}
		if fam != "?" {
			att++
		}
	}
	fmt.Printf(">>> offset=%dms ; COUVERTURE (temps-frame) = %d/%d (%.0f%%)\n", bB, att, len(feed), 100*float64(att)/float64(len(feed)))
}
