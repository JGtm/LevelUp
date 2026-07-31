// tmp_kfworldpos — parseur OFFLINE DURCI de la table keyframe type-2 d'un film Halo,
// + décodage de la POSITION i0 des bipeds au keyframe. THROWAWAY / harness de validation.
//
// DURCISSEMENT vs tmp_kftable (le code fait foi) :
//   - En-tête RÉEL du record keyframe type-2 = [id:32][field:26][ti:6] = 64 bits
//     (RE FUN_141f86704). tmp_kftable lisait ti en 32-bit ; ça "tombe juste" seulement
//     quand field26==0 (spawn). ICI : ti = readBits(q+58, 6), field26 = readBits(q+32, 26).
//   - gen==1 hardcodé RETIRÉ : gen = id>>30 accepté ∈ {1,2,3} (respawns mid-match),
//     rejet de id==0xFFFFFFFF (sentinelle) et gen==0 (handle null).
//   - startState = hdrBit+64 INCHANGÉ (largeur d'en-tête = 64 bits confirmée).
//
// WALKER : garde le FILTRE FORT anti-faux-positif (mot 32-bit après id < archMax, vrai
// pour tout record au spawn car field26==0) MAIS extrait ti/gen de façon durcie. Régression
// chunk_02 identique à tmp_kftable.
//
// POSITION i0 : au keyframe, le défaut biped = representation + le SEUL composant i0
// (default-mask={i0}, ground-truth CE). i0 default = R(1) rangeSel + consumeAbsoluteWithGate
// (precHigh R1 + idxSel R1 + idx R(IndexW) + 3×R(axisW)). On LOCALISE le début d'i0 par
// CONSENSUS des bits de gate (precHigh=0,idxSel=0,idx=0) sur les 8 bipeds, puis on déquantifie
// le vec3 via filmdec.WorldPositionRange (mêmes bornes/formule que dequantWorldAxis).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_kfworldpos [filmDir] [worldDump]
package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
	defDump = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/crack/world_dump_000d5950.txt`
	film2   = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/7344d24f`
	indexW  = 1 // DAT_144632be0 = 1 (largeur d'index i0) ; le walker durci vit dans filmdec.WalkKeyframeWorld
)

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

func bitAt(buf []byte, p int) uint64 {
	if idx := p >> 3; idx < len(buf) {
		return uint64(buf[idx]>>(7-uint(p&7))) & 1
	}
	return 0
}

// findAnchorHardened : 1re position bit p>=from où id occupe [p,p+32) ET ti occupe
// [p+58,p+64) (6 bits), en IGNORANT le field26 [p+32,p+58). Fenêtre glissante 64-bit.
// Retourne (p, field26) ; p<0 si absente.
func findAnchorHardened(buf []byte, from, id, ti, total int) (int, uint32) {
	if from+64 > total {
		return -1, 0
	}
	wantID := uint64(uint32(id))
	acc := readBits(buf, from, 64)
	for p := from; p+64 <= total; p++ {
		if (acc>>32) == wantID && (acc&0x3F) == uint64(uint32(ti)) {
			return p, uint32((acc >> 6) & 0x3FFFFFF)
		}
		acc = acc<<1 | bitAt(buf, p+64)
	}
	return -1, 0
}

func loadOracle(p string) map[int]int {
	f, err := os.Open(p)
	if err != nil {
		return nil
	}
	defer f.Close()
	m := map[int]int{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		for _, tok := range strings.Fields(line) {
			kv := strings.SplitN(tok, ":", 2)
			if len(kv) != 2 {
				continue
			}
			s, e1 := strconv.Atoi(kv[0])
			t, e2 := strconv.Atoi(kv[1])
			if e1 == nil && e2 == nil {
				m[s] = t
			}
		}
	}
	return m
}

func finite(v float32) bool { return !math.IsNaN(float64(v)) && !math.IsInf(float64(v), 0) }

// ---- Décodage POSITION i0 (absolu quantifié) ----

// zeroGate : à p, precHigh(0)+idxSel(0)+idx(IndexW bits ==0) ? = pattern d'un i0 absolu
// in-map (consumeAbsoluteWithGate). Retourne aussi le bit de début des axes.
func zeroGate(pay []byte, p int) (axStart int, ok bool) {
	if readBits(pay, p, 1) != 0 { // precHigh
		return 0, false
	}
	if readBits(pay, p+1, 1) != 0 { // idxSel (0 -> lit l'index)
		return 0, false
	}
	if readBits(pay, p+2, indexW) != 0 { // index (0 -> in-map, DAT_14462cbe0[0])
		return 0, false
	}
	return p + 2 + indexW, true
}

