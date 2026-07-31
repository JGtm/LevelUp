// tmp_rootblocks — inventaire des tag-blocks racine du plus gros sbsp d'un module :
// quels blocs sont réellement peuplés dans un build donné (ds / any / pc).
//
// Usage : CGO_ENABLED=1 go run ./cmd/tmp_rootblocks <module...>
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"levelup/go-api/internal/himap"
)

func main() {
	for _, p := range os.Args[1:] {
		fmt.Printf("=== %s / %s ===\n", filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(p))))), filepath.Base(filepath.Dir(p)))
		infos, err := himap.DescribeRootBlocks(p)
		if err != nil {
			fmt.Println("  ERR", err)
			continue
		}
		for _, in := range infos {
			if in.Count == 0 && in.Target < 0 {
				continue // bloc vide : bruit
			}
			fmt.Printf("  %2d off%#-6x %-46s target=%-5d size=%-9d count=%d\n",
				in.Rank, in.FieldOffset, in.PluginName, in.Target, in.BlockSize, in.Count)
		}
	}
}
