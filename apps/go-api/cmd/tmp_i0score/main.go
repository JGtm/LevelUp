// tmp_i0score — VALIDATION offline du fix i0 (accumulation) contre l'oracle CE.
//
// Étage 1 (gate) : reconstruit les trajectoires OFFLINE des bipeds (slots 512-519) en rejouant
// les paquets type-0 sur un World persistant seedé au keyframe, avec accumulation i0 vivant dans
// filmdec (SetPositionAccumulator). Puis calcule sur ces trajectoires les 4 invariants de
// signature dérivés de l'oracle : (i) grille = quantum 0.0138 (diffs = multiples du quantum),
// (ii) distribution du pas, (iii) z quasi-constant par slot, (iv) repère commun étalé
// (barycentres distincts, pas collés à 0). Le quantum est MESURÉ sur l'oracle puis injecté
// (SetDeltaQuantum) — il n'est pas deviné.
//
// Étage 2 (confirmatoire) : DTW sur les séquences de pas offline<->oracle + assignation gloutonne.
// Les slots offline (512-519) et oracle (datum store) sont des entités DISJOINTES : Étage 2 est
// indicatif, il NE bloque PAS (Étage 1 reste le gate).
//
// Sorties : scratchpad/offline_traj.txt (trajectoires) + scratchpad/i0_score.txt (scoring).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_i0score [chunkLo chunkHi]
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
)

var bipedSlots = []uint32{512, 513, 514, 515, 516, 517, 518, 519}

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

// ---- oracle ----

type rec struct {
	slot    int
	x, y, z float64
}

func loadOracle(p string) []rec {
	f, err := os.Open(p)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []rec
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	line := 0
	for sc.Scan() {
		t := strings.TrimSpace(sc.Text())
		line++
		if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "eid") {
			continue
		}
		f := strings.Split(t, ",")
		if len(f) < 6 {
			continue
		}
		slot, e1 := strconv.Atoi(strings.TrimSpace(f[1]))
		x, e2 := strconv.ParseFloat(strings.TrimSpace(f[3]), 64)
		y, e3 := strconv.ParseFloat(strings.TrimSpace(f[4]), 64)
		z, e4 := strconv.ParseFloat(strings.TrimSpace(f[5]), 64)
		if e1 == nil && e2 == nil && e3 == nil && e4 == nil {
			out = append(out, rec{slot, x, y, z})
		}
	}
	return out
}

// bySlot groupe en séquences ordonnées (ordre du fichier = ordre de capture).
func bySlot(rs []rec) map[int][][3]float64 {
	m := map[int][][3]float64{}
	for _, r := range rs {
		m[r.slot] = append(m[r.slot], [3]float64{r.x, r.y, r.z})
	}
	return m
}

// ---- mesure de grille (quantum) ----

// gridQuantum : plus petit PAS consécutif (même slot, ordre temporel) positif d'un axe, seuil
// >0.005 pour écarter le bruit float. C'est le cran de grille physique (un delta unitaire).
// L'inter-slots serait faussé par les offsets absolus distincts des entités.
func gridQuantum(seqs map[int][][3]float64, axis int) float64 {
	best := math.MaxFloat64
	for _, s := range seqs {
		for i := 1; i < len(s); i++ {
			d := math.Abs(s[i][axis] - s[i-1][axis])
			if d > 0.005 && d < best {
				best = d
			}
		}
	}
	if best == math.MaxFloat64 {
		return 0
	}
	return best
}

// multipleFrac : fraction des diffs consécutives (par axe, par slot) qui sont des multiples
// ENTIERS de q (tolérance 8%). Diffs nuls ignorés.
func multipleFrac(seqs map[int][][3]float64, q float64) (frac float64, n int) {
	hit := 0
	for _, s := range seqs {
		for i := 1; i < len(s); i++ {
			for a := 0; a < 3; a++ {
				d := math.Abs(s[i][a] - s[i-1][a])
				if d < 1e-4 {
					continue
				}
				n++
				k := d / q
				if math.Abs(k-math.Round(k)) < 0.08*math.Max(1, math.Round(k)) {
					hit++
				}
			}
		}
	}
	if n == 0 {
		return 0, 0
	}
	return float64(hit) / float64(n), n
}

// ---- distribution du pas ----

