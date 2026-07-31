// tmp_modfiles — liste toutes les entrées d'un .module (tags + resources) avec
// group, nb blocs, tailles. Sert à repérer les resources de géométrie.
package main

import (
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/himodule"
)

const defMod = `D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/ds/levels/multi/catalyst/catalyst-rtx-new.module`

func main() {
	p := defMod
	if len(os.Args) > 1 {
		p = os.Args[1]
	}
	m, err := himodule.Open(p)
	if err != nil {
		fmt.Println(err)
		return
	}
	files := m.Files("")
	fmt.Printf("%d fichiers\n", len(files))
	// tri par taille décompressée décroissante.
	sort.Slice(files, func(i, j int) bool { return files[i].UncompSize > files[j].UncompSize })
	fmt.Println("idx    group   blocks   compSize  uncompSize")
	for i := 0; i < 25 && i < len(files); i++ {
		f := files[i]
		g := f.Group
		if g == "" {
			g = "(res)"
		}
		fmt.Printf("%5d  %-6s  %5d   %9x  %9x\n", f.Index, g, f.BlockCount, f.CompSize, f.UncompSize)
	}
}
