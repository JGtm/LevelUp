// tmp_grenval — THROWAWAY : lit les VALEURS de i22/i47/i48/i56 directement dans le buffer
// a CompResult.StartBit (aucune modification du decodeur), et les croise avec les events de
// LANCER de grenade (marqueur 0x4c0c00) pour produire le TEMOIN : le compte doit DECROITRE
// chez le lanceur et RESTER STABLE chez les autres.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_grenval [chunkLo chunkHi]
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

const (
	cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
	idLow = 11
	t0Us  = uint64(4537898226)
)

var grenadeIDs = map[uint32]string{0xB0171062: "Frag", 0xC0E34C44: "Plasma", 0x3B2567D4: "Shock", 0x9212E428: "Spike"}
var piName = map[int]string{0: "whiteknight2519", 1: "JAVIERLOLITO540", 2: "JGtm", 3: "LORD PEINX13", 4: "IKE ILYA", 5: "Akatsuki fire17", 6: "aldusbroncus", 7: "VitaminA1688"}

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

func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p < 0 || p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

// --- lectures des 4 composants a partir du buffer, a StartBit ---

type grenCounts struct {
	n int
	c [8]int
}

func readI22(d []byte, bp int) grenCounts {
	var g grenCounts
	g.n = int(bitsAt(d, bp, 3))
	for i := 0; i < g.n && i < 8; i++ {
		g.c[i] = int(bitsAt(d, bp+3+8*i, 8))
	}
	return g
}

type i47v struct{ a, b int } // R(6), R(3)

func readI47(d []byte, bp int) i47v { return i47v{int(bitsAt(d, bp, 6)), int(bitsAt(d, bp+6, 3))} }

type i48v struct {
	slot int  // R(3)
	gate int  // R(1)
	idx  int  // R(6) si gate==0
	has  bool // gate==0
}

func readI48(d []byte, bp int) i48v {
	v := i48v{slot: int(bitsAt(d, bp, 3)), gate: int(bitsAt(d, bp+3, 1))}
	if v.gate == 0 {
		v.idx, v.has = int(bitsAt(d, bp+4, 6)), true
	}
	return v
}

type i56v struct {
	mask int
	ch   [3]int // 0x7F par defaut (plein) si bit du masque a 0
}

func readI56(d []byte, bp int) i56v {
	v := i56v{mask: int(bitsAt(d, bp, 3))}
	p := bp + 3
	for i := 0; i < 3; i++ {
		if v.mask&(1<<uint(i)) != 0 {
			v.ch[i] = int(bitsAt(d, p, 7))
			p += 7
		} else {
			v.ch[i] = 0x7F
		}
	}
	return v
}

// --- events de lancer ---

type throwEv struct {
	tms  int
	kind string
	pidx int
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

func scanThrows(lo, hi int) []throwEv {
	var out []throwEv
	for n := lo; n <= hi; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		total := len(d) * 8
		for bp := 0; bp+110 < total; bp++ {
			if bitsAt(d, bp, 24) != 0x4c0c00 {
				continue
			}
			gid := uint32(bitsAt(d, bp+24, 32))
			name, ok := grenadeIDs[gid]
			if !ok {
				continue
			}
			pidx := int(bitsAt(d, bp+24+32+47, 5))
			tms, okt := tsAtBit(d, bp)
			if !okt {
				continue
			}
			out = append(out, throwEv{tms, name, pidx})
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].tms < out[b].tms })
	return out
}

// --- echantillons temporels par slot ---

type sample struct {
	tms                       int
	slot                      uint32
	g                         grenCounts
	i47                       i47v
	i48                       i48v
	i56                       i56v
	hasG, has47, has48, has56 bool
}