func stepStats(seqs map[int][][3]float64) (mean, p50, p90, p99, mx float64, teleport int) {
	var steps []float64
	for _, s := range seqs {
		for i := 1; i < len(s); i++ {
			dx, dy, dz := s[i][0]-s[i-1][0], s[i][1]-s[i-1][1], s[i][2]-s[i-1][2]
			d := math.Sqrt(dx*dx + dy*dy + dz*dz)
			if d < 1e-9 {
				continue
			}
			steps = append(steps, d)
			if d > 5 {
				teleport++
			}
		}
	}
	if len(steps) == 0 {
		return
	}
	sort.Float64s(steps)
	sum := 0.0
	for _, v := range steps {
		sum += v
	}
	mean = sum / float64(len(steps))
	pc := func(p float64) float64 { return steps[int(p*float64(len(steps)-1))] }
	return mean, pc(0.5), pc(0.9), pc(0.99), steps[len(steps)-1], teleport
}

// ---- z-flatness + barycentres ----

func zFlat(seqs map[int][][3]float64) map[int]float64 {
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

func barycentres(seqs map[int][][3]float64) map[int][3]float64 {
	out := map[int][3]float64{}
	for s, seq := range seqs {
		if len(seq) == 0 {
			continue
		}
		var c [3]float64
		for _, p := range seq {
			c[0] += p[0]
			c[1] += p[1]
			c[2] += p[2]
		}
		n := float64(len(seq))
		out[s] = [3]float64{c[0] / n, c[1] / n, c[2] / n}
	}
	return out
}

// ---- reconstruction offline (World persistant, accumulation dans filmdec) ----

func offlineTraj(reg *filmdec.Registry, offBs []binding, chunkLo, chunkHi int) map[int][][3]float64 {
	w := filmdec.NewWorld(reg)
	bipedSet := map[uint32]bool{}
	for _, s := range bipedSlots {
		bipedSet[s] = true
	}
	for _, b := range offBs {
		w.BindFull(b.full, b.ti)
	}
	traj := map[int][][3]float64{}
	filmdec.SetPositionAccumulator(w)
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) {
		if bipedSet[s.Slot] {
			traj[int(s.Slot)] = append(traj[int(s.Slot)], [3]float64{float64(s.Vec[0]), float64(s.Vec[1]), float64(s.Vec[2])})
		}
	})
	filmdec.SetRecordStateParam(2)
	cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: idLow}
	// ScanFrameTargets récolte TOUS les deltas bipeds d'une frame au-delà des desyncs
	// (DecodeFrameRecords s'arrête au 1er desync et n'attrape qu'un record/paquet). Les records
	// acceptés sont re-décodés avec l'accumulation vivante ; les essais sont supprimés (patch
	// ScanFrameTargets). Le World persiste entre paquets -> accumulation continue par slot.
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
	return traj
}

type binding struct {
	full uint32
	ti   uint32
}

func offlineBindings(pay []byte) []binding {
	recs := filmdec.WalkKeyframeWorld(pay)
	out := make([]binding, 0, len(recs))
	for _, r := range recs {
		out = append(out, binding{full: uint32((r.Gen << 30) | r.Slot), ti: uint32(r.TI)})
	}
	return out
}

// ---- diagnostic DELTA BRUT (accumWorld=nil) : teste le quantum indépendamment du seed absolu
// (Cliffhanger, miscalibré) et du mur de desync i63. Chaque record biped décodé fait feu du hook
// pour i0 AVANT tout desync aval ; avec accumWorld=nil, les chemins delta émettent le delta BRUT,
// les chemins absolus la position brute, le keep rien. On récolte donc directement la distribution
// des deltas répliqués (le coeur du fix), sans dépendre de la reconstruction spatiale complète.

type deltaDiag struct {
	kindCount map[string]int
	dmag      []float64 // magnitude euclidienne des deltas (chemins delta)
	dax       []float64 // |composantes| des deltas (par axe), pour test grille
	globalMax float64   // max |composante| sur TOUS les slots (preuve anti-aberrant ~1e28)
	aberrant  int       // # émissions |composante| > 1e6 (doit être 0 après le fix)
	nAll      int       // # total d'émissions i0 (tous slots)
}

