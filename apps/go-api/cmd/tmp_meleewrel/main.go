// tmp_meleewrel — THROWAWAY : teste le player_index MÊLÉE à un offset RELATIF AU WEAPON id
// (comme grenade : player = weapon_start+79 = weapon_end+47), sur une fenêtre large qui couvre
// AUSSI la zone post-weapon que tmp_meleepidx (borné anchor+120) ne balayait pas.
//
// Marqueur 0x534/0x535 (11b), anchor=bp+3, type@anchor+76 ∈ {0x47,0x42,0x60},
// weapon high32 @ anchor+86(0x47)/+88(0x42)/+101|103(0x60).
//
// Pour chaque δ (relatif au weapon_start) et largeur W, lit le champ, score :
//   - distribution 0-7 propre (borné, non dégénéré)
//   - ground truth : hammer (0x47) près des kills narrés IKE(pi4)->JGtm == 4.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis"
)

const t0Us = uint64(4537898226)

var h32 = map[uint32]string{}
var pi = map[int]string{0: "whiteknight2519", 1: "JAVIERLOLITO540", 2: "JGtm", 3: "LORD PEINX13", 4: "IKE ILYA", 5: "Akatsuki fire17", 6: "aldusbroncus", 7: "VitaminA1688"}

// kills narrés IKE(pi4)->JGtm au marteau (000d5950)
var ikeKills = []float64{115.5, 292.5, 355.7, 375.1}

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
func buildCat() {
	for id, n := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = n
	}
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
func tsAtBit(d []byte, bp int) (int, bool) {
	pos := bp >> 3
	off := 0
	for off+16 <= len(d) {
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if pos >= off+16 && pos < off+16+sz {
			return int((ts - t0Us) / 1000), true
		}
		off += 16 + sz
	}
	return -1, false
}

type mev struct {
	d      []byte
	anchor int
	typ    uint8
	woff   int // offset bit du weapon high32 (absolu dans d)
	wpn    string
	tms    int
}

func collectMelee(cache string) []mev {
	var out []mev
	for n := 0; n <= 41; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		total := len(d) * 8
		for bp := 0; bp+200 < total; bp++ {
			m := bitsAt(d, bp, 11)
			if m != 0x534 && m != 0x535 {
				continue
			}
			anchor := bp + 3
			typ := uint8(bitsAt(d, anchor+76, 8))
			woff := -1
			switch typ {
			case 0x47:
				woff = anchor + 86
			case 0x42:
				woff = anchor + 88
			case 0x60:
				woff = anchor + 101
			default:
				continue
			}
			hi := uint32(bitsAt(d, woff, 32))
			name, ok := h32[hi]
			if !ok {
				if typ == 0x60 {
					woff = anchor + 103
					hi = uint32(bitsAt(d, woff, 32))
					name, ok = h32[hi]
				}
				if !ok {
					continue
				}
			}
			tms, okt := tsAtBit(d, bp)
			if !okt {
				continue
			}
			out = append(out, mev{d, anchor, typ, woff, name, tms})
		}
	}
	return out
}

func nearIKE(tms int, win float64) bool {
	s := float64(tms) / 1000
	for _, k := range ikeKills {
		if s >= k-win && s <= k+win {
			return true
		}
	}
	return false
}

func fmtMap(m map[int]int) string {
	ks := []int{}
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	s := ""
	for _, k := range ks {
		s += fmt.Sprintf("%d:%d ", k, m[k])
	}
	return s
}

type res struct {
	delta, w       int
	distinct, maxv int
	dist           map[int]int
	nearN, nearPi4 int // hammer près kills IKE ±4s
	nearVals       map[int]int
}

func main() {
	film := "000d5950"
	if len(os.Args) >= 2 {
		film = os.Args[1]
	}
	cache := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/` + film
	buildCat()
	evs := collectMelee(cache)
	nH := 0
	for _, e := range evs {
		if e.typ == 0x47 {
			nH++
		}
	}
	fmt.Printf("=== %s : %d melee events (hammer 0x47: %d) ===\n", film, len(evs), nH)

	// balaye δ RELATIF au weapon_start (woff), de -80 à +140, largeurs 5/4/6.
	// (couvre grenade-analog weapon+79, ET la zone anchor+23 = weapon-63.)
	for _, W := range []int{5, 4, 6} {
		fmt.Printf("\n############ W=%d bits (offset relatif au WEAPON high32) ############\n", W)
		var rs []res
		for delta := -80; delta <= 140; delta++ {
			r := res{delta: delta, w: W, dist: map[int]int{}, nearVals: map[int]int{}}
			bad := false
			for _, e := range evs {
				o := e.woff + delta
				// exclure recouvrement avec le weapon 32b et le type 8b
				if o+W > e.woff && o < e.woff+32 { // dans le weapon
					bad = true
					break
				}
				t0 := e.anchor + 76
				if o < t0+8 && o+W > t0 { // dans le type
					bad = true
					break
				}
				if o < 0 {
					bad = true
					break
				}
				v := int(bitsAt(e.d, o, W))
				r.dist[v]++
				if v > r.maxv {
					r.maxv = v
				}
				if e.typ == 0x47 && nearIKE(e.tms, 4) {
					r.nearN++
					r.nearVals[v]++
					if v == 4 {
						r.nearPi4++
					}
				}
			}
			if bad {
				continue
			}
			r.distinct = len(r.dist)
			if r.maxv <= 7 && r.distinct >= 5 {
				rs = append(rs, r)
			}
		}
		// trier par nearPi4 desc puis distinct proche 8
		sort.Slice(rs, func(a, b int) bool {
			if rs[a].nearPi4 != rs[b].nearPi4 {
				return rs[a].nearPi4 > rs[b].nearPi4
			}
			return absi(rs[a].distinct-8) < absi(rs[b].distinct-8)
		})
		fmt.Printf("-- %d offsets valides (0-7, distinct>=5) ; top par (hammer près IKE ==pi4) --\n", len(rs))
		for i, r := range rs {
			if i >= 25 {
				break
			}
			fmt.Printf("  δ=%+4d (abs~anchor%+d) distinct=%d maxv=%d | nearIKE=%d ==pi4=%d vals=%v\n",
				r.delta, r.delta+86, r.distinct, r.maxv, r.nearN, r.nearPi4, fmtMap(r.nearVals))
		}
	}

	// Détail : pour le meilleur δ grenade-analog (weapon+47..+79 zone), dump hammers near IKE kills.
	fmt.Printf("\n=== dump hammers ±8s des kills IKE, lecture à δ=+79 et δ=+47 (weapon-relatif, W=5) ===\n")
	sort.Slice(evs, func(a, b int) bool { return evs[a].tms < evs[b].tms })
	for _, k := range ikeKills {
		fmt.Printf("-- kill %.1fs --\n", k)
		for _, e := range evs {
			if e.typ != 0x47 {
				continue
			}
			s := float64(e.tms) / 1000
			if s < k-8 || s > k+8 {
				continue
			}
			v79 := int(bitsAt(e.d, e.woff+79, 5))
			v47 := int(bitsAt(e.d, e.woff+47, 5))
			fmt.Printf("   t=%7.1fs (Δ%+5.1f) δ+79=%d(%s) δ+47=%d(%s)\n", s, s-k, v79, pi[v79], v47, pi[v47])
		}
	}
}

func absi(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
