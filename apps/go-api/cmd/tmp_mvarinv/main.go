// tmp_mvarinv — outil jetable : inventaire par type_id + dump complet du sac #8.
package main

import (
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

type row struct {
	idx    int
	typeID int64
	pos    [3]float64
	flags  uint64
	bag    string
	inst   int64
}

func main() {
	buf, _ := os.ReadFile(os.Args[1])
	root, err := mapvar.DecodeRoot(buf)
	if err != nil {
		fmt.Println("erreur:", err)
		os.Exit(1)
	}
	objs, _ := root.Field(3)
	rows := make([]row, 0, len(objs.Items))
	counts := map[int64]int{}
	for i, o := range objs.Items {
		r := row{idx: i}
		if t, ok := o.Field(2); ok {
			if v, ok2 := t.Field(0); ok2 {
				r.typeID = v.Int
			}
		}
		if p, ok := o.Field(3); ok {
			for k := uint16(0); k < 3; k++ {
				if c, ok2 := p.Field(k); ok2 {
					r.pos[k] = c.Float
				}
			}
		}
		if f, ok := o.Field(7); ok {
			r.flags = f.Uint
		}
		if b, ok := o.Field(8); ok {
			r.bag = flat(b, "")
		}
		if b, ok := o.Field(10); ok {
			if v, ok2 := b.Field(0); ok2 {
				r.inst = v.Int
			}
		}
		counts[r.typeID]++
		rows = append(rows, r)
	}

	fmt.Printf("objets=%d types=%d\n", len(rows), len(counts))
	ids := make([]int64, 0, len(counts))
	for id := range counts {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(a, b int) bool { return counts[ids[a]] > counts[ids[b]] })
	fmt.Println("--- inventaire type_id (n, bornes) ---")
	for _, id := range ids {
		var mn, mx [3]float64
		first := true
		for _, r := range rows {
			if r.typeID != id {
				continue
			}
			if first {
				mn, mx = r.pos, r.pos
				first = false
				continue
			}
			for k := 0; k < 3; k++ {
				if r.pos[k] < mn[k] {
					mn[k] = r.pos[k]
				}
				if r.pos[k] > mx[k] {
					mx[k] = r.pos[k]
				}
			}
		}
		fmt.Printf("%12d n=%3d x[%.1f..%.1f] y[%.1f..%.1f] z[%.1f..%.1f]\n",
			id, counts[id], mn[0], mx[0], mn[1], mx[1], mn[2], mx[2])
	}

	fmt.Println("--- objets (idx,type,pos,flags,inst,bag) ---")
	for _, r := range rows {
		fmt.Printf("%3d %12d (%8.2f,%8.2f,%8.2f) fl=%d inst=%d bag=%s\n",
			r.idx, r.typeID, r.pos[0], r.pos[1], r.pos[2], r.flags, r.inst, r.bag)
	}
}

func flat(v mapvar.Value, path string) string {
	if v.Fields != nil {
		keys := make([]int, 0, len(v.Fields))
		for id := range v.Fields {
			keys = append(keys, int(id))
		}
		sort.Ints(keys)
		out := ""
		for _, k := range keys {
			out += flat(v.Fields[uint16(k)], fmt.Sprintf("%s[%d]", path, k))
		}
		return out
	}
	if v.Items != nil {
		out := ""
		for i, it := range v.Items {
			out += flat(it, fmt.Sprintf("%s.%d", path, i))
		}
		return out
	}
	switch v.Type {
	case 7, 8:
		return fmt.Sprintf("%s=%g ", path, v.Float)
	case 9, 18:
		return fmt.Sprintf("%s=%q ", path, v.Str)
	case 2, 3, 4, 5, 6:
		return fmt.Sprintf("%s=u%d ", path, v.Uint)
	default:
		return fmt.Sprintf("%s=%d ", path, v.Int)
	}
}