func rawDeltaDiag(reg *filmdec.Registry, offBs []binding, chunkLo, chunkHi int) deltaDiag {
	bipedSet := map[uint32]bool{}
	for _, s := range bipedSlots {
		bipedSet[s] = true
	}
	d := deltaDiag{kindCount: map[string]int{}}
	filmdec.SetPositionAccumulator(nil) // pas d'accumulation : le hook reçoit le delta BRUT
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) {
		d.nAll++
		for a := 0; a < 3; a++ {
			av := math.Abs(float64(s.Vec[a]))
			if av > d.globalMax {
				d.globalMax = av
			}
			if av > 1e6 || math.IsNaN(float64(s.Vec[a])) || math.IsInf(float64(s.Vec[a]), 0) {
				d.aberrant++
			}
		}
		if !bipedSet[s.Slot] {
			return
		}
		d.kindCount[s.Kind.String()]++
		switch s.Kind {
		case filmdec.PosKindDelta8, filmdec.PosKindDeltaAxis:
			m := math.Sqrt(float64(s.Vec[0]*s.Vec[0] + s.Vec[1]*s.Vec[1] + s.Vec[2]*s.Vec[2]))
			d.dmag = append(d.dmag, m)
			for a := 0; a < 3; a++ {
				if v := math.Abs(float64(s.Vec[a])); v > 1e-6 {
					d.dax = append(d.dax, v)
				}
			}
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
			w := filmdec.NewWorld(reg)
			for _, b := range offBs {
				w.BindFull(b.full, b.ti)
			}
			br := filmdec.NewBitReader(pay)
			_, _ = filmdec.DecodeFrameRecords(br, w, cfg) // le 1er record fait feu i0 avant tout desync
		}
	}
	filmdec.SetPositionCaptureHook(nil)
	return d
}

// fracMultipleOf : fraction de vals qui sont des multiples ENTIERS de q (tol 10%).
func fracMultipleOf(vals []float64, q float64) float64 {
	if len(vals) == 0 || q <= 0 {
		return 0
	}
	hit := 0
	for _, v := range vals {
		k := v / q
		if math.Abs(k-math.Round(k)) < 0.10*math.Max(1, math.Round(k)) {
			hit++
		}
	}
	return float64(hit) / float64(len(vals))
}

func pcts(vals []float64) (mean, p50, p90, p99, mx float64) {
	if len(vals) == 0 {
		return
	}
	s := append([]float64(nil), vals...)
	sort.Float64s(s)
	sum := 0.0
	for _, v := range s {
		sum += v
	}
	mean = sum / float64(len(s))
	pc := func(p float64) float64 { return s[int(p*float64(len(s)-1))] }
	return mean, pc(0.5), pc(0.9), pc(0.99), s[len(s)-1]
}

// ---- Étage 2 : DTW sur séquences de pas (magnitudes) ----

func stepSeq(seq [][3]float64, cap int) []float64 {
	var out []float64
	for i := 1; i < len(seq) && len(out) < cap; i++ {
		dx, dy, dz := seq[i][0]-seq[i-1][0], seq[i][1]-seq[i-1][1], seq[i][2]-seq[i-1][2]
		out = append(out, math.Sqrt(dx*dx+dy*dy+dz*dz))
	}
	return out
}

func dtw(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return math.MaxFloat64
	}
	prev := make([]float64, len(b)+1)
	cur := make([]float64, len(b)+1)
	for j := 1; j <= len(b); j++ {
		prev[j] = math.MaxFloat64
	}
	prev[0] = 0
	for i := 1; i <= len(a); i++ {
		cur[0] = math.MaxFloat64
		for j := 1; j <= len(b); j++ {
			cost := math.Abs(a[i-1] - b[j-1])
			m := prev[j]
			if prev[j-1] < m {
				m = prev[j-1]
			}
			if cur[j-1] < m {
				m = cur[j-1]
			}
			cur[j] = cost + m
		}
		prev, cur = cur, prev
	}
	return prev[len(b)] / float64(len(a)+len(b))
}

