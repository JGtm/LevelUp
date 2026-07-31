package main

import (
	"fmt"
	"sort"
)

// scanXUIDs locates each known player xuid (64-bit, both bit-orders) in the payload,
// and reports its bit position relative to the calibrated record starts. This tests
// whether the keyframe stores the xuid near each biped record (=> direct mapping).
func scanXUIDs(payload []byte, recordStarts []int) {
	fmt.Printf("\n================ SCAN xuid (64-bit) dans le payload ================\n")
	total := len(payload)*8 - 64
	type hit struct {
		pi   int
		bit  int
		mode string
	}
	var hits []hit
	for pi, x := range piXUID {
		for bp := 0; bp <= total; bp++ {
			if bitsAt(payload, bp, 64) == x {
				hits = append(hits, hit{pi, bp, "MSB64"})
			}
		}
		// Try low-32 alone too (xuid often stored truncated).
		low := uint32(x)
		for bp := 0; bp <= total+32; bp++ {
			if uint32(bitsAt(payload, bp, 32)) == low {
				hits = append(hits, hit{pi, bp, "low32"})
			}
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].bit < hits[j].bit })
	if len(hits) == 0 {
		fmt.Printf("  AUCUN xuid trouvé (ni 64-bit MSB ni low32) dans ce chunk.\n")
		return
	}
	for _, h := range hits {
		near := ""
		for ri, s := range recordStarts {
			if h.bit >= s-200 && h.bit <= s+3000 {
				near = fmt.Sprintf("  (dans/après record#%d start=%d, +%d bits)", ri, s, h.bit-s)
			}
		}
		fmt.Printf("  pi%d xuid %s @bit%-7d%s\n", h.pi, h.mode, h.bit, near)
	}
}
