// tmp_mvarhashes — outil jetable : liste tous les entiers int32 distincts presents
// dans les sacs de proprietes #8 des objets d'un .mvar (candidats hashs de labels).
package main

import (
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

func main() {
	seen := map[int64]int{}
	for _, path := range os.Args[1:] {
		buf, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		root, err := mapvar.DecodeRoot(buf)
		if err != nil {
			fmt.Fprintln(os.Stderr, path, err)
			continue
		}
		objs, _ := root.Field(3)
		for _, o := range objs.Items {
			if bag, ok := o.Field(8); ok {
				walk(bag, seen)
			}
			if t, ok := o.Field(2); ok {
				if v, ok2 := t.Field(0); ok2 {
					seen[v.Int]++
				}
			}
		}
	}
	keys := make([]int, 0, len(seen))
	for k := range seen {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, k := range keys {
		fmt.Println(k)
	}
	fmt.Fprintf(os.Stderr, "%d entiers distincts\n", len(keys))
}

func walk(v mapvar.Value, seen map[int64]int) {
	if v.Fields != nil {
		for _, f := range v.Fields {
			walk(f, seen)
		}
		return
	}
	if v.Items != nil {
		for _, it := range v.Items {
			walk(it, seen)
		}
		return
	}
	if v.Type == 16 || v.Type == 17 { // int32 / int64
		if v.Int != 0 && (v.Int > 65535 || v.Int < -65535) {
			seen[v.Int]++
		}
	}
}
