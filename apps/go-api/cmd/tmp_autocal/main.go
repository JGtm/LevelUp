// tmp_autocal — THROWAWAY : auto-calibration depuis le death-CSV. Pour CHAQUE composant biped,
// calcule la largeur-mode + la variabilité (depuis les cursors CE). Calibre tous les composants
// à largeur CONSTANTE (sauf i0 smart-calib + i11 dead-state à décoder). Mesure ensuite le taux de
// records biped propres (DesyncAt==-1) et la bijection EnumB->tueur. Liste les VARIABLES (à porter).
package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const deathCSV = `c:/Users/Guillaume/Downloads/filmdec_delta_death.csv`
const t0Us = uint64(4537898226)

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
var bipedSlots = map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}
var xuidName = map[uint64]string{
	2535467794760703: "whiteknight2519", 2535437947245250: "JAVIERLOLITO540", 2533274823110022: "JGtm",
	2533274980284321: "LORD PEINX13", 2533274815845110: "IKE ILYA", 2535444178793711: "Akatsuki fire17",
	2533274882097883: "aldusbroncus", 2533274826120416: "VitaminA1688",
}

func nm(x uint64) string {
	if g, ok := xuidName[x]; ok {
		return g
	}
	return fmt.Sprintf("%d", x)
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
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// largeurs CE par composant depuis le CSV : mode + nb de valeurs distinctes (variabilité).
func ceWidths() map[int]map[int]int {
	f, _ := os.Open(deathCSV)
	defer f.Close()
	type row struct{ eid, ti, ci, bc int }
	var rows []row
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		l := sc.Text()
		if l == "" || l[0] == '#' || l[0] == 'e' {
			continue
		}
		p := strings.Split(l, ",")
		if len(p) < 5 {
			continue
		}
		eid, _ := strconv.Atoi(p[0])
		ti, _ := strconv.Atoi(p[1])
		ci, _ := strconv.Atoi(p[2])
		bc, _ := strconv.Atoi(p[4])
		rows = append(rows, row{eid, ti, ci, bc})
	}
	w := map[int]map[int]int{} // compIndex -> width -> count
	var lastEid, lastCi, lastCur int = -1, -1, -1
	for _, r := range rows {
		if r.ti != 35 {
			lastEid, lastCi, lastCur = -1, -1, -1
			continue
		}
		if r.eid == lastEid && r.ci > lastCi && lastCur >= 0 {
			if w[lastCi] == nil {
				w[lastCi] = map[int]int{}
			}
			w[lastCi][r.bc-lastCur]++
		}
		lastEid, lastCi, lastCur = r.eid, r.ci, r.bc
	}
	return w
}

func freshWorld(reg *filmdec.Registry) *filmdec.World {
	raw, _ := os.ReadFile(cache + "/world_dump.txt")
	w := filmdec.NewWorld(reg)
	for _, tok := range bytes.Fields(raw) {
		s := string(tok)
		if len(s) == 0 || s[0] == '#' {
			continue
		}
		var slot, ti uint32
		if _, e := fmt.Sscanf(s, "%d:%d", &slot, &ti); e == nil {
			w.BindFull(slot, ti)
		}
	}
	return w
}
func listFrames(d []byte) []struct {
	ts uint64
	pl []byte
} {
	var out []struct {
		ts uint64
		pl []byte
	}
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, struct {
				ts uint64
				pl []byte
			}{ts, d[off+16 : off+16+sz]})
		}
		off += 16 + sz
	}
	return out
}

