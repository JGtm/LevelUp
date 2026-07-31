// tmp_meleepidx — THROWAWAY : CALAGE de l'offset player_index dans les events MELEE.
// Réutilise la calibration mêlée acquise :
//
//	marqueur 0x534(HIT)/0x535(MISS) 11b ; anchor=bp+3 ; type@anchor+76 (8b) ∈ {0x47,0x42,0x60} ;
//	weapon high32@anchor+86 (0x47) / +88 (0x42) / +101|+103 (0x60).
//
// On scanne chaque offset candidat o∈[anchor+0..+120] (en sautant zones weapon 32b + type 8b),
// lit un champ de largeur W bits, calcule la distribution. Offset VALIDE = borné 0-7, non dégénéré.
// Critère décisif : sur les events GRAVITY HAMMER (type 0x47) près des kills IKE->JGtm
//
//	(115.5/292.5/355.7/375.1s, ±2s), le pidx doit valoir 4 (IKE).
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

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)

var h32 = map[uint32]string{}
var pi = map[int]string{0: "whiteknight2519", 1: "JAVIERLOLITO540", 2: "JGtm", 3: "LORD PEINX13", 4: "IKE ILYA", 5: "Akatsuki fire17", 6: "aldusbroncus", 7: "VitaminA1688"}

// kills narrés IKE(pi4)->JGtm au marteau, secondes
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

// melee event décodé : on garde anchor (bit absolu), chunk d, type, weapon, ts.
type mev struct {
	d      []byte
	anchor int
	typ    uint8
	woff   int // offset bit du weapon 32b (relatif d), pour exclure de la zone candidate
	wpn    string
	tms    int
}

