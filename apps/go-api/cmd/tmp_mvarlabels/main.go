// tmp_mvarlabels — outil jetable : inventaire des labels de mode de jeu portes par
// les objets d'un ou plusieurs .mvar (hash brut + nb d'objets + type_id concernes).
package main

import (
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

func main() {
	agg := map[int32]map[int32]int{} // label -> type_id -> n
	for _, p := range os.Args[1:] {
		buf, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		v, err := mapvar.Parse(buf)
		if err != nil {
			fmt.Fprintln(os.Stderr, p, err)
			continue
		}
		nTeam := 0
		for _, o := range v.Objects {
			if o.TeamIndex != mapvar.TeamUnset {
				nTeam++
			}
			for _, l := range o.Labels {
				if agg[l] == nil {
					agg[l] = map[int32]int{}
				}
				agg[l][o.TypeID]++
			}
		}
		fmt.Fprintf(os.Stderr, "%s: %d objets, %d avec equipe, %d noms\n",
			p, len(v.Objects), nTeam, len(v.Names))
	}
	keys := make([]int, 0, len(agg))
	for k := range agg {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, k := range keys {
		types := agg[int32(k)]
		tot := 0
		tl := make([]int, 0, len(types))
		for t, n := range types {
			tot += n
			tl = append(tl, int(t))
		}
		sort.Ints(tl)
		name := mapvar.LabelName(int32(k))
		if name == "" {
			name = "INCONNU"
		}
		fmt.Printf("%12d  %-24s n=%3d types=%v\n", k, name, tot, tl)
	}
}
