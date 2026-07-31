// tmp_realrange — recherche HONNÊTE de la position i0 keyframe des 8 bipeds, jugée par
// la BOÎTE ORACLE (pas par la range de déquant, pas par le simple gate). On balaie
// (offset, width) : pour chaque candidat on déquantifie 3 axes avec la VRAIE range CE
// (filmdec.QuantRangeCEBiped) et on compte combien des 8 bipeds tombent DANS la boîte
// oracle x[-6.33,35.70] y[-25.14,27.50] z[-4.20,7.08], distincts et non dégénérés
// (chaque axe a un étalement réel, pas une constante).
//
// Le juge = la boîte oracle. Un offset qui donne gate 8/8 mais X sur 2 buckets ou Z
// constant est REJETÉ (dégénéré). THROWAWAY / validation.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_realrange
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const film = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

// boîte oracle (min/max mesurés sur ce_pos_oracle.csv, tous slots gameplay)
var oracleBox = [3][2]float64{{-6.33, 35.70}, {-25.14, 27.50}, {-4.20, 7.08}}

func inflate(p string) []byte {
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); (e2 == nil || e2 == io.ErrUnexpectedEOF) && len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

func framePayload(d []byte, want uint16) []byte {
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		if typ == want {
			return d[off+16 : off+16+sz]
		}
		off += 16 + sz
	}
	return nil
}

func readBits(buf []byte, pos, n int) uint64 {
	var r uint64
	for i := 0; i < n; i++ {
		p := pos + i
		var bit uint64
		if idx := p >> 3; idx < len(buf) {
			bit = uint64(buf[idx]>>(7-uint(p&7))) & 1
		}
		r = r<<1 | bit
	}
	return r
}

// decodeAxes lit 3 axes de largeur w à partir du bit axStart et déquantifie via la range CE.
func decodeAxes(pay []byte, axStart, w int) [3]float64 {
	rng := filmdec.WorldPositionRange
	var v [3]float64
	q := axStart
	for i := 0; i < 3; i++ {
		word := readBits(pay, q, w)
		q += w
		scale := float64(uint64(1) << uint(w))
		step := float64(rng[i].Max-rng[i].Min) / scale
		v[i] = float64(word)*step + float64(rng[i].Min) + step*0.5
	}
	return v
}

// gate0 : precHigh(0)+idxSel(0)+idx(1 bit ==0) à p ; retourne le bit de début des axes.
func gate0(pay []byte, p int) (axStart int, ok bool) {
	if readBits(pay, p, 1) != 0 {
		return 0, false
	}
	if readBits(pay, p+1, 1) != 0 {
		return 0, false
	}
	if readBits(pay, p+2, 1) != 0 { // indexW=1
		return 0, false
	}
	return p + 3, true
}

func inBox(v [3]float64) bool {
	for i := 0; i < 3; i++ {
		if v[i] < oracleBox[i][0] || v[i] > oracleBox[i][1] {
			return false
		}
	}
	return true
}

// evalCandidate : à (offset depuis stateBit, width w), décode les 8 bipeds.
func evalCandidate(pay []byte, stateBits []int, off, w int, withGate bool) (nIn, nDist int, spread [3]float64, vals map[int][3]float64) {
	vals = map[int][3]float64{}
	var axmin, axmax [3]float64
	for i := 0; i < 3; i++ {
		axmin[i] = 1e18
		axmax[i] = -1e18
	}
	seen := map[[3]int]bool{}
	for si, sb := range stateBits {
		p := sb + off
		axStart := p
		if withGate {
			as, ok := gate0(pay, p)
			if !ok {
				continue
			}
			axStart = as
		}
		v := decodeAxes(pay, axStart, w)
		vals[si] = v
		if inBox(v) {
			nIn++
		}
		key := [3]int{int(v[0] * 4), int(v[1] * 4), int(v[2] * 4)}
		seen[key] = true
		for i := 0; i < 3; i++ {
			if v[i] < axmin[i] {
				axmin[i] = v[i]
			}
			if v[i] > axmax[i] {
				axmax[i] = v[i]
			}
		}
	}
	nDist = len(seen)
	for i := 0; i < 3; i++ {
		spread[i] = axmax[i] - axmin[i]
	}
	return
}