// decodeAbsAt déquantifie le vec3 i0 à partir du bit de gate p (precHigh) avec axisW.
// Retourne (pos, ok) ; ok=false si les gates ne sont pas (0,0,0).
func decodeAbsAt(pay []byte, p, axisW int) ([3]float32, bool) {
	axStart, ok := zeroGate(pay, p)
	if !ok {
		return [3]float32{}, false
	}
	rng := filmdec.WorldPositionRange
	var v [3]float32
	q := axStart
	for i := 0; i < 3; i++ {
		w := readBits(pay, q, axisW)
		q += axisW
		scale := float32(uint64(1) << uint(axisW))
		step := (rng[i].Max - rng[i].Min) / scale
		v[i] = float32(w)*step + rng[i].Min + step*0.5
	}
	return v, true
}

// offStat : à un offset donné, combien de bipeds ont gate(0,0,0) et combien de positions
// distinctes (axisW=aw) en résultent. Le VRAI i0 = gate 8/8 ET positions distinctes (les 8
// joueurs sont à des endroits différents) ; une région constante (padding/zéros) donne
// gate 8/8 mais distinct=1.
func offStat(pay []byte, stateBits []int, off, aw int) (gate, distinct int) {
	seen := map[string]bool{}
	for _, sb := range stateBits {
		if v, ok := decodeAbsAt(pay, sb+off, aw); ok {
			gate++
			seen[fmt.Sprintf("%.2f_%.2f_%.2f", v[0], v[1], v[2])] = true
		}
	}
	return gate, len(seen)
}

// findI0Offset : localise le début d'i0 par consensus. Meilleur = MAXIMISE (gate, distinct)
// avec distinct calculé à axisW=aw. Un offset gate=8/distinct=1 (région constante) perd
// contre gate=8/distinct=8 (vraies positions joueurs).
func findI0Offset(pay []byte, stateBits []int, lo, hi, aw int) (bestOff, bestGate, bestDist int) {
	bestOff, bestGate, bestDist = -1, -1, -1
	for off := lo; off <= hi; off++ {
		g, d := offStat(pay, stateBits, off, aw)
		if g > bestGate || (g == bestGate && d > bestDist) {
			bestOff, bestGate, bestDist = off, g, d
		}
	}
	return
}

func plausible(pts map[int][3]float32) (allFinite, distinct, inRange bool, nDistinct int) {
	allFinite, inRange = true, true
	seen := map[string]bool{}
	rng := filmdec.WorldPositionRange
	for _, v := range pts {
		for i := 0; i < 3; i++ {
			if !finite(v[i]) {
				allFinite = false
			}
			span := rng[i].Max - rng[i].Min
			if v[i] < rng[i].Min-0.02*span || v[i] > rng[i].Max+0.02*span {
				inRange = false
			}
		}
		seen[fmt.Sprintf("%.1f_%.1f_%.1f", v[0], v[1], v[2])] = true
	}
	nDistinct = len(seen)
	distinct = nDistinct == len(pts)
	return
}

// ---- diagnostics ----

func discover(pay []byte, oracle map[int]int) (bipedHdr map[int]int, located, matched, f26nz int) {
	total := len(pay) * 8
	slots := make([]int, 0, len(oracle))
	for s := range oracle {
		slots = append(slots, s)
	}
	sort.Ints(slots)
	bipedHdr = map[int]int{}
	pos := 0
	for _, slot := range slots {
		ti := oracle[slot]
		p, f26 := findAnchorHardened(pay, pos, 0x40000000|slot, ti, total)
		if p < 0 {
			continue
		}
		located++
		matched++
		if f26 != 0 {
			f26nz++
		}
		if ti == 35 {
			bipedHdr[slot] = p
		}
		pos = p + 64
	}
	return
}

func walkStats(pay []byte, oracle map[int]int) (records, uniq, match, mismatch, bipeds8, genGt1, f26nz int, bipedHdr map[int]int) {
	walked := filmdec.WalkKeyframeWorld(pay)
	parsed := map[int]int{}
	bipedHdr = map[int]int{}
	want := map[int]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}
	for _, r := range walked {
		parsed[r.Slot] = r.TI
		if r.Gen > 1 {
			genGt1++
		}
		if readBits(pay, r.Bit+32, 26) != 0 { // field26 recalculé depuis l'en-tête (bit de record)
			f26nz++
		}
		if r.TI == 35 {
			if want[r.Slot] {
				bipeds8++
			}
			bipedHdr[r.Slot] = r.Bit
		}
	}
	records, uniq = len(walked), len(parsed)
	if oracle != nil {
		for slot, ti := range parsed {
			if ot, ok := oracle[slot]; ok {
				if ot == ti {
					match++
				} else {
					mismatch++
				}
			}
		}
	}
	return
}

