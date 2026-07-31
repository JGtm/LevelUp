// tmp_objsol — RECONNAISSANCE des objets au sol (armes ti=42, equipement ti=37) dans un
// film. THROWAWAY : outil de mesure, pas de production.
//
// Phase R1 : recensement des keyframes — combien d'entites par archetype, quels slots,
// quelles generations, a quels instants. C'est la seule source qui donne slot -> ti.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_objsol [filmDir]
package main

import (
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const defFilm = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

type kfObs struct {
	chunk, pkt    int
	tsUS          uint64
	slot, ti, gen int
}

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	n := filmdec.CountFilmChunks(dir)
	fmt.Printf("film %s : %d chunks\n", dir, n)

	reg := loadRegistry(dir)

	var obs []kfObs
	kfCount := 0
	var t0 uint64
	first := true
	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range filmdec.WalkPackets(data) {
			if first {
				t0, first = pk.TimestampUS, false
			}
			if pk.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			kfCount++
			for _, r := range filmdec.WalkKeyframeWorld(pk.Payload(data)) {
				obs = append(obs, kfObs{c, pk.Index, pk.TimestampUS, r.Slot, r.TI, r.Gen})
			}
		}
	}
	fmt.Printf("keyframes trouves : %d ; records cumules : %d ; t0=%d us\n\n", kfCount, len(obs), t0)

	// --- R1a : histogramme par archetype (records cumules et slots distincts) ---
	recsByTI := map[int]int{}
	slotsByTI := map[int]map[int]bool{}
	for _, o := range obs {
		recsByTI[o.ti]++
		if slotsByTI[o.ti] == nil {
			slotsByTI[o.ti] = map[int]bool{}
		}
		slotsByTI[o.ti][o.slot] = true
	}
	tis := make([]int, 0, len(recsByTI))
	for ti := range recsByTI {
		tis = append(tis, ti)
	}
	sort.Ints(tis)
	fmt.Println("ti | records cumules | slots distincts | slot min..max | nom d archetype")
	for _, ti := range tis {
		lo, hi := 1<<30, -1
		for s := range slotsByTI[ti] {
			if s < lo {
				lo = s
			}
			if s > hi {
				hi = s
			}
		}
		fmt.Printf("%3d | %7d | %5d | %5d..%-5d | %s\n",
			ti, recsByTI[ti], len(slotsByTI[ti]), lo, hi, archName(reg, ti))
	}

	// --- R1b : par keyframe, combien de ti=42 / ti=37 ---
	fmt.Println("\nkeyframe | t(s) | total | ti35 | ti37 | ti42 | ti41 | ti11")
	type kfKey struct{ c, p int }
	order := []kfKey{}
	byKF := map[kfKey][]kfObs{}
	for _, o := range obs {
		k := kfKey{o.chunk, o.pkt}
		if byKF[k] == nil {
			order = append(order, k)
		}
		byKF[k] = append(byKF[k], o)
	}
	for _, k := range order {
		v := byKF[k]
		cnt := map[int]int{}
		for _, o := range v {
			cnt[o.ti]++
		}
		fmt.Printf("c%02d/p%-4d | %7.1f | %5d | %4d | %4d | %4d | %4d | %4d\n",
			k.c, k.p, float64(v[0].tsUS-t0)/1e6, len(v),
			cnt[35], cnt[37], cnt[42], cnt[41], cnt[11])
	}

	// --- R1c : pour ti=42 et ti=37, la vie de chaque slot (generations et instants) ---
	for _, want := range []int{42, 37} {
		fmt.Printf("\n=== ti=%d : slot -> (gen, nb de keyframes, premier/dernier t) ===\n", want)
		type life struct {
			slot, gen     int
			nKF           int
			firstT, lastT float64
		}
		key := func(s, g int) [2]int { return [2]int{s, g} }
		m := map[[2]int]*life{}
		var ord [][2]int
		for _, o := range obs {
			if o.ti != want {
				continue
			}
			t := float64(o.tsUS-t0) / 1e6
			k := key(o.slot, o.gen)
			if m[k] == nil {
				m[k] = &life{slot: o.slot, gen: o.gen, firstT: t, lastT: t}
				ord = append(ord, k)
			}
			m[k].nKF++
			if t < m[k].firstT {
				m[k].firstT = t
			}
			if t > m[k].lastT {
				m[k].lastT = t
			}
		}
		sort.Slice(ord, func(i, j int) bool {
			if ord[i][0] != ord[j][0] {
				return ord[i][0] < ord[j][0]
			}
			return ord[i][1] < ord[j][1]
		})
		fmt.Printf("  %d couples (slot,gen) distincts\n", len(ord))
		shown := 0
		for _, k := range ord {
			l := m[k]
			fmt.Printf("  slot %5d gen %d : %3d keyframes, t %.1f..%.1f s\n",
				l.slot, l.gen, l.nKF, l.firstT, l.lastT)
			if shown++; shown >= 80 {
				fmt.Printf("  ... (%d couples supplementaires non affiches)\n", len(ord)-shown)
				break
			}
		}
	}
}

func loadRegistry(dir string) *filmdec.Registry {
	raw, err := os.ReadFile(dir + "/chunk_00.bin")
	if err != nil {
		return nil
	}
	reg, err := filmdec.ParseRegistryChunk(raw)
	if err != nil {
		return nil
	}
	return reg
}

func archName(reg *filmdec.Registry, ti int) string {
	if reg == nil {
		return "?"
	}
	a, ok := reg.Archetype(ti)
	if !ok || len(a.Components) == 0 {
		return "?"
	}
	last := a.Components[len(a.Components)-1]
	return fmt.Sprintf("%d composants, i0=%s ... i%d=%s", len(a.Components),
		a.Components[0], len(a.Components)-1, last)
}
