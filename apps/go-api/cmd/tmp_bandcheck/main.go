package main

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

func main() {
	dir := `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\cache\film_chunks\000d5950`
	n := filmdec.CountFilmChunks(dir)
	byTI := map[int]map[uint32]bool{}
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			for _, r := range filmdec.WalkKeyframeWorld(p.Payload(chunk)) {
				if byTI[r.TI] == nil {
					byTI[r.TI] = map[uint32]bool{}
				}
				byTI[r.TI][uint32(r.Slot)] = true
			}
		}
	}
	tis := []int{}
	for k := range byTI {
		tis = append(tis, k)
	}
	sort.Ints(tis)
	fmt.Printf("  %-6s %-8s %s\n", "ti", "slots", "plage brute (min..max)")
	for _, ti := range tis {
		s := byTI[ti]
		if len(s) == 0 {
			continue
		}
		lo, hi := uint32(1<<31), uint32(0)
		for k := range s {
			if k < lo {
				lo = k
			}
			if k > hi {
				hi = k
			}
		}
		mark := ""
		if ti == 37 || ti == 41 || ti == 42 {
			mark = "   <---"
		}
		fmt.Printf("  %-6d %-8d %d..%d  (etendue %d, remplie a %.0f %%)%s\n",
			ti, len(s), lo, hi, hi-lo+1, 100*float64(len(s))/float64(hi-lo+1), mark)
	}
}