// findBipedAnyGen : cherche le slot biped (ti=35) avec N'IMPORTE QUEL gen (1..3) via l'ancre
// durcie (id||ti6, field26 ignoré). Prouve la gestion gen≥2 + field26!=0 sur un vrai record.
func findBipedAnyGen(pay []byte, slot int) (hdrBit, gen, field26 int, found bool) {
	total := len(pay) * 8
	best := -1
	for g := 1; g <= 3; g++ {
		id := (g << 30) | slot
		p, f26 := findAnchorHardened(pay, 0, id, 35, total)
		if p >= 0 && (best < 0 || p < best) {
			best, gen, field26, found = p, g, int(f26), true
		}
	}
	return best, gen, field26, found
}

func decodePositions(pay []byte, hdr map[int]int, off, axisW int) map[int][3]float32 {
	pts := map[int][3]float32{}
	for slot, h := range hdr {
		if v, ok := decodeAbsAt(pay, h+64+off, axisW); ok {
			pts[slot] = v
		}
	}
	return pts
}

func main() {
	dir, dump := defFilm, defDump
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if len(os.Args) > 2 {
		dump = os.Args[2]
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	i0Level := uint32(0)
	if a, ok := reg.Archetype(35); ok && len(a.Components) > 0 {
		i0Level = a.Level(0)
	}
	pay02 := framePayload(inflate(dir+"/chunk_02.bin"), 2)
	if pay02 == nil {
		fmt.Println("type-2 introuvable dans chunk_02")
		return
	}
	oracle := loadOracle(dump)
	fmt.Printf("registre=%d archétypes ; biped#35 i0 level=%d ; range=%v\n", len(reg.Archetypes), i0Level, filmdec.WorldPositionRange)
	fmt.Printf("chunk_02 : payload %d o (%d bits) ; oracle %d entités\n\n", len(pay02), len(pay02)*8, len(oracle))

	// ===== PART A : DISCOVERY (en-tête durci id||ti6) =====
	fmt.Println("=== PART A : DISCOVERY en-tête durci [id:32][field:26][ti:6] ===")
	bhdr, located, matched, f26nz := discover(pay02, oracle)
	bslots := sortedKeys(bhdr)
	fmt.Printf("localisés=%d/%d matches(slot+ti)=%d field26!=0=%d/%d\n", located, len(oracle), matched, f26nz, located)
	fmt.Printf("bipeds(ti=35)=%d slots=%v\n\n", len(bhdr), bslots)

	// ===== PART B : WALKER durci chunk_02 (régression) =====
	fmt.Println("=== PART B : WALKER durci chunk_02 (régression, sans oracle) ===")
	rec, uniq, mt, mis, b8, ggt1, wf26, _ := walkStats(pay02, oracle)
	fmt.Printf("records=%d uniq=%d MATCH=%d/%d MISMATCH=%d bipeds512-519=%d/8 gen>1=%d field26!=0=%d\n\n",
		rec, uniq, mt, len(oracle), mis, b8, ggt1, wf26)

	// ===== PART C : POSITION i0 des 8 bipeds =====
	fmt.Println("=== PART C : POSITION i0 des bipeds (consensus gate + déquant Cliffhanger) ===")
	stateBits := make([]int, 0, len(bslots))
	for _, s := range bslots {
		stateBits = append(stateBits, bhdr[s]+64)
	}
	awMain := int(6 + i0Level)
	// paysage : offsets à gate 8/8, avec distinct@6 et @12 (le VRAI i0 = gate 8 + distinct 8)
	fmt.Println("paysage (offsets à gate>=7 ; distinct@aw6 / @aw12) :")
	for o := 120; o <= 340; o++ {
		g, d6 := offStat(pay02, stateBits, o, awMain)
		if g >= 7 {
			_, d12 := offStat(pay02, stateBits, o, 12)
			fmt.Printf("  +%-4d gate=%d distinct@6=%d distinct@12=%d\n", o, g, d6, d12)
		}
	}
	off, cnt, dst := findI0Offset(pay02, stateBits, 120, 340, awMain)
	fmt.Printf("offset i0 choisi = +%d (gate=%d/8, distinct@%d=%d)\n", off, cnt, awMain, dst)
	for _, aw := range []int{awMain, 12, 15} {
		p := decodePositions(pay02, bhdr, off, aw)
		af, _, inr, nd := plausible(p)
		fmt.Printf("  axisW=%-2d : décodés=%d/8 finite=%v distinct=%d inRange=%v\n", aw, len(p), af, nd, inr)
	}
	pts := decodePositions(pay02, bhdr, off, awMain)
	af, dist, inr, nd := plausible(pts)
	// cross-check : 2e région gate=8/distinct=8 (>off+40) — si elle reproduit les MÊMES
	// positions, c'est l'i0 de la boucle de composants qui confirme l'i0 default-state.
	off2, g2, d2 := findI0Offset(pay02, stateBits, off+40, 340, awMain)
	if g2 == 8 && d2 >= 7 {
		pts2 := decodePositions(pay02, bhdr, off2, awMain)
		same := 0
		for s, v := range pts {
			if v2, ok := pts2[s]; ok && v == v2 {
				same++
			}
		}
		fmt.Printf("note : 2e champ vec3 quantifié @+%d (gate=%d distinct=%d, %d/8 == +%d) = autre composant du record ; +%d est le 1er vec3 après la representation = i0 position.\n", off2, g2, d2, same, off, off)
	}
	pts12 := decodePositions(pay02, bhdr, off, 12)
	fmt.Printf("\nPOSITIONS i0 (offset=+%d) : axisW=%d (registre) puis axisW=12 (fin) :\n", off, awMain)
	for _, s := range bslots {
		if v, ok := pts[s]; ok {
			w := pts12[s]
			fmt.Printf("  slot=%d hdrBit=%d  aw6=(%.1f, %.1f, %.1f)  aw12=(%.1f, %.1f, %.1f)\n", s, bhdr[s], v[0], v[1], v[2], w[0], w[1], w[2])
		} else {
			fmt.Printf("  slot=%d hdrBit=%d  (gate!=0 à cet offset -> i0 absent/precHigh)\n", s, bhdr[s])
		}
	}
	fmt.Printf("=> décodés=%d/8 finite=%v distinct=%v(%d) inRange=%v\n\n", len(pts), af, dist, nd, inr)

	// ===== PART D : GÉNÉRALITÉ mid-match + 2e film =====
	fmt.Println("=== PART D : GÉNÉRALITÉ mid-match (chunk_04/06) + 2e film 7344d24f ===")
	for _, cn := range []string{"chunk_04.bin", "chunk_06.bin"} {
		pay := framePayload(inflate(dir+"/"+cn), 2)
		if pay == nil {
			fmt.Printf("  %s : pas de frame type-2\n", cn)
			continue
		}
		r, u, _, _, _, gg, f26, _ := walkStats(pay, nil)
		fmt.Printf("  %s : payload=%do walker records=%d uniq=%d gen>1=%d field26!=0=%d\n", cn, len(pay), r, u, gg, f26)
		// bipeds 512-519 par ancre durcie (gen wildcard) -> prouve gen/field26 mid-match + positions
		mh := map[int]int{}
		nGen2, nF26 := 0, 0
		for slot := 512; slot <= 519; slot++ {
			if h, g, f, ok := findBipedAnyGen(pay, slot); ok {
				mh[slot] = h
				if g >= 2 {
					nGen2++
				}
				if f != 0 {
					nF26++
				}
			}
		}
		mstate := make([]int, 0, len(mh))
		mslots := sortedKeys(mh)
		for _, s := range mslots {
			mstate = append(mstate, mh[s]+64)
		}
		moff, mcnt, mdst := -1, 0, 0
		if len(mstate) > 0 {
			moff, mcnt, mdst = findI0Offset(pay, mstate, 120, 340, awMain)
		}
		fmt.Printf("    bipeds512-519 retrouvés=%d/8 (gen≥2=%d, field26!=0=%d) ; i0 offset=+%d (gate=%d/%d distinct=%d) :\n",
			len(mh), nGen2, nF26, moff, mcnt, len(mh), mdst)
		if moff >= 0 {
			mp := decodePositions(pay, mh, moff, awMain)
			for _, s := range mslots {
				if v, ok := mp[s]; ok {
					_, g, f, _ := findBipedAnyGen(pay, s)
					fmt.Printf("      slot=%d gen=%d field26=%d pos=(%.2f, %.2f, %.2f)\n", s, g, f, v[0], v[1], v[2])
				}
			}
		}
	}
	// 2e film 7344d24f (autre map -> range Cliffhanger inadaptée ; on valide le PARSE + bipeds)
	if _, e := os.Stat(film2 + "/chunk_02.bin"); e == nil {
		pay := framePayload(inflate(film2+"/chunk_02.bin"), 2)
		if pay != nil {
			r, u, _, _, _, gg, f26, bh := walkStats(pay, nil)
			nb := 0
			for range bh {
				nb++
			}
			fmt.Printf("  7344d24f/chunk_02 : payload=%do records=%d uniq=%d bipeds(ti=35)=%d gen>1=%d field26!=0=%d\n",
				len(pay), r, u, nb, gg, f26)
		} else {
			fmt.Printf("  7344d24f : chunk_02 sans frame type-2\n")
		}
	} else {
		fmt.Printf("  7344d24f : absent\n")
	}
}

func sortedKeys(m map[int]int) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}