func main() {
	chunkLo, chunkHi := 3, 3
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[1], "%d", &chunkLo)
		fmt.Sscanf(os.Args[2], "%d", &chunkHi)
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	pay02 := framePayload(inflate(cache+"/chunk_02.bin"), 2)
	offBs := offlineBindings(pay02)

	// ===== ORACLE : signature de référence + quantum mesuré =====
	oracleSeqs := bySlot(loadOracle(oracleP))
	// filtrer aux slots avec assez de points (bipeds vivants)
	oq := [3]float64{gridQuantum(oracleSeqs, 0), gridQuantum(oracleSeqs, 1), gridQuantum(oracleSeqs, 2)}
	// quantum = MÉDIANE des 3 axes (X/Y concordent à 0.0138 ; Z peut avoir un pas plus fin isolé).
	sorted3 := []float64{oq[0], oq[1], oq[2]}
	sort.Float64s(sorted3)
	quantum := sorted3[1]
	filmdec.SetDeltaQuantum(float32(quantum)) // DÉRIVÉ de l'oracle, injecté dans le deser

	omean, op50, op90, op99, omx, otel := stepStats(oracleSeqs)
	ofrac, on := multipleFrac(oracleSeqs, quantum)

	// ===== DIAGNOSTIC DELTA BRUT (coeur du fix, indépendant du seed/desync) =====
	dd := rawDeltaDiag(reg, offBs, chunkLo, chunkHi)
	daxFrac := fracMultipleOf(dd.dax, quantum)
	dmMean, dmP50, dmP90, dmP99, dmMx := pcts(dd.dmag)

	// ===== OFFLINE : reconstruction + signature =====
	off := offlineTraj(reg, offBs, chunkLo, chunkHi)
	fq := [3]float64{gridQuantum(off, 0), gridQuantum(off, 1), gridQuantum(off, 2)}
	fmean, fp50, fp90, fp99, fmx, ftel := stepStats(off)
	ffrac, fn := multipleFrac(off, quantum)
	fz := zFlat(off)
	fbary := barycentres(off)

	// ===== Étage 2 : DTW offline<->oracle (indicatif) =====
	oSlots := []int{}
	for s, seq := range oracleSeqs {
		if len(seq) >= 20 {
			oSlots = append(oSlots, s)
		}
	}
	sort.Ints(oSlots)
	var e2lines []string
	usedOracle := map[int]bool{}
	var residuals []float64
	for _, s := range bipedSlots {
		fs := stepSeq(off[int(s)], 200)
		if len(fs) < 10 {
			e2lines = append(e2lines, fmt.Sprintf("  offline slot=%d : <10 pas, ignoré", s))
			continue
		}
		bestO, bestD := -1, math.MaxFloat64
		for _, os := range oSlots {
			if usedOracle[os] {
				continue
			}
			d := dtw(fs, stepSeq(oracleSeqs[os], 200))
			if d < bestD {
				bestD, bestO = d, os
			}
		}
		if bestO >= 0 {
			usedOracle[bestO] = true
			residuals = append(residuals, bestD)
			e2lines = append(e2lines, fmt.Sprintf("  offline slot=%d -> oracle slot=%d  DTW-résidu=%.4f", s, bestO, bestD))
		}
	}
	meanRes := 0.0
	for _, r := range residuals {
		meanRes += r
	}
	if len(residuals) > 0 {
		meanRes /= float64(len(residuals))
	}

	// ===== écriture trajectoires =====
	var tb strings.Builder
	fmt.Fprintf(&tb, "# trajectoires OFFLINE reconstruites (bipeds 512-519, chunks %02d..%02d)\n", chunkLo, chunkHi)
	fmt.Fprintf(&tb, "# quantum delta injecté (dérivé oracle) = %.6f\n", quantum)
	for _, s := range bipedSlots {
		seq := off[int(s)]
		fmt.Fprintf(&tb, "\n== slot %d : %d points ==\n", s, len(seq))
		for i, p := range seq {
			if i < 60 || i >= len(seq)-5 { // début + fin
				fmt.Fprintf(&tb, "  [%3d] (%.3f, %.3f, %.3f)\n", i, p[0], p[1], p[2])
			} else if i == 60 {
				fmt.Fprintf(&tb, "  ... (%d points intermédiaires) ...\n", len(seq)-65)
			}
		}
	}
	_ = os.MkdirAll(scratch, 0o755)
	_ = os.WriteFile(scratch+"/offline_traj.txt", []byte(tb.String()), 0o644)

	// ===== écriture scoring =====
	var b strings.Builder
	W := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	W("========== SCORING i0 (offline vs oracle CE) ==========\n\n")
	W("QUANTUM DE GRILLE (oracle, min diff distincte par axe) : X=%.5f Y=%.5f Z=%.5f -> quantum=%.5f\n", oq[0], oq[1], oq[2], quantum)
	W("  (injecté dans le deser via SetDeltaQuantum ; le quantum est MESURÉ, pas deviné)\n\n")

	W("---- ORACLE (référence) ----\n")
	W("  pas: mean=%.4f p50=%.4f p90=%.4f p99=%.4f max=%.4f teleports>5=%d\n", omean, op50, op90, op99, omx, otel)
	W("  diffs multiples du quantum : %.1f%% (n=%d)\n\n", ofrac*100, on)

	W("---- ANTI-ABERRANT (toutes émissions i0, tous slots) ----\n")
	W("  émissions i0 totales=%d  max|composante|=%.4f  aberrantes(|v|>1e6/NaN/Inf)=%d\n\n", dd.nAll, dd.globalMax, dd.aberrant)

	W("---- DIAGNOSTIC DELTA BRUT offline (accumWorld=nil, coeur du fix) ----\n")
	W("  répartition des chemins i0 bipeds : %v\n", dd.kindCount)
	W("  |composante delta| multiples du quantum : %.1f%% (n=%d)\n", daxFrac*100, len(dd.dax))
	W("  magnitude delta: mean=%.4f p50=%.4f p90=%.4f p99=%.4f max=%.4f (n=%d)\n\n", dmMean, dmP50, dmP90, dmP99, dmMx, len(dd.dmag))

	W("---- OFFLINE (reconstruit) ----\n")
	W("  quantum de grille reconstruit : X=%.5f Y=%.5f Z=%.5f\n", fq[0], fq[1], fq[2])
	W("  diffs multiples du quantum oracle : %.1f%% (n=%d)\n", ffrac*100, fn)
	W("  pas: mean=%.4f p50=%.4f p90=%.4f p99=%.4f max=%.4f teleports>5=%d\n\n", fmean, fp50, fp90, fp99, fmx, ftel)

	W("---- INVARIANT (iii) z quasi-constant (ecart-type z par slot) ----\n")
	zbad := 0
	for _, s := range bipedSlots {
		if sd, ok := fz[int(s)]; ok {
			flag := ""
			if sd > 2.0 {
				flag = "  <-- z NON plat"
				zbad++
			}
			W("  slot=%d  sigma_z=%.4f%s\n", s, sd, flag)
		}
	}
	W("\n---- INVARIANT (iv) repère commun étalé (barycentres par slot) ----\n")
	nAtOrigin := 0
	bcs := [][3]float64{}
	for _, s := range bipedSlots {
		if c, ok := fbary[int(s)]; ok {
			W("  slot=%d  barycentre=(%.2f, %.2f, %.2f)  npts=%d\n", s, c[0], c[1], c[2], len(off[int(s)]))
			bcs = append(bcs, c)
			if math.Abs(c[0]) < 0.5 && math.Abs(c[1]) < 0.5 && math.Abs(c[2]) < 0.5 {
				nAtOrigin++
			}
		}
	}
	// distinction des barycentres
	distinctB := 0
	for i := range bcs {
		uniq := true
		for j := range bcs {
			if i != j {
				d := math.Abs(bcs[i][0]-bcs[j][0]) + math.Abs(bcs[i][1]-bcs[j][1]) + math.Abs(bcs[i][2]-bcs[j][2])
				if d < 0.5 {
					uniq = false
				}
			}
		}
		if uniq {
			distinctB++
		}
	}
	W("  barycentres distincts=%d/%d  collés à l'origine=%d\n\n", distinctB, len(bcs), nAtOrigin)

	W("---- ÉTAGE 2 (DTW offline<->oracle, INDICATIF, entités disjointes) ----\n")
	for _, l := range e2lines {
		W("%s\n", l)
	}
	W("  DTW-résidu moyen=%.4f (indicatif : slots offline/oracle = entités disjointes)\n\n", meanRes)

	// ===== verdict Étage 1 =====
	quantumOK := quantum > 0.010 && quantum < 0.018
	gridOK := ffrac > 0.90
	stepOK := fmx < 6 && ftel == 0 && fmean < 0.5
	zOK := zbad == 0 && len(fz) > 0
	spreadOK := distinctB >= len(bcs)-1 && nAtOrigin == 0 && len(bcs) >= 6
	aberrantOK := dd.aberrant == 0 && dd.globalMax < 1e6 // preuve directe : aucune émission ~1e28/NaN/Inf
	W("========== VERDICT ÉTAGE 1 ==========\n")
	W("  quantum ~0.0138           : %v (%.5f)\n", quantumOK, quantum)
	W("  grille (diffs multiples)  : %v (%.1f%%)\n", gridOK, ffrac*100)
	W("  distribution du pas saine : %v (max=%.3f teleports=%d mean=%.4f)\n", stepOK, fmx, ftel, fmean)
	W("  z plat                    : %v\n", zOK)
	W("  repère commun étalé       : %v (distincts=%d/%d origine=%d)\n", spreadOK, distinctB, len(bcs), nAtOrigin)
	W("  aberrant éliminé          : %v (max|composante|=%.3f aberrantes=%d/%d émissions)\n", aberrantOK, dd.globalMax, dd.aberrant, dd.nAll)
	all := quantumOK && gridOK && stepOK && zOK && spreadOK && aberrantOK
	W("\n  ÉTAGE 1 COMPLET : %v\n", all)

	_ = os.WriteFile(scratch+"/i0_score.txt", []byte(b.String()), 0o644)
	fmt.Print(b.String())
	fmt.Printf("\nfichiers: %s/offline_traj.txt | %s/i0_score.txt\n", scratch, scratch)
}