func collectMelee() []mev {
	var out []mev
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		total := len(d) * 8
		for bp := 0; bp+160 < total; bp++ {
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

// overlapWeapon : true si le champ [o..o+W) recouvre la zone weapon 32b (woff..woff+32) ou type 8b (anchor+76..+84).
func overlaps(o, w, anchor, woff int) bool {
	// weapon zone
	if o < woff+32 && o+w > woff {
		return true
	}
	// type zone
	t0 := anchor + 76
	if o < t0+8 && o+w > t0 {
		return true
	}
	return false
}

func nearIKE(tms int) bool {
	s := float64(tms) / 1000
	for _, k := range ikeKills {
		if s >= k-2 && s <= k+2 {
			return true
		}
	}
	return false
}

type result struct {
	rel        int // offset relatif à anchor
	w          int
	distinct   int
	maxv       int
	total      int
	dist       map[int]int
	hammerN    int // # events marteau près kills IKE
	hammerIKE  int // # de ces events où champ==4
	hammerVals map[int]int
}

func main() {
	buildCat()
	evs := collectMelee()
	hammers := 0
	for _, e := range evs {
		if e.typ == 0x47 {
			hammers++
		}
	}
	fmt.Printf("=== MELEE events: %d (dont GravityHammer type0x47: %d) ===\n", len(evs), hammers)

	// Diagnostic dédié rel=+23 W=5 : distribution séparée hammer (0x47) vs autres types.
	fmt.Printf("=== rel=+23 W=5 : distribution par type de melee ===\n")
	d47 := map[int]int{}
	dOther := map[int]int{}
	mx47, mxOther := 0, 0
	for _, e := range evs {
		o := e.anchor + 23
		if o < 0 || overlaps(o, 5, e.anchor, e.woff) {
			continue
		}
		v := int(bitsAt(e.d, o, 5))
		if e.typ == 0x47 {
			d47[v]++
			if v > mx47 {
				mx47 = v
			}
		} else {
			dOther[v]++
			if v > mxOther {
				mxOther = v
			}
		}
	}
	fmt.Printf("  type0x47 (hammer): maxv=%d distinct=%d  %v\n", mx47, len(d47), fmtMap(d47))
	fmt.Printf("  autres types     : maxv=%d distinct=%d  %v\n", mxOther, len(dOther), fmtMap(dOther))

	widths := []int{5, 4, 6}
	for _, W := range widths {
		fmt.Printf("\n############ LARGEUR W=%d bits ############\n", W)
		var valid []result
		// rel de -16 (avant ancre) à +120
		for rel := -16; rel <= 120; rel++ {
			r := result{rel: rel, w: W, dist: map[int]int{}, hammerVals: map[int]int{}}
			skip := false
			for _, e := range evs {
				o := e.anchor + rel
				if o < 0 {
					skip = true
					break
				}
				if overlaps(o, W, e.anchor, e.woff) {
					skip = true
					break
				}
				v := int(bitsAt(e.d, o, W))
				r.dist[v]++
				r.total++
				if v > r.maxv {
					r.maxv = v
				}
				if e.typ == 0x47 && nearIKE(e.tms) {
					r.hammerN++
					r.hammerVals[v]++
					if v == 4 {
						r.hammerIKE++
					}
				}
			}
			if skip || r.total == 0 {
				continue
			}
			r.distinct = len(r.dist)
			// VALIDE : borné 0-7 et non dégénéré (>1 valeur, pas 100% en une seule)
			if r.maxv <= 7 && r.distinct >= 2 {
				valid = append(valid, r)
			}
		}
		// trier : d'abord hammerIKE desc, puis distinct proche de 8
		sort.Slice(valid, func(a, b int) bool {
			if valid[a].hammerIKE != valid[b].hammerIKE {
				return valid[a].hammerIKE > valid[b].hammerIKE
			}
			da := absI(valid[a].distinct - 8)
			db := absI(valid[b].distinct - 8)
			return da < db
		})
		fmt.Printf("-- %d offsets candidats valides (borné 0-7, >=2 distinct) --\n", len(valid))
		lim := len(valid)
		if lim > 40 {
			lim = 40
		}
		for _, r := range valid[:lim] {
			fmt.Printf("  rel=%+4d  distinct=%2d maxv=%d  hammerNearIKE=%d hammer==pi4=%d  hammerVals=%v\n",
				r.rel, r.distinct, r.maxv, r.hammerN, r.hammerIKE, fmtMap(r.hammerVals))
		}
		// détailler le meilleur
		if len(valid) > 0 {
			best := valid[0]
			fmt.Printf("  >>> MEILLEUR rel=%+d (W=%d): distribution complète:\n", best.rel, W)
			for i := 0; i <= best.maxv; i++ {
				if best.dist[i] > 0 {
					fmt.Printf("       v=%d (%s) x%d\n", i, pi[i], best.dist[i])
				}
			}
		}
	}

	// Dump TOUS les marteaux type0x47 triés par temps, en lisant rel=+23 (W=5) — le meilleur candidat.
	// Objectif : voir si les marteaux se regroupent autour des kills IKE et quel pidx ils portent.
	sort.Slice(evs, func(a, b int) bool { return evs[a].tms < evs[b].tms })
	fmt.Printf("\n=== TOUS les marteaux (type0x47) triés, lecture rel=+23 W=5 (meilleur cand) ===\n")
	fmt.Printf("    kills narrés IKE->JGtm: 115.5 / 292.5 / 355.7 / 375.1 s\n")
	for _, e := range evs {
		if e.typ != 0x47 {
			continue
		}
		o := e.anchor + 23
		v := -1
		if o >= 0 && !overlaps(o, 5, e.anchor, e.woff) {
			v = int(bitsAt(e.d, o, 5))
		}
		mark := ""
		if nearIKE(e.tms) {
			mark = "  <== près kill IKE"
		}
		fmt.Printf("   t=%7.1fs  rel+23=%d(%-16s)  %s%s\n", float64(e.tms)/1000, v, pi[v], e.wpn, mark)
	}

	// Pour chaque offset W=5 à distribution 0-7 propre (distinct=8), évaluer le pidx sur TOUS
	// les marteaux dans une fenêtre élargie ±4s autour des kills IKE et compter combien donnent pi4.
	fmt.Printf("\n=== Score IKE élargi (±4s) par offset W=5 distinct>=7 ===\n")
	type cand struct {
		rel, n4, ntot int
		vals          map[int]int
	}
	cands := []int{}
	for rel := -16; rel <= 120; rel++ {
		// reconstituer distinct rapide
		dist := map[int]int{}
		ok := true
		mx := 0
		for _, e := range evs {
			o := e.anchor + rel
			if o < 0 || overlaps(o, 5, e.anchor, e.woff) {
				ok = false
				break
			}
			v := int(bitsAt(e.d, o, 5))
			dist[v]++
			if v > mx {
				mx = v
			}
		}
		if ok && mx <= 7 && len(dist) >= 7 {
			cands = append(cands, rel)
		}
	}
	near4 := func(tms int) bool {
		s := float64(tms) / 1000
		for _, k := range ikeKills {
			if s >= k-4 && s <= k+4 {
				return true
			}
		}
		return false
	}
	var cs []cand
	for _, rel := range cands {
		c := cand{rel: rel, vals: map[int]int{}}
		for _, e := range evs {
			if e.typ != 0x47 || !near4(e.tms) {
				continue
			}
			o := e.anchor + rel
			if o < 0 || overlaps(o, 5, e.anchor, e.woff) {
				continue
			}
			v := int(bitsAt(e.d, o, 5))
			c.vals[v]++
			c.ntot++
			if v == 4 {
				c.n4++
			}
		}
		cs = append(cs, c)
	}
	sort.Slice(cs, func(a, b int) bool { return cs[a].n4 > cs[b].n4 })
	for _, c := range cs {
		fmt.Printf("   rel=%+4d  marteauxPrèsIKE±4s=%d  ==pi4:%d  vals=%v\n", c.rel, c.ntot, c.n4, fmtMap(c.vals))
	}

	// Vue par kill narré : tous les marteaux dans ±8s, lecture rel=+23.
	fmt.Printf("\n=== Marteaux ±8s autour de chaque kill narré IKE->JGtm (rel=+23 W=5) ===\n")
	for _, k := range ikeKills {
		fmt.Printf("-- kill narré %.1fs --\n", k)
		for _, e := range evs {
			if e.typ != 0x47 {
				continue
			}
			s := float64(e.tms) / 1000
			if s < k-8 || s > k+8 {
				continue
			}
			o := e.anchor + 23
			v := int(bitsAt(e.d, o, 5))
			fmt.Printf("    t=%7.1fs (Δ%+5.1f)  rel+23=%d(%-16s) %s\n", s, s-k, v, pi[v], e.wpn)
		}
	}
}

func absI(x int) int {
	if x < 0 {
		return -x
	}
	return x
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
