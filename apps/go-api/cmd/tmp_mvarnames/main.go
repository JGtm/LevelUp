// tmp_mvarnames — outil jetable : explore root[6], root[10], root[11] et le sac de
// proprietes #8 d'un .mvar, pour relier les noms lisibles aux objets places.
package main

import (
	"fmt"
	"os"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

func main() {
	buf, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("erreur:", err)
		os.Exit(1)
	}
	root, err := mapvar.DecodeRoot(buf)
	if err != nil {
		fmt.Println("erreur:", err)
		os.Exit(1)
	}

	fmt.Println("=== root[10] ===")
	if n, ok := root.Field(10); ok {
		for id, f := range n.Fields {
			fmt.Printf("  champ %d: items=%d pairs=%d\n", id, len(f.Items), len(f.Pairs))
			for i, it := range f.Items {
				fmt.Printf("    [%d] %s\n", i, brief(it))
			}
		}
	}

	fmt.Println("=== root[6] ===")
	if n, ok := root.Field(6); ok {
		for id, f := range n.Fields {
			fmt.Printf("  champ %d: %s\n", id, brief(f))
			for i, it := range f.Items {
				fmt.Printf("    [%d] %s\n", i, brief(it))
			}
		}
	}

	fmt.Println("=== root[11] (10 premieres paires) ===")
	if n, ok := root.Field(11); ok {
		for i, kv := range n.Pairs {
			if i >= 10 {
				break
			}
			fmt.Printf("  %s -> %s\n", brief(kv.Key), brief(kv.Val))
		}
	}

	fmt.Println("=== sac #8 de l'objet 0 ===")
	objs, _ := root.Field(3)
	if len(objs.Items) > 0 {
		if bag, ok := objs.Items[0].Field(8); ok {
			deep(bag, "    #8", 0)
		}
	}
}

func brief(v mapvar.Value) string {
	switch {
	case v.Fields != nil:
		out := fmt.Sprintf("struct{%d}", len(v.Fields))
		for id, f := range v.Fields {
			out += fmt.Sprintf(" %d=%s", id, leaf(f))
		}
		return out
	case v.Items != nil:
		out := fmt.Sprintf("list[%d]", len(v.Items))
		for i, it := range v.Items {
			if i >= 6 {
				out += " ..."
				break
			}
			out += " " + leaf(it)
		}
		return out
	case v.Pairs != nil:
		return fmt.Sprintf("map[%d]", len(v.Pairs))
	default:
		return leaf(v)
	}
}

func leaf(v mapvar.Value) string {
	if v.Str != "" {
		return fmt.Sprintf("%q", v.Str)
	}
	if v.Fields != nil {
		return fmt.Sprintf("{%d ch}", len(v.Fields))
	}
	if v.Items != nil {
		return fmt.Sprintf("[%d]", len(v.Items))
	}
	switch v.Type {
	case 7, 8:
		return fmt.Sprintf("%g", v.Float)
	case 3, 4, 5, 6, 2:
		return fmt.Sprintf("u%d", v.Uint)
	default:
		return fmt.Sprintf("i%d", v.Int)
	}
}

func deep(v mapvar.Value, path string, lvl int) {
	if lvl > 4 {
		return
	}
	if v.Fields != nil {
		for id, f := range v.Fields {
			deep(f, fmt.Sprintf("%s[%d]", path, id), lvl+1)
		}
		return
	}
	if v.Items != nil {
		for i, it := range v.Items {
			deep(it, fmt.Sprintf("%s.%d", path, i), lvl+1)
		}
		return
	}
	fmt.Printf("%s = %s\n", path, leaf(v))
}
