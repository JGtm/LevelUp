// tmp_kvalidate — THROWAWAY : validation BIJECTION du dead-state. Calibration des composants
// CONSTANTS des death-deltas (i2,i3,i5,i6,i9 ; PAS i0 qui est variable 47/101 et géré par son deser).
// Décode le film, lit les dead-states (Mort), rattache chaque lecture à la mort chunk_27 la plus
// proche (victime) -> kill -> tueur. Cross-tab : EnumB->tueur, EnumA->victime. Pureté élevée =
// dead-state lu correctement (peu importe l'encodage exact de l'index).
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
	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
var bipedSlots = map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}
var xuidName = map[uint64]string{
	2535467794760703: "whiteknight2519", 2535437947245250: "JAVIERLOLITO540",
	2533274823110022: "JGtm", 2533274980284321: "LORD PEINX13",
	2533274815845110: "IKE ILYA", 2535444178793711: "Akatsuki fire17",
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
func listFrames(d []byte) [][2]interface{} {
	var out [][2]interface{}
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, [2]interface{}{ts, d[off+16 : off+16+sz]})
		}
		off += 16 + sz
	}
	return out
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

type ev struct {
	x uint64
	t int
}
type obs struct {
	t            int
	enumA, enumB int32
	gid          uint32
}

func main() {
	win := 150
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &win)
	}
	filmdec.SetRecordStateParam(2)
	filmdec.PositionCalibratedSkip = true // i0 : saut intelligent 47/101 selon bUsePred
	// calibration des composants CONSTANTS dans les death-deltas (PAS i0).
	filmdec.SetCalibratedWidth("object-forward-and-up-component", 9)           // i2
	filmdec.SetCalibratedWidth("object-angular-velocity-component", 1)         // i3
	filmdec.SetCalibratedWidth("object-shield-vitality-component", 29)         // i5
	filmdec.SetCalibratedWidth("object-region-state-component", 358)           // i6
	filmdec.SetCalibratedWidth("object-multiplayer-properties-component", 334) // i9
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	// kill feed
	events, _ := analysis.ParseHighlightEvents(mustRead(cache+"/chunk_27.bin"), 0)
	var kills, deaths []ev
	for _, e := range events {
		if e.EventType == analysis.EventTypeKill {
			kills = append(kills, ev{e.XUID, e.TimeMS})
		} else if e.EventType == analysis.EventTypeDeath {
			deaths = append(deaths, ev{e.XUID, e.TimeMS})
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].t < kills[j].t })
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].t < deaths[j].t })
	used := make([]bool, len(deaths))
	type kf struct {
		killer, victim uint64
		t              int
	}
	var feed []kf
	for _, k := range kills {
		best, bd := -1, 400
		for i, d := range deaths {
			if used[i] || d.x == k.x {
				continue
			}
			dt := abs(k.t - d.t)
			if dt < bd {
				bd, best = dt, i
			}
		}
		if best >= 0 {
			used[best] = true
			feed = append(feed, kf{k.x, deaths[best].x, k.t})
		}
	}

	// dead-state reads (Mort), unique par (slot, span)
	var obsv []obs
	for idx := 2; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			ts := fr[0].(uint64)
			pl := fr[1].([]byte)
			w := freshWorld(reg)
			recs, _ := filmdec.DecodeFrameRecords(filmdec.NewBitReader(pl), w, calCfg)
			tms := int((ts - t0Us) / 1000)
			for _, r := range recs {
				if !bipedSlots[r.Slot] || r.Trace.Dead == nil || !r.Trace.Dead.Mort {
					continue
				}
				d := r.Trace.Dead
				obsv = append(obsv, obs{tms, d.EnumA, d.EnumB, d.GlobalID})
			}
		}
	}
	fmt.Printf("=== %d kills ; %d dead-state reads (Mort) ===\n\n", len(feed), len(obsv))

	// chaque obs -> mort la plus proche (victime) ; puis kill -> tueur. cross-tab pureté.
	vbA := map[int32]map[uint64]int{} // EnumA -> victim_xuid
	vbB := map[int32]map[uint64]int{} // EnumB -> killer_xuid
	matched := 0
	for _, o := range obsv {
		bestV, bestDt, vt := uint64(0), win+1, 0
		for _, d := range deaths {
			if dt := abs(o.t - d.t); dt < bestDt {
				bestDt, bestV, vt = dt, d.x, d.t
			}
		}
		if bestDt > win {
			continue
		}
		killer := uint64(0)
		kdt := 500
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
		add(vbA, o.enumA, bestV)
		add(vbB, o.enumB, killer)
	}
	fmt.Printf("=== %d reads rattachés à une mort (±%dms) ===\n\n", matched, win)
	fmt.Println("--- EnumA(+4) -> VICTIME (bijection ?) ---")
	purity(vbA)
	fmt.Println("\n--- EnumB(+8) -> TUEUR (bijection ?) ---")
	purity(vbB)
}

func add(m map[int32]map[uint64]int, k int32, v uint64) {
	if m[k] == nil {
		m[k] = map[uint64]int{}
	}
	m[k][v]++
}
func purity(m map[int32]map[uint64]int) {
	var keys []int32
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	tot, pure := 0, 0
	for _, k := range keys {
		var ps []struct {
			x uint64
			c int
		}
		sum, top := 0, 0
		for x, c := range m[k] {
			ps = append(ps, struct {
				x uint64
				c int
			}{x, c})
			sum += c
			if c > top {
				top = c
			}
		}
		sort.Slice(ps, func(i, j int) bool { return ps[i].c > ps[j].c })
		tot += sum
		pure += top
		s := ""
		for i, p := range ps {
			if i >= 3 {
				break
			}
			s += fmt.Sprintf(" %s×%d", nm(p.x), p.c)
		}
		fmt.Printf("  EnumB=%-3d (n=%-3d) =>%s\n", k, sum, s)
	}
	if tot > 0 {
		fmt.Printf("  >>> PURETÉ GLOBALE : %.0f%% (%d/%d)\n", 100*float64(pure)/float64(tot), pure, tot)
	}
}
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
func mustRead(p string) []byte { b, _ := os.ReadFile(p); return b }
