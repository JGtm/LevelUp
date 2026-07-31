package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
	"os"
	"sort"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)
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
	tms  int
	fam  string
	r5   int64
	isD2 bool
}

func main() {
	off := -20300
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
			fam, ok := h32[uint32(bitsAt(pl, s-32, 32))]
			if !ok {
				continue
			}
			tms := int((int64(ts) - int64(t0Us)) / 1000)
			r := rec{tms, fam, -1, pl[0] == 0xd2}
			if r.isD2 {
				br := filmdec.NewBitReader(pl)
				br.Skip(36)
				r.r5 = int64(br.ReadBits(5))
			}
			all = append(all, r)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].tms < all[j].tms })
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
	// dense nearest + d2-verifie
	dense, d2cov, agree, both := 0, 0, 0, 0
	for _, k := range feed {
		// dense : record arme le plus proche +-2s
		bd, dfam := 2001, "?"
		for _, r := range all {
			d := r.tms - (k.t + off)
			if d < 0 {
				d = -d
			}
			if d < bd {
				bd, dfam = d, r.fam
			}
		}
		if dfam != "?" {
			dense++
		}
		// d2 verifie : record 0xd2 avec R5>>1==pi(tueur), +-800ms
		pi := xpi(k.killer)
		bd2, d2fam := 801, "?"
		if pi >= 0 {
			for _, r := range all {
				if !r.isD2 || int(r.r5>>1) != pi {
					continue
				}
				d := r.tms - (k.t + off)
				if d < 0 {
					d = -d
				}
				if d < bd2 {
					bd2, d2fam = d, r.fam
				}
			}
		}
		if d2fam != "?" {
			d2cov++
		}
		if dfam != "?" && d2fam != "?" {
			both++
			if dfam == d2fam {
				agree++
			}
		}
	}
	fmt.Printf("=== %d records ; %d kills ; offset=%dms ===\n", len(all), len(feed), off)
	fmt.Printf("  couverture DENSE (record le plus proche +-2s)      : %d/%d\n", dense, len(feed))
	fmt.Printf("  couverture 0xd2-VERIFIE (R5==tueur +-800ms)        : %d/%d\n", d2cov, len(feed))
	fmt.Printf("  ACCORD dense vs 0xd2-verifie (sur recouvrement)    : %d/%d (%.0f%%)\n", agree, both, 100*float64(agree)/float64(both))
}
