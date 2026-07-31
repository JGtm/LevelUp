// tmp_sibfield : recherche SUPERVISÉE du champ joueur d'un marqueur frère, avec le roster connu.
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
const sfxV = uint32(0x42c9679f)

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
func suffixPos(d []byte) int {
	m := len(d)*8 - 32
	for bp := 0; bp <= m; bp++ {
		if uint32(bitsAt(d, bp, 32)) == sfxV {
			return bp
		}
	}
	return -1
}
func main() {
	mk := byte(0xe9)
	if len(os.Args) > 1 {
		var v int
		fmt.Sscanf(os.Args[1], "0x%x", &v)
		mk = byte(v)
	}
	// roster type-8
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
	// kill-feed
	var kills, deaths []struct {
		x uint64
		t int
	}
	for ch := 27; ch >= 27; ch-- {
		ev, _ := analysis.ParseHighlightEvents(mustRead(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch)), 0)
		for _, e := range ev {
			if e.EventType == analysis.EventTypeKill {
				kills = append(kills, struct {
					x uint64
					t int
				}{e.XUID, e.TimeMS})
			}
			if e.EventType == analysis.EventTypeDeath {
				deaths = append(deaths, struct {
					x uint64
					t int
				}{e.XUID, e.TimeMS})
			}
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].t < kills[j].t })
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].t < deaths[j].t })
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
	used := make([]bool, len(deaths))
	type kf struct{ ks, vs, t int }
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
			feed = append(feed, kf{slotOf[k.x], slotOf[deaths[best].x], k.t})
		}
	}
	// records du marqueur mk : ts + payload + suffixPos
	type rec struct {
		tms int
		pl  []byte
		sp  int
	}
	var recs []rec
	var t0 uint64
	firstTs := true
	for ch := 0; ch <= 27; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if typ != 0 || len(pl) == 0 || pl[0] != 0xd2 {
			}
			if typ == 0 && len(pl) > 0 && pl[0] == mk {
				sp := suffixPos(pl)
				if sp < 0 {
					continue
				}
				if firstTs {
					t0 = ts
					firstTs = false
				}
				recs = append(recs, rec{int((int64(ts) - int64(t0)) / 1000), pl, sp})
			}
		}
	}
	if len(recs) == 0 {
		fmt.Println("0 records")
		return
	}
	// offset temporel propre du marqueur
	cov := func(B int) int {
		c := 0
		for _, k := range feed {
			for _, r := range recs {
				dd := r.tms - (k.t - B)
				if dd < 0 {
					dd = -dd
				}
				if dd <= 500 {
					c++
					break
				}
			}
		}
		return c
	}
	bO, bC := 0, -1
	for B := -10000; B <= 70000; B += 300 {
		if c := cov(B); c > bC {
			bC, bO = c, B
		}
	}
	// assigner record -> kill le plus proche (a bO, fenetre 800)
	type asg struct {
		r      rec
		ks, vs int
	}
	var as []asg
	for _, r := range recs {
		bd, bk := 801, -1
		for i, k := range feed {
			dd := r.tms - (k.t - bO)
			if dd < 0 {
				dd = -dd
			}
			if dd < bd {
				bd, bk = dd, i
			}
		}
		if bk >= 0 {
			as = append(as, asg{r, feed[bk].ks, feed[bk].vs})
		}
	}
	fmt.Printf("=== 0x%02x : %d records ; offset=%dms ; couverture=%d/%d ; %d assignés ===\n", mk, len(recs), bO, bC, len(feed), len(as))
	// recherche supervisée : champ (offset depuis début OU avant suffixe, width 3-6, val ou >>1) == tueur OU victime
	type best struct {
		desc string
		n    int
	}
	var bk, bv best
	tk := func(get func(a asg) int, desc string) {
		nk, nv := 0, 0
		for _, a := range as {
			x := get(a)
			if x == a.ks {
				nk++
			}
			if x == a.vs {
				nv++
			}
		}
		if nk > bk.n {
			bk = best{desc, nk}
		}
		if nv > bv.n {
			bv = best{desc, nv}
		}
	}
	for w := 3; w <= 6; w++ {
		for o := 0; o <= 100; o++ {
			tk(func(a asg) int { return int(bitsAt(a.r.pl, o, w)) }, fmt.Sprintf("dbut o=%d w=%d", o, w))
			tk(func(a asg) int { return int(bitsAt(a.r.pl, o, w)) >> 1 }, fmt.Sprintf("dbut o=%d w=%d >>1", o, w))
			tk(func(a asg) int {
				p := a.r.sp - o - w
				if p < 0 {
					return -9
				}
				return int(bitsAt(a.r.pl, p, w))
			}, fmt.Sprintf("avSfx o=%d w=%d", o, w))
			tk(func(a asg) int {
				p := a.r.sp - o - w
				if p < 0 {
					return -9
				}
				return int(bitsAt(a.r.pl, p, w)) >> 1
			}, fmt.Sprintf("avSfx o=%d w=%d >>1", o, w))
		}
	}
	fmt.Printf("  meilleur champ ~TUEUR   : %s -> %d/%d (%.0f%%)\n", bk.desc, bk.n, len(as), 100*float64(bk.n)/float64(len(as)))
	fmt.Printf("  meilleur champ ~VICTIME : %s -> %d/%d (%.0f%%)\n", bv.desc, bv.n, len(as), 100*float64(bv.n)/float64(len(as)))
}
