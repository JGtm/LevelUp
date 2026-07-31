// tmp_absscore — SWEEP de calibration du CHEMIN ABSOLU i0 contre l'oracle CE.
//
// Le juge = la boîte oracle (positions i0 réelles du jeu, ce_pos_oracle.csv) :
//
//	X[-6.3..35.7] Y[-25.1..27.5] Z[-4.2..7.1], quantum de grille 0.0138.
//
// Deux mesures par combo {AxisW absolu × forme de déquant} :
//
//	PART A (keyframe, direct) : décode la position i0 ABSOLUE des 8 bipeds au keyframe
//	  (chunk_02, offset i0 localisé par consensus de gate) et teste : % dans la boîte,
//	  spans par axe (vs oracle 42/53/11), sigma_z.
//	PART B (replay type-0) : seede le World avec les positions keyframe, rejoue les
//	  paquets type-0 (ScanFrameTargets + accumulateur), et mesure : seeds absolus dans la
//	  boîte, sigma_z par slot, spans, et NB DE RECORDS DÉCODÉS CLEAN (DecodeFrameRecords
//	  séquentiel, DesyncAt==-1) = le mur i63 recule-t-il ?
//
// Sorties scratchpad : abs_sweep.txt (tableau) + best_traj.txt (trajectoires meilleure config).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_absscore [chunkLo chunkHi]
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
	cache   = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
	oracleP = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/ce_pos_oracle.csv`
	scratch = `c:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration/d6f109c3-2822-4e19-8fd4-cf65c50fec3d/scratchpad`
	idLow   = 11
	indexW  = 1
)

var bipedSlots = []uint32{512, 513, 514, 515, 516, 517, 518, 519}

// boîte oracle (dérivée de ce_pos_oracle.csv, avec petite marge)
var box = [3][2]float64{{-6.325, 35.701}, {-25.144, 27.500}, {-4.198, 7.077}}
var oracleSpan = [3]float64{42.03, 52.64, 11.28}

func inBox(v [3]float64) bool {
	for i := 0; i < 3; i++ {
		m := 0.15 * (box[i][1] - box[i][0]) // 15% de marge
		if v[i] < box[i][0]-m || v[i] > box[i][1]+m {
			return false
		}
	}
	return true
}

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

func listType0(d []byte) [][]byte {
	var out [][]byte
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, d[off+16:off+16+sz])
		}
		off += 16 + sz
	}
	return out
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

// ---- déquant (miroir de filmdec.dequantWorldAxis, les 2 formes) ----

func dequantCentered(q uint64, bits int, quantum float64) float64 {
	half := float64(uint64(1) << uint(bits-1))
	return (float64(q) - half) * quantum
}
func dequantRange(q uint64, bits, axis int) float64 {
	rng := filmdec.QuantRangeCliffhanger
	scale := float64(uint64(1) << uint(bits))
	step := float64(rng[axis].Max-rng[axis].Min) / scale
	return float64(q)*step + float64(rng[axis].Min) + step*0.5
}

// ---- oracle ----

func loadOracle(p string) map[int][][3]float64 {
	f, err := os.Open(p)
	if err != nil {
		return nil
	}
	defer f.Close()
	m := map[int][][3]float64{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		t := strings.TrimSpace(sc.Text())
		if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "eid") {
			continue
		}
		fs := strings.Split(t, ",")
		if len(fs) < 6 {
			continue
		}
		slot, e1 := strconv.Atoi(strings.TrimSpace(fs[1]))
		x, e2 := strconv.ParseFloat(strings.TrimSpace(fs[3]), 64)
		y, e3 := strconv.ParseFloat(strings.TrimSpace(fs[4]), 64)
		z, e4 := strconv.ParseFloat(strings.TrimSpace(fs[5]), 64)
		if e1 == nil && e2 == nil && e3 == nil && e4 == nil {
			m[slot] = append(m[slot], [3]float64{x, y, z})
		}
	}
	return m
}

// ---- keyframe : localisation de l'offset i0 (consensus de gate) ----

func zeroGate(pay []byte, p int) (axStart int, ok bool) {
	if readBits(pay, p, 1) != 0 { // precHigh
		return 0, false
	}
	if readBits(pay, p+1, 1) != 0 { // idxSel (0 -> lit l'index)
		return 0, false
	}
	if readBits(pay, p+2, indexW) != 0 { // index (0 -> in-map)
		return 0, false
	}
	return p + 2 + indexW, true
}

// gateConsensus : nb de bipeds avec gate(0,0,0) à l'offset off (relatif à startState).
func gateConsensus(pay []byte, stateBits []int, off int) int {
	g := 0
	for _, sb := range stateBits {
		if _, ok := zeroGate(pay, sb+off); ok {
			g++
		}
	}
	return g
}

// findI0Offset : offset dans [120,340] maximisant le gate consensus puis distinct@aw6 (Cliffhanger),
// reproduit tmp_kfworldpos. Indépendant du mode/width testés (le gate est aux 3 premiers bits).
func findI0Offset(pay []byte, stateBits []int) int {
	bestOff, bestGate, bestDist := -1, -1, -1
	for off := 120; off <= 340; off++ {
		g := gateConsensus(pay, stateBits, off)
		seen := map[string]bool{}
		for _, sb := range stateBits {
			if ax, ok := zeroGate(pay, sb+off); ok {
				var v [3]float64
				for i := 0; i < 3; i++ {
					v[i] = dequantRange(readBits(pay, ax+i*6, 6), 6, i)
				}
				seen[fmt.Sprintf("%.1f_%.1f_%.1f", v[0], v[1], v[2])] = true
			}
		}
		d := len(seen)
		if g > bestGate || (g == bestGate && d > bestDist) {
			bestOff, bestGate, bestDist = off, g, d
		}
	}
	return bestOff
}

// decodeKeyframe décode la position i0 des 8 bipeds au keyframe, avec (axisW, mode).
func decodeKeyframe(pay []byte, hdr map[int]int, off, axisW int, centered bool, quantum float64) map[int][3]float64 {
	out := map[int][3]float64{}
	for slot, h := range hdr {
		ax, ok := zeroGate(pay, h+64+off)
		if !ok {
			continue
		}
		var v [3]float64
		for i := 0; i < 3; i++ {
			q := readBits(pay, ax+i*axisW, axisW)
			if centered {
				v[i] = dequantCentered(q, axisW, quantum)
			} else {
				v[i] = dequantRange(q, axisW, i)
			}
		}
		out[slot] = v
	}
	return out
}

// ---- stats ----

func spans(pts map[int][3]float64) [3]float64 {
	var mn, mx [3]float64
	first := true
	for _, v := range pts {
		for i := 0; i < 3; i++ {
			if first || v[i] < mn[i] {
				mn[i] = v[i]
			}
			if first || v[i] > mx[i] {
				mx[i] = v[i]
			}
		}
		first = false
	}
	return [3]float64{mx[0] - mn[0], mx[1] - mn[1], mx[2] - mn[2]}
}

func nInBox(pts map[int][3]float64) (int, int) {
	in := 0
	for _, v := range pts {
		if inBox(v) {
			in++
		}
	}
	return in, len(pts)
}

func zSigma(seqs map[int][][3]float64) map[int]float64 {
	out := map[int]float64{}
	for s, seq := range seqs {
		if len(seq) < 2 {
			continue
		}
		var m, m2 float64
		for _, p := range seq {
			m += p[2]
		}
		m /= float64(len(seq))
		for _, p := range seq {
			m2 += (p[2] - m) * (p[2] - m)
		}
		out[s] = math.Sqrt(m2 / float64(len(seq)))
	}
	return out
}

// ---- keyframe bindings ----

type binding struct{ full, ti uint32 }

func offlineBindings(pay []byte) ([]binding, map[int]int) {
	recs := filmdec.WalkKeyframeWorld(pay)
	out := make([]binding, 0, len(recs))
	hdr := map[int]int{}
	for _, r := range recs {
		out = append(out, binding{full: uint32((r.Gen << 30) | r.Slot), ti: uint32(r.TI)})
		if r.TI == 35 {
			hdr[r.Slot] = r.Bit
		}
	}
	return out, hdr
}

// ---- replay type-0 (seeds + accumulation + clean records) ----

type replayResult struct {
	seedPts     map[int][3]float64   // dernières positions absolues seedées par slot (seeds)
	traj        map[int][][3]float64 // trajectoire accumulée par slot
	seedKinds   map[string]int       // répartition abs/absfb parmi les seeds bipeds
	cleanRec    int                  // records bipeds décodés CLEAN (DesyncAt==-1) en séquentiel
	desyncRec   int                  // records bipeds désyncés
	desyncAt    map[string]int       // où ça désync (composant)
	endReach    int                  // records dont endBit atteint la fin du paquet frontier
	replayAbsN  int                  // seeds absolus ÉMIS par le replay (predFlag==1/fallback), hors keyframe
	replayAbsIn int                  // ...dont dans la boîte oracle
}

func replay(reg *filmdec.Registry, offBs []binding, hdr map[int]int, kfPts map[int][3]float64, chunkLo, chunkHi int) replayResult {
	bipedSet := map[uint32]bool{}
	for _, s := range bipedSlots {
		bipedSet[s] = true
	}
	res := replayResult{seedPts: map[int][3]float64{}, traj: map[int][][3]float64{}, seedKinds: map[string]int{}, desyncAt: map[string]int{}}

	// --- trajectoire accumulée via ScanFrameTargets (World persistant, seedé keyframe) ---
	w := filmdec.NewWorld(reg)
	for _, b := range offBs {
		w.BindFull(b.full, b.ti)
	}
	// seed keyframe : les positions absolues décodées deviennent le point d'ancrage.
	for slot, v := range kfPts {
		w.SetPos(uint32(slot), [3]float32{float32(v[0]), float32(v[1]), float32(v[2])})
		res.seedPts[slot] = v // le seed keyframe compte comme seed absolu
	}
	filmdec.SetPositionAccumulator(w)
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) {
		if !bipedSet[s.Slot] {
			return
		}
		v := [3]float64{float64(s.Vec[0]), float64(s.Vec[1]), float64(s.Vec[2])}
		res.traj[int(s.Slot)] = append(res.traj[int(s.Slot)], v)
		if s.Kind == filmdec.PosKindAbsolute || s.Kind == filmdec.PosKindAbsFallback {
			res.seedKinds[s.Kind.String()]++
			res.replayAbsN++ // seed absolu ÉMIS PAR LE REPLAY (chemin predFlag==1/fallback), hors keyframe
			if inBox(v) {
				res.replayAbsIn++
			}
			res.seedPts[int(s.Slot)] = v // un seed absolu écrase le seed courant
		}
	})
	filmdec.SetRecordStateParam(2)
	cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: idLow}
	for c := chunkLo; c <= chunkHi; c++ {
		data := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, c))
		if data == nil {
			continue
		}
		for _, pay := range listType0(data) {
			filmdec.ScanFrameTargets(pay, w, cfg, bipedSet, filmdec.HarvestNextBound)
		}
	}
	filmdec.SetPositionAccumulator(nil)
	filmdec.SetPositionCaptureHook(nil)

	// --- clean records : décodage SÉQUENTIEL (World persistant frais) pour mesurer le mur i63 ---
	w2 := filmdec.NewWorld(reg)
	for _, b := range offBs {
		w2.BindFull(b.full, b.ti)
	}
	for c := chunkLo; c <= chunkHi; c++ {
		data := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, c))
		if data == nil {
			continue
		}
		for _, pay := range listType0(data) {
			br := filmdec.NewBitReader(pay)
			recs, _ := filmdec.DecodeFrameRecords(br, w2, cfg)
			frontier := len(pay) * 8
			for _, r := range recs {
				if !bipedSet[r.Slot] {
					continue
				}
				if r.DesyncAt < 0 {
					res.cleanRec++
					if r.Trace.EndBit >= frontier-64 {
						res.endReach++
					}
				} else {
					res.desyncRec++
					if arch, ok := reg.Archetype(int(r.TypeIndex)); ok && r.DesyncAt < len(arch.Components) {
						res.desyncAt[fmt.Sprintf("i%d %s", r.DesyncAt, arch.Components[r.DesyncAt])]++
					}
				}
			}
		}
	}
	return res
}

type comboResult struct {
	axisW    int
	centered bool
	// keyframe
	kfInBox, kfN int
	kfSpan       [3]float64
	// replay
	seedInBox, seedN int
	trajSpan         [3]float64
	zbad             int
	zmax             float64
	cleanRec         int
	desyncRec        int
	seedKinds        map[string]int
	desyncTop        string
	replayAbsN       int
	replayAbsIn      int
}

func main() {
	chunkLo, chunkHi := 3, 10
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[1], "%d", &chunkLo)
		fmt.Sscanf(os.Args[2], "%d", &chunkHi)
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	pay02 := framePayload(inflate(cache+"/chunk_02.bin"), 2)
	offBs, hdr := offlineBindings(pay02)

	// quantum dérivé de l'oracle : MODE des |diffs| consécutifs par axe (le bucket le plus fréquent
	// = 1 cran de grille). L'histogramme oracle donne 0.013/0.027/0.041 = multiples de 0.0138.
	oracleSeqs := loadOracle(oracleP)
	quantum := modalQuantum(oracleSeqs)
	filmdec.SetDeltaQuantum(float32(quantum))

	// DUMP diagnostic : q bruts au keyframe (offset i0) à axisW=6/12/14, pour voir où ils tombent
	// dans la plage (près de 0 = bord négatif en centré ; près de 2^(w-1) = centre).
	dumpKeyframeRawQ(pay02, hdr, findI0Offset(pay02, buildStateBits(pay02, hdr)))

	stateBits := make([]int, 0, len(hdr))
	hslots := sortedKeys(hdr)
	for _, s := range hslots {
		stateBits = append(stateBits, hdr[s]+64)
	}
	i0off := findI0Offset(pay02, stateBits)

	var b strings.Builder
	W := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	W("========== SWEEP CHEMIN ABSOLU i0 (juge = boîte oracle) ==========\n")
	W("quantum oracle (grille) = %.5f ; offset i0 keyframe = +%d ; bipeds=%d\n", quantum, i0off, len(hdr))
	W("boîte oracle : X[%.1f..%.1f] Y[%.1f..%.1f] Z[%.1f..%.1f]  spans=%.1f/%.1f/%.1f\n\n",
		box[0][0], box[0][1], box[1][0], box[1][1], box[2][0], box[2][1], oracleSpan[0], oracleSpan[1], oracleSpan[2])

	widths := []int{6, 10, 12, 14}
	var results []comboResult
	for _, centered := range []bool{true, false} {
		for _, aw := range widths {
			filmdec.SetAbsoluteAxisW(uint(aw))
			if centered {
				filmdec.SetAbsDequantMode(filmdec.AbsDequantCenteredQuantum)
			} else {
				filmdec.SetAbsDequantMode(filmdec.AbsDequantRange)
			}
			kf := decodeKeyframe(pay02, hdr, i0off, aw, centered, quantum)
			kfIn, kfN := nInBox(kf)
			kfSp := spans(kf)

			rr := replay(reg, offBs, hdr, kf, chunkLo, chunkHi)
			seedIn, seedN := nInBox(rr.seedPts)
			trSp := spans(mapFlatten(rr.traj))
			zs := zSigma(rr.traj)
			zbad, zmax := 0, 0.0
			for _, sd := range zs {
				if sd > 2.0 {
					zbad++
				}
				if sd > zmax {
					zmax = sd
				}
			}
			top, topn := "-", 0
			for k, n := range rr.desyncAt {
				if n > topn {
					top, topn = k, n
				}
			}
			results = append(results, comboResult{
				axisW: aw, centered: centered, kfInBox: kfIn, kfN: kfN, kfSpan: kfSp,
				seedInBox: seedIn, seedN: seedN, trajSpan: trSp, zbad: zbad, zmax: zmax,
				cleanRec: rr.cleanRec, desyncRec: rr.desyncRec, seedKinds: rr.seedKinds, desyncTop: fmt.Sprintf("%s×%d", top, topn),
				replayAbsN: rr.replayAbsN, replayAbsIn: rr.replayAbsIn,
			})
		}
	}

	W("%-8s %-4s | KF box   KFspanX/Y/Z        | seed box  trajspanX/Y/Z       zbad zmax  clean/desync  topDesync\n", "mode", "aw")
	W("%s\n", strings.Repeat("-", 130))
	for _, r := range results {
		mode := "range"
		if r.centered {
			mode = "centre"
		}
		W("%-8s %-4d | %2d/%-2d  %5.1f/%5.1f/%5.1f | %2d/%-2d   %5.1f/%5.1f/%5.1f  %3d %5.2f  %4d/%-5d  replayAbs=%d/%d  %s\n",
			mode, r.axisW, r.kfInBox, r.kfN, r.kfSpan[0], r.kfSpan[1], r.kfSpan[2],
			r.seedInBox, r.seedN, r.trajSpan[0], r.trajSpan[1], r.trajSpan[2],
			r.zbad, r.zmax, r.cleanRec, r.desyncRec, r.replayAbsIn, r.replayAbsN, r.desyncTop)
	}
	W("\ncible spans oracle : X~42 Y~53 Z~11 ; KF box et seed box = fraction dans la boîte oracle\n")

	// meilleure config : maximise seeds/keyframe dans la boîte, puis clean records
	best := -1
	for i, r := range results {
		if best < 0 || score(r) > score(results[best]) {
			best = i
		}
	}
	if best >= 0 {
		r := results[best]
		mode := "range"
		if r.centered {
			mode = "centre"
		}
		W("\n>>> MEILLEURE CONFIG : mode=%s axisW=%d  KFbox=%d/%d seedbox=%d/%d clean=%d\n",
			mode, r.axisW, r.kfInBox, r.kfN, r.seedInBox, r.seedN, r.cleanRec)

		// re-décode la meilleure config pour dumper les trajectoires + keyframe
		filmdec.SetAbsoluteAxisW(uint(r.axisW))
		if r.centered {
			filmdec.SetAbsDequantMode(filmdec.AbsDequantCenteredQuantum)
		} else {
			filmdec.SetAbsDequantMode(filmdec.AbsDequantRange)
		}
		kf := decodeKeyframe(pay02, hdr, i0off, r.axisW, r.centered, quantum)
		rr := replay(reg, offBs, hdr, kf, chunkLo, chunkHi)
		var tb strings.Builder
		fmt.Fprintf(&tb, "# MEILLEURE CONFIG mode=%s axisW=%d quantum=%.5f\n", mode, r.axisW, quantum)
		fmt.Fprintf(&tb, "# KEYFRAME positions i0 (offset +%d) :\n", i0off)
		for _, s := range hslots {
			if v, ok := kf[s]; ok {
				fmt.Fprintf(&tb, "  slot=%d  (%.3f, %.3f, %.3f)  inBox=%v\n", s, v[0], v[1], v[2], inBox(v))
			}
		}
		fmt.Fprintf(&tb, "\n# TRAJECTOIRES accumulées (bipeds, chunks %02d..%02d) :\n", chunkLo, chunkHi)
		for _, s := range bipedSlots {
			seq := rr.traj[int(s)]
			fmt.Fprintf(&tb, "\n== slot %d : %d points ==\n", s, len(seq))
			for i, p := range seq {
				if i < 40 || i >= len(seq)-5 {
					fmt.Fprintf(&tb, "  [%3d] (%.3f, %.3f, %.3f)\n", i, p[0], p[1], p[2])
				} else if i == 40 {
					fmt.Fprintf(&tb, "  ... (%d points) ...\n", len(seq)-45)
				}
			}
		}
		_ = os.MkdirAll(scratch, 0o755)
		_ = os.WriteFile(scratch+"/best_traj.txt", []byte(tb.String()), 0o644)
	}

	_ = os.MkdirAll(scratch, 0o755)
	_ = os.WriteFile(scratch+"/abs_sweep.txt", []byte(b.String()), 0o644)
	fmt.Print(b.String())
	fmt.Printf("\nfichiers: %s/abs_sweep.txt | %s/best_traj.txt\n", scratch, scratch)

	// reset config
	filmdec.SetAbsoluteAxisW(0)
	filmdec.SetAbsDequantMode(filmdec.AbsDequantRange)
}

// score : priorité seeds dans la boîte, puis keyframe box, puis clean records.
func score(r comboResult) float64 {
	sf := 0.0
	if r.seedN > 0 {
		sf = float64(r.seedInBox) / float64(r.seedN)
	}
	kf := 0.0
	if r.kfN > 0 {
		kf = float64(r.kfInBox) / float64(r.kfN)
	}
	return sf*1000 + kf*500 + float64(r.cleanRec) - float64(r.zbad)*10
}

func mapFlatten(m map[int][][3]float64) map[int][3]float64 {
	// pour spans : réutilise chaque point comme "pts" -> on aplatit en indexant
	out := map[int][3]float64{}
	i := 0
	for _, seq := range m {
		for _, p := range seq {
			out[i] = p
			i++
		}
	}
	return out
}

// modalQuantum : le pas de grille = bucket (0.0005) le plus fréquent parmi les |diffs| consécutifs
// non nuls, tous axes/slots. Robuste au bruit float (contrairement au min global).
func modalQuantum(seqs map[int][][3]float64) float64 {
	hist := map[int]int{}
	for _, s := range seqs {
		for i := 1; i < len(s); i++ {
			for a := 0; a < 3; a++ {
				d := math.Abs(s[i][a] - s[i-1][a])
				if d > 0.008 && d < 0.05 { // fenêtre autour d'1 cran (écarte 0 et les gros deltas)
					hist[int(d/0.0005)]++
				}
			}
		}
	}
	bestB, bestN := 0, 0
	for b, n := range hist {
		if n > bestN {
			bestB, bestN = b, n
		}
	}
	if bestB == 0 {
		return 0.0138
	}
	return (float64(bestB) + 0.5) * 0.0005
}

func buildStateBits(pay []byte, hdr map[int]int) []int {
	sb := make([]int, 0, len(hdr))
	for _, s := range sortedKeys(hdr) {
		sb = append(sb, hdr[s]+64)
	}
	return sb
}

// dumpKeyframeRawQ imprime les q bruts (avant déquant) au keyframe pour aw 6/12/14, par biped.
func dumpKeyframeRawQ(pay []byte, hdr map[int]int, off int) {
	fmt.Printf("=== q bruts keyframe (offset i0 +%d) : centre attendu = 2^(w-1) ===\n", off)
	for _, s := range sortedKeys(hdr) {
		ax, ok := zeroGate(pay, hdr[s]+64+off)
		if !ok {
			fmt.Printf("  slot=%d gate!=0\n", s)
			continue
		}
		var q6, q12, q14 [3]uint64
		for i := 0; i < 3; i++ {
			q6[i] = readBits(pay, ax+i*6, 6)
			q12[i] = readBits(pay, ax+i*12, 12)
			q14[i] = readBits(pay, ax+i*14, 14)
		}
		fmt.Printf("  slot=%d  q6=%v (c=32)  q12=%v (c=2048)  q14=%v (c=8192)\n", s, q6, q12, q14)
	}
	fmt.Println()
}

func sortedKeys(m map[int]int) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}
