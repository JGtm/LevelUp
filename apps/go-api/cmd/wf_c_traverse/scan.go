package main

// Scan direct des littéraux d'arme (high-32 famille) autour de R0 pour confirmer la
// structure récord (2 armes à +195 bits) et la position exacte de l'ancre.

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

func familyMap() map[uint32]string {
	m := map[uint32]string{}
	for id, n := range analysis.WeaponIDToName {
		m[uint32(id>>32)] = n
	}
	return m
}

// scanWeaponsRange : positions bit de chaque littéral high-32 d'arme dans [lo,hi).
func scanWeaponsRange(payload []byte, lo, hi int) []struct {
	Pos  int
	Name string
} {
	fam := familyMap()
	var out []struct {
		Pos  int
		Name string
	}
	br := filmdec.NewBitReader(payload)
	tot := len(payload) * 8
	if hi > tot-32 {
		hi = tot - 32
	}
	for bp := lo; bp < hi; bp++ {
		br2 := filmdec.NewBitReader(payload)
		br2.Skip(bp)
		v := uint32(br2.ReadBits(32))
		if n, ok := fam[v]; ok {
			out = append(out, struct {
				Pos  int
				Name string
			}{bp, n})
		}
	}
	_ = br
	sort.Slice(out, func(i, j int) bool { return out[i].Pos < out[j].Pos })
	return out
}

func dumpWeaponsAroundR0(payload []byte) {
	fmt.Println("\n=== littéraux d'arme dans [193000, 217000) (records R0..R7) ===")
	ws := scanWeaponsRange(payload, 193000, 217000)
	for _, w := range ws {
		fmt.Printf("  bit=%d  %s\n", w.Pos, w.Name)
	}
	fmt.Printf("  total : %d littéraux d'arme\n", len(ws))
}
