// tmp_kwcov — DIAGNOSTIC couverture kill-feed sous marqueurs FRÈRES (0xC0/C2/C3/CA/D3/E9) vs 0xD2.
//
// But (lane empirique) : pour chaque paquet FRÈRE fatal de 000d5950, mesurer pourquoi le kill est perdu :
// locateKillEventCursor OK/KO, field0/field1 dans [0,7], arme trouvée AVANT vs APRÈS le curseur, taille.
// Puis tester des variantes du détecteur (seuil sz, jeu de marqueurs, keFloor, arme-après) et mesurer
// couverture + accuracy vs chunk_27 (roster permutation, MÊME algo que tmp_kwval pairmatrix).
//
// Copie AUTONOME (le user fusionnera le fix dans tmp_kwval). Usage :
//
//	CGO_ENABLED=0 go run ./cmd/tmp_kwcov <film> <mode>
//	modes : diag (per-marqueur) | variants (grid seuil/marqueurs) | keallbreak (sz-breakdown)
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"

	"levelup/go-api/internal/analysis"
)

const root = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`
const sfx = uint32(0x42c9679f)

// -------- primitives copiées (tmp_kwval main.go / deserlen.go) --------

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

func keReadOpt(pl []byte, bp int) (int, int) {
	if bp < 0 || bp>>3 >= len(pl) {
		return -2, bp
	}
	if bitsAt(pl, bp, 1) == 0 {
		return int(bitsAt(pl, bp+1, 5)), bp + 6
	}
	return -1, bp + 1
}

func validKE(pl []byte, b int) bool {
	v, b2 := keReadOpt(pl, b)
	k, b3 := keReadOpt(pl, b2)
	if v < 0 || k < 0 || v >= 16 || k >= 16 || v == k {
		return false
	}
	f2 := bitsAt(pl, b3, 32)
	if f2 > 0xffff {
		return false
	}
	a, _ := keReadOpt(pl, b3+33)
	return a == -1 || (a >= 0 && a < 16)
}

func decodeKE(pl []byte, c int) (int, int) {
	v, b2 := keReadOpt(pl, c)
	k, _ := keReadOpt(pl, b2)
	return v, k
}

// keCandidates : positions field0 candidates dans [lo,hi[ (motif 10 bits 0x2A8 = R7=85 + 3 gates 000).
func keCandidates(pl []byte, lo, hi int) []int {
	var c []int
	if lo < 0 {
		lo = 0
	}
	for x := lo; x+17 <= hi && x+17 <= len(pl)*8; x++ {
		if bitsAt(pl, x, 10) == 0x2A8 && validKE(pl, x+10) {
			c = append(c, x+10)
		}
	}
	return c
}

// locateKillEventCursorF : locator paramétré par keFloor (baseline keFloor=140).
func locateKillEventCursorF(pl []byte, keFloor int) int {
	c := keCandidates(pl, keFloor, len(pl)*8)
	if len(c) == 0 {
		return -1
	}
	return c[0]
}

func weaponAnchor(pl []byte) int {
	for b := 0; b+64 <= len(pl)*8; b++ {
		if uint32(bitsAt(pl, b+32, 32)) == sfx {
			return b
		}
	}
	return -1
}

// weaponAnchorLast : dernière (famille+suffixe) AVANT limit.
func weaponAnchorLast(pl []byte, limit int) int {
	last := -1
	hi := len(pl) * 8
	if limit >= 0 && limit < hi {
		hi = limit
	}
	for b := 0; b+64 <= hi; b++ {
		if uint32(bitsAt(pl, b+32, 32)) == sfx {
			last = b
		}
	}
	return last
}

// weaponAnchorAfter : PREMIÈRE (famille+suffixe) APRÈS start (hypothèse c : arme après le kill-event).
func weaponAnchorAfter(pl []byte, start int) int {
	if start < 0 {
		start = 0
	}
	for b := start; b+64 <= len(pl)*8; b++ {
		if uint32(bitsAt(pl, b+32, 32)) == sfx {
			return b
		}
	}
	return -1
}

// -------- chunk_27 vérité-terrain (copié tmp_kwval) --------

func chunk27KV(m string) (kv []analysis.KVPair, nKills int) {
	cache := root + "/" + m
	var best []analysis.HighlightEvent
	for ch := 41; ch >= 18; ch-- {
		b := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		if len(b) == 0 {
			continue
		}
		evs, _ := analysis.ParseHighlightEvents(b, 0)
		nk := 0
		for _, e := range evs {
			if e.EventType == analysis.EventTypeKill {
				nk++
			}
		}
		if nk > nKills {
			nKills, best = nk, evs
		}
	}
	var raw []analysis.RawEvent
	for _, e := range best {
		if e.EventType == analysis.EventTypeKill || e.EventType == analysis.EventTypeDeath {
			raw = append(raw, analysis.RawEvent{EventType: e.EventType, XUID: fmt.Sprintf("%d", e.XUID), TimeMS: int64(e.TimeMS)})
		}
	}
	return analysis.ComputeKillerVictimPairs(raw, 5), nKills
}

func chunk27Pairs(m string) ([][2]uint64, int) {
	kv, nKills := chunk27KV(m)
	var pairs [][2]uint64
	for _, p := range kv {
		kx, _ := strconv.ParseUint(p.KillerXUID, 10, 64)
		vx, _ := strconv.ParseUint(p.VictimXUID, 10, 64)
		pairs = append(pairs, [2]uint64{kx, vx})
	}
	return pairs, nKills
}

// -------- décodage paramétré + roster overlap --------

type decKill struct {
	killer, victim int
	marker         byte
	ts             uint64
}

type cfg struct {
	minSz       int
	markers     map[byte]bool
	keFloor     int
	antiFP      bool
	weaponAfter bool // n'affecte pas les paires ; ici pour compat
}

func dmgMkSet() map[byte]bool {
	return map[byte]bool{0xD2: true, 0xC0: true, 0xC2: true, 0xC3: true, 0xCA: true, 0xD3: true, 0xE9: true}
}

// decode : réplique decodePipeline (paires seulement) paramétré par cfg. Renvoie dedup + diag par marqueur.
func decode(m string, c cfg) ([]decKill, map[byte][3]int) {
	cache := root + "/" + m
	var feed []decKill
	// diag[mk] = [nFatal, nLoc, nAntiFPreject]
	diag := map[byte][3]int{}
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
			if typ != 0 || len(pl) == 0 || !c.markers[pl[0]] || sz < c.minSz {
				continue
			}
			e := diag[pl[0]]
			e[0]++
			cur := locateKillEventCursorF(pl, c.keFloor)
			if cur < 0 {
				diag[pl[0]] = e
				continue
			}
			if c.antiFP && weaponAnchor(pl) < 0 && cur >= 1024 {
				e[2]++
				diag[pl[0]] = e
				continue
			}
			vic, kil := decodeKE(pl, cur)
			e[1]++
			diag[pl[0]] = e
			feed = append(feed, decKill{kil, vic, pl[0], ts})
		}
	}
	// roster : indices vus >=2x
	idxFreq := map[int]int{}
	for _, k := range feed {
		idxFreq[k.killer]++
		idxFreq[k.victim]++
	}
	roster := map[int]bool{}
	for i, n := range idxFreq {
		if n >= 2 {
			roster[i] = true
		}
	}
	var kept []decKill
	for _, k := range feed {
		if roster[k.killer] && roster[k.victim] {
			kept = append(kept, k)
		}
	}
	// dedup (killer,victim,~3ms)
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].ts < kept[j].ts })
	var dedup []decKill
	for _, k := range kept {
		dup := false
		for j := len(dedup) - 1; j >= 0 && k.ts-dedup[j].ts < 3000000; j-- {
			if dedup[j].killer == k.killer && dedup[j].victim == k.victim {
				dup = true
				break
			}
		}
		if !dup {
			dedup = append(dedup, k)
		}
	}
	return dedup, diag
}

// --- roster overlap (copié tmp_kwval) ---

func overlapForMap(dedup []decKill, assign map[int]uint64, c27cnt map[[2]uint64]int) int {
	decCnt := map[[2]uint64]int{}
	for _, k := range dedup {
		kx, ok1 := assign[k.killer]
		vx, ok2 := assign[k.victim]
		if !ok1 || !ok2 {
			continue
		}
		decCnt[[2]uint64{kx, vx}]++
	}
	ov := 0
	for key, dc := range decCnt {
		if cc := c27cnt[key]; cc < dc {
			ov += cc
		} else {
			ov += dc
		}
	}
	return ov
}

func bestInjectionOverlap(dedup []decKill, di []int, cx []uint64, c27cnt map[[2]uint64]int) (int, map[int]uint64) {
	best := -1
	var bestMap map[int]uint64
	assign := make(map[int]uint64, len(di))
	used := make([]bool, len(cx))
	var rec func(pos int)
	rec = func(pos int) {
		if pos == len(di) {
			if ov := overlapForMap(dedup, assign, c27cnt); ov > best {
				best = ov
				bestMap = map[int]uint64{}
				for k, v := range assign {
					bestMap[k] = v
				}
			}
			return
		}
		for j := range cx {
			if used[j] {
				continue
			}
			used[j] = true
			assign[di[pos]] = cx[j]
			rec(pos + 1)
			delete(assign, di[pos])
			used[j] = false
		}
	}
	if len(cx) >= len(di) {
		rec(0)
	}
	if best < 0 {
		best = 0
	}
	return best, bestMap
}

func solveOverlap(dedup []decKill, c27 [][2]uint64) int {
	idxSet := map[int]bool{}
	for _, k := range dedup {
		idxSet[k.killer] = true
		idxSet[k.victim] = true
	}
	var di []int
	for i := range idxSet {
		di = append(di, i)
	}
	sort.Ints(di)
	xf := map[uint64]int{}
	for _, p := range c27 {
		xf[p[0]]++
		xf[p[1]]++
	}
	var cx []uint64
	for x := range xf {
		cx = append(cx, x)
	}
	sort.Slice(cx, func(i, j int) bool {
		if xf[cx[i]] != xf[cx[j]] {
			return xf[cx[i]] > xf[cx[j]]
		}
		return cx[i] < cx[j]
	})
	if len(cx) > 8 {
		cx = cx[:8]
	}
	c27cnt := map[[2]uint64]int{}
	for _, p := range c27 {
		c27cnt[p]++
	}
	if len(di) > 8 {
		// garde les 8 indices les plus fréquents pour rester factoriel-borné
		freq := map[int]int{}
		for _, k := range dedup {
			freq[k.killer]++
			freq[k.victim]++
		}
		sort.Slice(di, func(i, j int) bool { return freq[di[i]] > freq[di[j]] })
		di = di[:8]
		sort.Ints(di)
	}
	best, _ := bestInjectionOverlap(dedup, di, cx, c27cnt)
	return best
}

func report(m, label string, dedup []decKill, nKills int) {
	c27, _ := chunk27Pairs(m)
	ov := solveOverlap(dedup, c27)
	acc := 0.0
	if len(dedup) > 0 {
		acc = float64(ov) * 100 / float64(len(dedup))
	}
	cov := float64(ov) * 100 / float64(max(nKills, 1))
	fmt.Printf("  %-42s décodées=%3d overlap=%3d | ACC %5.1f%% (ov/déc) | COUV %5.1f%% (ov/%d)\n",
		label, len(dedup), ov, acc, cov, nKills)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// -------- modes --------

func main() {
	m := "000d5950"
	mode := "diag"
	if len(os.Args) > 1 {
		m = os.Args[1]
	}
	if len(os.Args) > 2 {
		mode = os.Args[2]
	}
	switch mode {
	case "diag":
		runDiag(m)
	case "variants":
		runVariants(m)
	case "keallbreak":
		runKeAllBreak(m)
	case "oracle":
		runOracle(m)
	case "floorsweep":
		runFloorSweep(m)
	default:
		fmt.Println("modes : diag | variants | keallbreak | oracle | floorsweep")
	}
}

const ceDir = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce`