func main() {
	filmdec.SetRecordStateParam(2)
	filmdec.PositionCalibratedSkip = true
	reg, _ := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	arch, _ := reg.Archetype(35)
	names := arch.Components

	// auto-calibration : composant constant (1 seule largeur, n>=3) -> calibre (sauf i0/i11).
	cw := ceWidths()
	var variable []string
	calibrated := 0
	for ci := 0; ci < len(names); ci++ {
		wm := cw[ci]
		if len(wm) == 0 || ci == 0 || ci == 11 {
			continue // pas de data, ou i0 (smart-calib), ou i11 (dead-state à décoder)
		}
		// mode + total
		mode, modeN, tot := 0, 0, 0
		for wv, c := range wm {
			tot += c
			if c > modeN {
				modeN, mode = c, wv
			}
		}
		if len(wm) == 1 || float64(modeN)/float64(tot) >= 0.97 { // constant (ou ~constant)
			filmdec.SetCalibratedWidth(names[ci], mode)
			calibrated++
		} else {
			parts := []string{}
			for wv, c := range wm {
				parts = append(parts, fmt.Sprintf("%dx%d", wv, c))
			}
			sort.Strings(parts)
			variable = append(variable, fmt.Sprintf("i%d %s [%s]", ci, names[ci], strings.Join(parts, " ")))
		}
	}
	fmt.Printf("=== auto-calibration : %d composants constants calibrés ; %d VARIABLES (à porter) ===\n", calibrated, len(variable))
	for _, v := range variable {
		fmt.Printf("  %s\n", v)
	}

	// replay : taux de records biped propres + reachDead
	bipeds, clean, reachDead := 0, 0, 0
	slotRec, slotClean, slotDead := map[uint32]int{}, map[uint32]int{}, map[uint32]int{}
	desyncHist := map[int]int{}
	for idx := 2; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			w := freshWorld(reg)
			recs, _ := filmdec.DecodeFrameRecords(filmdec.NewBitReader(fr.pl), w, calCfg)
			for _, r := range recs {
				if !bipedSlots[r.Slot] {
					continue
				}
				bipeds++
				slotRec[r.Slot]++
				if r.DesyncAt == -1 {
					clean++
					slotClean[r.Slot]++
				} else {
					desyncHist[r.DesyncAt]++
				}
				if r.Trace.Dead != nil {
					reachDead++
					slotDead[r.Slot]++
				}
			}
		}
	}
	fmt.Printf("\n=== REPLAY : bipeds=%d clean(DesyncAt==-1)=%d (%.0f%%) reachDead=%d ===\n",
		bipeds, clean, 100*float64(clean)/float64(bipeds), reachDead)
	fmt.Println("=== par slot : records / clean / reachDead ===")
	var ss []uint32
	for s := range slotRec {
		ss = append(ss, s)
	}
	sort.Slice(ss, func(i, j int) bool { return ss[i] < ss[j] })
	for _, s := range ss {
		fmt.Printf("  slot%d : rec=%-6d clean=%-6d (%.0f%%) reachDead=%d\n",
			s, slotRec[s], slotClean[s], 100*float64(slotClean[s])/float64(slotRec[s]), slotDead[s])
	}
	fmt.Println("=== histogramme DesyncAt (composant où le record s'arrête) — top ===")
	type dh struct{ at, c int }
	var dhs []dh
	for at, c := range desyncHist {
		dhs = append(dhs, dh{at, c})
	}
	sort.Slice(dhs, func(i, j int) bool { return dhs[i].c > dhs[j].c })
	for i, e := range dhs {
		if i >= 12 {
			break
		}
		nm := "?"
		if e.at >= 0 && e.at < len(names) {
			nm = names[e.at]
		}
		fmt.Printf("  i%-2d %-44s : %d records\n", e.at, nm, e.c)
	}

	// bijection EnumB->tueur (reads avec DesyncAt==-1 uniquement = records bit-exacts)
	events, _ := analysis.ParseHighlightEvents(read(cache+"/chunk_27.bin"), 0)
	type ev struct {
		x uint64
		t int
	}
	var kills, deaths []ev
	for _, e := range events {
		if e.EventType == analysis.EventTypeKill {
			kills = append(kills, ev{e.XUID, e.TimeMS})
		} else if e.EventType == analysis.EventTypeDeath {
			deaths = append(deaths, ev{e.XUID, e.TimeMS})
		}
	}
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].t < deaths[j].t })
	used := make([]bool, len(deaths))
	type kf struct {
		killer, victim uint64
		t              int
	}
	var feed []kf
	sort.Slice(kills, func(i, j int) bool { return kills[i].t < kills[j].t })
	for _, k := range kills {
		best, bd := -1, 400
		for i, d := range deaths {
			if used[i] || d.x == k.x {
				continue
			}
			if dt := abs(k.t - d.t); dt < bd {
				bd, best = dt, i
			}
		}
		if best >= 0 {
			used[best] = true
			feed = append(feed, kf{k.x, deaths[best].x, k.t})
		}
	}
	vbB := map[int32]map[uint64]int{}
	slotA := map[uint32]map[int32]int{} // slot -> EnumA(victime) : doit être CONSTANT par slot
	matched := 0
	for idx := 2; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			w := freshWorld(reg)
			recs, _ := filmdec.DecodeFrameRecords(filmdec.NewBitReader(fr.pl), w, calCfg)
			tms := int((fr.ts - t0Us) / 1000)
			for _, r := range recs {
				if !bipedSlots[r.Slot] || r.DesyncAt != -1 || r.Trace.Dead == nil || !r.Trace.Dead.Mort {
					continue
				}
				if slotA[r.Slot] == nil {
					slotA[r.Slot] = map[int32]int{}
				}
				slotA[r.Slot][r.Trace.Dead.EnumA]++
				bestV, bestDt, vt := uint64(0), 200, 0
				for _, d := range deaths {
					if dt := abs(tms - d.t); dt < bestDt {
						bestDt, bestV, vt = dt, d.x, d.t
					}
				}
				if bestDt > 200 {
					continue
				}
				killer, kdt := uint64(0), 400
				for _, f := range feed {
					if f.victim != bestV {
						continue
					}
					if dt := abs(vt - f.t); dt < kdt {
						kdt, killer = dt, f.killer
					}
				}
				if killer == 0 {
					continue
				}
				matched++
				if vbB[r.Trace.Dead.EnumB] == nil {
					vbB[r.Trace.Dead.EnumB] = map[uint64]int{}
				}
				vbB[r.Trace.Dead.EnumB][killer]++
			}
		}
	}
	tot, pure := 0, 0
	for _, vm := range vbB {
		s, top := 0, 0
		for _, c := range vm {
			s += c
			if c > top {
				top = c
			}
		}
		tot += s
		pure += top
	}
	fmt.Printf("\n=== BIJECTION EnumB->tueur (records DesyncAt==-1 uniquement) : %d reads appariés ===\n", matched)
	if tot > 0 {
		fmt.Printf("  >>> PURETÉ : %.0f%% (%d/%d)\n", 100*float64(pure)/float64(tot), pure, tot)
	}

	// check interne : EnumA(victime) constant par slot ? (pas d'appariement temporel = pas de bruit)
	fmt.Println("\n=== EnumA(+4) par slot biped (victime = doit être CONSTANTE par slot) ===")
	var slots []uint32
	for s := range slotA {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	tA, pA := 0, 0
	for _, s := range slots {
		var ps []struct {
			a int32
			c int
		}
		sum, top := 0, 0
		for a, c := range slotA[s] {
			ps = append(ps, struct {
				a int32
				c int
			}{a, c})
			sum += c
			if c > top {
				top = c
			}
		}
		sort.Slice(ps, func(i, j int) bool { return ps[i].c > ps[j].c })
		tA += sum
		pA += top
		str := ""
		for i, p := range ps {
			if i >= 4 {
				break
			}
			str += fmt.Sprintf(" A=%d×%d", p.a, p.c)
		}
		fmt.Printf("  slot%d (n=%-4d) =>%s\n", s, sum, str)
	}
	if tA > 0 {
		fmt.Printf("  >>> PURETÉ EnumA par slot : %.0f%% (%d/%d) — élevé = tête dead-state décodée OK\n", 100*float64(pA)/float64(tA), pA, tA)
	}
}

func read(p string) []byte { b, _ := os.ReadFile(p); return b }
