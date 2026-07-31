// tmp_mvarexplore — outil jetable : dump structurel d'un .mvar decode en Bond CBv2.
// Sert a etablir la grammaire (quel champ porte quoi) avant de figer le parseur metier.
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: tmp_mvarexplore <fichier.mvar> [profondeur]")
		os.Exit(2)
	}
	depth := 3
	if len(os.Args) > 2 {
		fmt.Sscanf(os.Args[2], "%d", &depth)
	}
	buf, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("erreur lecture:", err)
		os.Exit(1)
	}
	root, err := mapvar.DecodeRoot(buf)
	if err != nil {
		fmt.Println("erreur decode:", err)
		os.Exit(1)
	}
	fmt.Printf("OK parse complet : %d octets\n", len(buf))
	dump(root, "root", 0, depth)
}

func dump(v mapvar.Value, path string, lvl, max int) {
	indent := strings.Repeat("  ", lvl)
	switch {
	case v.Fields != nil:
		ids := make([]int, 0, len(v.Fields))
		for id := range v.Fields {
			ids = append(ids, int(id))
		}
		sort.Ints(ids)
		fmt.Printf("%s%s: struct{%d champs} ids=%v\n", indent, path, len(ids), ids)
		if lvl >= max {
			return
		}
		for _, id := range ids {
			dump(v.Fields[uint16(id)], fmt.Sprintf("%s[%d]", path, id), lvl+1, max)
		}
	case v.Items != nil:
		fmt.Printf("%s%s: list[%d]\n", indent, path, len(v.Items))
		if lvl >= max {
			return
		}
		n := len(v.Items)
		if n > 4 {
			n = 4
		}
		for i := 0; i < n; i++ {
			dump(v.Items[i], fmt.Sprintf("%s.%d", path, i), lvl+1, max)
		}
		if len(v.Items) > n {
			fmt.Printf("%s  ... %d de plus\n", indent, len(v.Items)-n)
		}
	case v.Pairs != nil:
		fmt.Printf("%s%s: map[%d]\n", indent, path, len(v.Pairs))
	default:
		fmt.Printf("%s%s: scalar type=%d int=%d uint=%d float=%g str=%q\n",
			indent, path, v.Type, v.Int, v.Uint, v.Float, v.Str)
	}
}