func indexBytes(hay, needle []byte) int {
	for i := 0; i+len(needle) <= len(hay); i++ {
		match := true
		for j := range needle {
			if hay[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

type oraclePkt struct {
	pl     []byte
	cursor int
	marker byte
}

func loadOracle(m string) []oraclePkt {
	kd, err := os.ReadFile(ceDir + "/" + m + "_align_killdeser.bin")
	if err != nil {
		return nil
	}
	cache := root + "/" + m
	var chunks [][]byte
	for ch := 0; ch <= 41; ch++ {
		chunks = append(chunks, inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch)))
	}
	var out []oraclePkt
	for o := 0; o+128 <= len(kd); o += 128 {
		cursor := int(binary.LittleEndian.Uint32(kd[o+8:]))
		win := kd[o+16 : o+16+16]
		hit, hitCh := -1, -1
		for ci, d := range chunks {
			if x := indexBytes(d, win); x >= 0 {
				hit, hitCh = x, ci
				break
			}
		}
		if hit < 0 {
			continue
		}
		d := chunks[hitCh]
		off := 0
		for off+16 <= len(d) {
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			if hit >= off && hit < off+16+sz {
				pl := make([]byte, sz)
				copy(pl, d[off+16:off+16+sz])
				out = append(out, oraclePkt{pl, cursor, pl[0]})
				break
			}
			off += 16 + sz
		}
	}
	return out
}

// runOracle : sur les paquets fataux à CURSEUR VRAI connu (CE), mesure locator EXACT par keFloor + marqueur.
// Vérifie qu'abaisser keFloor ne DÉGRADE pas la localisation sur les vrais curseurs (anti-régression).
func runOracle(m string) {
	pk := loadOracle(m)
	if len(pk) == 0 {
		fmt.Printf("=== ORACLE %s : pas de _align_killdeser.bin ===\n", m)
		return
	}
	fmt.Printf("=== ORACLE %s : %d paquets à curseur vrai ===\n", m, len(pk))
	for _, f := range []int{80, 100, 110, 120, 125, 130, 140, 155} {
		exact, tot := 0, 0
		perMk := map[byte][2]int{}
		for _, p := range pk {
			tot++
			loc := locateKillEventCursorF(p.pl, f)
			e := perMk[p.marker]
			e[1]++
			if loc == p.cursor {
				exact++
				e[0]++
			}
			perMk[p.marker] = e
		}
		s := fmt.Sprintf("keFloor=%3d : EXACT %d/%d | ", f, exact, tot)
		var mks []byte
		for mk := range perMk {
			mks = append(mks, mk)
		}
		sort.Slice(mks, func(i, j int) bool { return mks[i] < mks[j] })
		for _, mk := range mks {
			s += fmt.Sprintf("0x%02X:%d/%d ", mk, perMk[mk][0], perMk[mk][1])
		}
		fmt.Println(s)
	}
}

// runFloorSweep : couverture+accuracy vs chunk_27 en balayant keFloor finement (sz>=700, dmgMk, antiFP).
func runFloorSweep(m string) {
	_, nKills := chunk27Pairs(m)
	fmt.Printf("=== FLOORSWEEP %s (chunk_27 = %d kills) ===\n", m, nKills)
	base := cfg{minSz: 700, markers: dmgMkSet(), antiFP: true}
	for _, f := range []int{60, 70, 80, 90, 100, 105, 110, 115, 120, 125, 130, 135, 140} {
		c := base
		c.keFloor = f
		dedup, _ := decode(m, c)
		report(m, fmt.Sprintf("keFloor=%d", f), dedup, nKills)
	}
}

// runDiag : per-marqueur fatal (dmgMk+sz>=700) — localisé/raté, field0/1 dans [0,7], arme avant/après.
func runDiag(m string) {
	cache := root + "/" + m
	dmg := dmgMkSet()
	type st struct {
		nFatal, nLoc, nField07, nWbefore, nWany, nWafterOnly, nNoW int
		curs                                                       []int
	}
	stats := map[byte]*st{}
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
			if typ != 0 || len(pl) == 0 || !dmg[pl[0]] || sz < 700 {
				continue
			}
			s := stats[pl[0]]
			if s == nil {
				s = &st{}
				stats[pl[0]] = s
			}
			s.nFatal++
			cur := locateKillEventCursorF(pl, 140)
			if cur < 0 {
				continue
			}
			s.nLoc++
			s.curs = append(s.curs, cur)
			vic, kil := decodeKE(pl, cur)
			if vic >= 0 && vic < 8 && kil >= 0 && kil < 8 {
				s.nField07++
			}
			wb := weaponAnchorLast(pl, cur)
			wany := weaponAnchor(pl)
			if wb >= 0 {
				s.nWbefore++
			}
			if wany >= 0 {
				s.nWany++
			}
			if wb < 0 && wany >= 0 && wany > cur {
				s.nWafterOnly++
			}
			if wany < 0 {
				s.nNoW++
			}
		}
	}
	fmt.Printf("=== DIAG %s : paquets fataux (dmgMk + sz>=700) par marqueur ===\n", m)
	fmt.Printf("  %-6s %6s %6s %8s %9s %7s %10s %6s  %s\n", "mk", "fatal", "loc", "f0/1<8", "wBefore", "wAny", "wAfterOnly", "noW", "curMin/med/max")
	for _, mk := range []byte{0xD2, 0xC0, 0xC2, 0xC3, 0xCA, 0xD3, 0xE9} {
		s := stats[mk]
		if s == nil {
			continue
		}
		cmin, cmed, cmax := 0, 0, 0
		if len(s.curs) > 0 {
			sort.Ints(s.curs)
			cmin, cmed, cmax = s.curs[0], s.curs[len(s.curs)/2], s.curs[len(s.curs)-1]
		}
		fmt.Printf("  0x%02X  %6d %6d %8d %9d %7d %10d %6d  %d/%d/%d\n",
			mk, s.nFatal, s.nLoc, s.nField07, s.nWbefore, s.nWany, s.nWafterOnly, s.nNoW, cmin, cmed, cmax)
	}
}

