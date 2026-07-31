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
const t0Us = uint64(4537898226)
const sfx = uint32(0x42c9679f)

var piX = map[int]uint64{0: 2535467794760703, 1: 2535437947245250, 2: 2533274823110022, 3: 2533274980284321, 4: 2533274815845110, 5: 2535444178793711, 6: 2533274882097883, 7: 2533274826120416}

func xpi(x uint64) int {
	for p, xu := range piX {
		if xu == x {
			return p
		}
	}
	return -1
}
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
func suffixPos(d []byte) int {
	m := len(d)*8 - 32
	for bp := 0; bp <= m; bp++ {
		if uint32(bitsAt(d, bp, 32)) == sfx {
			return bp
		}
	}
	return -1
}

type rec struct {
	tms int
	pl  []byte
	sfx int
}

func main() {
	mk := byte(0xe9)
	if len(os.Args) > 1 {
		var v int
		fmt.Sscanf(os.Args[1], "0x%x", &v)
		mk = byte(v)
	}
	var recs []rec
	for ch := 0; ch <= 27; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		if len(d) == 0 {
			continue
		}
		o := 0
		for o+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[o:])
			sz := int(binary.LittleEndian.Uint32(d[o+4:]))
			ts := binary.LittleEndian.Uint64(d[o+8:])
			if sz <= 0 || o+16+sz > len(d) {
				break
			}
			pl := d[o+16 : o+16+sz]
			o += 16 + sz
			if typ != 0 || len(pl) == 0 || pl[0] != mk {
				continue
			}
			s := suffixPos(pl)
			if s < 0 {
				continue
			}
			recs = append(recs, rec{int((int64(ts) - int64(t0Us)) / 1000), pl, s})
		}
	}
	events, _ := analysis.ParseHighlightEvents(mustRead(cache+"/chunk_27.bin"), 0)
	type ev struct {
		x uint64
		t int
	}
	var kills, deaths []ev
	for _, e := range events {
		switch e.EventType {
		case analysis.EventTypeKill:
			kills = append(kills, ev{e.XUID, e.TimeMS})
		case analysis.EventTypeDeath:
			deaths = append(deaths, ev{e.XUID, e.TimeMS})
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
		for i, dd := range deaths {
			if used[i] || dd.x == k.x {
				continue
			}
			dt := k.t - dd.t
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
	// 1) offset temporel propre du marqueur (coverage record<->kill)
	bestOff, bestC := 0, -1
	for off := -30000; off <= 5000; off += 100 {
		c := 0
		for _, k := range feed {
			for _, r := range recs {
				d := r.tms - (k.t + off)
				if d < 0 {
					d = -d
				}
				if d <= 500 {
					c++
					break
				}
			}
		}
		if c > bestC {
			bestC, bestOff = c, off
		}
	}
	fmt.Printf("=== 0x%02x : %d records ; offset propre=%+dms -> %d/%d kills couverts (+-500ms) ===\n", mk, len(recs), bestOff, bestC, len(feed))
	// 2) assigner chaque record au kill le plus proche a cet offset (+-700ms)
	type asg struct {
		r        rec
		vic, kil int
	}
	var as []asg
	for _, r := range recs {
		bd, bk := 701, -1
		for i, k := range feed {
			d := r.tms - (k.t + bestOff)
			if d < 0 {
				d = -d
			}
			if d < bd {
				bd, bk = d, i
			}
		}
		if bk >= 0 {
			as = append(as, asg{r, xpi(feed[bk].victim), xpi(feed[bk].killer)})
		}
	}
	fmt.Printf("  %d records assignes a un kill (+-700ms)\n", len(as))
	// 3) scan large champ joueur : width 3-7, offset depuis debut ET avant suffixe, valeur directe / >>1 vs tueur/victime
	type best struct {
		desc string
		n    int
	}
	var bv, bk best
	test := func(getv func(a asg) int, desc string) {
		nv, nk := 0, 0
		for _, a := range as {
			v := getv(a)
			if v == a.vic {
				nv++
			}
			if v == a.kil {
				nk++
			}
		}
		if nv > bv.n {
			bv = best{desc, nv}
		}
		if nk > bk.n {
			bk = best{desc, nk}
		}
	}
	for w := 3; w <= 7; w++ {
		for o := 0; o <= 120; o++ {
			test(func(a asg) int { return int(bitsAt(a.r.pl, o, w)) }, fmt.Sprintf("debut o=%d w=%d val", o, w))
			test(func(a asg) int { return int(bitsAt(a.r.pl, o, w)) >> 1 }, fmt.Sprintf("debut o=%d w=%d >>1", o, w))
			test(func(a asg) int {
				p := a.r.sfx - o - w
				if p < 0 {
					return -99
				}
				return int(bitsAt(a.r.pl, p, w))
			}, fmt.Sprintf("avantSfx o=%d w=%d val", o, w))
			test(func(a asg) int {
				p := a.r.sfx - o - w
				if p < 0 {
					return -99
				}
				return int(bitsAt(a.r.pl, p, w)) >> 1
			}, fmt.Sprintf("avantSfx o=%d w=%d >>1", o, w))
		}
	}
	fmt.Printf("  MEILLEUR champ ~VICTIME : %s -> %d/%d (%.0f%%)\n", bv.desc, bv.n, len(as), 100*float64(bv.n)/float64(len(as)))
	fmt.Printf("  MEILLEUR champ ~TUEUR   : %s -> %d/%d (%.0f%%)\n", bk.desc, bk.n, len(as), 100*float64(bk.n)/float64(len(as)))
}
