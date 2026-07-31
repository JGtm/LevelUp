// tmp_e6cause — LANE EMPIRIQUE : le kill-event 0xE6 du FILM porte-t-il un CODE DE CAUSE par-kill ?
//
// Question : après décodage RIGOUREUX du 0xE6 à sa VRAIE position (pas bit 93 fixe), un de ses
// champs distingue-t-il mêlée / grenade / arme-à-feu ? Vérité-terrain SAME-CLOCK (frame ts µs) :
//   - firearm : record de dégât 0xd2 (attaquant idx via parsePreamble(24)+famille) proche du kill.
//   - grenade : event lancer 0x4c0c00 (thrower idx) proche du kill.
//   - mêlée   : les 2 kills mêlée validés 000d5950 (Diminisher LORD PEINX(3)->Akatsuki(5) 147.6s,
//     Rushdown Akatsuki(5)->LORD PEINX(3) 323.9s) + scan marqueurs 0x532/0x534/0x535.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_e6cause [film] [mode]
//
//	modes : landscape (defaut) | calib | cause | melee
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

const root = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`
const sfx = uint32(0x42c9679f)

// t0 frame-clock par film (µs). 000d5950 = 4537898226 (cf. tmp_meleegrenade). Défaut : min ts observé.
var t0ByFilm = map[string]uint64{"000d5950": 4537898226}

// roster 000d5950 : idx -> gamertag (cf. tmp_meleegrenade).
var pi = map[int]string{0: "whiteknight2519", 1: "JAVIERLOLITO540", 2: "JGtm", 3: "LORD PEINX13", 4: "IKE ILYA", 5: "Akatsuki fire17", 6: "aldusbroncus", 7: "VitaminA1688"}

var grenades = map[uint32]string{0xB0171062: "Frag", 0xC0E34C44: "Plasma", 0x3B2567D4: "Shock", 0x9212E428: "Spike"}

var h32 = map[uint32]string{}

func buildCat() {
	for id, n := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = n
	}
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

type pkt struct {
	marker byte
	ts     uint64
	pl     []byte
}

// dmgEv : record de dégât firearm 0xd2 (attaquant idx, famille high32, ts).
type dmgEv struct {
	atk    int
	family uint32
	ts     uint64
}

// grenEv : lancer de grenade (thrower idx, type, ts).
type grenEv struct {
	pidx int
	name string
	ts   uint64
}

// e6 : un paquet kill-event 0xE6 (ts + payload).
type e6 struct {
	ts uint64
	pl []byte
}

func collect(m string) (e6s []e6, dmgs []dmgEv, grens []grenEv, sizes map[int]int) {
	cache := root + "/" + m
	sizes = map[int]int{}
	for ch := 0; ch <= 41; ch++ {
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
			if typ != 0 || len(pl) == 0 {
				continue
			}
			switch pl[0] {
			case 0xE6:
				cp := make([]byte, len(pl))
				copy(cp, pl)
				e6s = append(e6s, e6{ts, cp})
				sizes[sz]++
			case 0xd2:
				rec, ok := parsePreamble(pl, 24)
				if ok && rec.attacker >= 0 && damageClass(rec.variant) == dmgFirearm {
					dmgs = append(dmgs, dmgEv{rec.attacker, rec.family, ts})
				}
			}
		}
		// grenade : scan 24-bit marker 0x4c0c00 dans le chunk brut.
		total := len(d) * 8
		for bp := 0; bp+110 < total; bp++ {
			if bitsAt(d, bp, 24) != 0x4c0c00 {
				continue
			}
			gid := uint32(bitsAt(d, bp+24, 32))
			gname, ok := grenades[gid]
			if !ok {
				continue
			}
			pidx := int(bitsAt(d, bp+24+32+47, 5))
			gts, okt := tsAtBit(d, bp)
			if !okt {
				continue
			}
			grens = append(grens, grenEv{pidx, gname, gts})
		}
	}
	sort.Slice(e6s, func(i, j int) bool { return e6s[i].ts < e6s[j].ts })
	sort.Slice(dmgs, func(i, j int) bool { return dmgs[i].ts < dmgs[j].ts })
	sort.Slice(grens, func(i, j int) bool { return grens[i].ts < grens[j].ts })
	return
}

// tsAtBit : ts µs du paquet contenant le bit bp (pour les events scannés en brut).
func tsAtBit(d []byte, bp int) (uint64, bool) {
	pos := bp >> 3
	off := 0
	for off+16 <= len(d) {
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if pos >= off+16 && pos < off+16+sz {
			return ts, true
		}
		off += 16 + sz
	}
	return 0, false
}

// --- parsePreamble (record de dégât, base=24) — porté de tmp_kwval ---
type dmgRecord struct {
	attacker   int
	family     uint32
	variant    uint32
	variantPos int
}

func parsePreamble(pl []byte, base int) (dmgRecord, bool) {
	var r dmgRecord
	hi := len(pl) * 8
	bp := base
	if base < 0 || base+13 > hi {
		return r, false
	}
	rd := func(n int) uint32 { v := uint32(bitsAt(pl, bp, n)); bp += n; return v }
	rd(1)
	rd(1)
	rd(8)
	if rd(1) == 0 {
		r.attacker = int(rd(5))
	} else {
		r.attacker = -1
	}
	if rd(1) == 0 {
		rd(2)
	}
	if rd(1) == 1 {
		r.family = rd(32)
	} else {
		r.family = 0xffffffff
	}
	if bp+32 > hi {
		return r, false
	}
	r.variantPos = bp
	r.variant = rd(32)
	return r, true
}

const (
	dmgOther     = -1
	dmgFirearm   = 0
	dmgMelee     = 1
	dmgGrenade   = 2
	firearmVar24 = 0x42C967
	meleeVar24   = 0x592CF3
	grenadeVar24 = 0x164B3C
)

func damageClass(variant uint32) int {
	switch variant >> 8 {
	case firearmVar24:
		return dmgFirearm
	case meleeVar24:
		return dmgMelee
	case grenadeVar24:
		return dmgGrenade
	default:
		return dmgOther
	}
}

// secs : ts frame -> secondes depuis t0 (film 000d5950 : t0 connu ; sinon relatif au min).
func secs(m string, ts, t0 uint64) float64 {
	return float64(int64(ts)-int64(t0)) / 1e6
}

func main() {
	buildCat()
	m := "000d5950"
	if len(os.Args) > 1 {
		m = os.Args[1]
	}
	mode := "landscape"
	if len(os.Args) > 2 {
		mode = os.Args[2]
	}
	e6s, dmgs, grens, sizes := collect(m)
	t0 := t0ByFilm[m]
	if t0 == 0 && len(e6s) > 0 {
		t0 = e6s[0].ts
	}
	switch mode {
	case "landscape":
		runLandscape(m, e6s, dmgs, grens, sizes, t0)
	case "calib":
		runCalib(m, e6s, dmgs, t0)
	case "cause":
		runCause(m, e6s, dmgs, grens, t0)
	case "melee":
		runMeleeScan(m, t0)
	default:
		fmt.Println("mode inconnu")
	}
}

func runLandscape(m string, e6s []e6, dmgs []dmgEv, grens []grenEv, sizes map[int]int, t0 uint64) {
	fmt.Printf("=== LANDSCAPE %s : %d kill-events 0xE6 | %d dégâts firearm 0xd2 | %d grenades ===\n",
		m, len(e6s), len(dmgs), len(grens))
	fmt.Printf("tailles 0xE6 (octets) : %s\n", topInt(sizes, 12))
	// dump des 12 premiers 0xE6 : hex + readOpt-sequentiel depuis plusieurs starts candidats.
	fmt.Println("\n--- dump 0xE6 (hex 20o + readOpt-seq depuis bit 8/9/10) ---")
	for i, e := range e6s {
		if i >= 12 {
			break
		}
		hx := ""
		for j := 0; j < 20 && j < len(e.pl); j++ {
			hx += fmt.Sprintf("%02X", e.pl[j])
		}
		fmt.Printf("[%2d] t=%7.1fs sz=%d hex=%s\n", i, secs(m, e.ts, t0), len(e.pl), hx)
		for _, st := range []int{8, 9, 10} {
			f := decodeSeq(e.pl, st)
			fmt.Printf("      start=%2d : %s\n", st, f.String())
		}
	}
}

// keFields : champs décodés du kill-event 0xE6.
type keFields struct {
	start  int
	f0, f1 int // readOpt (killer/victim, ordre à confirmer)
	f2     uint32
	f3     int // R1
	assist int // readOpt
	f5     uint32
	tail   []uint32 // R32 suivants
	endBit int
	ok     bool
}

func (f keFields) String() string {
	return fmt.Sprintf("f0=%d f1=%d f2=%08X f3=%d assist=%d f5=%08X tail0=%08X tail1=%08X",
		f0v(f.f0), f0v(f.f1), f.f2, f.f3, f0v(f.assist), f.f5, tget(f.tail, 0), tget(f.tail, 1))
}

func f0v(x int) int { return x }
func tget(t []uint32, i int) uint32 {
	if i < len(t) {
		return t[i]
	}
	return 0
}

// readOpt : gate 1 bit ; si 0 -> R5 (index). Renvoie (val, nextbp). -1 = absent (gate=1).
func readOpt(pl []byte, bp int) (int, int) {
	if bp < 0 || bp>>3 >= len(pl) {
		return -2, bp
	}
	if bitsAt(pl, bp, 1) == 0 {
		return int(bitsAt(pl, bp+1, 5)), bp + 6
	}
	return -1, bp + 1
}

// decodeSeq : décode la grammaire kill-event [readOpt f0][readOpt f1][R32 f2][R1 f3][readOpt assist]
// [R32 f5][tail R32...] depuis le bit start.
func decodeSeq(pl []byte, start int) keFields {
	var f keFields
	f.start = start
	f0, b1 := readOpt(pl, start)
	f1, b2 := readOpt(pl, b1)
	f.f0, f.f1 = f0, f1
	f.f2 = uint32(bitsAt(pl, b2, 32))
	b3 := b2 + 32
	f.f3 = int(bitsAt(pl, b3, 1))
	a, b4 := readOpt(pl, b3+1)
	f.assist = a
	f.f5 = uint32(bitsAt(pl, b4, 32))
	b5 := b4 + 32
	for i := 0; i < 6; i++ {
		pos := b5 + i*32
		if pos+32 > len(pl)*8 {
			break
		}
		f.tail = append(f.tail, uint32(bitsAt(pl, pos, 32)))
	}
	f.endBit = b5
	f.ok = f0 >= 0 && f1 >= 0
	return f
}

func topInt(mm map[int]int, n int) string {
	type kv struct{ k, v int }
	var a []kv
	for k, v := range mm {
		a = append(a, kv{k, v})
	}
	sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
	s := ""
	for i, e := range a {
		if i >= n {
			break
		}
		s += fmt.Sprintf("%d:%d ", e.k, e.v)
	}
	return s
}

// nearestKiller : attaquant du dernier 0xd2 firearm dans [ts-win, ts]. -1 si aucun.
func nearestKiller(dmgs []dmgEv, ts uint64, win uint64) (int, uint32) {
	best := -1
	var fam uint32
	var bestT uint64
	for _, d := range dmgs {
		if d.ts > ts || ts-d.ts > win {
			continue
		}
		if d.ts >= bestT {
			bestT, best, fam = d.ts, d.atk, d.family
		}
	}
	return best, fam
}

// runCalib : trouve la position du champ KILLER dans les 0xE6. Pour chaque 0xE6 dont un 0xd2 firearm
// existe juste avant (attaquant connu), histogramme des offsets de bit où R5==attaquant. Un pic =
// la position (fixe ou variable). Teste aussi le readOpt-seq depuis chaque start candidat.
func runCalib(m string, e6s []e6, dmgs []dmgEv, t0 uint64) {
	fmt.Printf("=== CALIB %s : localisation du champ KILLER via corrélation 0xd2 firearm ===\n", m)
	// (A) histogramme brut : offset où R5(5b)==killerAttendu.
	offHist := map[int]int{}
	nWith := 0
	for _, e := range e6s {
		k, _ := nearestKiller(dmgs, e.ts, 1_000_000)
		if k < 0 {
			continue
		}
		nWith++
		for o := 0; o+5 <= len(e.pl)*8; o++ {
			if int(bitsAt(e.pl, o, 5)) == k {
				offHist[o]++
			}
		}
	}
	fmt.Printf("0xE6 avec dégât firearm <1s avant : %d/%d\n", nWith, len(e6s))
	fmt.Printf("offsets où R5==killerAttendu (top) : %s\n", topInt(offHist, 16))

	// (B) readOpt-seq : pour chaque start candidat, taux d'accord f0==killer et f1==killer.
	fmt.Println("\n--- readOpt-seq : accord f0/f1 == killerAttendu, par start ---")
	for st := 6; st <= 20; st++ {
		n, a0, a1 := 0, 0, 0
		for _, e := range e6s {
			k, _ := nearestKiller(dmgs, e.ts, 1_000_000)
			if k < 0 {
				continue
			}
			n++
			f := decodeSeq(e.pl, st)
			if f.f0 == k {
				a0++
			}
			if f.f1 == k {
				a1++
			}
		}
		fmt.Printf("  start=%2d : f0==killer %d/%d (%.0f%%) | f1==killer %d/%d (%.0f%%)\n",
			st, a0, n, pct(a0, n), a1, n, pct(a1, n))
	}
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) * 100 / float64(b)
}

// runCause : test décisif. Classe chaque 0xE6 (firearm/grenade/mêlée/inconnu) via les flux same-clock,
// puis pour CHAQUE champ du 0xE6 construit la distribution par classe. Un champ qui sépare = code cause.
func runCause(m string, e6s []e6, dmgs []dmgEv, grens []grenEv, t0 uint64) {
	fmt.Printf("=== CAUSE %s : séparation des classes par champ 0xE6 ===\n", m)
	// classification same-clock, indépendante du décodage 0xE6 (par temps du kill).
	type row struct {
		cls string
		f   keFields
		t   float64
	}
	var rows []row
	nF, nG, nMel, nUnk := 0, 0, 0, 0
	// mêlée connue : (killer,victim,timeSec) — proximité temporelle.
	knownMel := []float64{147.6, 323.9}
	for _, e := range e6s {
		ts := e.ts
		tsec := secs(m, ts, t0)
		// firearm : 0xd2 firearm <0.6s avant.
		fk, _ := nearestKiller(dmgs, ts, 600_000)
		// grenade : lancer <4s avant, +1s après.
		grNear := false
		for _, g := range grens {
			dt := int64(ts) - int64(g.ts)
			if dt > -1_000_000 && dt < 4_000_000 {
				grNear = true
				break
			}
		}
		// mêlée : proche d'un des 2 kills validés.
		melNear := false
		for _, mt := range knownMel {
			if abs64(tsec-mt) < 2.0 {
				melNear = true
			}
		}
		cls := "unk"
		switch {
		case melNear:
			cls = "melee"
			nMel++
		case fk >= 0:
			cls = "firearm"
			nF++
		case grNear:
			cls = "grenade"
			nG++
		default:
			nUnk++
		}
		// décodage au meilleur start (déterminé par calib ; on prend le start optimal empiriquement).
		f := decodeSeq(e.pl, bestStart)
		rows = append(rows, row{cls, f, tsec})
	}
	fmt.Printf("classes : firearm=%d grenade=%d melee=%d inconnu=%d\n", nF, nG, nMel, nUnk)

	// distribution de chaque champ par classe.
	report := func(name string, get func(keFields) int) {
		byCls := map[string]map[int]int{}
		for _, r := range rows {
			if byCls[r.cls] == nil {
				byCls[r.cls] = map[int]int{}
			}
			byCls[r.cls][get(r.f)]++
		}
		fmt.Printf("\n--- champ %s ---\n", name)
		for _, c := range []string{"firearm", "grenade", "melee", "unk"} {
			if byCls[c] != nil {
				fmt.Printf("  %-8s : %s\n", c, topInt(byCls[c], 10))
			}
		}
	}
	report("f0", func(f keFields) int { return f.f0 })
	report("f1", func(f keFields) int { return f.f1 })
	report("f2(low16)", func(f keFields) int { return int(f.f2 & 0xffff) })
	report("f2(full)", func(f keFields) int { return int(f.f2) })
	report("f3(R1)", func(f keFields) int { return f.f3 })
	report("assist", func(f keFields) int { return f.assist })
	report("f5(low16)", func(f keFields) int { return int(f.f5 & 0xffff) })
	report("tail0", func(f keFields) int { return int(tget(f.tail, 0) & 0xffff) })
	report("tail1", func(f keFields) int { return int(tget(f.tail, 1) & 0xffff) })

	// dump par-kill mêlée + un échantillon grenade/firearm pour inspection manuelle.
	fmt.Println("\n--- échantillon par-kill ---")
	for _, r := range rows {
		if r.cls == "melee" || r.cls == "grenade" {
			fmt.Printf("  %-8s t=%6.1fs %s\n", r.cls, r.t, r.f.String())
		}
	}
}

const bestStart = 9 // ajusté après calib

func abs64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// runMeleeScan : cherche des events mêlée (marqueurs candidats) autour des 2 kills mêlée connus.
func runMeleeScan(m string, t0 uint64) {
	cache := root + "/" + m
	// candidats marqueurs mêlée : 11-bit 0x532/0x534/0x535, 24-bit variants.
	cand11 := []uint64{0x532, 0x533, 0x534, 0x535}
	counts := map[uint64]int{}
	near := map[uint64]int{}
	knownMel := []float64{147.6, 323.9}
	for ch := 0; ch <= 27; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		total := len(d) * 8
		for bp := 0; bp+40 < total; bp++ {
			v := bitsAt(d, bp, 11)
			for _, c := range cand11 {
				if v == c {
					counts[c]++
					if ts, ok := tsAtBit(d, bp); ok {
						tsec := secs(m, ts, t0)
						for _, mt := range knownMel {
							if abs64(tsec-mt) < 3.0 {
								near[c]++
							}
						}
					}
				}
			}
		}
	}
	fmt.Printf("=== MELEESCAN %s : marqueurs 11-bit candidats ===\n", m)
	for _, c := range cand11 {
		fmt.Printf("  0x%03X : %d occurrences (dont %d dans ±3s des 2 kills mêlée)\n", c, counts[c], near[c])
	}
}
