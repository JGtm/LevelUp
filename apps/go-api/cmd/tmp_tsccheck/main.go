package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"
)

func main() {
	d, _ := os.ReadFile(`c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/dmgcapture_run2.bin`)
	type ev struct {
		atk, vic, fam uint32
		tsc           uint64
	}
	var evs []ev
	tscCount := map[uint64]int{}
	trip := map[[3]uint32]int{}
	for o := 0; o+32 <= len(d); o += 32 {
		a := binary.LittleEndian.Uint32(d[o:])
		if a == 0 {
			continue
		}
		v := binary.LittleEndian.Uint32(d[o+4:])
		f := binary.LittleEndian.Uint32(d[o+8:])
		tsc := uint64(binary.LittleEndian.Uint32(d[o+20:]))<<32 | uint64(binary.LittleEndian.Uint32(d[o+16:]))
		evs = append(evs, ev{a, v, f, tsc})
		tscCount[tsc]++
		trip[[3]uint32{a, f, uint32(tsc & 0xffffffff)}]++
	}
	// distinct tsc
	multi := 0
	for _, c := range tscCount {
		if c > 1 {
			multi++
		}
	}
	fmt.Printf("=== %d events live ; %d timestamps distincts ; %d timestamps partagés (>1 event = AoE) ===\n", len(evs), len(tscCount), multi)
	// distinct (attacker,family,tsc) = vrais records de dégât distincts
	fmt.Printf("=== %d records de dégât DISTINCTS (attaquant+arme+tsc) ===\n", len(trip))
	// distribution du nb d'events par tsc
	dist := map[int]int{}
	for _, c := range tscCount {
		dist[c]++
	}
	var ks []int
	for k := range dist {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	for _, k := range ks {
		fmt.Printf("  %d event(s) au même tsc : %d cas\n", k, dist[k])
	}
}
