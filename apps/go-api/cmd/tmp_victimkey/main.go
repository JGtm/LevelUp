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
const off = -20300
const sfx = uint32(0x42c9679f)

var h32 = map[uint32]string{}
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
	mk  byte
	pl  []byte
}

func main() {
	for id, n := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = n
	}
	var all []rec
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
			if typ != 0 || len(pl) == 0 {
				continue
			}
			s := suffixPos(pl)
			if s < 32 {
				continue
			}
			if _, ok := h32[uint32(bitsAt(pl, s-32, 32))]; ok {
				all = append(all, rec{int((int64(ts) - int64(t0Us)) / 1000), pl[0], pl})
			}
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
	// assigner chaque record a son kill le plus proche (+-1500ms a -20.3s)
	type asg struct {
		r        rec
		vic, kil int
	}
	byMk := map[byte][]asg{}
	for _, r := range all {
		bd, bk := 1501, -1
		for i, k := range feed {
			d := r.tms - (k.t + off)
			if d < 0 {
				d = -d
			}
			if d < bd {
				bd, bk = d, i
			}
		}
		if bk >= 0 {
			byMk[r.mk] = append(byMk[r.mk], asg{r, xpi(feed[bk].victim), xpi(feed[bk].killer)})
		}
	}
	var mks []byte
	for m := range byMk {
		if len(byMk[m]) >= 10 {
			mks = append(mks, m)
		}
	}
	sort.Slice(mks, func(i, j int) bool { return len(byMk[mks[i]]) > len(byMk[mks[j]]) })
	fmt.Println("=== par marqueur : meilleur offset d'un champ 5b matchant VICTIME vs TUEUR du kill assigne ===")
	for _, mk := range mks {
		as := byMk[mk]
		bestV, bestVoff, bestK, bestKoff := 0, -1, 0, -1
		for o := 0; o <= 80; o++ {
			nv, nk := 0, 0
			for _, a := range as {
				f := int(bitsAt(a.r.pl, o, 5)) >> 1
				if f == a.vic {
					nv++
				}
				if f == a.kil {
					nk++
				}
			}
			if nv > bestV {
				bestV, bestVoff = nv, o
			}
			if nk > bestK {
				bestK, bestKoff = nk, o
			}
		}
		fmt.Printf("  0x%02x (%3d assignes) : VICTIME off=%2d %d/%d (%.0f%%) ; TUEUR off=%2d %d/%d (%.0f%%)\n",
			mk, len(as), bestVoff, bestV, len(as), 100*float64(bestV)/float64(len(as)), bestKoff, bestK, len(as), 100*float64(bestK)/float64(len(as)))
	}
}
