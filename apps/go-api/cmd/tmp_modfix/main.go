// tmp_modfix — mesure le taux d'extraction d'entrées d'un .module avec la règle
// dataBase courante de internal/himodule. Sert de baseline AVANT/APRÈS correctif.
//
// Usage : CGO_ENABLED=1 go run ./cmd/tmp_modfix [module...]
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/himodule"
)

var defaults = []string{
	`D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/ds/levels/multi/ridgeline/ridgeline-rtx-new.module`,
	`D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/ds/levels/multi/catalyst/catalyst-rtx-new.module`,
	`D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/ds/levels/multi/ctf_aquarius/ctf_aquarius-rtx-new.module`,
	`D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/ds/levels/multi/ctf_bazaar/ctf_bazaar-rtx-new.module`,
	`D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/ds/levels/multi/btb_highpower/btb_highpower-rtx-new.module`,
	`D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/any/levels/multi/catalyst/catalyst-rtx-new.module`,
	`D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/any/levels/multi/ridgeline/ridgeline-rtx-new.module`,
}

func main() {
	mods := defaults
	if len(os.Args) > 1 {
		mods = os.Args[1:]
	}
	for _, p := range mods {
		m, err := himodule.Open(p)
		if err != nil {
			fmt.Printf("%-46s OPEN ERR %v\n", filepath.Base(filepath.Dir(p)), err)
			continue
		}
		files := m.Files("")
		ok, ko := 0, 0
		byGroupOK := map[string]int{}
		byGroupKO := map[string]int{}
		for _, f := range files {
			d, err := m.Extract(f)
			if err != nil || len(d) != f.UncompSize {
				ko++
				byGroupKO[f.Group]++
				continue
			}
			ok++
			byGroupOK[f.Group]++
		}
		fmt.Printf("%-24s (%s) : %d entrées, OK=%d KO=%d (%.1f%% OK)\n",
			filepath.Base(filepath.Dir(p)), parent2(p), len(files), ok, ko,
			100*float64(ok)/float64(max(1, len(files))))
		// détail groupes en échec
		var gs []string
		for g := range byGroupKO {
			gs = append(gs, g)
		}
		sort.Strings(gs)
		for _, g := range gs {
			if byGroupKO[g] > 0 {
				fmt.Printf("      KO %-6q %5d (OK %d)\n", g, byGroupKO[g], byGroupOK[g])
			}
		}
	}
}

func parent2(p string) string {
	d := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(p))))
	return filepath.Base(d)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
