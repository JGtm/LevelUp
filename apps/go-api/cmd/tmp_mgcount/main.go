package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"levelup/go-api/internal/analysis/weaponv3"
	"os"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

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
func main() {
	est := func(b int) float64 { return 0 }
	var melees []weaponv3.MeleeHit
	var nades []weaponv3.GrenadeThrow
	for ch := 0; ch <= 27; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		if len(d) == 0 {
			continue
		}
		melees = append(melees, weaponv3.ScanMeleeHits(d, est)...)
		nades = append(nades, weaponv3.ScanGrenadeThrows(d, est)...)
	}
	leth := 0
	piM := map[int]int{}
	for _, h := range melees {
		if weaponv3.MeleeHitLethal(h.HitType) {
			leth++
		}
		piM[h.PI]++
	}
	fmt.Printf("=== 000d5950 : %d melee hits (%d létaux) ; %d grenade throws ===\n", len(melees), leth, len(nades))
	fmt.Printf("PI distribution melee : %v\n", piM)
}