func main() {
	lo, hi := 1, 27
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[1], "%d", &lo)
		fmt.Sscanf(os.Args[2], "%d", &hi)
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	kfChunk := inflate(cache + "/chunk_02.bin")
	var kf []byte
	for _, p := range filmdec.WalkPackets(kfChunk) {
		if p.Type == filmdec.PacketTypeKeyframe {
			kf = p.Payload(kfChunk)
			break
		}
	}
	if kf == nil {
		panic("keyframe introuvable")
	}
	binds := filmdec.WalkKeyframeWorld(kf)
	filmdec.SetRecordStateParam(2)
	cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: idLow}
	w := filmdec.NewWorld(reg)
	for _, b := range binds {
		w.BindFull(uint32((b.Gen<<30)|b.Slot), uint32(b.TI))
	}

	var samples []sample
	for c := lo; c <= hi; c++ {
		chunk := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, c))
		if chunk == nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta {
				continue
			}
			pay := p.Payload(chunk)
			br := filmdec.NewBitReader(pay)
			recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
			tms := int((p.TimestampUS - t0Us) / 1000)
			for _, r := range recs {
				if r.TypeIndex != filmdec.BipedTypeIndex || r.Slot < 512 || r.Slot > 519 {
					continue
				}
				s := sample{tms: tms, slot: r.Slot}
				for _, cr := range r.Trace.Comps {
					if !cr.Ported {
						continue
					}
					switch cr.Index {
					case 22:
						s.g, s.hasG = readI22(pay, cr.StartBit), true
					case 47:
						s.i47, s.has47 = readI47(pay, cr.StartBit), true
					case 48:
						s.i48, s.has48 = readI48(pay, cr.StartBit), true
					case 56:
						s.i56, s.has56 = readI56(pay, cr.StartBit), true
					}
				}
				if s.hasG || s.has47 || s.has48 || s.has56 {
					samples = append(samples, s)
				}
			}
		}
	}
	sort.SliceStable(samples, func(a, b int) bool { return samples[a].tms < samples[b].tms })
	fmt.Printf("=== tmp_grenval (chunks %d..%d) : %d echantillons ===\n", lo, hi, len(samples))

	// --- distribution i22 ---
	nHist := map[int]int{}
	valHist := map[int]map[int]int{}
	for _, s := range samples {
		if !s.hasG {
			continue
		}
		nHist[s.g.n]++
		for i := 0; i < s.g.n; i++ {
			if valHist[i] == nil {
				valHist[i] = map[int]int{}
			}
			valHist[i][s.g.c[i]]++
		}
	}
	fmt.Println("\n-- i22 unit-grenade-counts : histogramme du count R(3) --")
	var ks []int
	for k := range nHist {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	for _, k := range ks {
		fmt.Printf("   count=%d : %d\n", k, nHist[k])
	}
	fmt.Println("-- valeurs R(8) par emplacement (top 10 par emplacement) --")
	var slotsIdx []int
	for k := range valHist {
		slotsIdx = append(slotsIdx, k)
	}
	sort.Ints(slotsIdx)
	for _, i := range slotsIdx {
		type kv struct{ v, n int }
		var a []kv
		for v, n := range valHist[i] {
			a = append(a, kv{v, n})
		}
		sort.Slice(a, func(x, y int) bool { return a[x].n > a[y].n })
		fmt.Printf("   emplacement %d : ", i)
		for j := 0; j < 10 && j < len(a); j++ {
			fmt.Printf("%d(x%d) ", a[j].v, a[j].n)
		}
		fmt.Println()
	}

	// --- distribution i47/i48/i56 ---
	h47a, h47b := map[int]int{}, map[int]int{}
	h48s, h48i := map[int]int{}, map[int]int{}
	h56m := map[int]int{}
	h56v := map[int]int{}
	for _, s := range samples {
		if s.has47 {
			h47a[s.i47.a]++
			h47b[s.i47.b]++
		}
		if s.has48 {
			h48s[s.i48.slot]++
			if s.i48.has {
				h48i[s.i48.idx]++
			} else {
				h48i[-1]++
			}
		}
		if s.has56 {
			h56m[s.i56.mask]++
			for i := 0; i < 3; i++ {
				h56v[s.i56.ch[i]]++
			}
		}
	}
	dump := func(title string, m map[int]int, top int) {
		type kv struct{ v, n int }
		var a []kv
		for v, n := range m {
			a = append(a, kv{v, n})
		}
		sort.Slice(a, func(x, y int) bool { return a[x].n > a[y].n })
		fmt.Printf("   %-28s ", title)
		for j := 0; j < top && j < len(a); j++ {
			fmt.Printf("%d(x%d) ", a[j].v, a[j].n)
		}
		fmt.Printf("  [%d valeurs distinctes]\n", len(a))
	}
	fmt.Println("\n-- i47 biped-desired-grenade-set --")
	dump("R(6) =", h47a, 12)
	dump("R(3) =", h47b, 12)
	fmt.Println("-- i48 biped-desired-ability-set --")
	dump("R(3) slot =", h48s, 12)
	dump("R(6) index (-1=porte1) =", h48i, 12)
	fmt.Println("-- i56 biped-spartan-ability-energy --")
	dump("masque R(3) =", h56m, 12)
	dump("charges (0x7F=127 plein) =", h56v, 14)

	// --- TEMOIN : lancers vs decroissance du compte ---
	throws := scanThrows(0, 27)
	fmt.Printf("\n=== TEMOIN : %d lancers de grenade (marqueur 0x4c0c00) ===\n", len(throws))
	tHist := map[int]int{}
	for _, t := range throws {
		tHist[t.pidx]++
	}
	var pk []int
	for k := range tHist {
		pk = append(pk, k)
	}
	sort.Ints(pk)
	for _, k := range pk {
		fmt.Printf("   pidx=%d (%-16s) x%d\n", k, piName[k], tHist[k])
	}

	// timeline par slot
	byslot := map[uint32][]sample{}
	for _, s := range samples {
		if s.hasG {
			byslot[s.slot] = append(byslot[s.slot], s)
		}
	}
	type cell struct{ dec, stable, nodata int }
	mat := map[int]map[uint32]*cell{}
	const winMs = 700 // fenetre de correlation lancer -> echantillon
	for _, t := range throws {
		if mat[t.pidx] == nil {
			mat[t.pidx] = map[uint32]*cell{}
		}
		for sl := uint32(512); sl <= 519; sl++ {
			if mat[t.pidx][sl] == nil {
				mat[t.pidx][sl] = &cell{}
			}
			ss := byslot[sl]
			var before, after *sample
			for i := range ss {
				if ss[i].tms <= t.tms && ss[i].tms >= t.tms-winMs {
					before = &ss[i]
				}
				if ss[i].tms > t.tms && ss[i].tms <= t.tms+winMs {
					after = &ss[i]
					break
				}
			}
			if before == nil || after == nil {
				mat[t.pidx][sl].nodata++
				continue
			}
			sumB, sumA := 0, 0
			for i := 0; i < before.g.n; i++ {
				sumB += before.g.c[i]
			}
			for i := 0; i < after.g.n; i++ {
				sumA += after.g.c[i]
			}
			if sumA < sumB {
				mat[t.pidx][sl].dec++
			} else {
				mat[t.pidx][sl].stable++
			}
		}
	}
	fmt.Printf("\n-- MATRICE pidx x slot : nb de lancers ou la SOMME des comptes DECROIT (fenetre +-%dms) --\n", winMs)
	fmt.Printf("%-6s", "pidx")
	for sl := uint32(512); sl <= 519; sl++ {
		fmt.Printf(" %7d", sl)
	}
	fmt.Println("   (n lancers)")
	for _, p := range pk {
		fmt.Printf("%-6d", p)
		for sl := uint32(512); sl <= 519; sl++ {
			c := mat[p][sl]
			if c == nil {
				fmt.Printf(" %7s", "-")
				continue
			}
			fmt.Printf(" %7d", c.dec)
		}
		fmt.Printf("   %d\n", tHist[p])
	}
	// --- CRITERE ABSOLU : valeurs physiquement impossibles ---
	// Halo Infinite : au plus 2 grenades par type (4 avec le mod Grenadier), et au plus
	// 4 types. Toute valeur R(8) > 4 est IMPOSSIBLE pour un compte de grenades.
	var nVal, nImp, nCnt, nCntImp int
	for _, s := range samples {
		if !s.hasG {
			continue
		}
		nCnt++
		if s.g.n > 4 {
			nCntImp++
		}
		for i := 0; i < s.g.n; i++ {
			nVal++
			if s.g.c[i] > 4 {
				nImp++
			}
		}
	}
	fmt.Println("\n=== CRITERE ABSOLU (aucune statistique de reference requise) ===")
	fmt.Printf("   i22 count R(3) > 4 (plus de 4 types de grenade) : %d / %d = %.2f%%\n",
		nCntImp, nCnt, 100*float64(nCntImp)/float64(maxi(nCnt, 1)))
	fmt.Printf("   i22 valeur R(8) > 4 (compte de grenades impossible) : %d / %d = %.2f%%\n",
		nImp, nVal, 100*float64(nImp)/float64(maxi(nVal, 1)))
	var nCh, nChFull, nChRamp int
	for _, s := range samples {
		if !s.has56 {
			continue
		}
		for i := 0; i < 3; i++ {
			nCh++
			if s.i56.ch[i] == 127 {
				nChFull++
			} else if s.i56.ch[i] > 0 && s.i56.ch[i] < 127 {
				nChRamp++
			}
		}
	}
	fmt.Printf("   i56 charges : %d lues ; pleines (127) %d (%.2f%%) ; intermediaires %d (%.2f%%)\n",
		nCh, nChFull, 100*float64(nChFull)/float64(maxi(nCh, 1)), nChRamp, 100*float64(nChRamp)/float64(maxi(nCh, 1)))

	// --- TEMOIN NEGATIF chiffre : combien de slots decroissent par lancer ? ---
	// Attendu si le decodage etait reel : EXACTEMENT 1 (le lanceur), 0 pour les 7 autres.
	decHist := map[int]int{}
	for _, t := range throws {
		n := 0
		for sl := uint32(512); sl <= 519; sl++ {
			ss := byslot[sl]
			var before, after *sample
			for i := range ss {
				if ss[i].tms <= t.tms && ss[i].tms >= t.tms-winMs {
					before = &ss[i]
				}
				if ss[i].tms > t.tms && ss[i].tms <= t.tms+winMs {
					after = &ss[i]
					break
				}
			}
			if before == nil || after == nil {
				continue
			}
			sumB, sumA := 0, 0
			for i := 0; i < before.g.n; i++ {
				sumB += before.g.c[i]
			}
			for i := 0; i < after.g.n; i++ {
				sumA += after.g.c[i]
			}
			if sumA < sumB {
				n++
			}
		}
		decHist[n]++
	}
	fmt.Println("\n-- TEMOIN NEGATIF : nb de slots dont la somme DECROIT autour d'un lancer --")
	fmt.Println("   (attendu si le decodage etait reel : 1 pour tous les lancers)")
	var dk []int
	for k := range decHist {
		dk = append(dk, k)
	}
	sort.Ints(dk)
	for _, k := range dk {
		fmt.Printf("   %d slot(s) decroissent : %d lancers\n", k, decHist[k])
	}

	fmt.Println("\n-- extrait timeline slot 512 (30 premiers echantillons i22) --")
	for i, s := range byslot[512] {
		if i >= 30 {
			break
		}
		fmt.Printf("   t=%7.2fs n=%d c=%v\n", float64(s.tms)/1000, s.g.n, s.g.c[:s.g.n])
	}
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}