// runKeAllBreak : tous les paquets type-0 avec un kill-event valide, ventilés par (marqueur, sz>=700?).
// Montre combien de kill-events détectables tombent sous sz<700 (perdus par le seuil du détecteur).
func runKeAllBreak(m string) {
	cache := root + "/" + m
	dmg := dmgMkSet()
	type c2 struct{ big, small int }
	byMk := map[byte]*c2{}
	bigDmg, smallDmg, bigOther, smallOther := 0, 0, 0, 0
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
			if typ != 0 || len(pl) == 0 {
				continue
			}
			if locateKillEventCursorF(pl, 140) < 0 {
				continue
			}
			e := byMk[pl[0]]
			if e == nil {
				e = &c2{}
				byMk[pl[0]] = e
			}
			big := sz >= 700
			if big {
				e.big++
			} else {
				e.small++
			}
			if dmg[pl[0]] {
				if big {
					bigDmg++
				} else {
					smallDmg++
				}
			} else {
				if big {
					bigOther++
				} else {
					smallOther++
				}
			}
		}
	}
	fmt.Printf("=== KEALLBREAK %s : kill-events valides par (marqueur, sz>=700?) ===\n", m)
	var mks []byte
	for mk := range byMk {
		mks = append(mks, mk)
	}
	sort.Slice(mks, func(i, j int) bool { return byMk[mks[i]].big+byMk[mks[i]].small > byMk[mks[j]].big+byMk[mks[j]].small })
	for _, mk := range mks {
		e := byMk[mk]
		tag := "OTHER"
		if dmg[mk] {
			tag = "dmgMk"
		}
		fmt.Printf("  0x%02X [%s] : sz>=700 %2d | sz<700 %2d\n", mk, tag, e.big, e.small)
	}
	fmt.Printf("dmgMk: big=%d small=%d | non-dmgMk: big=%d small=%d\n", bigDmg, smallDmg, bigOther, smallOther)
}

