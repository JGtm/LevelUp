// tmp_dense_i0 — DENSIFICATION des positions bipeds par ACCEPTATION d'i0 AVANT le mur 55.
//
// Blocage precedent (tmp_bipedframepos) : le filtre exigeait un record COMPLET clean
// (DesyncAt==-1). Or les vrais bipeds desyncent au composant 55/57/60 (composants unit
// non-portes) APRES i0 (composant 0). Ils etaient donc rejetes -> 527 records / 32 slots
// (1.3% de l'oracle), overlap par-slot 54%.
//
// FIX : i0 est le composant 0, lu AVANT le mur. On decode l'en-tete court (type+id+mask)
// + i0 via TryDeltaAt (qui execute decodeDelta jusqu'au desync : i0 emet AVANT). On
// ACCEPTE le sample i0 des qu'il decode, meme si le record desync plus loin. La densite
// explose -> il faut un filtre fort : CONTINUITE SERREE par slot. Un vrai pas biped fait
// ~0.04-0.2u ; un faux positif abs aleatoire dans une boite ~113u a <1% de chance de
// tomber a <=CONT (0.5u) de la derniere position. La continuite ecrase les faux positifs.
//
// SEED par slot : consensus — on amorce la chaine quand SEEDK candidats ABS in-box
// tombent dans un rayon SEEDR (vrai depart stable), puis les deltas s'accumulent (harness
// maintient lastPos par slot : abs=vec direct, delta=lastPos+d). RESEED : apres RESEED
// rejets consecutifs (respawn/teleport), on reforme un consensus (0=jamais).
//
// JUGE : densite (records/slot vs oracle), overlap par-slot (points offline DANS la boite
// oracle du MEME slot ; doit monter >85%), forme (PNG offline vs oracle).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_dense_i0 [filmDir] [out.png]
// Env : IDLOW(11) ABSW(13) QUANT(0.0138) CONT(0.5) SEEDK(3) SEEDR(0.5) RESEED(20)
//
//	OLDACCEPT=1 (reproduit AVANT : exige record clean + jump<=6, pour AVANT/APRES).
package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
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
	scratch = `c:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration/d6f109c3-2822-4e19-8fd4-cf65c50fec3d/scratchpad`
	oracleP = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/ce_pos_oracle.csv`
)

// boite oracle globale (marge 5%) — sanity in-box.
var boxMin = [3]float32{-6.33, -25.14, -4.20}
var boxMax = [3]float32{35.70, 27.50, 7.08}

func envI(k string, d int) int {
	if v := os.Getenv(k); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			return n
		}
	}
	return d
}
func envF(k string, d float64) float64 {
	if v := os.Getenv(k); v != "" {
		if n, e := strconv.ParseFloat(v, 64); e == nil {
			return n
		}
	}
	return d
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

func listFrames(d []byte) [][]byte {
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

func seedWorld(dir string, reg *filmdec.Registry) (*filmdec.World, map[uint32]bool) {
	raw, _ := os.ReadFile(dir + "/world_dump_full.txt")
	w := filmdec.NewWorld(reg)
	ti35 := map[uint32]bool{}
	sc := bufio.NewScanner(bytes.NewReader(raw))
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
			id, e1 := strconv.ParseUint(kv[0], 10, 64)
			ti, e2 := strconv.Atoi(kv[1])
			if e1 != nil || e2 != nil {
				continue
			}
			w.BindFull(uint32(id), uint32(ti))
			slot := uint32(id) & 0x3fffffff
			if ti == 35 && slot >= 528 {
				ti35[slot] = true
			}
		}
	}
	return w, ti35
}

func loadOracle() map[uint32][][3]float32 {
	f, _ := os.Open(oracleP)
	if f == nil {
		return nil
	}
	defer f.Close()
	m := map[uint32][][3]float32{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "eid") {
			continue
		}
		p := strings.Split(line, ",")
		if len(p) < 6 {
			continue
		}
		slot, _ := strconv.Atoi(p[1])
		x, _ := strconv.ParseFloat(p[3], 32)
		y, _ := strconv.ParseFloat(p[4], 32)
		z, _ := strconv.ParseFloat(p[5], 32)
		m[uint32(slot)] = append(m[uint32(slot)], [3]float32{float32(x), float32(y), float32(z)})
	}
	return m
}

func inBox(v [3]float32) bool {
	for i := 0; i < 3; i++ {
		sp := boxMax[i] - boxMin[i]
		if v[i] < boxMin[i]-0.05*sp || v[i] > boxMax[i]+0.05*sp {
			return false
		}
	}
	return true
}

func finite(v [3]float32) bool {
	for i := 0; i < 3; i++ {
		if math.IsNaN(float64(v[i])) || math.IsInf(float64(v[i]), 0) {
			return false
		}
	}
	return true
}

func dist(a, b [3]float32) float64 {
	dx, dy, dz := float64(a[0]-b[0]), float64(a[1]-b[1]), float64(a[2]-b[2])
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

func isAbs(k filmdec.PosKind) bool {
	return k == filmdec.PosKindAbsolute || k == filmdec.PosKindAbsFallback
}

// slotState : chaine de continuite d'un slot.
type slotState struct {
	seeded    bool
	lastPos   [3]float32
	seedBuf   [][3]float32 // candidats abs recents (pour consensus)
	rejectRun int
}

func main() {
	dir, out := defFilm, scratch+"/traj_dense_i0.png"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if len(os.Args) > 2 {
		out = os.Args[2]
	}
	idlow := envI("IDLOW", 11)
	absw := envI("ABSW", 13)
	quant := float32(envF("QUANT", 0.0138))
	cont := envF("CONT", 0.5)
	seedK := envI("SEEDK", 3)
	seedR := envF("SEEDR", 0.5)
	reseed := envI("RESEED", 20)
	oldAccept := os.Getenv("OLDACCEPT") != ""

	filmdec.SetRecordStateParam(2)
	filmdec.WorldPositionRange = filmdec.QuantRangeCEBiped
	filmdec.SetAbsoluteAxisW(uint(absw))
	filmdec.SetDeltaQuantum(quant)

	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	world, targets := seedWorld(dir, reg)
	cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: idlow}
	fmt.Printf("cfg IDLow=%d ABSW=%d QUANT=%.5f CONT=%.2f SEEDK=%d SEEDR=%.2f RESEED=%d OLDACCEPT=%v targets=%d\n",
		idlow, absw, quant, cont, seedK, seedR, reseed, oldAccept, len(targets))

	// capture du PREMIER sample i0 de chaque essai TryDeltaAt (reset avant chaque appel).
	var firstKind filmdec.PosKind
	var firstVec [3]float32
	var firstHave bool
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) {
		if !firstHave {
			firstKind, firstVec, firstHave = s.Kind, s.Vec, true
		}
	})
	defer filmdec.SetPositionCaptureHook(nil)

	state := map[uint32]*slotState{}
	bySlot := map[uint32][][3]float32{}
	byKind := map[uint32]map[filmdec.PosKind][][3]float32{}
	kindSeen := map[filmdec.PosKind]int{}
	var accepted, rejBox, rejJump, rejNoSeed, seeds, reseeds int

	for idx := 3; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))) {
			frameLen := len(fr) * 8
			b := 0
			for b < frameLen-24 {
				firstHave = false
				rec, end, ok := filmdec.TryDeltaAt(fr, b, world, cfg)
				if !targets[rec.Slot] || !firstHave || !finite(firstVec) {
					b++
					continue
				}
				// AVANT : exige record clean (DesyncAt==-1 => ok) + saut<=6 (reproduction).
				if oldAccept {
					if !ok {
						b++
						continue
					}
				}
				slot := rec.Slot
				kindSeen[firstKind]++
				st := state[slot]
				if st == nil {
					st = &slotState{}
					state[slot] = st
				}
				// resolution en position absolue.
				var cand [3]float32
				resolvable := false
				if isAbs(firstKind) {
					cand, resolvable = firstVec, true
				} else if st.seeded {
					cand = [3]float32{st.lastPos[0] + firstVec[0], st.lastPos[1] + firstVec[1], st.lastPos[2] + firstVec[2]}
					resolvable = true
				}
				if !resolvable {
					rejNoSeed++
					b++
					continue
				}
				if !inBox(cand) {
					rejBox++
					b++
					continue
				}

				accept := false
				if oldAccept {
					// AVANT : seed=1er in-box, puis continuite lache 6u.
					if !st.seeded {
						st.seeded, st.lastPos = true, cand
						accept = true
					} else if dist(cand, st.lastPos) <= 6.0 {
						st.lastPos, accept = cand, true
					} else {
						rejJump++
					}
				} else if !st.seeded {
					// SEED consensus : seulement depuis abs in-box.
					if isAbs(firstKind) {
						if seedFromConsensus(st, cand, seedK, seedR) {
							seeds++
							accept = true
						}
					}
				} else {
					if dist(cand, st.lastPos) <= cont {
						st.lastPos, st.rejectRun, accept = cand, 0, true
					} else {
						rejJump++
						st.rejectRun++
						// RESEED : trop de rejets consecutifs (respawn/teleport) -> reforme consensus.
						if reseed > 0 && st.rejectRun >= reseed && isAbs(firstKind) {
							st.seeded, st.seedBuf = false, nil
							if seedFromConsensus(st, cand, seedK, seedR) {
								reseeds++
								accept = true
							}
						}
					}
				}

				if accept {
					bySlot[slot] = append(bySlot[slot], cand)
					if byKind[slot] == nil {
						byKind[slot] = map[filmdec.PosKind][][3]float32{}
					}
					byKind[slot][firstKind] = append(byKind[slot][firstKind], cand)
					accepted++
					if end > b {
						b = end
					} else {
						b++
					}
					continue
				}
				b++
			}
		}
	}
	filmdec.SetPositionCaptureHook(nil)

	oracle := loadOracle()
	fmt.Printf("\naccepted=%d seeds=%d reseeds=%d rejBox=%d rejJump=%d rejNoSeed=%d ; i0-kinds vus=%v\n",
		accepted, seeds, reseeds, rejBox, rejJump, rejNoSeed, kindSeen)

	report(bySlot, byKind, oracle)
	writeTraj(out, bySlot, oracle)
	writeTxt(scratch+"/dense_i0.txt", bySlot, oracle, byKind, idlow, absw, quant, cont, seedK, seedR, reseed, oldAccept, accepted)
}

// seedFromConsensus bufferise un candidat abs ; amorce si >=k candidats du buffer tombent
// dans le rayon r du candidat courant. Seed = centroide du cluster. Retourne true si amorce.
func seedFromConsensus(st *slotState, cand [3]float32, k int, r float64) bool {
	st.seedBuf = append(st.seedBuf, cand)
	if len(st.seedBuf) > 32 {
		st.seedBuf = st.seedBuf[len(st.seedBuf)-32:]
	}
	var sum [3]float32
	n := 0
	for _, p := range st.seedBuf {
		if dist(p, cand) <= r {
			sum[0] += p[0]
			sum[1] += p[1]
			sum[2] += p[2]
			n++
		}
	}
	if n >= k {
		st.seeded = true
		st.lastPos = [3]float32{sum[0] / float32(n), sum[1] / float32(n), sum[2] / float32(n)}
		st.rejectRun = 0
		st.seedBuf = nil
		return true
	}
	return false
}

func report(bySlot map[uint32][][3]float32, byKind map[uint32]map[filmdec.PosKind][][3]float32, oracle map[uint32][][3]float32) {
	// boite oracle par slot.
	oraBox := map[uint32][2][3]float32{}
	for s, ps := range oracle {
		var mn, mx [3]float32
		for i, v := range ps {
			if i == 0 {
				mn, mx = v, v
			}
			for a := 0; a < 3; a++ {
				if v[a] < mn[a] {
					mn[a] = v[a]
				}
				if v[a] > mx[a] {
					mx[a] = v[a]
				}
			}
		}
		oraBox[s] = [2][3]float32{mn, mx}
	}
	inOraBox := func(s uint32, v [3]float32) bool {
		b, ok := oraBox[s]
		if !ok {
			return false
		}
		for a := 0; a < 3; a++ {
			sp := b[1][a] - b[0][a]
			if sp < 1 {
				sp = 1
			}
			if v[a] < b[0][a]-0.1*sp || v[a] > b[1][a]+0.1*sp {
				return false
			}
		}
		return true
	}

	var slots []uint32
	for s := range bySlot {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })

	totOff, totOra, totOverlap, totWithOra := 0, 0, 0, 0
	fmt.Printf("\n%-6s %8s %8s %8s %s\n", "slot", "off", "oracle", "inOraBx", "off box (x,y,z)")
	for _, s := range slots {
		pts := bySlot[s]
		totOff += len(pts)
		ov := 0
		var mn, mx [3]float32
		for i, v := range pts {
			if i == 0 {
				mn, mx = v, v
			}
			for a := 0; a < 3; a++ {
				if v[a] < mn[a] {
					mn[a] = v[a]
				}
				if v[a] > mx[a] {
					mx[a] = v[a]
				}
			}
			if inOraBox(s, v) {
				ov++
			}
		}
		if _, has := oracle[s]; has {
			totOverlap += ov
			totWithOra += len(pts)
		}
		no := len(oracle[s])
		totOra += no
		if len(pts) >= 5 {
			fmt.Printf("%-6d %8d %8d %7d%% x[%.1f,%.1f] y[%.1f,%.1f] z[%.1f,%.1f]\n",
				s, len(pts), no, ov*100/max1(len(pts)), mn[0], mx[0], mn[1], mx[1], mn[2], mx[2])
		}
	}
	overPct := 0
	if totWithOra > 0 {
		overPct = totOverlap * 100 / totWithOra
	}
	fmt.Printf("\nOVERLAP par-slot (points offline DANS boite oracle du MEME slot) = %d/%d = %d%%\n", totOverlap, totWithOra, overPct)

	kindOv := map[filmdec.PosKind][2]int{}
	for s, km := range byKind {
		if _, has := oracle[s]; !has {
			continue
		}
		for k, ps := range km {
			c := kindOv[k]
			for _, v := range ps {
				c[1]++
				if inOraBox(s, v) {
					c[0]++
				}
			}
			kindOv[k] = c
		}
	}
	fmt.Printf("OVERLAP par kind : ")
	for _, k := range []filmdec.PosKind{filmdec.PosKindAbsolute, filmdec.PosKindAbsFallback, filmdec.PosKindDelta8, filmdec.PosKindDeltaAxis} {
		c := kindOv[k]
		p := 0
		if c[1] > 0 {
			p = c[0] * 100 / c[1]
		}
		fmt.Printf("%s=%d/%d(%d%%) ", k, c[0], c[1], p)
	}
	fmt.Println()
	fmt.Printf("TOTAL offline=%d (sur %d slots) ; oracle=%d (sur %d slots) ; ratio=%.1f%%\n",
		totOff, len(slots), totOra, len(oracle), 100*float64(totOff)/float64(max1(totOra)))
}

var okabeIto = []color.RGBA{
	{0xE6, 0x9F, 0x00, 255}, {0x56, 0xB4, 0xE9, 255}, {0x00, 0x9E, 0x73, 255},
	{0xF0, 0xE4, 0x42, 255}, {0x00, 0x72, 0xB2, 255}, {0xD5, 0x5E, 0x00, 255},
	{0xCC, 0x79, 0xA7, 255}, {0xBB, 0xBB, 0xBB, 255},
}

func writeTraj(out string, bySlot, oracle map[uint32][][3]float32) {
	const S, pad = 900, 40
	span := float64(S/2 - 2*pad)
	img := image.NewRGBA(image.Rect(0, 0, S, S/2))
	for y := 0; y < S/2; y++ {
		for x := 0; x < S; x++ {
			img.Set(x, y, color.RGBA{14, 16, 22, 255})
		}
	}
	mnx, mxx, mny, mxy := float64(boxMin[0]), float64(boxMax[0]), float64(boxMin[1]), float64(boxMax[1])
	sxr, syr := mxx-mnx, mxy-mny
	pt := func(v [3]float32, xoff int) (int, int) {
		u := (float64(v[0]) - mnx) / sxr
		w := (float64(v[1]) - mny) / syr
		return xoff + pad + int(u*span), S/2 - pad - int(w*span)
	}
	plot := func(m map[uint32][][3]float32, xoff int) {
		i := 0
		ss := make([]uint32, 0, len(m))
		for s := range m {
			ss = append(ss, s)
		}
		sort.Slice(ss, func(a, b int) bool { return ss[a] < ss[b] })
		for _, s := range ss {
			col := okabeIto[i%len(okabeIto)]
			i++
			for _, v := range m[s] {
				x, y := pt(v, xoff)
				if x >= 0 && x < S && y >= 0 && y < S/2 {
					img.Set(x, y, col)
				}
			}
		}
	}
	plot(oracle, 0)
	plot(bySlot, S/2)
	f, _ := os.Create(out)
	if f != nil {
		defer f.Close()
		png.Encode(f, img)
	}
	fmt.Printf("\n-> PNG %s (gauche=oracle, droite=offline)\n", out)
}

func writeTxt(p string, bySlot, oracle map[uint32][][3]float32, byKind map[uint32]map[filmdec.PosKind][][3]float32,
	idlow, absw int, quant float32, cont float64, seedK int, seedR float64, reseed int, oldAccept bool, accepted int) {
	var b strings.Builder
	fmt.Fprintf(&b, "# tmp_dense_i0 — accept-i0-avant-55 + continuite serree par slot\n")
	fmt.Fprintf(&b, "# IDLow=%d ABSW=%d QUANT=%.5f CONT=%.2f SEEDK=%d SEEDR=%.2f RESEED=%d OLDACCEPT=%v accepted=%d\n",
		idlow, absw, quant, cont, seedK, seedR, reseed, oldAccept, accepted)
	fmt.Fprintf(&b, "# slot offlineRecords oracleRecords\n")
	var slots []uint32
	for s := range bySlot {
		slots = append(slots, s)
	}
	for s := range oracle {
		if _, ok := bySlot[s]; !ok {
			slots = append(slots, s)
		}
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	for _, s := range slots {
		fmt.Fprintf(&b, "%d %d %d\n", s, len(bySlot[s]), len(oracle[s]))
	}
	os.WriteFile(p, []byte(b.String()), 0644)
}