func main() {
	pay := framePayload(inflate(film+"/chunk_02.bin"), 2)
	if pay == nil {
		fmt.Println("pas de frame type-2")
		return
	}
	recs := filmdec.WalkKeyframeWorld(pay)
	hdr := map[int]int{}
	for _, r := range recs {
		if r.TI == 35 && r.Slot >= 512 && r.Slot <= 519 {
			hdr[r.Slot] = r.Bit
		}
	}
	slots := make([]int, 0, len(hdr))
	for s := range hdr {
		slots = append(slots, s)
	}
	sort.Ints(slots)
	stateBits := make([]int, len(slots))
	for i, s := range slots {
		stateBits[i] = hdr[s] + 64
	}
	fmt.Printf("range CE = %v\n", filmdec.WorldPositionRange)
	fmt.Printf("boîte oracle x%v y%v z%v\n", oracleBox[0], oracleBox[1], oracleBox[2])
	fmt.Printf("bipeds=%d slots=%v\n\n", len(slots), slots)

	type cand struct {
		off, w, nIn, nDist int
		gate               bool
		perm               [3]int
		spread             [3]float64
	}
	perms := [][3]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}}
	var best []cand
	maxIn := 0
	var maxCfg cand
	for _, gate := range []bool{true, false} {
		for w := 6; w <= 16; w++ {
			for off := 0; off <= 400; off++ {
				for _, pm := range perms {
					nIn, nDist, spread := evalPerm(pay, stateBits, off, w, gate, pm)
					if nIn > maxIn {
						maxIn = nIn
						maxCfg = cand{off, w, nIn, nDist, gate, pm, spread}
					}
					nonDeg := spread[0] > 3 && spread[1] > 3 && spread[2] > 1
					if nIn >= 6 && nDist >= 6 && nonDeg {
						best = append(best, cand{off, w, nIn, nDist, gate, pm, spread})
					}
				}
			}
		}
	}
	sort.Slice(best, func(i, j int) bool {
		if best[i].nIn != best[j].nIn {
			return best[i].nIn > best[j].nIn
		}
		return best[i].nDist > best[j].nDist
	})
	fmt.Printf("=== candidats in-box>=6 distinct>=6 non-dégénérés (avec permutations d'axes) : %d ===\n", len(best))
	for i, c := range best {
		if i >= 30 {
			fmt.Printf("  ... (%d de plus)\n", len(best)-30)
			break
		}
		fmt.Printf("  gate=%v off=+%-3d w=%-2d perm=%v inBox=%d/8 distinct=%d spread=(%.1f,%.1f,%.1f)\n",
			c.gate, c.off, c.w, c.perm, c.nIn, c.nDist, c.spread[0], c.spread[1], c.spread[2])
	}
	fmt.Printf("\n=== MAX in-box atteignable (toutes configs) = %d/8 : gate=%v off=+%d w=%d perm=%v spread=(%.1f,%.1f,%.1f) ===\n",
		maxIn, maxCfg.gate, maxCfg.off, maxCfg.w, maxCfg.perm, maxCfg.spread[0], maxCfg.spread[1], maxCfg.spread[2])
	_, _, _, vals := evalCandidate(pay, stateBits, maxCfg.off, maxCfg.w, maxCfg.gate)
	for i, s := range slots {
		if v, ok := vals[i]; ok {
			p := [3]float64{v[maxCfg.perm[0]], v[maxCfg.perm[1]], v[maxCfg.perm[2]]}
			mark := ""
			if inBox(p) {
				mark = " [in-box]"
			}
			fmt.Printf("  slot=%d pos=(%.3f, %.3f, %.3f)%s\n", s, p[0], p[1], p[2], mark)
		}
	}
}

func evalPerm(pay []byte, stateBits []int, off, w int, gate bool, pm [3]int) (nIn, nDist int, spread [3]float64) {
	_, _, _, vals := evalCandidate(pay, stateBits, off, w, gate)
	var axmin, axmax [3]float64
	for i := 0; i < 3; i++ {
		axmin[i] = 1e18
		axmax[i] = -1e18
	}
	seen := map[[3]int]bool{}
	for _, raw := range vals {
		v := [3]float64{raw[pm[0]], raw[pm[1]], raw[pm[2]]}
		if inBox(v) {
			nIn++
		}
		seen[[3]int{int(v[0] * 4), int(v[1] * 4), int(v[2] * 4)}] = true
		for i := 0; i < 3; i++ {
			if v[i] < axmin[i] {
				axmin[i] = v[i]
			}
			if v[i] > axmax[i] {
				axmax[i] = v[i]
			}
		}
	}
	nDist = len(seen)
	for i := 0; i < 3; i++ {
		spread[i] = axmax[i] - axmin[i]
	}
	return
}