// runVariants : grille de détecteurs — mesure couverture + accuracy vs chunk_27.
func runVariants(m string) {
	_, nKills := chunk27Pairs(m)
	fmt.Printf("=== VARIANTS %s (chunk_27 = %d kills) ===\n", m, nKills)

	dmg := dmgMkSet()
	base := cfg{minSz: 700, markers: dmg, keFloor: 140, antiFP: true}

	run := func(label string, c cfg) {
		dedup, _ := decode(m, c)
		report(m, label, dedup, nKills)
	}

	fmt.Println("-- baseline --")
	run("V0 baseline (sz>=700, dmgMk, floor140, antiFP)", base)

	fmt.Println("-- (d) abaisser le seuil de taille --")
	for _, s := range []int{500, 400, 300, 200, 100, 1} {
		c := base
		c.minSz = s
		run(fmt.Sprintf("minSz=%d", s), c)
	}

	fmt.Println("-- (d) seuil bas + sans anti-FP --")
	for _, s := range []int{300, 200, 100} {
		c := base
		c.minSz = s
		c.antiFP = false
		run(fmt.Sprintf("minSz=%d antiFP=off", s), c)
	}

	fmt.Println("-- (a) varier keFloor (sz>=700) --")
	for _, f := range []int{80, 100, 120, 155, 160} {
		c := base
		c.keFloor = f
		run(fmt.Sprintf("keFloor=%d", f), c)
	}

	fmt.Println("-- (a)+(d) keFloor bas + seuil bas --")
	for _, f := range []int{100, 120} {
		for _, s := range []int{300, 200} {
			c := base
			c.keFloor = f
			c.minSz = s
			run(fmt.Sprintf("keFloor=%d minSz=%d", f, s), c)
		}
	}

	fmt.Println("-- marqueurs élargis (ajoute 0xA0/0xDD/0x89/0xC7/0xE5) --")
	wide := map[byte]bool{}
	for k := range dmg {
		wide[k] = true
	}
	for _, k := range []byte{0xA0, 0xDD, 0x89, 0xC7, 0xE5} {
		wide[k] = true
	}
	{
		c := base
		c.markers = wide
		run("wideMarkers sz>=700", c)
		c.minSz = 200
		run("wideMarkers minSz=200", c)
	}
}
