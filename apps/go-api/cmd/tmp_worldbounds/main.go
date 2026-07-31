// tmp_worldbounds — dumpe `world bounds x/y/z` des tags sbsp de tous les modules de
// carte et la largeur d'axe prédite W = min(26, ceilLog2(ceil(60*extent))).
//
// Usage : CGO_ENABLED=1 go run ./cmd/tmp_worldbounds [dossier-levels]
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/himap"
	"levelup/go-api/internal/himodule"
)

// dumpGroups liste les groupes de tags d'un module (diagnostic : quel module porte
// vraiment la géométrie d'une carte).
func dumpGroups(mp string) {
	m, err := himodule.Open(mp)
	if err != nil {
		fmt.Printf("   open: %v\n", err)
		return
	}
	hist := map[string]int{}
	for _, f := range m.Files("") {
		hist[f.Group]++
	}
	keys := make([]string, 0, len(hist))
	for k := range hist {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, hist[k]))
	}
	fmt.Printf("   groupes: %v\n", parts)
}

const defLevels = `D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/ds/levels/multi`

func main() {
	levels := defLevels
	if len(os.Args) > 1 {
		levels = os.Args[1]
	}
	dirs, err := os.ReadDir(levels)
	if err != nil {
		fmt.Printf("levels: %v\n", err)
		return
	}
	names := make([]string, 0, len(dirs))
	for _, d := range dirs {
		if d.IsDir() {
			names = append(names, d.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		mods, _ := filepath.Glob(filepath.Join(levels, name, "*.module"))
		for _, mp := range mods {
			bsps, err := himap.ReadModuleBSPBounds(mp)
			if err != nil {
				fmt.Printf("== %-20s ERREUR %v\n", name, err)
				dumpGroups(mp)
				continue
			}
			fmt.Printf("== %-20s %d sbsp\n", name, len(bsps))
			for _, b := range bsps {
				w := b.Bounds.AxisWidths()
				fmt.Printf("   sbsp#%-3d %8d o  off=%#x valide=%v  x[%.5f,%.5f] y[%.5f,%.5f] z[%.5f,%.5f]  E=%.3f/%.3f/%.3f  W=%d/%d/%d\n",
					b.FileIndex, b.UncompSize, b.FieldOffset, b.Bounds.Valid(),
					b.Bounds.Min[0], b.Bounds.Max[0], b.Bounds.Min[1], b.Bounds.Max[1], b.Bounds.Min[2], b.Bounds.Max[2],
					b.Bounds.Extent(0), b.Bounds.Extent(1), b.Bounds.Extent(2),
					w[0], w[1], w[2])
			}
		}
	}
}
