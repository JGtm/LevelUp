// tmp_mvarsol — inventaire des objets places par la VARIANTE DE CARTE (.mvar) de
// Cliffhanger : type, position, categorie, labels, noms lisibles. THROWAWAY.
//
// C est la CHAINE B du chantier « objets au sol » : ce que la CARTE pose, par construction
// — un objet du .mvar n a jamais ete lache par un joueur.
package main

import (
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

const defMvar = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/mapvar/cliffhanger_ridgeline.mvar`

func main() {
	path := defMvar
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	buf, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("lecture:", err)
		return
	}
	v, err := mapvar.Parse(buf)
	if err != nil {
		fmt.Println("parse:", err)
		return
	}
	fmt.Printf("fichier %s\n", path)
	fmt.Printf("levelID %d · %d objets · %d chaines lisibles\n\n", v.LevelID, len(v.Objects), len(v.Names))

	fmt.Println("=== chaines lisibles (root[10][1]) ===")
	for i, s := range v.Names {
		fmt.Printf("  %3d  %q\n", i, s)
		if i >= 200 {
			fmt.Printf("  ... (%d de plus)\n", len(v.Names)-i-1)
			break
		}
	}

	fmt.Println("\n=== histogramme des type_id ===")
	byType := map[int32][]mapvar.Object{}
	for _, o := range v.Objects {
		byType[o.TypeID] = append(byType[o.TypeID], o)
	}
	ids := make([]int32, 0, len(byType))
	for t := range byType {
		ids = append(ids, t)
	}
	sort.Slice(ids, func(i, j int) bool { return len(byType[ids[i]]) > len(byType[ids[j]]) })
	fmt.Printf("  %d type_id distincts\n", len(ids))
	for _, t := range ids {
		os_ := byType[t]
		cats := map[int]int{}
		nlbl := 0
		for _, o := range os_ {
			cats[o.Category]++
			nlbl += len(o.Labels)
		}
		fmt.Printf("  type %11d : %4d objets · categories %v · labels cumules %d\n", t, len(os_), cats, nlbl)
	}

	fmt.Println("\n=== histogramme des categories ===")
	byCat := map[int]int{}
	for _, o := range v.Objects {
		byCat[o.Category]++
	}
	cs := make([]int, 0, len(byCat))
	for c := range byCat {
		cs = append(cs, c)
	}
	sort.Ints(cs)
	for _, c := range cs {
		fmt.Printf("  categorie %4d : %4d objets\n", c, byCat[c])
	}

	fmt.Println("\n=== labels distincts (hash -> nom si resolu) ===")
	lbl := map[int32]int{}
	for _, o := range v.Objects {
		for _, h := range o.Labels {
			lbl[h]++
		}
	}
	hs := make([]int32, 0, len(lbl))
	for h := range lbl {
		hs = append(hs, h)
	}
	sort.Slice(hs, func(i, j int) bool { return lbl[hs[i]] > lbl[hs[j]] })
	for _, h := range hs {
		fmt.Printf("  %11d x%-4d %s\n", h, lbl[h], mapvar.LabelName(h))
	}
}
