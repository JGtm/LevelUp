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
func idxK(h uint32) int {
	if h < 0xE1500000 || h > 0xE1600000 {
		return -1
	}
	return int((h - 0xE1500000) / 0x10002)
}
func idxD(h uint32) int {
	if h < 0xEC500000 || h > 0xEC600000 {
		return -1
	}
	return int((h - 0xEC500000) / 0x10002)
}
func main() {
	for id := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = analysis.WeaponIDToName[id]
	}
	// GROUND TRUTH live : par kill (killer,ktick)->arme via dmgcapture attacker+tick
	dd, _ := os.ReadFile(dc)
	type dmg struct {
		atk int
		w   string
		tk  uint32
	}
	var dmgs []dmg
	for o := 0; o+32 <= len(dd); o += 32 {
		a := idxD(binary.LittleEndian.Uint32(dd[o:]))
		if a < 0 {
			continue
		}
		w := h32[binary.LittleEndian.Uint32(dd[o+8:])]
		tk := binary.LittleEndian.Uint32(dd[o+20:])
		dmgs = append(dmgs, dmg{a, w, tk})
	}
	kd, _ := os.ReadFile(kc)
	type kill struct {
		k, v  int
		tk    uint32
		truth string
	}
	var kills []kill
	for o := 0; o+16 <= len(kd); o += 16 {
		ki := idxK(binary.LittleEndian.Uint32(kd[o:]))
		vi := idxK(binary.LittleEndian.Uint32(kd[o+4:]))
		if ki < 0 || vi < 0 || ki == vi {
			continue
		}
		tk := binary.LittleEndian.Uint32(kd[o+12:])
		best := ""
		var bt uint32
		for _, d := range dmgs {
			if d.atk == ki && d.tk <= tk && d.tk >= bt {
				bt = d.tk
				best = d.w
			}
		}
		kills = append(kills, kill{ki, vi, tk, best})
	}
	// OFFLINE 0xd2 : (atk, victim@bp, weapon, ts)
	type R struct {
		atk int
		w   string
		ts  uint64
		pl  []byte
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
				recs = append(recs, R{int(bitsAt(pl, 36, 5)) >> 1, nm, pkts, pl})
			}
		}
	}
	type res struct{ bp, agree, found int }
	var rs []res
	for _, vbp := range []int{12, 28, 29, 130, 151, 152, 179, 90, 112, 108, 146, 133} {
		ag, fo := 0, 0
		for _, k := range kills {
			if k.truth == "" {
				continue
			}
			var bw string
			var bts uint64
			for _, r := range recs {
				if r.atk != k.k {
					continue
				}
				if int(bitsAt(r.pl, vbp, 5))>>1 != k.v {
					continue
				}
				if r.ts >= bts {
					bts = r.ts
					bw = r.w
				}
			}
			if bw != "" {
				fo++
				if bw == k.truth {
					ag++
				}
			}
		}
		rs = append(rs, res{vbp, ag, fo})
	}
	sort.Slice(rs, func(i, j int) bool { return rs[i].agree > rs[j].agree })
	nk := 0
	for _, k := range kills {
		if k.truth != "" {
			nk++
		}
	}
	fmt.Printf("=== %d kills avec verite ; accord ARME offline (dernier record tueur->victime@bp) ===\n", nk)
	for _, r := range rs {
		fmt.Printf("  victime@bp%-3d : arme trouvee=%d ; arme CORRECTE=%d/%d (%.0f%%)\n", r.bp, r.found, r.agree, nk, 100*float64(r.agree)/float64(nk))
	}
}
